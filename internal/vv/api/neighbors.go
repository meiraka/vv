package api

import (
	"context"
	"errors"
	"net/http"

	"github.com/meiraka/vv/internal/mpd"
)

// NeighborsHandler provides neighbor storage name and uri.
type NeighborsHandler struct {
	mpd interface {
		ListNeighbors(context.Context) ([]map[string]string, error)
	}
	cache  *cache
	logger Logger
}

// NewNeighborsHandler initilize Neighbors cache with mpd connection.
func NewNeighborsHandler(mpd interface {
	ListNeighbors(context.Context) ([]map[string]string, error)
}, logger Logger) (*NeighborsHandler, error) {
	c, err := newCache(map[string]*httpStorage{})
	if err != nil {
		return nil, err
	}
	return &NeighborsHandler{
		mpd:    mpd,
		cache:  c,
		logger: logger,
	}, nil
}

// Update updates neighbors list.
func (a *NeighborsHandler) Update(ctx context.Context) error {
	ret := map[string]*httpStorage{}
	ms, err := a.mpd.ListNeighbors(ctx)
	if err != nil {
		// skip command error to support old mpd
		var perr *mpd.CommandError
		if errors.As(err, &perr) {
			a.cache.SetIfModified(ret)
			a.logger.Debugf("vv/api: neighbors: %v", err)
			return nil
		}
		return err
	}
	for _, m := range ms {
		ret[m["name"]] = &httpStorage{
			URI: stringPtr(m["neighbor"]),
		}
	}
	a.cache.SetIfModified(ret)
	return nil
}

// ServeHTTP responses neighbors list as json format.
func (a *NeighborsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.cache.ServeHTTP(w, r)
}

// Changed returns neighbors list update event chan.
func (a *NeighborsHandler) Changed() <-chan struct{} {
	return a.cache.Changed()
}

// Close closes update event chan.
func (a *NeighborsHandler) Close() {
	a.cache.Close()
}
