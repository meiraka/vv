package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/meiraka/vv/internal/mpd"
)

const (
	pathAPIMusicStorage = "/api/music/storage"
)

type httpStorage struct {
	URI      *string `json:"uri,omitempty"`
	Updating bool    `json:"updating,omitempty"`
}

func (a *api) StorageHandler() http.HandlerFunc {
	get := a.jsonCache.Handler(pathAPIMusicStorage)
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			get.ServeHTTP(w, r)
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
			if v.Updating {
				if _, err := a.client.Update(ctx, k); err != nil {
					writeHTTPError(w, http.StatusInternalServerError, err)
					return
				}
			} else if v.URI != nil {
				if err := a.client.Mount(ctx, k, *v.URI); err != nil {
					writeHTTPError(w, http.StatusInternalServerError, err)
					return
				}
				if _, err := a.client.Update(ctx, k); err != nil {
					writeHTTPError(w, http.StatusInternalServerError, err)
					return
				}
			} else {
				if err := a.client.Unmount(ctx, k); err != nil {
					writeHTTPError(w, http.StatusInternalServerError, err)
					return
				}
				if _, err := a.client.Update(ctx, ""); err != nil {
					writeHTTPError(w, http.StatusInternalServerError, err)
					return
				}
			}
		}
		if len(req) != 0 {
			now := time.Now().UTC()
			r = setUpdateTime(r, now)
		}
		r.Method = http.MethodGet
		get.ServeHTTP(w, r)
	}
}

func (a *api) updateStorage(ctx context.Context) error {
	ret := map[string]*httpStorage{}
	ms, err := a.client.ListMounts(ctx)
	if err != nil {
		// skip command error to support old mpd
		var perr *mpd.CommandError
		if errors.As(err, &perr) {
			a.jsonCache.SetIfModified(pathAPIMusicStorage, ret)
			return nil
		}
		return err
	}
	for _, m := range ms {
		ret[m["mount"]] = &httpStorage{
			URI: stringPtr(m["storage"]),
		}
	}
	a.jsonCache.SetIfModified(pathAPIMusicStorage, ret)
	return nil
}
