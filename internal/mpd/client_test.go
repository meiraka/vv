package mpd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"reflect"
	"sync"
	"syscall"
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
			want: []*mpdtest.WR{{Read: "password \"2434\"\n", Write: "OK\n"}},
		},
		"password(error)": {
			url:  ts.URL,
			opts: &ClientOptions{Password: "2434"},
			want: []*mpdtest.WR{{Read: "password \"2434\"\n", Write: "ACK [3@1] {password} error\n"}},
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
			want: []*mpdtest.WR{{Read: "commands\n", Write: "command: foo\nOK\n"}},
		},
		"fulloptions": { // without health check
			url:  ts.URL,
			opts: &ClientOptions{Password: "2434", BinaryLimit: 64, CacheCommandsResult: true},
			want: []*mpdtest.WR{
				{Read: "password \"2434\"\n", Write: "OK\n"},
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

var img = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\b\x06\x00\x00\x00\x1f\x15ĉ\x00\x00\x00\x11IDATx\x9cbb```\x00\x04\x00\x00\xff\xff\x00\x0f\x00\x03\xfe\x8f\xeb\xcf\x00\x00\x00\x00IEND\xaeB`\x82")
var imgSize = len(img)

func TestClient(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	ts, err := mpdtest.NewServer("OK MPD 0.19")
	if err != nil {
		t.Fatalf("failed to create test server: %v", err)
	}
	go func() {
		ts.Expect(ctx, &mpdtest.WR{Read: "password \"2434\"\n", Write: "OK\n"})
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
	for read, tt := range map[string]struct {
		cmd1 func(context.Context) error
		cmd2 func(context.Context) (interface{}, error)
		wr   []*mpdtest.WR
		want interface{}
	}{
		// Querying MPD’s status
		"currentsong": {
			cmd2: func(ctx context.Context) (interface{}, error) { return c.CurrentSong(ctx) },
			wr:   []*mpdtest.WR{{Read: "currentsong\n", Write: "file: foo\nOK\n"}},
			want: map[string][]string{"file": {"foo"}},
		},
		"status": {
			cmd2: func(ctx context.Context) (interface{}, error) { return c.Status(ctx) },
			wr:   []*mpdtest.WR{{Read: "status\n", Write: "volume: -1\nsong: 1\nelapsed: 1.1\nrepeat: 0\nrandom: 0\nsingle: 0\nconsume: 0\nstate: pause\nOK\n"}},
			want: map[string]string{"volume": "-1", "song": "1", "elapsed": "1.1", "repeat": "0", "random": "0", "single": "0", "consume": "0", "state": "pause"},
		},
		"stats": {
			cmd2: func(ctx context.Context) (interface{}, error) { return c.Stats(ctx) },
			wr:   []*mpdtest.WR{{Read: "stats\n", Write: "uptime: 506282\nplaytime: 45148\nartists: 981\nalbums: 597\nsongs: 6411\ndb_playtime: 1659296\ndb_update: 1610585747\nOK\n"}},
			want: map[string]string{"uptime": "506282", "playtime": "45148", "artists": "981", "albums": "597", "songs": "6411", "db_playtime": "1659296", "db_update": "1610585747"},
		},
		// Playback options
		"consume 1": {
			cmd1: func(ctx context.Context) error { return c.Consume(ctx, true) },
			wr:   []*mpdtest.WR{{Read: "consume 1\n", Write: "OK\n"}},
		},
		"consume 0": {
			cmd1: func(ctx context.Context) error { return c.Consume(ctx, false) },
			wr:   []*mpdtest.WR{{Read: "consume 0\n", Write: "OK\n"}},
		},
		"crossfade 10": {
			cmd1: func(ctx context.Context) error { return c.Crossfade(ctx, 10*time.Second) },
			wr:   []*mpdtest.WR{{Read: "crossfade 10\n", Write: "OK\n"}},
		},
		"crossfade 0": {
			cmd1: func(ctx context.Context) error { return c.Crossfade(ctx, 0) },
			wr:   []*mpdtest.WR{{Read: "crossfade 0\n", Write: "OK\n"}},
		},
		"random 1": {
			cmd1: func(ctx context.Context) error { return c.Random(ctx, true) },
			wr:   []*mpdtest.WR{{Read: "random 1\n", Write: "OK\n"}},
		},
		"random 0": {
			cmd1: func(ctx context.Context) error { return c.Random(ctx, false) },
			wr:   []*mpdtest.WR{{Read: "random 0\n", Write: "OK\n"}},
		},
		"repeat 1": {
			cmd1: func(ctx context.Context) error { return c.Repeat(ctx, true) },
			wr:   []*mpdtest.WR{{Read: "repeat 1\n", Write: "OK\n"}},
		},
		"single 1": {
			cmd1: func(ctx context.Context) error { return c.Single(ctx, true) },
			wr:   []*mpdtest.WR{{Read: "single 1\n", Write: "OK\n"}},
		},
		"single oneshot": {
			cmd1: c.OneShot,
			wr:   []*mpdtest.WR{{Read: "single \"oneshot\"\n", Write: "OK\n"}},
		},
		"setvol 100": {
			cmd1: func(ctx context.Context) error { return c.SetVol(ctx, 100) },
			wr:   []*mpdtest.WR{{Read: "setvol 100\n", Write: "OK\n"}},
		},
		"replay_gain_mode album": {
			cmd1: func(ctx context.Context) error { return c.ReplayGainMode(ctx, "album") },
			wr:   []*mpdtest.WR{{Read: "replay_gain_mode \"album\"\n", Write: "OK\n"}},
		},
		"replay_gain_status": {
			cmd2: func(ctx context.Context) (interface{}, error) { return c.ReplayGainStatus(ctx) },
			wr:   []*mpdtest.WR{{Read: "replay_gain_status\n", Write: "replay_gain_mode: off\nOK\n"}},
			want: map[string]string{"replay_gain_mode": "off"},
		},
		// Controlling playback
		"next": {
			cmd1: c.Next,
			wr:   []*mpdtest.WR{{Read: "next\n", Write: "OK\n"}},
		},
		"pause 1": {
			cmd1: func(ctx context.Context) error { return c.Pause(ctx, true) },
			wr:   []*mpdtest.WR{{Read: "pause 1\n", Write: "OK\n"}},
		},
		"play -1": {
			cmd1: func(ctx context.Context) error { return c.Play(ctx, -1) },
			wr:   []*mpdtest.WR{{Read: "play -1\n", Write: "OK\n"}},
		},
		"previous": {
			cmd1: c.Previous,
			wr:   []*mpdtest.WR{{Read: "previous\n", Write: "OK\n"}},
		},
		// The Queue
		"playlistinfo": {
			cmd2: func(ctx context.Context) (interface{}, error) { return c.PlaylistInfo(ctx) },
			wr:   []*mpdtest.WR{{Read: "playlistinfo\n", Write: "file: foo\nfile: bar\nOK\n"}},
			want: []map[string][]string{{"file": {"foo"}}, {"file": {"bar"}}},
		},
		// The music database
		"albumart": {
			cmd2: func(ctx context.Context) (interface{}, error) { return c.AlbumArt(ctx, "foo/bar.flac") },
			wr:   []*mpdtest.WR{{Read: "albumart \"foo/bar.flac\" 0\n", Write: fmt.Sprintf("size: %d\nbinary: %d\n%s\nOK\n", imgSize, imgSize, img)}},
			want: img,
		},
		"albumart(part)": {
			cmd2: func(ctx context.Context) (interface{}, error) { return c.AlbumArt(ctx, "foo/bar.flac") },
			wr: []*mpdtest.WR{
				{Read: "albumart \"foo/bar.flac\" 0\n", Write: fmt.Sprintf("size: %d\nbinary: %d\n%s\nOK\n", imgSize, 10, img[:10])},
				{Read: "albumart \"foo/bar.flac\" 10\n", Write: fmt.Sprintf("size: %d\nbinary: %d\n%s\nOK\n", imgSize, imgSize-10, img[10:])},
			},
			want: img,
		},
		"listallinfo /": {
			cmd2: func(ctx context.Context) (interface{}, error) { return c.ListAllInfo(ctx, "/") },
			wr:   []*mpdtest.WR{{Read: "listallinfo \"/\"\n", Write: "file: foo\nfile: bar\nfile: baz\nOK\n"}},
			want: []map[string][]string{{"file": {"foo"}}, {"file": {"bar"}}, {"file": {"baz"}}},
		},
		"readpicture": {
			cmd2: func(ctx context.Context) (interface{}, error) { return c.ReadPicture(ctx, "foo/bar.flac") },
			wr:   []*mpdtest.WR{{Read: "readpicture \"foo/bar.flac\" 0\n", Write: fmt.Sprintf("size: %d\nbinary: %d\n%s\nOK\n", imgSize, imgSize, img)}},
			want: img,
		},
		"readpicture(part)": {
			cmd2: func(ctx context.Context) (interface{}, error) { return c.ReadPicture(ctx, "foo/bar.flac") },
			wr: []*mpdtest.WR{
				{Read: "readpicture \"foo/bar.flac\" 0\n", Write: fmt.Sprintf("size: %d\nbinary: %d\n%s\nOK\n", imgSize, 10, img[:10])},
				{Read: "readpicture \"foo/bar.flac\" 10\n", Write: fmt.Sprintf("size: %d\nbinary: %d\n%s\nOK\n", imgSize, imgSize-10, img[10:])},
			},
			want: img,
		},
		"update /": {
			cmd2: func(ctx context.Context) (interface{}, error) { return c.Update(ctx, "/") },
			wr:   []*mpdtest.WR{{Read: "update \"/\"\n", Write: "updating_db: 1\nOK\n"}},
			want: map[string]string{"updating_db": "1"},
		},
		// Mounts and neighbors
		"mount": {
			cmd1: func(ctx context.Context) error { return c.Mount(ctx, "xxx", "xxx") },
			wr:   []*mpdtest.WR{{Read: "mount \"xxx\" \"xxx\"\n", Write: "OK\n"}},
		},
		"unmount": {
			cmd1: func(ctx context.Context) error { return c.Unmount(ctx, "xxx") },
			wr:   []*mpdtest.WR{{Read: "unmount \"xxx\"\n", Write: "OK\n"}},
		},
		"listmounts": {
			cmd2: func(ctx context.Context) (interface{}, error) { return c.ListMounts(ctx) },
			wr:   []*mpdtest.WR{{Read: "listmounts\n", Write: "mount: \nstorage: /home/foo/music\nmount: foo\nstorage: nfs://192.168.1.4/export/mp3\nOK\n"}},
			want: []map[string]string{{"mount": "", "storage": "/home/foo/music"}, {"mount": "foo", "storage": "nfs://192.168.1.4/export/mp3"}},
		},
		// Audio output devices
		"disableoutput": {
			cmd1: func(ctx context.Context) error { return c.DisableOutput(ctx, "1") },
			wr:   []*mpdtest.WR{{Read: "disableoutput \"1\"\n", Write: "OK\n"}},
		},
		"enableoutput": {
			cmd1: func(ctx context.Context) error { return c.EnableOutput(ctx, "1") },
			wr:   []*mpdtest.WR{{Read: "enableoutput \"1\"\n", Write: "OK\n"}},
		},
		"outputs": {
			cmd2: func(ctx context.Context) (interface{}, error) { return c.Outputs(ctx) },
			wr:   []*mpdtest.WR{{Read: "outputs\n", Write: "outputid: 0\noutputname: My ALSA Device\nplugin: alsa\noutputenabled: 0\nattribute: dop=0\nOK\n"}},
			want: []*Output{{ID: "0", Name: "My ALSA Device", Plugin: "alsa", Enabled: false, Attributes: map[string]string{"dop": "0"}}},
		},
		"outputset": {
			cmd1: func(ctx context.Context) error { return c.OutputSet(ctx, "0", "dop", "1") },
			wr:   []*mpdtest.WR{{Read: "outputset \"0\" \"dop\" \"1\"\n", Write: "OK\n"}},
		},
		// Reflection
		"config": {
			cmd2: func(ctx context.Context) (interface{}, error) { return c.Config(ctx) },
			wr:   []*mpdtest.WR{{Read: "config\n", Write: "music_directory: /var/lib/mpd/music\nOK\n"}},
			want: map[string]string{"music_directory": "/var/lib/mpd/music"},
		},
		"commands": {
			cmd2: func(ctx context.Context) (interface{}, error) { return c.Commands(ctx) },
			wr:   []*mpdtest.WR{{Read: "commands\n", Write: "command: foobar\nOK\n"}},
			want: []string{"foobar"},
		},
	} {
		select {
		case <-ctx.Done():
			t.Fatal("test exceeds timeout")
		default:
		}
		t.Run(read, func(t *testing.T) {
			if tt.cmd1 == nil && tt.cmd2 == nil {
				t.Fatalf("no test function: cmd1 and cmd2 are nil")
			}
			t.Run("ok", func(t *testing.T) {
				go func() {
					for _, wr := range tt.wr {
						ts.Expect(ctx, wr)
					}
				}()
				if tt.cmd1 != nil {
					if err := tt.cmd1(ctx); err != nil {
						t.Errorf("got %v; want <nil>", err)
					}
				} else if tt.cmd2 != nil {
					got, err := tt.cmd2(ctx)
					if !reflect.DeepEqual(got, tt.want) || err != nil {
						t.Errorf("got %v, %v; want %v, <nil>", got, err, tt.want)
					}
				}
			})
			t.Run("network read error", func(t *testing.T) {
				go func() {
					ts.Expect(ctx, &mpdtest.WR{Read: tt.wr[0].Read})
					ts.Disconnect(ctx)
				}()
				cmdCtx, cmdCancel := context.WithTimeout(ctx, testTimeout/100)
				defer cmdCancel()
				opErr := &net.OpError{}
				if tt.cmd1 != nil {
					err := tt.cmd1(cmdCtx)
					if !errors.As(err, &opErr) && !errors.Is(err, io.EOF) && !errors.Is(err, context.DeadlineExceeded) {
						t.Errorf("got %v; want (%v, %v or net.OpError)", err, io.EOF, context.DeadlineExceeded)
					}
				} else if tt.cmd2 != nil {
					got, err := tt.cmd2(cmdCtx)
					if !errors.As(err, &opErr) && !errors.Is(err, io.EOF) && !errors.Is(err, context.DeadlineExceeded) {
						t.Errorf("got %v, %v; want nil, (%v, %v or net.OpError)", got, err, io.EOF, context.DeadlineExceeded)
					}
				}
				go func() {
					ts.Expect(ctx, &mpdtest.WR{Read: "password \"2434\"\n", Write: "OK\n"})
					ts.Expect(ctx, &mpdtest.WR{Read: "ping\n", Write: "OK\n"})
				}()
				if err := c.Ping(ctx); err != nil {
					t.Errorf("ping got %v; want <nil>", err)
				}
			})
			t.Run("command error", func(t *testing.T) {
				go func() {
					ts.Expect(ctx, &mpdtest.WR{Read: tt.wr[0].Read, Write: "ACK [50@1] {test} test error\n"})
				}()
				var err error
				if tt.cmd1 != nil {
					err = tt.cmd1(ctx)
				} else if tt.cmd2 != nil {
					_, err = tt.cmd2(ctx)
				}
				if want := parseCommandError("ACK [50@1] {test} test error"); !errors.Is(err, want) {
					t.Errorf("got _, %v; want _, %v", err, want)
				}
			})
			t.Run("context cancel", func(t *testing.T) {
				cmdCtx, cmdCancel := context.WithCancel(ctx)
				go func() {
					ts.Expect(ctx, &mpdtest.WR{Read: tt.wr[0].Read})
					cmdCancel()
				}()
				var err error
				if tt.cmd1 != nil {
					err = tt.cmd1(cmdCtx)
				} else if tt.cmd2 != nil {
					_, err = tt.cmd2(cmdCtx)
				}
				if !errors.Is(err, context.Canceled) {
					t.Errorf("got _, %v; want nil, %v", err, context.Canceled)
				}
				go func() {
					ts.Expect(ctx, &mpdtest.WR{Read: "password \"2434\"\n", Write: "OK\n"})
					ts.Expect(ctx, &mpdtest.WR{Read: "ping\n", Write: "OK\n"})
				}()
				if err := c.Ping(ctx); err != nil {
					t.Errorf("ping got %v; want <nil>", err)
				}
			})
		})
	}
}

func TestClientCloseNetworkError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
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
	opErr := &net.OpError{}
	if err := c.Ping(ctx); !errors.As(err, &opErr) || opErr.Op != "write" {
		t.Errorf("Ping(ctx) got %v; want %v", err, &net.OpError{Op: "write", Err: syscall.EPIPE})
	}
	if err := c.Close(ctx); !errors.Is(err, ErrClosed) {
		t.Errorf("got %v; want %v", err, ErrClosed)
	}

}
