package mpd

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/meiraka/vv/internal/mpd/mpdtest"
)

var (
	testDialer = Dialer{
		ReconnectionTimeout:  time.Second,
		HealthCheckInterval:  time.Second,
		ReconnectionInterval: time.Second,
	}
)

func TestDial(t *testing.T) {
	ts, _ := mpdtest.NewEventServer("OK MPD 0.19", []*mpdtest.WR{
		{Read: "password 2434\n", Write: "OK\n"},
		{Read: "close"},
	})
	defer ts.Close()
	c, err := testDialer.Dial("tcp", ts.URL, "2434")
	if err != nil {
		t.Fatalf("Dial got error %v; want nil", err)
	}
	if g, w := c.Version(), "0.19"; g != w {
		t.Errorf("Version() got `%s`; want `%s`", g, w)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := c.Close(ctx); err != nil {
		t.Errorf("Close got error %v; want nil", err)
	}
}

func TestDialPasswordError(t *testing.T) {
	ts, _ := mpdtest.NewEventServer("OK MPD 0.19", []*mpdtest.WR{
		{Read: "password 2434\n", Write: "ACK [3@1] {password} error\n"},
		{Read: "close"},
	})
	defer ts.Close()
	c, err := testDialer.Dial("tcp", ts.URL, "2434")
	want := &CommandError{ID: 3, Index: 1, Command: "password", Message: "error"}
	if !reflect.DeepEqual(err, want) {
		t.Errorf("Dial got error %v; want %v", err, want)
	}
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	c.Close(ctx)
}