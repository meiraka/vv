package images

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// Local provides http server song conver art from local filesystem.
type Local struct {
	httpPrefix     string
	musicDirectory string
	files          []string
	cache          map[string][]string
	url2img        map[string]string
	img2req        map[string]string
	mu             sync.RWMutex
}

// NewLocal creates Local.
func NewLocal(httpPrefix string, dir string, files []string) (*Local, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	return &Local{
		httpPrefix:     httpPrefix,
		musicDirectory: dir,
		files:          files,
		cache:          map[string][]string{},
		url2img:        map[string]string{},
		img2req:        map[string]string{},
	}, nil
}

// ServeHTTP serves cover art with httpPrefix
func (l *Local) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	l.mu.RLock()
	path, ok := l.url2img[r.URL.Path]
	l.mu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	serveImage(path, w, r)
}

// Update rescans all songs images.
func (l *Local) Update(ctx context.Context, song map[string][]string) error {
	k, ok := l.songDirPath(song)
	if !ok {
		return nil
	}

	l.mu.RLock()
	_, ok = l.cache[k]
	l.mu.RUnlock()
	if ok {
		return nil
	}

	l.updateCache(k)
	return nil
}

// Rescan rescans song image.
func (l *Local) Rescan(ctx context.Context, song map[string][]string, reqid string) error {
	k, ok := l.songDirPath(song)
	if !ok {
		return nil
	}

	l.mu.Lock()
	lastReq, ok := l.img2req[k]
	l.img2req[k] = reqid
	l.mu.Unlock()
	if ok && lastReq == reqid {
		return nil
	}
	l.updateCache(k)
	return nil
}

func (l *Local) songDirPath(song map[string][]string) (string, bool) {
	file, ok := song["file"]
	if !ok {
		return "", false
	}
	if len(file) != 1 {
		return "", false
	}
	localPath := filepath.Join(filepath.FromSlash(l.musicDirectory), filepath.FromSlash(file[0]))
	return filepath.Dir(localPath), true
}

func (l *Local) updateCache(songDirPath string) []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	ret := []string{}
	for _, n := range l.files {
		rpath := filepath.Join(songDirPath, n)
		s, err := os.Stat(rpath)
		if err == nil {
			cover := path.Join(l.httpPrefix, strings.TrimPrefix(strings.TrimPrefix(filepath.ToSlash(rpath), filepath.ToSlash(l.musicDirectory)), "/"))

			ret = append(ret, cover+"?"+url.Values{"d": {strconv.FormatInt(s.ModTime().Unix(), 10)}}.Encode())
			l.url2img[cover] = rpath
		}
	}
	l.cache[songDirPath] = ret
	return ret
}

// GetURLs returns cover path for m
func (l *Local) GetURLs(m map[string][]string) ([]string, bool) {
	if l == nil {
		return nil, true
	}
	songDirPath, ok := l.songDirPath(m)
	if !ok {
		return nil, true
	}

	l.mu.RLock()
	v, ok := l.cache[songDirPath]
	l.mu.RUnlock()
	if ok {
		return v, true
	}
	return l.updateCache(songDirPath), true
}
