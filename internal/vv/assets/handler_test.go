package assets

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/meiraka/vv/internal/gzip"
)

func TestLocalHandler(t *testing.T) {
	assets := []string{"/assets/app-black.png", "/assets/app-black.svg", "/assets/app.css", "/assets/app.js", "/assets/app.png", "/assets/app.svg", "/assets/manifest.json", "/assets/nocover.svg", "/assets/w.png"}
	for _, conf := range []*Config{nil, {}, {Local: true, LocalDir: "."}} {
		t.Run(fmt.Sprintf("%+v", conf), func(t *testing.T) {
			h, err := NewHandler(conf)
			if err != nil {
				t.Fatalf("failed to init hander: %v", err)
			}
			for _, path := range assets {
				t.Run(fmt.Sprint(path), func(t *testing.T) {
					req := httptest.NewRequest(http.MethodGet, path, nil)
					w := httptest.NewRecorder()
					h.ServeHTTP(w, req)
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
	now := time.Now()
	testsets := map[string]struct {
		prehook   func(tb testing.TB)
		handler   http.Handler
		reqPath   string
		reqHeader http.Header

		resStatus int
		resHeader http.Header
		resBody   []byte
	}{
		"embed/text/no-gzip": {
			handler:   must(NewHandler(&Config{LastModified: now})),
			reqPath:   "/assets/app.svg",
			reqHeader: http.Header{"Accept-Encoding": {"identity"}}, resStatus: http.StatusOK,
			resHeader: http.Header{
				"Cache-Control":     {"max-age=86400"},
				"Content-Type":      {"image/svg+xml"},
				"Etag":              {`"` + calcMD5File(t, "app.svg") + `"`},
				"Last-Modified":     {now.UTC().Format(http.TimeFormat)},
				"Transfer-Encoding": nil,
				"Vary":              {"Accept-Encoding"},
			},
			resBody: readFile(t, "app.svg")},
		"embed/text/gzip": {
			handler:   must(NewHandler(&Config{LastModified: now})),
			reqPath:   "/assets/app.svg",
			reqHeader: http.Header{"Accept-Encoding": {"gzip"}}, resStatus: http.StatusOK,
			resHeader: http.Header{
				"Cache-Control":     {"max-age=86400"},
				"Content-Type":      {"image/svg+xml"},
				"Etag":              {`"` + calcGZMD5File(t, "app.svg") + `"`},
				"Last-Modified":     {now.UTC().Format(http.TimeFormat)},
				"Transfer-Encoding": nil,
				"Vary":              {"Accept-Encoding"},
			},
			resBody: gz(t, readFile(t, "app.svg"))},
		"embed/text/gzip with param h": {
			handler:   must(NewHandler(&Config{LastModified: now})),
			reqPath:   "/assets/app.svg?h=0",
			reqHeader: http.Header{"Accept-Encoding": {"gzip"}}, resStatus: http.StatusOK,
			resHeader: http.Header{
				"Cache-Control":     {"max-age=31536000"},
				"Content-Type":      {"image/svg+xml"},
				"Etag":              {`"` + calcGZMD5File(t, "app.svg") + `"`},
				"Last-Modified":     {now.UTC().Format(http.TimeFormat)},
				"Transfer-Encoding": nil,
				"Vary":              {"Accept-Encoding"},
			},
			resBody: gz(t, readFile(t, "app.svg"))},
		"embed/if none match": {
			handler:   must(NewHandler(&Config{LastModified: now})),
			reqPath:   "/assets/app.svg",
			reqHeader: http.Header{"If-None-Match": {`"` + calcMD5File(t, "app.svg") + `"`}}, resStatus: http.StatusNotModified,
			resHeader: http.Header{
				"Cache-Control":     nil,
				"Content-Type":      nil,
				"Etag":              nil,
				"Last-Modified":     nil,
				"Transfer-Encoding": nil,
				"Vary":              nil,
			},
			resBody: []byte("")},
		"embed/gzip if none match": {
			handler:   must(NewHandler(&Config{LastModified: now})),
			reqPath:   "/assets/app.svg",
			reqHeader: http.Header{"Accept-Encoding": {"gzip"}, "If-None-Match": {`"` + calcGZMD5File(t, "app.svg") + `"`}}, resStatus: http.StatusNotModified,
			resHeader: http.Header{
				"Cache-Control":     nil,
				"Content-Type":      nil,
				"Etag":              nil,
				"Last-Modified":     nil,
				"Transfer-Encoding": nil,
				"Vary":              nil,
			},
			resBody: []byte("")},
		"embed/binary no-gzip": {
			handler:   must(NewHandler(&Config{LastModified: now})),
			reqPath:   "/assets/app.png",
			reqHeader: http.Header{"Accept-Encoding": {"identity"}}, resStatus: http.StatusOK,
			resHeader: http.Header{
				"Cache-Control":     {"max-age=86400"},
				"Content-Type":      {"image/png"},
				"Etag":              {`"` + calcMD5File(t, "app.png") + `"`},
				"Last-Modified":     {now.UTC().Format(http.TimeFormat)},
				"Transfer-Encoding": nil,
				"Vary":              nil,
			},
			resBody: readFile(t, "app.png")},
		"embed/binary gzip": {
			handler:   must(NewHandler(&Config{LastModified: now})),
			reqPath:   "/assets/app.png",
			reqHeader: http.Header{"Accept-Encoding": {"gzip"}}, resStatus: http.StatusOK,
			resHeader: http.Header{
				"Cache-Control":     {"max-age=86400"},
				"Content-Type":      {"image/png"},
				"Etag":              {`"` + calcMD5File(t, "app.png") + `"`},
				"Last-Modified":     {now.UTC().Format(http.TimeFormat)},
				"Transfer-Encoding": nil,
				"Vary":              nil,
			},
			resBody: readFile(t, "app.png")},
	}
	for k, tt := range testsets {
		t.Run(k, func(t *testing.T) {
			if tt.prehook != nil {
				tt.prehook(t)
			}
			r := httptest.NewRequest("GET", "http://vv.local"+tt.reqPath, nil)
			for k, v := range tt.reqHeader {
				for i := range v {
					r.Header.Add(k, v[i])
				}
			}
			t.Log(r)
			w := httptest.NewRecorder()
			tt.handler.ServeHTTP(w, r)
			resp := w.Result()
			if got, want := w.Body.Bytes(), tt.resBody; !bytes.Equal(got, want) {
				t.Errorf("got body\n%s; want\n%s", got, want)
			}
			for k, v := range tt.resHeader {
				if !reflect.DeepEqual(resp.Header[k], v) {
					t.Errorf("got header %s=%v; want %v", k, resp.Header[k], v)
				}
			}
		})
	}
}

func must(t http.Handler, err error) http.Handler {
	if err != nil {
		panic(err)
	}
	return t
}

func gz(t *testing.T, b []byte) []byte {
	gz, err := gzip.Encode(b)
	if err != nil {
		t.Fatalf("failed to make gzip")
	}
	return gz
}

func calcGZMD5File(tb testing.TB, path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		tb.Fatalf("embed: readfile: %s: %v", path, err)
	}
	gz, err := gzip.Encode(b)
	if err != nil {
		tb.Fatalf("gzip: %s: %v", path, err)
	}
	hasher := md5.New()
	hasher.Write(gz)
	return hex.EncodeToString(hasher.Sum(nil))
}

func calcMD5File(tb testing.TB, path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		tb.Fatalf("embed: readfile: %s: %v", path, err)
	}
	hasher := md5.New()
	hasher.Write(b)
	return hex.EncodeToString(hasher.Sum(nil))
}

func readFile(tb testing.TB, path string) []byte {
	b, err := os.ReadFile(path)
	if err != nil {
		tb.Fatalf("embed: readfile: %s: %v", path, err)
	}
	return b
}
