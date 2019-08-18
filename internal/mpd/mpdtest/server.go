package mpdtest

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

// Server is mock mpd server.
type Server struct {
	ln         net.Listener
	URL        string
	disconnect chan struct{}
	mu         sync.Mutex
	closed     bool
}

// Disconnect closes current connection.
func (s *Server) Disconnect(ctx context.Context) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()
	select {
	case s.disconnect <- struct{}{}:
	case <-ctx.Done():
	}
}

// Close closes connection
func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	close(s.disconnect)
	return s.ln.Close()
}

// WR represents testserver Write / Read string
type WR struct {
	Read  string
	Write string
}

// Expect expects mpd read/write message
func Expect(ctx context.Context, w chan string, r <-chan string, m *WR) {
	ws := m.Write
	select {
	case <-ctx.Done():
		return
	case s := <-r:
		if s != m.Read {
			ws = fmt.Sprintf("ACK [5@0] {} got %s; want %s\n", strings.TrimSuffix(m.Read, "\n"), strings.TrimSuffix(s, "\n"))
		}
	}
	select {
	case <-ctx.Done():
	case w <- ws:
	}

}

// NewServer creates new mpd mock Server for idle command.
func NewServer(firstResp string) (chan string, <-chan string, *Server, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, nil, nil, err
	}
	s := &Server{
		ln:         ln,
		URL:        ln.Addr().String(),
		disconnect: make(chan struct{}, 1),
	}
	wc := make(chan string)
	rc := make(chan string)
	go func(ln net.Listener) {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			if _, err := fmt.Fprintln(conn, firstResp); err != nil {
				break
			}
			go func(conn net.Conn) {
				ctx, cancel := context.WithCancel(context.Background())
				go func(conn net.Conn) {
					defer cancel()
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

				select {
				case <-s.disconnect:
				case <-ctx.Done():
				}
				conn.SetDeadline(time.Now().Add(-1 * time.Second))
				conn.Close()
			}(conn)
		}
	}(ln)
	return wc, rc, s, nil
}
