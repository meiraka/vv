package api_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/meiraka/vv/internal/mpd"
	"github.com/meiraka/vv/internal/songs"
	"github.com/meiraka/vv/internal/vv/api"
)

var testSongs = []map[string][]string{
	{"file": {"/foo/bar.mp3"}, "Title": {"bar"}, "Album": {"foo"}},
	{"file": {"/foo/foo.mp3"}, "Title": {"foo"}, "Album": {"foo"}},
	{"file": {"/baz/qux.mp3"}, "Title": {"qux"}, "Album": {"baz"}},
	{"file": {"/baz/baz.mp3"}, "Title": {"baz"}, "Album": {"baz"}},
}

func TestPlaylistHandler(t *testing.T) {
	for label, tt := range map[string][]struct {
		label      string
		method     string
		body       io.Reader
		library    []map[string][]string
		playlist   []map[string][]string
		pos        *int
		want       string
		wantStatus int
		mpd        *mpdPlaylist
	}{
		"ok/GET": {{
			method:     http.MethodGet,
			want:       `{}`,
			wantStatus: http.StatusOK,
		}},
		"error/POST/invalid json": {{
			method:     http.MethodPost,
			body:       strings.NewReader(`invalid json`),
			want:       `{"error":"invalid character 'i' looking for beginning of value"}`,
			wantStatus: http.StatusBadRequest,
		}},
		"ok/sort": {{
			label:      `POST/{"current":1,"filters":[["Album","baz"],["Title","qux"]],"sort":["Album","Title"]}`,
			library:    songs.Copy(testSongs),
			method:     http.MethodPost,
			body:       strings.NewReader(`{"current":1,"filters":[["Album","baz"],["Title","qux"]],"sort":["Album","Title"]}`),
			want:       `{}`,
			wantStatus: http.StatusAccepted,
			mpd: &mpdPlaylist{
				execCommandList: func(t *testing.T, got *mpd.CommandList) error {
					want := &mpd.CommandList{}
					want.Clear()
					want.Add("/baz/baz.mp3")
					want.Add("/baz/qux.mp3")
					want.Add("/foo/bar.mp3")
					want.Add("/foo/foo.mp3")
					want.Play(1)
					t.Helper()
					if !mpd.CommandListEqual(got, want) {
						t.Errorf("call mpd.ExecCommandList(ctx,\n%v); want mpd.ExecCommandList(ctx,\n%v)", got, want)
					}
					return nil
				},
			},
		}, {
			label:      `GET`,
			playlist:   []map[string][]string{testSongs[3], testSongs[2], testSongs[0], testSongs[1]},
			pos:        intptr(1),
			method:     http.MethodGet,
			want:       `{"current":1,"sort":["Album","Title"]}`,
			wantStatus: http.StatusOK,
		}, {
			label:      `GET/playlist changed`,
			playlist:   []map[string][]string{testSongs[3], testSongs[2], testSongs[1], testSongs[0]},
			pos:        intptr(1),
			method:     http.MethodGet,
			want:       `{"current":1}`,
			wantStatus: http.StatusOK,
		}},
		"error/sort": {{
			label:      `POST/{"current":1,"filters":[["Album","baz"],["Title","qux"]],"sort":["Album","Title"]}`,
			library:    songs.Copy(testSongs),
			method:     http.MethodPost,
			body:       strings.NewReader(`{"current":1,"filters":[["Album","baz"],["Title","qux"]],"sort":["Album","Title"]}`),
			want:       `{}`,
			wantStatus: http.StatusAccepted,
			mpd: &mpdPlaylist{
				execCommandList: func(t *testing.T, got *mpd.CommandList) error {
					want := &mpd.CommandList{}
					want.Clear()
					want.Add("/baz/baz.mp3")
					want.Add("/baz/qux.mp3")
					want.Add("/foo/bar.mp3")
					want.Add("/foo/foo.mp3")
					want.Play(1)
					t.Helper()
					if !mpd.CommandListEqual(got, want) {
						t.Errorf("call mpd.ExecCommandList(ctx,\n%v); want mpd.ExecCommandList(ctx,\n%v)", got, want)
					}
					return errTest
				},
			},
		}, {
			label:      `GET`,
			playlist:   []map[string][]string{testSongs[3], testSongs[2], testSongs[0], testSongs[1]},
			pos:        intptr(1),
			method:     http.MethodGet,
			want:       `{"current":1}`, // sort is not updated
			wantStatus: http.StatusOK,
		}},
		"ok/track": {{
			label:      `POST/{"current":1,"filters":[["Album","baz"],["Title","qux"]],"sort":["Album","Title"]}`,
			library:    songs.Copy(testSongs),
			playlist:   []map[string][]string{testSongs[3], testSongs[2], testSongs[0], testSongs[1]},
			method:     http.MethodPost,
			body:       strings.NewReader(`{"current":1,"filters":[["Album","baz"],["Title","qux"]],"sort":["Album","Title"]}`),
			want:       `{"sort":["Album","Title"]}`,
			wantStatus: http.StatusAccepted,
			mpd:        &mpdPlaylist{play: mockIntFunc("mpd.Play(ctx, %d)", 1, nil)},
		}, {
			label:      `GET`,
			pos:        intptr(1),
			method:     http.MethodGet,
			want:       `{"current":1,"sort":["Album","Title"]}`,
			wantStatus: http.StatusOK,
		}},
		"error/track": {{
			label:      `POST/{"current":1,"filters":[["Album","baz"],["Title","qux"]],"sort":["Album","Title"]}`,
			library:    songs.Copy(testSongs),
			playlist:   []map[string][]string{testSongs[3], testSongs[2], testSongs[0], testSongs[1]},
			method:     http.MethodPost,
			body:       strings.NewReader(`{"current":1,"filters":[["Album","baz"],["Title","qux"]],"sort":["Album","Title"]}`),
			want:       `{"error":"api_test: test error"}`,
			wantStatus: http.StatusInternalServerError,
			mpd:        &mpdPlaylist{play: mockIntFunc("mpd.Play(ctx, %d)", 1, errTest)},
		}},
	} {
		t.Run(label, func(t *testing.T) {
			mpd := &mpdPlaylist{t: t}
			h, err := api.NewPlaylistHandler(mpd, &api.Config{BackgroundTimeout: time.Second})
			if err != nil {
				t.Fatalf("api.NewPlaylistHandler(mpd, config) = %v", err)
			}
			defer h.Close()
			for i := range tt {
				f := func(t *testing.T) {
					mpd.t = t
					if tt[i].mpd != nil {
						mpd.play = tt[i].mpd.play
						mpd.execCommandList = tt[i].mpd.execCommandList
					}
					if tt[i].library != nil {
						h.UpdateLibrarySongs(tt[i].library)
					}
					if tt[i].playlist != nil {
						h.UpdatePlaylistSongs(tt[i].playlist)
					}
					if tt[i].pos != nil {
						h.UpdateCurrent(*tt[i].pos)
					}

					r := httptest.NewRequest(tt[i].method, "/", tt[i].body)
					w := httptest.NewRecorder()
					h.ServeHTTP(w, r)
					if status, got := w.Result().StatusCode, w.Body.String(); status != tt[i].wantStatus || got != tt[i].want {
						t.Errorf("ServeHTTP got\n%d %s; want\n%d %s", status, got, tt[i].wantStatus, tt[i].want)
					}
					h.Wait(context.TODO())
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

type mpdPlaylist struct {
	t               *testing.T
	play            func(*testing.T, int) error
	execCommandList func(*testing.T, *mpd.CommandList) error
}

func (m *mpdPlaylist) Play(ctx context.Context, i int) error {
	m.t.Helper()
	if m.play == nil {
		m.t.Fatal("no Play mock function")
	}
	return m.play(m.t, i)
}
func (m *mpdPlaylist) ExecCommandList(ctx context.Context, i *mpd.CommandList) error {
	m.t.Helper()
	if m.execCommandList == nil {
		m.t.Fatal("no ExecCommandList mock function")
	}
	return m.execCommandList(m.t, i)
}
