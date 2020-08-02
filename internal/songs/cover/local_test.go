package cover

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLocalCover(t *testing.T) {
	searcher, err := NewLocalSearcher("/foo", filepath.Join("..", "..", "../"), []string{"app.png"})
	if err != nil {
		t.Fatalf("failed to initialize LocalSearcher: %v", err)
	}
	for _, tt := range []struct {
		in         map[string][]string
		want       map[string][]string
		wantHeader http.Header
		wantBinary []byte
	}{
		{
			in:         map[string][]string{"file": {"assets/test.flac"}},
			want:       map[string][]string{"file": {"assets/test.flac"}, "cover": {"/foo/assets/app.png"}},
			wantHeader: http.Header{"Content-Type": {"image/png"}},
			wantBinary: readFile(t, filepath.Join("..", "..", "..", "assets", "app.png")),
		},
		{
			in:   map[string][]string{"file": {"notfound/test.flac"}},
			want: map[string][]string{"file": {"notfound/test.flac"}, "cover": {}}},
	} {
		t.Run(fmt.Sprint(tt.in), func(t *testing.T) {
			song := searcher.AddTags(tt.in)
			if !reflect.DeepEqual(song, tt.want) {
				t.Errorf("got AddTags=%v; want %v", song, tt.want)
			}
			if len(song["covers"]) == 0 {
				return
			}
			cover := song["covers"][0]
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
			if !reflect.DeepEqual(got, tt.want) {
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
