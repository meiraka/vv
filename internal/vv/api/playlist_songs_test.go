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

func TestPlaylistSongsHandlerGET(t *testing.T) {
	songsHook, randValue := testSongsHook()
	for label, tt := range map[string][]struct {
		label        string
		playlistInfo func(*testing.T) ([]map[string][]string, error)
		err          error
		want         string
		cache        []map[string][]string
		changed      bool
	}{
		`empty`: {{
			playlistInfo: func(t *testing.T) ([]map[string][]string, error) {
				return []map[string][]string{}, nil
			},
			want:  `[]`,
			cache: []map[string][]string{},
		}},
		`exists`: {{
			playlistInfo: func(t *testing.T) ([]map[string][]string, error) {
				return []map[string][]string{{"file": {"/foo/bar.mp3"}}}, nil
			},
			want:    fmt.Sprintf(`[{"%s":["%s"],"file":["/foo/bar.mp3"]}]`, randValue, randValue),
			cache:   []map[string][]string{{"file": {"/foo/bar.mp3"}, randValue: {randValue}}},
			changed: true,
		}},
		`error`: {{
			label: "prepare data",
			playlistInfo: func(t *testing.T) ([]map[string][]string, error) {
				return []map[string][]string{{"file": {"/foo/bar.mp3"}}}, nil
			},
			want:    fmt.Sprintf(`[{"%s":["%s"],"file":["/foo/bar.mp3"]}]`, randValue, randValue),
			cache:   []map[string][]string{{"file": {"/foo/bar.mp3"}, randValue: {randValue}}},
			changed: true,
		}, {
			label: "error",
			playlistInfo: func(t *testing.T) ([]map[string][]string, error) {
				t.Helper()
				return nil, errTest
			},
			err:   errTest,
			want:  fmt.Sprintf(`[{"%s":["%s"],"file":["/foo/bar.mp3"]}]`, randValue, randValue),
			cache: []map[string][]string{{"file": {"/foo/bar.mp3"}, randValue: {randValue}}},
		}},
	} {
		t.Run(label, func(t *testing.T) {
			mpd := &mpdPlaylistSongs{t: t}

			h, err := api.NewPlaylistSongsHandler(mpd, songsHook)
			if err != nil {
				t.Fatalf("api.NewPlaylistSongs() = %v, %v", h, err)
			}
			for i := range tt {
				f := func(t *testing.T) {
					mpd.playlistInfo = tt[i].playlistInfo
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
				}
				if len(tt) != 1 {
					t.Run(tt[i].label, f)
				} else {
					f(t)
				}
			}
		})
	}

}

type mpdPlaylistSongs struct {
	t            *testing.T
	playlistInfo func(*testing.T) ([]map[string][]string, error)
}

func (m *mpdPlaylistSongs) PlaylistInfo(ctx context.Context) ([]map[string][]string, error) {
	m.t.Helper()
	if m.playlistInfo == nil {
		m.t.Fatal("no PlaylistInfo mock function")
	}
	return m.playlistInfo(m.t)
}
