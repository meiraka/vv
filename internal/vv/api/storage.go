package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/meiraka/vv/internal/mpd"
)

type httpStorage struct {
	URI      *string `json:"uri,omitempty"`
	Updating bool    `json:"updating,omitempty"`
}

// MPDStorage represents mpd api for Storage API.
type MPDStorage interface {
	ListMounts(context.Context) ([]map[string]string, error)
	Mount(context.Context, string, string) error
	Unmount(context.Context, string) error
	Update(context.Context, string) (map[string]string, error)
}

// StorageHandler provides mount, unmount, list storage api.
type StorageHandler struct {
	mpd   MPDStorage
	cache *cache
}

func NewStorageHandler(mpd MPDStorage) (*StorageHandler, error) {
	c, err := newCache(map[string]*httpStorage{})
	if err != nil {
		return nil, err
	}
	return &StorageHandler{
		mpd:   mpd,
		cache: c,
	}, nil
}

func (a *StorageHandler) Update(ctx context.Context) error {
	ret := map[string]*httpStorage{}
	ms, err := a.mpd.ListMounts(ctx)
	if err != nil {
		// skip command error to support old mpd
		var perr *mpd.CommandError
		if errors.As(err, &perr) {
			a.cache.SetIfModified(ret)
			return nil
		}
		return err
	}
	for _, m := range ms {
		ret[m["mount"]] = &httpStorage{
			URI: stringPtr(m["storage"]),
		}
	}
	a.cache.SetIfModified(ret)
	return nil
}

// ServeHTTP responses storage api.
func (a *StorageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		a.cache.ServeHTTP(w, r)
		return
	}
	var req map[string]*httpStorage
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeHTTPError(w, http.StatusBadRequest, err)
		return
	}
	ctx := r.Context()
	for k, v := range req {
		if k == "" {
			writeHTTPError(w, http.StatusBadRequest, errors.New("storage name is empty"))
			return
		}
		if v != nil && v.Updating {
			// TODO: This is not intuitive
			if _, err := a.mpd.Update(ctx, k); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			// updating request does not affect response and always returns 202
			now := time.Now().UTC()
			r = setUpdateTime(r, now)
		} else if v != nil && v.URI != nil {
			if err := a.mpd.Mount(ctx, k, *v.URI); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			now := time.Now().UTC()
			r = setUpdateTime(r, now)
			if _, err := a.mpd.Update(ctx, k); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
		} else {
			if err := a.mpd.Unmount(ctx, k); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			now := time.Now().UTC()
			r = setUpdateTime(r, now)
			if _, err := a.mpd.Update(ctx, ""); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
		}
	}
	r.Method = http.MethodGet
	a.cache.ServeHTTP(w, r)
}

// Changed returns storage list update event chan.
func (a *StorageHandler) Changed() <-chan struct{} {
	return a.cache.Changed()
}

// Close closes update event chan.
func (a *StorageHandler) Close() {
	a.cache.Close()
}
