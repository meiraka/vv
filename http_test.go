package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/meiraka/vv/mpd"
	"github.com/meiraka/vv/mpd/mpdtest"
)

const (
	testTimeout = time.Second
)

var (
	testDialer = mpd.Dialer{
		ReconnectionTimeout:  time.Second,
		HelthCheckInterval:   time.Hour,
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
	}
)

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
			want:   `{"current":1,"sort":null,"filters":null}`,
			event:  mpdtest.Append(testMPDEvent, &mpdtest.WR{Read: "close\n"}),
		},
		{
			Method: http.MethodGet,
			Path:   "/api/music/playlist/songs",
			status: http.StatusOK,
			want:   `[{"file":["foo"]},{"file":["bar"]}]`,
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
			want:   `[{"file":["foo"]},{"file":["bar"]},{"file":["baz"]}]`,
			event:  mpdtest.Append(testMPDEvent, &mpdtest.WR{Read: "close\n"}),
		},
		{
			Method: http.MethodGet,
			Path:   "/api/music/playlist/songs/current",
			status: http.StatusOK,
			want:   `{"file":["bar"]}`,
			event:  mpdtest.Append(testMPDEvent, &mpdtest.WR{Read: "close\n"}),
		},
		{
			Method: http.MethodGet,
			Path:   "/api/music",
			status: http.StatusOK,
			want:   `{"volume":-1,"repeat":false,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1}`,
			event:  mpdtest.Append(testMPDEvent, &mpdtest.WR{Read: "close\n"}),
		},
		{
			Method: http.MethodPost,
			Path:   "/api/music",
			Body:   `{"volume":100}`,
			status: http.StatusOK,
			want:   `{"volume":100,"repeat":false,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1}`,
			event: mpdtest.Append(testMPDEvent, []*mpdtest.WR{
				{Read: "setvol 100\n", Write: "OK\n"},
				{Read: "status\n", Write: "volume: 100\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"},
				{Read: "close\n"},
			}...),
		},
		{
			Method: http.MethodPost,
			Path:   "/api/music",
			Body:   `{"repeat":true}`,
			status: http.StatusOK,
			want:   `{"volume":-1,"repeat":true,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1}`,
			event: mpdtest.Append(testMPDEvent, []*mpdtest.WR{
				{Read: "repeat 1\n", Write: "OK\n"},
				{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"},
				{Read: "close\n"},
			}...),
		},
		{
			Method: http.MethodPost,
			Path:   "/api/music",
			Body:   `{"random":true}`,
			status: http.StatusOK,
			want:   `{"volume":-1,"repeat":false,"random":true,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1}`,
			event: mpdtest.Append(testMPDEvent, []*mpdtest.WR{
				{Read: "random 1\n", Write: "OK\n"},
				{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 1\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"},
				{Read: "close\n"},
			}...),
		},
		{
			Method: http.MethodPost,
			Path:   "/api/music",
			Body:   `{"single":true}`,
			status: http.StatusOK,
			want:   `{"volume":-1,"repeat":false,"random":false,"single":true,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1}`,
			event: mpdtest.Append(testMPDEvent, []*mpdtest.WR{
				{Read: "single 1\n", Write: "OK\n"},
				{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 1\nconsume: 0\nstate: pause\nOK\n"},
				{Read: "close\n"},
			}...),
		},
		{
			Method: http.MethodPost,
			Path:   "/api/music",
			Body:   `{"oneshot":true}`,
			status: http.StatusOK,
			want:   `{"volume":-1,"repeat":false,"random":false,"single":false,"oneshot":true,"consume":false,"state":"pause","song_elapsed":1.1}`,
			event: mpdtest.Append(testMPDEvent, []*mpdtest.WR{
				{Read: "single oneshot\n", Write: "OK\n"},
				{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: oneshot\nconsume: 0\nstate: pause\nOK\n"},
				{Read: "close\n"},
			}...),
		},
		{
			Method: http.MethodPost,
			Path:   "/api/music",
			Body:   `{"consume":true}`,
			status: http.StatusOK,
			want:   `{"volume":-1,"repeat":false,"random":false,"single":false,"oneshot":false,"consume":true,"state":"pause","song_elapsed":1.1}`,
			event: mpdtest.Append(testMPDEvent, []*mpdtest.WR{
				{Read: "consume 1\n", Write: "OK\n"},
				{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 1\nstate: pause\nOK\n"},
				{Read: "close\n"},
			}...),
		},
		{
			Method: http.MethodPost,
			Path:   "/api/music",
			Body:   `{"state":"play"}`,
			status: http.StatusOK,
			want:   `{"volume":-1,"repeat":false,"random":false,"single":false,"oneshot":false,"consume":false,"state":"play","song_elapsed":1.1}`,
			event: mpdtest.Append(testMPDEvent, []*mpdtest.WR{
				{Read: "play -1\n", Write: "OK\n"},
				{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: play\nOK\n"},
				{Read: "close\n"},
			}...),
		},
		{
			Method: http.MethodPost,
			Path:   "/api/music",
			Body:   `{"state":"pause"}`,
			status: http.StatusOK,
			want:   `{"volume":-1,"repeat":false,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1}`,
			event: mpdtest.Append(testMPDEvent, []*mpdtest.WR{
				{Read: "pause 1\n", Write: "OK\n"},
				{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"},
				{Read: "close\n"},
			}...),
		},
		{
			Method: http.MethodPost,
			Path:   "/api/music",
			Body:   `{"state":"next"}`,
			status: http.StatusOK,
			want:   `{"volume":-1,"repeat":false,"random":false,"single":false,"oneshot":false,"consume":false,"state":"play","song_elapsed":1.1}`,
			event: mpdtest.Append(testMPDEvent, []*mpdtest.WR{
				{Read: "next\n", Write: "OK\n"},
				{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: play\nOK\n"},
				{Read: "close\n"},
			}...),
		},
		{
			Method: http.MethodPost,
			Path:   "/api/music",
			Body:   `{"state":"previous"}`,
			status: http.StatusOK,
			want:   `{"volume":-1,"repeat":false,"random":false,"single":false,"oneshot":false,"consume":false,"state":"play","song_elapsed":1.1}`,
			event: mpdtest.Append(testMPDEvent, []*mpdtest.WR{
				{Read: "previous\n", Write: "OK\n"},
				{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: play\nOK\n"},
				{Read: "close\n"},
			}...),
		},
		{
			Method: http.MethodPost,
			Path:   "/api/music/playlist",
			Body:   `{"current":0,"sort":["file"],"filters":[]}`,
			want:   `{"current":1,"sort":null,"filters":null}`,
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
			status: http.StatusOK,
			want:   `{"current":2,"sort":["file"],"filters":[]}`,
			event: []*mpdtest.WR{
				{Read: "listallinfo /\n", Write: "file: foo\nfile: bar\nfile: baz\nOK\n"},
				{Read: "playlistinfo\n", Write: "file: bar\nfile: baz\nfile: foo\nOK\n"},
				{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"},
				{Read: "currentsong\n", Write: "file: bar\nOK\n"},
				{Read: "play 2\n", Write: "OK\n"},
				{Read: "status\n", Write: "volume: -1\nsong: 2\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"},
				{Read: "close\n"},
			},
		},
		{
			Method: http.MethodPost,
			Path:   "/api/music/library",
			Body:   `{"updating":true}`,
			status: http.StatusOK,
			want:   `{"updating":true}`,
			event: mpdtest.Append(testMPDEvent, []*mpdtest.WR{
				{Read: "update \n", Write: "updating_db: 2\nOK\n"},
				{Read: "status\n", Write: "volume: -1\nsong: 2\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nupdating_db: 2\nOK\n"},
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

			h, err := testHTTPConfig.NewHTTPHandler(ctx, c, wl, nil)
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
