package api

import (
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	pathAPIMusicOutputs       = "/api/music/outputs"
	pathAPIMusicOutputsStream = "/api/music/outputs/stream"
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

func (a *api) OutputsHandler() http.HandlerFunc {
	get := a.jsonCache.Handler(pathAPIMusicOutputs)
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			get.ServeHTTP(w, r)
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
					err = a.client.EnableOutput(ctx, k)
				} else {
					err = a.client.DisableOutput(ctx, k)
				}
				if err != nil {
					writeHTTPError(w, http.StatusInternalServerError, err)
					return
				}
			}
			if v.Attributes != nil {
				if v.Attributes.DoP != nil {
					changed = true
					if err := a.client.OutputSet(ctx, k, "dop", btoa(*v.Attributes.DoP, "1", "0")); err != nil {
						writeHTTPError(w, http.StatusInternalServerError, err)
						return
					}
				}
				if v.Attributes.AllowedFormats != nil {
					changed = true
					if err := a.client.OutputSet(ctx, k, "allowed_formats", strings.Join(*v.Attributes.AllowedFormats, " ")); err != nil {
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
		get.ServeHTTP(w, r)

	}
}

func (a *api) OutputsStreamHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dev := r.URL.Query().Get("name")
		url, ok := a.config.AudioProxy[dev]
		if !ok {
			http.NotFound(w, r)
			return
		}
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		pr, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			log.Println(url, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		resp, err := http.DefaultClient.Do(pr)
		if err != nil {
			log.Println(url, err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()
		for k, v := range resp.Header {
			for i := range v {
				w.Header().Add(k, v[i])
			}
		}
		go func() {
			select {
			case <-ctx.Done():
			case <-a.stopCh:
				// disconnect audio stream by stop()
				log.Println("disconnecting audio stream")
				cancel()
			}
		}()
		io.Copy(w, resp.Body)
	}
}

func (a *api) updateOutputs(ctx context.Context) error {
	l, err := a.client.Outputs(ctx)
	if err != nil {
		return err
	}
	data := make(map[string]*httpOutput, len(l))
	for _, v := range l {
		var stream string
		if _, ok := a.config.AudioProxy[v.Name]; ok {
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
	return a.jsonCache.SetIfModified(pathAPIMusicOutputs, data)
}

func btoa(b bool, t, f string) string {
	if b {
		return t
	}
	return f
}
