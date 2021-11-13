package assets

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/meiraka/vv/internal/gzip"
)

const (
	testTimeout = time.Second
)

var (
	testHTTPClient = &http.Client{Timeout: testTimeout}
)

func Gzip(t *testing.T, b []byte) []byte {
	gz, err := gzip.Encode(b)
	if err != nil {
		t.Fatalf("failed to make gzip")
	}
	return gz
}

func TestLocalHandler(t *testing.T) {
	assets := []string{"/assets/app-black.png", "/assets/app-black.svg", "/assets/app.css", "/assets/app.js", "/assets/app.png", "/assets/app.svg", "/assets/manifest.json", "/assets/nocover.svg", "/assets/w.png"}
	for _, conf := range []*Config{nil, {}, {Local: true, LocalDir: filepath.Join("..", "..", "..", "assets")}} {
		t.Run(fmt.Sprintf("%+v", conf), func(t *testing.T) {
			h, err := NewHandler(conf)
			if err != nil {
				t.Fatalf("failed to init hander: %v", err)
			}
			for _, path := range assets {
				t.Run(fmt.Sprint(path), func(t *testing.T) {
					req := httptest.NewRequest(http.MethodGet, path, nil)
					w := httptest.NewRecorder()
					h(w, req)
					resp := w.Result()
					if resp.StatusCode != 200 {
						t.Errorf("got %d; want %d", resp.StatusCode, 200)
					}

				})
			}
		})
	}

}

func TestHandler(t *testing.T) {
	h, err := NewHandler(nil)
	if err != nil {
		t.Fatalf("failed to init hander: %v", err)
	}
	ts := httptest.NewServer(h)
	defer ts.Close()
	testsets := map[string]struct {
		path       string
		header     http.Header
		status     int
		wantHeader http.Header
		wantBody   []byte
	}{
		"text no-gzip": {path: "/assets/app.svg", header: http.Header{"Accept-Encoding": {"identity"}}, status: http.StatusOK,
			wantHeader: http.Header{
				"Cache-Control":     {"max-age=86400"},
				"Etag":              {`"` + string(AppSVGHash) + `"`},
				"Vary":              {"Accept-Encoding"},
				"Transfer-Encoding": nil,
			},
			wantBody: AppSVG},
		"text gzip": {path: "/assets/app.svg", header: http.Header{"Accept-Encoding": {"gzip"}}, status: http.StatusOK,
			wantHeader: http.Header{
				"Cache-Control":     {"max-age=86400"},
				"Etag":              {`"` + string(AppSVGHash) + `"`},
				"Vary":              {"Accept-Encoding"},
				"Transfer-Encoding": nil,
			},
			wantBody: Gzip(t, AppSVG)},
		"text gzip with param d": {path: "/assets/app.svg?h=0", header: http.Header{"Accept-Encoding": {"gzip"}}, status: http.StatusOK,
			wantHeader: http.Header{
				"Cache-Control":     {"max-age=31536000"},
				"Etag":              {`"` + string(AppSVGHash) + `"`},
				"Vary":              {"Accept-Encoding"},
				"Transfer-Encoding": nil,
			},
			wantBody: Gzip(t, AppSVG)},
		"if none match": {path: "/assets/app.svg", header: http.Header{"If-None-Match": {fmt.Sprintf(`"%s"`, AppSVGHash)}}, status: http.StatusNotModified,
			wantHeader: http.Header{
				"Transfer-Encoding": nil,
			},
			wantBody: []byte("")},

		"binary no-gzip": {path: "/assets/app.png", header: http.Header{"Accept-Encoding": {"identity"}}, status: http.StatusOK,
			wantHeader: http.Header{
				"Cache-Control":     {"max-age=86400"},
				"Etag":              {`"` + string(AppPNGHash) + `"`},
				"Transfer-Encoding": nil,
			},
			wantBody: AppPNG},
		"binary gzip": {path: "/assets/app.png", header: http.Header{"Accept-Encoding": {"gzip"}}, status: http.StatusOK,
			wantHeader: http.Header{
				"Cache-Control":     {"max-age=86400"},
				"Etag":              {`"` + string(AppPNGHash) + `"`},
				"Transfer-Encoding": nil,
			},
			wantBody: AppPNG},
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
			for k, v := range tt.wantHeader {
				if got := resp.Header[k]; !reflect.DeepEqual(got, v) {
					t.Errorf("got header %s %v; want %v", k, got, v)
				}

			}
			if !bytes.Equal(got, tt.wantBody) || resp.StatusCode != tt.status {
				t.Errorf("got %d %s; want %d %s", resp.StatusCode, got, tt.status, tt.wantBody)
			}
		})
	}
}
