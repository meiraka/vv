package api

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCacheSet(t *testing.T) {
	var oldDate time.Time
	b, err := newCache(nil)
	if err != nil {
		t.Fatalf("failed to init cache: %v", err)
	}

	if b, _, date := b.get(); string(b) != `null` || date.Equal(time.Time{}) {
		t.Errorf("got %s, _, %v; want nil, _, not time.Time{}", b, date)
	}
	b.Set(map[string]int{"a": 1})
	if b, _, date := b.get(); string(b) != `{"a":1}` || date.Equal(time.Time{}) {
		t.Errorf("got %s, _, %v; want %s, _, not time.Time{}", b, date, `{"a":1}`)
	} else {
		oldDate = date
	}
	b.SetIfModified(map[string]int{"a": 1})
	if b, _, date := b.get(); string(b) != `{"a":1}` || !date.Equal(oldDate) {
		t.Errorf("got %s, _, %v; want %s, _, %v", b, date, `{"a":1}`, oldDate)
	} else {
		oldDate = date
	}
	b.Set(map[string]int{"a": 1})
	if b, _, date := b.get(); string(b) != `{"a":1}` || date.Equal(oldDate) {
		t.Errorf("got %s, _, %v; want %s, _, not %v", b, date, `{"a":1}`, oldDate)
	} else {
		oldDate = date
	}
	b.SetIfModified(map[string]int{"a": 2})
	if b, _, date := b.get(); string(b) != `{"a":2}` || date.Equal(oldDate) {
		t.Errorf("got %s, _, %v; want %s, _, not %v", b, date, `{"a":2}`, oldDate)
	}

}

func TestCacheHandler(t *testing.T) {
	b, err := newCache(nil)
	if err != nil {
		t.Fatalf("failed to init cache: %v", err)
	}
	b.SetIfModified(map[string]int{"a": 1})
	ts := httptest.NewServer(b)
	defer ts.Close()
	body, gz, date := b.get()
	testsets := []struct {
		header http.Header
		status int
		want   []byte
	}{
		{header: http.Header{"Accept-Encoding": {"identity"}}, status: 200, want: body},
		{header: http.Header{"Accept-Encoding": {"gzip"}}, status: 200, want: gz},
		{header: http.Header{"Accept-Encoding": {"identity"}, "If-None-Match": {fmt.Sprintf(`"%d.%d"`, date.Unix(), date.Nanosecond())}}, status: 304, want: []byte{}},
		{header: http.Header{"Accept-Encoding": {"identity"}, "If-None-Match": {""}}, status: 200, want: body},
		{header: http.Header{"Accept-Encoding": {"identity"}, "If-Modified-Since": {date.Format(http.TimeFormat)}}, status: 304, want: []byte{}},
		{header: http.Header{"Accept-Encoding": {"identity"}, "If-Modified-Since": {date.Add(time.Second).Format(http.TimeFormat)}}, status: 304, want: []byte{}},
		{header: http.Header{"Accept-Encoding": {"identity"}, "If-Modified-Since": {date.Add(-1 * time.Second).Format(http.TimeFormat)}}, status: 200, want: body},
	}
	for _, tt := range testsets {
		t.Run(fmt.Sprint(tt.header), func(t *testing.T) {
			req, _ := http.NewRequest(http.MethodGet, ts.URL, nil)
			for k, v := range tt.header {
				for i := range v {
					req.Header.Add(k, v[i])
				}
			}
			resp, err := testHTTPClient.Do(req)
			if err != nil {
				t.Fatalf("failed to request: %v", err)
			}
			defer resp.Body.Close()
			got, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Errorf("got resp.Body error: %v", err)
			}
			if !bytes.Equal(got, tt.want) || resp.StatusCode != tt.status {
				t.Errorf("got %d %s; want %d %s", resp.StatusCode, got, tt.status, tt.want)
			}

		})

	}
}
