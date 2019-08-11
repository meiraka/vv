package main

import (
	"bytes"
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
	"runtime"
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
	httpUpdateTime = httpContextKey("updateTime")
	httpImagePath  = "/api/music/image/"
)

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

type headResponseWriter struct{ w http.ResponseWriter }

func (h *headResponseWriter) Header() http.Header         { return h.w.Header() }
func (h *headResponseWriter) Write(p []byte) (int, error) { return ioutil.Discard.Write(p) }
func (h *headResponseWriter) WriteHeader(statusCode int)  { h.w.WriteHeader(statusCode) }

// GetOrHead returns MethdNotAllowed if not GET or HEAD
func GetOrHead(alter http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.Method == http.MethodHead {
			alter.ServeHTTP(&headResponseWriter{w: w}, r)
			return
		}
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		alter.ServeHTTP(w, r)
	}
}

// HTTPHandlerConfig holds HTTPHandler config
type HTTPHandlerConfig struct {
	BackgroundTimeout time.Duration
	LocalAssets       bool
	MusicDirectory    string
}

// addHTTPPrefix adds prefix path /api/music/storage to song cover path.
func addHTTPPrefix(m map[string][]string) map[string][]string {
	if v, ok := m["cover"]; ok {
		for i := range v {
			v[i] = path.Join(httpImagePath, v[i])
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
	cover     *LocalCoverSearcher

	mu          sync.Mutex
	playlist    []Song
	library     []Song
	librarySort []Song
	sort        []string
	filters     [][]string
	current     int
}

// NewHTTPHandler creates MPD http handler
func (c HTTPHandlerConfig) NewHTTPHandler(ctx context.Context, cl *mpd.Client, w *mpd.Watcher) (http.Handler, error) {
	if c.BackgroundTimeout == 0 {
		c.BackgroundTimeout = 30 * time.Second
	}
	var cover *LocalCoverSearcher
	tagger := make([]TagAdder, 0, 3)
	tagger = append(tagger, TagAdderFunc(AddTags))
	if len(c.MusicDirectory) != 0 {
		var err error
		cover, err = NewLocalCoverSearcher(c.MusicDirectory, "cover.*")
		if err != nil {
			return nil, err
		}
		tagger = append(tagger, cover)
	}
	h := &httpHandler{
		config:    &c,
		client:    cl,
		watcher:   w,
		jsonCache: newJSONCache(),
		tagger:    append(tagger, TagAdderFunc(addHTTPPrefix)),
		cover:     cover,
	}
	if err := h.updateVersion(); err != nil {
		return nil, err
	}
	for _, v := range []func(context.Context) error{h.updateLibrary, h.updatePlaylist, h.updateStatus, h.updateCurrentSong, h.updateOutputs, h.updateStats} {
		if err := v(ctx); err != nil {
			return nil, err
		}
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
		event:  make(chan string, 10),
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
	return GetOrHead(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			w.WriteHeader(status)
			w.Write(gz)
			return
		}
		w.WriteHeader(status)
		w.Write(b)
	}))
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

type httpAPIVersion struct {
	App string `json:"app"`
	Go  string `json:"go"`
	MPD string `json:"mpd"`
}

func (h *httpHandler) updateVersion() error {
	goVersion := fmt.Sprintf("%s %s %s", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	return h.jsonCache.SetIfModified("/api/version", &httpAPIVersion{App: version, Go: goVersion, MPD: h.client.Version()})
}

func (h *httpHandler) updateLibrary(ctx context.Context) error {
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
	h.updatePlaylistInfo()

	h.mu.Unlock()
	return nil
}

func (h *httpHandler) updatePlaylistInfo() error {
	return h.jsonCache.SetIfModified("/api/music/playlist", &httpPlaylistInfo{
		Current: h.current,
		Sort:    h.sort,
		Filters: h.filters,
	})
}

func (h *httpHandler) updatePlaylist(ctx context.Context) error {
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
	if h.sort != nil && !EqualSongs(h.playlist, h.librarySort) {
		h.sort = nil
		h.filters = nil
		h.librarySort = nil
		h.updatePlaylistInfo()
	}
	h.mu.Unlock()

	return err
}

func (h *httpHandler) updateCurrentSong(ctx context.Context) error {
	l, err := h.client.CurrentSong(ctx)
	if err != nil {
		return err
	}
	return h.jsonCache.SetIfModified("/api/music/playlist/songs/current", h.convSong(l))
}

type httpOutput struct {
	Name      string `json:"name"`
	Plugin    string `json:"plugin,omitempty"`
	Enabled   *bool  `json:"enabled"`
	Attribute string `json:"attribute,omitempty"` // TODO fix type
}

func (h *httpHandler) updateOutputs(ctx context.Context) error {
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

func (h *httpHandler) outputPost(alter http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			alter.ServeHTTP(w, r)
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
		alter.ServeHTTP(w, r)

	}
}

func (h *httpHandler) updateStatus(ctx context.Context) error {
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
	if err := h.updatePlaylistInfo(); err != nil {
		return err
	}
	_, updating := s["updating_db"]
	return h.jsonCache.SetIfModified("/api/music/library", &httpLibraryInfo{
		Updating: updating,
	})
}

func (h *httpHandler) updateStats(ctx context.Context) error {
	s, err := h.client.Stats(ctx)
	if err != nil {
		return err
	}
	// force update to Last-Modified header to calc current playing time
	// TODO: add millisec update time to JSON
	// TODO: cast string to int
	return h.jsonCache.Set("/api/music/stats", s)
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
		librarySort, filters, newpos := SortSongs(h.library, req.Sort, req.Filters, 9999, req.Current)
		update := !EqualSongs(h.playlist, librarySort)
		cl := h.client.BeginCommandList()
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
			alter.ServeHTTP(w, setUpdateTime(r, now))
			return
		}
		r.Method = http.MethodGet
		alter.ServeHTTP(w, setUpdateTime(r, time.Now().UTC()))
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
		now := time.Now().UTC()
		if _, err := h.client.Update(ctx, ""); err != nil {
			writeHTTPError(w, http.StatusBadRequest, err)
			return
		}
		r.Method = http.MethodGet
		alter.ServeHTTP(w, setUpdateTime(r, now))
	}
}

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

type httpPlaylistInfo struct {
	Current int        `json:"current"`
	Sort    []string   `json:"sort,omitempty"`
	Filters [][]string `json:"filters,omitempty"`
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
		if err := ws.WriteMessage(websocket.TextMessage, []byte("ok")); err != nil {
			return
		}
		for {
			select {
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

func (h *httpHandler) statusPost(alter http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			alter.ServeHTTP(w, r)
			return
		}
		var s httpMusicStatus
		if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
			writeHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		ctx := r.Context()
		now := time.Now().UTC()
		if s.Volume != nil {
			if err := h.client.SetVol(ctx, *s.Volume); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
		}
		if s.Repeat != nil {
			if err := h.client.Repeat(ctx, *s.Repeat); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
		}
		if s.Random != nil {
			if err := h.client.Random(ctx, *s.Random); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
		}
		if s.Single != nil {
			if err := h.client.Single(ctx, *s.Single); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
		}
		if s.Oneshot != nil {
			if err := h.client.OneShot(ctx); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
		}
		if s.Consume != nil {
			if err := h.client.Consume(ctx, *s.Consume); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
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
		}
		r.Method = "GET"
		alter.ServeHTTP(w, setUpdateTime(r, now))
	}
}

func writeHTTPError(w http.ResponseWriter, status int, err error) {
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	b, _ := json.Marshal(map[string]string{"error": err.Error()})
	w.Header().Add("Content-Length", strconv.Itoa(len(b)))
	w.WriteHeader(status)
	w.Write(b)
}

func noneMatch(r *http.Request, etag string) bool {
	return r.Header.Get("If-None-Match") == etag
}

func (h *httpHandler) assetsHandler(rpath string, b []byte, hash []byte) http.HandlerFunc {
	if h.config.LocalAssets {
		return func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Cache-Control", "max-age=1")
			http.ServeFile(w, r, rpath)
		}
	}
	m := mime.TypeByExtension(path.Ext(rpath))
	var gz []byte
	var err error
	if m != "image/png" && m != "image/jpg" {
		if gz, err = makeGZip(b); err != nil {
			log.Fatalf("failed to make gzip for static %s: %v", rpath, err)
		}
	}
	length := strconv.Itoa(len(b))
	etag := fmt.Sprintf(`"%s"`, hash)
	lastModified := time.Now().Format(http.TimeFormat)
	return GetOrHead(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if noneMatch(r, etag) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Add("Cache-Control", "max-age=86400")
		if m != "" {
			w.Header().Add("Content-Type", m)
		}
		w.Header().Add("ETag", etag)
		w.Header().Add("Last-Modified", lastModified)
		w.Header().Add("Content-Length", length)
		if gz != nil {
			w.Header().Add("Vary", "Accept-Encoding")
			if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") && gz != nil {
				w.Header().Add("Content-Encoding", "gzip")
				w.WriteHeader(http.StatusOK)
				w.Write(gz)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write(b)
	}))
}

func determineLanguage(r *http.Request, m language.Matcher) language.Tag {
	t, _, _ := language.ParseAcceptLanguage(r.Header.Get("Accept-Language"))
	tag, _, _ := m.Match(t...)
	return tag
}

func (h *httpHandler) i18nAssetsHandler(rpath string, b []byte, hash []byte) http.Handler {
	matcher := language.NewMatcher(translatePrio)
	m := mime.TypeByExtension(path.Ext(rpath))
	if h.config.LocalAssets {
		return GetOrHead(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
			w.Header().Add("Cache-Control", "max-age=1")
			w.Header().Add("Content-Language", tag.String())
			w.Header().Add("Content-Length", strconv.Itoa(len(data)))
			w.Header().Add("Content-Type", m+"; charset=utf-8")
			w.Header().Add("Last-Modified", l.Format(http.TimeFormat))
			w.Header().Add("Vary", "Accept-Encoding, Accept-Language")
			w.Write(data)
			return
		}))
	}
	gz, err := makeGZip(b)
	if err != nil {
		log.Fatalf("failed to make gzip for static %s: %v", rpath, err)
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
	etag := fmt.Sprintf(`"%s"`, hash)
	lastModified := time.Now().Format(http.TimeFormat)
	return GetOrHead(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if noneMatch(r, etag) {
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

		w.Header().Add("Cache-Control", "max-age=86400")
		w.Header().Add("Content-Language", tag.String())
		w.Header().Add("Content-Length", strconv.Itoa(len(b)))
		w.Header().Add("Content-Type", m+"; charset=utf-8")
		w.Header().Add("Etag", etag)
		w.Header().Add("Last-Modified", lastModified)
		w.Header().Add("Vary", "Accept-Encoding, Accept-Language")
		if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") && gz != nil {
			w.Header().Add("Content-Encoding", "gzip")
			w.Write(gz)
			return
		}
		w.Write(b)
	}))

}

func boolPtr(b bool) *bool       { return &b }
func stringPtr(s string) *string { return &s }

func (h *httpHandler) Handle() http.Handler {
	m := http.NewServeMux()
	m.Handle("/", h.i18nAssetsHandler("assets/app.html", AssetsAppHTML, AssetsAppHTMLHash))
	m.Handle("/assets/app.css", h.assetsHandler("assets/app.css", AssetsAppCSS, AssetsAppCSSHash))
	m.Handle("/assets/app.png", h.assetsHandler("assets/app.png", AssetsAppPNG, AssetsAppPNGHash))
	m.Handle("/assets/manifest.json", h.assetsHandler("assets/manifest.json", AssetsManifestJSON, AssetsManifestJSONHash))
	m.Handle("/assets/app-black.png", h.assetsHandler("assets/app-black.png", AssetsAppBlackPNG, AssetsAppBlackPNGHash))
	m.Handle("/assets/w.png", h.assetsHandler("assets/w.png", AssetsWPNG, AssetsWPNGHash))
	m.Handle("/assets/app.js", h.assetsHandler("assets/appv2.js", AssetsAppv2JS, AssetsAppv2JSHash))
	m.Handle("/assets/nocover.svg", h.assetsHandler("assets/nocover.svg", AssetsNocoverSVG, AssetsNocoverSVGHash))
	m.Handle("/api/version", h.jsonCache.Handler("/api/version"))
	m.Handle("/api/music", h.statusWebSocket(h.statusPost(h.jsonCache.Handler("/api/music"))))
	m.Handle("/api/music/stats", h.jsonCache.Handler("/api/music/stats"))
	m.Handle("/api/music/playlist", h.playlistPost(h.jsonCache.Handler("/api/music/playlist")))
	m.Handle("/api/music/playlist/songs", h.jsonCache.Handler("/api/music/playlist/songs"))
	m.Handle("/api/music/playlist/songs/current", h.jsonCache.Handler("/api/music/playlist/songs/current"))
	m.Handle("/api/music/library", h.libraryPost(h.jsonCache.Handler("/api/music/library")))
	m.Handle("/api/music/library/songs", h.jsonCache.Handler("/api/music/library/songs"))
	m.Handle("/api/music/outputs", h.outputPost(h.jsonCache.Handler("/api/music/outputs")))
	m.Handle(httpImagePath, http.StripPrefix(httpImagePath, h.cover.Handler()))

	return m
}
