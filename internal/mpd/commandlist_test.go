package mpd

import (
	"context"
	"testing"
	"time"

	"github.com/meiraka/vv/internal/mpd/mpdtest"
)

func TestCommandList(t *testing.T) {
	ts, _ := mpdtest.NewServer("OK MPD 0.19", map[string]string{
		"password 2434": "OK\n",
		"ping":          "OK\n",
		"command_list_ok_begin\nclear\nadd \"/foo/bar\"\ncommand_list_end": "list_OK\nlist_OK\nOK\n",
		"close": "OK\n",
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
