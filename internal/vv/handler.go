package vv

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/meiraka/vv/internal/gzip"
	"github.com/meiraka/vv/internal/log"
	"github.com/meiraka/vv/internal/request"
	"github.com/meiraka/vv/internal/vv/assets"
	"golang.org/x/text/language"
)

// Config represents options for root page generator.
type Config struct {
	Local        bool      // use local index.html(default: false)
	LocalDir     string    // path to local index.html directory(default: filepath.Join("internal", "vv"))
	LastModified time.Time // Last-Modified value(default: time.Now())
	Tree         Tree      // playlist view definition(default: DefaultTree)
	TreeOrder    []string  // order of playlist tree(default: DefaultTreeOrder)
	Data         []byte    // index.html data(default: embed index.html)
	Logger       interface {
		Debugf(format string, v ...interface{})
	}
}

// Handler serves app root page.
type Handler struct {
	conf         *Config
	hashData     *dataHash
	configData   *dataConfig
	lastModified string
	plainBody    [][]byte
	plainLength  []string
	plainEtag    []string
	gzBody       [][]byte
	gzLength     []string
	gzEtag       []string
}

// New creates http.Handler for app root page.
func New(c *Config) (*Handler, error) {
	conf := &Config{}
	if c != nil {
		*conf = *c
	}
	if conf.LocalDir == "" {
		conf.LocalDir = filepath.Join("internal", "vv")
	}
	if conf.LastModified.IsZero() {
		conf.LastModified = time.Now()
	}
	conf.LastModified = conf.LastModified.UTC()
	if conf.Tree == nil && conf.TreeOrder == nil {
		conf.Tree = DefaultTree
		conf.TreeOrder = DefaultTreeOrder
	}
	if conf.Tree == nil && conf.TreeOrder != nil {
		return nil, errors.New("vv: invalid config: no tree")
	}
	if conf.Tree != nil && conf.TreeOrder == nil {
		return nil, errors.New("vv: invalid config: no tree order")
	}
	if conf.Data == nil {
		conf.Data = indexHTML
	}
	if conf.Logger == nil {
		conf.Logger = log.New(io.Discard)
	}
	// setup handler
	hashData, err := newHashData()
	if err != nil {
		return nil, err
	}
	configData, err := newConfigData(&conf.Tree, conf.TreeOrder)
	if err != nil {
		return nil, err
	}
	langs := len(langPrio)
	h := &Handler{
		conf:         conf,
		hashData:     hashData,
		configData:   configData,
		lastModified: conf.LastModified.UTC().Format(http.TimeFormat),
		plainBody:    make([][]byte, langs),
		plainLength:  make([]string, langs),
		plainEtag:    make([]string, langs),
		gzBody:       make([][]byte, langs),
		gzLength:     make([]string, langs),
		gzEtag:       make([]string, langs),
	}
	for i, lang := range langPrio {
		b, err := h.generate(conf.Data, lang, false)
		if err != nil {
			return nil, err
		}
		h.plainBody[i] = b
		h.plainLength[i] = strconv.Itoa(len(b))
		h.plainEtag[i] = etag2(b)
		gz, err := gzip.Encode(b)
		if err != nil {
			return nil, err
		}
		h.gzBody[i] = gz
		h.gzLength[i] = strconv.Itoa(len(gz))
		h.gzEtag[i] = etag2(gz)
	}
	return h, nil
}

func etag2(b []byte) string {
	hasher := md5.New()
	hasher.Write(b)
	return `"` + hex.EncodeToString(hasher.Sum(nil)) + `"`
}

func (h *Handler) serveHTTPLocal(w http.ResponseWriter, r *http.Request) error {
	tag, _ := determineLang(r)
	localPath := filepath.Join(h.conf.LocalDir, "index.html")
	lastModified := h.conf.LastModified.UTC()
	s, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("stat: %s: %v", localPath, err)
	}
	if modtime := s.ModTime().UTC(); modtime.After(lastModified) {
		lastModified = modtime
	}
	if !request.ModifiedSince(r, lastModified) {
		w.WriteHeader(304)
		return nil
	}
	src, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("read file: %s: %v", localPath, err)
	}
	b, err := h.generate(src, tag, true)
	if err != nil {
		return fmt.Errorf("generate: %s: %v", localPath, err)
	}
	w.Header().Add("Content-Language", tag.String())
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.Header().Add("Vary", "Accept-Language")
	w.Header().Add("Cache-Control", "max-age=1")
	w.Header().Add("Content-Length", strconv.Itoa(len(b)))
	w.Header().Add("Last-Modified", lastModified.Format(http.TimeFormat))
	w.Write(b)
	return nil
}

// ServeHTTP serves root page.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.conf.Local {
		err := h.serveHTTPLocal(w, r)
		if err == nil {
			return
		}
		h.conf.Logger.Debugf("vv: %v", err)
	}

	tag, index := determineLang(r)
	gzBody := h.gzBody[index]
	useGZ := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") && gzBody != nil
	var etag string
	if useGZ {
		etag = h.gzEtag[index]
	} else {
		etag = h.plainEtag[index]
	}
	if request.NoneMatch(r, etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Add("Cache-Control", "max-age=86400")
	w.Header().Add("Content-Language", tag.String())
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.Header().Add("Etag", etag)
	w.Header().Add("Last-Modified", h.lastModified)
	w.Header().Add("Vary", "Accept-Encoding, Accept-Language")
	if useGZ {
		w.Header().Add("Content-Encoding", "gzip")
		w.Header().Add("Content-Length", h.gzLength[index])
		w.Write(gzBody)
		return
	}
	w.Header().Add("Content-Length", h.plainLength[index])
	w.Write(h.plainBody[index])
}

func (h *Handler) generate(b []byte, lang language.Tag, local bool) ([]byte, error) {
	data := &data{
		Hash:    h.hashData,
		Config:  h.configData,
		Message: langData[lang],
	}
	if local {
		data.Hash = &dataHash{}
	}
	t, err := template.New("index.html").Parse(string(b))
	if err != nil {
		return nil, err
	}
	buf := new(bytes.Buffer)
	if err := t.Execute(buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type data struct {
	Hash    *dataHash
	Config  *dataConfig
	Message map[string]string
}

type dataHash struct {
	AppJS  string
	AppCSS string
}

func newHashData() (*dataHash, error) {
	css, ok := assets.Hash("/assets/app.css")
	if !ok {
		return nil, errors.New("no app.css in assets")
	}
	js, ok := assets.Hash("/assets/app.js")
	if !ok {
		return nil, errors.New("no app.js in assets")
	}
	return &dataHash{AppJS: js, AppCSS: css}, nil
}

type dataConfig struct {
	Tree      string
	TreeOrder string
}

func newConfigData(tree *Tree, treeOrder []string) (*dataConfig, error) {
	jsonTree, err := json.Marshal(tree)
	if err != nil {
		return nil, fmt.Errorf("tree: %v", err)
	}
	jsonTreeOrder, err := json.Marshal(treeOrder)
	if err != nil {
		return nil, fmt.Errorf("tree order: %v", err)
	}
	return &dataConfig{Tree: string(jsonTree), TreeOrder: string(jsonTreeOrder)}, nil
}
