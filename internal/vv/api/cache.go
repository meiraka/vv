package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/meiraka/vv/internal/gzip"
	"github.com/meiraka/vv/internal/request"
)

type cache struct {
	changed  chan struct{}
	changedB bool
	json     []byte
	gzjson   []byte
	date     time.Time
	mu       sync.RWMutex
}

func newCache(i interface{}) (*cache, error) {
	b, gz, err := cacheBinary(i)
	if err != nil {
		return nil, err
	}
	c := &cache{
		changed:  make(chan struct{}, 1),
		changedB: true,
		json:     b,
		gzjson:   gz,
		date:     time.Now().UTC(),
	}
	return c, nil
}

func (c *cache) Close() {
	c.mu.Lock()
	if c.changedB {
		close(c.changed)
		c.changedB = false
	}
	c.mu.Unlock()
}

func (c *cache) Changed() <-chan struct{} {
	return c.changed
}

func (c *cache) Set(i interface{}) error {
	_, err := c.set(i, true)
	return err
}

func (c *cache) SetIfModified(i interface{}) (changed bool, err error) {
	return c.set(i, false)
}

func (c *cache) get() ([]byte, []byte, time.Time) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.json, c.gzjson, c.date
}

func (c *cache) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.mu.RLock()
	b, gz, date := c.json, c.gzjson, c.date
	c.mu.RUnlock()
	etag := fmt.Sprintf(`"%d.%d"`, date.Unix(), date.Nanosecond())
	if request.NoneMatch(r, etag) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	if !request.ModifiedSince(r, date) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Add("Cache-Control", "max-age=0")
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	w.Header().Add("Last-Modified", date.Format(http.TimeFormat))
	w.Header().Add("Vary", "Accept-Encoding")
	w.Header().Add("ETag", etag)
	status := http.StatusOK
	if getUpdateTime(r).After(date) {
		status = http.StatusAccepted
	}
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") && gz != nil {
		w.Header().Add("Content-Encoding", "gzip")
		w.Header().Add("Content-Length", strconv.Itoa(len(gz)))
		w.WriteHeader(status)
		w.Write(gz)
		return
	}
	w.Header().Add("Content-Length", strconv.Itoa(len(b)))
	w.WriteHeader(status)
	w.Write(b)
}

func (c *cache) set(i interface{}, force bool) (bool, error) {
	n, gz, err := cacheBinary(i)
	if err != nil {
		return false, err
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	o := c.json
	if force || !bytes.Equal(o, n) {
		c.json = n
		c.date = time.Now().UTC()
		c.gzjson = gz
		if c.changedB {
			select {
			case c.changed <- struct{}{}:
			default:
			}
		}
		return true, nil
	}
	return false, nil
}

func cacheBinary(i interface{}) ([]byte, []byte, error) {
	n, err := json.Marshal(i)
	if err != nil {
		return nil, nil, err
	}
	gz, err := gzip.Encode(n)
	if err != nil {
		return n, nil, nil
	}
	return n, gz, nil
}

type httpContextKey string

const httpUpdateTime = httpContextKey("updateTime")

func getUpdateTime(r *http.Request) time.Time {
	if v := r.Context().Value(httpUpdateTime); v != nil {
		if i, ok := v.(time.Time); ok {
			return i
		}
	}
	return time.Time{}
}

func setUpdateTime(r *http.Request, u time.Time) *http.Request {
	ctx := context.WithValue(r.Context(), httpUpdateTime, u)
	return r.WithContext(ctx)
}
