package mpdtest

import (
	"bufio"
	"fmt"
	"net"
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
	ln, err := net.Listen("tcp", ":8092")
	if err != nil {
		return nil, err
	}
	s := &Server{
		ln:  ln,
		URL: "localhost:8092",
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

// NewChanServer creates new mpd mock Server for idle command.
func NewChanServer(firstResp string) (chan string, <-chan string, *Server, error) {
	ln, err := net.Listen("tcp", ":8092")
	if err != nil {
		return nil, nil, nil, err
	}
	s := &Server{
		ln:  ln,
		URL: "localhost:8092",
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
