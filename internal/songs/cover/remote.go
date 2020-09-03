package cover

import (
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/meiraka/vv/internal/mpd"
	"github.com/syndtr/goleveldb/leveldb"
)

const (
	timeout = 5 * time.Second
)

// RemoteSearcher searches song cover art by mpd albumart api.
type RemoteSearcher struct {
	httpPrefix string
	cacheDir   string
	client     *mpd.Client
	db         *leveldb.DB       // song path - img name
	url2img    map[string]string // url path - img path
	mu         sync.RWMutex
}

// NewRemoteSearcher creates MPDSearcher.
func NewRemoteSearcher(httpPrefix string, c *mpd.Client, cacheDir string) (*RemoteSearcher, error) {
	if err := os.MkdirAll(cacheDir, 0766); err != nil {
		return nil, err
	}
	db, err := leveldb.OpenFile(filepath.Join(cacheDir, "db"), nil)
	if err != nil {
		return nil, err
	}

	s := &RemoteSearcher{
		httpPrefix: httpPrefix,
		cacheDir:   cacheDir,
		client:     c,
		db:         db,
		url2img:    map[string]string{},
	}
	iter := s.db.NewIterator(nil, nil)
	for iter.Next() {
		value := string(iter.Value())
		if len(value) != 0 {
			s.url2img[path.Join(s.httpPrefix, value)] = filepath.Join(s.cacheDir, value)
		}
	}
	iter.Release()
	return s, nil
}

// Rescan rescans all songs images.
func (s *RemoteSearcher) Rescan(songs []map[string][]string) {
	t := make(map[string]string, len(songs))
	for i := range songs {
		if k, ok := s.songPath(songs[i]); ok {
			t[path.Dir(k)] = k
		}
	}
	for _, k := range t {
		s.updateCache(k)
	}
}

func (s *RemoteSearcher) songPath(song map[string][]string) (string, bool) {
	file, ok := song["file"]
	if !ok {
		return "", false
	}
	if len(file) != 1 {
		return "", false
	}
	return file[0], true
}

// Close finalizes cache db, coroutines.
func (s *RemoteSearcher) Close() error {
	s.db.Close()
	return nil
}

// ServeHTTP serves local cover art with httpPrefix
func (s *RemoteSearcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	path, ok := s.url2img[r.URL.Path]
	s.mu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	serveImage(path, w, r)
}

func (s *RemoteSearcher) getURLPath(songPath string) ([]string, error) {
	key := []byte(path.Dir(songPath))
	name, err := s.db.Get(key, nil)
	if err != nil {
		return nil, err
	}
	if len(name) == 0 {
		return []string{}, nil
	}
	return []string{path.Join(s.httpPrefix, string(name))}, nil
}

func (s *RemoteSearcher) updateCache(songPath string) []string {
	ret := make([]string, 0, 1)
	key := []byte(path.Dir(songPath))

	filename := ""
	if name, err := s.db.Get(key, nil); err == leveldb.ErrNotFound {
	} else if err != nil {
		s.db.Put(key, []byte{}, nil)
		return ret
	} else {
		filename = string(name)
		// remove ext
		i := strings.LastIndex(filename, ".")
		if i < 0 {
			i = len(filename)
		}
		filename = filename[0:i]
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	b, err := s.client.AlbumArt(ctx, songPath)
	if err != nil {
		// set zero value for not found
		// TODO: check error value
		s.db.Put(key, []byte{}, nil)
		return ret
	}
	// save image to random filename.
	ext, err := ext(b)
	if err != nil {
		s.db.Put(key, []byte{}, nil)
		return ret
	}
	var f *os.File
	if len(filename) == 0 {
		f, err = ioutil.TempFile(s.cacheDir, "*."+ext)
	} else {
		f, err = os.Create(filepath.Join(s.cacheDir, filename+"."+ext))
	}
	if err != nil {
		s.db.Put(key, []byte{}, nil)
		return ret
	}
	f.Write(b)
	if err := f.Close(); err != nil {
		s.db.Put(key, []byte{}, nil)
		return ret
	}
	// stores filename to db
	value := filepath.Base(f.Name())
	if err := s.db.Put(key, []byte(value), nil); err != nil {
		os.Remove(filepath.Join(s.cacheDir, value))
		return ret
	}
	s.mu.Lock()
	s.url2img[path.Join(s.httpPrefix, value)] = filepath.Join(s.cacheDir, value)
	s.mu.Unlock()
	_, err = s.getURLPath(songPath)
	return append(ret, path.Join(s.httpPrefix, value))
}

// GetURLs returns cover path for m
func (s *RemoteSearcher) GetURLs(m map[string][]string) ([]string, bool) {
	if s == nil {
		return nil, true
	}
	songPath, ok := s.songPath(m)
	if !ok {
		return nil, true
	}
	cover, err := s.getURLPath(songPath)
	if err == nil {
		return cover, true
	}
	return nil, false
}
