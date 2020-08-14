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
		Timeout:              time.Second,
		HealthCheckInterval:  time.Hour,
		ReconnectionInterval: time.Second,
	}
	testHTTPClient = &http.Client{Timeout: testTimeout}
)

func TestAPIPHandlerJSON(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	main, err := mpdtest.NewServer("OK MPD 0.19")
	defer main.Close()
	if err != nil {
		t.Fatalf("failed to create mpd test server: %v", err)
	}
	sub, err := mpdtest.NewServer("OK MPD 0.19")
	defer sub.Close()
	if err != nil {
		t.Fatalf("failed to create mpd test server: %v", err)
	}
	c, err := testDialer.Dial("tcp", main.URL, "")
	if err != nil {
		t.Fatalf("Dial got error %v; want nil", err)
	}
	defer func() {
		if err := c.Close(ctx); err != nil {
			t.Errorf("mpd.Client.Close got err %v; want nil", err)
		}
	}()
	wl, err := testDialer.NewWatcher("tcp", sub.URL, "")
	if err != nil {
		t.Fatalf("Dial got error %v; want nil", err)
	}
	defer func() {
		if err := wl.Close(ctx); err != nil {
			t.Errorf("mpd.Watcher.Close got err %v; want nil", err)
		}
	}()

	go func() {
		main.Expect(ctx, &mpdtest.WR{Read: "listallinfo /\n", Write: "file: foo\nfile: bar\nfile: baz\nOK\n"})
		main.Expect(ctx, &mpdtest.WR{Read: "playlistinfo\n", Write: "file: foo\nfile: bar\nOK\n"})
		main.Expect(ctx, &mpdtest.WR{Read: "replay_gain_status\n", Write: "replay_gain_mode: off\nOK\n"})
		main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
		main.Expect(ctx, &mpdtest.WR{Read: "currentsong\n", Write: "file: bar\nPos: 1\nOK\n"})
		main.Expect(ctx, &mpdtest.WR{Read: "outputs\n", Write: "outputid: 0\noutputname: My ALSA Device\nplugin: alsa\noutputenabled: 0\nattribute: dop=0\nOK\n"})
		main.Expect(ctx, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
	}()
	h, err := APIConfig{
		BackgroundTimeout: time.Second,
	}.NewAPIHandler(ctx, c, wl)
	if err != nil {
		t.Fatalf("NewHTTPHandler got error %v; want nil", err)
	}

	ts := httptest.NewServer(h)
	defer ts.Close()

	testsets := []struct {
		f         func()
		websocket []string
		req       *http.Request
		want      map[int]string
	}{
		{
			req:  NewRequest(http.MethodGet, ts.URL+"/api/music", nil),
			want: map[int]string{http.StatusOK: `{"repeat":false,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`},
		},
		{
			req:  NewRequest(http.MethodGet, ts.URL+"/api/music/playlist", nil),
			want: map[int]string{http.StatusOK: `{"current":1}`},
		},
		{
			req:  NewRequest(http.MethodGet, ts.URL+"/api/music/playlist/songs", nil),
			want: map[int]string{http.StatusOK: `[{"DiscNumber":["0001"],"Length":["00:00"],"TrackNumber":["0000"],"file":["foo"]},{"DiscNumber":["0001"],"Length":["00:00"],"TrackNumber":["0000"],"file":["bar"]}]`},
		},
		{
			req:  NewRequest(http.MethodGet, ts.URL+"/api/music/library", nil),
			want: map[int]string{http.StatusOK: `{"updating":false}`},
		},
		{
			req:  NewRequest(http.MethodGet, ts.URL+"/api/music/library/songs", nil),
			want: map[int]string{http.StatusOK: `[{"DiscNumber":["0001"],"Length":["00:00"],"TrackNumber":["0000"],"file":["foo"]},{"DiscNumber":["0001"],"Length":["00:00"],"TrackNumber":["0000"],"file":["bar"]},{"DiscNumber":["0001"],"Length":["00:00"],"TrackNumber":["0000"],"file":["baz"]}]`},
		},
		{
			req:  NewRequest(http.MethodGet, ts.URL+"/api/music/playlist/songs/current", nil),
			want: map[int]string{http.StatusOK: `{"DiscNumber":["0001"],"Length":["00:00"],"Pos":["1"],"TrackNumber":["0000"],"file":["bar"]}`},
		},
		{
			req:  NewRequest(http.MethodGet, ts.URL+"/api/music/outputs", nil),
			want: map[int]string{http.StatusOK: `{"0":{"name":"My ALSA Device","plugin":"alsa","enabled":false,"attribute":"dop=0"}}`},
		},
		{
			req:  NewRequest(http.MethodGet, ts.URL+"/api/music/stats", nil),
			want: map[int]string{http.StatusOK: `{"uptime":667505,"playtime":0,"artists":835,"albums":528,"songs":5715,"library_playtime":1475220,"library_update":1560656023}`},
		},
		{
			req: NewRequest(http.MethodPost, ts.URL+"/api/music", strings.NewReader(`{"repeat":true}`)),
			want: map[int]string{
				http.StatusAccepted: `{"repeat":false,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
				http.StatusOK:       `{"repeat":true,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
			},
			f: func() {
				main.Expect(ctx, &mpdtest.WR{Read: "repeat 1\n", Write: "OK\n"})
				sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: options\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "replay_gain_status\n", Write: "replay_gain_mode: off\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
			},
			websocket: []string{"/api/music"},
		},
		{
			req: NewRequest(http.MethodPost, ts.URL+"/api/music", strings.NewReader(`{"random":true}`)),
			want: map[int]string{
				http.StatusAccepted: `{"repeat":true,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
				http.StatusOK:       `{"repeat":true,"random":true,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
			},
			f: func() {
				main.Expect(ctx, &mpdtest.WR{Read: "random 1\n", Write: "OK\n"})
				sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: options\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "replay_gain_status\n", Write: "replay_gain_mode: off\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
			},
			websocket: []string{"/api/music"},
		},
		{
			req: NewRequest(http.MethodPost, ts.URL+"/api/music", strings.NewReader(`{"oneshot":true}`)),
			want: map[int]string{
				http.StatusAccepted: `{"repeat":true,"random":true,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
				http.StatusOK:       `{"repeat":true,"random":true,"single":false,"oneshot":true,"consume":false,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
			},
			f: func() {
				main.Expect(ctx, &mpdtest.WR{Read: "single oneshot\n", Write: "OK\n"})
				sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: options\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "replay_gain_status\n", Write: "replay_gain_mode: off\nOK\n"})

				main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: oneshot\nconsume: 0\nstate: pause\nOK\n"})
			},
			websocket: []string{"/api/music"},
		},
		{
			req:  NewRequest(http.MethodPost, ts.URL+"/api/music", strings.NewReader(`{invalid json}`)),
			want: map[int]string{http.StatusBadRequest: `{"error":"invalid character 'i' looking for beginning of object key string"}`},
		},
		{
			req: NewRequest(http.MethodPost, ts.URL+"/api/music", strings.NewReader(`{"single":true}`)),
			want: map[int]string{
				http.StatusAccepted: `{"repeat":true,"random":true,"single":false,"oneshot":true,"consume":false,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
				http.StatusOK:       `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
			},
			f: func() {
				main.Expect(ctx, &mpdtest.WR{Read: "single 1\n", Write: "OK\n"})
				sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: options\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "replay_gain_status\n", Write: "replay_gain_mode: off\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 1\nconsume: 0\nstate: pause\nOK\n"})
			},
			websocket: []string{"/api/music"},
		},
		{
			req: NewRequest(http.MethodPost, ts.URL+"/api/music", strings.NewReader(`{"consume":true}`)),
			want: map[int]string{
				http.StatusAccepted: `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
				http.StatusOK:       `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
			},
			f: func() {
				main.Expect(ctx, &mpdtest.WR{Read: "consume 1\n", Write: "OK\n"})
				sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: options\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "replay_gain_status\n", Write: "replay_gain_mode: off\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 1\nconsume: 1\nstate: pause\nOK\n"})
			},
			websocket: []string{"/api/music"},
		},
		{
			req: NewRequest(http.MethodPost, ts.URL+"/api/music", strings.NewReader(`{"state":"play"}`)),
			want: map[int]string{
				http.StatusAccepted: `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
				http.StatusOK:       `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"play","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
			},
			f: func() {
				main.Expect(ctx, &mpdtest.WR{Read: "play -1\n", Write: "OK\n"})
				sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: player\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 1\nconsume: 1\nstate: play\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "currentsong\n", Write: "file: bar\nPos: 1\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
			},
			websocket: []string{"/api/music", "/api/music/stats"},
		},
		{
			req: NewRequest(http.MethodPost, ts.URL+"/api/music", strings.NewReader(`{"state":"next"}`)),
			want: map[int]string{
				http.StatusAccepted: `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"play","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
				http.StatusOK:       `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"play","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
			},
			f: func() {
				main.Expect(ctx, &mpdtest.WR{Read: "next\n", Write: "OK\n"})
				sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: player\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 2\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 1\nconsume: 1\nstate: play\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "currentsong\n", Write: "file: baz\nPos: 2\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
			},
			websocket: []string{"/api/music", "/api/music/playlist", "/api/music/playlist/songs/current", "/api/music/stats"},
		},
		{
			req: NewRequest(http.MethodPost, ts.URL+"/api/music", strings.NewReader(`{"state":"previous"}`)),
			want: map[int]string{
				http.StatusAccepted: `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"play","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
				http.StatusOK:       `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"play","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
			},
			f: func() {
				main.Expect(ctx, &mpdtest.WR{Read: "previous\n", Write: "OK\n"})
				sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: player\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 1\nconsume: 1\nstate: play\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "currentsong\n", Write: "file: bar\nPos: 1\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
			},
			websocket: []string{"/api/music", "/api/music/playlist", "/api/music/playlist/songs/current", "/api/music/stats"},
		},
		{
			req: NewRequest(http.MethodPost, ts.URL+"/api/music", strings.NewReader(`{"state":"pause"}`)),
			want: map[int]string{
				http.StatusAccepted: `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"play","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
				http.StatusOK:       `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
			},
			f: func() {
				main.Expect(ctx, &mpdtest.WR{Read: "pause 1\n", Write: "OK\n"})
				sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: player\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 1\nconsume: 1\nstate: pause\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "currentsong\n", Write: "file: bar\nPos: 1\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
			},
			websocket: []string{"/api/music", "/api/music/stats"},
		},
		{
			req: NewRequest(http.MethodPost, ts.URL+"/api/music", strings.NewReader(`{"volume":100}`)),
			want: map[int]string{
				http.StatusAccepted: `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
				http.StatusOK:       `{"volume":100,"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
			},
			f: func() {
				main.Expect(ctx, &mpdtest.WR{Read: "setvol 100\n", Write: "OK\n"})
				sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: mixer\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: 100\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 1\nconsume: 1\nstate: pause\nOK\n"})
			},
			websocket: []string{"/api/music"},
		},
		{
			req: NewRequest(http.MethodPost, ts.URL+"/api/music", strings.NewReader(`{"replay_gain":"track"}`)),
			want: map[int]string{
				http.StatusAccepted: `{"volume":100,"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
				http.StatusOK:       `{"volume":100,"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"pause","song_elapsed":1.1,"replay_gain":"track","crossfade":0}`,
			},
			f: func() {
				main.Expect(ctx, &mpdtest.WR{Read: "replay_gain_mode track\n", Write: "OK\n"})
				sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: options\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "replay_gain_status\n", Write: "replay_gain_mode: track\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: 100\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 1\nconsume: 1\nstate: pause\nOK\n"})
			},
			websocket: []string{"/api/music"},
		},
		{
			req: NewRequest(http.MethodPost, ts.URL+"/api/music", strings.NewReader(`{"crossfade":1}`)),
			want: map[int]string{
				http.StatusAccepted: `{"volume":100,"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"pause","song_elapsed":1.1,"replay_gain":"track","crossfade":0}`,
				http.StatusOK:       `{"volume":100,"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"pause","song_elapsed":1.1,"replay_gain":"track","crossfade":1}`,
			},
			f: func() {
				main.Expect(ctx, &mpdtest.WR{Read: "crossfade 1\n", Write: "OK\n"})
				sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: options\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "replay_gain_status\n", Write: "replay_gain_mode: track\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: 100\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 1\nconsume: 1\nstate: pause\nxfade: 1\nOK\n"})
			},
			websocket: []string{"/api/music"},
		},
		{
			req:  NewRequest(http.MethodPost, ts.URL+"/api/music/outputs", strings.NewReader(`{invalid json}`)),
			want: map[int]string{http.StatusBadRequest: `{"error":"invalid character 'i' looking for beginning of object key string"}`},
		},
		{
			req: NewRequest(http.MethodPost, ts.URL+"/api/music/outputs", strings.NewReader(`{"0":{"enabled":true}}`)),
			want: map[int]string{
				http.StatusAccepted: `{"0":{"name":"My ALSA Device","plugin":"alsa","enabled":false,"attribute":"dop=0"}}`,
				http.StatusOK:       `{"0":{"name":"My ALSA Device","plugin":"alsa","enabled":true,"attribute":"dop=0"}}`,
			},
			f: func() {
				main.Expect(ctx, &mpdtest.WR{Read: "enableoutput 0\n", Write: "OK\n"})
				sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: output\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "outputs\n", Write: "outputid: 0\noutputname: My ALSA Device\nplugin: alsa\noutputenabled: 1\nattribute: dop=0\nOK\n"})
			},
			websocket: []string{"/api/music/outputs"},
		},
		{
			req: NewRequest(http.MethodPost, ts.URL+"/api/music/outputs", strings.NewReader(`{"0":{"enabled":false}}`)),
			want: map[int]string{
				http.StatusAccepted: `{"0":{"name":"My ALSA Device","plugin":"alsa","enabled":true,"attribute":"dop=0"}}`,
				http.StatusOK:       `{"0":{"name":"My ALSA Device","plugin":"alsa","enabled":false,"attribute":"dop=0"}}`,
			},
			f: func() {
				main.Expect(ctx, &mpdtest.WR{Read: "disableoutput 0\n", Write: "OK\n"})
				sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: output\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "outputs\n", Write: "outputid: 0\noutputname: My ALSA Device\nplugin: alsa\noutputenabled: 0\nattribute: dop=0\nOK\n"})
			},
			websocket: []string{"/api/music/outputs"},
		},
		{
			req:  NewRequest(http.MethodPost, ts.URL+"/api/music/playlist", strings.NewReader(`{invalid json}`)),
			want: map[int]string{http.StatusBadRequest: `{"error":"invalid character 'i' looking for beginning of object key string"}`},
		},
		{
			req:  NewRequest(http.MethodPost, ts.URL+"/api/music/playlist", strings.NewReader(`{}`)),
			want: map[int]string{http.StatusBadRequest: `{"error":"filters and sort fields are required"}`},
		},
		{
			req:  NewRequest(http.MethodPost, ts.URL+"/api/music/playlist", strings.NewReader(`{"current":0,"sort":["file"],"filters":[]}`)),
			want: map[int]string{http.StatusAccepted: `{"current":1}`},
			f: func() {
				main.Expect(ctx, &mpdtest.WR{Read: "command_list_ok_begin\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "clear\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "add \"bar\"\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "add \"baz\"\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "add \"foo\"\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "play 0\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "command_list_end\n", Write: "list_OK\nlist_OK\nlist_OK\nlist_OK\nlist_OK\nOK\n"})
				sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: playlist\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "playlistinfo\n", Write: "file: bar\nfile: baz\nfile: foo\nOK\n"})
				sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: player\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 0\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "currentsong\n", Write: "file: bar\nPos: 0\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
			},
			websocket: []string{"/api/music/playlist/songs", "/api/music", "/api/music/playlist", "/api/music/playlist/songs/current", "/api/music/stats"},
		},
		{
			req: NewRequest(http.MethodPost, ts.URL+"/api/music/playlist", strings.NewReader(`{"current":1,"sort":["file"],"filters":[]}`)),
			want: map[int]string{
				http.StatusAccepted: `{"current":0,"sort":["file"]}`,
				http.StatusOK:       `{"current":1,"sort":["file"]}`,
			},
			f: func() {
				main.Expect(ctx, &mpdtest.WR{Read: "play 1\n", Write: "OK\n"})
				sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: player\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "currentsong\n", Write: "file: bar\nPos: 1\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
			},
			websocket: []string{"/api/music", "/api/music/playlist", "/api/music/playlist/songs/current", "/api/music/stats"},
		},
		{
			f: func() {
				sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: playlist\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "playlistinfo\n", Write: "file: foo\nfile: bar\nOK\n"})
			},
			websocket: []string{"/api/music/playlist/songs", "/api/music/playlist"},
		},
		{
			req:  NewRequest(http.MethodPost, ts.URL+"/api/music/library", strings.NewReader(`{invalid json}`)),
			want: map[int]string{http.StatusBadRequest: `{"error":"invalid character 'i' looking for beginning of object key string"}`},
		},
		{
			req:  NewRequest(http.MethodPost, ts.URL+"/api/music/library", strings.NewReader(`{}`)),
			want: map[int]string{http.StatusBadRequest: `{"error":"requires updating=true"}`},
		},
		{
			req:  NewRequest(http.MethodPost, ts.URL+"/api/music/library", strings.NewReader(`{"updating":true}`)),
			want: map[int]string{http.StatusInternalServerError: `{"error":"mpd: update: error"}`},
			f: func() {
				main.Expect(ctx, &mpdtest.WR{Read: "update \n", Write: "ACK [2@1] {update} error\n"})
			},
		},
		{
			req: NewRequest(http.MethodPost, ts.URL+"/api/music/library", strings.NewReader(`{"updating":true}`)),
			want: map[int]string{
				http.StatusAccepted: `{"updating":false}`,
				http.StatusOK:       `{"updating":true}`,
			},
			f: func() {
				main.Expect(ctx, &mpdtest.WR{Read: "update \n", Write: "updating_db: 1\nOK\n"})
				sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: update\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nupdating_db: 1\nOK\n"})
			},
			websocket: []string{"/api/music", "/api/music/library"},
		},
		{
			req:  NewRequest(http.MethodGet, ts.URL+"/api/music/library", nil),
			want: map[int]string{http.StatusOK: `{"updating":true}`},
		},
		{
			f: func() {
				sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: update\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
				sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: database\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "listallinfo /\n", Write: "file: foo\nfile: bar\nfile: baz\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
			},
			websocket: []string{"/api/music", "/api/music/library", "/api/music/library/songs", "/api/music", "/api/music/stats"},
		},
		{
			req:  NewRequest(http.MethodGet, ts.URL+"/api/music/library", nil),
			want: map[int]string{http.StatusOK: `{"updating":false}`},
		},
		{
			f: func() {
				sub.Expect(ctx, &mpdtest.WR{Read: "idle\n"})
				sub.Disconnect(ctx)
				main.Expect(ctx, &mpdtest.WR{Read: "listallinfo /\n", Write: "file: foo\nfile: bar\nfile: baz\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "playlistinfo\n", Write: "file: foo\nfile: bar\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "replay_gain_status\n", Write: "replay_gain_mode: off\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "currentsong\n", Write: "file: bar\nPos: 1\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "outputs\n", Write: "outputid: 0\noutputname: My ALSA Device\nplugin: alsa\noutputenabled: 0\nattribute: dop=0\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
				sub.Expect(ctx, &mpdtest.WR{Read: "idle\n"})
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
					tt.f()
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
				want, ok := tt.want[resp.StatusCode]
				if !ok {
					t.Errorf("got %d %s, want %v", resp.StatusCode, b, tt.want)
				} else if got := string(b); got != want {
					t.Errorf("got %d %v; want %d %v", resp.StatusCode, got, resp.StatusCode, want)
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
		sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: ""})
		sub.Expect(ctx, &mpdtest.WR{Read: "noidle\n", Write: "OK\n"})
	}()
}

func NewRequest(method, url string, body io.Reader) *http.Request {
	r, _ := http.NewRequest(method, url, body)
	return r
}
