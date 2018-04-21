package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/meiraka/gompd/mpd"
	"golang.org/x/text/language"
	"io/ioutil"
	"mime"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const musicDirectoryPrefix = "/music_directory/"

type httpClientError struct {
	message string
	err     error
}

func (e *httpClientError) Error() string {
	if e.err != nil {
		return fmt.Sprintf("%s: %s", e.message, e.err.Error())
	}
	return e.message
}

type jsonMap map[string]interface{}

/*Server http server for vv.*/
type Server struct {
	Addr           string
	Music          MusicIF
	MusicDirectory string
	StartTime      time.Time
	KeepAlive      bool
	upgrader       websocket.Upgrader
	debug          bool
	libraryCache   *gzCache
	matcher        language.Matcher
	rootCaches     sync.Map
	rootGzipCaches sync.Map
}

// Serve serves http request.
func (s *Server) Serve() {
	handler := s.makeHandle()
	srv := &http.Server{Addr: s.Addr, Handler: handler}
	srv.SetKeepAlivesEnabled(s.KeepAlive)
	go func() {
		srv.ListenAndServe()
	}()
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGTERM, syscall.SIGINT)
	<-sc
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

func (s *Server) makeHandle() http.Handler {
	s.libraryCache = new(gzCache)
	s.matcher = language.NewMatcher(translatePrio)
	s.rootCachesInit()
	h := http.NewServeMux()
	h.HandleFunc("/api/images/", s.apiImages)
	h.HandleFunc("/api/music/status", s.apiMusicStatus)
	h.HandleFunc("/api/music/library", s.apiMusicLibrary)
	h.HandleFunc("/api/music/library/", s.apiMusicLibraryOne)
	h.HandleFunc("/api/music/notify", s.apiMusicNotify)
	h.HandleFunc("/api/music/outputs", s.apiMusicOutputs)
	h.HandleFunc("/api/music/outputs/", s.apiMusicOutputs)
	h.HandleFunc("/api/music/playlist", s.apiMusicPlaylist)
	h.HandleFunc("/api/music/playlist/", s.apiMusicPlaylistOne)
	h.HandleFunc("/api/music/playlist/current", s.apiMusicPlaylistCurrent)
	h.HandleFunc("/api/music/playlist/sort", s.apiMusicPlaylistSort)
	h.HandleFunc("/api/music/stats", s.apiMusicStats)
	h.HandleFunc("/api/version", s.apiVersion)
	h.HandleFunc("/", s.root)
	fs := http.StripPrefix(musicDirectoryPrefix, http.FileServer(http.Dir(s.MusicDirectory)))
	h.HandleFunc(musicDirectoryPrefix, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Cache-Control", "max-age=86400")
		fs.ServeHTTP(w, r)
	})
	for _, f := range AssetNames() {
		p := "/" + f
		if s.debug {
			if _, err := os.Stat(f); err == nil {
				func(path, rpath string) {
					h.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
						w.Header().Add("Vary", "Accept-Encoding")
						http.ServeFile(w, r, rpath)
					})
				}(p, f)
				continue
			}
		}
		h.HandleFunc(p, makeHandleAssets(f))
	}
	return h
}

func (s *Server) apiImages(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	prefix := "/api/images" + musicDirectoryPrefix
	if !strings.HasPrefix(r.URL.Path, prefix) {
		http.NotFound(w, r)
		return
	}
	q := r.URL.Query()
	ws, wok := q["width"]
	hs, hok := q["height"]
	if !wok || !hok || len(ws) != 1 || len(hs) != 1 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	width, err := strconv.Atoi(ws[0])
	height, err := strconv.Atoi(hs[0])
	imgurl := strings.TrimPrefix(r.URL.Path, prefix)
	imgpath := filepath.Join(s.MusicDirectory, imgurl)
	info, err := os.Stat(imgpath)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	l := info.ModTime()
	if !modifiedSince(r, l) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	data, err := ioutil.ReadFile(imgpath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	data, err = resizeImage(data, width, height)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Add("Content-Type", "image/jpeg")
	w.Header().Add("Last-Modified", l.Format(http.TimeFormat))
	w.Header().Add("Content-Length", strconv.Itoa(len(data)))
	w.Header().Add("Cache-Control", "max-age=86400")
	w.Write(data)
}

func (s *Server) apiMusicStatus(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		decoder := json.NewDecoder(r.Body)
		var data struct {
			Volume  *int    `json:"volume"`
			Repeat  *bool   `json:"repeat"`
			Random  *bool   `json:"random"`
			Single  *bool   `json:"single"`
			Consume *bool   `json:"consume"`
			State   *string `json:"state"`
		}
		err := decoder.Decode(&data)
		if err != nil {
			writeError(w, &httpClientError{message: "failed to get request parameters", err: err})
			return
		}
		if data.Volume != nil {
			err = s.Music.Volume(*data.Volume)
			if err != nil {
				writeError(w, err)
				return
			}
		}
		if data.Repeat != nil {
			err = s.Music.Repeat(*data.Repeat)
			if err != nil {
				writeError(w, err)
				return
			}
		}
		if data.Single != nil {
			err = s.Music.Single(*data.Single)
			if err != nil {
				writeError(w, err)
				return
			}
		}
		if data.Random != nil {
			s.Music.Random(*data.Random)
			if err != nil {
				writeError(w, err)
				return
			}
		}
		if data.State != nil {
			switch *data.State {
			case "play":
				err = s.Music.Play()
			case "pause":
				err = s.Music.Pause()
			case "next":
				err = s.Music.Next()
			case "prev":
				err = s.Music.Prev()
			default:
				writeError(w, &httpClientError{message: "unknown state value: " + *data.State})
				return
			}
			if err != nil {
				writeError(w, err)
				return
			}
		}
		writeError(w, nil)
		return
	case "GET":
		d, l := s.Music.Status()
		writeInterfaceIfModified(w, r, d, l, nil)
	}
}

func (s *Server) apiMusicLibrary(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		d, l := s.Music.Library()
		writeInterfaceIfCached(w, r, d, l, s.libraryCache, nil)
	case "POST":
		decoder := json.NewDecoder(r.Body)
		var data struct {
			Action string `json:"action"`
		}
		err := decoder.Decode(&data)
		if err != nil {
			writeError(w, &httpClientError{message: "failed to get request parameters", err: err})
			return
		}
		if data.Action == "rescan" {
			s.Music.RescanLibrary()
		} else {
			err = &httpClientError{message: "unknown action: " + data.Action}
		}
		writeError(w, err)
		return
	}
}

func (s *Server) apiMusicLibraryOne(w http.ResponseWriter, r *http.Request) {
	p := strings.Replace(r.URL.Path, "/api/music/library/", "", -1)
	if p == "" {
		s.apiMusicLibrary(w, r)
		return
	}
	d, l := s.Music.Library()
	writeSongInList(w, r, p, d, l)
}

func (s *Server) apiMusicNotify(w http.ResponseWriter, r *http.Request) {
	c, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close()
	n := make(chan string, 10)
	s.Music.Subscribe(n)
	defer s.Music.Unsubscribe(n)
	for {
		select {
		case e := <-n:
			err = c.WriteMessage(websocket.TextMessage, []byte(e))
			if err != nil {
				return
			}
		case <-time.After(time.Second * 5):
			err = c.WriteMessage(websocket.TextMessage, []byte("ping"))
			if err != nil {
				return
			}
		}
	}

}

func (s *Server) apiMusicOutputs(w http.ResponseWriter, r *http.Request) {
	d, l := s.Music.Outputs()
	if r.Method == "POST" {
		id, err := strconv.Atoi(
			strings.Replace(r.URL.Path, "/api/music/outputs/", "", -1),
		)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		decoder := json.NewDecoder(r.Body)
		var data = struct {
			OutputEnabled bool `json:"outputenabled"`
		}{}
		err = decoder.Decode(&data)
		if err != nil {
			writeError(w, &httpClientError{message: "failed to get request parameters", err: err})
			return
		}
		writeError(w, s.Music.Output(id, data.OutputEnabled))
		return
	}
	writeInterfaceIfModified(w, r, d, l, nil)
}

func (s *Server) apiMusicPlaylist(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		d, l := s.Music.Playlist()
		writeInterfaceIfModified(w, r, d, l, nil)
	}
}

func (s *Server) apiMusicPlaylistSort(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		s, k, f, l := s.Music.PlaylistIsSorted()
		d := struct {
			Sorted  bool       `json:"sorted"`
			Keys    []string   `json:"keys"`
			Filters [][]string `json:"filters"`
		}{s, k, f}
		writeInterfaceIfModified(w, r, d, l, nil)
	case "POST":
		decoder := json.NewDecoder(r.Body)
		var data struct {
			Keys    []string   `json:"keys"`
			Filters [][]string `json:"filters"`
			Play    int        `json:"play"`
		}
		err := decoder.Decode(&data)
		if err != nil {
			writeError(w, &httpClientError{message: "failed to get request parameters", err: err})
			return
		}
		if data.Keys == nil || data.Filters == nil {
			writeError(w, &httpClientError{message: "failed to get request parameters. missing fields: keys or/and filters"})
			return
		}
		err = s.Music.SortPlaylist(data.Keys, data.Filters, data.Play)
		writeError(w, err)
		return
	}
}

func (s *Server) apiMusicPlaylistOne(w http.ResponseWriter, r *http.Request) {
	p := strings.Replace(r.URL.Path, "/api/music/playlist/", "", -1)
	if p == "" {
		s.apiMusicPlaylist(w, r)
		return
	}
	d, l := s.Music.Playlist()
	writeSongInList(w, r, p, d, l)
}

func (s *Server) apiMusicPlaylistCurrent(w http.ResponseWriter, r *http.Request) {
	d, l := s.Music.Current()
	writeInterfaceIfModified(w, r, d, l, nil)
}

func (s *Server) apiMusicStats(w http.ResponseWriter, r *http.Request) {
	d, l := s.Music.Stats()
	writeInterfaceIfModified(w, r, d, l, nil)
}

func (s *Server) apiVersion(w http.ResponseWriter, r *http.Request) {
	if !modifiedSince(r, s.StartTime) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	vvPostfix := ""
	if s.debug {
		vvPostfix = vvPostfix + " dev mode"
	}
	vvVersion := version
	if len(vvVersion) == 0 {
		vvVersion = staticVersion
	}
	goVersion := fmt.Sprintf("%s %s %s", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	d := map[string]string{"vv": vvVersion + vvPostfix, "go": goVersion}
	writeInterface(w, d, s.StartTime, nil)
}

func (s *Server) root(w http.ResponseWriter, r *http.Request) {
	t, _, _ := language.ParseAcceptLanguage(r.Header.Get("Accept-Language"))
	tag, _, _ := s.matcher.Match(t...)
	specifiedLang := r.URL.Path != "/"
	if specifiedLang {
		lang := strings.Trim(r.URL.Path, "/")
		found := false
		for i := range translatePrio {
			if translatePrio[i].String() == lang {
				tag = translatePrio[i]
				found = true
				break
			}
		}
		if !found {
			http.NotFound(w, r)
			return
		}
	}
	if s.debug {
		if info, err := os.Stat("assets/app.html"); err == nil {
			l := info.ModTime()
			if !modifiedSince(r, l) {
				w.WriteHeader(304)
				return
			}
			data, err := ioutil.ReadFile("assets/app.html")
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			data, err = translate(data, tag)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if specifiedLang {
				w.Header().Add("Vary", "Accept-Encoding")
			} else {
				w.Header().Add("Vary", "Accept-Encoding, Accept-Language")
			}
			w.Header().Add("Content-Language", tag.String())
			w.Header().Add("Content-Type", "text/html; charset=utf-8")
			w.Header().Add("Last-Modified", l.Format(http.TimeFormat))
			w.Header().Add("Content-Length", strconv.Itoa(len(data)))
			w.Write(data)
			return
		}
	}

	l := mustAssetInfo("assets/app.html").ModTime()
	if !modifiedSince(r, l) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	var cvalue interface{}
	gzip := false
	ok := false
	if strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		gzip = true
		cvalue, ok = s.rootGzipCaches.Load(tag)
	} else {
		cvalue, ok = s.rootCaches.Load(tag)
	}
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	data, ok := cvalue.([]byte)
	if !ok {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if specifiedLang {
		w.Header().Add("Vary", "Accept-Encoding")
	} else {
		w.Header().Add("Vary", "Accept-Encoding, Accept-Language")
	}
	w.Header().Add("Content-Language", tag.String())
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.Header().Add("Content-Length", strconv.Itoa(len(data)))
	if gzip {
		w.Header().Add("Content-Encoding", "gzip")
	}
	w.Header().Add("Cache-Control", "max-age=86400")
	w.Header().Add("Last-Modified", l.Format(http.TimeFormat))
	w.Write(data)
}

func (s *Server) rootCachesInit() {
	data := mustAsset("assets/app.html")
	for i := range translatePrio {
		data, err := translate(data, translatePrio[i])
		if err != nil {
			continue
		}
		s.rootCaches.Store(translatePrio[i], data)
		data, err = makeGZip(data)
		if err != nil {
			continue
		}
		s.rootGzipCaches.Store(translatePrio[i], data)
	}
}

type gzCache struct {
	data     []byte
	modified time.Time
	m        sync.RWMutex
}

func (c *gzCache) get() ([]byte, time.Time) {
	c.m.RLock()
	defer c.m.RUnlock()
	return c.data, c.modified
}

func (c *gzCache) set(data []byte, modified time.Time) {
	c.m.Lock()
	defer c.m.Unlock()
	c.data = data
	c.modified = modified
}

func makeGZip(data []byte) ([]byte, error) {
	var gz bytes.Buffer
	zw := gzip.NewWriter(&gz)
	_, err := zw.Write(data)
	if err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return gz.Bytes(), nil
}

func makeHandleAssets(f string) func(http.ResponseWriter, *http.Request) {
	data := mustAsset(f)
	info := mustAssetInfo(f)
	n := info.ModTime()
	m := mime.TypeByExtension(path.Ext(f))
	var gzdata []byte
	if !strings.Contains(m, "image") {
		gzdata, _ = makeGZip(data)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if !modifiedSince(r, n) {
			w.WriteHeader(304)
			return
		}
		w.Header().Add("Last-Modified", n.Format(http.TimeFormat))
		w.Header().Add("Vary", "Accept-Encoding")
		if m != "" {
			w.Header().Add("Content-Type", m)
		}
		if gzdata != nil && strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			w.Header().Add("Content-Length", strconv.Itoa(len(gzdata)))
			w.Header().Add("Cache-Control", "max-age=86400")
			w.Header().Add("Content-Encoding", "gzip")
			w.Write(gzdata)
		} else {
			w.Header().Add("Content-Length", strconv.Itoa(len(data)))
			w.Write(data)
		}
	}
}

/*modifiedSince compares If-Modified-Since header given time.Time.*/
func modifiedSince(r *http.Request, l time.Time) bool {
	t, err := time.Parse(http.TimeFormat, r.Header.Get("If-Modified-Since"))
	if err != nil {
		return true
	}
	return !l.Before(t.Add(time.Second))
}

func writeError(w http.ResponseWriter, err error) {
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	errstr := ""
	if err != nil {
		errstr = err.Error()
	}
	v := jsonMap{"error": errstr}
	b, jsonerr := json.Marshal(v)
	if jsonerr != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "{\"error\": \"failed to create json\"}")
		return
	}
	if err != nil {
		if _, ok := err.(*httpClientError); ok {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
	fmt.Fprintf(w, string(b))
	return
}

func writeInterfaceIfModified(w http.ResponseWriter, r *http.Request, d interface{}, l time.Time, err error) {
	if !modifiedSince(r, l) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	writeInterface(w, d, l, err)
}

func writeInterface(w http.ResponseWriter, d interface{}, l time.Time, err error) {
	w.Header().Add("Last-Modified", l.Format(http.TimeFormat))
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	errstr := ""
	if err != nil {
		errstr = err.Error()
	}
	v := jsonMap{"error": errstr, "data": d}
	b, jsonerr := json.Marshal(v)
	if jsonerr != nil {
		writeError(w, jsonerr)
		return
	}
	w.Header().Add("Content-Length", strconv.Itoa(len(b)))
	fmt.Fprintf(w, string(b))
	return
}

func writeInterfaceIfCached(w http.ResponseWriter, r *http.Request, d interface{}, l time.Time, g *gzCache, err error) {
	if !modifiedSince(r, l) {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Add("Vary", "Accept-Encoding")
	if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") || g == nil {
		writeInterface(w, d, l, err)
		return
	}
	w.Header().Add("Last-Modified", l.Format(http.TimeFormat))
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	gzdata, gzmodified := g.get()
	if gzdata != nil && gzmodified.Equal(l) {
		w.Header().Add("Content-Encoding", "gzip")
		w.Header().Add("Content-Length", strconv.Itoa(len(gzdata)))
		w.Write(gzdata)
		return
	}
	errstr := ""
	if err != nil {
		errstr = err.Error()
	}
	v := jsonMap{"error": errstr, "data": d}
	b, jsonerr := json.Marshal(v)
	if jsonerr != nil {
		writeError(w, jsonerr)
		return
	}
	newgzdata, err := makeGZip(b)
	if err != nil {
		w.Header().Add("Content-Length", strconv.Itoa(len(b)))
		fmt.Fprintf(w, string(b))
		return
	}
	w.Header().Add("Content-Encoding", "gzip")
	w.Header().Add("Content-Length", strconv.Itoa(len(newgzdata)))
	w.Write(newgzdata)
	g.set(newgzdata, l)
}

func writeSongInList(w http.ResponseWriter, r *http.Request, path string, d []Song, l time.Time) {
	id, err := strconv.Atoi(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if len(d) <= id || id < 0 {
		http.NotFound(w, r)
		return
	}
	writeInterfaceIfModified(w, r, d[id], l, nil)
}

// MusicIF Represents music player.
type MusicIF interface {
	Play() error
	Pause() error
	Next() error
	Prev() error
	Volume(int) error
	Single(bool) error
	Repeat(bool) error
	Random(bool) error
	Playlist() ([]Song, time.Time)
	PlaylistIsSorted() (bool, []string, [][]string, time.Time)
	Library() ([]Song, time.Time)
	RescanLibrary() error
	Current() (Song, time.Time)
	Status() (Status, time.Time)
	Stats() (mpd.Attrs, time.Time)
	Output(int, bool) error
	Outputs() ([]mpd.Attrs, time.Time)
	SortPlaylist([]string, [][]string, int) error
	Subscribe(chan string)
	Unsubscribe(chan string)
}

func mustAsset(name string) []byte {
	d, err := Asset(name)
	if err != nil {
		panic(err)
	}
	return d
}

func mustAssetInfo(name string) os.FileInfo {
	i, err := AssetInfo(name)
	if err != nil {
		panic(err)
	}
	return i
}
