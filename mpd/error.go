package mpd

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// CommandError represents mpd command error.
type CommandError struct {
	ID      int
	Index   int
	Command string
	Message string
}

func newCommandError(s string) error {
	u := s[5:]
	at := strings.IndexRune(u, '@')
	if at < 0 {
		return errors.New(s)
	}
	id, err := strconv.Atoi(u[:at])
	if err != nil {
		return errors.New(s)
	}
	b := strings.IndexRune(u, ']')
	if b < 0 {
		return errors.New(s)
	}
	index, err := strconv.Atoi(u[at+1 : b])
	if err != nil {
		return errors.New(s)
	}
	bb := strings.IndexRune(u, '}')
	if bb < 0 {
		return errors.New(s)
	}
	return &CommandError{
		ID:      id,
		Index:   index,
		Command: u[b+3 : bb],
		Message: u[bb+2:],
	}
}

func (f *CommandError) Error() string {
	return fmt.Sprintf("mpd: %s: %s", f.Command, f.Message)
}
