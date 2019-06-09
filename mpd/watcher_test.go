package mpd

import (
	"context"
	"testing"
	"time"

	"github.com/meiraka/vv/mpd/mpdtest"
)

func TestWatcher(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ts, _ := mpdtest.NewServer("OK MPD 0.19", map[string]string{
		"password 2434": "OK",
		"idle":          "changed: player\nOK",
		"noidle":        "OK",
		"close":         "OK",
	})
	defer ts.Close()
	c, err := testDialer.NewWatcher("tcp", ts.URL, "2434")
	if err != nil {
		t.Fatalf("Dial got error %v; want nil", err)
	}
	for i := 0; i < 2; i++ {
		select {
		case got := <-c.C:
			if want := "player"; got != want {
				t.Errorf("got %s; want %s", got, want)
			}
		case <-ctx.Done():
			t.Fatalf("test timeout")
		}
	}
	if err := c.Close(ctx); err != nil {
		t.Errorf("Close got error %v; want nil", err)
	}

}

func TestWatcherNoIdle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	ts, _ := mpdtest.NewServer("OK MPD 0.19", map[string]string{
		"password 2434": "OK",
		"idle\nnoidle":  "OK",
		"close":         "OK",
	})
	defer ts.Close()
	c, err := testDialer.NewWatcher("tcp", ts.URL, "2434")
	if err != nil {
		t.Fatalf("Dial got error %v; want nil", err)
	}
	if err := c.Close(ctx); err != nil {
		t.Errorf("Close got error %v; want nil", err)
	}

}
