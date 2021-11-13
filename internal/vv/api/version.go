package api

import (
	"fmt"
	"net/http"
	"runtime"
)

var goVersion = fmt.Sprintf("%s %s %s", runtime.Version(), runtime.GOOS, runtime.GOARCH)

type httpVersion struct {
	App string `json:"app"`
	Go  string `json:"go"`
	MPD string `json:"mpd"`
}

type VersionHandler struct {
	mpd     MPDVersion
	cache   *cache
	version string
}

// MPDVersion represents mpd api for Version API.
type MPDVersion interface {
	Version() string
}

func NewVersionHandler(mpd MPDVersion, version string) (*VersionHandler, error) {
	c, err := newCache(map[string]*httpVersion{})
	if err != nil {
		return nil, err
	}
	return &VersionHandler{
		mpd:     mpd,
		cache:   c,
		version: version,
	}, nil
}

func (a *VersionHandler) Update() error {
	mpdVersion := a.mpd.Version()
	if len(mpdVersion) == 0 {
		mpdVersion = "unknown"
	}
	_, err := a.cache.SetIfModified(&httpVersion{App: a.version, Go: goVersion, MPD: mpdVersion})
	return err
}

func (a *VersionHandler) UpdateNoMPD() error {
	_, err := a.cache.SetIfModified(&httpVersion{App: a.version, Go: goVersion})
	return err
}

// ServeHTTP responses version as json format.
func (a *VersionHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.cache.ServeHTTP(w, r)
}

// Changed returns version update event chan.
func (a *VersionHandler) Changed() <-chan struct{} {
	return a.cache.Changed()
}

// Close closes update event chan.
func (a *VersionHandler) Close() {
	a.cache.Close()
}
