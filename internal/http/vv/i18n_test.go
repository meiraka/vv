package vv

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
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

func TestI18NHandler(t *testing.T) {
	b := []byte(`{{ or .lang "en" }}`)
	h, err := i18nHandler("assets/app.html", b, nil)
	if err != nil {
		t.Fatalf("failed to init handler: %v", err)
	}
	etag, err := etag(b, nil)
	if err != nil {
		t.Fatalf("failed to generate etag: %v", err)
	}
	if len(etag) <= 20 {
		t.Fatalf("got etag %s; want etag length should be greater than 20", etag)
	}
	ts := httptest.NewServer(h)
	defer ts.Close()
	testsets := map[string]struct {
		header http.Header
		status int
		body   []byte
	}{
		"no-gzip":             {header: http.Header{"Accept-Encoding": {"identity"}}, status: http.StatusOK, body: []byte("en")},
		"gzip":                {header: http.Header{"Accept-Encoding": {"gzip"}}, status: http.StatusOK, body: Gzip(t, []byte("en"))},
		"if none match":       {header: http.Header{"If-None-Match": {etag}}, status: http.StatusNotModified, body: []byte("")},
		"ja:no-gzip":          {header: http.Header{"Accept-Language": {"ja"}, "Accept-Encoding": {"identity"}}, status: http.StatusOK, body: []byte("ja")},
		"ja:gzip":             {header: http.Header{"Accept-Language": {"ja"}, "Accept-Encoding": {"gzip"}}, status: http.StatusOK, body: Gzip(t, []byte("ja"))},
		"ja:if none match":    {header: http.Header{"Accept-Language": {"ja"}, "If-None-Match": {etag}}, status: http.StatusNotModified, body: []byte("")},
		"ja-JP:no-gzip":       {header: http.Header{"Accept-Language": {"ja-JP"}, "Accept-Encoding": {"identity"}}, status: http.StatusOK, body: []byte("ja")},
		"ja-JP:gzip":          {header: http.Header{"Accept-Language": {"ja-JP"}, "Accept-Encoding": {"gzip"}}, status: http.StatusOK, body: Gzip(t, []byte("ja"))},
		"ja-JP:if none match": {header: http.Header{"Accept-Language": {"ja-JP"}, "If-None-Match": {etag}}, status: http.StatusNotModified, body: []byte("")},
		"en-US:no-gzip":       {header: http.Header{"Accept-Language": {"en-US"}, "Accept-Encoding": {"identity"}}, status: http.StatusOK, body: []byte("en")},
		"en-US:gzip":          {header: http.Header{"Accept-Language": {"en-US"}, "Accept-Encoding": {"gzip"}}, status: http.StatusOK, body: Gzip(t, []byte("en"))},
		"en-US:if none match": {header: http.Header{"Accept-Language": {"en-US"}, "If-None-Match": {etag}}, status: http.StatusNotModified, body: []byte("")},
		"en-GB:no-gzip":       {header: http.Header{"Accept-Language": {"en-US"}, "Accept-Encoding": {"identity"}}, status: http.StatusOK, body: []byte("en")},
		"en-GB:gzip":          {header: http.Header{"Accept-Language": {"en-US"}, "Accept-Encoding": {"gzip"}}, status: http.StatusOK, body: Gzip(t, []byte("en"))},
		"en-GB:if none match": {header: http.Header{"Accept-Language": {"en-US"}, "If-None-Match": {etag}}, status: http.StatusNotModified, body: []byte("")},
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
