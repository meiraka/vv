package mpd

import (
	"context"
	"sync"
	"time"
)

type pool struct {
	proto                string
	addr                 string
	password             string
	Timeout              time.Duration
	ReconnectionInterval time.Duration
	connC                chan *conn
	connCtx              context.Context
	connCancel           context.CancelFunc
	mu                   sync.RWMutex
	version              string
}

func newPool(proto string, addr string, password string, timeout time.Duration, reconnectionInterval time.Duration) (*pool, error) {
	ctx, cancel := context.WithCancel(context.Background())
	p := &pool{
		proto:                proto,
		addr:                 addr,
		password:             password,
		Timeout:              timeout,
		ReconnectionInterval: reconnectionInterval,
		connC:                make(chan *conn, 1),
		connCtx:              ctx,
		connCancel:           cancel,
	}
	if err := p.connectOnce(); err != nil {
		return nil, err
	}
	return p, nil
}

func (c *pool) Exec(ctx context.Context, f func(*conn) error) error {
	conn, err := c.get(ctx)
	if err != nil {
		return err
	}
	errs := make(chan error)
	go func() {
		errs <- f(conn)
		close(errs)
	}()
	select {
	case err = <-errs:
	case <-ctx.Done():
		err = ctx.Err()
		conn.SetDeadline(time.Now())
	}
	return c.returnConn(conn, err)
}

func (c *pool) Close(ctx context.Context) error {
	c.connCancel()
	conn, err := c.get(ctx)
	if err != nil {
		return err
	}
	close(c.connC)
	return conn.Close()
}

func (c *pool) Version() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.version
}

func (c *pool) get(ctx context.Context) (*conn, error) {
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

func (c *pool) returnConn(conn *conn, err error) error {
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

func (c *pool) connect() {
	for {
		if err := c.connectOnce(); err != nil {
			select {
			case <-c.connCtx.Done():
				close(c.connC)
				return
			case <-time.After(c.ReconnectionInterval):
			}
			continue
		}
		return
	}
}

func (c *pool) connectOnce() error {
	ctx := c.connCtx
	if c.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.Timeout)
		defer cancel()
	}
	conn, err := newConn(ctx, c.proto, c.addr)
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
	c.version = conn.Version
	return nil
}
