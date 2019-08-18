package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
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
)

func TestAPIPHandler(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	w, r, main, err := mpdtest.NewServer("OK MPD 0.19")
	defer main.Close()
	if err != nil {
		t.Fatalf("failed to create mpd test server: %v", err)
	}
	iw, ir, sub, err := mpdtest.NewServer("OK MPD 0.19")
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

	go func() {
		mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "listallinfo /\n", Write: "file: foo\nfile: bar\nfile: baz\nOK\n"})
		mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "playlistinfo\n", Write: "file: foo\nfile: bar\nOK\n"})
		mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
		mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "currentsong\n", Write: "file: bar\nPos: 1\nOK\n"})
		mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "outputs\n", Write: "outputid: 0\noutputname: My ALSA Device\nplugin: alsa\noutputenabled: 0\nattribute: dop=0\nOK\n"})
		mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
	}()
	h, err := testHTTPConfig.NewHTTPHandler(ctx, c, wl)
	if err != nil {
		t.Fatalf("NewHTTPHandler got error %v; want nil", err)
	}

	ts := httptest.NewServer(h)
	defer ts.Close()

	testsets := []struct {
		f         func(chan string, <-chan string, chan string, <-chan string)
		websocket []string
		req       *http.Request
		status    int
		want      string
	}{
		{
			req:    NewRequest(http.MethodGet, ts.URL+"/api/music", nil),
			status: http.StatusOK,
			want:   `{"repeat":false,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1}`,
		},
		{
			req:    NewRequest(http.MethodGet, ts.URL+"/api/music/playlist", nil),
			status: http.StatusOK,
			want:   `{"current":1}`,
		},
		{
			req:    NewRequest(http.MethodGet, ts.URL+"/api/music/playlist/songs", nil),
			status: http.StatusOK,
			want:   `[{"DiscNumber":["0001"],"Length":["00:00"],"TrackNumber":["0000"],"file":["foo"]},{"DiscNumber":["0001"],"Length":["00:00"],"TrackNumber":["0000"],"file":["bar"]}]`,
		},
		{
			req:    NewRequest(http.MethodGet, ts.URL+"/api/music/library", nil),
			status: http.StatusOK,
			want:   `{"updating":false}`,
		},
		{
			req:    NewRequest(http.MethodGet, ts.URL+"/api/music/library/songs", nil),
			status: http.StatusOK,
			want:   `[{"DiscNumber":["0001"],"Length":["00:00"],"TrackNumber":["0000"],"file":["foo"]},{"DiscNumber":["0001"],"Length":["00:00"],"TrackNumber":["0000"],"file":["bar"]},{"DiscNumber":["0001"],"Length":["00:00"],"TrackNumber":["0000"],"file":["baz"]}]`,
		},
		{
			req:    NewRequest(http.MethodGet, ts.URL+"/api/music/playlist/songs/current", nil),
			status: http.StatusOK,
			want:   `{"DiscNumber":["0001"],"Length":["00:00"],"Pos":["1"],"TrackNumber":["0000"],"file":["bar"]}`,
		},
		{
			req:    NewRequest(http.MethodGet, ts.URL+"/api/music/outputs", nil),
			status: http.StatusOK,
			want:   `{"0":{"name":"My ALSA Device","plugin":"alsa","enabled":false,"attribute":"dop=0"}}`,
		},
		{
			req:    NewRequest(http.MethodGet, ts.URL+"/api/music/stats", nil),
			status: http.StatusOK,
			want:   `{"uptime":667505,"playtime":0,"artists":835,"albums":528,"songs":5715,"library_playtime":1475220,"library_update":1560656023}`,
		},
		{
			req:    NewRequest(http.MethodPost, ts.URL+"/api/music", strings.NewReader(`{"repeat":true}`)),
			status: http.StatusAccepted,
			want:   `{"repeat":false,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1}`,
			f: func(w chan string, r <-chan string, iw chan string, ir <-chan string) {
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "repeat 1\n", Write: "OK\n"})
				mpdtest.Expect(ctx, iw, ir, &mpdtest.WR{Read: "idle\n", Write: "changed: options\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
			},
			websocket: []string{"/api/music"},
		},
		{
			req:    NewRequest(http.MethodPost, ts.URL+"/api/music", strings.NewReader(`{"random":true}`)),
			status: http.StatusAccepted,
			want:   `{"repeat":true,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1}`,
			f: func(w chan string, r <-chan string, iw chan string, ir <-chan string) {
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "random 1\n", Write: "OK\n"})
				mpdtest.Expect(ctx, iw, ir, &mpdtest.WR{Read: "idle\n", Write: "changed: options\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
			},
			websocket: []string{"/api/music"},
		},
		{
			req:    NewRequest(http.MethodPost, ts.URL+"/api/music", strings.NewReader(`{"oneshot":true}`)),
			status: http.StatusAccepted,
			want:   `{"repeat":true,"random":true,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1}`,
			f: func(w chan string, r <-chan string, iw chan string, ir <-chan string) {
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "single oneshot\n", Write: "OK\n"})
				mpdtest.Expect(ctx, iw, ir, &mpdtest.WR{Read: "idle\n", Write: "changed: options\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: oneshot\nconsume: 0\nstate: pause\nOK\n"})
			},
			websocket: []string{"/api/music"},
		},
		{
			req:    NewRequest(http.MethodPost, ts.URL+"/api/music", strings.NewReader(`{"single":true}`)),
			status: http.StatusAccepted,
			want:   `{"repeat":true,"random":true,"single":false,"oneshot":true,"consume":false,"state":"pause","song_elapsed":1.1}`,
			f: func(w chan string, r <-chan string, iw chan string, ir <-chan string) {
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "single 1\n", Write: "OK\n"})
				mpdtest.Expect(ctx, iw, ir, &mpdtest.WR{Read: "idle\n", Write: "changed: options\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 1\nconsume: 0\nstate: pause\nOK\n"})
			},
			websocket: []string{"/api/music"},
		},
		{
			req:    NewRequest(http.MethodPost, ts.URL+"/api/music", strings.NewReader(`{"consume":true}`)),
			status: http.StatusAccepted,
			want:   `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1}`,
			f: func(w chan string, r <-chan string, iw chan string, ir <-chan string) {
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "consume 1\n", Write: "OK\n"})
				mpdtest.Expect(ctx, iw, ir, &mpdtest.WR{Read: "idle\n", Write: "changed: options\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 1\nconsume: 1\nstate: pause\nOK\n"})
			},
			websocket: []string{"/api/music"},
		},
		{
			req:    NewRequest(http.MethodPost, ts.URL+"/api/music", strings.NewReader(`{"state":"play"}`)),
			status: http.StatusAccepted,
			want:   `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"pause","song_elapsed":1.1}`,
			f: func(w chan string, r <-chan string, iw chan string, ir <-chan string) {
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "play -1\n", Write: "OK\n"})
				mpdtest.Expect(ctx, iw, ir, &mpdtest.WR{Read: "idle\n", Write: "changed: player\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 1\nconsume: 1\nstate: play\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "currentsong\n", Write: "file: bar\nPos: 1\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
			},
			websocket: []string{"/api/music", "/api/music/stats"},
		},
		{
			req:    NewRequest(http.MethodPost, ts.URL+"/api/music", strings.NewReader(`{"state":"next"}`)),
			status: http.StatusAccepted,
			want:   `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"play","song_elapsed":1.1}`,
			f: func(w chan string, r <-chan string, iw chan string, ir <-chan string) {
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "next\n", Write: "OK\n"})
				mpdtest.Expect(ctx, iw, ir, &mpdtest.WR{Read: "idle\n", Write: "changed: player\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 2\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 1\nconsume: 1\nstate: play\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "currentsong\n", Write: "file: baz\nPos: 2\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
			},
			websocket: []string{"/api/music", "/api/music/playlist", "/api/music/playlist/songs/current", "/api/music/stats"},
		},
		{
			req:    NewRequest(http.MethodPost, ts.URL+"/api/music", strings.NewReader(`{"state":"previous"}`)),
			status: http.StatusAccepted,
			want:   `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"play","song_elapsed":1.1}`,
			f: func(w chan string, r <-chan string, iw chan string, ir <-chan string) {
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "previous\n", Write: "OK\n"})
				mpdtest.Expect(ctx, iw, ir, &mpdtest.WR{Read: "idle\n", Write: "changed: player\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 1\nconsume: 1\nstate: play\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "currentsong\n", Write: "file: bar\nPos: 1\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
			},
			websocket: []string{"/api/music", "/api/music/playlist", "/api/music/playlist/songs/current", "/api/music/stats"},
		},
		{
			req:    NewRequest(http.MethodPost, ts.URL+"/api/music", strings.NewReader(`{"state":"pause"}`)),
			status: http.StatusAccepted,
			want:   `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"play","song_elapsed":1.1}`,
			f: func(w chan string, r <-chan string, iw chan string, ir <-chan string) {
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "pause 1\n", Write: "OK\n"})
				mpdtest.Expect(ctx, iw, ir, &mpdtest.WR{Read: "idle\n", Write: "changed: player\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 1\nconsume: 1\nstate: pause\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "currentsong\n", Write: "file: bar\nPos: 1\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
			},
			websocket: []string{"/api/music", "/api/music/stats"},
		},
		{
			req:    NewRequest(http.MethodPost, ts.URL+"/api/music", strings.NewReader(`{"volume":100}`)),
			status: http.StatusAccepted,
			want:   `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"pause","song_elapsed":1.1}`,
			f: func(w chan string, r <-chan string, iw chan string, ir <-chan string) {
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "setvol 100\n", Write: "OK\n"})
				mpdtest.Expect(ctx, iw, ir, &mpdtest.WR{Read: "idle\n", Write: "changed: mixer\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "status\n", Write: "volume: 100\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 1\nconsume: 1\nstate: pause\nOK\n"})
			},
			websocket: []string{"/api/music"},
		},
		{
			req:    NewRequest(http.MethodPost, ts.URL+"/api/music/outputs", strings.NewReader(`{"0":{"enabled":true}}`)),
			status: http.StatusAccepted,
			want:   `{"0":{"name":"My ALSA Device","plugin":"alsa","enabled":false,"attribute":"dop=0"}}`,
			f: func(w chan string, r <-chan string, iw chan string, ir <-chan string) {
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "enableoutput 0\n", Write: "OK\n"})
				mpdtest.Expect(ctx, iw, ir, &mpdtest.WR{Read: "idle\n", Write: "changed: output\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "outputs\n", Write: "outputid: 0\noutputname: My ALSA Device\nplugin: alsa\noutputenabled: 1\nattribute: dop=0\nOK\n"})
			},
			websocket: []string{"/api/music/outputs"},
		},
		{
			req:    NewRequest(http.MethodPost, ts.URL+"/api/music/outputs", strings.NewReader(`{"0":{"enabled":false}}`)),
			status: http.StatusAccepted,
			want:   `{"0":{"name":"My ALSA Device","plugin":"alsa","enabled":true,"attribute":"dop=0"}}`,
			f: func(w chan string, r <-chan string, iw chan string, ir <-chan string) {
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "disableoutput 0\n", Write: "OK\n"})
				mpdtest.Expect(ctx, iw, ir, &mpdtest.WR{Read: "idle\n", Write: "changed: output\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "outputs\n", Write: "outputid: 0\noutputname: My ALSA Device\nplugin: alsa\noutputenabled: 0\nattribute: dop=0\nOK\n"})
			},
			websocket: []string{"/api/music/outputs"},
		},
		{
			req:    NewRequest(http.MethodPost, ts.URL+"/api/music/playlist", strings.NewReader(`{"current":0,"sort":["file"],"filters":[]}`)),
			status: http.StatusAccepted,
			want:   `{"current":1}`,
			f: func(w chan string, r <-chan string, iw chan string, ir <-chan string) {
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "command_list_ok_begin\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "clear\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "add \"bar\"\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "add \"baz\"\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "add \"foo\"\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "play 0\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "command_list_end\n", Write: "list_OK\nlist_OK\nlist_OK\nlist_OK\nlist_OK\nOK\n"})
				mpdtest.Expect(ctx, iw, ir, &mpdtest.WR{Read: "idle\n", Write: "changed: playlist\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "playlistinfo\n", Write: "file: bar\nfile: baz\nfile: foo\nOK\n"})
				mpdtest.Expect(ctx, iw, ir, &mpdtest.WR{Read: "idle\n", Write: "changed: player\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 0\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "currentsong\n", Write: "file: bar\nPos: 0\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
			},
			websocket: []string{"/api/music/playlist/songs", "/api/music", "/api/music/playlist", "/api/music/playlist/songs/current", "/api/music/stats"},
		},
		{
			req:    NewRequest(http.MethodPost, ts.URL+"/api/music/playlist", strings.NewReader(`{"current":1,"sort":["file"],"filters":[]}`)),
			status: http.StatusAccepted,
			want:   `{"current":0,"sort":["file"]}`,
			f: func(w chan string, r <-chan string, iw chan string, ir <-chan string) {
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "play 1\n", Write: "OK\n"})
				mpdtest.Expect(ctx, iw, ir, &mpdtest.WR{Read: "idle\n", Write: "changed: player\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "currentsong\n", Write: "file: bar\nPos: 1\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
			},
			websocket: []string{"/api/music", "/api/music/playlist", "/api/music/playlist/songs/current", "/api/music/stats"},
		},
		{
			f: func(w chan string, r <-chan string, iw chan string, ir <-chan string) {
				mpdtest.Expect(ctx, iw, ir, &mpdtest.WR{Read: "idle\n", Write: "changed: playlist\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "playlistinfo\n", Write: "file: foo\nfile: bar\nOK\n"})
			},
			websocket: []string{"/api/music/playlist/songs", "/api/music/playlist"},
		},
		{
			req:    NewRequest(http.MethodPost, ts.URL+"/api/music/library", strings.NewReader(`{"updating":true}`)),
			status: http.StatusAccepted,
			want:   `{"updating":false}`,
			f: func(w chan string, r <-chan string, iw chan string, ir <-chan string) {
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "update \n", Write: "updating_db: 1\nOK\n"})
				mpdtest.Expect(ctx, iw, ir, &mpdtest.WR{Read: "idle\n", Write: "changed: update\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nupdating_db: 1\nOK\n"})
			},
			websocket: []string{"/api/music", "/api/music/library"},
		},
		{
			req:    NewRequest(http.MethodGet, ts.URL+"/api/music/library", nil),
			status: http.StatusOK,
			want:   `{"updating":true}`,
		},
		{
			f: func(w chan string, r <-chan string, iw chan string, ir <-chan string) {
				mpdtest.Expect(ctx, iw, ir, &mpdtest.WR{Read: "idle\n", Write: "changed: update\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
				mpdtest.Expect(ctx, iw, ir, &mpdtest.WR{Read: "idle\n", Write: "changed: database\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "listallinfo /\n", Write: "file: foo\nfile: bar\nfile: baz\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
			},
			websocket: []string{"/api/music", "/api/music/library", "/api/music/library/songs", "/api/music", "/api/music/stats"},
		},
		{
			req:    NewRequest(http.MethodGet, ts.URL+"/api/music/library", nil),
			status: http.StatusOK,
			want:   `{"updating":false}`,
		},
		{
			f: func(w chan string, r <-chan string, iw chan string, ir <-chan string) {
				sub.Disconnect(ctx)
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "listallinfo /\n", Write: "file: foo\nfile: bar\nfile: baz\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "playlistinfo\n", Write: "file: foo\nfile: bar\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "currentsong\n", Write: "file: bar\nPos: 1\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "outputs\n", Write: "outputid: 0\noutputname: My ALSA Device\nplugin: alsa\noutputenabled: 0\nattribute: dop=0\nOK\n"})
				mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
			},
			websocket: []string{"/api/music/library/songs", "/api/music/playlist/songs", "/api/music", "/api/music/stats"},
		},
	}
	for _, tt := range testsets {
		label := ""
		if tt.req != nil {
			label = fmt.Sprintf("%s:%s", tt.req.Method, tt.req.URL.Path)
		}
		if len(tt.websocket) != 0 {
			label += fmt.Sprint(tt.websocket)
		}
		t.Run(label, func(t *testing.T) {
			ws, _, err := websocket.DefaultDialer.Dial(strings.Replace(ts.URL, "http://", "ws://", 1)+"/api/music", nil)
			if err != nil {
				t.Fatalf("failed to connect websocket: %v", err)
			}
			defer ws.Close()
			timeout, _ := ctx.Deadline()
			ws.SetReadDeadline(timeout)
			if _, msg, err := ws.ReadMessage(); string(msg) != "ok" || err != nil {
				t.Fatalf("got message: %s, %v, want: ok <nil>", msg, err)
			}

			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				if tt.f != nil {
					tt.f(w, r, iw, ir)
				}
			}()

			if tt.req != nil {
				resp, err := testHTTPClient.Do(tt.req)
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
			}
			wg.Wait()

			if len(tt.websocket) != 0 {
				got := make([]string, 0, 10)
				for {
					_, msg, err := ws.ReadMessage()
					if err != nil {
						t.Errorf("failed to get websocket message: %v, got: %v, want: %v", err, got, tt.websocket)
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
			}
		})
	}
	go func() {
		mpdtest.Expect(ctx, iw, ir, &mpdtest.WR{Read: "idle\n", Write: ""})
		mpdtest.Expect(ctx, iw, ir, &mpdtest.WR{Read: "noidle\n", Write: "OK\n"})
	}()
}

func NewRequest(method, url string, body io.Reader) *http.Request {
	r, _ := http.NewRequest(method, url, body)
	return r
}
