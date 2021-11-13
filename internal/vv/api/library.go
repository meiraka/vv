package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

type httpLibraryInfo struct {
	Updating bool `json:"updating"`
}

type MPDLibrary interface {
	Update(context.Context, string) (map[string]string, error)
}

type LibraryHandler struct {
	mpd   MPDLibrary
	cache *cache
}

func NewLibraryHandler(mpd MPDLibrary) (*LibraryHandler, error) {
	c, err := newCache(&httpLibraryInfo{})
	if err != nil {
		return nil, err
	}
	return &LibraryHandler{
		mpd:   mpd,
		cache: c,
	}, nil
}

func (a *LibraryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		a.cache.ServeHTTP(w, r)
		return
	}
	var req httpLibraryInfo
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeHTTPError(w, http.StatusBadRequest, err)
		return
	}
	if !req.Updating {
		writeHTTPError(w, http.StatusBadRequest, errors.New("requires updating=true"))
		return
	}
	ctx := r.Context()
	now := time.Now().UTC()
	if _, err := a.mpd.Update(ctx, ""); err != nil {
		writeHTTPError(w, http.StatusInternalServerError, err)
		return
	}
	r.Method = http.MethodGet
	a.cache.ServeHTTP(w, setUpdateTime(r, now))
}

func (a *LibraryHandler) UpdateStatus(updating bool) error {
	_, err := a.cache.SetIfModified(&httpLibraryInfo{Updating: updating})
	return err
}

// Changed returns library song list update event chan.
func (a *LibraryHandler) Changed() <-chan struct{} {
	return a.cache.Changed()
}

// Close closes update event chan.
func (a *LibraryHandler) Close() {
	a.cache.Close()
}
