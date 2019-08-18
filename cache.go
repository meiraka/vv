package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type jsonCache struct {
	event  chan string
	data   map[string][]byte
	gzdata map[string][]byte
	date   map[string]time.Time
	mu     sync.RWMutex
}

func newJSONCache() *jsonCache {
	return &jsonCache{
		event:  make(chan string, 100),
		data:   map[string][]byte{},
		gzdata: map[string][]byte{},
		date:   map[string]time.Time{},
	}
}

func (b *jsonCache) Close() {
	b.mu.Lock()
	close(b.event)
	b.mu.Unlock()
}

func (b *jsonCache) Event() <-chan string {
	return b.event
}

func (b *jsonCache) Set(path string, i interface{}) error {
	return b.set(path, i, true)
}

func (b *jsonCache) SetIfModified(path string, i interface{}) error {
	return b.set(path, i, false)
}

func (b *jsonCache) set(path string, i interface{}, force bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	n, err := json.Marshal(i)
	if err != nil {
		return err
	}
	o := b.data[path]
	if force || !bytes.Equal(o, n) {
		b.data[path] = n
		b.date[path] = time.Now().UTC()
		gz, err := makeGZip(n)
		if err == nil {
			b.gzdata[path] = gz
		}
		select {
		case b.event <- path:
		default:
		}
	}
	return nil
}

func (b *jsonCache) Get(path string) (data, gzdata []byte, l time.Time) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.data[path], b.gzdata[path], b.date[path]

}

func (b *jsonCache) Handler(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		b, gz, date := b.Get(path)
		etag := fmt.Sprintf(`"%d.%d"`, date.Unix(), date.Nanosecond())
		if noneMatch(r, etag) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		if !modifiedSince(r, date) {
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
}
