package cover

import (
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/meiraka/vv/internal/mpd"
	"github.com/syndtr/goleveldb/leveldb"
)

// Remote provides http server and song cover art from mpd albumart api.
type Remote struct {
	httpPrefix string
	cacheDir   string
	client     *mpd.Client
	db         *leveldb.DB       // song path - img name
	url2img    map[string]string // url path - img path
	mu         sync.RWMutex
}

// NewRemote initializes Remote with cacheDir.
func NewRemote(httpPrefix string, client *mpd.Client, cacheDir string) (*Remote, error) {
	if err := os.MkdirAll(cacheDir, 0766); err != nil {
		return nil, err
	}
	db, err := leveldb.OpenFile(filepath.Join(cacheDir, "db2"), nil)
	if err != nil {
		return nil, err
	}

	s := &Remote{
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
func (s *Remote) Rescan(ctx context.Context, song map[string][]string) error {
	k, ok := s.songPath(song)
	if !ok {
		return nil
	}
	return s.updateCache(ctx, k)
}

func (s *Remote) songPath(song map[string][]string) (string, bool) {
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
func (s *Remote) Close() error {
	s.db.Close()
	return nil
}

// ServeHTTP serves local cover art with httpPrefix
func (s *Remote) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	path, ok := s.url2img[r.URL.Path]
	s.mu.RUnlock()
	if !ok {
		http.NotFound(w, r)
		return
	}
	serveImage(path, w, r)
}

func (s *Remote) getURLPath(songPath string) ([]string, error) {
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

func (s *Remote) updateCache(ctx context.Context, songPath string) error {
	key := []byte(path.Dir(songPath))

	filename := ""
	var version int64
	if name, err := s.db.Get(key, nil); err == leveldb.ErrNotFound {
	} else if err != nil {
		return err
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
	b, err := s.client.AlbumArt(ctx, songPath)
	if err != nil {
		// set zero value for not found
		var perr *mpd.CommandError
		if errors.As(err, &perr) {
			s.db.Put(key, []byte{}, nil)
			return nil
		}
		return err
	}
	// save image to random filename.
	ext, err := ext(b)
	if err != nil {
		return err
	}
	var f *os.File
	if len(filename) == 0 {
		f, err = ioutil.TempFile(s.cacheDir, "*."+ext)
	} else {
		f, err = os.Create(filepath.Join(s.cacheDir, filename+"."+ext))
	}
	if err != nil {
		return err
	}
	f.Write(b)
	if err := f.Close(); err != nil {
		return err
	}
	// stores filename to db
	value := filepath.Base(f.Name())
	if err := s.db.Put(key, []byte(value+"?v="+strconv.FormatInt(version, 10)), nil); err != nil {
		os.Remove(filepath.Join(s.cacheDir, value))
		return err
	}
	s.mu.Lock()
	s.url2img[path.Join(s.httpPrefix, value)] = filepath.Join(s.cacheDir, value)
	s.mu.Unlock()
	_, err = s.getURLPath(songPath)
	return err
}

// GetURLs returns cover path for m
func (s *Remote) GetURLs(m map[string][]string) ([]string, bool) {
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
