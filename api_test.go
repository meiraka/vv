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
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/meiraka/vv/internal/mpd"
	"github.com/meiraka/vv/internal/mpd/mpdtest"
	"github.com/meiraka/vv/internal/songs/cover"
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

type testRequest struct {
	initFunc      func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server)
	preWebSocket  []string
	method        string
	path          string
	body          io.Reader
	want          map[int]string
	postWebSocket []string
}

func TestAPIJSONPHandler(t *testing.T) {
	for label, tt := range map[string]struct {
		config   APIConfig
		initFunc func(ctx context.Context, main *mpdtest.Server)
		tests    []*testRequest
	}{
		"init": {
			config: APIConfig{BackgroundTimeout: time.Second},
			initFunc: func(ctx context.Context, main *mpdtest.Server) {
				main.Expect(ctx, &mpdtest.WR{Read: "listallinfo /\n", Write: "file: foo\nfile: bar\nfile: baz\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "playlistinfo\n", Write: "file: foo\nfile: bar\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "replay_gain_status\n", Write: "replay_gain_mode: off\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "currentsong\n", Write: "file: bar\nPos: 1\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "outputs\n", Write: "outputid: 0\noutputname: My ALSA Device\noutputenabled: 0\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "listmounts\n", Write: "mount: \nstorage: /home/foo/music\nmount: foo\nstorage: nfs://192.168.1.4/export/mp3\nOK\n"})
			},
			tests: []*testRequest{
				{
					method: http.MethodGet, path: "/api/music",
					want: map[int]string{http.StatusOK: `{"repeat":false,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`},
				},
				{
					method: http.MethodGet, path: "/api/music/playlist",
					want: map[int]string{http.StatusOK: `{"current":1}`},
				},
				{
					method: http.MethodGet, path: "/api/music/playlist/songs",
					want: map[int]string{http.StatusOK: `[{"DiscNumber":["0001"],"Length":["00:00"],"TrackNumber":["0000"],"file":["foo"]},{"DiscNumber":["0001"],"Length":["00:00"],"TrackNumber":["0000"],"file":["bar"]}]`},
				},
				{
					method: http.MethodGet, path: "/api/music/library",
					want: map[int]string{http.StatusOK: `{"updating":false}`},
				},
				{
					method: http.MethodGet, path: "/api/music/library/songs",
					want: map[int]string{http.StatusOK: `[{"DiscNumber":["0001"],"Length":["00:00"],"TrackNumber":["0000"],"file":["foo"]},{"DiscNumber":["0001"],"Length":["00:00"],"TrackNumber":["0000"],"file":["bar"]},{"DiscNumber":["0001"],"Length":["00:00"],"TrackNumber":["0000"],"file":["baz"]}]`},
				},
				{
					method: http.MethodGet, path: "/api/music/playlist/songs/current",
					want: map[int]string{http.StatusOK: `{"DiscNumber":["0001"],"Length":["00:00"],"Pos":["1"],"TrackNumber":["0000"],"file":["bar"]}`},
				},
				{
					method: http.MethodGet, path: "/api/music/outputs",
					want: map[int]string{http.StatusOK: `{"0":{"name":"My ALSA Device","enabled":false}}`},
				},
				{
					method: http.MethodGet, path: "/api/music/stats",
					want: map[int]string{http.StatusOK: `{"uptime":667505,"playtime":0,"artists":835,"albums":528,"songs":5715,"library_playtime":1475220,"library_update":1560656023}`},
				},
				{
					method: http.MethodGet, path: "/api/music/storage",
					want: map[int]string{http.StatusOK: `{"":{"uri":"/home/foo/music"},"foo":{"uri":"nfs://192.168.1.4/export/mp3"}}`},
				},
			},
		},
		"reconnect": {
			config: APIConfig{BackgroundTimeout: time.Second, skipInit: true},

			tests: []*testRequest{
				{
					method: http.MethodGet, path: "/api/music/playlist",
					want: map[int]string{http.StatusOK: ""},
				},
				{
					method: http.MethodGet, path: "/api/music/playlist/songs",
					want: map[int]string{http.StatusOK: ""},
				},
				{
					method: http.MethodGet, path: "/api/music/library",
					want: map[int]string{http.StatusOK: ""},
				},
				{
					method: http.MethodGet, path: "/api/music/library/songs",
					want: map[int]string{http.StatusOK: ""},
				},
				{
					method: http.MethodGet, path: "/api/music/playlist/songs/current",
					want: map[int]string{http.StatusOK: ""},
				},
				{
					method: http.MethodGet, path: "/api/music/outputs",
					want: map[int]string{http.StatusOK: ""},
				},
				{
					method: http.MethodGet, path: "/api/music/stats",
					want: map[int]string{http.StatusOK: ""},
				},
				{
					method: http.MethodGet, path: "/api/music/storage",
					want: map[int]string{http.StatusOK: ""},
				},
				{
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n"})
						sub.Disconnect(ctx)
						main.Expect(ctx, &mpdtest.WR{Read: "listallinfo /\n", Write: "file: foo\nfile: bar\nfile: baz\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "playlistinfo\n", Write: "file: foo\nfile: bar\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "replay_gain_status\n", Write: "replay_gain_mode: off\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "currentsong\n", Write: "file: bar\nPos: 1\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "outputs\n", Write: "outputid: 0\noutputname: My ALSA Device\noutputenabled: 0\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "listmounts\n", Write: "mount: \nstorage: /home/foo/music\nmount: foo\nstorage: nfs://192.168.1.4/export/mp3\nOK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n"})
					},
					preWebSocket: []string{"/api/music/library/songs", "/api/music/playlist", "/api/music/playlist/songs", "/api/music", "/api/music/playlist", "/api/music/library", "/api/music/playlist/songs/current", "/api/music/outputs", "/api/music/stats", "/api/music/storage"},
					method:       http.MethodGet, path: "/api/music",
					want: map[int]string{http.StatusOK: `{"repeat":false,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`},
				},
				{
					method: http.MethodGet, path: "/api/music/playlist",
					want: map[int]string{http.StatusOK: `{"current":1}`},
				},
				{
					method: http.MethodGet, path: "/api/music/playlist/songs",
					want: map[int]string{http.StatusOK: `[{"DiscNumber":["0001"],"Length":["00:00"],"TrackNumber":["0000"],"file":["foo"]},{"DiscNumber":["0001"],"Length":["00:00"],"TrackNumber":["0000"],"file":["bar"]}]`},
				},
				{
					method: http.MethodGet, path: "/api/music/library",
					want: map[int]string{http.StatusOK: `{"updating":false}`},
				},
				{
					method: http.MethodGet, path: "/api/music/library/songs",
					want: map[int]string{http.StatusOK: `[{"DiscNumber":["0001"],"Length":["00:00"],"TrackNumber":["0000"],"file":["foo"]},{"DiscNumber":["0001"],"Length":["00:00"],"TrackNumber":["0000"],"file":["bar"]},{"DiscNumber":["0001"],"Length":["00:00"],"TrackNumber":["0000"],"file":["baz"]}]`},
				},
				{
					method: http.MethodGet, path: "/api/music/playlist/songs/current",
					want: map[int]string{http.StatusOK: `{"DiscNumber":["0001"],"Length":["00:00"],"Pos":["1"],"TrackNumber":["0000"],"file":["bar"]}`},
				},
				{
					method: http.MethodGet, path: "/api/music/outputs",
					want: map[int]string{http.StatusOK: `{"0":{"name":"My ALSA Device","enabled":false}}`},
				},
				{
					method: http.MethodGet, path: "/api/music/stats",
					want: map[int]string{http.StatusOK: `{"uptime":667505,"playtime":0,"artists":835,"albums":528,"songs":5715,"library_playtime":1475220,"library_update":1560656023}`},
				},
				{
					method: http.MethodGet, path: "/api/music/storage",
					want: map[int]string{http.StatusOK: `{"":{"uri":"/home/foo/music"},"foo":{"uri":"nfs://192.168.1.4/export/mp3"}}`},
				},
			},
		},
		`POST /api/music {"repeat":true}`: {
			config: APIConfig{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music", body: strings.NewReader(`{"repeat":true}`),
					want: map[int]string{
						http.StatusAccepted: "",
						http.StatusOK:       `{"repeat":true,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
					},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "repeat 1\n", Write: "OK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: options\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "replay_gain_status\n", Write: "replay_gain_mode: off\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
					},
					postWebSocket: []string{"/api/music", "/api/music/playlist", "/api/music/library"},
				},
				{
					method: http.MethodGet, path: "/api/music",
					want: map[int]string{http.StatusOK: `{"repeat":true,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`},
				},
			}},
		`POST /api/music {"random":true}`: {
			config: APIConfig{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music", body: strings.NewReader(`{"random":true}`),
					want: map[int]string{
						http.StatusAccepted: "",
						http.StatusOK:       `{"repeat":true,"random":true,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
					},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "random 1\n", Write: "OK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: options\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "replay_gain_status\n", Write: "replay_gain_mode: off\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
					},
					postWebSocket: []string{"/api/music", "/api/music/playlist", "/api/music/library"},
				},
				{
					method: http.MethodGet, path: "/api/music",
					want: map[int]string{http.StatusOK: `{"repeat":true,"random":true,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`},
				},
			}},
		`POST /api/music {"oneshot":true}`: {
			config: APIConfig{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music", body: strings.NewReader(`{"oneshot":true}`),
					want: map[int]string{
						http.StatusAccepted: "",
						http.StatusOK:       `{"repeat":true,"random":true,"single":false,"oneshot":true,"consume":false,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
					},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "single oneshot\n", Write: "OK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: options\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "replay_gain_status\n", Write: "replay_gain_mode: off\nOK\n"})

						main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: oneshot\nconsume: 0\nstate: pause\nOK\n"})
					},
					postWebSocket: []string{"/api/music", "/api/music/playlist", "/api/music/library"},
				},
				{
					method: http.MethodGet, path: "/api/music",
					want: map[int]string{http.StatusOK: `{"repeat":true,"random":true,"single":false,"oneshot":true,"consume":false,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`},
				},
			}},
		`POST /api/music {invalid json}`: {
			config: APIConfig{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music", body: strings.NewReader(`{invalid json}`),
					want: map[int]string{http.StatusBadRequest: `{"error":"invalid character 'i' looking for beginning of object key string"}`},
				}}},
		`POST /api/music {"single":true}`: {
			config: APIConfig{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music", body: strings.NewReader(`{"single":true}`),
					want: map[int]string{
						http.StatusAccepted: "",
						http.StatusOK:       `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
					},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "single 1\n", Write: "OK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: options\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "replay_gain_status\n", Write: "replay_gain_mode: off\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 1\nconsume: 0\nstate: pause\nOK\n"})
					},
					postWebSocket: []string{"/api/music", "/api/music/playlist", "/api/music/library"},
				},
				{
					method: http.MethodGet, path: "/api/music",
					want: map[int]string{http.StatusOK: `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`},
				},
			}},
		`POST /api/music {"consume":true}`: {
			config: APIConfig{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music", body: strings.NewReader(`{"consume":true}`),
					want: map[int]string{
						http.StatusAccepted: "",
						http.StatusOK:       `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
					},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "consume 1\n", Write: "OK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: options\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "replay_gain_status\n", Write: "replay_gain_mode: off\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 1\nconsume: 1\nstate: pause\nOK\n"})
					},
					postWebSocket: []string{"/api/music", "/api/music/playlist", "/api/music/library"},
				},
				{
					method: http.MethodGet, path: "/api/music",
					want: map[int]string{http.StatusOK: `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`},
				},
			}},
		`POST /api/music {"state":"unknown"}`: {
			config: APIConfig{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music", body: strings.NewReader(`{"state":"unknown"}`),
					want: map[int]string{http.StatusBadRequest: `{"error":"unknown state: unknown"}`},
				},
			}},
		`POST /api/music {"state":"play"}`: {
			config: APIConfig{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music", body: strings.NewReader(`{"state":"play"}`),
					want: map[int]string{
						http.StatusAccepted: "",
						http.StatusOK:       `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"play","song_elapsed":1.1,"replay_gain":"","crossfade":0}`,
					},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "play -1\n", Write: "OK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: player\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 1\nconsume: 1\nstate: play\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "currentsong\n", Write: "file: bar\nPos: 1\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
					},
					postWebSocket: []string{"/api/music", "/api/music/playlist", "/api/music/library", "/api/music/playlist/songs/current", "/api/music/stats"},
				},
				{
					method: http.MethodGet, path: "/api/music",
					want: map[int]string{http.StatusOK: `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"play","song_elapsed":1.1,"replay_gain":"","crossfade":0}`},
				},
			}},
		`POST /api/music {"state":"next"}`: {
			config: APIConfig{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music", body: strings.NewReader(`{"state":"next"}`),
					want: map[int]string{
						http.StatusAccepted: "",
						http.StatusOK:       `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"play","song_elapsed":1.1,"replay_gain":"","crossfade":0}`,
					},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "next\n", Write: "OK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: player\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 2\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 1\nconsume: 1\nstate: play\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "currentsong\n", Write: "file: baz\nPos: 2\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
					},
					postWebSocket: []string{"/api/music", "/api/music/playlist", "/api/music/library", "/api/music/playlist/songs/current", "/api/music/stats"},
				},
				{
					method: http.MethodGet, path: "/api/music",
					want: map[int]string{http.StatusOK: `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"play","song_elapsed":1.1,"replay_gain":"","crossfade":0}`},
				},
			}},
		`POST /api/music {"state":"previous"}`: {
			config: APIConfig{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music", body: strings.NewReader(`{"state":"previous"}`),
					want: map[int]string{
						http.StatusAccepted: "",
						http.StatusOK:       `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"play","song_elapsed":1.1,"replay_gain":"","crossfade":0}`,
					},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "previous\n", Write: "OK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: player\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 1\nconsume: 1\nstate: play\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "currentsong\n", Write: "file: bar\nPos: 1\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
					},
					postWebSocket: []string{"/api/music", "/api/music/playlist", "/api/music/library", "/api/music/playlist/songs/current", "/api/music/stats"},
				},
				{
					method: http.MethodGet, path: "/api/music",
					want: map[int]string{http.StatusOK: `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"play","song_elapsed":1.1,"replay_gain":"","crossfade":0}`},
				},
			}},
		`POST /api/music {"state":"pause"}`: {
			config: APIConfig{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music", body: strings.NewReader(`{"state":"pause"}`),
					want: map[int]string{
						http.StatusAccepted: "",
						http.StatusOK:       `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"pause","song_elapsed":1.1,"replay_gain":"","crossfade":0}`,
					},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "pause 1\n", Write: "OK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: player\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 1\nconsume: 1\nstate: pause\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "currentsong\n", Write: "file: bar\nPos: 1\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
					},
					postWebSocket: []string{"/api/music", "/api/music/playlist", "/api/music/library", "/api/music/playlist/songs/current", "/api/music/stats"},
				},
				{
					method: http.MethodGet, path: "/api/music",
					want: map[int]string{http.StatusOK: `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"pause","song_elapsed":1.1,"replay_gain":"","crossfade":0}`},
				},
			}},
		`POST /api/music {"volume":"100"}`: {
			config: APIConfig{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music", body: strings.NewReader(`{"volume":100}`),
					want: map[int]string{
						http.StatusAccepted: "",
						http.StatusOK:       `{"volume":100,"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"pause","song_elapsed":1.1,"replay_gain":"","crossfade":0}`,
					},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "setvol 100\n", Write: "OK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: mixer\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: 100\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 1\nconsume: 1\nstate: pause\nOK\n"})
					},
					postWebSocket: []string{"/api/music"},
				},
				{
					method: http.MethodGet, path: "/api/music",
					want: map[int]string{http.StatusOK: `{"volume":100,"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"pause","song_elapsed":1.1,"replay_gain":"","crossfade":0}`},
				},
			}},
		`POST /api/music {"song_elapsed":"100.1"}`: {
			config: APIConfig{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music", body: strings.NewReader(`{"song_elapsed":100.1}`),
					want: map[int]string{
						http.StatusAccepted: "",
						http.StatusOK:       `{"volume":100,"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"pause","song_elapsed":100.1,"replay_gain":"","crossfade":0}`,
					},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "seekcur 100.1\n", Write: "OK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: player\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: 100\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 1\nconsume: 1\nstate: pause\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "currentsong\n", Write: "file: bar\nPos: 0\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
					},
					postWebSocket: []string{"/api/music"},
				},
				{
					method: http.MethodGet, path: "/api/music",
					want: map[int]string{http.StatusOK: `{"volume":100,"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"pause","song_elapsed":1.1,"replay_gain":"","crossfade":0}`},
				},
			}},
		`POST /api/music {"replay_gain":"track"}`: {
			config: APIConfig{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music", body: strings.NewReader(`{"replay_gain":"track"}`),
					want: map[int]string{
						http.StatusAccepted: "",
						http.StatusOK:       `{"volume":100,"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"pause","song_elapsed":1.1,"replay_gain":"track","crossfade":0}`,
					},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "replay_gain_mode track\n", Write: "OK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: options\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "replay_gain_status\n", Write: "replay_gain_mode: track\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: 100\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 1\nconsume: 1\nstate: pause\nOK\n"})
					},
					postWebSocket: []string{"/api/music"},
				},
				{
					method: http.MethodGet, path: "/api/music",
					want: map[int]string{http.StatusOK: `{"volume":100,"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"pause","song_elapsed":1.1,"replay_gain":"track","crossfade":0}`},
				},
			}},
		`POST /api/music {"crossfade":1}`: {
			config: APIConfig{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music", body: strings.NewReader(`{"crossfade":1}`),
					want: map[int]string{
						http.StatusAccepted: "",
						http.StatusOK:       `{"volume":100,"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"pause","song_elapsed":1.1,"replay_gain":"track","crossfade":1}`,
					},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "crossfade 1\n", Write: "OK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: options\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "replay_gain_status\n", Write: "replay_gain_mode: track\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: 100\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 1\nconsume: 1\nstate: pause\nxfade: 1\nOK\n"})
					},
					postWebSocket: []string{"/api/music"},
				},
				{
					method: http.MethodGet, path: "/api/music",
					want: map[int]string{http.StatusOK: `{"volume":100,"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"pause","song_elapsed":1.1,"replay_gain":"track","crossfade":1}`},
				},
			}},
		"GET /api/music/images": {
			config: APIConfig{skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodGet, path: "/api/music/images",
					want: map[int]string{http.StatusOK: `{"updating":false}`},
				},
			},
		},
		"GET /api/music/storage unknown command": {
			config: APIConfig{skipInit: true},
			tests: []*testRequest{
				{
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: mount\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "listmounts\n", Write: "ACK [5@0] {} unknown command \"listmounts\"\nOK\n"})
					},
					preWebSocket: []string{"/api/music/storage"},
					method:       http.MethodGet, path: "/api/music/storage",
					want: map[int]string{http.StatusOK: `{}`},
				},
			},
		},
		"GET /api/music/storage": {
			config: APIConfig{skipInit: true},
			tests: []*testRequest{
				{
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: mount\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "listmounts\n", Write: "mount: \nstorage: /home/foo/music\nmount: foo\nstorage: nfs://192.168.1.4/export/mp3\nOK\n"})
					},
					preWebSocket: []string{"/api/music/storage"},
					method:       http.MethodGet, path: "/api/music/storage",
					want: map[int]string{http.StatusOK: `{"":{"uri":"/home/foo/music"},"foo":{"uri":"nfs://192.168.1.4/export/mp3"}}`},
				},
			},
		},
		`POST /api/music/storage {"foo":{"uri":"nfs://192.168.1.4/export/mp3"}}`: {
			config: APIConfig{skipInit: true},
			tests: []*testRequest{
				{
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "mount \"foo\" \"nfs://192.168.1.4/export/mp3\"\n", Write: "OK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "update foo\n", Write: "OK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: mount\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "listmounts\n", Write: "mount: \nstorage: /home/foo/music\nmount: foo\nstorage: nfs://192.168.1.4/export/mp3\nOK\n"})
					},
					method: http.MethodPost, path: "/api/music/storage", body: strings.NewReader(`{"foo":{"uri":"nfs://192.168.1.4/export/mp3"}}`),
					want: map[int]string{
						http.StatusAccepted: "",
						http.StatusOK:       `{"":{"uri":"/home/foo/music"},"foo":{"uri":"nfs://192.168.1.4/export/mp3"}}`,
					},
					postWebSocket: []string{"/api/music/storage"},
				},
			},
		},
		`POST /api/music/storage {"foo":{"uri":null}}`: {
			config: APIConfig{skipInit: true},
			tests: []*testRequest{
				{
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "unmount \"foo\"\n", Write: "OK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "update \n", Write: "OK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: mount\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "listmounts\n", Write: "mount: \nstorage: /home/foo/music\nOK\n"})
					},
					method: http.MethodPost, path: "/api/music/storage", body: strings.NewReader(`{"foo":{"uri":null}}`),
					want: map[int]string{
						http.StatusAccepted: "",
						http.StatusOK:       `{"":{"uri":"/home/foo/music"}}`,
					},
					postWebSocket: []string{"/api/music/storage"},
				},
			},
		},
		`POST /api/music/storage {"foo":{"updating":true}}`: {
			config: APIConfig{skipInit: true},
			tests: []*testRequest{
				{
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "update foo\n", Write: "OK\n"})
					},
					method: http.MethodPost, path: "/api/music/storage", body: strings.NewReader(`{"foo":{"updating":true}}`),
					want: map[int]string{http.StatusAccepted: ""},
				},
			},
		},
		"GET /api/music/outputs": {
			config: APIConfig{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: output\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "outputs\n", Write: "outputid: 1\noutputname: My ALSA Device\noutputenabled: 1\nOK\n"})
					},
					preWebSocket: []string{"/api/music/outputs"},
					method:       http.MethodGet, path: "/api/music/outputs",
					want: map[int]string{http.StatusOK: `{"1":{"name":"My ALSA Device","enabled":true}}`},
				},
			},
		},
		"GET /api/music/outputs dop": {
			config: APIConfig{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: output\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "outputs\n", Write: "outputid: 1\noutputname: My ALSA Device\noutputenabled: 1\nattribute: allowed_formats=\nattribute: dop=0\nOK\n"})
					},
					preWebSocket: []string{"/api/music/outputs"},
					method:       http.MethodGet, path: "/api/music/outputs",
					want: map[int]string{http.StatusOK: `{"1":{"name":"My ALSA Device","enabled":true,"attributes":{"dop":false,"allowed_formats":[]}}}`},
				},
			},
		},
		"GET /api/music/outputs with stream url": {
			config: APIConfig{BackgroundTimeout: time.Second, skipInit: true, AudioProxy: map[string]string{"My HTTP Stream": "http://localhost:8080/"}},
			tests: []*testRequest{
				{
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: output\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "outputs\n", Write: "outputid: 1\noutputname: My HTTP Stream\noutputenabled: 1\nOK\n"})
					},
					preWebSocket: []string{"/api/music/outputs"},
					method:       http.MethodGet, path: "/api/music/outputs",
					want: map[int]string{http.StatusOK: `{"1":{"name":"My HTTP Stream","enabled":true,"stream":"/api/music/outputs/My HTTP Stream"}}`},
				},
			},
		},
		"POST /api/music/outputs {invalid json}": {
			config: APIConfig{BackgroundTimeout: time.Second, skipInit: true, AudioProxy: map[string]string{"My HTTP Stream": "http://localhost:8080/"}},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music/outputs", body: strings.NewReader(`{invalid json}`),
					want: map[int]string{http.StatusBadRequest: `{"error":"invalid character 'i' looking for beginning of object key string"}`},
				},
			}},
		`POST /api/music/outputs {"0":{"enabled":true}}`: {
			config: APIConfig{BackgroundTimeout: time.Second, skipInit: true, AudioProxy: map[string]string{"My HTTP Stream": "http://localhost:8080/"}},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music/outputs", body: strings.NewReader(`{"0":{"enabled":true}}`),
					want: map[int]string{
						http.StatusAccepted: "",
						http.StatusOK:       `{"0":{"name":"My ALSA Device","enabled":true}}`,
					},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "enableoutput 0\n", Write: "OK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: output\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "outputs\n", Write: "outputid: 0\noutputname: My ALSA Device\noutputenabled: 1\nOK\n"})
					},
					postWebSocket: []string{"/api/music/outputs"},
				},
				{
					method: http.MethodGet, path: "/api/music/outputs",
					want: map[int]string{http.StatusOK: `{"0":{"name":"My ALSA Device","enabled":true}}`},
				},
			}},
		`POST /api/music/outputs {"0":{"enabled":false}}`: {
			config: APIConfig{BackgroundTimeout: time.Second, skipInit: true, AudioProxy: map[string]string{"My HTTP Stream": "http://localhost:8080/"}},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music/outputs", body: strings.NewReader(`{"0":{"enabled":false}}`),
					want: map[int]string{
						http.StatusAccepted: "",
						http.StatusOK:       `{"0":{"name":"My ALSA Device","enabled":false}}`,
					},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "disableoutput 0\n", Write: "OK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: output\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "outputs\n", Write: "outputid: 0\noutputname: My ALSA Device\noutputenabled: 0\nOK\n"})
					},
					postWebSocket: []string{"/api/music/outputs"},
				},
				{
					method: http.MethodGet, path: "/api/music/outputs",
					want: map[int]string{http.StatusOK: `{"0":{"name":"My ALSA Device","enabled":false}}`},
				},
			}},
		`POST /api/music/outputs {"0":{"attributes":{"dop":true}}}`: {
			config: APIConfig{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music/outputs", body: strings.NewReader(`{"0":{"attributes":{"dop":true}}}`),
					want: map[int]string{
						http.StatusAccepted: "",
						http.StatusOK:       `{"0":{"name":"My ALSA Device","plugin":"alsa","enabled":false,"attributes":{"dop":true}}}`,
					},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "outputset \"0\" \"dop\" \"1\"\n", Write: "OK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: output\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "outputs\n", Write: "outputid: 0\noutputname: My ALSA Device\noutputenabled: 0\nplugin: alsa\nattribute: dop=1\nOK\n"})
					},
					postWebSocket: []string{"/api/music/outputs"},
				},
				{
					method: http.MethodGet, path: "/api/music/outputs",
					want: map[int]string{http.StatusOK: `{"0":{"name":"My ALSA Device","plugin":"alsa","enabled":false,"attributes":{"dop":true}}}`},
				},
			}},
		`POST /api/music/outputs {"0":{"attributes":{"allowed_formats":[]}}}`: {
			config: APIConfig{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music/outputs", body: strings.NewReader(`{"0":{"attributes":{"allowed_formats":[]}}}`),
					want: map[int]string{
						http.StatusAccepted: "",
						http.StatusOK:       `{"0":{"name":"My ALSA Device","plugin":"alsa","enabled":false,"attributes":{"allowed_formats":[]}}}`,
					},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "outputset \"0\" \"allowed_formats\" \"\"\n", Write: "OK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: output\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "outputs\n", Write: "outputid: 0\noutputname: My ALSA Device\noutputenabled: 0\nplugin: alsa\nattribute: allowed_formats=\nOK\n"})
					},
					postWebSocket: []string{"/api/music/outputs"},
				},
				{
					method: http.MethodGet, path: "/api/music/outputs",
					want: map[int]string{http.StatusOK: `{"0":{"name":"My ALSA Device","plugin":"alsa","enabled":false,"attributes":{"allowed_formats":[]}}}`},
				},
			}},
		`POST /api/music/outputs {"0":{"attributes":{"allowed_formats":["96000:16:*","192000:24:*","dsd32:*=dop"]}}}`: {
			config: APIConfig{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music/outputs", body: strings.NewReader(`{"0":{"attributes":{"allowed_formats":["96000:16:*","192000:24:*","dsd32:*=dop"]}}}`),
					want: map[int]string{
						http.StatusAccepted: "",
						http.StatusOK:       `{"0":{"name":"My ALSA Device","plugin":"alsa","enabled":false,"attributes":{"allowed_formats":["96000:16:*","192000:24:*","dsd32:*=dop"]}}}`,
					},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "outputset \"0\" \"allowed_formats\" \"96000:16:* 192000:24:* dsd32:*=dop\"\n", Write: "OK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: output\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "outputs\n", Write: "outputid: 0\noutputname: My ALSA Device\noutputenabled: 0\nplugin: alsa\nattribute: allowed_formats=96000:16:* 192000:24:* dsd32:*=dop\nOK\n"})
					},
					postWebSocket: []string{"/api/music/outputs"},
				},
				{
					method: http.MethodGet, path: "/api/music/outputs",
					want: map[int]string{http.StatusOK: `{"0":{"name":"My ALSA Device","plugin":"alsa","enabled":false,"attributes":{"allowed_formats":["96000:16:*","192000:24:*","dsd32:*=dop"]}}}`},
				},
			}},
		`POST /api/music/playlist {invalid json}`: {
			config: APIConfig{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music/playlist", body: strings.NewReader(`{invalid json}`),
					want: map[int]string{http.StatusBadRequest: `{"error":"invalid character 'i' looking for beginning of object key string"}`},
				},
			}},
		`POST /api/music/playlist {}`: {
			config: APIConfig{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music/playlist", body: strings.NewReader(`{}`),
					want: map[int]string{http.StatusBadRequest: `{"error":"filters and sort fields are required"}`},
				},
			}},
		`POST /api/music/playlist {"current":0,"sort":["file"],"filters":[]}`: {
			config: APIConfig{BackgroundTimeout: time.Second},
			initFunc: func(ctx context.Context, main *mpdtest.Server) {
				main.Expect(ctx, &mpdtest.WR{Read: "listallinfo /\n", Write: "file: foo\nfile: bar\nfile: baz\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "playlistinfo\n", Write: "file: foo\nfile: bar\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "replay_gain_status\n", Write: "replay_gain_mode: off\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "currentsong\n", Write: "file: bar\nPos: 1\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "outputs\n", Write: "outputid: 0\noutputname: My ALSA Device\noutputenabled: 0\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "listmounts\n", Write: "mount: \nstorage: /home/foo/music\nmount: foo\nstorage: nfs://192.168.1.4/export/mp3\nOK\n"})
			},
			tests: []*testRequest{
				{ // update playlist and current song
					method: http.MethodPost, path: "/api/music/playlist", body: strings.NewReader(`{"current":0,"sort":["file"],"filters":[]}`),
					want: map[int]string{http.StatusAccepted: `{"current":1}`},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
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
					postWebSocket: []string{"/api/music/playlist/songs", "/api/music", "/api/music/playlist", "/api/music/playlist/songs/current", "/api/music/stats"},
				},
				{
					method: http.MethodGet, path: "/api/music/playlist",
					want: map[int]string{http.StatusOK: `{"current":0,"sort":["file"]}`},
				},
				{ // update current song only
					method: http.MethodPost, path: "/api/music/playlist", body: strings.NewReader(`{"current":1,"sort":["file"],"filters":[]}`),
					want: map[int]string{
						http.StatusAccepted: `{"current":0,"sort":["file"]}`,
						http.StatusOK:       `{"current":1,"sort":["file"]}`,
					},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "play 1\n", Write: "OK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: player\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "currentsong\n", Write: "file: bar\nPos: 1\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
					},
					postWebSocket: []string{"/api/music", "/api/music/playlist", "/api/music/playlist/songs/current", "/api/music/stats"},
				},
				{
					method: http.MethodGet, path: "/api/music/playlist",
					want: map[int]string{http.StatusOK: `{"current":1,"sort":["file"]}`},
				},
				{ // changed playlist removes sort info
					method: http.MethodGet, path: "/api/music/playlist",
					want: map[int]string{http.StatusOK: `{"current":1}`},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: playlist\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "playlistinfo\n", Write: "file: foo\nfile: bar\nOK\n"})
					},
					preWebSocket: []string{"/api/music/playlist/songs", "/api/music/playlist"},
				},
			}},

		`POST /api/music/library {invalid json}`: {
			config: APIConfig{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music/library", body: strings.NewReader(`{invalid json}`),
					want: map[int]string{http.StatusBadRequest: `{"error":"invalid character 'i' looking for beginning of object key string"}`},
				},
			}},

		`POST /api/music/library {}`: {
			config: APIConfig{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music/library", body: strings.NewReader(`{}`),
					want: map[int]string{http.StatusBadRequest: `{"error":"requires updating=true"}`},
				},
			}},

		`POST /api/music/library {"updating":true} error`: {
			config: APIConfig{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music/library", body: strings.NewReader(`{"updating":true}`),
					want: map[int]string{http.StatusInternalServerError: `{"error":"mpd: update: error"}`},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "update \n", Write: "ACK [2@1] {update} error\n"})
					},
				},
			}},

		`POST /api/music/library {"updating":true}`: {
			config: APIConfig{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music/library", body: strings.NewReader(`{"updating":true}`),
					want: map[int]string{
						http.StatusAccepted: ``,
						http.StatusOK:       `{"updating":true}`,
					},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "update \n", Write: "updating_db: 1\nOK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: update\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nupdating_db: 1\nOK\n"})
					},
					postWebSocket: []string{"/api/music", "/api/music/playlist", "/api/music/library"},
				},
				{
					method: http.MethodGet, path: "/api/music/library",
					want: map[int]string{http.StatusOK: `{"updating":true}`},
				},

				{
					preWebSocket: []string{"/api/music", "/api/music/library", "/api/music/library/songs", "/api/music", "/api/music/stats"},
					method:       http.MethodGet, path: "/api/music/library",
					want: map[int]string{http.StatusOK: `{"updating":false}`},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: update\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: database\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "listallinfo /\n", Write: "file: foo\nfile: bar\nfile: baz\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
					},
				},
			}},
	} {
		t.Run(label, func(t *testing.T) {
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

			if tt.initFunc != nil {
				go tt.initFunc(ctx, main)
			}
			b := cover.NewBatch([]cover.Cover{})
			h, err := tt.config.NewAPIHandler(ctx, c, wl, b)
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
			timeout, _ := ctx.Deadline()
			ws.SetReadDeadline(timeout)
			if _, msg, err := ws.ReadMessage(); string(msg) != "ok" || err != nil {
				t.Fatalf("got message: %s, %v, want: ok <nil>", msg, err)
			}
			for i, test := range tt.tests {
				t.Run(fmt.Sprintf("#%d", i), func(t *testing.T) {
					if test.initFunc != nil {
						go test.initFunc(ctx, main, sub)
					}
					if len(test.preWebSocket) != 0 {
						got := make([]string, 0, 10)
						for {
							_, msg, err := ws.ReadMessage()
							if err != nil {
								t.Errorf("failed to get preWebSocket message: %v, got: %v, want: %v", err, got, test.preWebSocket)
								break
							}
							got = append(got, string(msg))
							if len(got) == len(test.preWebSocket) && !reflect.DeepEqual(got, test.preWebSocket) {
								t.Errorf("got %v; want %v", got, test.preWebSocket)
								break
							} else if reflect.DeepEqual(got, test.preWebSocket) {
								break
							}
						}
					}
					req, err := http.NewRequest(test.method, ts.URL+test.path, test.body)
					if err != nil {
						t.Fatalf("failed to create reuqest: %v", err)
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
					want, ok := test.want[resp.StatusCode]
					if !ok {
						t.Errorf("got %d %s, want %v", resp.StatusCode, b, test.want)
					} else if got := string(b); got != want {
						t.Errorf("got %d %v; want %d %v", resp.StatusCode, got, resp.StatusCode, want)
					}

					if len(test.postWebSocket) != 0 {
						got := make([]string, 0, 10)
						for {
							_, msg, err := ws.ReadMessage()
							if err != nil {
								t.Errorf("failed to get postWebSocket message: %v, got: %v, want: %v", err, got, test.postWebSocket)
								break
							}
							got = append(got, string(msg))
							if len(got) == len(test.postWebSocket) && !reflect.DeepEqual(got, test.postWebSocket) {
								t.Errorf("got %v; want %v", got, test.postWebSocket)
								break
							} else if reflect.DeepEqual(got, test.postWebSocket) {
								break
							}
						}
					}
				})
			}
			if err := b.Shutdown(ctx); err != nil {
				t.Errorf("cover.Batch.Shutdown got err %v; want nil", err)
			}
			go func() {
				sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: ""})
				sub.Expect(ctx, &mpdtest.WR{Read: "noidle\n", Write: "OK\n"})
			}()
		})
	}
}
