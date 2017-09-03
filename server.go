package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fhs/gompd/mpd"
	"github.com/gorilla/websocket"
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

// Serve serves http request.
func Serve(music MusicIF, musicDirectory string, port string) {
	s := Server{Music: music, MusicDirectory: musicDirectory, Port: port, StartTime: time.Now().UTC(), debug: true}
	s.Serve()
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
			writeError(w, err)
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
					return errors.New("unknown state value: " + state)
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
		if err == nil {
			if data.Action == "rescan" {
				s.Music.RescanLibrary()
			} else {
				err = errors.New("unknown action: " + data.Action)
			}
		}
		writeError(w, err)
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
			writeError(w, err)
			return
		}
		decoder := json.NewDecoder(r.Body)
		var data = struct {
			OutputEnabled bool `json:"outputenabled"`
		}{}
		err = decoder.Decode(&data)
		if err != nil {
			writeError(w, err)
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
	case "POST":
		decoder := json.NewDecoder(r.Body)
		var data struct {
			Action string   `json:"action"`
			Keys   []string `json:"keys"`
			URI    string   `json:"uri"`
		}
		err := decoder.Decode(&data)
		if err == nil {
			s.Music.SortPlaylist(data.Keys, data.URI)
		}
		writeError(w, err)
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
	w.Header().Add("Content-Type", "application/json")
	errstr := ""
	if err != nil {
		errstr = err.Error()
	}
	v := jsonMap{"error": errstr}
	b, jsonerr := json.Marshal(v)
	if jsonerr != nil {
		return
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

func writeSongInList(w http.ResponseWriter, r *http.Request, path string, d []mpd.Attrs, l time.Time) {
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
	Playlist() ([]mpd.Attrs, time.Time)
	Library() ([]mpd.Attrs, time.Time)
	RescanLibrary() error
	Current() (mpd.Attrs, time.Time)
	Status() (PlayerStatus, time.Time)
	Stats() (mpd.Attrs, time.Time)
	Output(int, bool) error
	Outputs() ([]mpd.Attrs, time.Time)
	SortPlaylist([]string, string) error
	Subscribe(chan string)
	Unsubscribe(chan string)
}
