package main

import (
	"bytes"
	"image"
	"image/color"
	_ "image/gif"
	"image/jpeg"
	"image/png"
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

	_ "golang.org/x/image/bmp"
	"golang.org/x/image/draw"
)

func expandImage(data []byte, width, height int) ([]byte, error) {
	r := bytes.NewReader(data)
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, err
	}
	outRect := image.Rectangle{image.ZP, image.Pt(width, height)}
	out := image.NewRGBA(outRect)
	w := color.RGBA{255, 255, 255, 255}
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			out.Set(x, y, w)
		}
	}
	s := img.Bounds().Size()
	l := image.Pt((width-s.X)/2, (height-s.Y)/2)
	target := image.Rectangle{l, image.Pt(l.X+s.X, l.Y+s.Y)}
	draw.Draw(out, target, img, image.ZP, draw.Over)
	outwriter := new(bytes.Buffer)
	png.Encode(outwriter, out)
	return outwriter.Bytes(), nil
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

// LocalCoverSearcher searches song conver art
type LocalCoverSearcher struct {
	dir   string
	glob  string
	cache map[string]string
	image map[string]struct{}
	mu    sync.RWMutex
}

// NewLocalCoverSearcher creates LocalCoverSearcher.
func NewLocalCoverSearcher(dir, glob string) (*LocalCoverSearcher, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	return &LocalCoverSearcher{
		dir:   dir,
		glob:  glob,
		cache: map[string]string{},
		image: map[string]struct{}{},
	}, nil
}

// CachedImage returns true if given path is cached image
func (l *LocalCoverSearcher) CachedImage(path string) (cached bool) {
	l.mu.RLock()
	_, cached = l.image[path]
	l.mu.RUnlock()
	return
}

// AddTags adds cover path to m
func (l *LocalCoverSearcher) AddTags(m map[string][]string) map[string][]string {
	file, ok := m["file"]
	if !ok {
		return m
	}
	if len(file) != 1 {
		return m
	}
	localPath := filepath.Join(filepath.FromSlash(l.dir), filepath.FromSlash(file[0]))
	localGlob := filepath.Join(filepath.Dir(localPath), l.glob)
	l.mu.Lock()
	defer l.mu.Unlock()
	v, ok := l.cache[localGlob]
	if ok {
		if len(v) != 0 {
			m["cover"] = []string{v}
		}
		return m
	}
	p, err := filepath.Glob(localGlob)
	if err != nil || p == nil {
		l.cache[localGlob] = ""
		return m
	}
	cover := strings.TrimPrefix(strings.TrimPrefix(filepath.ToSlash(p[0]), filepath.ToSlash(l.dir)), "/")
	l.cache[localGlob] = cover
	l.image[cover] = struct{}{}
	m["cover"] = []string{cover}
	return m
}

// Handler returns HTTP handler for image
func (l *LocalCoverSearcher) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !l.CachedImage(r.URL.Path) {
			http.NotFound(w, r)
			return
		}
		rpath := filepath.Join(filepath.FromSlash(l.dir), filepath.FromSlash(r.URL.Path))
		f, err := os.Open(rpath)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer f.Close()
		i, err := f.Stat()
		if err != nil {
			writeHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		if !modifiedSince(r, i.ModTime()) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
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
			writeHTTPError(w, http.StatusBadRequest, err)
		}
		hi, err := strconv.Atoi(hs)
		if err != nil {
			writeHTTPError(w, http.StatusBadRequest, err)
		}
		b, err := resizeImage(f, wi, hi)
		if err != nil {
			writeHTTPError(w, http.StatusInternalServerError, err)
		}
		w.Header().Add("Cache-Control", "max-age=86400")
		w.Header().Add("Content-Length", strconv.Itoa(len(b)))
		w.Header().Add("Content-Type", mime.TypeByExtension(path.Ext(rpath)))
		w.Header().Add("Last-Modified", i.ModTime().Format(http.TimeFormat))
		w.Write(b)
	}
}
