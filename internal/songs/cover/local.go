package cover

import (
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// LocalSearcher searches song conver art
type LocalSearcher struct {
	prefix string
	dir    string
	files  []string
	cache  map[string][]string
	rcache map[string]string
	mu     sync.RWMutex
}

// NewLocalSearcher creates LocalSearcher.
func NewLocalSearcher(httpPrefix string, dir string, files []string) (*LocalSearcher, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	return &LocalSearcher{
		prefix: httpPrefix,
		dir:    dir,
		files:  files,
		cache:  map[string][]string{},
		rcache: map[string]string{},
	}, nil
}

// ServeHTTP serves local cover art with httpPrefix
func (l *LocalSearcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path, ok := l.rcache[r.URL.Path]
	if !ok {
		http.NotFound(w, r)
		return
	}
	serveImage(path, w, r)
}

// AddTags adds cover path to m
func (l *LocalSearcher) AddTags(m map[string][]string) map[string][]string {
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

/*modifiedSince compares If-Modified-Since header given time.Time.*/
func modifiedSince(r *http.Request, l time.Time) bool {
	t, err := time.Parse(http.TimeFormat, r.Header.Get("If-Modified-Since"))
	if err != nil {
		return true
	}
	return !l.Before(t.Add(time.Second))
}
