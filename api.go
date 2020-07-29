package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/meiraka/vv/internal/mpd"
	"github.com/meiraka/vv/internal/songs"
	"github.com/meiraka/vv/internal/songs/cover"
)

// APIConfig holds HTTPHandler config
type APIConfig struct {
	BackgroundTimeout time.Duration
	MusicDirectory    string
	Cover             []string
}

type coverSearcher interface {
	ServeHTTP(w http.ResponseWriter, r *http.Request)
	AddTags(map[string][]string) map[string][]string
}

// NewAPIHandler creates json api handler.
func (c APIConfig) NewAPIHandler(ctx context.Context, cl *mpd.Client, w *mpd.Watcher) (http.Handler, error) {
	if c.BackgroundTimeout == 0 {
		c.BackgroundTimeout = 30 * time.Second
	}
	covers := map[string]coverSearcher{}
	if len(c.MusicDirectory) != 0 {
		if len(c.Cover) == 0 {
			c.Cover = []string{"cover.jpg", "cover.jpeg", "cover.png", "cover.gif", "cover.bmp"}
		}
		var err error
		covers["/api/music/images/"], err = cover.NewLocalSearcher("/api/music/images/", c.MusicDirectory, c.Cover)
		if err != nil {
			return nil, err
		}
	}
	h := &api{
		config:    &c,
		client:    cl,
		watcher:   w,
		jsonCache: newJSONCache(),
		covers:    covers,
	}
	if err := h.updateVersion(); err != nil {
		return nil, err
	}
	all := []func(context.Context) error{h.updateLibrarySongs, h.updatePlaylistSongs, h.updateStatus, h.updateCurrentSong, h.updateOutputs, h.updateStats}
	for _, v := range all {
		if err := v(ctx); err != nil {
			return nil, err
		}
	}
	go func() {
		defer h.jsonCache.Close()
		for e := range w.Event() {
			ctx, cancel := context.WithTimeout(context.Background(), c.BackgroundTimeout)
			switch e {
			case "reconnect":
				h.updateVersion()
				for _, v := range all {
					v(ctx)
				}
			case "database":
				h.updateLibrarySongs(ctx)
				h.updateStatus(ctx)
				h.updateStats(ctx)
			case "playlist":
				h.updatePlaylistSongs(ctx)
			case "player":
				h.updateStatus(ctx)
				h.updateCurrentSong(ctx)
				h.updateStats(ctx)
			case "mixer":
				h.updateStatus(ctx)
			case "options":
				h.updateStatus(ctx)
			case "update":
				h.updateStatus(ctx)
			case "output":
				h.updateOutputs(ctx)
			}
			cancel()
		}
	}()
	return h.handle(), nil
}

type api struct {
	config    *APIConfig
	client    *mpd.Client
	watcher   *mpd.Watcher
	jsonCache *jsonCache
	upgrader  websocket.Upgrader
	covers    map[string]coverSearcher

	mu          sync.Mutex
	playlist    []map[string][]string
	library     []map[string][]string
	librarySort []map[string][]string
	sort        []string
	filters     [][]string
	current     int
}

func (h *api) handle() http.HandlerFunc {
	version := h.jsonCache.Handler("/api/version")
	music := h.statusWebSocket(h.statusHandler())
	musicStats := h.jsonCache.Handler("/api/music/stats")
	musicPlaylist := h.playlistHandler()
	musicPlaylistSongs := h.jsonCache.Handler("/api/music/playlist/songs")
	musicPlaylistSongsCurrent := h.jsonCache.Handler("/api/music/playlist/songs/current")
	musicLibrary := h.libraryHandler()
	musicLibrarySongs := h.jsonCache.Handler("/api/music/library/songs")
	musicOutputs := h.outputHandler()
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/version":
			version(w, r)
		case "/api/music":
			music(w, r)
		case "/api/music/stats":
			musicStats(w, r)
		case "/api/music/playlist":
			musicPlaylist(w, r)
		case "/api/music/playlist/songs":
			musicPlaylistSongs(w, r)
		case "/api/music/playlist/songs/current":
			musicPlaylistSongsCurrent(w, r)
		case "/api/music/library":
			musicLibrary(w, r)
		case "/api/music/library/songs":
			musicLibrarySongs(w, r)
		case "/api/music/outputs":
			musicOutputs(w, r)
		default:
			for k, v := range h.covers {
				if strings.HasPrefix(r.URL.Path, k) {
					v.ServeHTTP(w, r)
					return
				}
			}
			http.NotFound(w, r)
		}
	}
}

func (h *api) convSong(s map[string][]string) map[string][]string {
	s = songs.AddTags(s)
	for _, v := range h.covers {
		s = v.AddTags(s)
	}
	return s
}

func (h *api) convSongs(s []map[string][]string) []map[string][]string {
	ret := make([]map[string][]string, len(s))
	for i := range s {
		ret[i] = h.convSong(s[i])
	}
	return ret
}

type httpAPIVersion struct {
	App string `json:"app"`
	Go  string `json:"go"`
	MPD string `json:"mpd"`
}

func (h *api) updateVersion() error {
	goVersion := fmt.Sprintf("%s %s %s", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	return h.jsonCache.SetIfModified("/api/version", &httpAPIVersion{App: version, Go: goVersion, MPD: h.client.Version()})
}

type httpPlaylistInfo struct {
	Current int        `json:"current"`
	Sort    []string   `json:"sort,omitempty"`
	Filters [][]string `json:"filters,omitempty"`
	Must    int        `json:"must,omitempty"`
}

func (h *api) updatePlaylist() error {
	return h.jsonCache.SetIfModified("/api/music/playlist", &httpPlaylistInfo{
		Current: h.current,
		Sort:    h.sort,
		Filters: h.filters,
	})
}

func (h *api) playlistHandler() http.HandlerFunc {
	sem := make(chan struct{}, 1)
	sem <- struct{}{}
	fallback := h.jsonCache.Handler("/api/music/playlist")
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			fallback.ServeHTTP(w, r)
			return
		}
		var req httpPlaylistInfo
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeHTTPError(w, http.StatusBadRequest, err)
			return
		}

		if req.Filters == nil || req.Sort == nil {
			writeHTTPError(w, http.StatusBadRequest, errors.New("filters and sort fields are required"))
			return
		}

		select {
		case <-sem:
		default:
			// TODO: switch to better status code
			writeHTTPError(w, http.StatusServiceUnavailable, errors.New("updating playlist"))
			return
		}
		defer func() { sem <- struct{}{} }()

		h.mu.Lock()
		librarySort, filters, newpos := songs.WeakFilterSort(h.library, req.Sort, req.Filters, req.Must, 9999, req.Current)
		update := !songs.SortEqual(h.playlist, librarySort)
		cl := &mpd.CommandList{}
		cl.Clear()
		for i := range librarySort {
			cl.Add(librarySort[i]["file"][0])
		}
		cl.Play(newpos)
		h.sort = req.Sort
		h.filters = filters
		h.librarySort = librarySort
		h.mu.Unlock()
		if !update {
			now := time.Now().UTC()
			ctx := r.Context()
			if err := h.client.Play(ctx, newpos); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				h.mu.Lock()
				h.sort = nil
				h.filters = nil
				h.librarySort = nil
				h.mu.Unlock()
				return
			}
			r.Method = http.MethodGet
			fallback.ServeHTTP(w, setUpdateTime(r, now))
			return
		}
		r.Method = http.MethodGet
		fallback.ServeHTTP(w, setUpdateTime(r, time.Now().UTC()))
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), h.config.BackgroundTimeout)
			defer cancel()
			select {
			case <-sem:
			case <-ctx.Done():
				return
			}
			defer func() { sem <- struct{}{} }()
			if err := h.client.ExecCommandList(ctx, cl); err != nil {
				h.mu.Lock()
				h.sort = nil
				h.filters = nil
				h.librarySort = nil
				h.mu.Unlock()
				return
			}
		}()
	}
}

func (h *api) updatePlaylistSongs(ctx context.Context) error {
	l, err := h.client.PlaylistInfo(ctx)
	if err != nil {
		return err
	}
	v := h.convSongs(l)
	// force update to skip []byte compare
	if err := h.jsonCache.Set("/api/music/playlist/songs", v); err != nil {
		return err
	}

	h.mu.Lock()
	h.playlist = v
	if h.sort != nil && !songs.SortEqual(h.playlist, h.librarySort) {
		h.sort = nil
		h.filters = nil
		h.librarySort = nil
		h.updatePlaylist()
	}
	h.mu.Unlock()

	return err
}

func (h *api) updateCurrentSong(ctx context.Context) error {
	l, err := h.client.CurrentSong(ctx)
	if err != nil {
		return err
	}
	return h.jsonCache.SetIfModified("/api/music/playlist/songs/current", h.convSong(l))
}

type httpLibraryInfo struct {
	Updating bool `json:"updating"`
}

func (h *api) libraryHandler() http.HandlerFunc {
	fallback := h.jsonCache.Handler("/api/music/library")
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			fallback.ServeHTTP(w, r)
			return
		}
		var req httpLibraryInfo
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeHTTPError(w, http.StatusBadRequest, err)
			return
		}
		if !req.Updating {
			writeHTTPError(w, http.StatusBadRequest, errors.New("requires updating=true"))
			return
		}
		ctx := r.Context()
		now := time.Now().UTC()
		if _, err := h.client.Update(ctx, ""); err != nil {
			writeHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		r.Method = http.MethodGet
		fallback.ServeHTTP(w, setUpdateTime(r, now))
	}
}

func (h *api) updateLibrarySongs(ctx context.Context) error {
	l, err := h.client.ListAllInfo(ctx, "/")
	if err != nil {
		return err
	}
	v := h.convSongs(l)
	// force update to skip []byte compare
	if err := h.jsonCache.Set("/api/music/library/songs", v); err != nil {
		return err
	}
	h.mu.Lock()
	h.library = v
	h.sort = nil
	h.filters = nil
	h.librarySort = nil
	h.updatePlaylist()

	h.mu.Unlock()
	return nil
}

// status

type httpMusicStatus struct {
	Volume      *int     `json:"volume,omitempty"`
	Repeat      *bool    `json:"repeat,omitempty"`
	Random      *bool    `json:"random,omitempty"`
	Single      *bool    `json:"single,omitempty"`
	Oneshot     *bool    `json:"oneshot,omitempty"`
	Consume     *bool    `json:"consume,omitempty"`
	State       *string  `json:"state,omitempty"`
	SongElapsed *float64 `json:"song_elapsed,omitempty"`
}

func (h *api) updateStatus(ctx context.Context) error {
	s, err := h.client.Status(ctx)
	if err != nil {
		return err
	}
	var volume *int
	v, err := strconv.Atoi(s["volume"])
	if err == nil && v >= 0 {
		volume = &v
	}
	pos, err := strconv.Atoi(s["song"])
	if err != nil {
		pos = 0
	}
	elapsed, err := strconv.ParseFloat(s["elapsed"], 64)
	if err != nil {
		elapsed = 0
		// return fmt.Errorf("elapsed: %v", err)
	}
	// force update to Last-Modified header to calc current SongElapsed
	// TODO: add millisec update time to JSON
	if err := h.jsonCache.Set("/api/music", &httpMusicStatus{
		Volume:      volume,
		Repeat:      boolPtr(s["repeat"] == "1"),
		Random:      boolPtr(s["random"] == "1"),
		Single:      boolPtr(s["single"] == "1"),
		Oneshot:     boolPtr(s["single"] == "oneshot"),
		Consume:     boolPtr(s["consume"] == "1"),
		State:       stringPtr(s["state"]),
		SongElapsed: &elapsed,
	}); err != nil {
		return err
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.current = pos
	if err := h.updatePlaylist(); err != nil {
		return err
	}
	_, updating := s["updating_db"]
	return h.jsonCache.SetIfModified("/api/music/library", &httpLibraryInfo{
		Updating: updating,
	})
}

func (h *api) statusHandler() http.HandlerFunc {
	fallback := h.jsonCache.Handler("/api/music")
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			fallback.ServeHTTP(w, r)
			return
		}
		var s httpMusicStatus
		if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
			writeHTTPError(w, http.StatusBadRequest, err)
			return
		}
		ctx := r.Context()
		now := time.Now().UTC()
		changed := false
		if s.Volume != nil {
			if err := h.client.SetVol(ctx, *s.Volume); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			changed = true
		}
		if s.Repeat != nil {
			if err := h.client.Repeat(ctx, *s.Repeat); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			changed = true
		}
		if s.Random != nil {
			if err := h.client.Random(ctx, *s.Random); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			changed = true
		}
		if s.Single != nil {
			if err := h.client.Single(ctx, *s.Single); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			changed = true
		}
		if s.Oneshot != nil {
			if err := h.client.OneShot(ctx); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			changed = true
		}
		if s.Consume != nil {
			if err := h.client.Consume(ctx, *s.Consume); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			changed = true
		}
		if s.SongElapsed != nil {
			if err := h.client.SeekCur(ctx, *s.SongElapsed); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			changed = true
		}
		if s.State != nil {
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
				writeHTTPError(w, http.StatusBadRequest, fmt.Errorf("unknown state: %s", *s.State))
				return
			}
			if err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			changed = true
		}
		r.Method = "GET"
		if changed {
			r = setUpdateTime(r, now)
		}
		fallback.ServeHTTP(w, r)
	}
}

func (h *api) statusWebSocket(fallback http.Handler) http.HandlerFunc {
	subs := make([]chan string, 0, 10)
	var mu sync.Mutex

	go func() {
		for e := range h.jsonCache.Event() {
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
			fallback.ServeHTTP(w, r)
			return
		}
		ws, err := h.upgrader.Upgrade(w, r, nil)
		if err != nil {
			fallback.ServeHTTP(w, r)
			return
		}
		c := make(chan string, 100)
		mu.Lock()
		subs = append(subs, c)
		mu.Unlock()
		defer func() {
			mu.Lock()
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
			close(c)
			ws.Close()
			mu.Unlock()
		}()
		if err := ws.WriteMessage(websocket.TextMessage, []byte("ok")); err != nil {
			return
		}
		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			defer cancel()
			for {
				_, _, err := ws.ReadMessage()
				if err != nil {
					return
				}
			}
		}()
		for {
			select {
			case <-ctx.Done():
				return
			case e, ok := <-c:
				if !ok {
					return
				}
				if err := ws.WriteMessage(websocket.TextMessage, []byte(e)); err != nil {
					return
				}
			case <-time.After(time.Second * 5):
				if err := ws.WriteMessage(websocket.TextMessage, []byte("ping")); err != nil {
					return
				}
			}

		}

	}
}

type httpOutput struct {
	Name      string `json:"name"`
	Plugin    string `json:"plugin,omitempty"`
	Enabled   *bool  `json:"enabled"`
	Attribute string `json:"attribute,omitempty"` // TODO fix type
}

func (h *api) updateOutputs(ctx context.Context) error {
	l, err := h.client.Outputs(ctx)
	if err != nil {
		return err
	}
	data := make(map[string]*httpOutput, len(l))
	for _, v := range l {
		data[v["outputid"]] = &httpOutput{
			Name:      v["outputname"],
			Plugin:    v["plugin"],
			Enabled:   boolPtr(v["outputenabled"] == "1"),
			Attribute: v["attribute"],
		}
	}
	return h.jsonCache.SetIfModified("/api/music/outputs", data)
}

func (h *api) outputHandler() http.HandlerFunc {
	fallback := h.jsonCache.Handler("/api/music/outputs")
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			fallback.ServeHTTP(w, r)
			return
		}
		var req map[string]*httpOutput
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeHTTPError(w, http.StatusBadRequest, err)
			return
		}
		ctx := r.Context()
		now := time.Now().UTC()
		for k, v := range req {
			if v.Enabled != nil {
				var err error
				r = setUpdateTime(r, now)
				if *v.Enabled {
					err = h.client.EnableOutput(ctx, k)
				} else {
					err = h.client.DisableOutput(ctx, k)
				}
				if err != nil {
					writeHTTPError(w, http.StatusInternalServerError, err)
					return
				}
			}
		}
		r.Method = http.MethodGet
		fallback.ServeHTTP(w, r)

	}
}

type httpMusicStats struct {
	Uptime          int `json:"uptime"`
	Playtime        int `json:"playtime"`
	Artists         int `json:"artists"`
	Albums          int `json:"albums"`
	Songs           int `json:"songs"`
	LibraryPlaytime int `json:"library_playtime"`
	LibraryUpdate   int `json:"library_update"`
}

var updateStatsIntKeys = []string{"artists", "albums", "songs", "uptime", "db_playtime", "db_update", "playtime"}

func (h *api) updateStats(ctx context.Context) error {
	s, err := h.client.Stats(ctx)
	if err != nil {
		return err
	}
	ret := &httpMusicStats{}
	for _, k := range updateStatsIntKeys {
		v, ok := s[k]
		if !ok {
			continue
		}
		iv, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		switch k {
		case "artists":
			ret.Artists = iv
		case "albums":
			ret.Albums = iv
		case "songs":
			ret.Songs = iv
		case "uptime":
			ret.Uptime = iv
		case "db_playtime":
			ret.LibraryPlaytime = iv
		case "db_update":
			ret.LibraryUpdate = iv
		case "playtime":
			ret.Playtime = iv
		}
	}

	// force update to Last-Modified header to calc current playing time
	return h.jsonCache.Set("/api/music/stats", ret)
}

func writeHTTPError(w http.ResponseWriter, status int, err error) {
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	b, _ := json.Marshal(map[string]string{"error": err.Error()})
	w.Header().Add("Content-Length", strconv.Itoa(len(b)))
	w.WriteHeader(status)
	w.Write(b)
}

func boolPtr(b bool) *bool       { return &b }
func stringPtr(s string) *string { return &s }
