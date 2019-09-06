package mpdtest

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
)

// Server is mock mpd server.
type Server struct {
	ln         net.Listener
	URL        string
	disconnect chan struct{}
	rc         chan *rConn
	mu         sync.Mutex
	closed     bool
}

type rConn struct {
	read string
	wc   chan string
}

// Disconnect closes current connection.
func (s *Server) Disconnect(ctx context.Context) {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	select {
	case s.disconnect <- struct{}{}:
	case <-ctx.Done():
	}
	s.mu.Unlock()
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
func (s *Server) Expect(ctx context.Context, m *WR) {
	select {
	case <-ctx.Done():
		return
	case r := <-s.rc:
		w := m.Write
		if r.read != m.Read {
			w = fmt.Sprintf("ACK [5@0] {} got %s; want %s\n", strings.TrimSuffix(m.Read, "\n"), strings.TrimSuffix(r.read, "\n"))
		}
		select {
		case <-ctx.Done():
		case r.wc <- w:
		}
	}

}

// NewServer creates new mpd mock Server for idle command.
func NewServer(firstResp string) (*Server, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	rc := make(chan *rConn)
	s := &Server{
		ln:         ln,
		URL:        ln.Addr().String(),
		disconnect: make(chan struct{}, 1),
		rc:         rc,
	}
	go func(ln net.Listener) {
		var wg sync.WaitGroup
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			if _, err := fmt.Fprintln(conn, firstResp); err != nil {
				break
			}
			wg.Add(1)
			go func(conn net.Conn) {
				defer wg.Done()
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				wg.Add(1)
				go func() {
					defer wg.Done()
					defer cancel()
					defer conn.Close()
					r := bufio.NewReader(conn)
					wc := make(chan string, 1)
					for {
						nl, err := r.ReadString('\n')
						if err != nil {
							return
						}
						rc <- &rConn{
							read: nl,
							wc:   wc,
						}
						select {
						case <-ctx.Done():
							return
						case l := <-wc:
							if len(l) != 0 {
								if _, err := fmt.Fprint(conn, l); err != nil {
									return
								}
							}
						}
					}
				}()
				select {
				case <-ctx.Done():
				case <-s.disconnect:
				}
				conn.Close()
			}(conn)
		}
	}(ln)
	return s, nil
}
