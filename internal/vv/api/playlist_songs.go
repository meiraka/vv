package api

import (
	"context"
	"net/http"
	"sync"
)

type MPDPlaylistSongs interface {
	PlaylistInfo(context.Context) ([]map[string][]string, error)
}

type PlaylistSongsHandler struct {
	mpd       MPDPlaylistSongs
	cache     *cache
	changed   chan struct{}
	songsHook func([]map[string][]string) []map[string][]string
	data      []map[string][]string
	mu        sync.RWMutex
}

func NewPlaylistSongsHandler(mpd MPDPlaylistSongs, songsHook func([]map[string][]string) []map[string][]string) (*PlaylistSongsHandler, error) {
	cache, err := newCache([]map[string][]string{})
	if err != nil {
		return nil, err
	}
	return &PlaylistSongsHandler{
		mpd:       mpd,
		cache:     cache,
		changed:   make(chan struct{}, cap(cache.Changed())),
		songsHook: songsHook,
	}, nil

}

func (a *PlaylistSongsHandler) Update(ctx context.Context) error {
	l, err := a.mpd.PlaylistInfo(ctx)
	if err != nil {
		return err
	}
	v := a.songsHook(l)
	changed, err := a.cache.SetIfModified(v)
	if err != nil {
		return err
	}
	a.mu.Lock()
	a.data = v
	a.mu.Unlock()
	if changed {
		select {
		case a.changed <- struct{}{}:
		default:
		}
	}
	return nil
}

func (a *PlaylistSongsHandler) Cache() []map[string][]string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.data
}

// ServeHTTP responses neighbors list as json format.
func (a *PlaylistSongsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.cache.ServeHTTP(w, r)
}

// Changed returns neighbors list update event chan.
func (a *PlaylistSongsHandler) Changed() <-chan struct{} {
	return a.changed
}

// Close closes update event chan.
func (a *PlaylistSongsHandler) Close() {
	a.cache.Close()
}
