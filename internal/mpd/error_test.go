package mpd

import (
	"errors"
	"testing"
)

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
