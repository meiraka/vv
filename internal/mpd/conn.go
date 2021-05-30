package mpd

import (
	"bufio"
	"context"
	"net"
	"strings"
	"time"
)

type conn struct {
	*bufio.Reader
	conn    net.Conn
	Version string
}

func newConn(ctx context.Context, proto, addr string) (*conn, error) {
	dialer := net.Dialer{}
	c, err := dialer.DialContext(ctx, proto, addr)
	if err != nil {
		return nil, err
	}
	conn := &conn{
		Reader: bufio.NewReader(c),
		conn:   c,
	}
	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
	}
	v, err := readln(conn)
	if err != nil {
		conn.Close()
		return nil, err
	}
	conn.Version = strings.TrimPrefix(v, "OK MPD ")
	return conn, nil
}

func (c *conn) Write(p []byte) (n int, err error) {
	return c.conn.Write(p)
}

func (c *conn) Close() error {
	// log.Println("TRACE", "close")
	return c.conn.Close()
}

func (c *conn) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}
