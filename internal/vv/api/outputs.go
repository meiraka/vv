package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/meiraka/vv/internal/mpd"
)

type httpOutput struct {
	Name       string               `json:"name"`
	Plugin     string               `json:"plugin,omitempty"`
	Enabled    *bool                `json:"enabled"`
	Attributes *httpOutputAttrbutes `json:"attributes,omitempty"`
	Stream     string               `json:"stream,omitempty"`
}

type httpOutputAttrbutes struct {
	DoP            *bool     `json:"dop,omitempty"`
	AllowedFormats *[]string `json:"allowed_formats,omitempty"`
}

type MPDOutputs interface {
	EnableOutput(context.Context, string) error
	DisableOutput(context.Context, string) error
	OutputSet(context.Context, string, string, string) error
	Outputs(context.Context) ([]*mpd.Output, error)
}

type OutputsHandler struct {
	mpd   MPDOutputs
	cache *cache
	proxy map[string]string
}

func NewOutputsHandler(mpd MPDOutputs, proxy map[string]string) (*OutputsHandler, error) {
	c, err := newCache(map[string]*httpOutput{})
	if err != nil {
		return nil, err
	}
	return &OutputsHandler{
		mpd:   mpd,
		cache: c,
		proxy: proxy,
	}, nil
}

func (a *OutputsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		a.cache.ServeHTTP(w, r)
		return
	}
	var req map[string]*httpOutput
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeHTTPError(w, http.StatusBadRequest, err)
		return
	}
	ctx := r.Context()
	now := time.Now().UTC()
	changed := false
	for k, v := range req {
		if v.Enabled != nil {
			var err error
			changed = true
			if *v.Enabled {
				err = a.mpd.EnableOutput(ctx, k)
			} else {
				err = a.mpd.DisableOutput(ctx, k)
			}
			if err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
		}
		if v.Attributes != nil {
			if v.Attributes.DoP != nil {
				changed = true
				if err := a.mpd.OutputSet(ctx, k, "dop", btoa(*v.Attributes.DoP, "1", "0")); err != nil {
					writeHTTPError(w, http.StatusInternalServerError, err)
					return
				}
			}
			if v.Attributes.AllowedFormats != nil {
				allowedFormats := *v.Attributes.AllowedFormats
				for i := range allowedFormats {
					if strings.Contains(allowedFormats[i], " ") {
						writeHTTPError(w, http.StatusBadRequest, fmt.Errorf("api: invalid allowed formats: #%d: %q", i, allowedFormats[i]))
						return
					}
				}
				changed = true
				if err := a.mpd.OutputSet(ctx, k, "allowed_formats", strings.Join(allowedFormats, " ")); err != nil {
					writeHTTPError(w, http.StatusInternalServerError, err)
					return
				}
			}
		}
	}
	if changed {
		r = setUpdateTime(r, now)
	}
	r.Method = http.MethodGet
	a.cache.ServeHTTP(w, r)

}

func (a *OutputsHandler) Update(ctx context.Context) error {
	l, err := a.mpd.Outputs(ctx)
	if err != nil {
		return err
	}
	data := make(map[string]*httpOutput, len(l))
	for _, v := range l {
		var stream string
		if _, ok := a.proxy[v.Name]; ok {
			stream = pathAPIMusicOutputsStream + "?" + url.Values{"name": {v.Name}}.Encode()
		}
		output := &httpOutput{
			Name:    v.Name,
			Plugin:  v.Plugin,
			Enabled: &v.Enabled,
			Stream:  stream,
		}
		if v.Attributes != nil {
			output.Attributes = &httpOutputAttrbutes{}
			if dop, ok := v.Attributes["dop"]; ok {
				output.Attributes.DoP = boolPtr(dop == "1")
			}
			if allowedFormats, ok := v.Attributes["allowed_formats"]; ok {
				if len(allowedFormats) == 0 {
					output.Attributes.AllowedFormats = stringSlicePtr([]string{})
				} else {
					output.Attributes.AllowedFormats = stringSlicePtr(strings.Split(allowedFormats, " "))
				}
			}
		}
		data[v.ID] = output
	}
	_, err = a.cache.SetIfModified(data)
	return err
}

// Changed returns outputs update event chan.
func (a *OutputsHandler) Changed() <-chan struct{} {
	return a.cache.Changed()
}

// Close closes update event chan.
func (a *OutputsHandler) Close() {
	a.cache.Close()
}
