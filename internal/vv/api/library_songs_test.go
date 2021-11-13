package api_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/meiraka/vv/internal/vv/api"
)

func TestLibrarySongsHandlerGet(t *testing.T) {
	songsHook, randValue := testSongsHook()
	for label, tt := range map[string][]struct {
		label       string
		listAllInfo func(*testing.T, string) ([]map[string][]string, error)
		err         error
		want        string
		cache       []map[string][]string
		changed     bool
	}{
		"ok": {{
			label: "empty",
			listAllInfo: func(t *testing.T, path string) ([]map[string][]string, error) {
				t.Helper()
				if path != "/" {
					t.Errorf("called mpd.ListAllInfo(ctx, %q); want mpd.ListAllInfo(ctx, %q)", path, "/")
				}
				return []map[string][]string{}, nil
			},
			want:    `[]`,
			cache:   []map[string][]string{},
			changed: true,
		}, {
			label: "some data",
			listAllInfo: func(t *testing.T, path string) ([]map[string][]string, error) {
				t.Helper()
				if path != "/" {
					t.Errorf("called mpd.ListAllInfo(ctx, %q); want mpd.ListAllInfo(ctx, %q)", path, "/")
				}
				return []map[string][]string{{"file": {"/foo/bar.mp3"}}}, nil
			},
			want:    fmt.Sprintf(`[{"%s":["%s"],"file":["/foo/bar.mp3"]}]`, randValue, randValue),
			cache:   []map[string][]string{{"file": {"/foo/bar.mp3"}, randValue: {randValue}}},
			changed: true,
		}, {
			label: "remove",
			listAllInfo: func(t *testing.T, path string) ([]map[string][]string, error) {
				t.Helper()
				if path != "/" {
					t.Errorf("called mpd.ListAllInfo(ctx, %q); want mpd.ListAllInfo(ctx, %q)", path, "/")
				}
				return []map[string][]string{}, nil
			},
			want:    `[]`,
			cache:   []map[string][]string{},
			changed: true,
		}},
		`error`: {{
			label: "prepare data",
			listAllInfo: func(t *testing.T, path string) ([]map[string][]string, error) {
				t.Helper()
				if path != "/" {
					t.Errorf("called mpd.ListAllInfo(ctx, %q); want mpd.ListAllInfo(ctx, %q)", path, "/")
				}
				return []map[string][]string{{"file": {"/foo/bar.mp3"}}}, nil
			},
			want:    fmt.Sprintf(`[{"%s":["%s"],"file":["/foo/bar.mp3"]}]`, randValue, randValue),
			cache:   []map[string][]string{{"file": {"/foo/bar.mp3"}, randValue: {randValue}}},
			changed: true,
		}, {
			label: "error",
			listAllInfo: func(t *testing.T, path string) ([]map[string][]string, error) {
				t.Helper()
				if path != "/" {
					t.Errorf("called mpd.ListAllInfo(ctx, %q); want mpd.ListAllInfo(ctx, %q)", path, "/")
				}
				return nil, errTest
			},
			err:   errTest,
			want:  fmt.Sprintf(`[{"%s":["%s"],"file":["/foo/bar.mp3"]}]`, randValue, randValue),
			cache: []map[string][]string{{"file": {"/foo/bar.mp3"}, randValue: {randValue}}},
		}},
	} {
		t.Run(label, func(t *testing.T) {
			mpd := &mpdLibrarySongs{t: t}
			h, err := api.NewLibrarySongsHandler(mpd, songsHook)
			if err != nil {
				t.Fatalf("api.NewLibrarySongs() = %v, %v", h, err)
			}
			for i := range tt {
				t.Run(tt[i].label, func(t *testing.T) {
					mpd.listAllInfo = tt[i].listAllInfo
					if err := h.Update(context.TODO()); !errors.Is(err, tt[i].err) {
						t.Errorf("handler.Update(context.TODO()) = %v; want %v", err, tt[i].err)
					}

					r := httptest.NewRequest(http.MethodGet, "/", nil)
					w := httptest.NewRecorder()
					h.ServeHTTP(w, r)
					if status, got := w.Result().StatusCode, w.Body.String(); status != http.StatusOK || got != tt[i].want {
						t.Errorf("ServeHTTP got\n%d %s; want\n%d %s", status, got, http.StatusOK, tt[i].want)
					}
					if cache := h.Cache(); !reflect.DeepEqual(cache, tt[i].cache) {
						t.Errorf("got cache\n%v; want\n%v", cache, tt[i].cache)
					}
					if changed := recieveMsg(h.Changed()); changed != tt[i].changed {
						t.Errorf("changed = %v; want %v", changed, tt[i].changed)
					}
				})
			}
		})
	}

}

func testSongsHook() (func(s []map[string][]string) []map[string][]string, string) {
	f, key := testSongHook()
	return func(s []map[string][]string) []map[string][]string {
		for i := range s {
			s[i] = f(s[i])
		}
		return s
	}, key
}

type mpdLibrarySongs struct {
	t           *testing.T
	listAllInfo func(*testing.T, string) ([]map[string][]string, error)
}

func (m *mpdLibrarySongs) ListAllInfo(ctx context.Context, s string) ([]map[string][]string, error) {
	m.t.Helper()
	if m.listAllInfo == nil {
		m.t.Fatal("no ListAllInfo mock function")
	}
	return m.listAllInfo(m.t, s)
}
