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
	got, ok := readChan(ctx, t, c.Event())
	if want := "player"; !ok || got != want {
		t.Fatalf("got client %s, %v; want %s, true", got, ok, want)
	}
	ts.Expect(ctx, &mpdtest.WR{Read: "idle\n"})
	errs := make(chan error, 1)
	go func() { errs <- c.Close(ctx) }()

	ts.Expect(ctx, &mpdtest.WR{Read: "noidle\n", Write: "OK\n"})
	if err := <-errs; err != nil {
		t.Errorf("Close got error %v; want nil", err)
	}
	got, ok = readChan(ctx, t, c.Event())
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
