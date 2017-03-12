package main

import (
	"encoding/json"
	"fmt"
	"github.com/fhs/gompd/mpd"
	"net/http"
)

func writeJSON(w http.ResponseWriter, d []mpd.Attrs) {
	b, err := json.Marshal(d)
	if err != nil {
		return
	}
	fmt.Fprintf(w, string(b))
	return
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, World")
}

type apiHandler struct {
	player *Player
}

func (h *apiHandler) playlist(w http.ResponseWriter, r *http.Request) {
	playlist := h.player.Playlist()
	writeJSON(w, playlist)

}

func (h *apiHandler) library(w http.ResponseWriter, r *http.Request) {
	library := h.player.Library()
	writeJSON(w, library)
}

// App serves http request.
func App(p *Player, config ServerConfig) {
	var api = new(apiHandler)
	api.player = p
	http.HandleFunc("/", handler)
	http.HandleFunc("/api/playlist", api.playlist)
	http.HandleFunc("/api/library", api.library)
	http.ListenAndServe(fmt.Sprintf(":%s", config.Port), nil)
}
