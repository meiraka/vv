package mpd

import (
	"bufio"
	"context"
	"fmt"
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
	v, err := conn.Readln()
	if err != nil {
		conn.Close()
		return nil, err
	}
	conn.Version = strings.TrimPrefix(v, "OK MPD ")
	return conn, nil
}

func (c *conn) Readln() (string, error) {
	s, err := c.ReadString('\n')
	if err != nil {
		// log.Println("TRACE", "read err:", err)
		return s, err
	}
	// log.Println("TRACE", "read:", s[0:len(s)-1])
	return s[0 : len(s)-1], nil
}

func (c *conn) Close() error {
	// log.Println("TRACE", "close")
	return c.conn.Close()
}

func (c *conn) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

func (c *conn) Writeln(f ...interface{}) (int, error) {
	// log.Println(append([]interface{}{"TRACE", "write:"}, f...)...)
	return fmt.Fprintln(c.conn, f...)
}

func (c *conn) OK(cmd ...interface{}) error {
	if _, err := c.Writeln(cmd...); err != nil {
		return err
	}
	return c.ReadEnd("OK")
}

// ReadEnd reads and checks end message of mpd response.
func (c *conn) ReadEnd(end string) error {
	if v, err := c.Readln(); err != nil {
		return err
	} else if v != end {
		return newCommandError(v)
	}
	return nil
}
