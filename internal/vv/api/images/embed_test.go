package images

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"

	"github.com/meiraka/vv/internal/mpd"
	"github.com/meiraka/vv/internal/mpd/mpdtest"
)

func TestEmbed(t *testing.T) {
	ts := mpdtest.NewServer("OK MPD 0.19")
	defer ts.Close()
	c, err := mpd.Dial("tcp", ts.URL, nil)
	if err != nil {
		t.Fatalf("dial got err: %v", err)
	}
	testDir, err := os.MkdirTemp(".", "testdata")
	if err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	api, err := NewEmbed("/api/images", c, testDir)
	if err != nil {
		t.Fatalf("failed to initialize cover.Embed: %v", err)
	}
	defer api.Close()
	for _, tt := range []map[string][]string{
		{"Album": {"foo"}, "file": {"assets/test.flac"}},
		{"Album": {"notfound"}, "file": {"notfound/test.flac"}},
	} {
		t.Run(fmt.Sprint(tt), func(t *testing.T) {
			covers, ok := api.GetURLs(tt)
			if len(covers) != 0 || ok {
				t.Errorf(`GetURLs("%s") = %v %v; want nil, false`, tt, covers, ok)
			}
		})
	}
}

func TestEmbedUpdate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	ts := mpdtest.NewServer("OK MPD 0.19")
	defer ts.Close()
	go func() { ts.Expect(ctx, &mpdtest.WR{Read: "commands\n", Write: "command: readpicture\nOK\n"}) }()
	c, err := mpd.Dial("tcp", ts.URL, &mpd.ClientOptions{Timeout: testTimeout, CacheCommandsResult: true})
	if err != nil {
		t.Fatalf("dial got err: %v", err)
	}
	testDir, err := os.MkdirTemp(".", "testdata")
	if err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	png1 := readFile(t, filepath.Join("..", "..", "assets", "app.png"))
	for _, tt := range []struct {
		label      string
		song       map[string][]string
		mpd        []*mpdtest.WR
		err        error
		indexed    bool
		query      []url.Values
		respHeader []http.Header
		respBinary [][]byte
	}{
		{
			label:   "unknown error",
			song:    map[string][]string{"Album": {"error"}, "file": {"error/test.flac"}},
			mpd:     []*mpdtest.WR{{Read: `readpicture "error/test.flac" 0` + "\n", Write: "ACK [5@0] {readpicture} Unknown error\n"}},
			err:     &mpd.CommandError{ID: 5, Index: 0, Command: "readpicture", Message: "Unknown error"},
			indexed: true,
		},
		{
			label:   "unknown error(indexed: no mpd request)",
			song:    map[string][]string{"Album": {"error"}, "file": {"error/test.flac"}},
			indexed: true,
		},
		{
			label:   "not found",
			song:    map[string][]string{"Album": {"notfound"}, "file": {"notfound/test.flac"}},
			mpd:     []*mpdtest.WR{{Read: `readpicture "notfound/test.flac" 0` + "\n", Write: "OK\n"}},
			indexed: true,
		},
		{
			label:   "not found(indexed: no mpd request)",
			song:    map[string][]string{"Album": {"notfound"}, "file": {"notfound/test.flac"}},
			indexed: true,
		},
		{
			label:   "no tags",
			song:    map[string][]string{"file": {"notag/test.flac"}},
			indexed: true,
		},
		{
			label:      "found",
			song:       map[string][]string{"Album": {"foo"}, "file": {"foo/bar.flac"}},
			mpd:        []*mpdtest.WR{{Read: `readpicture "foo/bar.flac" 0` + "\n", Write: fmt.Sprintf("size: %d\nbinary: %d\n%s\nOK\n", len(png1), len(png1), png1)}},
			indexed:    true,
			query:      []url.Values{{"v": {"0"}}},
			respBinary: [][]byte{png1},
			respHeader: []http.Header{{"Content-Type": {"image/png"}, "Cache-Control": {"max-age=31536000"}}},
		},
		{
			label:      "found(indexed: no mpd request)",
			song:       map[string][]string{"Album": {"foo"}, "file": {"foo/bar.flac"}},
			indexed:    true,
			query:      []url.Values{{"v": {"0"}}},
			respBinary: [][]byte{png1},
			respHeader: []http.Header{{"Content-Type": {"image/png"}, "Cache-Control": {"max-age=31536000"}}},
		},
	} {
		t.Run(tt.label, func(t *testing.T) {
			var wg sync.WaitGroup
			defer wg.Wait()
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := range tt.mpd {
					if err := ts.Expect(ctx, tt.mpd[i]); err != nil {
						t.Errorf("mpd: %v", err)
					}
				}
			}()
			api, err := NewEmbed("/api/images", c, testDir)
			if err != nil {
				t.Fatalf("failed to initialize cover.Embed: %v", err)
			}
			defer api.Close()
			if err := api.Update(ctx, tt.song); !errors.Is(err, tt.err) {
				t.Errorf("Update(ctx, %v) = %v; want %v", tt.song, err, tt.err)
			}
			covers, ok := api.GetURLs(tt.song)
			var queries []url.Values
			for _, cover := range covers {
				u, err := url.Parse(cover)
				if err != nil {
					t.Fatalf("GetURLs(%v) = %v, %v; url parse error: %v", tt.song, covers, ok, err)
				}
				queries = append(queries, u.Query())
			}
			if !reflect.DeepEqual(queries, tt.query) || ok != tt.indexed {
				t.Errorf("GetURLs(%v) = %v, %v; want %v, %v", tt.song, queries, ok, tt.query, tt.indexed)
			}
			for i := range covers {
				cover := covers[i]
				req := httptest.NewRequest("GET", cover, nil)
				w := httptest.NewRecorder()
				api.ServeHTTP(w, req)
				resp := w.Result()
				if resp.StatusCode != 200 {
					t.Errorf("%s: got status %d; want 200", cover, resp.StatusCode)
				}
				for k, v := range tt.respHeader[i] {
					if !reflect.DeepEqual(resp.Header[k], v) {
						t.Errorf("%s: got header %s %v; want %v", k, cover, resp.Header[k], v)
					}
				}
				got, _ := io.ReadAll(resp.Body)
				if !reflect.DeepEqual(got, tt.respBinary[i]) {
					t.Errorf("%s: got invalid binary response", cover)
				}
			}
		})
	}
}

func TestEmbedRescan(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	ts := mpdtest.NewServer("OK MPD 0.19")
	defer ts.Close()
	go func() { ts.Expect(ctx, &mpdtest.WR{Read: "commands\n", Write: "command: readpicture\nOK\n"}) }()
	c, err := mpd.Dial("tcp", ts.URL, &mpd.ClientOptions{Timeout: testTimeout, CacheCommandsResult: true})
	if err != nil {
		t.Fatalf("dial got err: %v", err)
	}
	testDir, err := os.MkdirTemp(".", "testdata")
	if err != nil {
		t.Fatalf("failed to create test dir: %v", err)
	}
	defer os.RemoveAll(testDir)

	png1 := readFile(t, filepath.Join("..", "..", "assets", "app.png"))
	png2 := readFile(t, filepath.Join("..", "..", "assets", "app-black.png"))
	for _, tt := range []struct {
		label      string
		song       map[string][]string
		reqid      string
		mpd        []*mpdtest.WR
		err        error
		indexed    bool
		query      []url.Values
		respHeader []http.Header
		respBinary [][]byte
	}{
		{
			label: "unknown error",
			song:  map[string][]string{"Album": {"error"}, "file": {"error/test.flac"}},
			mpd:   []*mpdtest.WR{{Read: `readpicture "error/test.flac" 0` + "\n", Write: "ACK [5@0] {readpicture} Unknown error\n"}},
			err:   &mpd.CommandError{ID: 5, Index: 0, Command: "readpicture", Message: "Unknown error"},
			reqid: "1",
		},
		{
			label: "unknown error(same request id)",
			song:  map[string][]string{"Album": {"error"}, "file": {"error/test.flac"}},
			mpd:   []*mpdtest.WR{{Read: `readpicture "error/test.flac" 0` + "\n", Write: "ACK [5@0] {readpicture} Unknown error\n"}},
			err:   &mpd.CommandError{ID: 5, Index: 0, Command: "readpicture", Message: "Unknown error"},
			reqid: "1",
		},
		{
			label:   "not found",
			song:    map[string][]string{"Album": {"notfound"}, "file": {"notfound/test.flac"}},
			reqid:   "1",
			mpd:     []*mpdtest.WR{{Read: `readpicture "notfound/test.flac" 0` + "\n", Write: "OK\n"}},
			indexed: true,
		},
		{
			label:   "not found(same request id: indexed: no mpd request)",
			song:    map[string][]string{"Album": {"notfound"}, "file": {"notfound/test.flac"}},
			reqid:   "1",
			indexed: true,
		},
		{
			label:   "no tags",
			song:    map[string][]string{"file": {"notag/test.flac"}},
			indexed: true,
		},
		{
			label:   "not found(different request id)",
			song:    map[string][]string{"Album": {"notfound"}, "file": {"notfound/test.flac"}},
			reqid:   "2", // != "1"
			mpd:     []*mpdtest.WR{{Read: `readpicture "notfound/test.flac" 0` + "\n", Write: "OK\n"}},
			indexed: true,
		},
		{
			label:      "found",
			song:       map[string][]string{"Album": {"foo"}, "file": {"foo/bar.flac"}},
			reqid:      "1",
			mpd:        []*mpdtest.WR{{Read: `readpicture "foo/bar.flac" 0` + "\n", Write: fmt.Sprintf("size: %d\nbinary: %d\n%s\nOK\n", len(png1), len(png1), png1)}},
			indexed:    true,
			query:      []url.Values{{"v": {"0"}}},
			respBinary: [][]byte{png1},
			respHeader: []http.Header{{"Content-Type": {"image/png"}, "Cache-Control": {"max-age=31536000"}}},
		},
		{
			label:      "found(same request id: indexed: no mpd request)",
			song:       map[string][]string{"Album": {"foo"}, "file": {"foo/bar.flac"}},
			reqid:      "1",
			indexed:    true,
			query:      []url.Values{{"v": {"0"}}},
			respBinary: [][]byte{png1},
			respHeader: []http.Header{{"Content-Type": {"image/png"}, "Cache-Control": {"max-age=31536000"}}},
		},
		{
			label:      "found(different request id)",
			song:       map[string][]string{"Album": {"foo"}, "file": {"foo/bar.flac"}},
			reqid:      "2", // != "1"
			mpd:        []*mpdtest.WR{{Read: `readpicture "foo/bar.flac" 0` + "\n", Write: fmt.Sprintf("size: %d\nbinary: %d\n%s\nOK\n", len(png1), len(png1), png1)}},
			indexed:    true,
			query:      []url.Values{{"v": {"0"}}},
			respBinary: [][]byte{png1},
			respHeader: []http.Header{{"Content-Type": {"image/png"}, "Cache-Control": {"max-age=31536000"}}},
		},
		{
			label:      "found(different request id, different binary)",
			song:       map[string][]string{"Album": {"foo"}, "file": {"foo/bar.flac"}},
			reqid:      "3", // != "2"
			mpd:        []*mpdtest.WR{{Read: `readpicture "foo/bar.flac" 0` + "\n", Write: fmt.Sprintf("size: %d\nbinary: %d\n%s\nOK\n", len(png2), len(png2), png2)}},
			indexed:    true,
			query:      []url.Values{{"v": {"1"}}},
			respBinary: [][]byte{png2},
			respHeader: []http.Header{{"Content-Type": {"image/png"}, "Cache-Control": {"max-age=31536000"}}},
		},
	} {
		t.Run(tt.label, func(t *testing.T) {
			var wg sync.WaitGroup
			defer wg.Wait()
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := range tt.mpd {
					if err := ts.Expect(ctx, tt.mpd[i]); err != nil {
						t.Errorf("mpd: %v", err)
					}
				}
			}()
			api, err := NewEmbed("/api/images", c, testDir)
			if err != nil {
				t.Fatalf("failed to initialize cover.Embed: %v", err)
			}
			defer api.Close()
			if err := api.Rescan(ctx, tt.song, tt.reqid); !errors.Is(err, tt.err) {
				t.Errorf("Rescan(ctx, %v, %s) = %v; want %v", tt.song, tt.reqid, err, tt.err)
			}
			covers, ok := api.GetURLs(tt.song)
			var queries []url.Values
			for _, cover := range covers {
				u, err := url.Parse(cover)
				if err != nil {
					t.Fatalf("GetURLs(%v) = %v, %v; url parse error: %v", tt.song, covers, ok, err)
				}
				queries = append(queries, u.Query())
			}
			if !reflect.DeepEqual(queries, tt.query) || ok != tt.indexed {
				t.Errorf("GetURLs(%v) = %v, %v; want %v, %v", tt.song, queries, ok, tt.query, tt.indexed)
			}
			for i := range covers {
				cover := covers[i]
				req := httptest.NewRequest("GET", cover, nil)
				w := httptest.NewRecorder()
				api.ServeHTTP(w, req)
				resp := w.Result()
				if resp.StatusCode != 200 {
					t.Errorf("%s: got status %d; want 200", cover, resp.StatusCode)
				}
				for k, v := range tt.respHeader[i] {
					if !reflect.DeepEqual(resp.Header[k], v) {
						t.Errorf("%s: got header %s %v; want %v", k, cover, resp.Header[k], v)
					}
				}
				got, _ := io.ReadAll(resp.Body)
				if !reflect.DeepEqual(got, tt.respBinary[i]) {
					t.Errorf("%s: got invalid binary response", cover)
				}
			}
		})
	}
}
