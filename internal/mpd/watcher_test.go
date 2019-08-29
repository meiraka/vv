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
	ts, err := mpdtest.NewServer("OK MPD 0.19")
	if err != nil {
		t.Fatalf("failed to create test server: %v", err)
	}
	defer ts.Close()
	c, err := testDialer.NewWatcher("tcp", ts.URL, "")
	if err != nil {
		t.Fatalf("Dial got error %v; want nil", err)
	}
	ts.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: player\nOK\n"})
	if got, want := readChan(ctx, t, c.C), "player"; got != want {
		t.Fatalf("got client %s; want %s", got, want)
	}
	ts.Expect(ctx, &mpdtest.WR{Read: "idle\n"})
	errs := make(chan error, 1)
	go func() { errs <- c.Close(ctx) }()

	ts.Expect(ctx, &mpdtest.WR{Read: "noidle\n", Write: "OK\n"})
	ts.Expect(ctx, &mpdtest.WR{Read: "close\n"})
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
