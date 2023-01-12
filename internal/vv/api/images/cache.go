package images

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
)

var (
	bucketKeyToURL   = []byte("key2url")
	bucketURLToLocal = []byte("url2local")
	bucketKeyToReqID = []byte("key2req")
)

type cache struct {
	cacheDir string
	db       *bolt.DB
}

func newCache(cacheDir string) (*cache, error) {
	if err := os.MkdirAll(cacheDir, 0766); err != nil {
		return nil, err
	}
	db, err := bolt.Open(filepath.Join(cacheDir, "db4"), 0666, &bolt.Options{Timeout: time.Second})
	if err != nil {
		if errors.Is(err, bolt.ErrTimeout) {
			return nil, fmt.Errorf("obtain cache lock: %w", err)
		}
		return nil, err
	}

	if err := db.Update(func(tx *bolt.Tx) error {
		for _, s := range [][]byte{bucketKeyToURL, bucketURLToLocal, bucketKeyToReqID} {
			_, err := tx.CreateBucketIfNotExists(s)
			if err != nil {
				return fmt.Errorf("create bucket: %s", err)
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return &cache{
		cacheDir: cacheDir,
		db:       db,
	}, nil
}

// GetLocalPath returns local file path from url path.
func (c *cache) GetLocalPath(r string) (string, bool) {
	var localName []byte
	if err := c.db.View(func(tx *bolt.Tx) error {
		localName = tx.Bucket(bucketURLToLocal).Get([]byte(r))
		return nil
	}); err != nil {
		return "", false
	}
	if localName == nil {
		return "", false
	}
	return filepath.Join(c.cacheDir, string(localName)), true
}

// GetURL returns url string from song key.
func (c *cache) GetURL(key string) (string, bool) {
	var url []byte
	if err := c.db.View(func(tx *bolt.Tx) error {
		url = tx.Bucket(bucketKeyToURL).Get([]byte(key))
		return nil
	}); err != nil {
		return "", false
	}
	if url == nil {
		return "", false
	}
	if len(url) == 0 {
		return "", true
	}
	return string(url), true
}

func (c *cache) GetLastRequestID(key string) (string, bool) {
	var b []byte
	if err := c.db.View(func(tx *bolt.Tx) error {
		b = tx.Bucket(bucketKeyToReqID).Get([]byte(key))
		return nil
	}); err != nil {
		return "", false
	}
	if b == nil {
		return "", false
	}
	return string(b), true
}

// Set updates image cache and reqid by key.
func (c *cache) Set(key, reqid string, b []byte) (err error) {
	bkey := []byte(key)
	ext, err := ext(b)
	if err != nil {
		return err
	}

	// fetch old url
	var url []byte
	if err := c.db.View(func(tx *bolt.Tx) error {
		url = tx.Bucket(bucketKeyToURL).Get(bkey)
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
		f, err = os.CreateTemp(c.cacheDir, "*."+ext)
	} else {
		// compare to old binary
		path := filepath.Join(c.cacheDir, filename)
		if _, err := os.Stat(path); err == nil {
			ob, err := os.ReadFile(path)
			if err == nil {
				if bytes.Equal(b, ob) {
					// same binary
					return c.db.Update(func(tx *bolt.Tx) error {
						tx.Bucket(bucketKeyToReqID).Put(bkey, []byte(reqid))
						return nil
					})
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
	return c.db.Update(func(tx *bolt.Tx) error {
		tx.Bucket(bucketKeyToURL).Put(bkey, []byte(value+"?v="+strconv.FormatInt(version, 10)))
		tx.Bucket(bucketURLToLocal).Put([]byte(value), []byte(value))
		tx.Bucket(bucketKeyToReqID).Put(bkey, []byte(reqid))
		return nil
	})
}

// Set updates image cache and reqid by key.
func (c *cache) SetEmpty(key, reqid string) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		bkey := []byte(key)
		tx.Bucket(bucketKeyToURL).Put(bkey, []byte(""))
		tx.Bucket(bucketKeyToReqID).Put(bkey, []byte(reqid))
		return nil
	})
}

// Close finalizes cache db, coroutines.
func (c *cache) Close() error {
	return c.db.Close()
}
