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

func TestJSONCacheSet(t *testing.T) {
	path := "/test"
	var oldDate time.Time
	b := newJSONCache()
	if b, gz, date := b.Get(path); b != nil || gz != nil || !date.Equal(time.Time{}) {
		t.Errorf("got %s, %s, %v; want nil, nil, time.Time{}", b, gz, date)
	}
	b.Set(path, map[string]int{"a": 1})
	if b, _, date := b.Get(path); string(b) != `{"a":1}` || date.Equal(time.Time{}) {
		t.Errorf("got %s, _, %v; want %s, _, not time.Time{}", b, date, `{"a":1}`)
	} else {
		oldDate = date
	}
	b.SetIfModified(path, map[string]int{"a": 1})
	if b, _, date := b.Get(path); string(b) != `{"a":1}` || !date.Equal(oldDate) {
		t.Errorf("got %s, _, %v; want %s, _, %v", b, date, `{"a":1}`, oldDate)
	} else {
		oldDate = date
	}
	b.Set(path, map[string]int{"a": 1})
	if b, _, date := b.Get(path); string(b) != `{"a":1}` || date.Equal(oldDate) {
		t.Errorf("got %s, _, %v; want %s, _, not %v", b, date, `{"a":1}`, oldDate)
	} else {
		oldDate = date
	}
	b.SetIfModified(path, map[string]int{"a": 2})
	if b, _, date := b.Get(path); string(b) != `{"a":2}` || date.Equal(oldDate) {
		t.Errorf("got %s, _, %v; want %s, _, not %v", b, date, `{"a":2}`, oldDate)
	}

}

func TestJSONCacheHandler(t *testing.T) {
	b := newJSONCache()
	_ = b.SetIfModified("/test", map[string]int{"a": 1})
	ts := httptest.NewServer(b.Handler("/test"))
	defer ts.Close()
	body, gz, date := b.Get("/test")
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
