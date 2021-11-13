package api_test

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/meiraka/vv/internal/vv/api"
)

func TestCurrentSongHandlerGET(t *testing.T) {
	songHook, randValue := testSongHook()
	for label, tt := range map[string][]struct {
		label       string
		currentSong func() (map[string][]string, error)
		err         error
		want        string
		cache       map[string][]string
		changed     bool
	}{
		"ok": {{
			label:       "empty",
			currentSong: func() (map[string][]string, error) { return map[string][]string{}, nil },
			want:        fmt.Sprintf(`{"%s":["%s"]}`, randValue, randValue),
			cache:       map[string][]string{randValue: {randValue}},
			changed:     true,
		}, {
			label:       "some data",
			currentSong: func() (map[string][]string, error) { return map[string][]string{"file": {"/foo/bar.mp3"}}, nil },
			want:        fmt.Sprintf(`{"%s":["%s"],"file":["/foo/bar.mp3"]}`, randValue, randValue),
			cache:       map[string][]string{"file": {"/foo/bar.mp3"}, randValue: {randValue}},
			changed:     true,
		}, {
			label:       "remove",
			currentSong: func() (map[string][]string, error) { return map[string][]string{}, nil },
			want:        fmt.Sprintf(`{"%s":["%s"]}`, randValue, randValue),
			cache:       map[string][]string{randValue: {randValue}},
			changed:     true,
		}},
		"error": {{
			label:       "prepare data",
			currentSong: func() (map[string][]string, error) { return map[string][]string{"file": {"/foo/bar.mp3"}}, nil },
			want:        fmt.Sprintf(`{"%s":["%s"],"file":["/foo/bar.mp3"]}`, randValue, randValue),
			cache:       map[string][]string{"file": {"/foo/bar.mp3"}, randValue: {randValue}},
			changed:     true,
		}, {
			label:       "error",
			currentSong: func() (map[string][]string, error) { return nil, errTest },
			err:         errTest,
			want:        fmt.Sprintf(`{"%s":["%s"],"file":["/foo/bar.mp3"]}`, randValue, randValue),
			cache:       map[string][]string{"file": {"/foo/bar.mp3"}, randValue: {randValue}},
		}},
	} {
		t.Run(label, func(t *testing.T) {
			mpd := &mpdPlaylistSongsCurrent{}
			h, err := api.NewCurrentSongHandler(mpd, songHook)
			if err != nil {
				t.Fatalf("api.NewPlaylistSongsCurrentHandler() = %v, %v", h, err)
			}
			for i := range tt {
				t.Run(tt[i].label, func(t *testing.T) {
					mpd.t = t
					mpd.currentSong = tt[i].currentSong
					if err := h.Update(context.TODO()); !errors.Is(err, tt[i].err) {
						t.Errorf("handler.Update(context.TODO()) = %v; want %v", err, tt[i].err)
					}
					r := httptest.NewRequest(http.MethodGet, "/", nil)
					w := httptest.NewRecorder()
					h.ServeHTTP(w, r)
					if status, got := w.Result().StatusCode, w.Body.String(); status != http.StatusOK || got != tt[i].want {
						t.Errorf("ServeHTTP got\n%d %s; want\n%d %s", status, got, http.StatusOK, tt[i].want)
					}
					if changed := recieveMsg(h.Changed()); changed != tt[i].changed {
						t.Errorf("changed = %v; want %v", changed, tt[i].changed)
					}
				})
			}
		})
	}
}

type mpdPlaylistSongsCurrent struct {
	t           *testing.T
	currentSong func() (map[string][]string, error)
}

func (m *mpdPlaylistSongsCurrent) CurrentSong(context.Context) (map[string][]string, error) {
	m.t.Helper()
	if m.currentSong == nil {
		m.t.Fatal("no CurrentSong mock function")
	}
	return m.currentSong()
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func testSongHook() (func(s map[string][]string) map[string][]string, string) {
	key := fmt.Sprint(rand.Int())
	return func(s map[string][]string) map[string][]string {
		s[key] = []string{key}
		return s
	}, key
}
