package assets

import (
	"fmt"
	"mime"
	"net/http"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/meiraka/vv/internal/gzip"
	"github.com/meiraka/vv/internal/http/request"
)

// Config represents  configuration
type Config struct {
	Local    bool   // use local asset files
	LocalDir string // path to asset files directory
}

// NewHandler returns hander for asset files.
func NewHandler(conf *Config) (http.HandlerFunc, error) {
	c := new(Config)
	if conf != nil {
		*c = *conf
	}
	if c.LocalDir == "" {
		c.LocalDir = "assets"
	}
	appCSS, err := c.assetsHandler(filepath.Join(c.LocalDir, "app.css"), AppCSS, AppCSSHash)
	if err != nil {
		return nil, err
	}
	appJS, err := c.assetsHandler(filepath.Join(c.LocalDir, "app.js"), AppJS, AppJSHash)
	if err != nil {
		return nil, err
	}
	appPNG, err := c.assetsHandler(filepath.Join(c.LocalDir, "app.png"), AppPNG, AppPNGHash)
	if err != nil {
		return nil, err
	}
	appSVG, err := c.assetsHandler(filepath.Join(c.LocalDir, "app.svg"), AppSVG, AppSVGHash)
	if err != nil {
		return nil, err
	}
	manifestJSON, err := c.assetsHandler(filepath.Join(c.LocalDir, "manifest.json"), ManifestJSON, ManifestJSONHash)
	if err != nil {
		return nil, err
	}
	appBlackPNG, err := c.assetsHandler(filepath.Join(c.LocalDir, "app-black.png"), AppBlackPNG, AppBlackPNGHash)
	if err != nil {
		return nil, err
	}
	appBlackSVG, err := c.assetsHandler(filepath.Join(c.LocalDir, "app-black.svg"), AppBlackSVG, AppBlackSVGHash)
	if err != nil {
		return nil, err
	}
	wPNG, err := c.assetsHandler(filepath.Join(c.LocalDir, "w.png"), WPNG, WPNGHash)
	if err != nil {
		return nil, err
	}
	nocoverSVG, err := c.assetsHandler(filepath.Join(c.LocalDir, "nocover.svg"), NocoverSVG, NocoverSVGHash)
	if err != nil {
		return nil, err
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.NotFound(w, r)
			return
		}
		switch r.URL.Path {
		case "/assets/app.css":
			appCSS(w, r)
		case "/assets/app.js":
			appJS(w, r)
		case "/assets/app.png":
			appPNG(w, r)
		case "/assets/app.svg":
			appSVG(w, r)
		case "/assets/manifest.json":
			manifestJSON(w, r)
		case "/assets/app-black.png":
			appBlackPNG(w, r)
		case "/assets/app-black.svg":
			appBlackSVG(w, r)
		case "/assets/w.png":
			wPNG(w, r)
		case "/assets/nocover.svg":
			nocoverSVG(w, r)
		default:
			http.NotFound(w, r)
		}
	}, nil
}

func (c *Config) assetsHandler(rpath string, b []byte, hash []byte) (http.HandlerFunc, error) {
	if c.Local {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Cache-Control", "max-age=1")
			http.ServeFile(w, r, rpath)
		}, nil
	}
	m := mime.TypeByExtension(path.Ext(rpath))
	var gz []byte
	var err error
	if m != "image/png" && m != "image/jpg" {
		if gz, err = gzip.Encode(b); err != nil {
			return nil, fmt.Errorf("%s: gzip: %w", rpath, err)
		}
	}
	length := strconv.Itoa(len(b))
	gzLength := strconv.Itoa(len(gz))
	etag := fmt.Sprintf(`"%s"`, hash)
	lastModified := time.Now().Format(http.TimeFormat)
	return func(w http.ResponseWriter, r *http.Request) {
		if request.NoneMatch(r, etag) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Add("ETag", etag)
		w.Header().Add("Last-Modified", lastModified)
		if m != "" {
			w.Header().Add("Content-Type", m)
		}
		// extend the expiration date for versioned request
		if r.URL.Query().Get("h") != "" {
			w.Header().Add("Cache-Control", "max-age=31536000")
		} else {
			w.Header().Add("Cache-Control", "max-age=86400")
		}
		if gz != nil {
			w.Header().Add("Vary", "Accept-Encoding")
			if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") && gz != nil {
				w.Header().Add("Content-Encoding", "gzip")
				w.Header().Add("Content-Length", gzLength)
				w.WriteHeader(http.StatusOK)
				w.Write(gz)
				return
			}
		}
		w.Header().Add("Content-Length", length)
		w.Write(b)
	}, nil
}
