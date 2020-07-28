package songs

import (
	"bytes"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"math"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	_ "golang.org/x/image/bmp"
	"golang.org/x/image/draw"
)

// LocalCoverSearcher searches song conver art
type LocalCoverSearcher struct {
	prefix string
	dir    string
	files  []string
	cache  map[string][]string
	rcache map[string]string
	mu     sync.RWMutex
}

// NewLocalCoverSearcher creates LocalCoverSearcher.
func NewLocalCoverSearcher(httpPrefix string, dir string, files []string) (*LocalCoverSearcher, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	return &LocalCoverSearcher{
		prefix: httpPrefix,
		dir:    dir,
		files:  files,
		cache:  map[string][]string{},
		rcache: map[string]string{},
	}, nil
}

// ServeHTTP serves local cover art with httpPrefix
func (l *LocalCoverSearcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path, ok := l.rcache[r.URL.Path]
	if !ok {
		http.NotFound(w, r)
		return
	}
	serveImage(path, w, r)
}

// AddTags adds cover path to m
func (l *LocalCoverSearcher) AddTags(m map[string][]string) map[string][]string {
	if l == nil {
		return m
	}
	file, ok := m["file"]
	if !ok {
		return m
	}
	if len(file) != 1 {
		return m
	}
	localPath := filepath.Join(filepath.FromSlash(l.dir), filepath.FromSlash(file[0]))
	localDir := filepath.Dir(localPath)
	l.mu.Lock()
	defer l.mu.Unlock()
	v, ok := l.cache[localDir]
	if ok {
		d := make([]string, len(v))
		copy(d, v)
		m["cover"] = d
		return m
	}
	v = []string{}
	for _, n := range l.files {
		rpath := filepath.Join(localDir, n)
		_, err := os.Stat(rpath)
		if err == nil {
			cover := path.Join(l.prefix, strings.TrimPrefix(strings.TrimPrefix(filepath.ToSlash(rpath), filepath.ToSlash(l.dir)), "/"))
			v = append(v, cover)
			l.rcache[cover] = rpath
		}
	}
	l.cache[localDir] = v
	d := make([]string, len(v))
	copy(d, v)
	m["cover"] = d
	return m
}

func resizeImage(data io.ReadSeeker, width, height int) ([]byte, error) {
	info, _, err := image.DecodeConfig(data)
	if err != nil {
		return nil, err
	}
	if _, err := data.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	img, _, err := image.Decode(data)
	if err != nil {
		return nil, err
	}
	imgRatio := float64(info.Width) / float64(info.Height)
	outRatio := float64(width) / float64(height)
	if imgRatio > outRatio {
		height = int(math.Round(float64(height*info.Height) / float64(info.Width)))
	} else {
		width = int(math.Round(float64(width*info.Width) / float64(info.Height)))
	}
	rect := image.Rect(0, 0, width, height)
	out := image.NewRGBA(rect)
	draw.CatmullRom.Scale(out, rect, img, img.Bounds(), draw.Over, nil)
	outwriter := new(bytes.Buffer)
	opt := jpeg.Options{Quality: 100}
	jpeg.Encode(outwriter, out, &opt)
	return outwriter.Bytes(), nil
}

func serveImage(rpath string, w http.ResponseWriter, r *http.Request) {
	i, err := os.Stat(rpath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if !modifiedSince(r, i.ModTime()) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	f, err := os.Open(rpath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer f.Close()
	q := r.URL.Query()
	ws, hs := q.Get("width"), q.Get("height")
	if len(ws) == 0 || len(hs) == 0 {
		w.Header().Add("Cache-Control", "max-age=86400")
		w.Header().Add("Content-Length", strconv.FormatInt(i.Size(), 10))
		w.Header().Add("Content-Type", mime.TypeByExtension(path.Ext(rpath)))
		w.Header().Add("Last-Modified", i.ModTime().Format(http.TimeFormat))
		io.CopyN(w, f, i.Size())
		return
	}
	wi, err := strconv.Atoi(ws)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	hi, err := strconv.Atoi(hs)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	b, err := resizeImage(f, wi, hi)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Add("Cache-Control", "max-age=86400")
	w.Header().Add("Content-Length", strconv.Itoa(len(b)))
	w.Header().Add("Content-Type", mime.TypeByExtension(path.Ext(rpath)))
	w.Header().Add("Last-Modified", i.ModTime().Format(http.TimeFormat))
	w.Write(b)
}

/*modifiedSince compares If-Modified-Since header given time.Time.*/
func modifiedSince(r *http.Request, l time.Time) bool {
	t, err := time.Parse(http.TimeFormat, r.Header.Get("If-Modified-Since"))
	if err != nil {
		return true
	}
	return !l.Before(t.Add(time.Second))
}
