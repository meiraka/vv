package cover

import (
	"context"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/meiraka/vv/internal/mpd"
	"github.com/syndtr/goleveldb/leveldb"
)

// RemoteSearcherConfig represents optional searcher settings.
type RemoteSearcherConfig struct {
	Timeout time.Duration // fetching request timeout per file
}

// RemoteSearcher searches song cover art by mpd albumart api.
type RemoteSearcher struct {
	config     *RemoteSearcherConfig
	httpPrefix string
	cacheDir   string
	client     *mpd.Client
	db         *leveldb.DB       // song path - img name
	url2img    map[string]string // url path - img path
	mu         sync.RWMutex
}

// NewRemoteSearcher creates MPDSearcher.
func (c RemoteSearcherConfig) NewRemoteSearcher(httpPrefix string, client *mpd.Client, cacheDir string) (*RemoteSearcher, error) {
	if err := os.MkdirAll(cacheDir, 0766); err != nil {
		return nil, err
	}
	db, err := leveldb.OpenFile(filepath.Join(cacheDir, "db2"), nil)
	if err != nil {
		return nil, err
	}

	s := &RemoteSearcher{
		config:     &c,
		httpPrefix: httpPrefix,
		cacheDir:   cacheDir,
		client:     client,
		db:         db,
		url2img:    map[string]string{},
	}
	iter := s.db.NewIterator(nil, nil)
	for iter.Next() {
		value := string(iter.Value())
		if len(value) != 0 {
			i := strings.LastIndex(value, "?")
			if i > 0 {
				value = value[0:i]
			}
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
	ctx := context.Background()
	for _, k := range t {
		s.updateCache(ctx, k)
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

func (s *RemoteSearcher) updateCache(ctx context.Context, songPath string) []string {
	ret := make([]string, 0, 1)
	key := []byte(path.Dir(songPath))

	filename := ""
	var version int64
	if name, err := s.db.Get(key, nil); err == leveldb.ErrNotFound {
	} else if err != nil {
		return ret
	} else {
		filename = string(name)
		// get version
		i := strings.LastIndex(filename, "=")
		if i > 0 {
			if v, err := strconv.ParseInt(filename[i+1:], 10, 64); err == nil {
				version = v
				if version == 9223372036854775807 {
					version = 0
				} else {
					version = version + 1
				}
			}
		}
		// remove ext
		i = strings.LastIndex(filename, ".")
		if i < 0 {
			i = len(filename)
		}
		filename = filename[0:i]
	}
	if s.config.Timeout != 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.config.Timeout)
		defer cancel()
	}
	b, err := s.client.AlbumArt(ctx, songPath)
	if err != nil {
		log.Printf("failed to fetch cover art for %s: %v", songPath, err)
		// set zero value for not found
		var perr *mpd.CommandError
		if errors.As(err, &perr) {
			s.db.Put(key, []byte{}, nil)
		}
		return ret
	}
	// save image to random filename.
	ext, err := ext(b)
	if err != nil {
		return ret
	}
	var f *os.File
	if len(filename) == 0 {
		f, err = ioutil.TempFile(s.cacheDir, "*."+ext)
	} else {
		f, err = os.Create(filepath.Join(s.cacheDir, filename+"."+ext))
	}
	if err != nil {
		return ret
	}
	f.Write(b)
	if err := f.Close(); err != nil {
		return ret
	}
	// stores filename to db
	value := filepath.Base(f.Name())
	if err := s.db.Put(key, []byte(value+"?v="+strconv.FormatInt(version, 10)), nil); err != nil {
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
