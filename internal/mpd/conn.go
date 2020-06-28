package mpd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
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

// ReadBinary reads key-value map and mpd binary responses.
func (c *conn) ReadBinary(cmd ...interface{}) (map[string]string, []byte, error) {
	if _, err := c.Writeln(cmd...); err != nil {
		return nil, nil, err
	}
	m := map[string]string{}
	var key, value string
	for {
		line, err := c.Readln()
		if err != nil {
			return nil, nil, err
		}
		if line == "OK" {
			return m, nil, nil
		}
		i := strings.Index(line, ": ")
		if i < 0 {
			return nil, nil, newCommandError(line)
		}
		key = line[0:i]
		value = line[i+2:]
		m[key] = value
		if key == "binary" {
			length, err := strconv.Atoi(value)
			if err != nil {
				return nil, nil, err
			}
			// binary
			b := make([]byte, length)
			_, err = io.ReadFull(c, b)
			if err != nil {
				return nil, nil, err
			}
			// newline
			_, err = c.ReadString('\n')
			if err != nil {
				return nil, nil, err
			}
			// OK
			return m, b, c.ReadEnd("OK")
		}
	}
}
