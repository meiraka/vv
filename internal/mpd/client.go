package mpd

import (
	"context"
	"errors"
	"strconv"
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
func (c *Client) CurrentSong(ctx context.Context) (map[string][]string, error) {
	ch := make(chan map[string][]string, 1)
	err := c.pool.Exec(ctx, func(conn *conn) error {
		defer close(ch)
		if err := request(conn, "currentsong"); err != nil {
			return err
		}
		song, err := parseSong(conn, responseOK)
		ch <- song
		return err
	})
	if err != nil {
		return nil, addCommandInfo(err, "currentsong")
	}
	return <-ch, nil
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
	return c.ok(ctx, "consume", state)
}

// Crossfade sets crossfading between song
func (c *Client) Crossfade(ctx context.Context, t time.Duration) error {
	return c.ok(ctx, "crossfade", int(t/time.Second))
}

// Random sets random state.
func (c *Client) Random(ctx context.Context, state bool) error {
	return c.ok(ctx, "random", state)
}

// Repeat sets repeat state.
func (c *Client) Repeat(ctx context.Context, state bool) error {
	return c.ok(ctx, "repeat", state)
}

// Single sets single state.
func (c *Client) Single(ctx context.Context, state bool) error {
	return c.ok(ctx, "single", state)
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
	return c.ok(ctx, "pause", state)
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
func (c *Client) PlaylistInfo(ctx context.Context) ([]map[string][]string, error) {
	ch := make(chan []map[string][]string, 1)
	err := c.pool.Exec(ctx, func(conn *conn) error {
		defer close(ch)
		if err := request(conn, "playlistinfo"); err != nil {
			return err
		}
		songs, err := parseSongs(conn, responseOK)
		ch <- songs
		return err
	})
	if err != nil {
		return nil, addCommandInfo(err, "playlistinfo")
	}
	return <-ch, nil
}

// The music database

// AlbumArt locates album art for the given song.
func (c *Client) AlbumArt(ctx context.Context, uri string) ([]byte, error) {
	return c.binary(ctx, "albumart", uri)
}

// ReadPicture locates picture for the given song.
// If song has no picture, returns nil, nil.
func (c *Client) ReadPicture(ctx context.Context, uri string) ([]byte, error) {
	return c.binary(ctx, "readpicture", uri)
}

// ListAllInfo lists all songs and directories in uri.
func (c *Client) ListAllInfo(ctx context.Context, uri string) ([]map[string][]string, error) {
	ch := make(chan []map[string][]string, 1)
	err := c.pool.Exec(ctx, func(conn *conn) error {
		defer close(ch)
		if err := request(conn, "listallinfo", uri); err != nil {
			return err
		}
		songs, err := parseSongs(conn, responseOK)
		ch <- songs
		return err
	})
	if err != nil {
		return nil, addCommandInfo(err, "listallinfo")
	}
	return <-ch, nil
}

// Update updates the music database.
func (c *Client) Update(ctx context.Context, uri string) (map[string]string, error) {
	return c.mapStr(ctx, "update", uri)
}

// Mounts and neighbors

// Mount the specified storage uri at the given path.
func (c *Client) Mount(ctx context.Context, path, url string) error {
	return c.ok(ctx, "mount", path, url)
}

// Unmount the specified path.
func (c *Client) Unmount(ctx context.Context, path string) error {
	return c.ok(ctx, "unmount", path)
}

// ListMounts queries a list of all mounts.
func (c *Client) ListMounts(ctx context.Context) ([]map[string]string, error) {
	return c.listMap(ctx, "mount", "listmounts")
}

// ListNeighbors queries a list of “neighbors” (e.g. accessible file servers on the local net).
func (c *Client) ListNeighbors(ctx context.Context) ([]map[string]string, error) {
	return c.listMap(ctx, "neighbor", "listneighbors")
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
func (c *Client) Outputs(ctx context.Context) ([]*Output, error) {
	ch := make(chan []*Output, 1)
	err := c.pool.Exec(ctx, func(conn *conn) error {
		defer close(ch)
		if err := request(conn, "outputs"); err != nil {
			return err
		}
		outputs, err := parseOutputs(conn, responseOK)
		ch <- outputs
		return err
	})
	if err != nil {
		return nil, addCommandInfo(err, "outputs")
	}
	return <-ch, nil
}

// OutputSet sets a runtime attribute.
func (c *Client) OutputSet(ctx context.Context, id, name, value string) error {
	return c.ok(ctx, "outputset", id, name, value)
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
func (c *Client) Commands(ctx context.Context) ([]string, error) {
	if !c.opts.CacheCommandsResult {
		ch := make(chan []string, 1)
		err := c.pool.Exec(ctx, func(conn *conn) error {
			defer close(ch)
			if err := request(conn, "commands"); err != nil {
				return err
			}
			commands, err := parseList(conn, responseOK, "command")
			ch <- commands
			return err
		})
		if err != nil {
			return nil, addCommandInfo(err, "commands")
		}
		return <-ch, nil
	}
	c.mu.RLock()
	commands := c.commands
	c.mu.RUnlock()
	return commands, nil
}

func (c *Client) updateCommands(conn *conn) error {
	if err := request(conn, "commands"); err != nil {
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

func (c *Client) ok(ctx context.Context, cmd string, args ...interface{}) error {
	return c.pool.Exec(ctx, func(conn *conn) error {
		return execOK(conn, cmd, args...)
	})
}

func (c *Client) binaryPart(ctx context.Context, pos int, cmd string, args ...interface{}) (map[string]string, []byte, error) {
	ch1, ch2 := make(chan map[string]string, 1), make(chan []byte, 1)
	err := c.pool.Exec(ctx, func(conn *conn) error {
		defer close(ch1)
		defer close(ch2)
		if err := request(conn, cmd, append(args, pos)...); err != nil {
			return err
		}
		m, b, err := parseBinary(conn, responseOK)
		ch1 <- m
		ch2 <- b
		return err
	})
	if err != nil {
		return nil, nil, addCommandInfo(err, cmd)
	}
	return <-ch1, <-ch2, nil
}

func (c *Client) binary(ctx context.Context, cmd string, args ...interface{}) ([]byte, error) {
	m, b, err := c.binaryPart(ctx, 0, cmd, args...)
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
		_, nb, err := c.binaryPart(ctx, len(b), cmd, args...)
		if err != nil {
			return nil, err
		}
		b = append(b, nb...)
	}
}

func (c *Client) mapStr(ctx context.Context, cmd string, args ...interface{}) (map[string]string, error) {
	ch := make(chan map[string]string, 1)
	err := c.pool.Exec(ctx, func(conn *conn) error {
		defer close(ch)
		if err := request(conn, cmd, args...); err != nil {
			return err
		}
		m, err := parseMap(conn, responseOK)
		ch <- m
		return err
	})
	if err != nil {
		return nil, addCommandInfo(err, cmd)
	}
	return <-ch, nil
}

func (c *Client) listMap(ctx context.Context, newKey string, cmd string, args ...interface{}) ([]map[string]string, error) {
	ch := make(chan []map[string]string, 1)
	err := c.pool.Exec(ctx, func(conn *conn) error {
		defer close(ch)
		if err := request(conn, cmd, args...); err != nil {
			return err
		}
		l, err := parseListMap(conn, responseOK, newKey)
		ch <- l
		return err
	})
	if err != nil {
		return nil, addCommandInfo(err, cmd)
	}
	return <-ch, nil
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

func execOK(c *conn, cmd string, args ...interface{}) error {
	if err := request(c, cmd, args...); err != nil {
		return err
	}
	if err := parseEnd(c, responseOK); err != nil {
		return addCommandInfo(err, cmd)
	}
	return nil
}
