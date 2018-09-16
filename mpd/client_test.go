package mpd

import (
	"context"
	"fmt"
	"github.com/meiraka/vv/mpd/mpdtest"
	"testing"
	"time"
)

var (
	testDialer = Dialer{
		ReconnectionTimeout:  time.Second,
		HelthCheckInterval:   time.Second,
		ReconnectionInterval: time.Second,
	}
)

func TestDial(t *testing.T) {
	ts, _ := mpdtest.NewServer("OK MPD 0.19", map[string]string{
		"password 2434": "OK",
		"ping":          "OK",
		"close":         "OK",
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
	ts, _ := mpdtest.NewServer("OK MPD 0.19", map[string]string{
		"password 2434": "ACK [3@1] {password} error",
		"ping":          "OK",
		"close":         "OK",
	})
	defer ts.Close()
	c, err := testDialer.Dial("tcp", ts.URL, "2434")
	if g, w := fmt.Sprint(err), "ACK [3@1] {password} error"; g != w {
		t.Errorf("Dial got error %s; want %s", g, w)
	}
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	c.Close(ctx)
}
