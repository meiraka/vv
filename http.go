package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/meiraka/vv/mpd"
	"golang.org/x/text/language"
)

type httpContextKey string

const (
	httpParentStatus = httpContextKey("status")
)

// getParentStatus gets status code from http.Request witch is set by parentStatus
func getParentStatus(r *http.Request, alter int) int {
	if v := r.Context().Value(httpParentStatus); v != nil {
		if i, ok := v.(int); ok {
			return i
		}
	}
	return alter
}

// parentStatus sets status code to http.Request
func parentStatus(r *http.Request, status int) *http.Request {
	ctx := context.WithValue(r.Context(), httpParentStatus, status)
	return r.WithContext(ctx)
}

// HTTPHandlerConfig holds HTTPHandler config
type HTTPHandlerConfig struct {
	BackgroundTimeout time.Duration
	LocalAssets       bool
}

// addHTTPPrefix adds prefix path /api/music/storage to song cover path.
func addHTTPPrefix(m map[string][]string) map[string][]string {
	if v, ok := m["cover"]; ok {
		for i := range v {
			v[i] = path.Join("/api/music/storage/", v[i])
		}
	}
	return m
}

type httpHandler struct {
	config    *HTTPHandlerConfig
	client    *mpd.Client
	watcher   *mpd.Watcher
	jsonCache *jsonCache
	upgrader  websocket.Upgrader
	tagger    []TagAdder

	mu       sync.Mutex
	playlist []Song
	library  []Song
	sort     []string
	filters  [][]string
	current  int
}

// NewHTTPHandler creates MPD http handler
func (c HTTPHandlerConfig) NewHTTPHandler(ctx context.Context, cl *mpd.Client, w *mpd.Watcher, t []TagAdder) (http.Handler, error) {
	if c.BackgroundTimeout == 0 {
		c.BackgroundTimeout = 30 * time.Second
	}
	h := &httpHandler{
		config:    &c,
		client:    cl,
		watcher:   w,
		jsonCache: newJSONCache(),
		tagger:    append(t, TagAdderFunc(addHTTPPrefix)),
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
	if err := h.updateOutputs(ctx); err != nil {
		return nil, err
	}
	go func() {
		defer h.jsonCache.Close()
		for e := range w.C {
			ctx, cancel := context.WithTimeout(context.Background(), c.BackgroundTimeout)
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
				h.updateOutputs(ctx)
			}
			cancel()
		}
	}()
	return h.Handle(), nil
}

type jsonCache struct {
	event  chan string
	data   map[string][]byte
	gzdata map[string][]byte
	date   map[string]time.Time
	mu     sync.RWMutex
}

func newJSONCache() *jsonCache {
	return &jsonCache{
		event:  make(chan string),
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
	h.mu.Lock()
	h.library = v
	h.mu.Unlock()

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

	h.mu.Lock()
	h.playlist = v
	h.mu.Unlock()

	return nil
}

func (h *httpHandler) updateCurrentSong(ctx context.Context) error {
	l, err := h.client.CurrentSong(ctx)
	if err != nil {
		return err
	}
	return h.jsonCache.Set("/api/music/playlist/songs/current", h.convSong(l), false)
}

type httpOutput struct {
	Name      string `json:"name"`
	Plugin    string `json:"plugin"`
	Enabled   bool   `json:"enabled"`
	Attribute string `json:"attribute"` // TODO fix type
}

func (h *httpHandler) updateOutputs(ctx context.Context) error {
	l, err := h.client.Outputs(ctx)
	if err != nil {
		return err
	}
	data := make([]*httpOutput, len(l))
	for i, v := range l {
		data[i] = &httpOutput{
			Name:      v["outputname"],
			Plugin:    v["plugin"],
			Enabled:   v["outputenabled"] == "1",
			Attribute: v["attribute"],
		}
	}
	return h.jsonCache.Set("/api/music/outputs", data, false)
}

func (h *httpHandler) updateStatus(ctx context.Context) error {
	s, err := h.client.Status(ctx)
	if err != nil {
		return err
	}
	v, err := strconv.Atoi(s["volume"])
	if err != nil {
		v = -1
	}
	pos, err := strconv.Atoi(s["song"])
	if err != nil {
		return fmt.Errorf("song: %v", err)
	}
	elapsed, err := strconv.ParseFloat(s["elapsed"], 64)
	if err != nil {
		elapsed = 0
		// return fmt.Errorf("elapsed: %v", err)
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
	h.mu.Lock()
	defer h.mu.Unlock()
	h.current = pos
	if err := h.jsonCache.Set("/api/music/playlist", &httpPlaylistInfo{
		Current: h.current,
		Sort:    h.sort,
		Filters: h.filters,
	}, false); err != nil {
		return err
	}
	_, updating := s["updating_db"]
	return h.jsonCache.Set("/api/music/library", &httpLibraryInfo{
		Updating: updating,
	}, false)
}

func (h *httpHandler) playlistPost(alter http.Handler) http.HandlerFunc {
	sem := make(chan struct{}, 1)
	sem <- struct{}{}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			alter.ServeHTTP(w, r)
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
		library, filters, newpos := SortSongs(h.library, req.Sort, req.Filters, 9999, req.Current)
		playlist := h.playlist
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
		h.mu.Unlock()
		if !update {
			ctx := r.Context()
			if err := h.client.Play(ctx, newpos); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			h.mu.Lock()
			h.sort = req.Sort
			h.filters = filters
			h.mu.Unlock()
			h.updateStatus(ctx)
			r.Method = http.MethodGet
			alter.ServeHTTP(w, r)
			return
		}
		r.Method = http.MethodGet
		alter.ServeHTTP(w, parentStatus(r, http.StatusAccepted))
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), h.config.BackgroundTimeout)
			defer cancel()
			select {
			case <-sem:
			case <-ctx.Done():
				return
			}
			defer func() { sem <- struct{}{} }()
			if err := cl.End(ctx); err != nil {
				return
			}
			h.mu.Lock()
			h.sort = req.Sort
			h.filters = filters
			h.mu.Unlock()
		}()

	}
}

func (h *httpHandler) libraryPost(alter http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			alter.ServeHTTP(w, r)
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
		if _, err := h.client.Update(ctx, ""); err != nil {
			writeHTTPError(w, http.StatusBadRequest, err)
			return
		}
		h.updateStatus(ctx)
		r.Method = http.MethodGet
		alter.ServeHTTP(w, r)
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

type httpLibraryInfo struct {
	Updating bool `json:"updating"`
}

func (h *httpHandler) statusWebSocket(alter http.Handler) http.HandlerFunc {
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
			writeHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		ctx := r.Context()
		var changed bool
		if s.Volume != nil {
			changed = true
			if err := h.client.SetVol(ctx, *s.Volume); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
		}
		if s.Repeat != nil {
			changed = true
			if err := h.client.Repeat(ctx, *s.Repeat); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
		}
		if s.Random != nil {
			changed = true
			if err := h.client.Random(ctx, *s.Random); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
		}
		if s.Single != nil {
			changed = true
			if err := h.client.Single(ctx, *s.Single); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
		}
		if s.Oneshot != nil {
			changed = true
			if err := h.client.OneShot(ctx); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
		}
		if s.Consume != nil {
			changed = true
			if err := h.client.Consume(ctx, *s.Consume); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
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
				writeHTTPError(w, http.StatusBadRequest, fmt.Errorf("unknown state: %s", *s.State))
				return
			}
			if err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
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
			w.WriteHeader(getParentStatus(r, http.StatusOK))
			w.Write(gz)
			return
		}
		w.WriteHeader(getParentStatus(r, http.StatusOK))
		w.Write(b)
	}
}

func (h *httpHandler) assetsHandler(rpath string, gz []byte, date time.Time) http.HandlerFunc {
	if h.config.LocalAssets {
		return func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, rpath)
		}
	}
	r, err := gzip.NewReader(bytes.NewReader(gz))
	if err != nil {
		log.Fatalf("failed to create gzip.NewReader for static %s: %v", rpath, err)
	}
	b, err := ioutil.ReadAll(r)
	if err != nil {
		log.Fatalf("failed to read static %s: %v", rpath, err)
	}
	m := mime.TypeByExtension(path.Ext(rpath))
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Method != "HEAD" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if !modifiedSince(r, date) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Add("Last-Modified", date.Format(http.TimeFormat))
		w.Header().Add("Content-Type", "application/json; charset=utf-8")
		if r.Method == "HEAD" {
			return
		}
		if m != "" {
			w.Header().Add("Content-Type", m)
		}
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") && gz != nil {
			w.Header().Add("Content-Encoding", "gzip")
			w.WriteHeader(getParentStatus(r, http.StatusOK))
			w.Write(gz)
			return
		}
		w.WriteHeader(getParentStatus(r, http.StatusOK))
		w.Write(b)
	}
}

func determineLanguage(r *http.Request, m language.Matcher) language.Tag {
	t, _, _ := language.ParseAcceptLanguage(r.Header.Get("Accept-Language"))
	tag, _, _ := m.Match(t...)
	return tag
}

func (h *httpHandler) i18nAssetsHandler(rpath string, gz []byte, date time.Time) http.HandlerFunc {
	matcher := language.NewMatcher(translatePrio)
	m := mime.TypeByExtension(path.Ext(rpath))
	if h.config.LocalAssets {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "GET" && r.Method != "HEAD" {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			info, err := os.Stat(rpath)
			if err != nil {
				http.NotFound(w, r)
				return
			}
			l := info.ModTime()
			if !modifiedSince(r, l) {
				w.WriteHeader(304)
				return
			}
			tag := determineLanguage(r, matcher)
			data, err := ioutil.ReadFile(rpath)
			if err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			data, err = translate(data, tag)
			if err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			w.Header().Add("Vary", "Accept-Encoding, Accept-Language")
			w.Header().Add("Content-Language", tag.String())
			w.Header().Add("Content-Type", m+"; charset=utf-8")
			w.Header().Add("Last-Modified", l.Format(http.TimeFormat))
			w.Header().Add("Content-Length", strconv.Itoa(len(data)))
			w.Write(data)
			return
		}
	}
	r, err := gzip.NewReader(bytes.NewReader(gz))
	if err != nil {
		log.Fatalf("failed to create gzip.NewReader for static %s: %v", rpath, err)
	}
	b, err := ioutil.ReadAll(r)
	if err != nil {
		log.Fatalf("failed to read static %s: %v", rpath, err)
	}
	bt := make([][]byte, len(translatePrio))
	gt := make([][]byte, len(translatePrio))
	for i := range translatePrio {
		data, err := translate(b, translatePrio[i])
		if err != nil {
			log.Fatalf("failed to translate %s to %v: %v", rpath, translatePrio[i], err)
		}
		bt[i] = data
		data, err = makeGZip(data)
		if err != nil {
			log.Fatalf("failed to translate %s to %v: %v", rpath, translatePrio[i], err)
		}
		gt[i] = data
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" && r.Method != "HEAD" {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if !modifiedSince(r, date) {
			w.WriteHeader(304)
			return
		}
		tag := determineLanguage(r, matcher)
		index := 0
		for ; index < len(translatePrio); index++ {
			if translatePrio[index] == tag {
				break
			}
		}
		b = bt[index]
		gz = gt[index]

		w.Header().Add("Vary", "Accept-Encoding, Accept-Language")
		w.Header().Add("Content-Language", tag.String())
		w.Header().Add("Content-Type", m+"; charset=utf-8")
		w.Header().Add("Content-Length", strconv.Itoa(len(b)))
		w.Header().Add("Last-Modified", date.Format(http.TimeFormat))
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") && gz != nil {
			w.Header().Add("Content-Encoding", "gzip")
			w.Write(gz)
			return
		}
		w.Write(b)
	}

}

func (h *httpHandler) Handle() http.Handler {
	m := http.NewServeMux()
	m.Handle("/", h.i18nAssetsHandler("assets/app.html", AssetsAppHTML, AssetsAppHTMLDate))
	m.Handle("/assets/app.png", h.assetsHandler("assets/app.png", AssetsAppPNG, AssetsAppPNGDate))
	m.Handle("/api/music", h.statusWebSocket(h.statusPost(h.jsonCacheHandler("/api/music"))))
	m.Handle("/api/music/playlist", h.playlistPost(h.jsonCacheHandler("/api/music/playlist")))
	m.Handle("/api/music/playlist/songs", h.jsonCacheHandler("/api/music/playlist/songs"))
	m.Handle("/api/music/playlist/songs/current", h.jsonCacheHandler("/api/music/playlist/songs/current"))
	m.Handle("/api/music/library", h.libraryPost(h.jsonCacheHandler("/api/music/library")))
	m.Handle("/api/music/library/songs", h.jsonCacheHandler("/api/music/library/songs"))
	m.Handle("/api/music/outputs", h.jsonCacheHandler("/api/music/outputs"))
	return m
}
