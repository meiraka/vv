package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/meiraka/vv/internal/mpd"
	"github.com/meiraka/vv/internal/songs"
)

// keys for playlist json cache
const (
	pathAPIMusicPlaylist             = "/api/music/playlist"
	pathAPIMusicPlaylistSongs        = "/api/music/playlist/songs"
	pathAPIMusicPlaylistSongsCurrent = "/api/music/playlist/songs/current"
)

type httpPlaylistInfo struct {
	Current *int         `json:"current,omitempty"`
	Sort    []string     `json:"sort,omitempty"`
	Filters [][2]*string `json:"filters,omitempty"`
	Must    int          `json:"must,omitempty"`
}

func (a *api) PlaylistHandler() http.HandlerFunc {
	sem := make(chan struct{}, 1)
	sem <- struct{}{}
	get := a.jsonCache.Handler(pathAPIMusicPlaylist)
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			get.ServeHTTP(w, r)
			return
		}
		var req httpPlaylistInfo
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeHTTPError(w, http.StatusBadRequest, err)
			return
		}

		if req.Current == nil || req.Filters == nil || req.Sort == nil {
			writeHTTPError(w, http.StatusBadRequest, errors.New("current, filters and sort fields are required"))
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

		a.mu.Lock()
		librarySort, filters, newpos := songs.WeakFilterSort(a.library, req.Sort, req.Filters, req.Must, 9999, *req.Current)
		update := !songs.SortEqual(a.playlist, librarySort)
		cl := &mpd.CommandList{}
		cl.Clear()
		for i := range librarySort {
			cl.Add(librarySort[i]["file"][0])
		}
		cl.Play(newpos)
		a.playlistInfo.Sort = req.Sort
		a.playlistInfo.Filters = filters
		a.playlistInfo.Must = req.Must
		a.librarySort = librarySort
		a.mu.Unlock()
		if !update {
			now := time.Now().UTC()
			ctx := r.Context()
			if err := a.client.Play(ctx, newpos); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				a.mu.Lock()
				a.playlistInfo.Sort = nil
				a.playlistInfo.Filters = nil
				a.playlistInfo.Must = 0
				a.librarySort = nil
				a.mu.Unlock()
				return
			}
			r.Method = http.MethodGet
			get.ServeHTTP(w, setUpdateTime(r, now))
			return
		}
		r.Method = http.MethodGet
		get.ServeHTTP(w, setUpdateTime(r, time.Now().UTC()))
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), a.config.BackgroundTimeout)
			defer cancel()
			select {
			case <-sem:
			case <-ctx.Done():
				return
			}
			defer func() { sem <- struct{}{} }()
			if err := a.client.ExecCommandList(ctx, cl); err != nil {
				a.mu.Lock()
				a.playlistInfo.Sort = nil
				a.playlistInfo.Filters = nil
				a.playlistInfo.Must = 0
				a.librarySort = nil
				a.mu.Unlock()
				return
			}
		}()
	}
}

func (a *api) PlaylistSongsHandler() http.Handler {
	return a.jsonCache.Handler(pathAPIMusicPlaylistSongs)
}

func (a *api) PlaylistSongsCurrentHandler() http.Handler {
	return a.jsonCache.Handler(pathAPIMusicPlaylistSongsCurrent)
}

func (a *api) updatePlaylist() error {
	return a.jsonCache.SetIfModified(pathAPIMusicPlaylist, a.playlistInfo)
}

func (a *api) updatePlaylistSongs(ctx context.Context) error {
	l, err := a.client.PlaylistInfo(ctx)
	if err != nil {
		return err
	}
	v := a.convSongs(l)
	// force update to skip []byte compare
	if err := a.jsonCache.Set(pathAPIMusicPlaylistSongs, v); err != nil {
		return err
	}

	a.mu.Lock()
	a.playlist = v
	if a.playlistInfo.Sort != nil && !songs.SortEqual(a.playlist, a.librarySort) {
		a.playlistInfo.Sort = nil
		a.playlistInfo.Filters = nil
		a.playlistInfo.Must = 0
		a.librarySort = nil
		a.updatePlaylist()
	}
	a.mu.Unlock()

	return err
}

func (a *api) updatePlaylistSongsCurrent(ctx context.Context) error {
	l, err := a.client.CurrentSong(ctx)
	if err != nil {
		return err
	}
	l, _ = a.convSong(l)
	return a.jsonCache.SetIfModified(pathAPIMusicPlaylistSongsCurrent, l)
}
