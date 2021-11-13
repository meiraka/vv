package mpd

import (
	"context"
	"fmt"
	"reflect"
)

// CommandListEqual compares command list a and b.
func CommandListEqual(a, b *CommandList) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return reflect.DeepEqual(a.commands, b.commands)
}

// CommandList represents Client commandlist.
type CommandList struct {
	commands [][]interface{}
	parser   []func(*conn) error
}

const responseListOK = "list_OK"

// Clear clears playlist
func (cl *CommandList) Clear() {
	cl.commands = append(cl.commands, []interface{}{"clear"})
	cl.parser = append(cl.parser, func(c *conn) error {
		return parseEnd(c, responseListOK)
	})
}

// Add adds uri to playlist.
func (cl *CommandList) Add(uri string) {
	cl.commands = append(cl.commands, []interface{}{"add", quote(uri)})
	cl.parser = append(cl.parser, func(c *conn) error {
		return parseEnd(c, responseListOK)
	})
}

// Play begins playing the playlist at song number pos.
func (cl *CommandList) Play(pos int) {
	cl.commands = append(cl.commands, []interface{}{"play", pos})
	cl.parser = append(cl.parser, func(c *conn) error {
		return parseEnd(c, responseListOK)
	})
}

// ExecCommandList executes commandlist.
func (c *Client) ExecCommandList(ctx context.Context, cl *CommandList) error {
	commands := append([][]interface{}{{"command_list_ok_begin"}}, cl.commands...)
	commands = append(commands, []interface{}{"command_list_end"})
	defer func() {
		cl.commands = [][]interface{}{}
		cl.parser = []func(*conn) error{}
	}()
	return c.pool.Exec(ctx, func(conn *conn) error {
		for i := range commands {
			if _, err := fmt.Fprintln(conn, commands[i]...); err != nil {
				return err
			}
		}
		for i := range cl.parser {
			if err := cl.parser[i](conn); err != nil {
				return addCommandInfo(err, cl.commands[i][0].(string))
			}
		}
		if err := parseEnd(conn, responseOK); err != nil {
			return addCommandInfo(err, "command_list_end")
		}
		return nil
	})
}
