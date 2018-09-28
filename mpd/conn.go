package mpd

import (
	"bufio"
	"fmt"
	"net"
	"time"
)

type conn struct {
	*bufio.Reader
	conn    net.Conn
	version string
}

func NewConn(proto, addr string, deadline time.Time) (*conn, string, error) {
	c, err := net.Dial(proto, addr)
	if err != nil {
		return nil, "", err
	}
	conn := &conn{
		Reader: bufio.NewReader(c),
		conn:   c,
	}
	conn.SetDeadline(deadline)
	s, err := conn.ReadString('\n')
	if err != nil {
		conn.Close()
		return nil, "", err
	}
	return conn, s[len("OK MPD ") : len(s)-1], nil
}

func (c *conn) Readln() (string, error) {
	s, err := c.ReadString('\n')
	if err != nil {
		return s, err
	}
	return s[0 : len(s)-1], nil
}

func (c *conn) Close() error {
	return c.conn.Close()
}

func (c *conn) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

func (c *conn) Writeln(f ...interface{}) (int, error) {
	return fmt.Fprintln(c.conn, f...)
}
