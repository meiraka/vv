package images

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/meiraka/vv/internal/mpd"
	bolt "go.etcd.io/bbolt"
)

var (
	bucketDirToURL   = []byte("dir")
	bucketURLToLocal = []byte("url")
)

// Remote provides http server and song cover art from mpd albumart api.
type Remote struct {
	httpPrefix string
	cacheDir   string
	client     *mpd.Client
	db         *bolt.DB
}

// NewRemote initializes Remote with cacheDir.
func NewRemote(httpPrefix string, client *mpd.Client, cacheDir string) (*Remote, error) {
	if err := os.MkdirAll(cacheDir, 0766); err != nil {
		return nil, err
	}
	db, err := bolt.Open(filepath.Join(cacheDir, "db3"), 0666, nil)
	if err != nil {
		return nil, err
	}

	s := &Remote{
		httpPrefix: httpPrefix,
		cacheDir:   cacheDir,
		client:     client,
		db:         db,
	}
	if err := db.Update(func(tx *bolt.Tx) error {
		for _, s := range [][]byte{bucketDirToURL, bucketURLToLocal} {
			_, err := tx.CreateBucketIfNotExists(s)
			if err != nil {
				return fmt.Errorf("create bucket: %s", err)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
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

var errNotFound = errors.New("not found")

// ServeHTTP serves local cover art with httpPrefix
func (s *Remote) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// strip httpPrefix
	p := s.httpPrefix
	if p[len(p)-1] != '/' {
		p += "/"
	}
	if !strings.HasPrefix(r.URL.Path, p) {
		http.NotFound(w, r)
		return
	}
	urlName := r.URL.Path[len(p):]

	// get local filename
	var localName []byte
	if err := s.db.View(func(tx *bolt.Tx) error {
		localName = tx.Bucket(bucketURLToLocal).Get([]byte(urlName))
		return nil
	}); err != nil {
		http.NotFound(w, r)
		return
	}
	if localName == nil {
		http.NotFound(w, r)
		return
	}

	serveImage(filepath.Join(s.cacheDir, string(localName)), w, r)
}

func (s *Remote) getURLPath(songPath string) ([]string, error) {
	key := []byte(path.Dir(songPath))

	var url []byte
	if err := s.db.View(func(tx *bolt.Tx) error {
		url = tx.Bucket(bucketDirToURL).Get(key)
		return nil
	}); err != nil {
		return nil, err
	}
	if url == nil {
		return nil, errNotFound
	}
	if len(url) == 0 {
		return []string{}, nil
	}
	return []string{path.Join(s.httpPrefix, string(url))}, nil
}

func (s *Remote) updateCache(ctx context.Context, songPath string) error {
	b, err := s.client.AlbumArt(ctx, songPath)
	if err != nil {
		// set zero value for not found
		if errors.Is(err, mpd.ErrNoExist) {
			return s.setEmptyCache(ctx, songPath)
		}
		return err
	}
	return s.setCache(ctx, songPath, b)
}

func (s *Remote) setEmptyCache(ctx context.Context, songPath string) error {
	key := []byte(path.Dir(songPath))
	return s.db.Update(func(tx *bolt.Tx) error {
		tx.Bucket(bucketDirToURL).Put(key, []byte(""))
		return nil
	})
}

func (s *Remote) setCache(ctx context.Context, songPath string, b []byte) error {
	key := []byte(path.Dir(songPath))
	ext, err := ext(b)
	if err != nil {
		return err
	}

	// fetch old url
	var url []byte
	if err := s.db.View(func(tx *bolt.Tx) error {
		url = tx.Bucket(bucketDirToURL).Get(key)
		return nil
	}); err != nil {
		return err
	}

	var filename string
	var version int64
	if len(url) != 0 {
		// get old version
		url := string(url)
		i := strings.LastIndex(url, "=")
		if i > 0 {
			if v, err := strconv.ParseInt(url[i+1:], 10, 64); err == nil {
				version = v
				if version == 9223372036854775807 {
					version = 0
				} else {
					version = version + 1
				}
			}
		}
		// update ext
		i = strings.LastIndex(url, ".")
		if i < 0 {
			i = len(url)
		}
		filename = url[0:i] + "." + ext
	}

	// save image to random filename.
	var f *os.File
	if len(filename) == 0 {
		f, err = ioutil.TempFile(s.cacheDir, "*."+ext)
	} else {
		// compare to old binary
		path := filepath.Join(s.cacheDir, filename)
		if _, err := os.Stat(path); err == nil {
			ob, err := ioutil.ReadFile(path)
			if err == nil {
				if bytes.Equal(b, ob) {
					// same binary
					return nil
				}
			}
		}
		f, err = os.Create(path)
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
	return s.db.Update(func(tx *bolt.Tx) error {
		tx.Bucket(bucketDirToURL).Put(key, []byte(value+"?v="+strconv.FormatInt(version, 10)))
		tx.Bucket(bucketURLToLocal).Put([]byte(value), []byte(value))
		return nil
	})
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
