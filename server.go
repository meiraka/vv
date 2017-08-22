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
	"strconv"
	"strings"
	"time"
)

var version string
var startTime time.Time

const musicDirectory = "/music_directory/"

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

type apiHandler struct {
	player   Music
	upgrader websocket.Upgrader
	devMode  bool
}

func (a *apiHandler) playlist(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		d, l := a.player.Playlist()
		a.returnList(w, r, d, l)
	case "POST":
		decoder := json.NewDecoder(r.Body)
		var s struct {
			Action string   `json:"action"`
			Keys   []string `json:"keys"`
			URI    string   `json:"uri"`
		}
		err := decoder.Decode(&s)
		if err == nil {
			a.player.SortPlaylist(s.Keys, s.URI)
		}
		writeJSON(w, err)
	}
}

func (a *apiHandler) playlistOne(w http.ResponseWriter, r *http.Request) {
	p := strings.Replace(r.URL.Path, "/api/music/songs/", "", -1)
	if p == "" {
		a.playlist(w, r)
		return
	}
	d, l := a.player.Playlist()
	a.returnListInSong(w, r, p, d, l)
}

func (a *apiHandler) library(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		d, l := a.player.Library()
		a.returnList(w, r, d, l)
	case "POST":
		decoder := json.NewDecoder(r.Body)
		var s struct {
			Action string `json:"action"`
		}
		err := decoder.Decode(&s)
		if err == nil {
			if s.Action == "rescan" {
				a.player.RescanLibrary()
			} else {
				err = errors.New("unknown action: " + s.Action)
			}
		}
		writeJSON(w, err)
	}
}

func (a *apiHandler) libraryOne(w http.ResponseWriter, r *http.Request) {
	p := strings.Replace(r.URL.Path, "/api/music/library/", "", -1)
	if p == "" {
		a.library(w, r)
		return
	}
	d, l := a.player.Library()
	a.returnListInSong(w, r, p, d, l)
}

func (a *apiHandler) current(w http.ResponseWriter, r *http.Request) {
	d, l := a.player.Current()
	a.returnSong(w, r, d, l)
}

func (a *apiHandler) stats(w http.ResponseWriter, r *http.Request) {
	d, l := a.player.Stats()
	writeJSONInterface(w, d, l, nil)
}

func (a *apiHandler) control(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		j, err := parseSimpleJSON(r.Body)
		if err != nil {
			writeJSON(w, err)
			return
		}
		funcs := []func() error{
			func() error { return j.execIfInt("volume", a.player.Volume) },
			func() error { return j.execIfBool("repeat", a.player.Repeat) },
			func() error { return j.execIfBool("random", a.player.Random) },
			func() error {
				return j.execIfString("state", func(s string) error {
					switch s {
					case "play":
						return a.player.Play()
					case "pause":
						return a.player.Pause()
					case "next":
						return a.player.Next()
					case "prev":
						return a.player.Prev()
					}
					return errors.New("unknown state value: " + s)
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
		d, l := a.player.Status()
		if modifiedSince(r, l) {
			writeJSONInterface(w, d, l, nil)
		} else {
			notModified(w, l)
		}
	}
}

func (a *apiHandler) outputs(w http.ResponseWriter, r *http.Request) {
	d, l := a.player.Outputs()
	if r.Method == "POST" {
		id, err := strconv.Atoi(
			strings.Replace(r.URL.Path, "/api/music/outputs/", "", -1),
		)
		if err != nil {
			writeJSON(w, err)
			return
		}
		decoder := json.NewDecoder(r.Body)
		var s = struct {
			OutputEnabled bool `json:"outputenabled"`
		}{}
		err = decoder.Decode(&s)
		if err != nil {
			writeJSON(w, err)
			return
		}
		writeJSON(w, a.player.Output(id, s.OutputEnabled))
		return
	}
	if modifiedSince(r, l) {
		writeJSONInterface(w, d, l, nil)
	} else {
		notModified(w, l)
	}
}

func (a *apiHandler) version(w http.ResponseWriter, r *http.Request) {
	if modifiedSince(r, startTime) {
		vvPostfix := ""
		if a.devMode {
			vvPostfix = vvPostfix + " dev mode"
		}
		d := map[string]string{"vv": version + vvPostfix}
		writeJSONInterface(w, d, startTime, nil)
	} else {
		notModified(w, startTime)
	}
}

func (a *apiHandler) notify(w http.ResponseWriter, r *http.Request) {
	c, err := a.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer c.Close()
	n := make(chan string, 10)
	a.player.Subscribe(n)
	defer a.player.Unsubscribe(n)
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

func (a *apiHandler) returnSong(w http.ResponseWriter, r *http.Request, s mpd.Attrs, l time.Time) {
	if modifiedSince(r, l) {
		writeJSONInterface(w, s, l, nil)
	} else {
		notModified(w, l)
	}
}

func (a *apiHandler) returnListInSong(w http.ResponseWriter, r *http.Request, path string, d []mpd.Attrs, l time.Time) {
	id, err := strconv.Atoi(path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if len(d) <= id || id < 0 {
		http.NotFound(w, r)
		return
	}
	s := d[id]
	a.returnSong(w, r, s, l)
}

func (a *apiHandler) returnList(w http.ResponseWriter, r *http.Request, d []mpd.Attrs, l time.Time) {
	if modifiedSince(r, l) {
		writeJSONInterface(w, d, l, nil)
	} else {
		notModified(w, l)
	}
}

func makeHandleAssets(f string, data []byte) func(http.ResponseWriter, *http.Request) {
	n := time.Now()
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

func makeHandle(p Music, c Config, bindata bool) http.Handler {
	api := apiHandler{player: p, devMode: false}
	h := http.NewServeMux()
	h.HandleFunc("/api/version", api.version)
	h.HandleFunc("/api/music/library", api.library)
	h.HandleFunc("/api/music/library/", api.libraryOne)
	h.HandleFunc("/api/music/songs", api.playlist)
	h.HandleFunc("/api/music/songs/", api.playlistOne)
	h.HandleFunc("/api/music/songs/current", api.current)
	h.HandleFunc("/api/music/control", api.control)
	h.HandleFunc("/api/music/outputs", api.outputs)
	h.HandleFunc("/api/music/outputs/", api.outputs)
	h.HandleFunc("/api/music/stats", api.stats)
	h.HandleFunc("/api/music/notify", api.notify)
	fs := http.StripPrefix(musicDirectory, http.FileServer(http.Dir(c.Mpd.MusicDirectory)))
	h.HandleFunc(musicDirectory, func(w http.ResponseWriter, r *http.Request) {
		fs.ServeHTTP(w, r)
	})
	for _, f := range AssetNames() {
		p := "/" + f
		if f == "assets/app.html" {
			p = "/"
		}
		_, err := os.Stat(f)
		if os.IsNotExist(err) || bindata {
			data, _ := Asset(f)
			h.HandleFunc(p, makeHandleAssets(f, data))
		} else {
			api.devMode = true
			func(path, rpath string) {
				h.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
					http.ServeFile(w, r, rpath)
				})
			}(p, f)
		}
	}
	return h
}

// App serves http request.
func App(p Music, c Config) {
	startTime = time.Now()
	handler := makeHandle(p, c, false)
	http.ListenAndServe(fmt.Sprintf(":%s", c.Server.Port), handler)
}

// Music Represents music player.
type Music interface {
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
