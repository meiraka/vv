package mpd

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	// ErrClosed is returned when connection is closed by client.
	ErrClosed = errors.New("mpd: connection closed")
)

const (
	responseOK = "OK"
)

// Client is a mpd client.
type Client struct {
	pool            *pool
	stopHealthCheck func()
	opts            *ClientOptions
	commands        []string
	mu              sync.RWMutex
}

// Dial connects to mpd server.
func Dial(proto, addr string, opts *ClientOptions) (*Client, error) {
	if opts == nil {
		opts = &ClientOptions{}
	}
	c := &Client{opts: opts}
	pool, err := newPool(proto, addr, opts.Timeout, opts.ReconnectionInterval, func(conn *conn) error {
		if err := opts.connectHook(conn); err != nil {
			return err
		}
		if opts.CacheCommandsResult {
			if err := c.updateCommands(conn); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	hCtx, hStop := context.WithCancel(context.Background())
	c.pool = pool
	c.stopHealthCheck = hStop
	go c.healthCheck(hCtx)
	return c, nil
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
		if _, err := fmt.Fprintln(conn, "currentsong"); err != nil {
			return err
		}
		song, err = parseSong(conn, responseOK)
		return err
	})
	if err != nil {
		return nil, addCommandInfo(err, "currentsong")
	}
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

// ReplayGainMode sets the replay gain mode.
func (c *Client) ReplayGainMode(ctx context.Context, mode string) error {
	return c.ok(ctx, "replay_gain_mode", mode)
}

// ReplayGainStatus prints replay gain options.
func (c *Client) ReplayGainStatus(ctx context.Context) (map[string]string, error) {
	return c.mapStr(ctx, "replay_gain_status")
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

// SeekCur seeks to the position t within the current song
func (c *Client) SeekCur(ctx context.Context, t float64) error {
	return c.ok(ctx, "seekcur", t)
}

// The Queue

// PlaylistInfo displays a list of all songs in the playlist.
func (c *Client) PlaylistInfo(ctx context.Context) (songs []map[string][]string, err error) {
	err = c.pool.Exec(ctx, func(conn *conn) error {
		if _, err := fmt.Fprintln(conn, "playlistinfo"); err != nil {
			return err
		}
		songs, err = parseSongs(conn, responseOK)
		return err
	})
	if err != nil {
		return nil, addCommandInfo(err, "playlistinfo")
	}
	return
}

// The music database

// AlbumArt locates album art for the given song.
func (c *Client) AlbumArt(ctx context.Context, uri string) ([]byte, error) {
	return c.binary(ctx, "albumart", quote(uri))
}

// ReadPicture locates picture for the given song.
// If song has no picture, returns nil, nil.
func (c *Client) ReadPicture(ctx context.Context, uri string) ([]byte, error) {
	return c.binary(ctx, "readpicture", quote(uri))
}

// ListAllInfo lists all songs and directories in uri.
func (c *Client) ListAllInfo(ctx context.Context, uri string) (songs []map[string][]string, err error) {
	err = c.pool.Exec(ctx, func(conn *conn) error {
		if _, err := fmt.Fprintln(conn, "listallinfo", uri); err != nil {
			return err
		}
		songs, err = parseSongs(conn, responseOK)
		return err
	})
	if err != nil {
		return nil, addCommandInfo(err, "listallinfo")
	}
	return
}

// Update updates the music database.
func (c *Client) Update(ctx context.Context, uri string) (map[string]string, error) {
	return c.mapStr(ctx, "update", quote(uri))
}

// Mounts and neighbors

// Mount the specified storage uri at the given path.
func (c *Client) Mount(ctx context.Context, path, url string) error {
	return c.ok(ctx, "mount", quote(path), quote(url))
}

// Unmount the specified path.
func (c *Client) Unmount(ctx context.Context, path string) error {
	return c.ok(ctx, "unmount", quote(path))
}

// ListMounts queries a list of all mounts.
func (c *Client) ListMounts(ctx context.Context) ([]map[string]string, error) {
	return c.listMap(ctx, "mount", "listmounts")
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

// Output represents mpd output struct.
type Output struct {
	ID         string
	Name       string
	Enabled    bool
	Plugin     string
	Attributes map[string]string
}

// Outputs shows information about all outputs.
func (c *Client) Outputs(ctx context.Context) (outputs []*Output, err error) {
	err = c.pool.Exec(ctx, func(conn *conn) error {
		if _, err := fmt.Fprintln(conn, "outputs"); err != nil {
			return err
		}
		outputs, err = parseOutputs(conn, responseOK)
		return err
	})
	if err != nil {
		return nil, addCommandInfo(err, "outputs")
	}
	return
}

// OutputSet sets a runtime attribute.
func (c *Client) OutputSet(ctx context.Context, id, name, value string) error {
	return c.ok(ctx, "outputset", quote(id), quote(name), quote(value))
}

func (c *Client) healthCheck(ctx context.Context) {
	if c.opts.HealthCheckInterval == 0 {
		return
	}
	ticker := time.NewTicker(c.opts.HealthCheckInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(ctx, c.opts.HealthCheckInterval)
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

// Reflection

// Config dumps configuration values that may be interesting for the client.
// This command is only permitted to “local” clients (connected via local socket).
func (c *Client) Config(ctx context.Context) (map[string]string, error) {
	return c.mapStr(ctx, "config")
}

// Commands returns which commands the current user has access to.
func (c *Client) Commands(ctx context.Context) (commands []string, err error) {
	if !c.opts.CacheCommandsResult {
		err = c.pool.Exec(ctx, func(conn *conn) error {
			if _, err := fmt.Fprintln(conn, "commands"); err != nil {
				return err
			}
			commands, err = parseList(conn, responseOK, "command")
			return err
		})
		if err != nil {
			return nil, addCommandInfo(err, "commands")
		}
		return
	}
	c.mu.RLock()
	commands = c.commands
	c.mu.RUnlock()
	return commands, nil
}

func (c *Client) updateCommands(conn *conn) error {
	if _, err := fmt.Fprintln(conn, "commands"); err != nil {
		return err
	}
	commands, err := parseList(conn, responseOK, "command")
	if err != nil {
		return addCommandInfo(err, "commands")
	}
	c.mu.Lock()
	c.commands = commands
	c.mu.Unlock()
	return nil
}

func (c *Client) ok(ctx context.Context, cmd ...interface{}) error {
	return c.pool.Exec(ctx, func(conn *conn) error {
		return execOK(conn, cmd...)
	})
}

func (c *Client) binaryPart(ctx context.Context, cmd, uri string, pos int) (m map[string]string, b []byte, err error) {
	err = c.pool.Exec(ctx, func(conn *conn) error {
		if _, err := fmt.Fprintln(conn, cmd, uri, pos); err != nil {
			return err
		}
		m, b, err = parseBinary(conn, responseOK)
		return err
	})
	if err != nil {
		return nil, nil, addCommandInfo(err, cmd)
	}
	return
}

func (c *Client) binary(ctx context.Context, cmd, uri string) ([]byte, error) {
	m, b, err := c.binaryPart(ctx, cmd, uri, 0)
	if err != nil {
		return nil, err
	}
	if len(b) == 0 {
		return nil, nil
	}
	size, err := strconv.Atoi(m["size"])
	if err != nil {
		return nil, err
	}
	for {
		if size == len(b) {
			return b, nil
		}
		if size < len(b) {
			return nil, errors.New("oversize")
		}
		_, nb, err := c.binaryPart(ctx, cmd, uri, len(b))
		if err != nil {
			return nil, err
		}
		b = append(b, nb...)
	}
}

func (c *Client) mapStr(ctx context.Context, cmd ...interface{}) (m map[string]string, err error) {
	err = c.pool.Exec(ctx, func(conn *conn) error {
		if _, err := fmt.Fprintln(conn, cmd...); err != nil {
			return err
		}
		m, err = parseMap(conn, responseOK)
		return err
	})
	if err != nil {
		return nil, addCommandInfo(err, cmd[0].(string))
	}
	return
}

func (c *Client) listMap(ctx context.Context, newKey string, cmd ...interface{}) (l []map[string]string, err error) {
	err = c.pool.Exec(ctx, func(conn *conn) error {
		if _, err := fmt.Fprintln(conn, cmd...); err != nil {
			return err
		}
		l, err = parseListMap(conn, responseOK, newKey)
		return err
	})
	if err != nil {
		return nil, addCommandInfo(err, cmd[0].(string))
	}
	return
}

// ClientOptions contains options for mpd client connection.
type ClientOptions struct {
	Password string
	// Timeout is the maximum amount of time a dial will wait for a connect to complete.
	Timeout              time.Duration
	HealthCheckInterval  time.Duration
	ReconnectionInterval time.Duration
	// BinaryLimit sets maximum binary response size.
	BinaryLimit int
	// CacheCommandsResult caches mpd command "commands" result
	CacheCommandsResult bool
}

func (c *ClientOptions) connectHook(conn *conn) error {
	if len(c.Password) > 0 {
		if err := execOK(conn, "password", c.Password); err != nil {
			return err
		}
	}
	if c.BinaryLimit > 0 {
		err := execOK(conn, "binarylimit", c.BinaryLimit)
		if err != nil && !errors.Is(err, ErrUnknown) { // MPD-0.22.3 or earlier returns ErrUnknown
			return err
		}
	}
	return nil
}

// quote escaping strings values for mpd.
func quote(s string) string {
	return `"` + strings.Replace(
		strings.Replace(s, "\\", "\\\\", -1),
		`"`, `\"`, -1) + `"`
}

func execOK(c *conn, cmd ...interface{}) error {
	if _, err := fmt.Fprintln(c, cmd...); err != nil {
		return err
	}
	if err := parseEnd(c, responseOK); err != nil {
		return addCommandInfo(err, cmd[0].(string))
	}
	return nil
}
