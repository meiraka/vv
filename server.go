package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/meiraka/gompd/mpd"
	"mime"
	"net/http"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"sync"
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

/*Server http server for vv.*/
type Server struct {
	Port           string
	Music          MusicIF
	MusicDirectory string
	StartTime      time.Time
	upgrader       websocket.Upgrader
	debug          bool
	libraryCache   *gzCache
}

// Serve serves http request.
func (s *Server) Serve() {
	handler := s.makeHandle()
	http.ListenAndServe(fmt.Sprintf(":%s", s.Port), handler)
}

func (s *Server) makeHandle() http.Handler {
	s.libraryCache = new(gzCache)
	h := http.NewServeMux()
	h.HandleFunc("/api/music/control", s.apiMusicControl)
	h.HandleFunc("/api/music/library", s.apiMusicLibrary)
	h.HandleFunc("/api/music/library/", s.apiMusicLibraryOne)
	h.HandleFunc("/api/music/notify", s.apiMusicNotify)
	h.HandleFunc("/api/music/outputs", s.apiMusicOutputs)
	h.HandleFunc("/api/music/outputs/", s.apiMusicOutputs)
	h.HandleFunc("/api/music/songs", s.apiMusicSongs)
	h.HandleFunc("/api/music/songs/", s.apiMusicSongsOne)
	h.HandleFunc("/api/music/songs/current", s.apiMusicSongsCurrent)
	h.HandleFunc("/api/music/songs/sort", s.apiMusicSongsSort)
	h.HandleFunc("/api/music/stats", s.apiMusicStats)
	h.HandleFunc("/api/version", s.apiVersion)
	fs := http.StripPrefix(musicDirectoryPrefix, http.FileServer(http.Dir(s.MusicDirectory)))
	h.HandleFunc(musicDirectoryPrefix, func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	})
	for _, f := range AssetNames() {
		p := "/" + f
		if f == "assets/app.html" {
			p = "/"
		}
		_, err := os.Stat(f)
		if os.IsNotExist(err) || !s.debug {
			data, _ := Asset(f)
			h.HandleFunc(p, makeHandleAssets(f, data))
		} else {
			func(path, rpath string) {
				h.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
					http.ServeFile(w, r, rpath)
				})
			}(p, f)
		}
	}
	return h
}

func (s *Server) apiMusicControl(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		j, err := parseSimpleJSON(r.Body)
		if err != nil {
			writeError(w, &httpClientError{message: "failed to get request parameters", err: err})
			return
		}
		funcs := []func() error{
			func() error { return j.execIfInt("volume", s.Music.Volume) },
			func() error { return j.execIfBool("repeat", s.Music.Repeat) },
			func() error { return j.execIfBool("random", s.Music.Random) },
			func() error {
				return j.execIfString("state", func(state string) error {
					switch state {
					case "play":
						return s.Music.Play()
					case "pause":
						return s.Music.Pause()
					case "next":
						return s.Music.Next()
					case "prev":
						return s.Music.Prev()
					}
					return &httpClientError{message: "unknown state value: " + state}
				})
			},
		}
		for i := range funcs {
			err = funcs[i]()
			if err != nil {
				writeError(w, err)
				return
			}
		}
		writeError(w, err)
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

func (s *Server) apiMusicSongs(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		d, l := s.Music.Playlist()
		writeInterfaceIfModified(w, r, d, l, nil)
	}
}

func (s *Server) apiMusicSongsSort(w http.ResponseWriter, r *http.Request) {
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

func (s *Server) apiMusicSongsOne(w http.ResponseWriter, r *http.Request) {
	p := strings.Replace(r.URL.Path, "/api/music/songs/", "", -1)
	if p == "" {
		s.apiMusicSongs(w, r)
		return
	}
	d, l := s.Music.Playlist()
	writeSongInList(w, r, p, d, l)
}

func (s *Server) apiMusicSongsCurrent(w http.ResponseWriter, r *http.Request) {
	d, l := s.Music.Current()
	writeInterfaceIfModified(w, r, d, l, nil)
}

func (s *Server) apiMusicStats(w http.ResponseWriter, r *http.Request) {
	d, l := s.Music.Stats()
	writeInterfaceIfModified(w, r, d, l, nil)
}

func (s *Server) apiVersion(w http.ResponseWriter, r *http.Request) {
	if modifiedSince(r, s.StartTime) {
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
	} else {
		writeNotModified(w, s.StartTime)
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

func makeHandleAssets(f string, data []byte) func(http.ResponseWriter, *http.Request) {
	n := time.Now().UTC()
	m := mime.TypeByExtension(path.Ext(f))
	var gzdata []byte
	if !strings.Contains(m, "image") {
		gzdata, _ = makeGZip(data)
	}
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Last-Modified", n.Format(http.TimeFormat))
		if m != "" {
			w.Header().Add("Content-Type", m)
		}
		if gzdata != nil && strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			w.Header().Add("Content-Length", strconv.Itoa(len(gzdata)))
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
	return r.Header.Get("If-Modified-Since") != l.Format(http.TimeFormat)
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
		w.WriteHeader(500)
		fmt.Fprintf(w, "{\"error\": \"failed to create json\"}")
		return
	}
	if err != nil {
		if _, ok := err.(*httpClientError); ok {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(500)
		}
	}
	fmt.Fprintf(w, string(b))
	return
}

func writeInterfaceIfModified(w http.ResponseWriter, r *http.Request, d interface{}, l time.Time, err error) {
	if !modifiedSince(r, l) {
		writeNotModified(w, l)
	} else {
		writeInterface(w, d, l, err)
	}
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
		w.WriteHeader(500)
		fmt.Fprintf(w, "{\"error\": \"failed to create json\"}")
		return
	}
	w.Header().Add("Content-Length", strconv.Itoa(len(b)))
	fmt.Fprintf(w, string(b))
	return
}

func writeInterfaceIfCached(w http.ResponseWriter, r *http.Request, d interface{}, l time.Time, g *gzCache, err error) {
	if !modifiedSince(r, l) {
		writeNotModified(w, l)
		return
	}
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
		w.WriteHeader(500)
		fmt.Fprintf(w, "{\"error\": \"failed to create json\"}")
		return
	}
	newgzdata, _ := makeGZip(b)
	if newgzdata == nil {
		w.Header().Add("Content-Length", strconv.Itoa(len(b)))
		fmt.Fprintf(w, string(b))
		return
	}
	w.Header().Add("Content-Encoding", "gzip")
	w.Header().Add("Content-Length", strconv.Itoa(len(newgzdata)))
	w.Write(newgzdata)
	g.set(newgzdata, l)
}

func writeNotModified(w http.ResponseWriter, l time.Time) {
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Last-Modified", l.Format(http.TimeFormat))
	w.WriteHeader(304)
	return
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
