package mpd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/meiraka/vv/internal/mpd/mpdtest"
)

func TestDial(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	ts, err := mpdtest.NewServer("OK MPD 0.19")
	if err != nil {
		t.Fatalf("failed to create test server: %v", err)
	}
	defer ts.Close()
	for label, tt := range map[string]struct {
		url  string
		opts *ClientOptions
		want []*mpdtest.WR
		err  bool
	}{
		"urlerr":     {url: "", err: true},
		"nopassword": {url: ts.URL},
		"password": {
			url:  ts.URL,
			opts: &ClientOptions{Password: "2434"},
			want: []*mpdtest.WR{{Read: "password 2434\n", Write: "OK\n"}},
		},
		"password(error)": {
			url:  ts.URL,
			opts: &ClientOptions{Password: "2434"},
			want: []*mpdtest.WR{{Read: "password 2434\n", Write: "ACK [3@1] {password} error\n"}},
			err:  true,
		},
		"binarylimit": {
			url:  ts.URL,
			opts: &ClientOptions{BinaryLimit: 64},
			want: []*mpdtest.WR{{Read: "binarylimit 64\n", Write: "OK\n"}},
		},
		"binarylimit(invalid value)": {
			url:  ts.URL,
			opts: &ClientOptions{BinaryLimit: 2},
			want: []*mpdtest.WR{{Read: "binarylimit 2\n", Write: `ACK [2@0] {binarylimit} Value too small` + "\n"}},
			err:  true,
		},
		"binarylimit(unsupported)": {
			url:  ts.URL,
			opts: &ClientOptions{BinaryLimit: 64},
			want: []*mpdtest.WR{{Read: "binarylimit 64\n", Write: `ACK [5@0] {} unknown command "binarylimit"` + "\n"}},
			err:  false,
		},
		"cache commands result": {
			url:  ts.URL,
			opts: &ClientOptions{CacheCommandsResult: true},
			want: []*mpdtest.WR{{Read: "commands\n", Write: "OK\n"}},
		},
		"fulloptions": { // without health check
			url:  ts.URL,
			opts: &ClientOptions{Password: "2434", BinaryLimit: 64, CacheCommandsResult: true},
			want: []*mpdtest.WR{
				{Read: "password 2434\n", Write: "OK\n"},
				{Read: "binarylimit 64\n", Write: "OK\n"},
				{Read: "commands\n", Write: "OK\n"},
			},
		},
	} {
		t.Run(label, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				for _, w := range tt.want {
					ts.Expect(ctx, w)
				}
				wg.Done()
			}()
			c, err := Dial("tcp", tt.url, tt.opts)
			if !tt.err && err != nil {
				t.Errorf("got err: %v; want <nil>", err)
			}
			if tt.err && err == nil {
				t.Errorf("got no err; want non nil err")
			}
			wg.Wait()
			if err == nil {
				c.Close(ctx)
			}
		})
	}
}

func TestClient(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	ts, err := mpdtest.NewServer("OK MPD 0.19")
	if err != nil {
		t.Fatalf("failed to create test server: %v", err)
	}
	go func() {
		ts.Expect(ctx, &mpdtest.WR{Read: "password 2434\n", Write: "OK\n"})
	}()
	defer ts.Close()
	c, err := Dial("tcp", ts.URL,
		&ClientOptions{Password: "2434", Timeout: testTimeout, ReconnectionInterval: time.Millisecond})
	if err != nil {
		t.Fatalf("Dial got error %v; want nil", err)
	}
	defer func() {
		if err := c.Close(ctx); err != nil {
			t.Errorf("Close got error %v; want nil", err)
		}
		if err := c.Close(ctx); err != ErrClosed {
			t.Errorf("Close got error %v; want %v", err, ErrClosed)
		}
	}()
	if g, w := c.Version(), "0.19"; g != w {
		t.Errorf("Version() got `%s`; want `%s`", g, w)
	}
	for k, v := range map[string]func(t *testing.T){
		"one": func(t *testing.T) {
			testsets := map[string]func(context.Context) error{
				"play -1\n":        func(ctx context.Context) error { return c.Play(ctx, -1) },
				"next\n":           c.Next,
				"previous\n":       c.Previous,
				"pause 1\n":        func(ctx context.Context) error { return c.Pause(ctx, true) },
				"random 1\n":       func(ctx context.Context) error { return c.Random(ctx, true) },
				"random 0\n":       func(ctx context.Context) error { return c.Random(ctx, false) },
				"single 1\n":       func(ctx context.Context) error { return c.Single(ctx, true) },
				"single oneshot\n": c.OneShot,
				"repeat 1\n":       func(ctx context.Context) error { return c.Repeat(ctx, true) },
				"setvol 100\n":     func(ctx context.Context) error { return c.SetVol(ctx, 100) },
			}
			for read, cmd := range testsets {
				t.Run(read, func(t *testing.T) {
					go func() {
						ts.Expect(ctx, &mpdtest.WR{Read: read, Write: "OK\n"})
					}()
					if err := cmd(ctx); err != nil {
						t.Errorf("got %v; want <nil>", err)
					}
				})
				t.Run(read+" network read error", func(t *testing.T) {
					go func() {
						ts.Expect(ctx, &mpdtest.WR{Read: read})
						ts.Disconnect(ctx)
					}()
					cmdCtx, cmdCancel := context.WithTimeout(ctx, testTimeout/100)
					defer cmdCancel()
					err := cmd(cmdCtx)
					if _, ok := err.(net.Error); !ok {
						if err != io.EOF {
							t.Errorf("got %v; want %v or net.Error", err, io.EOF)
						}
					}
					go func() {
						ts.Expect(ctx, &mpdtest.WR{Read: "password 2434\n", Write: "OK\n"})
						ts.Expect(ctx, &mpdtest.WR{Read: "ping\n", Write: "OK\n"})
					}()
					if err := c.Ping(ctx); err != nil {
						t.Errorf("ping got %v; want <nil>", err)
					}
				})
				t.Run(read+" command error", func(t *testing.T) {
					go func() {
						ts.Expect(ctx, &mpdtest.WR{Read: read, Write: "ACK [50@1] {test} test error\n"})
					}()
					if got, want := cmd(ctx), newCommandError("ACK [50@1] {test} test error"); !errors.Is(got, want) {
						t.Errorf("got _, %v; want nil, %v", got, want)
					}
				})
				t.Run(read+" context cancel", func(t *testing.T) {
					cmdCtx, cmdCancel := context.WithCancel(ctx)
					go func() {
						ts.Expect(ctx, &mpdtest.WR{Read: read})
						cmdCancel()
					}()
					if err := cmd(cmdCtx); err != context.Canceled {
						t.Errorf("got _, %v; want nil, %v", err, io.EOF)
					}
					go func() {
						ts.Expect(ctx, &mpdtest.WR{Read: "password 2434\n", Write: "OK\n"})
						ts.Expect(ctx, &mpdtest.WR{Read: "ping\n", Write: "OK\n"})
					}()
					if err := c.Ping(ctx); err != nil {
						t.Errorf("ping got %v; want <nil>", err)
					}
				})
			}

		},
		"get": func(t *testing.T) {
			for read, tt := range map[string]struct {
				cmd   func(context.Context) (interface{}, error)
				write string
				want  interface{}
			}{
				"currentsong\n": {
					cmd:   func(ctx context.Context) (interface{}, error) { return c.CurrentSong(ctx) },
					write: "file: foo\nOK\n",
					want:  map[string][]string{"file": {"foo"}},
				},
				"status\n": {
					cmd:   func(ctx context.Context) (interface{}, error) { return c.Status(ctx) },
					write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n",
					want:  map[string]string{"volume": "-1", "song": "1", "elapsed": "1.1", "repeat": "0", "random": "0", "single": "0", "consume": "0", "state": "pause"},
				},
				"playlistinfo\n": {
					cmd:   func(ctx context.Context) (interface{}, error) { return c.PlaylistInfo(ctx) },
					write: "file: foo\nfile: bar\nOK\n",
					want:  []map[string][]string{{"file": {"foo"}}, {"file": {"bar"}}},
				},
				"listallinfo /\n": {
					cmd:   func(ctx context.Context) (interface{}, error) { return c.ListAllInfo(ctx, "/") },
					write: "file: foo\nfile: bar\nfile: baz\nOK\n",
					want:  []map[string][]string{{"file": {"foo"}}, {"file": {"bar"}}, {"file": {"baz"}}},
				},
				"outputs\n": {
					cmd:   func(ctx context.Context) (interface{}, error) { return c.Outputs(ctx) },
					write: "outputid: 0\noutputname: My ALSA Device\nplugin: alsa\noutputenabled: 0\nattribute: dop=0\nOK\n",
					want:  []*Output{{ID: "0", Name: "My ALSA Device", Plugin: "alsa", Enabled: false, Attributes: map[string]string{"dop": "0"}}},
				},
			} {
				t.Run(read, func(t *testing.T) {
					go func() {
						ts.Expect(ctx, &mpdtest.WR{Read: read, Write: tt.write})
					}()
					got, err := tt.cmd(ctx)
					if !reflect.DeepEqual(got, tt.want) || err != nil {
						t.Errorf("got %v, %v; want %v, <nil>", got, err, tt.want)
					}
				})
				t.Run(read+" network read error", func(t *testing.T) {
					go func() {
						ts.Expect(ctx, &mpdtest.WR{Read: read})
						ts.Disconnect(ctx)
					}()
					cmdCtx, cmdCancel := context.WithTimeout(ctx, testTimeout/100)
					defer cmdCancel()
					got, err := tt.cmd(cmdCtx)
					if _, ok := err.(net.Error); !ok {
						if err != io.EOF {
							t.Errorf("got %v, %v; want nil, %v or net.Error", got, err, io.EOF)
						}
					}
					go func() {
						ts.Expect(ctx, &mpdtest.WR{Read: "password 2434\n", Write: "OK\n"})
						ts.Expect(ctx, &mpdtest.WR{Read: "ping\n", Write: "OK\n"})
					}()
					if err := c.Ping(ctx); err != nil {
						t.Errorf("ping got %v; want <nil>", err)
					}
				})
				t.Run(read+" command error", func(t *testing.T) {
					go func() {
						ts.Expect(ctx, &mpdtest.WR{Read: read, Write: "ACK [50@1] {test} test error\n"})
					}()
					_, err := tt.cmd(ctx)
					if want := newCommandError("ACK [50@1] {test} test error"); !errors.Is(err, want) {
						t.Errorf("got _, %v; want nil, %v", err, want)
					}
				})
				t.Run(read+" context cancel", func(t *testing.T) {
					cmdCtx, cmdCancel := context.WithCancel(ctx)
					go func() {
						ts.Expect(ctx, &mpdtest.WR{Read: read})
						cmdCancel()
					}()
					got, err := tt.cmd(cmdCtx)
					if err != context.Canceled {
						t.Errorf("got %v, %v; want nil, %v", got, err, io.EOF)
					}
					go func() {
						ts.Expect(ctx, &mpdtest.WR{Read: "password 2434\n", Write: "OK\n"})
						ts.Expect(ctx, &mpdtest.WR{Read: "ping\n", Write: "OK\n"})
					}()
					if err := c.Ping(ctx); err != nil {
						t.Errorf("ping got %v; want <nil>", err)
					}
				})
			}
		},
	} {
		select {
		case <-ctx.Done():
			t.Fatalf("test timeout")
		default:
			t.Run(k, v)
		}
	}
}

func TestClientCloseNetworkError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen mock server addr: %v", err)
	}
	svr := make(chan struct{})
	cli := make(chan struct{})
	defer close(svr)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			t.Errorf("failed to accept by mock server: %v", err)
			return
		}
		fmt.Fprintln(conn, "OK MPD 0.19")
		<-svr
		conn.Close()
		ln.Close()
		cli <- struct{}{}
	}()
	c, err := Dial("tcp", ln.Addr().String(),
		&ClientOptions{Timeout: testTimeout, ReconnectionInterval: time.Millisecond})
	if err != nil {
		t.Fatalf("failed to connect mock server: %v", err)
	}
	svr <- struct{}{}
	<-cli
	if err := c.Ping(ctx); err != io.EOF {
		t.Errorf("Ping(ctx) got %v; want %v", err, io.EOF)
	}
	if err := c.Close(ctx); err != ErrClosed {
		t.Errorf("got %v; want %v", err, ErrClosed)
	}

}
