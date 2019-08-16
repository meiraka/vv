package mpd

import (
	"context"
	"reflect"
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
		mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "close\n"})
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

func TestCommandListCommandError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	w, r, ts, _ := mpdtest.NewServer("OK MPD 0.19")
	go func() {
		mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "password 2434\n", Write: "OK\n"})
		mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "command_list_ok_begin\n"})
		mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "clear\n"})
		mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "play 0\n"})
		mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "add \"/foo/bar\"\n"})
		mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "command_list_end\n", Write: "list_OK\nACK [2@1] {} Bad song index\n"})
		mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "close\n"})
	}()
	defer ts.Close()
	c, err := testDialer.Dial("tcp", ts.URL, "2434")
	if err != nil {
		t.Fatalf("Dial got error %v; want nil", err)
	}
	cl := c.BeginCommandList()
	cl.Clear()
	cl.Play(0)
	cl.Add("/foo/bar")
	if err, want := cl.End(ctx), newCommandError("ACK [2@1] {} Bad song index"); !reflect.DeepEqual(err, want) {
		t.Errorf("CommandList got error %v; want %v", err, want)
	}
	if err := c.Close(ctx); err != nil {
		t.Errorf("Close got error %v; want nil", err)
	}
}

func TestCommandListNetworkError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	w, r, ts, _ := mpdtest.NewServer("OK MPD 0.19")
	go func() {
		mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "password 2434\n", Write: "OK\n"})
		ts.Disconnect(ctx)
		mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "password 2434\n", Write: "OK\n"})
		mpdtest.Expect(ctx, w, r, &mpdtest.WR{Read: "close\n"})
	}()
	defer ts.Close()
	c, err := testDialer.Dial("tcp", ts.URL, "2434")
	if err != nil {
		t.Fatalf("Dial got error %v; want nil", err)
	}
	cl := c.BeginCommandList()
	cl.Clear()
	cl.Play(0)
	cl.Add("/foo/bar")
	if err := cl.End(ctx); err == nil {
		t.Error("CommandList got nil; want non nil error at network error")
	}
	if err := c.Close(ctx); err != nil {
		t.Errorf("Close got error %v; want nil", err)
	}
}
