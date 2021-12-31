package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/meiraka/vv/internal/mpd"
	"github.com/meiraka/vv/internal/songs"
)

type httpPlaylistInfo struct {
	// current track
	Current *int `json:"current,omitempty"`
	// sort functions
	Sort    []string     `json:"sort,omitempty"`
	Filters [][2]*string `json:"filters,omitempty"`
	Must    int          `json:"must,omitempty"`
}

// PlaylistHandler provides current playlist sort function.
type PlaylistHandler struct {
	mpd         MPDPlaylist
	library     []map[string][]string
	librarySort []map[string][]string
	playlist    []map[string][]string
	cache       *cache
	data        *httpPlaylistInfo
	mu          sync.RWMutex
	sem         chan struct{}
	config      *Config
}

type MPDPlaylist interface {
	Play(context.Context, int) error
	ExecCommandList(context.Context, *mpd.CommandList) error
}

func NewPlaylistHandler(mpd MPDPlaylist, config *Config) (*PlaylistHandler, error) {
	c, err := newCache(&httpPlaylistInfo{})
	if err != nil {
		return nil, err
	}
	sem := make(chan struct{}, 1)
	sem <- struct{}{}
	return &PlaylistHandler{
		mpd:    mpd,
		cache:  c,
		data:   &httpPlaylistInfo{},
		sem:    sem,
		config: config,
	}, nil
}

func (a *PlaylistHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		a.cache.ServeHTTP(w, r)
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
	case <-a.sem:
	default:
		// TODO: switch to better status code
		writeHTTPError(w, http.StatusServiceUnavailable, errors.New("updating playlist"))
		return
	}

	a.mu.Lock()
	librarySort, filters, newpos := songs.WeakFilterSort(a.library, req.Sort, req.Filters, req.Must, 9999, *req.Current)
	a.librarySort = librarySort
	update := !songs.SortEqual(a.playlist, a.librarySort)
	a.mu.Unlock()
	cl := &mpd.CommandList{}
	cl.Clear()
	for i := range a.librarySort {
		cl.Add(a.librarySort[i]["file"][0])
	}
	cl.Play(newpos)
	if !update {
		defer func() { a.sem <- struct{}{} }()
		a.updateSort(req.Sort, filters, req.Must)
		a.mu.Lock()
		a.cache.SetIfModified(a.data)
		a.mu.Unlock()
		now := time.Now().UTC()
		ctx := r.Context()
		if err := a.mpd.Play(ctx, newpos); err != nil {
			writeHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		r.Method = http.MethodGet
		a.cache.ServeHTTP(w, setUpdateTime(r, now))
		return
	}
	r.Method = http.MethodGet
	a.cache.ServeHTTP(w, setUpdateTime(r, time.Now().UTC()))
	go func() {
		defer func() { a.sem <- struct{}{} }()
		ctx, cancel := context.WithTimeout(context.Background(), a.config.BackgroundTimeout)
		defer cancel()
		if err := a.mpd.ExecCommandList(ctx, cl); err != nil {
			return
		}
		a.updateSort(req.Sort, filters, req.Must)
	}()
}

func (a *PlaylistHandler) UpdateCurrent(pos int) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	data := &httpPlaylistInfo{
		Current: &pos,
		Sort:    a.data.Sort,
		Filters: a.data.Filters,
		Must:    a.data.Must,
	}
	_, err := a.cache.SetIfModified(data)
	if err != nil {
		return err
	}
	a.data = data
	return nil
}

func (a *PlaylistHandler) updateSort(sort []string, filters [][2]*string, must int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	data := &httpPlaylistInfo{
		Current: a.data.Current,
		Sort:    sort,
		Filters: filters,
		Must:    must,
	}
	a.data = data
}

func (a *PlaylistHandler) UpdatePlaylistSongs(i []map[string][]string) {
	a.mu.Lock()
	a.playlist = i
	unsort := a.data.Sort != nil && !songs.SortEqual(a.playlist, a.librarySort)
	a.mu.Unlock()
	if unsort {
		a.updateSort(nil, nil, 0)
		a.mu.Lock()
		a.cache.SetIfModified(a.data)
		a.mu.Unlock()
	}
}

func (a *PlaylistHandler) UpdateLibrarySongs(i []map[string][]string) {
	a.mu.Lock()
	a.library = songs.Copy(i)
	a.librarySort = nil
	a.mu.Unlock()
}

// Changed returns library song list update event chan.
func (a *PlaylistHandler) Changed() <-chan struct{} {
	return a.cache.Changed()
}

// Close closes update event chan.
func (a *PlaylistHandler) Close() {
	a.cache.Close()
}

// Wait waits playlist updates.
func (a *PlaylistHandler) Wait(ctx context.Context) error {
	select {
	case <-a.sem:
		a.sem <- struct{}{}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Shutdown waits playlist updates. Shutdown does not allow no playlist updates request.
func (a *PlaylistHandler) Shutdown(ctx context.Context) error {
	select {
	case <-a.sem:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
