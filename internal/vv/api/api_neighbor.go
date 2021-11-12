package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/meiraka/vv/internal/mpd"
)

const (
	pathAPIMusicStorageNeighbors = "/api/music/storage/neighbors"
)

type neighbors struct {
	client    *mpd.Client
	jsonCache *jsonCache
	handler   http.Handler
}

func newNeighbors(client *mpd.Client, cache *jsonCache) *neighbors {
	return &neighbors{
		client:    client,
		jsonCache: cache,
		handler:   cache.Handler(pathAPIMusicStorageNeighbors),
	}
}

func (a *neighbors) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.handler.ServeHTTP(w, r)
}

func (a *neighbors) Update(ctx context.Context) error {
	ret := map[string]*httpStorage{}
	ms, err := a.client.ListNeighbors(ctx)
	if err != nil {
		// skip command error to support old mpd
		var perr *mpd.CommandError
		if errors.As(err, &perr) {
			a.jsonCache.SetIfModified(pathAPIMusicStorageNeighbors, ret)
			return nil
		}
		return err
	}
	for _, m := range ms {
		ret[m["name"]] = &httpStorage{
			URI: stringPtr(m["neighbor"]),
		}
	}
	a.jsonCache.SetIfModified(pathAPIMusicStorageNeighbors, ret)
	return nil
}
