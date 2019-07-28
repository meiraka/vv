package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/meiraka/vv/mpd"
)

// HTTPHandlerConfig holds HTTPHandler config
type HTTPHandlerConfig struct {
	BackgroundTimeout time.Duration
}

func (c *HTTPHandlerConfig) backgroundTimeout() time.Duration {
	if c.BackgroundTimeout == 0 {
		return 30 * time.Second
	}
	return c.BackgroundTimeout
}

func addHTTPPrefix(m map[string][]string) map[string][]string {
	if v, ok := m["cover"]; ok {
		for i := range v {
			v[i] = path.Join("/api/music/storage/", v[i])
		}
	}
	return m
}

// NewHTTPHandler creates MPD http handler
func (c HTTPHandlerConfig) NewHTTPHandler(ctx context.Context, cl *mpd.Client, w *mpd.Watcher, t []TagAdder) (http.Handler, error) {
	h := &httpHandler{
		client:  cl,
		watcher: w,
		jsonCache: &jsonCache{
			event:  make(chan string, 1),
			data:   map[string][]byte{},
			gzdata: map[string][]byte{},
			date:   map[string]time.Time{},
		},
		tagger:    append(t, TagAdderFunc(addHTTPPrefix)),
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
		timeout := c.backgroundTimeout()
		for e := range w.C {
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
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
			cancel()
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
	tagger    []TagAdder
}

type jsonCache struct {
	event  chan string
	data   map[string][]byte
	gzdata map[string][]byte
	date   map[string]time.Time
	mu     sync.RWMutex
}

func (b *jsonCache) Set(path string, i interface{}, force bool) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	n, err := json.Marshal(i)
	if err != nil {
		return err
	}
	o := b.data[path]
	if force || !bytes.Equal(o, n) {
		b.data[path] = n
		b.date[path] = time.Now()
		b.sendCh(path)
	}
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

type songCache struct {
	playlist []Song
	library  []Song
	sort     []string
	filters  [][]string
	current  int
	mu       sync.Mutex
}

func (h *httpHandler) convSong(s map[string][]string) Song {
	for i := range h.tagger {
		s = h.tagger[i].AddTags(s)
	}
	return Song(s)
}

func (h *httpHandler) convSongs(s []map[string][]string) []Song {
	ret := make([]Song, len(s))
	for i := range s {
		ret[i] = h.convSong(s[i])
	}
	return ret
}

func (h *httpHandler) updateLibrary(ctx context.Context) error {
	l, err := h.client.ListAllInfo(ctx, "/")
	if err != nil {
		return err
	}
	v := h.convSongs(l)
	if err := h.jsonCache.Set("/api/music/library/songs", v, true); err != nil {
		return err
	}
	h.songCache.mu.Lock()
	h.songCache.library = v
	h.songCache.mu.Unlock()

	return nil
}

func (h *httpHandler) updatePlaylist(ctx context.Context) error {
	l, err := h.client.PlaylistInfo(ctx)
	if err != nil {
		return err
	}
	v := h.convSongs(l)
	if err := h.jsonCache.Set("/api/music/playlist/songs", v, true); err != nil {
		return err
	}

	h.songCache.mu.Lock()
	h.songCache.playlist = v
	h.songCache.mu.Unlock()

	return nil
}

func (h *httpHandler) updateCurrentSong(ctx context.Context) error {
	l, err := h.client.CurrentSong(ctx)
	if err != nil {
		return err
	}
	return h.jsonCache.Set("/api/music/playlist/songs/current", h.convSong(l), false)
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
	}, false); err != nil {
		return err
	}
	h.songCache.mu.Lock()
	defer h.songCache.mu.Unlock()
	h.songCache.current = pos
	return h.jsonCache.Set("/api/music/playlist", &httpPlaylistInfo{
		Current: h.songCache.current,
		Sort:    h.songCache.sort,
		Filters: h.songCache.filters,
	}, false)
}

func (h *httpHandler) playlistPost(alter http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			alter.ServeHTTP(w, r)
			return
		}
		var req httpPlaylistInfo
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeHTTPError(w, 400, err)
			return
		}

		if req.Filters == nil || req.Sort == nil {
			writeHTTPError(w, 400, errors.New("filters and sort fields are required"))
			return
		}

		h.songCache.mu.Lock()
		library, filters, newpos := SortSongs(h.songCache.library, req.Sort, req.Filters, 9999, req.Current)
		playlist := h.songCache.playlist
		var update bool
		if len(library) != len(playlist) {
			update = true
		} else {
			for i := range library {
				n := library[i]["file"][0]
				o := playlist[i]["file"][0]
				if n != o {
					update = true
					break
				}
			}
		}
		cl := h.client.BeginCommandList()
		cl.Clear()
		for i := range library {
			cl.Add(library[i]["file"][0])
		}
		cl.Play(newpos)
		h.songCache.mu.Unlock()
		if !update {
			ctx := r.Context()
			if err := h.client.Play(ctx, newpos); err != nil {
				writeHTTPError(w, 500, err)
				return
			}
			h.songCache.mu.Lock()
			h.songCache.sort = req.Sort
			h.songCache.filters = filters
			h.songCache.mu.Unlock()
			h.updateStatus(ctx)
			r.Method = http.MethodGet
			alter.ServeHTTP(w, r)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if err := cl.End(ctx); err != nil {
				return
			}
			h.songCache.mu.Lock()
			h.songCache.sort = req.Sort
			h.songCache.filters = filters
			h.songCache.mu.Unlock()
		}()

	}
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

type httpPlaylistInfo struct {
	Current int        `json:"current"`
	Sort    []string   `json:"sort"`
	Filters [][]string `json:"filters"`
}

func (h *httpHandler) statusWebSocket(alter http.Handler) http.HandlerFunc {
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

func (h *httpHandler) statusPost(alter http.Handler) http.HandlerFunc {
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
		ctx := r.Context()
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
			var err error
			switch *s.State {
			case "play":
				err = h.client.Play(ctx, -1)
			case "pause":
				err = h.client.Pause(ctx, true)
			case "next":
				err = h.client.Next(ctx)
			case "previous":
				err = h.client.Previous(ctx)
			default:
				writeHTTPError(w, 400, fmt.Errorf("unknown state: %s", *s.State))
				return
			}
			if err != nil {
				writeHTTPError(w, 500, err)
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
	m.Handle("/api/music", h.statusWebSocket(h.statusPost(h.jsonCacheHandler("/api/music"))))
	m.Handle("/api/music/playlist", h.playlistPost(h.jsonCacheHandler("/api/music/playlist")))
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
