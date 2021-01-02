package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

const (
	pathAPIMusicImages = "/api/music/images"
)

type httpImages struct {
	Updating bool `json:"updating"`
}

func (a *api) ImagesHandler() http.HandlerFunc {
	get := a.jsonCache.Handler(pathAPIMusicImages)
	a.jsonCache.SetIfNone(pathAPIMusicImages, &httpImages{})
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			get.ServeHTTP(w, r)
			return
		}
		var req httpImages
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeHTTPError(w, http.StatusBadRequest, err)
			return
		}
		if !req.Updating {
			writeHTTPError(w, http.StatusBadRequest, errors.New("requires updating=true"))
			return
		}
		a.covers.Update(a.library)
		now := time.Now().UTC()
		r.Method = http.MethodGet
		get.ServeHTTP(w, setUpdateTime(r, now))
	}
}
