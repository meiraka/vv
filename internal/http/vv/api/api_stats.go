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

const pathAPIMusicStats = "/api/music/stats"

var updateStatsIntKeys = []string{"artists", "albums", "songs", "uptime", "db_playtime", "db_update", "playtime"}

func (a *api) StatsHandler() http.Handler {
	return a.jsonCache.Handler(pathAPIMusicStats)
}

func (a *api) updateStats(ctx context.Context) error {
	s, err := a.client.Stats(ctx)
	if err != nil {
		return err
	}
	ret := &httpMusicStats{}
	for _, k := range updateStatsIntKeys {
		v, ok := s[k]
		if !ok {
			continue
		}
		iv, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		switch k {
		case "artists":
			ret.Artists = iv
		case "albums":
			ret.Albums = iv
		case "songs":
			ret.Songs = iv
		case "uptime":
			ret.Uptime = iv
		case "db_playtime":
			ret.LibraryPlaytime = iv
		case "db_update":
			ret.LibraryUpdate = iv
		case "playtime":
			ret.Playtime = iv
		}
	}

	// force update to Last-Modified header to calc current playing time
	return a.jsonCache.Set(pathAPIMusicStats, ret)
}
