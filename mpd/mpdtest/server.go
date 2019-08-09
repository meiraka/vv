package mpdtest

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
)

// Server is mock mpd server.
type Server struct {
	ln  net.Listener
	URL string
}

// Close closes connection
func (s *Server) Close() error {
	return s.ln.Close()
}

// NewServer creates new mpd mock Server.
func NewServer(firstResp string, resp map[string]string) (*Server, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	s := &Server{
		ln:  ln,
		URL: ln.Addr().String(),
	}
	go func(ln net.Listener) {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			if _, err := fmt.Fprintln(conn, firstResp); err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				r := bufio.NewReader(conn)
				cmd := ""
				for {
					nl, err := r.ReadString('\n')
					if err != nil {
						continue
					}
					cmd = cmd + nl
					for k, v := range resp {
						if k+"\n" == cmd {
							cmd = ""
							_, err := fmt.Fprint(conn, v)
							if err != nil {
								return
							}
							break
						}
					}
				}

			}(conn)
		}
	}(ln)
	return s, nil
}

// WR represents testserver Write / Read string
type WR struct {
	Read  string
	Write string
}

// Append appends new WR
func Append(o []*WR, n ...*WR) []*WR {
	lo := len(o)
	ret := make([]*WR, lo+len(n))
	for i := range o {
		ret[i] = o[i]
	}
	for i := range n {
		ret[i+lo] = n[i]
	}
	return ret
}

// NewEventServer creates new mpd mock Server.
func NewEventServer(firstResp string, resp []*WR) (*Server, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	s := &Server{
		ln:  ln,
		URL: ln.Addr().String(),
	}
	go func(ln net.Listener) {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			if _, err := fmt.Fprintln(conn, firstResp); err != nil {
				return
			}
			go func(conn net.Conn) {
				defer conn.Close()
				r := bufio.NewReader(conn)
				cmd := ""
				for {
					nl, err := r.ReadString('\n')
					if err != nil {
						return
					}
					cmd = cmd + nl
					if len(resp) == 0 {
						_, err := fmt.Fprintf(conn, "ACK [5@0] {} resp length is zero: got %s", cmd[:len(cmd)-1])
						cmd = ""
						if err != nil {
							return
						}
					} else if !strings.Contains(resp[0].Read, cmd) {
						_, err := fmt.Fprintf(conn, "ACK [5@0] {} got %s; want %s", cmd[:len(cmd)-1], resp[0].Read)
						cmd = ""
						if err != nil {
							return
						}
						resp = resp[1:]
					} else if resp[0].Read == cmd {
						cmd = ""
						_, err := fmt.Fprint(conn, resp[0].Write)
						if err != nil {
							return
						}
						resp = resp[1:]
					}
				}

			}(conn)
		}
	}(ln)
	return s, nil
}

// DefineMessage defines mpd read/write message
func DefineMessage(ctx context.Context, w chan string, r <-chan string, m *WR) {
	ws := m.Write
	select {
	case <-ctx.Done():
		return
	case s := <-r:
		if s != m.Read {
			ws = fmt.Sprintf("ACK [5@0] {} got %s; want %s\n", strings.TrimSuffix(s, "\n"), strings.TrimSuffix(m.Read, "\n"))
		}
	}
	select {
	case <-ctx.Done():
	case w <- ws:
	}

}

// NewChanServer creates new mpd mock Server for idle command.
func NewChanServer(firstResp string) (chan string, <-chan string, *Server, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, nil, nil, err
	}
	s := &Server{
		ln:  ln,
		URL: ln.Addr().String(),
	}
	wc := make(chan string, 10)
	rc := make(chan string, 10)
	go func(ln net.Listener) {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			if _, err := fmt.Fprintln(conn, firstResp); err != nil {
				return
			}
			go func(conn net.Conn) {
				for m := range wc {
					if _, err := fmt.Fprint(conn, m); err != nil {
						return
					}
				}
			}(conn)
			go func(conn net.Conn) {
				defer conn.Close()
				r := bufio.NewReader(conn)
				for {
					nl, err := r.ReadString('\n')
					if err != nil {
						return
					}
					rc <- nl
				}

			}(conn)
		}
	}(ln)
	return wc, rc, s, nil
}
