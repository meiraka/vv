package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/fhs/gompd/mpd"
	"mime"
	"net/http"
	"os"
	"path"
	"strconv"
	"strings"
	"time"
)

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

func notModified(w http.ResponseWriter, l time.Time) {
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Last-Modified", l.Format(http.TimeFormat))
	w.WriteHeader(304)
	return
}

type apiHandler struct {
	player Music
}

func (h *apiHandler) playlist(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		d, l := h.player.Playlist()
		if modified(r, l) {
			writeJSONInterface(w, d, l, nil)
		} else {
			notModified(w, l)
		}
	case "POST":
		decoder := json.NewDecoder(r.Body)
		var s struct {
			Action string   `json:"action"`
			Keys   []string `json:"keys"`
			URI    string   `json:"uri"`
		}
		err := decoder.Decode(&s)
		if err == nil {
			h.player.SortPlaylist(s.Keys, s.URI)
		}
		writeJSON(w, err)
	}
}

func (h *apiHandler) library(w http.ResponseWriter, r *http.Request) {
	d, l := h.player.Library()
	if modified(r, l) {
		writeJSONInterface(w, d, l, nil)
	} else {
		notModified(w, l)
	}
}

func (h *apiHandler) current(w http.ResponseWriter, r *http.Request) {
	d, l := h.player.Current()
	if modified(r, l) {
		writeJSONInterface(w, d, l, nil)
	} else {
		notModified(w, l)
	}
}

func (h *apiHandler) control(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		j, err := parseSimpleJSON(r.Body)
		if err != nil {
			writeJSON(w, err)
			return
		}
		funcs := []func() error{
			func() error {
				return j.execIfInt("volume", func(i int) error {
					return h.player.Volume(i)
				})
			},
			func() error {
				return j.execIfBool("repeat", func(b bool) error {
					return h.player.Repeat(b)
				})
			},
			func() error {
				return j.execIfBool("random", func(b bool) error {
					return h.player.Random(b)
				})
			},
			func() error {
				return j.execIfString("state", func(s string) error {
					switch s {
					case "play":
						return h.player.Play()
					case "pause":
						return h.player.Pause()
					case "next":
						return h.player.Next()
					case "prev":
						return h.player.Prev()
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
		// TODO: post action
		method := r.FormValue("action")
		if method == "prev" {
			writeJSON(w, h.player.Prev())
			return
		} else if method == "play" {
			writeJSON(w, h.player.Play())
			return
		} else if method == "pause" {
			writeJSON(w, h.player.Pause())
			return
		} else if method == "next" {
			writeJSON(w, h.player.Next())
			return
		} else {
			d, l := h.player.Status()
			if modified(r, l) {
				writeJSONInterface(w, d, l, nil)
			} else {
				notModified(w, l)
			}
		}
	}
}

func (h *apiHandler) outputs(w http.ResponseWriter, r *http.Request) {
	d, l := h.player.Outputs()
	if r.Method == "POST" {
		id, err := strconv.Atoi(
			strings.Replace(r.URL.Path, "/api/outputs/", "", -1),
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
		writeJSON(w, h.player.Output(id, s.OutputEnabled))
		return
	}
	if modified(r, l) {
		writeJSONInterface(w, d, l, nil)
	} else {
		notModified(w, l)
	}
}

func modified(r *http.Request, l time.Time) bool {
	return r.Header.Get("If-Modified-Since") != l.Format(http.TimeFormat)
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

func setHandle(p Music) {
	var api = new(apiHandler)
	api.player = p
	http.HandleFunc("/api/library", api.library)
	http.HandleFunc("/api/songs", api.playlist)
	http.HandleFunc("/api/songs/current", api.current)
	http.HandleFunc("/api/control", api.control)
	http.HandleFunc("/api/outputs", api.outputs)
	http.HandleFunc("/api/outputs/", api.outputs)
	for _, f := range AssetNames() {
		p := "/" + f
		if f == "assets/app.html" {
			p = "/"
		}
		_, err := os.Stat(f)
		if !os.IsNotExist(err) {
			func(path, rpath string) {
				http.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
					http.ServeFile(w, r, rpath)
				})
			}(p, f)
		} else {
			data, _ := Asset(f)
			http.HandleFunc(p, makeHandleAssets(f, data))
		}
	}
}

// App serves http request.
func App(p Music, config ServerConfig) {
	setHandle(p)
	http.ListenAndServe(fmt.Sprintf(":%s", config.Port), nil)
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
	Current() (mpd.Attrs, time.Time)
	Status() (PlayerStatus, time.Time)
	Output(int, bool) error
	Outputs() ([]mpd.Attrs, time.Time)
	SortPlaylist([]string, string) error
}
