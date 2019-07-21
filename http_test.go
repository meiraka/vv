package main

import (
	"context"
	"fmt"
	"io"
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
)

func TestHTTPHandler(t *testing.T) {
	mainw, mainr, maints, _ := mpdtest.NewChanServer("OK MPD 0.19")
	defer maints.Close()
	c, err := testDialer.Dial("tcp", maints.URL, "")
	if err != nil {
		t.Fatalf("Dial got error %v; want nil", err)
	}
	_, _, watchts, _ := mpdtest.NewChanServer("OK MPD 0.19")
	wl, err := testDialer.NewWatcher("tcp", watchts.URL, "")
	if err != nil {
		t.Fatalf("Dial got error %v; want nil", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	go func() {
		for i := 0; i < 4; i++ {
			switch <-mainr {
			case "playlistinfo\n":
				mainw <- "file: foo\nfile: bar\nOK\n"
			case "listallinfo /\n":
				mainw <- "file: foo\nfile: bar\nfile: baz\nOK\n"
			case "status\n":
				mainw <- "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"
			case "currentsong\n":
				mainw <- "file: bar\nOK\n"
			}

		}

	}()
	h, err := NewHTTPHandler(ctx, c, wl)
	if err != nil {
		t.Fatalf("NewHTTPHandler got error %v; want nil", err)
	}
	ts := httptest.NewServer(h)
	defer ts.Close()
	testsets := []struct {
		req    *http.Request
		status int
		want   string
		f      func() error
	}{
		{
			req:    NewTestRequest(t, "GET", ts.URL+"/api/music/playlist", nil),
			status: 200,
			want:   `{"current":1,"sort":null,"filters":null}`,
		},
		{
			req:    NewTestRequest(t, "GET", ts.URL+"/api/music/playlist/songs", nil),
			status: 200,
			want:   `[{"file":["foo"]},{"file":["bar"]}]`,
		},
		{
			req:    NewTestRequest(t, "GET", ts.URL+"/api/music/library/songs", nil),
			status: 200,
			want:   `[{"file":["foo"]},{"file":["bar"]},{"file":["baz"]}]`,
		},
		{
			req:    NewTestRequest(t, "GET", ts.URL+"/api/music/playlist/songs/current", nil),
			status: 200,
			want:   `{"file":["bar"]}`,
		},
		{
			req:    NewTestRequest(t, "GET", ts.URL+"/api/music", nil),
			status: 200,
			want:   `{"volume":-1,"repeat":false,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1}`,
		},
		{
			req:    NewTestRequest(t, "POST", ts.URL+"/api/music", strings.NewReader(`{"volume":100}`)),
			status: 200,
			want:   `{"volume":100,"repeat":false,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1}`,
			f: func() error {
				if got, want := readChan(ctx, t, mainr), "setvol 100\n"; got != want {
					return fmt.Errorf("got %s; want %s", got, want)
				}
				mainw <- "OK\n"
				if got, want := readChan(ctx, t, mainr), "status\n"; got != want {
					return fmt.Errorf("got %s; want %s", got, want)
				}
				mainw <- "volume: 100\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"
				return nil
			},
		},
		{
			req:    NewTestRequest(t, "POST", ts.URL+"/api/music", strings.NewReader(`{"repeat":true}`)),
			status: 200,
			want:   `{"volume":-1,"repeat":true,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1}`,
			f: func() error {
				if got, want := readChan(ctx, t, mainr), "repeat 1\n"; got != want {
					return fmt.Errorf("got %s; want %s", got, want)
				}
				mainw <- "OK\n"
				if got, want := readChan(ctx, t, mainr), "status\n"; got != want {
					return fmt.Errorf("got %s; want %s", got, want)
				}
				mainw <- "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 1\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"
				return nil
			},
		},
		{
			req:    NewTestRequest(t, "POST", ts.URL+"/api/music", strings.NewReader(`{"random":true}`)),
			status: 200,
			want:   `{"volume":-1,"repeat":false,"random":true,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1}`,
			f: func() error {
				if got, want := readChan(ctx, t, mainr), "random 1\n"; got != want {
					return fmt.Errorf("got %s; want %s", got, want)
				}
				mainw <- "OK\n"
				if got, want := readChan(ctx, t, mainr), "status\n"; got != want {
					return fmt.Errorf("got %s; want %s", got, want)
				}
				mainw <- "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 1\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"
				return nil
			},
		},
		{
			req:    NewTestRequest(t, "POST", ts.URL+"/api/music", strings.NewReader(`{"single":true}`)),
			status: 200,
			want:   `{"volume":-1,"repeat":false,"random":false,"single":true,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1}`,
			f: func() error {
				if got, want := readChan(ctx, t, mainr), "single 1\n"; got != want {
					return fmt.Errorf("got %s; want %s", got, want)
				}
				mainw <- "OK\n"
				if got, want := readChan(ctx, t, mainr), "status\n"; got != want {
					return fmt.Errorf("got %s; want %s", got, want)
				}
				mainw <- "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 1\nconsume: 0\nstate: pause\nOK\n"
				return nil
			},
		},
		{
			req:    NewTestRequest(t, "POST", ts.URL+"/api/music", strings.NewReader(`{"oneshot":true}`)),
			status: 200,
			want:   `{"volume":-1,"repeat":false,"random":false,"single":false,"oneshot":true,"consume":false,"state":"pause","song_elapsed":1.1}`,
			f: func() error {
				if got, want := readChan(ctx, t, mainr), "single oneshot\n"; got != want {
					return fmt.Errorf("got %s; want %s", got, want)
				}
				mainw <- "OK\n"
				if got, want := readChan(ctx, t, mainr), "status\n"; got != want {
					return fmt.Errorf("got %s; want %s", got, want)
				}
				mainw <- "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: oneshot\nconsume: 0\nstate: pause\nOK\n"
				return nil
			},
		},
		{
			req:    NewTestRequest(t, "POST", ts.URL+"/api/music", strings.NewReader(`{"consume":true}`)),
			status: 200,
			want:   `{"volume":-1,"repeat":false,"random":false,"single":false,"oneshot":false,"consume":true,"state":"pause","song_elapsed":1.1}`,
			f: func() error {
				if got, want := readChan(ctx, t, mainr), "consume 1\n"; got != want {
					return fmt.Errorf("got %s; want %s", got, want)
				}
				mainw <- "OK\n"
				if got, want := readChan(ctx, t, mainr), "status\n"; got != want {
					return fmt.Errorf("got %s; want %s", got, want)
				}
				mainw <- "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 1\nstate: pause\nOK\n"
				return nil
			},
		},
		{
			req:    NewTestRequest(t, "POST", ts.URL+"/api/music", strings.NewReader(`{"state":"play"}`)),
			status: 200,
			want:   `{"volume":-1,"repeat":false,"random":false,"single":false,"oneshot":false,"consume":false,"state":"play","song_elapsed":1.1}`,
			f: func() error {
				if got, want := readChan(ctx, t, mainr), "play -1\n"; got != want {
					return fmt.Errorf("got %s; want %s", got, want)
				}
				mainw <- "OK\n"
				if got, want := readChan(ctx, t, mainr), "status\n"; got != want {
					return fmt.Errorf("got %s; want %s", got, want)
				}
				mainw <- "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: play\nOK\n"
				return nil
			},
		},
		{
			req:    NewTestRequest(t, "POST", ts.URL+"/api/music", strings.NewReader(`{"state":"next"}`)),
			status: 200,
			want:   `{"volume":-1,"repeat":false,"random":false,"single":false,"oneshot":false,"consume":false,"state":"play","song_elapsed":1.1}`,
			f: func() error {
				if got, want := readChan(ctx, t, mainr), "next\n"; got != want {
					return fmt.Errorf("got %s; want %s", got, want)
				}
				mainw <- "OK\n"
				if got, want := readChan(ctx, t, mainr), "status\n"; got != want {
					return fmt.Errorf("got %s; want %s", got, want)
				}
				mainw <- "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: play\nOK\n"
				return nil
			},
		},
		{
			req:    NewTestRequest(t, "POST", ts.URL+"/api/music", strings.NewReader(`{"state":"previous"}`)),
			status: 200,
			want:   `{"volume":-1,"repeat":false,"random":false,"single":false,"oneshot":false,"consume":false,"state":"play","song_elapsed":1.1}`,
			f: func() error {
				if got, want := readChan(ctx, t, mainr), "previous\n"; got != want {
					return fmt.Errorf("got %s; want %s", got, want)
				}
				mainw <- "OK\n"
				if got, want := readChan(ctx, t, mainr), "status\n"; got != want {
					return fmt.Errorf("got %s; want %s", got, want)
				}
				mainw <- "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: play\nOK\n"
				return nil
			},
		},
		{
			req:    NewTestRequest(t, "POST", ts.URL+"/api/music", strings.NewReader(`{"state":"pause"}`)),
			status: 200,
			want:   `{"volume":-1,"repeat":false,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":1.1}`,
			f: func() error {
				if got, want := readChan(ctx, t, mainr), "pause 1\n"; got != want {
					return fmt.Errorf("got %s; want %s", got, want)
				}
				mainw <- "OK\n"
				if got, want := readChan(ctx, t, mainr), "status\n"; got != want {
					return fmt.Errorf("got %s; want %s", got, want)
				}
				mainw <- "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"
				return nil
			},
		},
		{
			req:    NewTestRequest(t, "POST", ts.URL+"/api/music", strings.NewReader(`{"state":"foobar"}`)),
			status: 400,
			want:   `{"error":"unknown state: foobar"}`,
		},
	}
	for _, tt := range testsets {
		t.Run(fmt.Sprintf("%s %s", tt.req.Method, tt.req.URL.Path), func(t *testing.T) {
			errs := make(chan error, 1)
			if tt.f != nil {
				go func() {
					errs <- tt.f()
					close(errs)
				}()
			} else {
				close(errs)
			}
			resp, err := http.DefaultClient.Do(tt.req.WithContext(ctx))
			if err != nil {
				t.Fatalf("failed to request: %v", err)
			}
			select {
			case err := <-errs:
				if err != nil {
					t.Fatalf("communication error: %v", err)
				}
			case <-ctx.Done():
				t.Fatalf("communication timeout")
			}
			defer resp.Body.Close()
			b, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				t.Fatalf("failed to read: %v", err)
			}
			if got := string(b); resp.StatusCode != tt.status || got != tt.want {
				t.Errorf("got %d %s; want %d %s", resp.StatusCode, got, tt.status, tt.want)
			}
		})
	}
}

func NewTestRequest(t *testing.T, method, url string, body io.Reader) *http.Request {
	t.Helper()
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	return req
}

func readChan(ctx context.Context, t *testing.T, c <-chan string) (ret string) {
	t.Helper()
	select {
	case ret = <-c:
	case <-ctx.Done():
		t.Fatalf("read timeout %v", ctx.Err())
	}
	return
}
