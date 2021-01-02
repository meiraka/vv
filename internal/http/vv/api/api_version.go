package api

import (
	"fmt"
	"net/http"
	"runtime"
)

const (
	pathAPIVersion = "/api/version"
)

type httpAPIVersion struct {
	App string `json:"app"`
	Go  string `json:"go"`
	MPD string `json:"mpd"`
}

func (a *api) VersionHandler() http.Handler {
	return a.jsonCache.Handler(pathAPIVersion)

}

func (a *api) updateVersion() error {
	goVersion := fmt.Sprintf("%s %s %s", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	mpdVersion := a.client.Version()
	if len(mpdVersion) == 0 {
		mpdVersion = "unknown"
	}
	return a.jsonCache.SetIfModified(pathAPIVersion, &httpAPIVersion{App: a.config.AppVersion, Go: goVersion, MPD: mpdVersion})
}

func (a *api) updateVersionNoMPD() error {
	goVersion := fmt.Sprintf("%s %s %s", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	return a.jsonCache.SetIfModified(pathAPIVersion, &httpAPIVersion{App: a.config.AppVersion, Go: goVersion, MPD: ""})
}
