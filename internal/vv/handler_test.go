package vv_test

import (
	"crypto/md5"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/meiraka/vv/internal/gzip"
	"github.com/meiraka/vv/internal/vv"
)

func TestHandler(t *testing.T) {
	src := "index.html"
	now := time.Now()
	for label, tt := range map[string]struct {
		handler   http.Handler
		prehook   func(tb testing.TB)
		reqHeader http.Header
		resHeader http.Header
		resBody   string
	}{
		"embed": {
			handler: must(vv.New(&vv.Config{LastModified: now})),
			resHeader: http.Header{
				"Cache-Control":    {"max-age=86400"},
				"Content-Encoding": nil,
				"Content-Language": {"en-US"},
				"Content-Type":     {"text/html; charset=utf-8"},
				"Last-Modified":    {now.UTC().Format(http.TimeFormat)},
				"Vary":             {"Accept-Encoding, Accept-Language"},
			},
		},
		"embed/gzip": {
			handler:   must(vv.New(&vv.Config{LastModified: now})),
			reqHeader: http.Header{"Accept-Encoding": {"gzip"}},
			resHeader: http.Header{
				"Cache-Control":    {"max-age=86400"},
				"Content-Encoding": {"gzip"},
				"Content-Language": {"en-US"},
				"Content-Type":     {"text/html; charset=utf-8"},
				"Last-Modified":    {now.UTC().Format(http.TimeFormat)},
				"Vary":             {"Accept-Encoding, Accept-Language"},
			},
		},
		"embed/ja": {
			handler:   must(vv.New(&vv.Config{LastModified: now})),
			reqHeader: http.Header{"Accept-Language": {"ja;q=0.9,en-US;q=0.7"}},
			resHeader: http.Header{
				"Cache-Control":    {"max-age=86400"},
				"Content-Encoding": nil,
				"Content-Language": {"ja"},
				"Content-Type":     {"text/html; charset=utf-8"},
				"Last-Modified":    {now.UTC().Format(http.TimeFormat)},
				"Vary":             {"Accept-Encoding, Accept-Language"},
			},
		},
		"local/older than exec binary": {
			prehook: func(tb testing.TB) {
				if err := os.Chtimes(src, now, now.Add(-1*time.Hour)); err != nil {
					t.Fatalf("failed to update src mod time: %v", err)
				}
			},
			handler: must(vv.New(&vv.Config{Local: true, LocalDir: ".", LastModified: now})),
			resHeader: http.Header{
				"Cache-Control":    {"max-age=1"},
				"Content-Encoding": nil,
				"Content-Language": {"en-US"},
				"Content-Type":     {"text/html; charset=utf-8"},
				"Etag":             nil,
				"Last-Modified":    {now.UTC().Format(http.TimeFormat)},
				"Vary":             {"Accept-Language"},
			},
		},
		"local/newer than exec binary": {
			prehook: func(tb testing.TB) {
				if err := os.Chtimes(src, now, now.Add(time.Hour)); err != nil {
					t.Fatalf("failed to update src mod time: %v", err)
				}
			},
			handler: must(vv.New(&vv.Config{Local: true, LocalDir: ".", LastModified: now})),
			resHeader: http.Header{
				"Cache-Control":    {"max-age=1"},
				"Content-Language": {"en-US"},
				"Content-Type":     {"text/html; charset=utf-8"},
				"Etag":             nil,
				"Last-Modified":    {now.UTC().Add(time.Hour).Format(http.TimeFormat)}, // use file mod time
				"Vary":             {"Accept-Language"},
			},
		},
	} {
		t.Run(label, func(t *testing.T) {
			if tt.prehook != nil {
				tt.prehook(t)
			}
			r := httptest.NewRequest("GET", "http://vv.local/", nil)
			for k, v := range tt.reqHeader {
				for i := range v {
					r.Header.Add(k, v[i])
				}
			}
			w := httptest.NewRecorder()
			tt.handler.ServeHTTP(w, r)
			resp := w.Result()
			if got := w.Body.String(); got == "" {
				t.Errorf("got body\n%s; want non empty string", got)
			}
			for k, v := range tt.resHeader {
				if !reflect.DeepEqual(resp.Header[k], v) {
					t.Errorf("got header %s=%v; want %v", k, resp.Header[k], v)
				}
			}
		})
	}
}

var testfile = filepath.Join("testdata", "index.html")

func TestHandlerTestData(t *testing.T) {
	testdata, err := os.ReadFile(testfile)
	if err != nil {
		t.Fatalf("failed to open %s: %v", testfile, err)
	}
	now := time.Now()
	en := `<!DOCTYPE html><html lang="en"><title></title></html>` + "\n"
	ja := `<!DOCTYPE html><html lang="ja"><title></title></html>` + "\n"
	for label, tt := range map[string]struct {
		handler   http.Handler
		prehook   func(tb testing.TB)
		reqHeader http.Header
		resHeader http.Header
		resBody   string
	}{
		"embed": {
			handler: must(vv.New(&vv.Config{LastModified: now, Data: testdata})),
			resHeader: http.Header{
				"Cache-Control":    {"max-age=86400"},
				"Content-Encoding": nil,
				"Content-Language": {"en-US"},
				"Content-Length":   {strconv.Itoa(len(en))},
				"Content-Type":     {"text/html; charset=utf-8"},
				"Etag":             {`"` + md5str(en) + `"`},
				"Last-Modified":    {now.UTC().Format(http.TimeFormat)},
				"Vary":             {"Accept-Encoding, Accept-Language"},
			},
			resBody: en,
		},
		"embed/gzip": {
			handler:   must(vv.New(&vv.Config{LastModified: now, Data: testdata})),
			reqHeader: http.Header{"Accept-Encoding": {"gzip"}},
			resHeader: http.Header{
				"Cache-Control":    {"max-age=86400"},
				"Content-Encoding": {"gzip"},
				"Content-Language": {"en-US"},
				"Content-Length":   {strconv.Itoa(len(gzstr(en)))},
				"Content-Type":     {"text/html; charset=utf-8"},
				"Etag":             {`"` + md5str(gzstr(en)) + `"`},
				"Last-Modified":    {now.UTC().Format(http.TimeFormat)},
				"Vary":             {"Accept-Encoding, Accept-Language"},
			},
			resBody: gzstr(en),
		},
		"embed/ja": {
			handler:   must(vv.New(&vv.Config{LastModified: now, Data: testdata})),
			reqHeader: http.Header{"Accept-Language": {"ja;q=0.9,en-US;q=0.7"}},
			resHeader: http.Header{
				"Cache-Control":    {"max-age=86400"},
				"Content-Encoding": nil,
				"Content-Language": {"ja"},
				"Content-Length":   {strconv.Itoa(len(ja))},
				"Content-Type":     {"text/html; charset=utf-8"},
				"Etag":             {`"` + md5str(ja) + `"`},
				"Last-Modified":    {now.UTC().Format(http.TimeFormat)},
				"Vary":             {"Accept-Encoding, Accept-Language"},
			},
			resBody: ja,
		},
		"local/older than exec binary": {
			prehook: func(tb testing.TB) {
				if err := os.Chtimes(testfile, now, now.Add(-1*time.Hour)); err != nil {
					t.Fatalf("failed to update testfile mod time: %v", err)
				}
			},
			handler: must(vv.New(&vv.Config{Local: true, LocalDir: "testdata", LastModified: now, Data: testdata})),
			resHeader: http.Header{
				"Cache-Control":    {"max-age=1"},
				"Content-Encoding": nil,
				"Content-Language": {"en-US"},
				"Content-Length":   {strconv.Itoa(len(en))},
				"Content-Type":     {"text/html; charset=utf-8"},
				"Etag":             nil,
				"Last-Modified":    {now.UTC().Format(http.TimeFormat)},
				"Vary":             {"Accept-Language"},
			},
			resBody: en,
		},
		"local/newer than exec binary": {
			prehook: func(tb testing.TB) {
				if err := os.Chtimes(testfile, now, now.Add(time.Hour)); err != nil {
					t.Fatalf("failed to update testfile mod time: %v", err)
				}
			},
			handler: must(vv.New(&vv.Config{Local: true, LocalDir: "testdata", LastModified: now, Data: testdata})),
			resHeader: http.Header{
				"Cache-Control":    {"max-age=1"},
				"Content-Language": {"en-US"},
				"Content-Length":   {strconv.Itoa(len(en))},
				"Content-Type":     {"text/html; charset=utf-8"},
				"Etag":             nil,
				"Last-Modified":    {now.UTC().Add(time.Hour).Format(http.TimeFormat)}, // use file mod time
				"Vary":             {"Accept-Language"},
			},
			resBody: en,
		},
	} {
		t.Run(label, func(t *testing.T) {
			if tt.prehook != nil {
				tt.prehook(t)
			}
			r := httptest.NewRequest("GET", "http://vv.local/", nil)
			for k, v := range tt.reqHeader {
				for i := range v {
					r.Header.Add(k, v[i])
				}
			}
			t.Log(r)
			w := httptest.NewRecorder()
			tt.handler.ServeHTTP(w, r)
			resp := w.Result()
			if got, want := w.Body.String(), tt.resBody; got != want {
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

func gzstr(b string) string {
	g, err := gzip.Encode([]byte(b))
	if err != nil {
		panic(err)
	}
	return string(g)
}

func md5str(b string) string {
	hasher := md5.New()
	hasher.Write([]byte(b))
	return hex.EncodeToString(hasher.Sum(nil))
}
