package mpd

import (
	"context"
	"testing"
	"time"

	"github.com/meiraka/vv/internal/mpd/mpdtest"
)

const (
	testTimeout = time.Second
)

func TestWatcher(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	wc, rc, ts, nil := mpdtest.NewChanServer("OK MPD 0.19")
	defer ts.Close()
	c, err := testDialer.NewWatcher("tcp", ts.URL, "")
	if err != nil {
		t.Fatalf("Dial got error %v; want nil", err)
	}
	if got, want := readChan(ctx, t, rc), "idle\n"; got != want {
		t.Fatalf("got server %s; want %s", got, want)
	}
	wc <- "changed: player\nOK\n"
	if got, want := readChan(ctx, t, c.C), "player"; got != want {
		t.Fatalf("got client %s; want %s", got, want)
	}
	if got, want := readChan(ctx, t, rc), "idle\n"; got != want {
		t.Fatalf("got server %s; want %s", got, want)
	}
	errs := make(chan error, 1)
	go func() { errs <- c.Close(ctx) }()
	if got, want := readChan(ctx, t, rc), "noidle\n"; got != want {
		t.Errorf("got server %s; want %s", got, want)
	}
	wc <- "OK\n"
	if got, want := readChan(ctx, t, rc), "close\n"; got != want {
		t.Errorf("got server %s; want %s", got, want)
	}
	if err := <-errs; err != nil {
		t.Errorf("Close got error %v; want nil", err)
	}

}

func readChan(ctx context.Context, t *testing.T, c <-chan string) (ret string) {
	t.Helper()
	select {
	case ret = <-c:
	case <-ctx.Done():
		t.Fatalf("read timeout %v", ctx.Err())
	}
	return
}
