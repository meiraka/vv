package cover

import (
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// LocalSearcher searches song conver art
type LocalSearcher struct {
	httpPrefix     string
	musicDirectory string
	files          []string
	cache          map[string][]string
	url2img        map[string]string
	mu             sync.RWMutex
	event          chan struct{}
}

// NewLocalSearcher creates LocalSearcher.
func NewLocalSearcher(httpPrefix string, dir string, files []string) (*LocalSearcher, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	return &LocalSearcher{
		httpPrefix:     httpPrefix,
		musicDirectory: dir,
		files:          files,
		cache:          map[string][]string{},
		url2img:        map[string]string{},
	}, nil
}

// ServeHTTP serves local cover art with httpPrefix
func (l *LocalSearcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	l.mu.RLock()
	path, ok := l.url2img[r.URL.Path]
	l.mu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	serveImage(path, w, r)
}

// Rescan rescans all songs images.
func (l *LocalSearcher) Rescan(songs []map[string][]string) {
	t := make(map[string]struct{}, len(songs))
	for i := range songs {
		if k, ok := l.songDirPath(songs[i]); ok {
			t[k] = struct{}{}
		}
	}
	for k := range t {
		l.updateCache(k)
	}
}

func (l *LocalSearcher) songDirPath(song map[string][]string) (string, bool) {
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

func (l *LocalSearcher) updateCache(songDirPath string) []string {
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
func (l *LocalSearcher) GetURLs(m map[string][]string) ([]string, bool) {
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
