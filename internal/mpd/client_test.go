package mpd

import (
	"context"
	"io"
	"reflect"
	"testing"
	"time"

	"github.com/meiraka/vv/internal/mpd/mpdtest"
)

var (
	testDialer = Dialer{
		ReconnectionTimeout:  time.Second,
		HealthCheckInterval:  time.Second,
		ReconnectionInterval: time.Microsecond,
	}
)

func TestClient(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	w, r, ts, err := mpdtest.NewServer("OK MPD 0.19")
	if err != nil {
		t.Fatalf("failed to create test server: %v", err)
	}
	go func() {
		mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "password 2434\n", Write: "OK\n"})
	}()
	defer ts.Close()
	c, err := testDialer.Dial("tcp", ts.URL, "2434")
	if err != nil {
		t.Fatalf("Dial got error %v; want nil", err)
	}
	defer func() {
		if err := c.Close(ctx); err != nil {
			t.Errorf("Close got error %v; want nil", err)
		}
	}()
	if g, w := c.Version(), "0.19"; g != w {
		t.Errorf("Version() got `%s`; want `%s`", g, w)
	}
	for k, v := range map[string]func(t *testing.T){
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
					want:  []map[string]string{{"outputid": "0", "outputname": "My ALSA Device", "plugin": "alsa", "outputenabled": "0", "attribute": "dop=0"}},
				},
			} {
				t.Run(read, func(t *testing.T) {
					go func() {
						mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: read, Write: tt.write})
					}()
					got, err := tt.cmd(ctx)
					if !reflect.DeepEqual(got, tt.want) || err != nil {
						t.Errorf("got %v, %v; want %v, <nil>", got, err, tt.want)
					}
				})
				t.Run(read+" network read error", func(t *testing.T) {
					go func() {
						mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: read})
						ts.Disconnect(ctx)
					}()
					got, err := tt.cmd(ctx)
					if err != io.EOF {
						t.Errorf("got %v, %v; want nil, %v", got, err, io.EOF)
					}
					mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "password 2434\n", Write: "OK\n"})
				})
				t.Run(read+" command error", func(t *testing.T) {
					go func() {
						mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: read, Write: "ACK [50@1] {test} test error\n"})
					}()
					_, err := tt.cmd(ctx)
					if want := newCommandError("ACK [50@1] {test} test error"); !reflect.DeepEqual(err, want) {
						t.Errorf("got _, %v; want nil, %v", err, want)
					}
				})
				t.Run(read+" context cancel", func(t *testing.T) {
					cmdCtx, cmdCancel := context.WithCancel(ctx)
					go func() {
						mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: read})
						cmdCancel()
					}()
					got, err := tt.cmd(cmdCtx)
					if err != context.Canceled {
						t.Errorf("got %v, %v; want nil, %v", got, err, io.EOF)
					}
					mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "password 2434\n", Write: "OK\n"})
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

func TestDialPasswordError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	w, r, ts, _ := mpdtest.NewServer("OK MPD 0.19")
	go func() {
		mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "password 2434\n", Write: "ACK [3@1] {password} error\n"})
	}()
	defer ts.Close()
	c, err := testDialer.Dial("tcp", ts.URL, "2434")
	want := &CommandError{ID: 3, Index: 1, Command: "password", Message: "error"}
	if !reflect.DeepEqual(err, want) {
		t.Errorf("Dial got error %v; want %v", err, want)
	}
	if err != nil {
		return
	}
	if err := c.Close(ctx); err != nil {
		t.Errorf("Close got error %v; want nil", err)
	}
}
