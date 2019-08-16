package mpd

import (
	"context"
	"testing"
	"time"

	"github.com/meiraka/vv/internal/mpd/mpdtest"
)

func TestCommandList(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	w, r, ts, _ := mpdtest.NewServer("OK MPD 0.19")
	go func() {
		mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "password 2434\n", Write: "OK\n"})
		mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "command_list_ok_begin\n"})
		mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "clear\n"})
		mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "add \"/foo/bar\"\n"})
		mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "command_list_end\n", Write: "list_OK\nlist_OK\nOK\n"})
		mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "close", Write: "OK\n"})
	}()
	defer ts.Close()
	c, err := testDialer.Dial("tcp", ts.URL, "2434")
	if err != nil {
		t.Fatalf("Dial got error %v; want nil", err)
	}
	cl := c.BeginCommandList()
	cl.Clear()
	cl.Add("/foo/bar")
	if err := cl.End(ctx); err != nil {
		t.Errorf("CommandList got error %v; want nil", err)
	}
	if err := c.Close(ctx); err != nil {
		t.Errorf("Close got error %v; want nil", err)
	}
}
