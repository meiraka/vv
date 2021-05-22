package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

const (
	pathAPIMusicLibrary      = "/api/music/library"
	pathAPIMusicLibrarySongs = "/api/music/library/songs"
)

type httpLibraryInfo struct {
	Updating bool `json:"updating"`
}

func (a *api) LibraryHandler() http.HandlerFunc {
	get := a.jsonCache.Handler(pathAPIMusicLibrary)
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			get.ServeHTTP(w, r)
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
		if _, err := a.client.Update(ctx, ""); err != nil {
			writeHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		r.Method = http.MethodGet
		get.ServeHTTP(w, setUpdateTime(r, now))
	}
}

func (a *api) LibrarySongsHandler() http.Handler {
	return a.jsonCache.Handler(pathAPIMusicLibrarySongs)
}

func (a *api) updateLibrarySongs(ctx context.Context) error {
	l, err := a.client.ListAllInfo(ctx, "/")
	if err != nil {
		return err
	}
	v := a.convSongs(l)
	// force update to skip []byte compare
	if err := a.jsonCache.Set(pathAPIMusicLibrarySongs, v); err != nil {
		return err
	}
	a.mu.Lock()
	a.library = v
	a.playlistInfo.Sort = nil
	a.playlistInfo.Filters = nil
	a.playlistInfo.Must = 0
	a.librarySort = nil
	a.updatePlaylist()

	a.mu.Unlock()
	return nil
}
