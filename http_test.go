package main

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/meiraka/vv/internal/mpd"
	"github.com/meiraka/vv/internal/mpd/mpdtest"
)

const (
	testTimeout = time.Second
)

var (
	testDialer = mpd.Dialer{
		ReconnectionTimeout:  time.Second,
		HealthCheckInterval:  time.Hour,
		ReconnectionInterval: time.Second,
	}
	testHTTPClient = &http.Client{Timeout: testTimeout}
	testHTTPConfig = HTTPHandlerConfig{
		BackgroundTimeout: time.Second,
	}
	testMPDEvent = []*mpdtest.WR{
		{Read: "listallinfo /\n", Write: "file: foo\nfile: bar\nfile: baz\nOK\n"},
		{Read: "playlistinfo\n", Write: "file: foo\nfile: bar\nOK\n"},
		{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"},
		{Read: "currentsong\n", Write: "file: bar\nOK\n"},
		{Read: "outputs\n", Write: "outputid: 0\noutputname: My ALSA Device\nplugin: alsa\noutputenabled: 0\nattribute: dop=0\nOK\n"},
		{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"},
	}
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
			if !bytes.Equal(got, tt.want) || resp.StatusCode != tt.status {
				t.Errorf("got %d %s; want %d %s", resp.StatusCode, got, tt.status, tt.want)
			}

		})

	}
}

func TestHTTPHandlerRequest(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	testsets := []struct {
		Method string
		Path   string
		Body   string
		status int
		want   string
		event  []*mpdtest.WR
	}{
		{
			Method: http.MethodGet,
			Path:   "/api/music/playlist",
			status: http.StatusOK,
			want:   `{"current":1}`,
			event:  mpdtest.Append(testMPDEvent, &mpdtest.WR{Read: "close\n"}),
		},
		{
			Method: http.MethodGet,
			Path:   "/api/music/playlist/songs",
			status: http.StatusOK,
			want:   `[{"DiscNumber":["0001"],"Length":["00:00"],"TrackNumber":["0000"],"file":["foo"]},{"DiscNumber":["0001"],"Length":["00:00"],"TrackNumber":["0000"],"file":["bar"]}]`,
			event:  mpdtest.Append(testMPDEvent, &mpdtest.WR{Read: "close\n"}),
		},
		{
			Method: http.MethodGet,
			Path:   "/api/music/library",
			status: http.StatusOK,
			want:   `{"updating":false}`,
			event:  mpdtest.Append(testMPDEvent, &mpdtest.WR{Read: "close\n"}),
		},
		{
			Method: http.MethodGet,
			Path:   "/api/music/library/songs",
			status: http.StatusOK,
			want:   `[{"DiscNumber":["0001"],"Length":["00:00"],"TrackNumber":["0000"],"file":["foo"]},{"DiscNumber":["0001"],"Length":["00:00"],"TrackNumber":["0000"],"file":["bar"]},{"DiscNumber":["0001"],"Length":["00:00"],"TrackNumber":["0000"],"file":["baz"]}]`,
			event:  mpdtest.Append(testMPDEvent, &mpdtest.WR{Read: "close\n"}),
		},
		{
			Method: http.MethodGet,
			Path:   "/api/music/playlist/songs/current",
			status: http.StatusOK,
			want:   `{"DiscNumber":["0001"],"Length":["00:00"],"TrackNumber":["0000"],"file":["bar"]}`,
			event:  mpdtest.Append(testMPDEvent, &mpdtest.WR{Read: "close\n"}),
		},
		{
			Method: http.MethodGet,
			Path:   "/api/music/outputs",
			status: http.StatusOK,
			want:   `{"0":{"name":"My ALSA Device","plugin":"alsa","enabled":false,"attribute":"dop=0"}}`,
			event:  mpdtest.Append(testMPDEvent, &mpdtest.WR{Read: "close\n"}),
		},
		{
			Method: http.MethodGet,
			Path:   "/api/music",
			status: http.StatusOK,
			want:   `{"repeat":false,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1}`,
			event:  mpdtest.Append(testMPDEvent, &mpdtest.WR{Read: "close\n"}),
		},
		{
			Method: http.MethodPost,
			Path:   "/api/music",
			Body:   `{"volume":100}`,
			status: http.StatusAccepted,
			want:   `{"repeat":false,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1}`,
			event: mpdtest.Append(testMPDEvent, []*mpdtest.WR{
				{Read: "setvol 100\n", Write: "OK\n"},
				{Read: "close\n"},
			}...),
		},
		{
			Method: http.MethodPost,
			Path:   "/api/music",
			Body:   `{"repeat":true}`,
			status: http.StatusAccepted,
			want:   `{"repeat":false,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1}`,
			event: mpdtest.Append(testMPDEvent, []*mpdtest.WR{
				{Read: "repeat 1\n", Write: "OK\n"},
				{Read: "close\n"},
			}...),
		},
		{
			Method: http.MethodPost,
			Path:   "/api/music",
			Body:   `{"random":true}`,
			status: http.StatusAccepted,
			want:   `{"repeat":false,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1}`,
			event: mpdtest.Append(testMPDEvent, []*mpdtest.WR{
				{Read: "random 1\n", Write: "OK\n"},
				{Read: "close\n"},
			}...),
		},
		{
			Method: http.MethodPost,
			Path:   "/api/music",
			Body:   `{"single":true}`,
			status: http.StatusAccepted,
			want:   `{"repeat":false,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1}`,
			event: mpdtest.Append(testMPDEvent, []*mpdtest.WR{
				{Read: "single 1\n", Write: "OK\n"},
				{Read: "close\n"},
			}...),
		},
		{
			Method: http.MethodPost,
			Path:   "/api/music",
			Body:   `{"oneshot":true}`,
			status: http.StatusAccepted,
			want:   `{"repeat":false,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1}`,
			event: mpdtest.Append(testMPDEvent, []*mpdtest.WR{
				{Read: "single oneshot\n", Write: "OK\n"},
				{Read: "close\n"},
			}...),
		},
		{
			Method: http.MethodPost,
			Path:   "/api/music",
			Body:   `{"consume":true}`,
			status: http.StatusAccepted,
			want:   `{"repeat":false,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1}`,
			event: mpdtest.Append(testMPDEvent, []*mpdtest.WR{
				{Read: "consume 1\n", Write: "OK\n"},
				{Read: "close\n"},
			}...),
		},
		{
			Method: http.MethodPost,
			Path:   "/api/music",
			Body:   `{"state":"play"}`,
			status: http.StatusAccepted,
			want:   `{"repeat":false,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1}`,
			event: mpdtest.Append(testMPDEvent, []*mpdtest.WR{
				{Read: "play -1\n", Write: "OK\n"},
				{Read: "close\n"},
			}...),
		},
		{
			Method: http.MethodPost,
			Path:   "/api/music",
			Body:   `{"state":"pause"}`,
			status: http.StatusAccepted,
			want:   `{"repeat":false,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1}`,
			event: mpdtest.Append(testMPDEvent, []*mpdtest.WR{
				{Read: "pause 1\n", Write: "OK\n"},
				{Read: "close\n"},
			}...),
		},
		{
			Method: http.MethodPost,
			Path:   "/api/music",
			Body:   `{"state":"next"}`,
			status: http.StatusAccepted,
			want:   `{"repeat":false,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1}`,
			event: mpdtest.Append(testMPDEvent, []*mpdtest.WR{
				{Read: "next\n", Write: "OK\n"},
				{Read: "close\n"},
			}...),
		},
		{
			Method: http.MethodPost,
			Path:   "/api/music",
			Body:   `{"state":"previous"}`,
			status: http.StatusAccepted,
			want:   `{"repeat":false,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1}`,
			event: mpdtest.Append(testMPDEvent, []*mpdtest.WR{
				{Read: "previous\n", Write: "OK\n"},
				{Read: "close\n"},
			}...),
		},
		{
			Method: http.MethodPost,
			Path:   "/api/music/playlist",
			Body:   `{"current":0,"sort":["file"],"filters":[]}`,
			want:   `{"current":1}`,
			status: http.StatusAccepted,
			event: mpdtest.Append(testMPDEvent, []*mpdtest.WR{
				{Read: "command_list_ok_begin\nclear\nadd \"bar\"\nadd \"baz\"\nadd \"foo\"\nplay 0\ncommand_list_end\n",
					Write: "list_OK\nlist_OK\nlist_OK\nlist_OK\nlist_OK\nOK\n"},
				{Read: "close\n"},
			}...),
		},
		{
			Method: http.MethodPost,
			Path:   "/api/music/playlist",
			Body:   `{"current":2,"sort":["file"],"filters":[["file","foo"]]}`,
			status: http.StatusAccepted,
			want:   `{"current":1}`,
			event: []*mpdtest.WR{
				{Read: "listallinfo /\n", Write: "file: foo\nfile: bar\nfile: baz\nOK\n"},
				{Read: "playlistinfo\n", Write: "file: bar\nfile: baz\nfile: foo\nOK\n"},
				{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"},
				{Read: "currentsong\n", Write: "file: bar\nOK\n"},
				{Read: "outputs\n", Write: "outputid: 0\noutputname: My ALSA Device\nplugin: alsa\noutputenabled: 0\nattribute: dop=0\nOK\n"},
				{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"},
				{Read: "play 2\n", Write: "OK\n"},
				{Read: "status\n", Write: "volume: -1\nsong: 2\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"},
				{Read: "close\n"},
			},
		},
		{
			Method: http.MethodPost,
			Path:   "/api/music/library",
			Body:   `{"updating":true}`,
			status: http.StatusAccepted,
			want:   `{"updating":false}`,
			event: mpdtest.Append(testMPDEvent, []*mpdtest.WR{
				{Read: "update \n", Write: "updating_db: 2\nOK\n"},
				{Read: "status\n", Write: "volume: -1\nsong: 2\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nupdating_db: 2\nOK\n"},
				{Read: "close\n"},
			}...),
		},
		{
			Method: http.MethodPost,
			Path:   "/api/music/outputs",
			Body:   `{"0":{"enabled":false}}`,
			status: http.StatusAccepted,
			want:   `{"0":{"name":"My ALSA Device","plugin":"alsa","enabled":false,"attribute":"dop=0"}}`,
			event: mpdtest.Append(testMPDEvent, []*mpdtest.WR{
				{Read: "disableoutput 0\n", Write: "OK\n"},
				{Read: "close\n"},
			}...),
		},
		{
			Method: http.MethodPost,
			Path:   "/api/music/outputs",
			Body:   `{"0":{"enabled":true}}`,
			status: http.StatusAccepted,
			want:   `{"0":{"name":"My ALSA Device","plugin":"alsa","enabled":false,"attribute":"dop=0"}}`,
			event: mpdtest.Append(testMPDEvent, []*mpdtest.WR{
				{Read: "enableoutput 0\n", Write: "OK\n"},
				{Read: "close\n"},
			}...),
		},
	}
	for _, tt := range testsets {
		t.Run(fmt.Sprintf("%s %s %s", tt.Method, tt.Path, tt.Body), func(t *testing.T) {
			main, err := mpdtest.NewEventServer("OK MPD 0.19", tt.event)
			defer main.Close()
			if err != nil {
				t.Fatalf("failed to create mpd test server: %v", err)
			}
			sub, err := mpdtest.NewEventServer("OK MPD 0.19", []*mpdtest.WR{{Read: "idle\nnoidle\n", Write: "OK\n"}, {Read: "close\n"}})
			defer sub.Close()
			if err != nil {
				t.Fatalf("failed to create mpd test server: %v", err)
			}
			c, err := testDialer.Dial("tcp", main.URL, "")
			if err != nil {
				t.Fatalf("Dial got error %v; want nil", err)
			}
			defer c.Close(ctx)
			wl, err := testDialer.NewWatcher("tcp", sub.URL, "")
			if err != nil {
				t.Fatalf("Dial got error %v; want nil", err)
			}
			defer wl.Close(ctx)

			h, err := testHTTPConfig.NewHTTPHandler(ctx, c, wl)
			if err != nil {
				t.Fatalf("NewHTTPHandler got error %v; want nil", err)
			}

			ts := httptest.NewServer(h)
			defer ts.Close()
			req, err := http.NewRequest(tt.Method, ts.URL+tt.Path, strings.NewReader(tt.Body))
			if err != nil {
				t.Fatalf("failed to create http request: %v", err)
			}
			resp, err := testHTTPClient.Do(req)
			if err != nil {
				t.Fatalf("failed to request: %v", err)
			}
			defer resp.Body.Close()
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read response: %v", err)
			}
			if got := string(b); got != tt.want || resp.StatusCode != tt.status {
				t.Errorf("got %d %v; want %d %v", resp.StatusCode, got, tt.status, tt.want)
			}

		})
	}
}

func TestHTTPHandlerWebSocket(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	testsets := map[string]struct {
		watcher   string
		event     []*mpdtest.WR
		websocket []string
	}{
		"database updating_db": {
			watcher: "changed: database\nOK\n",
			event: []*mpdtest.WR{
				{Read: "listallinfo /\n", Write: "file: foo\nfile: bar\nfile: baz\nOK\n"},
				{Read: "playlistinfo\n", Write: "file: foo\nfile: bar\nOK\n"},
				{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nupdating_db: 3\nOK\n"},
				{Read: "currentsong\n", Write: "file: bar\nOK\n"},
				{Read: "outputs\n", Write: "outputid: 0\noutputname: My ALSA Device\nplugin: alsa\noutputenabled: 0\nattribute: dop=0\nOK\n"},
				{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"},
				{Read: "listallinfo /\n", Write: "file: foo\nfile: bar\nfile: baz\nfile: qux\nOK\n"},
				{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"},
				{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"},
				{Read: "close\n"},
			},
			websocket: []string{"/api/music/library/songs", "/api/music", "/api/music/library", "/api/music/stats"},
		},
		"database": {
			watcher: "changed: database\nOK\n",
			event: mpdtest.Append(testMPDEvent, []*mpdtest.WR{
				{Read: "listallinfo /\n", Write: "file: foo\nfile: bar\nfile: baz\nfile: qux\nOK\n"},
				{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"},
				{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"},
				{Read: "close\n"},
			}...),
			websocket: []string{"/api/music/library/songs", "/api/music", "/api/music/stats"},
		},
		"playlist": {
			watcher: "changed: playlist\nOK\n",
			event: mpdtest.Append(testMPDEvent, []*mpdtest.WR{
				{Read: "playlistinfo\n", Write: "file: foo\nfile: bar\nfile: baz\nOK\n"},
				{Read: "close\n"},
			}...),
			websocket: []string{"/api/music/playlist/songs"},
		},
		"player": {
			watcher: "changed: player\nOK\n",
			event: mpdtest.Append(testMPDEvent, []*mpdtest.WR{
				{Read: "status\n", Write: "volume: -1\nsong: 2\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: play\nOK\n"},
				{Read: "currentsong\n", Write: "file: baz\nOK\n"},
				{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"},
				{Read: "close\n"},
			}...),
			websocket: []string{"/api/music", "/api/music/playlist", "/api/music/playlist/songs/current", "/api/music/stats"},
		},
		"mixer": {
			watcher: "changed: mixer\nOK\n",
			event: mpdtest.Append(testMPDEvent, []*mpdtest.WR{
				{Read: "status\n", Write: "volume: 100\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"},
				{Read: "close\n"},
			}...),
			websocket: []string{"/api/music"},
		},
		"options": {
			watcher: "changed: options\nOK\n",
			event: mpdtest.Append(testMPDEvent, []*mpdtest.WR{
				{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 1\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"},
				{Read: "close\n"},
			}...),
			websocket: []string{"/api/music"},
		},
		"update": {
			watcher: "changed: update\nOK\n",
			event: mpdtest.Append(testMPDEvent, []*mpdtest.WR{
				{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nupdating_db: 1\nOK\n"},
				{Read: "close\n"},
			}...),
			websocket: []string{"/api/music", "/api/music/library"},
		},
		"outputs": {
			watcher: "changed: output\nOK\n",
			event: mpdtest.Append(testMPDEvent, []*mpdtest.WR{
				{Read: "outputs\n", Write: "outputid: 0\noutputname: My ALSA Device\nplugin: alsa\noutputenabled: 1\nattribute: dop=0\nOK\n"},
				{Read: "close\n"},
			}...),
			websocket: []string{"/api/music/outputs"},
		},
	}
	for k, tt := range testsets {
		t.Run(fmt.Sprintf("%s", k), func(t *testing.T) {
			main, err := mpdtest.NewEventServer("OK MPD 0.19", tt.event)
			defer main.Close()
			if err != nil {
				t.Fatalf("failed to create mpd test server: %v", err)
			}
			w, r, sub, err := mpdtest.NewChanServer("OK MPD 0.19")
			defer sub.Close()
			if err != nil {
				t.Fatalf("failed to create mpd test server: %v", err)
			}
			c, err := testDialer.Dial("tcp", main.URL, "")
			if err != nil {
				t.Fatalf("Dial got error %v; want nil", err)
			}
			defer c.Close(ctx)
			wl, err := testDialer.NewWatcher("tcp", sub.URL, "")
			if err != nil {
				t.Fatalf("Dial got error %v; want nil", err)
			}
			defer wl.Close(ctx)

			h, err := testHTTPConfig.NewHTTPHandler(ctx, c, wl)
			if err != nil {
				t.Fatalf("NewHTTPHandler got error %v; want nil", err)
			}

			ts := httptest.NewServer(h)
			defer ts.Close()

			ws, _, err := websocket.DefaultDialer.Dial(strings.Replace(ts.URL, "http://", "ws://", 1)+"/api/music", nil)
			if err != nil {
				t.Fatalf("failed to connect websocket: %v", err)
			}
			defer ws.Close()
			ws.SetReadDeadline(time.Now().Add(testTimeout))
			if _, msg, err := ws.ReadMessage(); string(msg) != "ok" || err != nil {
				t.Fatalf("got message: %s, %v, want: ok <nil>", msg, err)
			}
			go func() {
				<-r // idle
				w <- tt.watcher
				<-r // idle
				<-r // noidle
				w <- "OK\n"
			}()
			got := make([]string, 0, 10)
			for {
				_, msg, err := ws.ReadMessage()
				if err != nil {
					t.Errorf("failed to get message: %v, got: %v, want: %v", err, got, tt.websocket)
					break
				}
				got = append(got, string(msg))
				if len(got) == len(tt.websocket) && !reflect.DeepEqual(got, tt.websocket) {
					t.Errorf("got %v; want %v", got, tt.websocket)
					break
				} else if reflect.DeepEqual(got, tt.websocket) {
					return
				}
			}
		})
	}
}

func TestHTTPHandlerPlaylist(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	testsets := map[string]struct {
		watcher   string
		f         func(chan string, <-chan string, chan string, <-chan string)
		websocket []string
		Method    string
		Path      string
		Body      string
		status    int
		want      string
	}{
		"player": {
			f: func(w chan string, r <-chan string, iw chan string, ir <-chan string) {
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "listallinfo /\n", Write: "file: foo\nfile: bar\nfile: baz\nOK\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "playlistinfo\n", Write: "file: bar\nfile: baz\nfile: foo\nOK\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "currentsong\n", Write: "file: bar\nPos: 1\nOK\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "outputs\n", Write: "outputid: 0\noutputname: My ALSA Device\nplugin: alsa\noutputenabled: 0\nattribute: dop=0\nOK\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "play 0\n", Write: "OK\n"})
				mpdtest.DefineMessage(ctx, iw, ir, &mpdtest.WR{Read: "idle\n", Write: "changed: player\nOK\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 0\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "currentsong\n", Write: "file: bar\nPos: 0\nOK\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
				mpdtest.DefineMessage(ctx, iw, ir, &mpdtest.WR{Read: "idle\n"})
				mpdtest.DefineMessage(ctx, iw, ir, &mpdtest.WR{Read: "noidle\n", Write: "OK\n"})
				mpdtest.DefineMessage(ctx, iw, ir, &mpdtest.WR{Read: "close\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "close\n"})
			},
			websocket: []string{"/api/music", "/api/music/playlist", "/api/music/playlist/songs/current", "/api/music/stats"},
			Method:    http.MethodGet,
			Path:      "/api/music/playlist",
			status:    http.StatusOK,
			want:      `{"current":0,"sort":["file"]}`,
		},
		"playlist": {
			f: func(w chan string, r <-chan string, iw chan string, ir <-chan string) {
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "listallinfo /\n", Write: "file: foo\nfile: bar\nfile: baz\nOK\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "playlistinfo\n", Write: "file: baz\nfile: foo\nOK\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "currentsong\n", Write: "file: bar\nPos: 1\nOK\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "outputs\n", Write: "outputid: 0\noutputname: My ALSA Device\nplugin: alsa\noutputenabled: 0\nattribute: dop=0\nOK\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "command_list_ok_begin\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "clear\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "add \"bar\"\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "add \"baz\"\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "add \"foo\"\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "play 0\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "command_list_end\n", Write: "list_OK\nlist_OK\nlist_OK\nlist_OK\nlist_OK\nOK\n"})
				mpdtest.DefineMessage(ctx, iw, ir, &mpdtest.WR{Read: "idle\n", Write: "changed: playlist\nOK\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "playlistinfo\n", Write: "file: bar\nfile: baz\nfile: foo\nOK\n"})
				mpdtest.DefineMessage(ctx, iw, ir, &mpdtest.WR{Read: "idle\n", Write: "changed: player\nOK\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 0\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "currentsong\n", Write: "file: bar\nPos: 0\nOK\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
				mpdtest.DefineMessage(ctx, iw, ir, &mpdtest.WR{Read: "idle\n"})
				mpdtest.DefineMessage(ctx, iw, ir, &mpdtest.WR{Read: "noidle\n", Write: "OK\n"})
				mpdtest.DefineMessage(ctx, iw, ir, &mpdtest.WR{Read: "close\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "close\n"})
			},
			websocket: []string{"/api/music/playlist/songs", "/api/music", "/api/music/playlist", "/api/music/playlist/songs/current", "/api/music/stats"},
			Method:    http.MethodGet,
			Path:      "/api/music/playlist",
			status:    http.StatusOK,
			want:      `{"current":0,"sort":["file"]}`,
		},
		"playlist changed": {
			f: func(w chan string, r <-chan string, iw chan string, ir <-chan string) {
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "listallinfo /\n", Write: "file: foo\nfile: bar\nfile: baz\nOK\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "playlistinfo\n", Write: "file: baz\nfile: foo\nOK\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "currentsong\n", Write: "file: bar\nPos: 1\nOK\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "outputs\n", Write: "outputid: 0\noutputname: My ALSA Device\nplugin: alsa\noutputenabled: 0\nattribute: dop=0\nOK\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "command_list_ok_begin\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "clear\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "add \"bar\"\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "add \"baz\"\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "add \"foo\"\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "play 0\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "command_list_end\n", Write: "list_OK\nlist_OK\nlist_OK\nlist_OK\nlist_OK\nOK\n"})
				mpdtest.DefineMessage(ctx, iw, ir, &mpdtest.WR{Read: "idle\n", Write: "changed: player\nOK\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 0\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "currentsong\n", Write: "file: bar\nPos: 0\nOK\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
				mpdtest.DefineMessage(ctx, iw, ir, &mpdtest.WR{Read: "idle\n", Write: "changed: playlist\nOK\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "playlistinfo\n", Write: "file: bar\nOK\n"})
				mpdtest.DefineMessage(ctx, iw, ir, &mpdtest.WR{Read: "idle\n"})
				mpdtest.DefineMessage(ctx, iw, ir, &mpdtest.WR{Read: "noidle\n", Write: "OK\n"})
				mpdtest.DefineMessage(ctx, iw, ir, &mpdtest.WR{Read: "close\n"})
				mpdtest.DefineMessage(ctx, w, r, &mpdtest.WR{Read: "close\n"})
			},
			websocket: []string{"/api/music", "/api/music/playlist", "/api/music/playlist/songs/current", "/api/music/stats", "/api/music/playlist/songs", "/api/music/playlist"},
			Method:    http.MethodGet,
			Path:      "/api/music/playlist",
			status:    http.StatusOK,
			want:      `{"current":0}`,
		},
	}
	for k, tt := range testsets {
		t.Run(fmt.Sprintf("%s", k), func(t *testing.T) {
			w, r, main, err := mpdtest.NewChanServer("OK MPD 0.19")
			defer main.Close()
			if err != nil {
				t.Fatalf("failed to create mpd test server: %v", err)
			}
			iw, ir, sub, err := mpdtest.NewChanServer("OK MPD 0.19")
			defer sub.Close()
			if err != nil {
				t.Fatalf("failed to create mpd test server: %v", err)
			}
			c, err := testDialer.Dial("tcp", main.URL, "")
			if err != nil {
				t.Fatalf("Dial got error %v; want nil", err)
			}
			defer c.Close(ctx)
			wl, err := testDialer.NewWatcher("tcp", sub.URL, "")
			if err != nil {
				t.Fatalf("Dial got error %v; want nil", err)
			}
			defer wl.Close(ctx)

			go tt.f(w, r, iw, ir)
			h, err := testHTTPConfig.NewHTTPHandler(ctx, c, wl)
			if err != nil {
				t.Fatalf("NewHTTPHandler got error %v; want nil", err)
			}

			ts := httptest.NewServer(h)
			defer ts.Close()

			ws, _, err := websocket.DefaultDialer.Dial(strings.Replace(ts.URL, "http://", "ws://", 1)+"/api/music", nil)
			if err != nil {
				t.Fatalf("failed to connect websocket: %v", err)
			}
			defer ws.Close()

			resp, err := testHTTPClient.Post(ts.URL+"/api/music/playlist", "application/json", strings.NewReader(`{"current":0,"sort":["file"],"filters":[]}`))
			if err != nil {
				t.Fatalf("got playlist %v, want nil", err)
			}
			defer resp.Body.Close()
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("got playlist %v, want nil", err)
			}
			if got, want := string(b), `{"current":1}`; got != want {
				t.Fatalf("got %s; want %s", got, want)
			}

			ws.SetReadDeadline(time.Now().Add(testTimeout))
			if _, msg, err := ws.ReadMessage(); string(msg) != "ok" || err != nil {
				t.Fatalf("got message: %s, %v, want: ok <nil>", msg, err)
			}
			got := make([]string, 0, 10)
			for {
				_, msg, err := ws.ReadMessage()
				if err != nil {
					t.Errorf("failed to get message: %v, got: %v, want: %v", err, got, tt.websocket)
					break
				}
				got = append(got, string(msg))
				if len(got) == len(tt.websocket) && !reflect.DeepEqual(got, tt.websocket) {
					t.Errorf("got %v; want %v", got, tt.websocket)
					break
				} else if reflect.DeepEqual(got, tt.websocket) {
					break
				}
			}
			req, err := http.NewRequest(tt.Method, ts.URL+tt.Path, strings.NewReader(tt.Body))
			if err != nil {
				t.Fatalf("failed to create http request: %v", err)
			}
			resp, err = testHTTPClient.Do(req)
			if err != nil {
				t.Fatalf("failed to request: %v", err)
			}
			defer resp.Body.Close()
			b, err = ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read response: %v", err)
			}
			if got := string(b); got != tt.want || resp.StatusCode != tt.status {
				t.Errorf("got %d %v; want %d %v", resp.StatusCode, got, tt.status, tt.want)
			}
		})
	}
}
