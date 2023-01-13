package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/meiraka/vv/internal/mpd"
	"github.com/meiraka/vv/internal/mpd/mpdtest"
)

const (
	testTimeout = 10 * time.Second
)

var (
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

func TestHandler(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	for label, tt := range map[string]struct {
		config   Config
		initFunc func(ctx context.Context, main *mpdtest.Server)
		tests    []*testRequest
	}{
		"init": {
			config: Config{BackgroundTimeout: time.Second, AudioProxy: map[string]string{"My HTTP Stream": "http://foo/bar"}},
			initFunc: func(ctx context.Context, main *mpdtest.Server) {
				main.Expect(ctx, &mpdtest.WR{Read: "listallinfo \"/\"\n", Write: "file: foo\nfile: bar\nfile: baz\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "playlistinfo\n", Write: "file: foo\nfile: bar\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "replay_gain_status\n", Write: "replay_gain_mode: off\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "currentsong\n", Write: "file: bar\nPos: 1\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "outputs\n", Write: "outputid: 0\noutputname: My HTTP Stream\noutputenabled: 0\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "listmounts\n", Write: "mount: \nstorage: /home/foo/music\nmount: foo\nstorage: nfs://192.168.1.4/export/mp3\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "listneighbors\n", Write: "neighbor: smb://FOO\nname: FOO (Samba 4.1.11-Debian)\nOK\n"})
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
					want: map[int]string{http.StatusOK: `{"0":{"name":"My HTTP Stream","enabled":false,"stream":"/api/music/outputs/stream?name=My+HTTP+Stream"}}`},
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
			config: Config{AppVersion: "0.0.0", BackgroundTimeout: time.Second, skipInit: true},

			tests: []*testRequest{
				{
					method: http.MethodGet, path: "/api/version",
					want: map[int]string{http.StatusOK: fmt.Sprintf(`{"app":"%s","go":"%s","mpd":"0.19"}`, "0.0.0", fmt.Sprintf("%s %s %s", runtime.Version(), runtime.GOOS, runtime.GOARCH))},
				},
				{
					method: http.MethodGet, path: "/api/music/playlist",
					want: map[int]string{http.StatusOK: "{}"},
				},
				{
					method: http.MethodGet, path: "/api/music/playlist/songs",
					want: map[int]string{http.StatusOK: "[]"},
				},
				{
					method: http.MethodGet, path: "/api/music/library",
					want: map[int]string{http.StatusOK: `{"updating":false}`},
				},
				{
					method: http.MethodGet, path: "/api/music/library/songs",
					want: map[int]string{http.StatusOK: "[]"},
				},
				{
					method: http.MethodGet, path: "/api/music/playlist/songs/current",
					want: map[int]string{http.StatusOK: "{}"},
				},
				{
					method: http.MethodGet, path: "/api/music/outputs",
					want: map[int]string{http.StatusOK: "{}"},
				},
				{
					method: http.MethodGet, path: "/api/music/stats",
					want: map[int]string{http.StatusOK: `{"uptime":0,"playtime":0,"artists":0,"albums":0,"songs":0,"library_playtime":0,"library_update":0}`},
				},
				{
					method: http.MethodGet, path: "/api/music/storage",
					want: map[int]string{http.StatusOK: "{}"},
				},
				{
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n"})
						sub.Disconnect(ctx)
						main.Expect(ctx, &mpdtest.WR{Read: "listallinfo \"/\"\n", Write: "file: foo\nfile: bar\nfile: baz\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "playlistinfo\n", Write: "file: foo\nfile: bar\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "replay_gain_status\n", Write: "replay_gain_mode: off\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "currentsong\n", Write: "file: bar\nPos: 1\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "outputs\n", Write: "outputid: 0\noutputname: My ALSA Device\noutputenabled: 0\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "listmounts\n", Write: "mount: \nstorage: /home/foo/music\nmount: foo\nstorage: nfs://192.168.1.4/export/mp3\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "listneighbors\n", Write: "neighbor: smb://FOO\nname: FOO (Samba 4.1.11-Debian)\nOK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n"})
					},
					// preWebSocket: []string{"/api/version", "/api/version", "/api/music/library/songs", "/api/music/playlist", "/api/music/playlist/songs", "/api/music", "/api/music/playlist", "/api/music/library", "/api/music/playlist/songs/current", "/api/music/outputs", "/api/music/stats", "/api/music/storage"},
					preWebSocket: []string{"/api/version", "/api/version", "/api/music/library/songs", "/api/music/playlist/songs", "/api/music", "/api/music/playlist/songs/current", "/api/music/outputs", "/api/music/playlist", "/api/music/stats", "/api/music/storage", "/api/music/storage/neighbors"},
					method:       http.MethodGet, path: "/api/music",
					want: map[int]string{http.StatusOK: `{"repeat":false,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`},
				},
				{
					method: http.MethodGet, path: "/api/version",
					want: map[int]string{http.StatusOK: fmt.Sprintf(`{"app":"%s","go":"%s","mpd":"0.19"}`, "0.0.0", fmt.Sprintf("%s %s %s", runtime.Version(), runtime.GOOS, runtime.GOARCH))},
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
			config: Config{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music", body: strings.NewReader(`{"repeat":true}`),
					want: map[int]string{
						http.StatusAccepted: "{}",
						http.StatusOK:       `{"repeat":true,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
					},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "repeat 1\n", Write: "OK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: options\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "replay_gain_status\n", Write: "replay_gain_mode: off\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
					},
					postWebSocket: []string{"/api/music", "/api/music/playlist"},
				},
				{
					method: http.MethodGet, path: "/api/music",
					want: map[int]string{http.StatusOK: `{"repeat":true,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`},
				},
			}},
		`POST /api/music {"random":true}`: {
			config: Config{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music", body: strings.NewReader(`{"random":true}`),
					want: map[int]string{
						http.StatusAccepted: "{}",
						http.StatusOK:       `{"repeat":true,"random":true,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
					},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "random 1\n", Write: "OK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: options\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "replay_gain_status\n", Write: "replay_gain_mode: off\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
					},
					postWebSocket: []string{"/api/music", "/api/music/playlist"},
				},
				{
					method: http.MethodGet, path: "/api/music",
					want: map[int]string{http.StatusOK: `{"repeat":true,"random":true,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`},
				},
			}},
		`POST /api/music {"oneshot":true}`: {
			config: Config{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music", body: strings.NewReader(`{"oneshot":true}`),
					want: map[int]string{
						http.StatusAccepted: "{}",
						http.StatusOK:       `{"repeat":true,"random":true,"single":false,"oneshot":true,"consume":false,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
					},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "single \"oneshot\"\n", Write: "OK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: options\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "replay_gain_status\n", Write: "replay_gain_mode: off\nOK\n"})

						main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: oneshot\nconsume: 0\nstate: pause\nOK\n"})
					},
					postWebSocket: []string{"/api/music", "/api/music/playlist"},
				},
				{
					method: http.MethodGet, path: "/api/music",
					want: map[int]string{http.StatusOK: `{"repeat":true,"random":true,"single":false,"oneshot":true,"consume":false,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`},
				},
			}},
		`POST /api/music {invalid json}`: {
			config: Config{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music", body: strings.NewReader(`{invalid json}`),
					want: map[int]string{http.StatusBadRequest: `{"error":"invalid character 'i' looking for beginning of object key string"}`},
				}}},
		`POST /api/music {"single":true}`: {
			config: Config{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music", body: strings.NewReader(`{"single":true}`),
					want: map[int]string{
						http.StatusAccepted: "{}",
						http.StatusOK:       `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
					},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "single 1\n", Write: "OK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: options\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "replay_gain_status\n", Write: "replay_gain_mode: off\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 1\nconsume: 0\nstate: pause\nOK\n"})
					},
					postWebSocket: []string{"/api/music", "/api/music/playlist"},
				},
				{
					method: http.MethodGet, path: "/api/music",
					want: map[int]string{http.StatusOK: `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`},
				},
			}},
		`POST /api/music {"consume":true}`: {
			config: Config{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music", body: strings.NewReader(`{"consume":true}`),
					want: map[int]string{
						http.StatusAccepted: "{}",
						http.StatusOK:       `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
					},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "consume 1\n", Write: "OK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: options\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "replay_gain_status\n", Write: "replay_gain_mode: off\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 1\nconsume: 1\nstate: pause\nOK\n"})
					},
					postWebSocket: []string{"/api/music", "/api/music/playlist"},
				},
				{
					method: http.MethodGet, path: "/api/music",
					want: map[int]string{http.StatusOK: `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`},
				},
			}},
		`POST /api/music {"state":"unknown"}`: {
			config: Config{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music", body: strings.NewReader(`{"state":"unknown"}`),
					want: map[int]string{http.StatusBadRequest: `{"error":"unknown state: unknown"}`},
				},
			}},
		`POST /api/music {"state":"play"}`: {
			config: Config{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music", body: strings.NewReader(`{"state":"play"}`),
					want: map[int]string{
						http.StatusAccepted: "{}",
						http.StatusOK:       `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"play","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
					},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "play -1\n", Write: "OK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: player\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 1\nconsume: 1\nstate: play\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "currentsong\n", Write: "file: bar\nPos: 1\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
					},
					postWebSocket: []string{"/api/music", "/api/music/playlist", "/api/music/playlist/songs/current", "/api/music/stats"},
				},
				{
					method: http.MethodGet, path: "/api/music",
					want: map[int]string{http.StatusOK: `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"play","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`},
				},
			}},
		`POST /api/music {"state":"next"}`: {
			config: Config{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music", body: strings.NewReader(`{"state":"next"}`),
					want: map[int]string{
						http.StatusAccepted: "{}",
						http.StatusOK:       `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"play","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
					},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "next\n", Write: "OK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: player\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 2\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 1\nconsume: 1\nstate: play\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "currentsong\n", Write: "file: baz\nPos: 2\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
					},
					postWebSocket: []string{"/api/music", "/api/music/playlist", "/api/music/playlist/songs/current", "/api/music/stats"},
				},
				{
					method: http.MethodGet, path: "/api/music",
					want: map[int]string{http.StatusOK: `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"play","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`},
				},
			}},
		`POST /api/music {"state":"previous"}`: {
			config: Config{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music", body: strings.NewReader(`{"state":"previous"}`),
					want: map[int]string{
						http.StatusAccepted: "{}",
						http.StatusOK:       `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"play","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
					},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "previous\n", Write: "OK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: player\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 1\nconsume: 1\nstate: play\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "currentsong\n", Write: "file: bar\nPos: 1\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
					},
					postWebSocket: []string{"/api/music", "/api/music/playlist", "/api/music/playlist/songs/current", "/api/music/stats"},
				},
				{
					method: http.MethodGet, path: "/api/music",
					want: map[int]string{http.StatusOK: `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"play","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`},
				},
			}},
		`POST /api/music {"state":"pause"}`: {
			config: Config{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music", body: strings.NewReader(`{"state":"pause"}`),
					want: map[int]string{
						http.StatusAccepted: "{}",
						http.StatusOK:       `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
					},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "pause 1\n", Write: "OK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: player\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 1\nsingle: 1\nconsume: 1\nstate: pause\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "currentsong\n", Write: "file: bar\nPos: 1\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
					},
					postWebSocket: []string{"/api/music", "/api/music/playlist", "/api/music/playlist/songs/current", "/api/music/stats"},
				},
				{
					method: http.MethodGet, path: "/api/music",
					want: map[int]string{http.StatusOK: `{"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`},
				},
			}},
		`POST /api/music {"volume":"100"}`: {
			config: Config{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music", body: strings.NewReader(`{"volume":100}`),
					want: map[int]string{
						http.StatusAccepted: "{}",
						http.StatusOK:       `{"volume":100,"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`,
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
					want: map[int]string{http.StatusOK: `{"volume":100,"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`},
				},
			}},
		`POST /api/music {"song_elapsed":100.1}`: {
			config: Config{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music", body: strings.NewReader(`{"song_elapsed":100.1}`),
					want: map[int]string{
						http.StatusAccepted: "{}",
						http.StatusOK:       `{"volume":100,"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"pause","song_elapsed":100.1,"replay_gain":"off","crossfade":0}`,
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
					want: map[int]string{http.StatusOK: `{"volume":100,"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"pause","song_elapsed":1.1,"replay_gain":"off","crossfade":0}`},
				},
			}},
		`POST /api/music {"replay_gain":"track"}`: {
			config: Config{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music", body: strings.NewReader(`{"replay_gain":"track"}`),
					want: map[int]string{
						http.StatusAccepted: "{}",
						http.StatusOK:       `{"volume":100,"repeat":true,"random":true,"single":true,"oneshot":false,"consume":true,"state":"pause","song_elapsed":1.1,"replay_gain":"track","crossfade":0}`,
					},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "replay_gain_mode \"track\"\n", Write: "OK\n"})
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
			config: Config{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music", body: strings.NewReader(`{"crossfade":1}`),
					want: map[int]string{
						http.StatusAccepted: "{}",
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
			config: Config{skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodGet, path: "/api/music/images",
					want: map[int]string{http.StatusOK: `{"updating":false}`},
				},
			},
		},
		"GET /api/music/storage unknown command": {
			config: Config{skipInit: true},
			tests: []*testRequest{
				{
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: mount\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "listmounts\n", Write: "ACK [5@0] {} unknown command \"listmounts\"\nOK\n"})
					},
					method: http.MethodGet, path: "/api/music/storage",
					want: map[int]string{http.StatusOK: `{}`},
				},
			},
		},
		"GET /api/music/storage": {
			config: Config{skipInit: true},
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
			config: Config{skipInit: true},
			tests: []*testRequest{
				{
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "mount \"foo\" \"nfs://192.168.1.4/export/mp3\"\n", Write: "OK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "update \"foo\"\n", Write: "OK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: mount\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "listmounts\n", Write: "mount: \nstorage: /home/foo/music\nmount: foo\nstorage: nfs://192.168.1.4/export/mp3\nOK\n"})
					},
					method: http.MethodPost, path: "/api/music/storage", body: strings.NewReader(`{"foo":{"uri":"nfs://192.168.1.4/export/mp3"}}`),
					want: map[int]string{
						http.StatusAccepted: "{}",
						http.StatusOK:       `{"":{"uri":"/home/foo/music"},"foo":{"uri":"nfs://192.168.1.4/export/mp3"}}`,
					},
					postWebSocket: []string{"/api/music/storage"},
				},
			},
		},
		`POST /api/music/storage {"foo":{"uri":null}}`: {
			config: Config{skipInit: true},
			tests: []*testRequest{
				{
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "unmount \"foo\"\n", Write: "OK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "update \"\"\n", Write: "OK\n"})
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: mount\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "listmounts\n", Write: "mount: \nstorage: /home/foo/music\nOK\n"})
					},
					method: http.MethodPost, path: "/api/music/storage", body: strings.NewReader(`{"foo":{"uri":null}}`),
					want: map[int]string{
						http.StatusAccepted: "{}",
						http.StatusOK:       `{"":{"uri":"/home/foo/music"}}`,
					},
					postWebSocket: []string{"/api/music/storage"},
				},
			},
		},
		`POST /api/music/storage {"foo":{"updating":true}}`: {
			config: Config{skipInit: true},
			tests: []*testRequest{
				{
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "update \"foo\"\n", Write: "OK\n"})
					},
					method: http.MethodPost, path: "/api/music/storage", body: strings.NewReader(`{"foo":{"updating":true}}`),
					want: map[int]string{http.StatusAccepted: "{}"},
				},
			},
		},
		"GET /api/music/outputs": {
			config: Config{BackgroundTimeout: time.Second, skipInit: true},
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
			config: Config{BackgroundTimeout: time.Second, skipInit: true},
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
			config: Config{BackgroundTimeout: time.Second, skipInit: true, AudioProxy: map[string]string{"My HTTP Stream": "http://localhost:8080/"}},
			tests: []*testRequest{
				{
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: output\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "outputs\n", Write: "outputid: 1\noutputname: My HTTP Stream\noutputenabled: 1\nOK\n"})
					},
					preWebSocket: []string{"/api/music/outputs"},
					method:       http.MethodGet, path: "/api/music/outputs",
					want: map[int]string{http.StatusOK: `{"1":{"name":"My HTTP Stream","enabled":true,"stream":"/api/music/outputs/stream?name=My+HTTP+Stream"}}`},
				},
			},
		},
		"POST /api/music/outputs {invalid json}": {
			config: Config{BackgroundTimeout: time.Second, skipInit: true, AudioProxy: map[string]string{"My HTTP Stream": "http://localhost:8080/"}},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music/outputs", body: strings.NewReader(`{invalid json}`),
					want: map[int]string{http.StatusBadRequest: `{"error":"invalid character 'i' looking for beginning of object key string"}`},
				},
			}},
		`POST /api/music/outputs {"0":{"enabled":true}}`: {
			config: Config{BackgroundTimeout: time.Second, skipInit: true, AudioProxy: map[string]string{"My HTTP Stream": "http://localhost:8080/"}},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music/outputs", body: strings.NewReader(`{"0":{"enabled":true}}`),
					want: map[int]string{
						http.StatusAccepted: "{}",
						http.StatusOK:       `{"0":{"name":"My ALSA Device","enabled":true}}`,
					},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "enableoutput \"0\"\n", Write: "OK\n"})
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
			config: Config{BackgroundTimeout: time.Second, skipInit: true, AudioProxy: map[string]string{"My HTTP Stream": "http://localhost:8080/"}},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music/outputs", body: strings.NewReader(`{"0":{"enabled":false}}`),
					want: map[int]string{
						http.StatusAccepted: "{}",
						http.StatusOK:       `{"0":{"name":"My ALSA Device","enabled":false}}`,
					},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "disableoutput \"0\"\n", Write: "OK\n"})
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
			config: Config{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music/outputs", body: strings.NewReader(`{"0":{"attributes":{"dop":true}}}`),
					want: map[int]string{
						http.StatusAccepted: "{}",
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
			config: Config{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music/outputs", body: strings.NewReader(`{"0":{"attributes":{"allowed_formats":[]}}}`),
					want: map[int]string{
						http.StatusAccepted: "{}",
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
			config: Config{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music/outputs", body: strings.NewReader(`{"0":{"attributes":{"allowed_formats":["96000:16:*","192000:24:*","dsd32:*=dop"]}}}`),
					want: map[int]string{
						http.StatusAccepted: "{}",
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
			config: Config{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music/playlist", body: strings.NewReader(`{invalid json}`),
					want: map[int]string{http.StatusBadRequest: `{"error":"invalid character 'i' looking for beginning of object key string"}`},
				},
			}},
		`POST /api/music/playlist {}`: {
			config: Config{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music/playlist", body: strings.NewReader(`{}`),
					want: map[int]string{http.StatusBadRequest: `{"error":"current, filters and sort fields are required"}`},
				},
			}},
		`POST /api/music/playlist {"current":0,"sort":["file"],"filters":[]}`: {
			config: Config{BackgroundTimeout: time.Second},
			initFunc: func(ctx context.Context, main *mpdtest.Server) {
				main.Expect(ctx, &mpdtest.WR{Read: "listallinfo \"/\"\n", Write: "file: foo\nfile: bar\nfile: baz\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "playlistinfo\n", Write: "file: foo\nfile: bar\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "replay_gain_status\n", Write: "replay_gain_mode: off\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "currentsong\n", Write: "file: bar\nPos: 1\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "outputs\n", Write: "outputid: 0\noutputname: My ALSA Device\noutputenabled: 0\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "listmounts\n", Write: "mount: \nstorage: /home/foo/music\nmount: foo\nstorage: nfs://192.168.1.4/export/mp3\nOK\n"})
				main.Expect(ctx, &mpdtest.WR{Read: "listneighbors\n", Write: "neighbor: smb://FOO\nname: FOO (Samba 4.1.11-Debian)\nOK\n"})
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
			config: Config{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music/library", body: strings.NewReader(`{invalid json}`),
					want: map[int]string{http.StatusBadRequest: `{"error":"invalid character 'i' looking for beginning of object key string"}`},
				},
			}},

		`POST /api/music/library {}`: {
			config: Config{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music/library", body: strings.NewReader(`{}`),
					want: map[int]string{http.StatusBadRequest: `{"error":"requires updating=true"}`},
				},
			}},

		`POST /api/music/library {"updating":true} error`: {
			config: Config{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music/library", body: strings.NewReader(`{"updating":true}`),
					want: map[int]string{http.StatusInternalServerError: `{"error":"mpd: update: error"}`},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "update \"\"\n", Write: "ACK [2@1] {update} error\n"})
					},
				},
			}},

		`POST /api/music/library {"updating":true}`: {
			config: Config{BackgroundTimeout: time.Second, skipInit: true},
			tests: []*testRequest{
				{
					method: http.MethodPost, path: "/api/music/library", body: strings.NewReader(`{"updating":true}`),
					want: map[int]string{
						http.StatusAccepted: `{"updating":false}`,
						http.StatusOK:       `{"updating":true}`,
					},
					initFunc: func(ctx context.Context, main *mpdtest.Server, sub *mpdtest.Server) {
						main.Expect(ctx, &mpdtest.WR{Read: "update \"\"\n", Write: "updating_db: 1\nOK\n"})
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
						main.Expect(ctx, &mpdtest.WR{Read: "listallinfo \"/\"\n", Write: "file: foo\nfile: bar\nfile: baz\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"})
						main.Expect(ctx, &mpdtest.WR{Read: "stats\n", Write: "uptime: 667505\nplaytime: 0\nartists: 835\nalbums: 528\nsongs: 5715\ndb_playtime: 1475220\ndb_update: 1560656023\nOK\n"})
					},
				},
			}},
	} {
		select {
		case <-ctx.Done():
			t.Fatal("test exceeds timeout")
		default:
		}
		tt := tt
		t.Run(label, func(t *testing.T) {
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			main := mpdtest.NewServer("OK MPD 0.19")
			defer main.Close()
			sub := mpdtest.NewServer("OK MPD 0.19")
			defer sub.Close()
			c, err := mpd.Dial("tcp", main.URL,
				&mpd.ClientOptions{Timeout: testTimeout, ReconnectionInterval: time.Millisecond})
			if err != nil {
				t.Fatalf("Dial got error %v; want nil", err)
			}
			defer func() {
				if err := c.Close(ctx); err != nil {
					t.Errorf("mpd.Client.Close got err %v; want nil", err)
				}
			}()
			wl, err := mpd.NewWatcher("tcp", sub.URL,
				&mpd.WatcherOptions{Timeout: testTimeout, ReconnectionInterval: time.Millisecond})
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
			h, err := NewHandler(ctx, c, wl, &tt.config)
			if err != nil {
				t.Fatalf("NewHTTPHandler got error = %v; want <nil>", err)
			}
			defer h.Stop()
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
						want := sortUniq(test.preWebSocket)
						got := make([]string, 0, 10)
						for {
							_, msg, err := ws.ReadMessage()
							if err != nil {
								t.Errorf("failed to get preWebSocket message: %v, got:\n%v, want:\n%v", err, got, want)
								break
							}
							got = sortUniq(append(got, string(msg)))
							if len(got) == len(want) && !reflect.DeepEqual(got, want) {
								t.Errorf("got\n%v; want\n%v", got, want)
								break
							} else if reflect.DeepEqual(got, want) {
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
					b, err := io.ReadAll(resp.Body)
					if err != nil {
						t.Fatalf("failed to read response: %v", err)
					}
					want, ok := test.want[resp.StatusCode]
					if !ok {
						t.Errorf("%s %s =\n%d %s, want\n%v", test.method, test.path, resp.StatusCode, b, test.want)
					} else if got := string(b); got != want {
						t.Errorf("%s %s =\n%d %v; want\n%d %v", test.method, test.path, resp.StatusCode, got, resp.StatusCode, want)
					}

					if len(test.postWebSocket) != 0 {
						want := sortUniq(test.postWebSocket)
						got := make([]string, 0, 10)
						for {
							_, msg, err := ws.ReadMessage()
							if err != nil {
								t.Errorf("failed to get postWebSocket message: %v, got: %v, want: %v", err, got, want)
								break
							}
							got = sortUniq(append(got, string(msg)))
							if len(got) == len(want) && !reflect.DeepEqual(got, want) {
								t.Errorf("got %v; want %v", got, want)
								break
							} else if reflect.DeepEqual(got, want) {
								break
							}
						}
					}
				})
			}
			if err := h.Shutdown(ctx); err != nil {
				t.Errorf("Handler.Shutdown got err %v; want nil", err)
			}
			go func() {
				sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: ""})
				sub.Expect(ctx, &mpdtest.WR{Read: "noidle\n", Write: "OK\n"})
			}()
		})
	}
}

func TestAPIOutputStreamHandler(t *testing.T) {
	// setup
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	main := mpdtest.NewServer("OK MPD 0.19")
	defer main.Close()
	sub := mpdtest.NewServer("OK MPD 0.19")
	defer sub.Close()
	c, err := mpd.Dial("tcp", main.URL,
		&mpd.ClientOptions{Timeout: testTimeout, ReconnectionInterval: time.Millisecond})
	if err != nil {
		t.Fatalf("Dial got error %v; want nil", err)
	}
	defer func() {
		if err := c.Close(ctx); err != nil {
			t.Errorf("mpd.Client.Close got err %v; want nil", err)
		}
	}()
	wl, err := mpd.NewWatcher("tcp", sub.URL,
		&mpd.WatcherOptions{Timeout: testTimeout, ReconnectionInterval: time.Millisecond})
	if err != nil {
		t.Fatalf("Dial got error %v; want nil", err)
	}
	defer func() {
		if err := wl.Close(ctx); err != nil {
			t.Errorf("mpd.Watcher.Close got err %v; want nil", err)
		}
	}()
	// write 0 until context.Done() or client closes connection.
	audioProxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Transfer-Encoding", "chunked")
		w.WriteHeader(http.StatusOK)
		for {
			select {
			case <-ctx.Done():
			default:
				_, err := w.Write([]byte("0"))
				if err != nil {
					return
				}
			}
		}
	}))
	defer audioProxy.Close()
	h, err := NewHandler(ctx, c, wl, &Config{
		skipInit:   true,
		AudioProxy: map[string]string{"My / HTTP / Stream": audioProxy.URL},
	})
	if err != nil {
		t.Fatalf("failed to initialize api handler: %v", err)
	}
	defer h.Stop()
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
	// send mpd output event
	sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: output\nOK\n"})
	main.Expect(ctx, &mpdtest.WR{Read: "outputs\n", Write: "outputid: 1\noutputname: My / HTTP / Stream\noutputenabled: 1\nOK\n"})
	// wait for cache update
	if _, msg, err := ws.ReadMessage(); string(msg) != "/api/music/outputs" || err != nil {
		t.Fatalf("got message: %s, %v, want: ok <nil>", msg, err)
	}
	resp, err := http.Get(ts.URL + "/api/music/outputs")
	if err != nil {
		t.Fatalf("failed to get outputs list")
	}
	outputs := map[string]struct {
		Stream string `json:"stream"`
	}{}
	err = json.NewDecoder(resp.Body).Decode(&outputs)
	resp.Body.Close()
	if err != nil {
		t.Fatalf("failed to parse outputs list: %v", err)
	}
	device, ok := outputs["1"]
	if !ok {
		t.Fatalf("failed to parse outputs list: no outputid 1: %v", outputs)
	}

	t.Run("disconnect from client", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		r, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+device.Stream, nil)
		if err != nil {
			t.Fatalf("failed to create test http request: %v", err)
		}
		resp, err := http.DefaultClient.Do(r)
		if err != nil {
			t.Fatalf("failed to request: %v", err)
		}
		defer resp.Body.Close()
		cancel() // cancel stops http client
		if _, err := io.Copy(io.Discard, resp.Body); err != context.Canceled {
			t.Errorf("read http stream %s body got err: %v; want %v", device.Stream, err, context.Canceled)
		}
	})
	t.Run("disconnect by server", func(t *testing.T) {
		ctx, cancel := context.WithCancel(ctx)
		defer cancel()
		r, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL+device.Stream, nil)
		if err != nil {
			t.Fatalf("failed to create test http request: %v", err)
		}
		resp, err := http.DefaultClient.Do(r)
		if err != nil {
			t.Fatalf("failed to request: %v", err)
		}
		defer resp.Body.Close()
		h.Stop() // Shutdown stops http server audio stream
		if _, err := io.Copy(io.Discard, resp.Body); err != nil {
			t.Errorf("read http stream %s body got err: %v; want %v", device.Stream, err, nil)
		}
	})
	go func() {
		sub.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: ""})
		sub.Expect(ctx, &mpdtest.WR{Read: "noidle\n", Write: "OK\n"})
	}()
}

func sortUniq(s []string) []string {
	set := map[string]struct{}{}
	for i := range s {
		set[s[i]] = struct{}{}
	}
	s = make([]string, 0, len(s))
	for k := range set {
		s = append(s, k)
	}
	sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })
	return s
}
