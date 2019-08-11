package mpd

import (
	"reflect"
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
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("got %+v; want %+v", got, tt.want)
		}
	}

}
