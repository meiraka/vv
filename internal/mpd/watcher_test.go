package mpd

import (
	"context"
	"testing"
	"time"

	"github.com/meiraka/vv/internal/mpd/mpdtest"
)

const (
	testTimeout = 10 * time.Second
)

func TestWatcher(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	ts := mpdtest.NewServer("OK MPD 0.19")
	defer ts.Close()
	w, err := NewWatcher("tcp", ts.URL,
		&WatcherOptions{Timeout: testTimeout, ReconnectionInterval: time.Millisecond})
	if err != nil {
		t.Fatalf("Dial got error %v; want nil", err)
	}
	ts.Expect(ctx, &mpdtest.WR{Read: "idle\n", Write: "changed: playlist\nchanged: player\nOK\n"})
	got, ok := readChan(ctx, t, w.Event())
	if want := "playlist"; !ok || got != want {
		t.Fatalf("got client %s, %v; want %s, true", got, ok, want)
	}
	got, ok = readChan(ctx, t, w.Event())
	if want := "player"; !ok || got != want {
		t.Fatalf("got client %s, %v; want %s, true", got, ok, want)
	}
	ts.Expect(ctx, &mpdtest.WR{Read: "idle\n"})
	errs := make(chan error, 1)
	go func() { errs <- w.Close(ctx) }()

	ts.Expect(ctx, &mpdtest.WR{Read: "noidle\n", Write: "OK\n"})
	if err := <-errs; err != nil {
		t.Errorf("Close got error %v; want nil", err)
	}
	got, ok = readChan(ctx, t, w.Event())
	if ok {
		t.Errorf("got \"%s\", %v; want \"\", false", got, ok)
	}

}

func readChan(ctx context.Context, t *testing.T, c <-chan string) (ret string, ok bool) {
	t.Helper()
	select {
	case ret, ok = <-c:
	case <-ctx.Done():
		t.Fatalf("read timeout %v", ctx.Err())
	}
	return
}
