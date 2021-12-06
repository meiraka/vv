package mpd

import (
	"context"
	"fmt"
)

// CommandListEqual compares command list a and b.
func CommandListEqual(a, b *CommandList) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if len(a.requests) != len(b.requests) {
		return false
	}
	for i := range a.requests {
		if a.requests[i] != b.requests[i] {
			return false
		}
	}
	return true
}

// CommandList represents Client commandlist.
type CommandList struct {
	requests []string
	commands []string
	parsers  []func(*conn) error
}

const responseListOK = "list_OK"

// Clear clears playlist
func (cl *CommandList) Clear() {
	req, _ := srequest("clear")
	cl.requests = append(cl.requests, req)
	cl.commands = append(cl.commands, "clear")
	cl.parsers = append(cl.parsers, func(c *conn) error {
		return parseEnd(c, responseListOK)
	})
}

// Add adds uri to playlist.
func (cl *CommandList) Add(uri string) {
	req, _ := srequest("add", uri)
	cl.requests = append(cl.requests, req)
	cl.commands = append(cl.commands, "add")
	cl.parsers = append(cl.parsers, func(c *conn) error {
		return parseEnd(c, responseListOK)
	})
}

// Play begins playing the playlist at song number pos.
func (cl *CommandList) Play(pos int) {
	req, _ := srequest("play", pos)
	cl.requests = append(cl.requests, req)
	cl.commands = append(cl.commands, "play")
	cl.parsers = append(cl.parsers, func(c *conn) error {
		return parseEnd(c, responseListOK)
	})
}

// ExecCommandList executes commandlist.
func (c *Client) ExecCommandList(ctx context.Context, cl *CommandList) error {
	defer func() {
		cl.requests = []string{}
		cl.commands = []string{}
		cl.parsers = []func(*conn) error{}
	}()
	return c.pool.Exec(ctx, func(conn *conn) error {
		if err := request(conn, "command_list_ok_begin"); err != nil {
			return err
		}
		for i := range cl.requests {
			if _, err := fmt.Fprint(conn, cl.requests[i]); err != nil {
				return err
			}
		}
		if err := request(conn, "command_list_end"); err != nil {
			return err
		}
		for i := range cl.parsers {
			if err := cl.parsers[i](conn); err != nil {
				return addCommandInfo(err, cl.commands[i])
			}
		}
		if err := parseEnd(conn, responseOK); err != nil {
			return addCommandInfo(err, "command_list_end")
		}
		return nil
	})
}
