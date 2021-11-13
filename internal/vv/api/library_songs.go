package api

import (
	"context"
	"net/http"
	"sync"
)

type MPDLibrarySongs interface {
	ListAllInfo(context.Context, string) ([]map[string][]string, error)
}

func NewLibrarySongsHandler(mpd MPDLibrarySongs, songsHook func([]map[string][]string) []map[string][]string) (*LibrarySongsHandler, error) {
	cache, err := newCache([]map[string][]string{})
	if err != nil {
		return nil, err
	}
	return &LibrarySongsHandler{
		mpd:       mpd,
		cache:     cache,
		changed:   make(chan struct{}, cap(cache.Changed())),
		songsHook: songsHook,
	}, nil

}

type LibrarySongsHandler struct {
	mpd       MPDLibrarySongs
	cache     *cache
	changed   chan struct{}
	songsHook func([]map[string][]string) []map[string][]string
	data      []map[string][]string
	mu        sync.RWMutex
}

func (a *LibrarySongsHandler) Update(ctx context.Context) error {
	l, err := a.mpd.ListAllInfo(ctx, "/")
	if err != nil {
		return err
	}
	v := a.songsHook(l)
	// force update to skip []byte compare
	if err := a.cache.Set(v); err != nil {
		return err
	}
	a.mu.Lock()
	a.data = v
	a.mu.Unlock()
	select {
	case a.changed <- struct{}{}:
	default:
	}
	return nil
}

func (a *LibrarySongsHandler) Cache() []map[string][]string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.data
}

// ServeHTTP responses library song list as json format.
func (a *LibrarySongsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.cache.ServeHTTP(w, r)
}

// Changed returns library song list update event chan.
func (a *LibrarySongsHandler) Changed() <-chan struct{} {
	return a.changed
}

// Close closes update event chan.
func (a *LibrarySongsHandler) Close() {
	a.cache.Close()
}
