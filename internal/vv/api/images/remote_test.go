package images

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/meiraka/vv/internal/mpd"
	"github.com/meiraka/vv/internal/mpd/mpdtest"
)

const testTimeout = time.Second

func TestRemote(t *testing.T) {
	svr, err := mpdtest.NewServer("OK MPD 0.19")
	if err != nil {
		t.Fatalf("failed to create mpd test server: %v", err)
	}
	defer svr.Close()
	c, err := mpd.Dial("tcp", svr.URL, nil)
	if err != nil {
		t.Fatalf("dial got err: %v", err)
	}
	path, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to find current directory for test dir: %v", err)
	}
	testDir := filepath.Join(path, "testdata")
	if err := os.RemoveAll(testDir); err != nil {
		t.Fatalf("failed to cleanup test dir")
	}
	defer os.RemoveAll(testDir)

	api, err := NewRemote("/api/images", c, testDir)
	if err != nil {
		t.Fatalf("failed to initialize cover.Remote: %v", err)
	}
	defer api.Close()
	for _, tt := range []map[string][]string{
		{"file": {"assets/test.flac"}},
		{"file": {"notfound/test.flac"}},
	} {
		t.Run(fmt.Sprint(tt), func(t *testing.T) {
			covers, ok := api.GetURLs(tt)
			if len(covers) != 0 || ok {
				t.Errorf(`GetURLs("%s") = %v %v; want nil, false`, tt, covers, ok)
			}
		})
	}
}

func TestRemoteUpdate(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	svr, err := mpdtest.NewServer("OK MPD 0.19")
	if err != nil {
		t.Fatalf("failed to create mpd test server: %v", err)
	}
	defer svr.Close()
	go func() { svr.Expect(ctx, &mpdtest.WR{Read: "commands\n", Write: "commands: albumart\nOK\n"}) }()
	c, err := mpd.Dial("tcp", svr.URL, &mpd.ClientOptions{Timeout: testTimeout, CacheCommandsResult: true})
	if err != nil {
		t.Fatalf("dial got err: %v", err)
	}
	path, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to find current directory for test dir: %v", err)
	}
	testDir := filepath.Join(path, "testdata")
	if err := os.RemoveAll(testDir); err != nil {
		t.Fatalf("failed to cleanup test dir")
	}
	defer os.RemoveAll(testDir)
	t.Run("", func(t *testing.T) {

	})

	png1 := readFile(t, filepath.Join(path, "..", "..", "..", "..", "assets", "app.png"))
	for _, tt := range []struct {
		label      string
		song       map[string][]string
		mpd        []*mpdtest.WR
		hasCover   bool
		query      url.Values
		respHeader http.Header
		respBinary []byte
	}{
		{
			label:    "not found",
			song:     map[string][]string{"file": {"notfound/test.flac"}},
			mpd:      []*mpdtest.WR{{Read: `albumart "notfound/test.flac" 0` + "\n", Write: "ACK [50@0] {albumart} No file exists\n"}},
			hasCover: false,
		},
		{
			label:    "not found(no mpd request)",
			song:     map[string][]string{"file": {"notfound/test.flac"}},
			hasCover: false,
		},
		{
			label:      "found",
			song:       map[string][]string{"file": {"foo/bar.flac"}},
			mpd:        []*mpdtest.WR{{Read: `albumart "foo/bar.flac" 0` + "\n", Write: fmt.Sprintf("size: %d\nbinary: %d\n%s\nOK\n", len(png1), len(png1), png1)}},
			hasCover:   true,
			query:      url.Values{"v": {"0"}},
			respBinary: png1,
			respHeader: http.Header{"Content-Type": {"image/png"}, "Cache-Control": {"max-age=31536000"}},
		},
		{
			label:      "found(no mpd request)",
			song:       map[string][]string{"file": {"foo/bar.flac"}},
			hasCover:   true,
			query:      url.Values{"v": {"0"}},
			respBinary: png1,
			respHeader: http.Header{"Content-Type": {"image/png"}, "Cache-Control": {"max-age=31536000"}},
		},
	} {
		t.Run(tt.label, func(t *testing.T) {
			go func() {
				for i := range tt.mpd {
					svr.Expect(ctx, tt.mpd[i])
				}
			}()
			api, err := NewRemote("/api/images", c, testDir)
			if err != nil {
				t.Fatalf("failed to initialize cover.Remote: %v", err)
			}
			defer api.Close()
			if err := api.Update(ctx, tt.song); err != nil {
				t.Fatalf("Update: %v", err)
			}
			covers, ok := api.GetURLs(tt.song)
			if !ok {
				t.Errorf("cover %s is not indexed", tt.song)
			}
			if len(covers) == 0 && tt.hasCover {
				t.Fatalf("got no covers; want 1 cover")
			}
			if len(covers) == 1 && !tt.hasCover {
				t.Fatalf("got 1 cover %v; want no covers", covers)
			}
			if len(covers) == 0 {
				return
			}
			cover := covers[0]
			u, err := url.Parse(cover)
			if err != nil {
				t.Fatalf("failed to parse url %s: %v", cover, err)
			}
			if !reflect.DeepEqual(u.Query(), tt.query) {
				t.Errorf("got query %+v; want %+v", u.Query(), tt.query)
			}
			req := httptest.NewRequest("GET", cover, nil)
			w := httptest.NewRecorder()
			api.ServeHTTP(w, req)
			resp := w.Result()
			if resp.StatusCode != 200 {
				t.Errorf("got status %d; want 200", resp.StatusCode)
			}
			for k, v := range tt.respHeader {
				if !reflect.DeepEqual(resp.Header[k], v) {
					t.Errorf("got header %s %v; want %v", k, resp.Header[k], v)
				}
			}
			got, _ := ioutil.ReadAll(resp.Body)
			if !reflect.DeepEqual(got, tt.respBinary) {
				t.Errorf("got invalid binary response")
			}
		})
	}
}

func TestRemoteRescan(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	svr, err := mpdtest.NewServer("OK MPD 0.19")
	if err != nil {
		t.Fatalf("failed to create mpd test server: %v", err)
	}
	defer svr.Close()
	go func() { svr.Expect(ctx, &mpdtest.WR{Read: "commands\n", Write: "commands: albumart\nOK\n"}) }()
	c, err := mpd.Dial("tcp", svr.URL, &mpd.ClientOptions{Timeout: testTimeout, CacheCommandsResult: true})
	if err != nil {
		t.Fatalf("dial got err: %v", err)
	}
	path, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to find current directory for test dir: %v", err)
	}
	testDir := filepath.Join(path, "testdata")
	if err := os.RemoveAll(testDir); err != nil {
		t.Fatalf("failed to cleanup test dir")
	}
	defer os.RemoveAll(testDir)
	t.Run("", func(t *testing.T) {

	})

	png1 := readFile(t, filepath.Join(path, "..", "..", "..", "..", "assets", "app.png"))
	png2 := readFile(t, filepath.Join(path, "..", "..", "..", "..", "assets", "app-black.png"))
	for _, tt := range []struct {
		label      string
		song       map[string][]string
		reqid      string
		mpd        []*mpdtest.WR
		hasCover   bool
		query      url.Values
		respHeader http.Header
		respBinary []byte
	}{
		{
			label:    "not found",
			song:     map[string][]string{"file": {"notfound/test.flac"}},
			reqid:    "1",
			mpd:      []*mpdtest.WR{{Read: `albumart "notfound/test.flac" 0` + "\n", Write: "ACK [50@0] {albumart} No file exists\n"}},
			hasCover: false,
		},
		{
			label:    "not found(same request id)",
			song:     map[string][]string{"file": {"notfound/test.flac"}},
			reqid:    "1",
			hasCover: false,
		},
		{
			label:    "not found(different request id)",
			song:     map[string][]string{"file": {"notfound/test.flac"}},
			reqid:    "2", // != "1"
			mpd:      []*mpdtest.WR{{Read: `albumart "notfound/test.flac" 0` + "\n", Write: "ACK [50@0] {albumart} No file exists\n"}},
			hasCover: false,
		},
		{
			label:      "found",
			song:       map[string][]string{"file": {"foo/bar.flac"}},
			reqid:      "1",
			mpd:        []*mpdtest.WR{{Read: `albumart "foo/bar.flac" 0` + "\n", Write: fmt.Sprintf("size: %d\nbinary: %d\n%s\nOK\n", len(png1), len(png1), png1)}},
			hasCover:   true,
			query:      url.Values{"v": {"0"}},
			respBinary: png1,
			respHeader: http.Header{"Content-Type": {"image/png"}, "Cache-Control": {"max-age=31536000"}},
		},
		{
			label:      "found(same request id)",
			song:       map[string][]string{"file": {"foo/bar.flac"}},
			reqid:      "1",
			hasCover:   true,
			query:      url.Values{"v": {"0"}},
			respBinary: png1,
			respHeader: http.Header{"Content-Type": {"image/png"}, "Cache-Control": {"max-age=31536000"}},
		},
		{
			label:      "found(different request id)",
			song:       map[string][]string{"file": {"foo/bar.flac"}},
			reqid:      "2", // != "1"
			mpd:        []*mpdtest.WR{{Read: `albumart "foo/bar.flac" 0` + "\n", Write: fmt.Sprintf("size: %d\nbinary: %d\n%s\nOK\n", len(png1), len(png1), png1)}},
			hasCover:   true,
			query:      url.Values{"v": {"0"}},
			respBinary: png1,
			respHeader: http.Header{"Content-Type": {"image/png"}, "Cache-Control": {"max-age=31536000"}},
		},
		{
			label:      "found(different request id, different binary)",
			song:       map[string][]string{"file": {"foo/bar.flac"}},
			reqid:      "3", // != "2"
			mpd:        []*mpdtest.WR{{Read: `albumart "foo/bar.flac" 0` + "\n", Write: fmt.Sprintf("size: %d\nbinary: %d\n%s\nOK\n", len(png2), len(png2), png2)}},
			hasCover:   true,
			query:      url.Values{"v": {"1"}},
			respBinary: png2,
			respHeader: http.Header{"Content-Type": {"image/png"}, "Cache-Control": {"max-age=31536000"}},
		},
	} {
		t.Run(tt.label, func(t *testing.T) {
			go func() {
				for i := range tt.mpd {
					svr.Expect(ctx, tt.mpd[i])
				}
			}()
			api, err := NewRemote("/api/images", c, testDir)
			if err != nil {
				t.Fatalf("failed to initialize cover.Remote: %v", err)
			}
			defer api.Close()
			if err := api.Rescan(ctx, tt.song, tt.reqid); err != nil {
				t.Fatalf("Rescan: %v", err)
			}
			covers, ok := api.GetURLs(tt.song)
			if !ok {
				t.Errorf("cover %s is not indexed", tt.song)
			}
			if len(covers) == 0 && tt.hasCover {
				t.Fatalf("got no covers; want 1 cover")
			}
			if len(covers) == 1 && !tt.hasCover {
				t.Fatalf("got 1 cover %v; want no covers", covers)
			}
			if len(covers) == 0 {
				return
			}
			cover := covers[0]
			u, err := url.Parse(cover)
			if err != nil {
				t.Fatalf("failed to parse url %s: %v", cover, err)
			}
			if !reflect.DeepEqual(u.Query(), tt.query) {
				t.Errorf("got query %+v; want %+v", u.Query(), tt.query)
			}
			req := httptest.NewRequest("GET", cover, nil)
			w := httptest.NewRecorder()
			api.ServeHTTP(w, req)
			resp := w.Result()
			if resp.StatusCode != 200 {
				t.Errorf("got status %d; want 200", resp.StatusCode)
			}
			for k, v := range tt.respHeader {
				if !reflect.DeepEqual(resp.Header[k], v) {
					t.Errorf("got header %s %v; want %v", k, resp.Header[k], v)
				}
			}
			got, _ := ioutil.ReadAll(resp.Body)
			if !reflect.DeepEqual(got, tt.respBinary) {
				t.Errorf("got invalid binary response")
			}
		})
	}
}
