package mpdtest

import (
	"bufio"
	"fmt"
	"log"
	"net"
)

type Server struct {
	ln  net.Listener
	URL string
}

func (s *Server) Close() error {
	return s.ln.Close()
}

func NewServer(firstResp string, resp map[string]string) (*Server, error) {
	ln, err := net.Listen("tcp", ":8092")
	if err != nil {
		log.Println("listen", err)
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
							_, err := fmt.Fprintln(conn, v)
							if err != nil {
								log.Println("write", err)
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
