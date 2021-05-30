package mpd

import (
	"errors"
	"fmt"
)

// Predefined error codes in https://github.com/MusicPlayerDaemon/MPD/blob/master/src/protocol/Ack.hxx
const (
	ErrNotList AckError = 1

	ErrArg        AckError = 2
	ErrPassword   AckError = 3
	ErrPermission AckError = 4
	ErrUnknown    AckError = 5

	ErrNoExist       AckError = 50
	ErrPlaylistMax   AckError = 51
	ErrSystem        AckError = 52
	ErrPlaylistLoad  AckError = 53
	ErrUpdateAlready AckError = 54
	ErrPlayerSync    AckError = 55
	ErrExist         AckError = 56
)

// AckError is the numeric value in CommandError.
type AckError int

func (a AckError) Error() string {
	switch a {
	case 1:
		return "ErrNotList"
	case 2:
		return "ErrArg"
	case 3:
		return "ErrPassword"
	case 4:
		return "ErrPermission"
	case 5:
		return "ErrUnknown"
	case 50:
		return "ErrNoExist"
	case 51:
		return "ErrPlaylistMax"
	case 52:
		return "ErrSystem"
	case 53:
		return "ErrPlaylistLoad"
	case 54:
		return "ErrUpdateAlready"
	case 55:
		return "ErrPlayerSync"
	case 56:
		return "ErrExist"
	}
	return ""
}

// CommandError represents mpd command error.
type CommandError struct {
	ID      AckError
	Index   int
	Command string
	Message string
}

func (f *CommandError) Error() string {
	if len(f.Command) == 0 {
		return fmt.Sprintf("mpd: %s", f.Message)
	}
	return fmt.Sprintf("mpd: %s: %s", f.Command, f.Message)
}

// Unwrap returns AckError.
func (f *CommandError) Unwrap() error {
	return f.ID
}

// Is returns true if pointer or all values are same.
func (f *CommandError) Is(target error) bool {
	if target == f {
		return true
	}
	var t *CommandError
	if !errors.As(target, &t) {
		return false
	}
	return f.ID == t.ID &&
		f.Index == t.Index &&
		f.Command == t.Command &&
		f.Message == t.Message
}
