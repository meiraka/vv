package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func Gzip(t *testing.T, b []byte) []byte {
	gz, err := makeGZip(b)
	if err != nil {
		t.Fatalf("failed to make gzip")
	}
	return gz
}

func TestLocalAssetsHandler(t *testing.T) {
	assets := []string{"/", "/assets/app-black.png", "/assets/app-black.svg", "/assets/app.css", "/assets/app.js", "/assets/app.png", "/assets/app.svg", "/assets/manifest.json", "/assets/nocover.svg", "/assets/w.png"}
	for _, local := range []bool{true, false} {
		h := AssetsConfig{LocalAssets: local}.NewAssetsHandler()
		for _, path := range assets {
			t.Run(fmt.Sprintf("local=%t, %s", local, path), func(t *testing.T) {
				req := httptest.NewRequest(http.MethodGet, path, nil)
				w := httptest.NewRecorder()
				h(w, req)
				resp := w.Result()
				if resp.StatusCode != 200 {
					t.Errorf("got %d; want %d", resp.StatusCode, 200)
				}

			})
		}

	}

}

func TestAssetsHandler(t *testing.T) {
	conf := AssetsConfig{LocalAssets: false}
	ts := httptest.NewServer(conf.NewAssetsHandler())
	defer ts.Close()
	testsets := map[string]struct {
		path   string
		header http.Header
		status int
		body   []byte
	}{
		"text no-gzip":  {path: "/assets/app.svg", header: http.Header{"Accept-Encoding": {"identity"}}, status: http.StatusOK, body: AssetsAppSVG},
		"text gzip":     {path: "/assets/app.svg", header: http.Header{"Accept-Encoding": {"gzip"}}, status: http.StatusOK, body: Gzip(t, AssetsAppSVG)},
		"if none match": {path: "/assets/app.svg", header: http.Header{"If-None-Match": {fmt.Sprintf(`"%s"`, AssetsAppSVGHash)}}, status: http.StatusNotModified, body: []byte("")},

		"binary no-gzip": {path: "/assets/app.png", header: http.Header{"Accept-Encoding": {"identity"}}, status: http.StatusOK, body: AssetsAppPNG},
		"binary gzip":    {path: "/assets/app.png", header: http.Header{"Accept-Encoding": {"gzip"}}, status: http.StatusOK, body: AssetsAppPNG},
	}
	for k, tt := range testsets {
		t.Run(k, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, ts.URL+tt.path, nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			for k, v := range tt.header {
				req.Header[k] = append(req.Header[k], v...)
			}
			resp, err := testHTTPClient.Do(req)
			if err != nil {
				t.Fatalf("failed to request: %v", err)
			}
			defer resp.Body.Close()
			got, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read response: %v", err)
			}
			if !bytes.Equal(got, tt.body) || resp.StatusCode != tt.status {
				t.Errorf("got %d %s; want %d %s", resp.StatusCode, got, tt.status, tt.body)
			}
		})
	}
}

func TestI18NAssetsHandler(t *testing.T) {
	conf := AssetsConfig{LocalAssets: false}
	b := []byte(`{{ or .lang "en" }}`)
	ts := httptest.NewServer(conf.i18nAssetsHandler("assets/app.html", b, AssetsAppHTMLHash))
	defer ts.Close()
	testsets := map[string]struct {
		header http.Header
		status int
		body   []byte
	}{
		"no-gzip":             {header: http.Header{"Accept-Encoding": {"identity"}}, status: http.StatusOK, body: []byte("en")},
		"gzip":                {header: http.Header{"Accept-Encoding": {"gzip"}}, status: http.StatusOK, body: Gzip(t, []byte("en"))},
		"if none match":       {header: http.Header{"If-None-Match": {fmt.Sprintf(`"%s"`, AssetsAppHTMLHash)}}, status: http.StatusNotModified, body: []byte("")},
		"ja:no-gzip":          {header: http.Header{"Accept-Language": {"ja"}, "Accept-Encoding": {"identity"}}, status: http.StatusOK, body: []byte("ja")},
		"ja:gzip":             {header: http.Header{"Accept-Language": {"ja"}, "Accept-Encoding": {"gzip"}}, status: http.StatusOK, body: Gzip(t, []byte("ja"))},
		"ja:if none match":    {header: http.Header{"Accept-Language": {"ja"}, "If-None-Match": {fmt.Sprintf(`"%s"`, AssetsAppHTMLHash)}}, status: http.StatusNotModified, body: []byte("")},
		"ja-JP:no-gzip":       {header: http.Header{"Accept-Language": {"ja-JP"}, "Accept-Encoding": {"identity"}}, status: http.StatusOK, body: []byte("ja")},
		"ja-JP:gzip":          {header: http.Header{"Accept-Language": {"ja-JP"}, "Accept-Encoding": {"gzip"}}, status: http.StatusOK, body: Gzip(t, []byte("ja"))},
		"ja-JP:if none match": {header: http.Header{"Accept-Language": {"ja-JP"}, "If-None-Match": {fmt.Sprintf(`"%s"`, AssetsAppHTMLHash)}}, status: http.StatusNotModified, body: []byte("")},
		"en-US:no-gzip":       {header: http.Header{"Accept-Language": {"en-US"}, "Accept-Encoding": {"identity"}}, status: http.StatusOK, body: []byte("en")},
		"en-US:gzip":          {header: http.Header{"Accept-Language": {"en-US"}, "Accept-Encoding": {"gzip"}}, status: http.StatusOK, body: Gzip(t, []byte("en"))},
		"en-US:if none match": {header: http.Header{"Accept-Language": {"en-US"}, "If-None-Match": {fmt.Sprintf(`"%s"`, AssetsAppHTMLHash)}}, status: http.StatusNotModified, body: []byte("")},
		"en-GB:no-gzip":       {header: http.Header{"Accept-Language": {"en-US"}, "Accept-Encoding": {"identity"}}, status: http.StatusOK, body: []byte("en")},
		"en-GB:gzip":          {header: http.Header{"Accept-Language": {"en-US"}, "Accept-Encoding": {"gzip"}}, status: http.StatusOK, body: Gzip(t, []byte("en"))},
		"en-GB:if none match": {header: http.Header{"Accept-Language": {"en-US"}, "If-None-Match": {fmt.Sprintf(`"%s"`, AssetsAppHTMLHash)}}, status: http.StatusNotModified, body: []byte("")},
	}
	for k, tt := range testsets {
		t.Run(k, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, ts.URL+"/", nil)
			if err != nil {
				t.Fatalf("failed to create request: %v", err)
			}
			for k, v := range tt.header {
				req.Header[k] = append(req.Header[k], v...)
			}
			resp, err := testHTTPClient.Do(req)
			if err != nil {
				t.Fatalf("failed to request: %v", err)
			}
			defer resp.Body.Close()
			got, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read response: %v", err)
			}
			if !bytes.Equal(got, tt.body) || resp.StatusCode != tt.status {
				t.Errorf("got %d %s; want %d %s", resp.StatusCode, got, tt.status, tt.body)
			}
		})
	}
}
