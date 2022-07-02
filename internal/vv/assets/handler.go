package assets

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/meiraka/vv/internal/log"
	"github.com/meiraka/vv/internal/request"
)

// Config represents  configuration
type Config struct {
	Local        bool      // use local asset files
	LocalDir     string    // path to asset files directory
	LastModified time.Time // asset LastModified
	Logger       interface {
		Debugf(format string, v ...interface{})
	}
}

type Handler struct {
	conf        *Config
	lastModifed string
}

func NewHandler(conf *Config) (*Handler, error) {
	if conf == nil {
		conf = &Config{}
	}
	if conf.LocalDir == "" {
		conf.LocalDir = filepath.Join("internal", "vv", "assets")
	}
	if conf.LastModified.IsZero() {
		conf.LastModified = time.Now()
	}
	conf.LastModified = conf.LastModified.UTC()
	if conf.Logger == nil {
		conf.Logger = log.New(io.Discard)
	}
	h := &Handler{
		conf:        conf,
		lastModifed: conf.LastModified.UTC().Format(http.TimeFormat),
	}
	return h, nil
}

// ServeHTTP serves asset files.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.conf.Local {
		name := strings.TrimPrefix(r.URL.Path, "/assets/")
		rpath := filepath.Join(h.conf.LocalDir, filepath.FromSlash(name))
		s, err := os.Stat(rpath)
		if err == nil {
			if request.ModifiedSince(r, s.ModTime().UTC()) {
				w.Header().Add("Cache-Control", "max-age=1")
				http.ServeFile(w, r, rpath)
			} else {
				w.WriteHeader(http.StatusNotModified)
			}
			return
		}
		h.conf.Logger.Debugf("assets: %s: %v", r.URL.Path, err)

	}
	i, ok := embedIndex(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	gzBody := embedGZBody[i]
	useGZ := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") && gzBody != nil
	var etag string
	if useGZ {
		etag = `"` + embedGZHash[i] + `"`
	} else {
		etag = `"` + embedHash[i] + `"`
	}
	if request.NoneMatch(r, etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	// extend the expiration date for versioned request
	if r.URL.Query().Get("h") != "" {
		w.Header().Add("Cache-Control", "max-age=31536000")
	} else {
		w.Header().Add("Cache-Control", "max-age=86400")
	}
	if m := embedMine[i]; m != "" {
		w.Header().Add("Content-Type", m)
	}
	w.Header().Add("ETag", etag)
	w.Header().Add("Last-Modified", h.lastModifed)
	if gzBody != nil {
		w.Header().Add("Vary", "Accept-Encoding")
		if useGZ {
			w.Header().Add("Content-Encoding", "gzip")
			w.Header().Add("Content-Length", embedGZLength[i])
			w.WriteHeader(http.StatusOK)
			w.Write(gzBody)
			return
		}
	}
	w.Header().Add("Content-Length", embedLength[i])
	w.Write(embedBody[i])
}
