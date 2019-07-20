package mpd

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

var (
	// ErrClosed is returned when connection is closed by client.
	ErrClosed = errors.New("mpd: connection closed")
)

// Song represents song tag.
type Song map[string][]string

// Dialer contains options for connecting to mpd
type Dialer struct {
	ReconnectionTimeout  time.Duration
	HelthCheckInterval   time.Duration
	ReconnectionInterval time.Duration
}

// Dial connects to mpd server.
func (d Dialer) Dial(proto, addr, password string) (*Client, error) {
	conn := &connKeeper{
		proto:                proto,
		addr:                 addr,
		password:             password,
		ReconnectionTimeout:  d.ReconnectionTimeout,
		ReconnectionInterval: d.ReconnectionInterval,
		connC:                make(chan *conn, 1),
	}
	if err := conn.connectOnce(); err != nil {
		return nil, err
	}
	c := &Client{
		close:  make(chan struct{}, 1),
		conn:   conn,
		dialer: &d,
	}
	go c.helthcheck()
	return c, nil
}

// Client is a mpd client.
type Client struct {
	proto    string
	addr     string
	password string
	conn     *connKeeper
	close    chan struct{}
	dialer   *Dialer
}

// Close closes mpd connection.
func (c *Client) Close(ctx context.Context) error {
	close(c.close)
	return c.conn.Close(ctx)
}

// Version returns mpd server version.
func (c *Client) Version() string {
	return c.conn.Version()
}

// Querying MPD’s status

// CurrentSong displays the song info of the current song
func (c *Client) CurrentSong(ctx context.Context) (song map[string][]string, err error) {
	err = c.conn.Exec(ctx, func(conn *conn) error {
		if _, err := conn.Writeln("currentsong"); err != nil {
			return err
		}
		song = map[string][]string{}
		for {
			line, err := conn.Readln()
			if err != nil {
				return err
			}
			if line == "OK" {
				return nil
			}
			i := strings.Index(line, ": ")
			if i < 0 {
				return fmt.Errorf("can't parse line: " + line)
			}
			key := line[0:i]
			if _, found := song[key]; !found {
				song[key] = []string{line[i+2:]}
			} else {
				song[key] = append(song[key], line[i+2:])
			}
		}
	})
	return
}

// Status reports the current status of the player and the volume level.
func (c *Client) Status(ctx context.Context) (map[string]string, error) {
	return c.mapStr(ctx, "status")
}

// Music Database Commands

// CountGroup counts the number of songs by a tag(ex: artist)
func (c *Client) CountGroup(ctx context.Context, group string) ([]map[string]string, error) {
	return c.mapsLastKeyOk(ctx, "playtime", "count", "group", group)
}

// Playback options

// Consume sets consume state.
func (c *Client) Consume(ctx context.Context, state bool) error {
	return c.ok(ctx, "consume", btoa(state, "1", "0"))
}

// Crossfade sets crossfading between song
func (c *Client) Crossfade(ctx context.Context, t time.Duration) error {
	return c.ok(ctx, "crossfade", int(t/time.Second))
}

func btoa(s bool, t string, f string) string {
	if s {
		return t
	}
	return f
}

// Random sets random state.
func (c *Client) Random(ctx context.Context, state bool) error {
	return c.ok(ctx, "random", btoa(state, "1", "0"))
}

// Repeat sets repeat state.
func (c *Client) Repeat(ctx context.Context, state bool) error {
	return c.ok(ctx, "repeat", btoa(state, "1", "0"))
}

// Single sets single state.
func (c *Client) Single(ctx context.Context, state bool) error {
	return c.ok(ctx, "single", btoa(state, "1", "0"))
}

// OneShot sets single state to oneshot.
func (c *Client) OneShot(ctx context.Context) error {
	return c.ok(ctx, "single", "oneshot")
}

// SetVol sets the volume to vol.
func (c *Client) SetVol(ctx context.Context, vol int) error {
	return c.ok(ctx, "setvol", vol)
}

// Controlling playback

// Next plays next song in the playlist.
func (c *Client) Next(ctx context.Context) error {
	return c.ok(ctx, "next")
}

// Pause toggles pause/resumes playing
func (c *Client) Pause(ctx context.Context, state bool) error {
	return c.ok(ctx, "pause", btoa(state, "1", "0"))
}

// Play Begins playing the playlist at song number pos
func (c *Client) Play(ctx context.Context, pos int) error {
	return c.ok(ctx, "play", pos)
}

// Previous plays next song in the playlist.
func (c *Client) Previous(ctx context.Context) error {
	return c.ok(ctx, "previous")
}

// The Queue

// PlaylistInfo displays a list of all songs in the playlist.
func (c *Client) PlaylistInfo(ctx context.Context) (songs []map[string][]string, err error) {
	err = c.conn.Exec(ctx, func(conn *conn) error {
		if _, err := conn.Writeln("playlistinfo"); err != nil {
			return err
		}
		var song map[string][]string
		for {
			line, err := conn.Readln()
			if err != nil {
				return err
			}
			if line == "OK" {
				return nil
			}
			if strings.HasPrefix(line, "file: ") {
				song = map[string][]string{}
				songs = append(songs, song)
			}
			if len(songs) == 0 {
				return fmt.Errorf("unexpected: " + line)
			}
			i := strings.Index(line, ": ")
			if i < 0 {
				return fmt.Errorf("can't parse line: " + line)
			}
			key := line[0:i]
			if _, found := song[key]; !found {
				song[key] = []string{line[i+2:]}
			} else {
				song[key] = append(song[key], line[i+2:])
			}
		}
	})
	return
}

// The music database

// ListAllInfo lists all songs and directories in uri.
func (c *Client) ListAllInfo(ctx context.Context, uri string) (songs []map[string][]string, err error) {
	err = c.conn.Exec(ctx, func(conn *conn) error {
		if _, err := conn.Writeln("playlistinfo"); err != nil {
			return err
		}
		var song map[string][]string
		var inEntry bool
		for {
			line, err := conn.Readln()
			if err != nil {
				return err
			}
			if line == "OK" {
				return nil
			}
			if strings.HasPrefix(line, "file: ") {
				song = map[string][]string{}
				songs = append(songs, song)
				inEntry = true
			} else if strings.HasPrefix(line, "directory: ") {
				inEntry = false

			}
			if inEntry {
				if len(songs) == 0 {
					return fmt.Errorf("unexpected: " + line)
				}
				i := strings.Index(line, ": ")
				if i < 0 {
					return fmt.Errorf("can't parse line: " + line)
				}
				key := line[0:i]
				if _, found := song[key]; !found {
					song[key] = []string{line[i+2:]}
				} else {
					song[key] = append(song[key], line[i+2:])
				}
			}
		}
	})
	return
}

// Audio output devices

// DisableOutput turns an output off.
func (c *Client) DisableOutput(ctx context.Context, id string) error {
	return c.ok(ctx, "disableoutput", id)
}

// EnableOutput turns an output on.
func (c *Client) EnableOutput(ctx context.Context, id string) error {
	return c.ok(ctx, "enableoutput", id)
}

// Outputs shows information about all outputs.
func (c *Client) Outputs(ctx context.Context) ([]map[string]string, error) {
	return c.listMap(ctx, "outputid: ", "outputs")
}

func (c *Client) helthcheck() {
	if c.dialer.HelthCheckInterval == 0 {
		return
	}
	ticker := time.NewTicker(c.dialer.HelthCheckInterval)
	go func() {
		select {
		case <-c.close:
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), c.dialer.HelthCheckInterval)
			c.ok(ctx, "ping")
			cancel()
		}
	}()
}

func (c *Client) mapsLastKeyOk(ctx context.Context, lastKey string, cmd ...interface{}) ([]map[string]string, error) {
	var ret []map[string]string
	err := c.conn.Exec(ctx, func(conn *conn) error {
		item := map[string]string{}
		ret = []map[string]string{}
		if len(cmd) == 0 {
			return nil
		}
		if _, err := conn.Writeln(cmd...); err != nil {
			return err
		}
		for {
			if s, err := conn.Readln(); err != nil {
				return err
			} else if s == "OK" {
				return nil
			} else {
				kv := strings.SplitN(s, ": ", 2)
				if len(kv) != 2 {
					continue
				}
				item[kv[0]] = kv[1]
				if kv[0] == lastKey {
					ret = append(ret, item)
					item = map[string]string{}
				}
			}
		}
	})
	return ret, err
}

func (c *Client) ok(ctx context.Context, cmd ...interface{}) error {
	return c.conn.Exec(ctx, func(conn *conn) error {
		return conn.OK(cmd...)
	})
}

func (c *Client) mapStr(ctx context.Context, cmd ...interface{}) (m map[string]string, err error) {
	err = c.conn.Exec(ctx, func(conn *conn) error {
		if _, err := conn.Writeln(cmd...); err != nil {
			return err
		}
		m = map[string]string{}
		for {
			line, err := conn.Readln()
			if err != nil {
				return err
			}
			if line == "OK" {
				return nil
			}
			i := strings.Index(line, ": ")
			if i < 0 {
				return fmt.Errorf("can't parse line: " + line)
			}
			m[line[0:i]] = line[i+2:]
		}
	})
	return
}

func (c *Client) listMap(ctx context.Context, newKey string, cmd ...interface{}) (l []map[string]string, err error) {
	err = c.conn.Exec(ctx, func(conn *conn) error {
		if _, err := conn.Writeln(cmd...); err != nil {
			return err
		}
		var m map[string]string
		for {
			line, err := conn.Readln()
			if err != nil {
				return err
			}
			if line == "OK" {
				return nil
			}
			if strings.HasPrefix(line, newKey) {
				m = map[string]string{}
				l = append(l, m)
			}
			if m == nil {
				return fmt.Errorf("unexpected: " + line)
			}
			i := strings.Index(line, ": ")
			if i < 0 {
				return fmt.Errorf("can't parse line: " + line)
			}
			m[line[0:i]] = line[i+2:]
		}
	})
	return
}

type connKeeper struct {
	proto                string
	addr                 string
	password             string
	ReconnectionTimeout  time.Duration
	ReconnectionInterval time.Duration
	connC                chan *conn
	mu                   sync.Mutex
	version              string
}

func (c *connKeeper) Exec(ctx context.Context, f func(*conn) error) error {
	conn, err := c.get(ctx)
	if err != nil {
		return err
	}
	return c.returnConn(conn, f(conn))
}

func (c *connKeeper) Close(ctx context.Context) error {
	conn, err := c.get(ctx)
	if err != nil {
		return err
	}
	close(c.connC)
	defer conn.Close()
	return conn.OK("close")
}

func (c *connKeeper) Version() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.version
}

func (c *connKeeper) get(ctx context.Context) (*conn, error) {
	select {
	case conn, ok := <-c.connC:
		if !ok {
			return nil, ErrClosed
		}
		if d, ok := ctx.Deadline(); ok {
			conn.SetDeadline(d)
		} else {
			conn.SetDeadline(time.Time{})
		}
		return conn, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *connKeeper) returnConn(conn *conn, err error) error {
	if err != nil {
		if _, ok := err.(*CommandError); !ok {
			conn.Close()
			go c.connect()
			return err
		}
	}
	c.connC <- conn
	return err
}

func (c *connKeeper) connect() {
	for {
		if err := c.connectOnce(); err != nil {
			time.Sleep(c.ReconnectionInterval)
			continue
		}
		return
	}
}

func (c *connKeeper) connectOnce() error {
	deadline := time.Time{}
	if c.ReconnectionTimeout != 0 {
		deadline = time.Now().Add(c.ReconnectionTimeout)
	}
	conn, ver, err := newConn(c.proto, c.addr, deadline)
	if err != nil {
		return err
	}
	if len(c.password) > 0 {
		if err := conn.OK("password", c.password); err != nil {
			conn.Close()
			return err
		}
	}
	c.connC <- conn
	c.mu.Lock()
	defer c.mu.Unlock()
	c.version = ver
	return nil
}

// quote escaping strings values for mpd.
func quote(s string) string {
	return `"` + strings.Replace(
		strings.Replace(s, "\\", "\\\\", -1),
		`"`, `\"`, -1) + `"`
}
