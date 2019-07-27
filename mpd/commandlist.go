package mpd

import (
	"context"
)

// BeginCommandList creates a new CommandList
func (c *Client) BeginCommandList() *CommandList {
	return &CommandList{
		c:        c,
		commands: [][]interface{}{},
		parser:   []func(*conn) error{},
	}
}

// CommandList represents Client commandlist.
type CommandList struct {
	c        *Client
	commands [][]interface{}
	parser   []func(*conn) error
}

// Clear clears playlist
func (cl *CommandList) Clear() {
	cl.commands = append(cl.commands, []interface{}{"clear"})
	cl.parser = append(cl.parser, func(c *conn) error {
		return c.ReadEnd("list_OK")
	})
}

// Add adds uri to playlist.
func (cl *CommandList) Add(uri string) {
	cl.commands = append(cl.commands, []interface{}{"add", quote(uri)})
	cl.parser = append(cl.parser, func(c *conn) error {
		return c.ReadEnd("list_OK")
	})
}

// Play begins playing the playlist at song number pos.
func (cl *CommandList) Play(pos int) {
	cl.commands = append(cl.commands, []interface{}{"play", pos})
	cl.parser = append(cl.parser, func(c *conn) error {
		return c.ReadEnd("list_OK")
	})
}

// End executes commandlist.
func (cl *CommandList) End(ctx context.Context) error {
	commands := append([][]interface{}{{"command_list_ok_begin"}}, cl.commands...)
	commands = append(commands, []interface{}{"command_list_end"})
	defer func() {
		cl.commands = [][]interface{}{}
		cl.parser = []func(*conn) error{}
	}()
	return cl.c.conn.Exec(ctx, func(conn *conn) error {
		for i := range commands {
			if _, err := conn.Writeln(commands[i]...); err != nil {
				return err
			}
		}
		for i := range cl.parser {
			if err := cl.parser[i](conn); err != nil {
				return err
			}
		}
		return conn.ReadEnd("OK")
	})
}
