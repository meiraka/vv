package mpd

import (
	"context"
	"errors"
	"sync"
	"time"
)

var (
	ErrClosed = errors.New("mpd: connection closed")
)

type Dialer struct {
	ReconnectionTimeout  time.Duration
	HelthCheckInterval   time.Duration
	ReconnectionInterval time.Duration
}

func (d *Dialer) newConn(proto, addr, password string) (*Conn, string, error) {
	deadline := time.Time{}
	if d.ReconnectionTimeout != 0 {
		deadline = time.Now().Add(d.ReconnectionTimeout)
	}
	conn, ver, err := NewConn(proto, addr, deadline)
	if err != nil {
		return nil, "", err
	}
	if len(password) > 0 {
		if err := ok(conn, "password", password); err != nil {
			conn.Close()
			return nil, "", err
		}
	}
	return conn, ver, nil
}

func (d Dialer) Dial(proto, addr, password string) (*Client, error) {
	conn, ver, err := d.newConn(proto, addr, password)
	if err != nil {
		return nil, err
	}
	c := &Client{
		proto:    proto,
		addr:     addr,
		password: password,
		connC:    make(chan *Conn, 1),
		close:    make(chan struct{}, 1),
		version:  ver,
		dialer:   &d,
	}
	c.connC <- conn
	go c.helthcheck()
	return c, nil
}

type Client struct {
	proto    string
	addr     string
	password string
	connC    chan *Conn
	close    chan struct{}
	dialer   *Dialer
	version  string
	mutex    sync.RWMutex
}

func (c *Client) Close(ctx context.Context) error {
	conn, err := c.borrowConn(ctx)
	if err != nil {
		return err
	}
	close(c.close)
	close(c.connC)
	defer conn.Close()
	return ok(conn, "close")
}

func (c *Client) Version() string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.version
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

func (c *Client) borrowConn(ctx context.Context) (*Conn, error) {
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

func (c *Client) returnConn(conn *Conn, err error) error {
	if err != nil {
		conn.Close()
		go c.connect()
		return err
	}
	c.connC <- conn
	return err
}

func (c *Client) ok(ctx context.Context, cmd ...interface{}) error {
	conn, err := c.borrowConn(ctx)
	if err != nil {
		return err
	}
	return c.returnConn(conn, ok(conn, cmd...))
}

func ok(conn *Conn, cmd ...interface{}) error {
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

func (c *Client) connect() {
	for {
		conn, ver, err := c.dialer.newConn(c.proto, c.addr, c.password)
		if err != nil {
			time.Sleep(c.dialer.ReconnectionInterval)
			continue
		}
		c.connC <- conn
		c.mutex.Lock()
		defer c.mutex.Unlock()
		c.version = ver
		return
	}
}
