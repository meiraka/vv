package main

import (
	"encoding/json"
	"fmt"
	"github.com/fhs/gompd/mpd"
	"net/http"
	"time"
)

type m map[string]interface{}

func writeJSONAttrList(w http.ResponseWriter, d []mpd.Attrs, l time.Time, err error) {
	w.Header().Add("Last-Modified", l.Format(http.TimeFormat))
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	v := m{"errors": err, "data": d}
	b, jsonerr := json.Marshal(v)
	if jsonerr != nil {
		return
	}
	fmt.Fprintf(w, string(b))
	return
}

func writeJSONAttr(w http.ResponseWriter, d mpd.Attrs, l time.Time, err error) {
	w.Header().Add("Last-Modified", l.Format(http.TimeFormat))
	w.Header().Add("Content-Type", "application/json")
	v := m{"errors": err, "data": d}
	b, jsonerr := json.Marshal(v)
	if jsonerr != nil {
		return
	}
	fmt.Fprintf(w, string(b))
	return
}

func writeJSON(w http.ResponseWriter, err error) {
	w.Header().Add("Content-Type", "application/json")
	v := m{"errors": err}
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

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, World")
}

type apiHandler struct {
	player *Player
}

func (h *apiHandler) playlist(w http.ResponseWriter, r *http.Request) {
	d, l := h.player.Playlist()
	if modified(r, l) {
		writeJSONAttrList(w, d, l, nil)
	} else {
		notModified(w, l)
	}
}

func (h *apiHandler) library(w http.ResponseWriter, r *http.Request) {
	d, l := h.player.Library()
	if modified(r, l) {
		writeJSONAttrList(w, d, l, nil)
	} else {
		notModified(w, l)
	}
}

func (h *apiHandler) current(w http.ResponseWriter, r *http.Request) {
	method := r.FormValue("action")
	if method == "prev" {
		writeJSON(w, h.player.Prev())
	} else if method == "play" {
		writeJSON(w, h.player.Play())
	} else if method == "pause" {
		writeJSON(w, h.player.Pause())
	} else if method == "next" {
		writeJSON(w, h.player.Next())
	} else if method == "detail" {
		d, l := h.player.Comments()
		if modified(r, l) {
			writeJSONAttr(w, d, l, nil)
		} else {
			notModified(w, l)
		}
	} else {
		d, l := h.player.Current()
		if modified(r, l) {
			writeJSONAttr(w, d, l, nil)
		} else {
			notModified(w, l)
		}
	}
}

func modified(r *http.Request, l time.Time) bool {
	return r.Header.Get("If-Modified-Since") != l.Format(http.TimeFormat)
}

// App serves http request.
func App(p *Player, config ServerConfig) {
	var api = new(apiHandler)
	api.player = p
	http.HandleFunc("/api/library", api.library)
	http.HandleFunc("/api/songs", api.playlist)
	http.HandleFunc("/api/songs/current", api.current)
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "app.html")
	})
	http.HandleFunc("/app.css", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "app.css")
	})
	http.HandleFunc("/app.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "app.js")
	})
	http.HandleFunc("/jquery-3.1.1.js", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "jquery-3.1.1.js")
	})
	http.ListenAndServe(fmt.Sprintf(":%s", config.Port), nil)
}
