package cover

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/meiraka/vv/internal/mpd"
	"github.com/meiraka/vv/internal/mpd/mpdtest"
)

func TestRemoteSearcher(t *testing.T) {
	svr, err := mpdtest.NewServer("OK MPD 0.19")
	if err != nil {
		t.Fatalf("failed to create mpd test server: %v", err)
	}
	defer svr.Close()
	c, err := (&mpd.Dialer{}).Dial("tcp", svr.URL, "")
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
	png := readFile(t, filepath.Join(path, "..", "..", "..", "assets", "app.png"))
	go func() {
		ctx := context.Background()
		svr.Expect(ctx, &mpdtest.WR{Read: `albumart "assets/test.flac" 0` + "\n", Write: fmt.Sprintf("size: %d\nbinary: %d\n%s\nOK\n", len(png), len(png), png)})
		svr.Expect(ctx, &mpdtest.WR{Read: `albumart "notfound/test.flac" 0` + "\n", Write: "ACK [50@0] {albumart} No file exists\n"})
	}()
	for _, label := range []string{"empty db", "use db"} {
		t.Run(label, func(t *testing.T) {
			searcher, err := NewRemoteSearcher("/api/images", c, testDir)
			if err != nil {
				t.Fatalf("failed to initialize searcher: %v", err)
			}
			defer searcher.Close()
			for _, tt := range []struct {
				in         map[string][]string
				hasCover   bool
				wantBinary []byte
				wantHeader http.Header
			}{
				{
					in:         map[string][]string{"file": {"assets/test.flac"}},
					hasCover:   true,
					wantBinary: png,
					wantHeader: http.Header{"Content-Type": {"image/png"}},
				},
				{
					in:       map[string][]string{"file": {"notfound/test.flac"}},
					hasCover: false,
				},
			} {
				for i := 0; i < 2; i++ {
					t.Run(fmt.Sprint(tt.in, i), func(t *testing.T) {
						covers := searcher.GetURLs(tt.in)
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
						req := httptest.NewRequest("GET", cover, nil)
						w := httptest.NewRecorder()
						searcher.ServeHTTP(w, req)
						resp := w.Result()
						if resp.StatusCode != 200 {
							t.Errorf("got status %d; want 200", resp.StatusCode)
						}
						for k, v := range tt.wantHeader {
							if !reflect.DeepEqual(resp.Header[k], v) {
								t.Errorf("got header %s %v; want %v", k, resp.Header[k], v)
							}
						}
						got, _ := ioutil.ReadAll(resp.Body)
						if !reflect.DeepEqual(got, tt.wantBinary) {
							t.Errorf("got invalid binary response")
						}
					})
				}
			}
		})
	}
}

func TestRemoteSearcherRescan(t *testing.T) {
	svr, err := mpdtest.NewServer("OK MPD 0.19")
	if err != nil {
		t.Fatalf("failed to create mpd test server: %v", err)
	}
	defer svr.Close()
	c, err := (&mpd.Dialer{}).Dial("tcp", svr.URL, "")
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
	png := readFile(t, filepath.Join(path, "..", "..", "..", "assets", "app.png"))

	for _, label := range []string{"empty db", "use db"} {
		t.Run(label, func(t *testing.T) {
			searcher, err := NewRemoteSearcher("/api/images", c, testDir)
			if err != nil {
				t.Fatalf("failed to initialize searcher: %v", err)
			}
			go func() {
				ctx := context.Background()
				svr.Expect(ctx, &mpdtest.WR{Read: `albumart "assets/test.flac" 0` + "\n", Write: fmt.Sprintf("size: %d\nbinary: %d\n%s\nOK\n", len(png), len(png), png)})
				svr.Expect(ctx, &mpdtest.WR{Read: `albumart "notfound/test.flac" 0` + "\n", Write: "ACK [50@0] {albumart} No file exists\n"})
			}()
			defer searcher.Close()
			for _, tt := range []struct {
				in         map[string][]string
				hasCover   bool
				wantBinary []byte
				wantHeader http.Header
			}{
				{
					in:         map[string][]string{"file": {"assets/test.flac"}},
					hasCover:   true,
					wantBinary: png,
					wantHeader: http.Header{"Content-Type": {"image/png"}},
				},
				{
					in:       map[string][]string{"file": {"notfound/test.flac"}},
					hasCover: false,
				},
			} {
				searcher.Rescan([]map[string][]string{tt.in})
				for i := 0; i < 2; i++ {
					t.Run(fmt.Sprint(tt.in, i), func(t *testing.T) {
						covers := searcher.GetURLs(tt.in)
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
						req := httptest.NewRequest("GET", cover, nil)
						w := httptest.NewRecorder()
						searcher.ServeHTTP(w, req)
						resp := w.Result()
						if resp.StatusCode != 200 {
							t.Errorf("got status %d; want 200", resp.StatusCode)
						}
						for k, v := range tt.wantHeader {
							if !reflect.DeepEqual(resp.Header[k], v) {
								t.Errorf("got header %s %v; want %v", k, resp.Header[k], v)
							}
						}
						got, _ := ioutil.ReadAll(resp.Body)
						if !reflect.DeepEqual(got, tt.wantBinary) {
							t.Errorf("got invalid binary response")
						}
					})
				}
			}
		})
	}
}
