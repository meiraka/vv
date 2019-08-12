package mpd

import (
	"context"
	"testing"
	"time"

	"github.com/meiraka/vv/internal/mpd/mpdtest"
)

func TestCommandList(t *testing.T) {
	ts, _ := mpdtest.NewEventServer("OK MPD 0.19", []*mpdtest.WR{
		{Read: "password 2434\n", Write: "OK\n"},
		{Read: "command_list_ok_begin\n"},
		{Read: "clear\n"},
		{Read: "add \"/foo/bar\"\n"},
		{Read: "command_list_end\n", Write: "list_OK\nlist_OK\nOK\n"},
		{Read: "close", Write: "OK\n"},
	})
	defer ts.Close()
	c, err := testDialer.Dial("tcp", ts.URL, "2434")
	if err != nil {
		t.Fatalf("Dial got error %v; want nil", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
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
