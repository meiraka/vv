package main

import (
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
	"time"
)

const musicDirectoryPrefix = "/music_directory/"

/*Server http server for vv.*/
type Server struct {
	Port           string
	Music          MusicIF
	MusicDirectory string
	StartTime      time.Time
	upgrader       websocket.Upgrader
	debug          bool
}

// Serve serves http request.
func (s *Server) Serve() {
	handler := s.makeHandle()
	http.ListenAndServe(fmt.Sprintf(":%s", s.Port), handler)
}

func writeJSONInterface(w http.ResponseWriter, d interface{}, l time.Time, err error) {
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
	fmt.Fprintf(w, string(b))
	return
}

func writeJSON(w http.ResponseWriter, err error) {
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

/*modifiedSince compares If-Modified-Since header given time.Time.*/
func modifiedSince(r *http.Request, l time.Time) bool {
	return r.Header.Get("If-Modified-Since") != l.Format(http.TimeFormat)
}

func notModified(w http.ResponseWriter, l time.Time) {
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Last-Modified", l.Format(http.TimeFormat))
	w.WriteHeader(304)
	return
}

func (s *Server) apiMusicPlaylist(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		d, l := s.Music.Playlist()
		returnList(w, r, d, l)
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
		writeJSON(w, err)
	}
}

func (s *Server) apiMusicPlaylistOne(w http.ResponseWriter, r *http.Request) {
	p := strings.Replace(r.URL.Path, "/api/music/songs/", "", -1)
	if p == "" {
		s.apiMusicPlaylist(w, r)
		return
	}
	d, l := s.Music.Playlist()
	returnListInSong(w, r, p, d, l)
}

func (s *Server) apiMusicLibrary(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		d, l := s.Music.Library()
		returnList(w, r, d, l)
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
		writeJSON(w, err)
	}
}

func (s *Server) apiMusicLibraryOne(w http.ResponseWriter, r *http.Request) {
	p := strings.Replace(r.URL.Path, "/api/music/library/", "", -1)
	if p == "" {
		s.apiMusicLibrary(w, r)
		return
	}
	d, l := s.Music.Library()
	returnListInSong(w, r, p, d, l)
}

func (s *Server) apiMusicCurrent(w http.ResponseWriter, r *http.Request) {
	d, l := s.Music.Current()
	returnSong(w, r, d, l)
}

func (s *Server) apiMusicStats(w http.ResponseWriter, r *http.Request) {
	d, l := s.Music.Stats()
	writeJSONInterface(w, d, l, nil)
}

func (s *Server) apiMusicControl(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		j, err := parseSimpleJSON(r.Body)
		if err != nil {
			writeJSON(w, err)
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
				writeJSON(w, err)
				return
			}
		}
		writeJSON(w, err)
		return
	case "GET":
		d, l := s.Music.Status()
		if modifiedSince(r, l) {
			writeJSONInterface(w, d, l, nil)
		} else {
			notModified(w, l)
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
			writeJSON(w, err)
			return
		}
		decoder := json.NewDecoder(r.Body)
		var data = struct {
			OutputEnabled bool `json:"outputenabled"`
		}{}
		err = decoder.Decode(&data)
		if err != nil {
			writeJSON(w, err)
			return
		}
		writeJSON(w, s.Music.Output(id, data.OutputEnabled))
		return
	}
	if modifiedSince(r, l) {
		writeJSONInterface(w, d, l, nil)
	} else {
		notModified(w, l)
	}
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
		writeJSONInterface(w, d, s.StartTime, nil)
	} else {
		notModified(w, s.StartTime)
	}
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

func returnSong(w http.ResponseWriter, r *http.Request, song mpd.Attrs, l time.Time) {
	if modifiedSince(r, l) {
		writeJSONInterface(w, song, l, nil)
	} else {
		notModified(w, l)
	}
}

func returnListInSong(w http.ResponseWriter, r *http.Request, path string, d []mpd.Attrs, l time.Time) {
	id, err := strconv.Atoi(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if len(d) <= id || id < 0 {
		http.NotFound(w, r)
		return
	}
	returnSong(w, r, d[id], l)
}

func returnList(w http.ResponseWriter, r *http.Request, d []mpd.Attrs, l time.Time) {
	if modifiedSince(r, l) {
		writeJSONInterface(w, d, l, nil)
	} else {
		notModified(w, l)
	}
}

func makeHandleAssets(f string, data []byte) func(http.ResponseWriter, *http.Request) {
	n := time.Now().UTC()
	m := mime.TypeByExtension(path.Ext(f))
	return func(w http.ResponseWriter, r *http.Request) {
		// w.Header().Add("Content-Length", strconv.Itoa(len(data)))
		w.Header().Add("Last-Modified", n.Format(http.TimeFormat))
		if m != "" {
			w.Header().Add("Content-Type", m)
		}
		w.Write(data)
	}
}

func (s *Server) makeHandle() http.Handler {
	h := http.NewServeMux()
	h.HandleFunc("/api/version", s.apiVersion)
	h.HandleFunc("/api/music/library", s.apiMusicLibrary)
	h.HandleFunc("/api/music/library/", s.apiMusicLibraryOne)
	h.HandleFunc("/api/music/songs", s.apiMusicPlaylist)
	h.HandleFunc("/api/music/songs/", s.apiMusicPlaylistOne)
	h.HandleFunc("/api/music/songs/current", s.apiMusicCurrent)
	h.HandleFunc("/api/music/control", s.apiMusicControl)
	h.HandleFunc("/api/music/outputs", s.apiMusicOutputs)
	h.HandleFunc("/api/music/outputs/", s.apiMusicOutputs)
	h.HandleFunc("/api/music/stats", s.apiMusicStats)
	h.HandleFunc("/api/music/notify", s.apiMusicNotify)
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

// Serve serves http request.
func Serve(music MusicIF, musicDirectory string, port string) {
	s := Server{Music: music, MusicDirectory: musicDirectory, Port: port, StartTime: time.Now().UTC(), debug: true}
	s.Serve()
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
