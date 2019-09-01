package songs

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// LocalCoverSearcher searches song conver art
type LocalCoverSearcher struct {
	dir    string
	files  []string
	cache  map[string]string
	rcache map[string]struct{}
	mu     sync.RWMutex
}

// NewLocalCoverSearcher creates LocalCoverSearcher.
func NewLocalCoverSearcher(dir string, files []string) (*LocalCoverSearcher, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	return &LocalCoverSearcher{
		dir:    dir,
		files:  files,
		cache:  map[string]string{},
		rcache: map[string]struct{}{},
	}, nil
}

// Cached returns true if given path is valid cache image
func (l *LocalCoverSearcher) Cached(path string) (cached bool) {
	if l == nil {
		return false
	}
	l.mu.RLock()
	_, cached = l.rcache[path]
	l.mu.RUnlock()
	return
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
		if len(v) != 0 {
			m["cover"] = []string{v}
		}
		return m
	}
	for _, n := range l.files {
		rpath := filepath.Join(localDir, n)
		_, err := os.Stat(rpath)
		if err == nil {
			cover := strings.TrimPrefix(strings.TrimPrefix(filepath.ToSlash(rpath), filepath.ToSlash(l.dir)), "/")
			l.cache[localDir] = cover
			l.rcache[cover] = struct{}{}
			m["cover"] = []string{cover}
			return m
		}
	}
	l.cache[localDir] = ""
	return m
}
