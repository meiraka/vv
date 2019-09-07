package mpd

import (
	"context"
	"errors"
	"strings"
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
	// Timeout is the maximum amount of time a dial will wait for a connect to complete.
	Timeout              time.Duration
	HealthCheckInterval  time.Duration
	ReconnectionInterval time.Duration
}

// Dial connects to mpd server.
func (d Dialer) Dial(proto, addr, password string) (*Client, error) {
	pool, err := newPool(proto, addr, password, d.Timeout, d.ReconnectionInterval)
	if err != nil {
		return nil, err
	}
	hCtx, hStop := context.WithCancel(context.Background())
	c := &Client{
		pool:            pool,
		dialer:          &d,
		stopHealthCheck: hStop,
	}
	go c.healthCheck(hCtx)
	return c, nil
}

// Client is a mpd client.
type Client struct {
	proto           string
	addr            string
	password        string
	pool            *pool
	stopHealthCheck func()
	dialer          *Dialer
}

// Close closes mpd connection.
func (c *Client) Close(ctx context.Context) error {
	c.stopHealthCheck()
	return c.pool.Close(ctx)
}

// Version returns mpd server version.
func (c *Client) Version() string {
	return c.pool.Version()
}

// Querying MPD’s status

// CurrentSong displays the song info of the current song
func (c *Client) CurrentSong(ctx context.Context) (song map[string][]string, err error) {
	err = c.pool.Exec(ctx, func(conn *conn) error {
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
				return newCommandError(line)
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

// Stats displays statistics.
func (c *Client) Stats(ctx context.Context) (map[string]string, error) {
	return c.mapStr(ctx, "stats")
}

// Music Database Commands

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
	err = c.pool.Exec(ctx, func(conn *conn) error {
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
				return newCommandError(line)
			}
			i := strings.Index(line, ": ")
			if i < 0 {
				return newCommandError(line)
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
	err = c.pool.Exec(ctx, func(conn *conn) error {
		if _, err := conn.Writeln("listallinfo", uri); err != nil {
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
					return newCommandError(line)
				}
				i := strings.Index(line, ": ")
				if i < 0 {
					return newCommandError(line)
				}
				key := line[0:i]
				if _, found := song[key]; !found {
					song[key] = []string{line[i+2:]}
				} else {
					song[key] = append(song[key], line[i+2:])
				}
			} else if strings.HasPrefix(line, "ACK [") {
				return newCommandError(line)
			}
		}
	})
	return
}

// Update updates the music database.
func (c *Client) Update(ctx context.Context, uri string) (map[string]string, error) {
	return c.mapStr(ctx, "update", uri)
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

func (c *Client) healthCheck(ctx context.Context) {
	if c.dialer.HealthCheckInterval == 0 {
		return
	}
	ticker := time.NewTicker(c.dialer.HealthCheckInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(ctx, c.dialer.HealthCheckInterval)
				c.Ping(ctx)
				cancel()
			}
		}
	}()
}

// Ping tests connection.
func (c *Client) Ping(ctx context.Context) error {
	return c.ok(ctx, "ping")
}

func (c *Client) ok(ctx context.Context, cmd ...interface{}) error {
	return c.pool.Exec(ctx, func(conn *conn) error {
		return conn.OK(cmd...)
	})
}

func (c *Client) mapStr(ctx context.Context, cmd ...interface{}) (m map[string]string, err error) {
	err = c.pool.Exec(ctx, func(conn *conn) error {
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
				return newCommandError(line)
			}
			m[line[0:i]] = line[i+2:]
		}
	})
	return
}

func (c *Client) listMap(ctx context.Context, newKey string, cmd ...interface{}) (l []map[string]string, err error) {
	err = c.pool.Exec(ctx, func(conn *conn) error {
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
				return newCommandError(line)
			}
			i := strings.Index(line, ": ")
			if i < 0 {
				return newCommandError(line)
			}
			m[line[0:i]] = line[i+2:]
		}
	})
	return
}

// quote escaping strings values for mpd.
func quote(s string) string {
	return `"` + strings.Replace(
		strings.Replace(s, "\\", "\\\\", -1),
		`"`, `\"`, -1) + `"`
}
