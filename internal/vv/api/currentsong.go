package api

import (
	"context"
	"net/http"
)

type MPDCurrentSong interface {
	CurrentSong(context.Context) (map[string][]string, error)
}

type CurrentSongHandler struct {
	mpd      MPDCurrentSong
	cache    *cache
	songHook func(map[string][]string) map[string][]string
}

func NewCurrentSongHandler(mpd MPDCurrentSong, songHook func(map[string][]string) map[string][]string) (*CurrentSongHandler, error) {
	c, err := newCache(map[string][]string{})
	if err != nil {
		return nil, err
	}
	return &CurrentSongHandler{
		mpd:      mpd,
		cache:    c,
		songHook: songHook,
	}, nil
}

func (a *CurrentSongHandler) Update(ctx context.Context) error {
	l, err := a.mpd.CurrentSong(ctx)
	if err != nil {
		return err
	}
	_, err = a.cache.SetIfModified(a.songHook(l))
	return err
}

func (a *CurrentSongHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.cache.ServeHTTP(w, r)
}

func (a *CurrentSongHandler) Changed() <-chan struct{} {
	return a.cache.Changed()
}

func (a *CurrentSongHandler) Close() {
	a.cache.Close()
}
