package mpd

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
)

var (
	ErrClosed = errors.New("mpd: connection closed")
)

type Song map[string][]string

type connKeeper struct {
	proto                string
	addr                 string
	password             string
	ReconnectionTimeout  time.Duration
	ReconnectionInterval time.Duration
	connC                chan *conn
	close                chan struct{}
	version              string
	mutex                sync.Mutex
}

func (c *connKeeper) Exec(ctx context.Context, f func(*conn) error) error {
	conn, err := c.borrowConn(ctx)
	if err != nil {
		return err
	}
	return c.returnConn(conn, f(conn))
}

func (c *connKeeper) Close(ctx context.Context) error {
	conn, err := c.borrowConn(ctx)
	if err != nil {
		return err
	}
	close(c.close)
	close(c.connC)
	defer conn.Close()
	return ok(conn, "close")
}

func (c *connKeeper) Version() string {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	return c.version
}

func (c *connKeeper) newConn() (*conn, string, error) {
	deadline := time.Time{}
	if c.ReconnectionTimeout != 0 {
		deadline = time.Now().Add(c.ReconnectionTimeout)
	}
	conn, ver, err := NewConn(c.proto, c.addr, deadline)
	if err != nil {
		return nil, "", err
	}
	if len(c.password) > 0 {
		if err := ok(conn, "password", c.password); err != nil {
			conn.Close()
			return nil, "", err
		}
	}
	return conn, ver, nil
}

func (c *connKeeper) borrowConn(ctx context.Context) (*conn, error) {
	select {
	case <-c.close:
		return nil, ErrClosed
	case conn := <-c.connC:
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
		conn.Close()
		go c.connect()
		return err
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
	conn, ver, err := c.newConn()
	if err != nil {
		return err
	}
	c.connC <- conn
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.version = ver
	return nil
}

type Dialer struct {
	ReconnectionTimeout  time.Duration
	HelthCheckInterval   time.Duration
	ReconnectionInterval time.Duration
}

func (d Dialer) Dial(proto, addr, password string) (*Client, error) {
	conn := &connKeeper{
		proto:                proto,
		addr:                 addr,
		password:             password,
		ReconnectionTimeout:  d.ReconnectionTimeout,
		ReconnectionInterval: d.ReconnectionInterval,
		connC:                make(chan *conn, 1),
		close:                make(chan struct{}),
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

type Client struct {
	proto    string
	addr     string
	password string
	conn     *connKeeper
	close    chan struct{}
	dialer   *Dialer
}

func (c *Client) Close(ctx context.Context) error {
	close(c.close)
	return c.conn.Close(ctx)
}

func (c *Client) Version() string {
	return c.conn.Version()
}

// Music Database Commands

func (c *Client) CountGroup(ctx context.Context, group string) ([]map[string]string, error) {
	return c.mapsLastKeyOk(ctx, "playtime", "count", "group", group)
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

func (c *Client) printok(ctx context.Context, cmd ...interface{}) (string, error) {
	var ret string
	err := c.conn.Exec(ctx, func(conn *conn) error {
		if len(cmd) == 0 {
			return nil
		}
		if _, err := conn.Writeln(cmd...); err != nil {
			return err
		}
		for {
			if s, err := conn.ReadString('\n'); err != nil {
				return err
			} else if s == "OK\n" {
				return nil
			} else {
				ret = ret + s
			}
		}
	})
	return ret, err
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
		return ok(conn, cmd...)
	})
}

func ok(conn *conn, cmd ...interface{}) error {
	if len(cmd) == 0 {
		return nil
	}
	if _, err := conn.Writeln(cmd...); err != nil {
		return err
	}
	if s, err := conn.ReadString('\n'); err != nil {
		return err
	} else if s != "OK\n" {
		return errors.New(s[0 : len(s)-1])
	}
	return nil
}
