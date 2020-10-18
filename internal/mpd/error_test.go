package mpd

import (
	"errors"
	"testing"
)

func TestNewCommandError(t *testing.T) {
	for _, tt := range []struct {
		in   string
		want error
	}{
		{
			in:   `ACK [50@1] {play} song doesn't exist: "10240"`,
			want: &CommandError{ID: 50, Index: 1, Command: "play", Message: `song doesn't exist: "10240"`},
		},
	} {
		got := newCommandError(tt.in)
		if !errors.Is(got, tt.want) {
			t.Errorf("got %+v; want %+v", got, tt.want)
		}
	}
}

func TestAckError(t *testing.T) {
	for _, tt := range []struct {
		cmdErr error
		ackErr error
	}{
		{cmdErr: &CommandError{ID: 1}, ackErr: ErrNotList},
		{cmdErr: &CommandError{ID: 50}, ackErr: ErrNoExist},
	} {
		if !errors.Is(tt.cmdErr, tt.ackErr) {
			t.Errorf("%+v != %+v", tt.cmdErr, tt.ackErr)
		}
	}
}
