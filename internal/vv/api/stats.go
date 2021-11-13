package api

import (
	"context"
	"net/http"
	"strconv"
)

type httpMusicStats struct {
	Uptime          int `json:"uptime"`
	Playtime        int `json:"playtime"`
	Artists         int `json:"artists"`
	Albums          int `json:"albums"`
	Songs           int `json:"songs"`
	LibraryPlaytime int `json:"library_playtime"`
	LibraryUpdate   int `json:"library_update"`
}

type MPDStats interface {
	Stats(context.Context) (map[string]string, error)
}

// StatsHandler provides mpd stats.
type StatsHandler struct {
	mpd   MPDStats
	cache *cache
}

// NewStatsHandler initilize Stats cache with mpd connection.
func NewStatsHandler(mpd MPDStats) (*StatsHandler, error) {
	c, err := newCache(&httpMusicStats{})
	if err != nil {
		return nil, err
	}
	return &StatsHandler{
		mpd:   mpd,
		cache: c,
	}, nil
}

func (a *StatsHandler) Update(ctx context.Context) error {
	err := a.update(ctx)
	return err
}

func (a *StatsHandler) update(ctx context.Context) error {
	s, err := a.mpd.Stats(ctx)
	if err != nil {
		return err
	}
	ret := &httpMusicStats{}
	if _, ok := s["artists"]; ok {
		ret.Artists, err = strconv.Atoi(s["artists"])
		if err != nil {
			return err
		}
	}
	if _, ok := s["albums"]; ok {
		ret.Albums, err = strconv.Atoi(s["albums"])
		if err != nil {
			return err
		}
	}
	if _, ok := s["songs"]; ok {
		ret.Songs, err = strconv.Atoi(s["songs"])
		if err != nil {
			return err
		}
	}
	if _, ok := s["uptime"]; ok {
		ret.Uptime, err = strconv.Atoi(s["uptime"])
		if err != nil {
			return err
		}
	}
	if _, ok := s["db_playtime"]; ok {
		ret.LibraryPlaytime, err = strconv.Atoi(s["db_playtime"])
		if err != nil {
			return err
		}
	}
	if _, ok := s["db_update"]; ok {
		ret.LibraryUpdate, err = strconv.Atoi(s["db_update"])
		if err != nil {
			return err
		}
	}
	if _, ok := s["playtime"]; ok {
		ret.Playtime, err = strconv.Atoi(s["playtime"])
		if err != nil {
			return err
		}
	}
	// force update to Last-Modified header to calc current playing time
	return a.cache.Set(ret)
}

// ServeHTTP responses stats as json format.
func (a *StatsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.cache.ServeHTTP(w, r)
}

// Changed returns stats update event chan.
func (a *StatsHandler) Changed() <-chan struct{} {
	return a.cache.Changed()
}

// Close closes update event chan.
func (a *StatsHandler) Close() {
	a.cache.Close()
}
