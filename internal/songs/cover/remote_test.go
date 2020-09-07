package cover

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
				t.Errorf("got %v %v; want nil, false", covers, ok)
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

	for rescanIndex, label := range []string{"empty db", "use db"} {
		t.Run(label, func(t *testing.T) {
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
				api.Rescan(context.TODO(), tt.in)
				t.Logf("rescan: %v", tt.in)
				for i := 0; i < 2; i++ {
					t.Run(fmt.Sprint(tt.in, i), func(t *testing.T) {
						covers, ok := api.GetURLs(tt.in)
						if !ok {
							t.Error("cover is not indexed")
						}
						t.Logf("urls: %v", covers)
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
						want := url.Values{"v": {strconv.Itoa(rescanIndex)}}
						if !reflect.DeepEqual(u.Query(), want) {
							t.Errorf("got query %+v; want %+v", u, want)
						}
						req := httptest.NewRequest("GET", cover, nil)
						w := httptest.NewRecorder()
						api.ServeHTTP(w, req)
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
