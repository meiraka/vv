package mpd

import (
	"bufio"
	"fmt"
	"net"
	"time"
)

type Conn struct {
	*bufio.Reader
	conn    net.Conn
	version string
}

func NewConn(proto, addr string, deadline time.Time) (*Conn, string, error) {
	c, err := net.Dial(proto, addr)
	if err != nil {
		return nil, "", err
	}
	conn := &Conn{
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

func (c *Conn) Close() error {
	return c.conn.Close()
}

func (c *Conn) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

func (c *Conn) Writeln(f ...interface{}) (int, error) {
	return fmt.Fprintln(c.conn, f...)
}
