package main

import (
	"encoding/json"
	"github.com/fhs/gompd/mpd"
	"reflect"
	"testing"
)

func TestConvStatus(t *testing.T) {
	candidates := []struct {
		status mpd.Attrs
		expect PlayerStatus
	}{
		{
			mpd.Attrs{},
			PlayerStatus{
				-1, false, false, false, false,
				"stopped", 0, 0.0, 0,
			},
		},
		{
			mpd.Attrs{
				"volume":  "100",
				"repeat":  "1",
				"random":  "0",
				"single":  "1",
				"consume": "0",
				"state":   "playing",
				"song":    "1",
				"elapsed": "10.1",
			},
			PlayerStatus{
				100, true, false, true, false,
				"playing", 1, 10.1, 0,
			},
		},
	}
	for _, c := range candidates {
		r := convStatus(c.status, 0)
		if !reflect.DeepEqual(c.expect, r) {
			jr, _ := json.Marshal(r)
			je, _ := json.Marshal(c.expect)
			t.Errorf(
				"unexpected. input: %s\nexpected: %s\nactual:   %s",
				songString(c.status),
				je, jr,
			)
		}
	}
}
