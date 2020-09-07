package cover

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
)

func TestLocalCover(t *testing.T) {
	api, err := NewLocal("/foo", filepath.Join("..", "..", "../"), []string{"app.png"})
	if err != nil {
		t.Fatalf("failed to initialize cover.Local: %v", err)
	}
	for _, tt := range []struct {
		in         map[string][]string
		want       []string
		wantHeader http.Header
		wantBinary []byte
	}{
		{
			in:         map[string][]string{"file": {"assets/test.flac"}},
			want:       []string{"/foo/assets/app.png?d=" + strconv.FormatInt(stat(t, filepath.Join("..", "..", "..", "assets", "app.png")).ModTime().Unix(), 10)},
			wantHeader: http.Header{"Content-Type": {"image/png"}},
			wantBinary: readFile(t, filepath.Join("..", "..", "..", "assets", "app.png")),
		},
		{
			in:   map[string][]string{"file": {"notfound/test.flac"}},
			want: []string{},
		},
	} {
		t.Run(fmt.Sprint(tt.in), func(t *testing.T) {
			covers, _ := api.GetURLs(tt.in)
			if !reflect.DeepEqual(covers, tt.want) {
				t.Errorf("got GetURLs=%v; want %v", covers, tt.want)
			}
			if len(covers) == 0 {
				return
			}
			cover := covers[0]
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

func readFile(t *testing.T, path string) []byte {
	t.Helper()
	b, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}
	return b

}
func stat(t *testing.T, path string) os.FileInfo {
	s, err := os.Stat(path)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}
	return s
}
