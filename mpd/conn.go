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

func newConn(proto, addr string, deadline time.Time) (*conn, string, error) {
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
	// log.Println("TRACE", "read:", s[0:len(s)-1])
	return s[0 : len(s)-1], nil
}

func (c *conn) Close() error {
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
	if len(cmd) == 0 {
		return nil
	}
	if _, err := c.Writeln(cmd...); err != nil {
		return fmt.Errorf("write failed: %v", err)
	}
	return c.ReadEnd("OK")
}

// ReadEnd reads and checks end message of mpd response.
func (c *conn) ReadEnd(end string) error {
	if s, err := c.ReadString('\n'); err != nil {
		return fmt.Errorf("read failed: %v", err)
	} else if s != end+"\n" {
		return newCommandError(s[0 : len(s)-1])
	}
	return nil
}
