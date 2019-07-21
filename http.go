package main

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

	"github.com/gorilla/websocket"
	"github.com/meiraka/vv/mpd"
)

// NewHTTPHandler creates MPD http handler
func NewHTTPHandler(ctx context.Context, c *mpd.Client, w *mpd.Watcher) (http.Handler, error) {
	h := &httpHandler{
		client:  c,
		watcher: w,
		jsonCache: &jsonCache{
			event:  make(chan string, 1),
			data:   map[string][]byte{},
			gzdata: map[string][]byte{},
			date:   map[string]time.Time{},
		},
		songCache: &songCache{},
	}
	if err := h.updateLibrary(ctx); err != nil {
		return nil, err
	}
	if err := h.updatePlaylist(ctx); err != nil {
		return nil, err
	}
	if err := h.updateStatus(ctx); err != nil {
		return nil, err
	}
	if err := h.updateCurrentSong(ctx); err != nil {
		return nil, err
	}
	go func() {
		defer close(h.jsonCache.event)
		for e := range w.C {
			ctx := context.Background()
			switch e {
			case "database":
				h.updateLibrary(ctx)
				h.updateStatus(ctx)
			case "playlist":
				h.updatePlaylist(ctx)
			case "player":
				h.updateStatus(ctx)
				h.updateCurrentSong(ctx)
			case "mixer":
			case "options":
			case "update":
				h.updateStatus(ctx)
			}
		}
	}()
	return h.Handle(), nil
}

type httpHandler struct {
	client    *mpd.Client
	watcher   *mpd.Watcher
	jsonCache *jsonCache
	songCache *songCache
	upgrader  websocket.Upgrader
}

type jsonCache struct {
	event  chan string
	data   map[string][]byte
	gzdata map[string][]byte
	date   map[string]time.Time
	mu     sync.RWMutex
}

type songCache struct {
	playlist []mpd.Song
	library  []mpd.Song
	sort     []string
	filters  [][]string
	current  int
	mu       sync.RWMutex
}

func (b *jsonCache) Set(path string, i interface{}) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	d, err := json.Marshal(i)
	if err != nil {
		return err
	}
	if bytes.Equal(b.data[path], d) {
		return nil
	}
	b.data[path] = d
	b.date[path] = time.Now()
	b.sendCh(path)
	return nil
}

func (b *jsonCache) sendCh(e string) {
	select {
	case b.event <- e:
	default:
	}
}

func (b *jsonCache) Get(path string) (data, gzdata []byte, l time.Time) {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.data[path], b.gzdata[path], b.date[path]

}

func (h *httpHandler) updateLibrary(ctx context.Context) error {
	l, err := h.client.ListAllInfo(ctx, "/")
	if err != nil {
		return err
	}
	if err := h.jsonCache.Set("/api/music/library/songs", l); err != nil {
		return err
	}
	return nil
}

func (h *httpHandler) updatePlaylist(ctx context.Context) error {
	l, err := h.client.PlaylistInfo(ctx)
	if err != nil {
		return err
	}
	return h.jsonCache.Set("/api/music/playlist/songs", l)
}

func (h *httpHandler) updateCurrentSong(ctx context.Context) error {
	l, err := h.client.CurrentSong(ctx)
	if err != nil {
		return err
	}
	return h.jsonCache.Set("/api/music/playlist/songs/current", l)
}

func (h *httpHandler) updateStatus(ctx context.Context) error {
	s, err := h.client.Status(ctx)
	if err != nil {
		return err
	}
	v, err := strconv.Atoi(s["volume"])
	if err != nil {
		return fmt.Errorf("vol: %v", err)
	}
	pos, err := strconv.Atoi(s["song"])
	if err != nil {
		return fmt.Errorf("song: %v", err)
	}
	elapsed, err := strconv.ParseFloat(s["elapsed"], 64)
	if err != nil {
		return fmt.Errorf("elapsed: %v", err)
	}
	if err := h.jsonCache.Set("/api/music", &httpMusicStatus{
		Volume:      v,
		Repeat:      s["repeat"] == "1",
		Random:      s["random"] == "1",
		Single:      s["single"] == "1",
		Oneshot:     s["single"] == "oneshot",
		Consume:     s["consume"] == "1",
		State:       s["state"],
		SongElapsed: elapsed,
	}); err != nil {
		return err
	}
	h.songCache.mu.Lock()
	defer h.songCache.mu.Unlock()
	h.songCache.current = pos
	return h.jsonCache.Set("/api/music/playlist", &httpPlaylistInfo{
		Current: h.songCache.current,
		Sort:    h.songCache.sort,
		Filters: h.songCache.filters,
	})
}

type httpMusicStatus struct {
	Volume      int     `json:"volume"`
	Repeat      bool    `json:"repeat"`
	Random      bool    `json:"random"`
	Single      bool    `json:"single"`
	Oneshot     bool    `json:"oneshot"`
	Consume     bool    `json:"consume"`
	State       string  `json:"state"`
	SongElapsed float64 `json:"song_elapsed"`
}

type httpMusicPostStatus struct {
	Volume  *int    `json:"volume"`
	Repeat  *bool   `json:"repeat"`
	Random  *bool   `json:"random"`
	Single  *bool   `json:"single"`
	Oneshot *bool   `json:"oneshot"`
	Consume *bool   `json:"consume"`
	State   *string `json:"state"`
}

func (h *httpHandler) ws(alter http.Handler) http.HandlerFunc {
	subs := make([]chan string, 0, 10)
	var mu sync.Mutex

	go func() {
		for e := range h.jsonCache.event {
			mu.Lock()
			for _, c := range subs {
				select {
				case c <- e:
				default:
				}
			}
			mu.Unlock()
		}
	}()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") != "websocket" {
			alter.ServeHTTP(w, r)
			return
		}
		ws, err := h.upgrader.Upgrade(w, r, nil)
		if err != nil {
			alter.ServeHTTP(w, r)
			return
		}
		c := make(chan string, 2)
		mu.Lock()
		subs = append(subs, c)
		mu.Unlock()
		defer func() {
			mu.Lock()
			defer mu.Unlock()
			n := make([]chan string, len(subs)-1, len(subs)+10)
			diff := 0
			for i, ec := range subs {
				if ec == c {
					diff = -1
				} else {
					n[i+diff] = ec
				}
			}
			subs = n
		}()
		for e := range c {
			err = ws.WriteMessage(websocket.TextMessage, []byte(e))
			if err != nil {
				return
			}

		}

	}
}

type httpPlaylistInfo struct {
	Current int        `json:"current"`
	Sort    []string   `json:"sort"`
	Filters [][]string `json:"filters"`
}

func (h *httpHandler) postStatus(alter http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			alter.ServeHTTP(w, r)
			return
		}
		var s httpMusicPostStatus
		if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
			writeHTTPError(w, 500, err)
			return
		}
		ctx := context.Background()
		var changed bool
		if s.Volume != nil {
			changed = true
			if err := h.client.SetVol(ctx, *s.Volume); err != nil {
				writeHTTPError(w, 500, err)
				return
			}
		}
		if s.Repeat != nil {
			changed = true
			if err := h.client.Repeat(ctx, *s.Repeat); err != nil {
				writeHTTPError(w, 500, err)
				return
			}
		}
		if s.Random != nil {
			changed = true
			if err := h.client.Random(ctx, *s.Random); err != nil {
				writeHTTPError(w, 500, err)
				return
			}
		}
		if s.Single != nil {
			changed = true
			if err := h.client.Single(ctx, *s.Single); err != nil {
				writeHTTPError(w, 500, err)
				return
			}
		}
		if s.Oneshot != nil {
			changed = true
			if err := h.client.OneShot(ctx); err != nil {
				writeHTTPError(w, 500, err)
				return
			}
		}
		if s.Consume != nil {
			changed = true
			if err := h.client.Consume(ctx, *s.Consume); err != nil {
				writeHTTPError(w, 500, err)
				return
			}
		}
		if s.State != nil {
			changed = true
			switch *s.State {
			case "play":
				if err := h.client.Play(ctx, -1); err != nil {
					writeHTTPError(w, 500, err)
					return
				}
			case "pause":
				if err := h.client.Pause(ctx, true); err != nil {
					writeHTTPError(w, 500, err)
					return
				}
			case "next":
				if err := h.client.Next(ctx); err != nil {
					writeHTTPError(w, 500, err)
					return
				}
			case "previous":
				if err := h.client.Previous(ctx); err != nil {
					writeHTTPError(w, 500, err)
					return
				}
			default:
				writeHTTPError(w, 400, fmt.Errorf("unknown state: %s", *s.State))
				return
			}
		}
		if changed {
			h.updateStatus(ctx)
		}
		r.Method = "GET"
		alter.ServeHTTP(w, r)
	}
}

func writeHTTPError(w http.ResponseWriter, status int, err error) {
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	b, _ := json.Marshal(map[string]string{"error": err.Error()})
	w.Header().Add("Content-Length", strconv.Itoa(len(b)))
	w.WriteHeader(status)
	w.Write(b)
}

func (h *httpHandler) Handle() http.Handler {
	m := http.NewServeMux()
	m.Handle("/api/music", h.ws(h.postStatus(h.jsonCacheHandler("/api/music"))))
	m.Handle("/api/music/playlist", h.jsonCacheHandler("/api/music/playlist"))
	m.Handle("/api/music/playlist/songs", h.jsonCacheHandler("/api/music/playlist/songs"))
	m.Handle("/api/music/playlist/songs/current", h.jsonCacheHandler("/api/music/playlist/songs/current"))
	m.Handle("/api/music/library/songs", h.jsonCacheHandler("/api/music/library/songs"))
	return m
}

func (h *httpHandler) jsonCacheHandler(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Method != "HEAD" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		b, gz, date := h.jsonCache.Get(path)
		if !modifiedSince(r, date) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Add("Content-Type", "application/json; charset=utf-8")
		if r.Method == "HEAD" {
			return
		}
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") && gz != nil {
			w.Header().Add("Content-Encoding", "gzip")
			w.Write(gz)
			return
		}
		w.Write(b)
	}
}
