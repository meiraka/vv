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
	"strconv"
	"testing"

	"github.com/meiraka/vv/internal/mpd"
	"github.com/meiraka/vv/internal/mpd/mpdtest"
)

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

func TestRemoteRescan(t *testing.T) {
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

	for id, tt := range []struct {
		label string
		path  string
		query url.Values
	}{
		{label: "first", path: "app.png", query: url.Values{"v": {"0"}}},
		{label: "same image", path: "app.png", query: url.Values{"v": {"0"}}},
		{label: "different image", path: "app-black.png", query: url.Values{"v": {"1"}}},
	} {
		t.Run(tt.label, func(t *testing.T) {
			png := readFile(t, filepath.Join(path, "..", "..", "..", "..", "..", "assets", tt.path))
			api, err := NewRemote("/api/images", c, testDir)
			if err != nil {
				t.Fatalf("failed to initialize cover.Remote: %v", err)
			}
			go func() {
				ctx := context.Background()
				svr.Expect(ctx, &mpdtest.WR{Read: `albumart "assets/test.flac" 0` + "\n", Write: fmt.Sprintf("size: %d\nbinary: %d\n%s\nOK\n", len(png), len(png), png)})
				svr.Expect(ctx, &mpdtest.WR{Read: `albumart "notfound/test.flac" 0` + "\n", Write: "ACK [50@0] {albumart} No file exists\n"})
			}()
			defer api.Close()
			for _, tr := range []struct {
				in         map[string][]string
				hasCover   bool
				wantBinary []byte
				wantHeader http.Header
			}{
				{
					in:         map[string][]string{"file": {"assets/test.flac"}},
					hasCover:   true,
					wantBinary: png,
					wantHeader: http.Header{"Content-Type": {"image/png"}, "Cache-Control": {"max-age=31536000"}},
				},
				{
					in:       map[string][]string{"file": {"notfound/test.flac"}},
					hasCover: false,
				},
			} {
				t.Logf("Rescan: %v", tr.in)
				if err := api.Rescan(context.TODO(), tr.in, strconv.Itoa(id)); err != nil {
					t.Fatalf("Rescan: %v", err)
				}
				for i := 0; i < 2; i++ {
					t.Run(fmt.Sprint(tr.in, i), func(t *testing.T) {
						covers, ok := api.GetURLs(tr.in)
						if !ok {
							t.Errorf("cover %s is not indexed", tr.in)
						}
						t.Logf("urls: %v", covers)
						if len(covers) == 0 && tr.hasCover {
							t.Fatalf("got no covers; want 1 cover")
						}
						if len(covers) == 1 && !tr.hasCover {
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
						for k, v := range tr.wantHeader {
							if !reflect.DeepEqual(resp.Header[k], v) {
								t.Errorf("got header %s %v; want %v", k, resp.Header[k], v)
							}
						}
						got, _ := ioutil.ReadAll(resp.Body)
						if !reflect.DeepEqual(got, tr.wantBinary) {
							t.Errorf("got invalid binary response")
						}
					})
				}
			}
		})
	}
}

func TestRemoteUpdate(t *testing.T) {
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

	for _, tt := range []struct {
		label string
		path  string
		req   bool
		query url.Values
	}{
		{label: "first", req: true, query: url.Values{"v": {"0"}}},
		{label: "second", query: url.Values{"v": {"0"}}},
	} {
		t.Run(tt.label, func(t *testing.T) {
			png := readFile(t, filepath.Join(path, "..", "..", "..", "..", "..", "assets", "app.png"))
			api, err := NewRemote("/api/images", c, testDir)
			if err != nil {
				t.Fatalf("failed to initialize cover.Remote: %v", err)
			}
			if tt.req {
				go func() {
					ctx := context.Background()
					svr.Expect(ctx, &mpdtest.WR{Read: `albumart "assets/test.flac" 0` + "\n", Write: fmt.Sprintf("size: %d\nbinary: %d\n%s\nOK\n", len(png), len(png), png)})
					svr.Expect(ctx, &mpdtest.WR{Read: `albumart "notfound/test.flac" 0` + "\n", Write: "ACK [50@0] {albumart} No file exists\n"})
				}()
			}
			defer api.Close()
			for _, tr := range []struct {
				in         map[string][]string
				hasCover   bool
				wantBinary []byte
				wantHeader http.Header
			}{
				{
					in:         map[string][]string{"file": {"assets/test.flac"}},
					hasCover:   true,
					wantBinary: png,
					wantHeader: http.Header{"Content-Type": {"image/png"}, "Cache-Control": {"max-age=31536000"}},
				},
				{
					in:       map[string][]string{"file": {"notfound/test.flac"}},
					hasCover: false,
				},
			} {
				t.Logf("Update: %v", tr.in)
				if err := api.Update(context.TODO(), tr.in); err != nil {
					t.Fatalf("Update: %v", err)
				}
				for i := 0; i < 2; i++ {
					t.Run(fmt.Sprint(tr.in, i), func(t *testing.T) {
						covers, ok := api.GetURLs(tr.in)
						if !ok {
							t.Errorf("cover %s is not indexed", tr.in)
						}
						t.Logf("urls: %v", covers)
						if len(covers) == 0 && tr.hasCover {
							t.Fatalf("got no covers; want 1 cover")
						}
						if len(covers) == 1 && !tr.hasCover {
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
						for k, v := range tr.wantHeader {
							if !reflect.DeepEqual(resp.Header[k], v) {
								t.Errorf("got header %s %v; want %v", k, resp.Header[k], v)
							}
						}
						got, _ := ioutil.ReadAll(resp.Body)
						if !reflect.DeepEqual(got, tr.wantBinary) {
							t.Errorf("got invalid binary response")
						}
					})
				}
			}
		})
	}
}
