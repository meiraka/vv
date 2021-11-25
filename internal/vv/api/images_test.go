package api_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/meiraka/vv/internal/vv/api"
)

func TestImagesHandler(t *testing.T) {
	songs := []map[string][]string{
		{"file": {"/foo/bar.mp3"}},
		{"file": {"/baz/qux.mp3"}},
	}
	for label, tt := range map[string][]struct {
		label string
		// wait got := <-handler.Changed() and compare got and *changed before requests, if not nil
		changed    *bool
		method     string
		url        string
		body       io.Reader
		songs      []map[string][]string
		want       string
		wantStatus int
		img        *imageProvider
	}{
		"ok/GET": {{
			method:     http.MethodGet,
			songs:      songs,
			want:       `{"updating":false}`,
			wantStatus: http.StatusOK,
		}},
		"ok/updating/running": {{
			label:      `POST/{"updating":true}`,
			method:     http.MethodPost,
			body:       strings.NewReader(`{"updating":true}`),
			songs:      songs,
			want:       `{"updating":false}`,
			wantStatus: http.StatusAccepted,
			img: &imageProvider{
				getURLs: func(t *testing.T, s map[string][]string) ([]string, bool) {
					// TODO: check input
					return []string{"/foo/cover.jpg"}, true
				},
				rescan: func(ctx context.Context, t *testing.T, s map[string][]string, id string) error {
					// TODO: check input
					<-ctx.Done()
					return nil
				}},
		}, {
			label:      `POST/{"updating":true}/already running`,
			method:     http.MethodPost,
			body:       strings.NewReader(`{"updating":true}`),
			want:       `{"error":"api: update already started"}`,
			wantStatus: http.StatusInternalServerError,
		}, {
			label:      "GET",
			changed:    boolptr(true),
			method:     http.MethodGet,
			want:       `{"updating":true}`,
			wantStatus: http.StatusOK,
		}},
		"ok/updating/stopped": {{
			label:      `POST/{"updating":true}`,
			method:     http.MethodPost,
			body:       strings.NewReader(`{"updating":true}`),
			songs:      songs,
			want:       `{"updating":false}`,
			wantStatus: http.StatusAccepted,
			img: &imageProvider{
				getURLs: func(t *testing.T, s map[string][]string) ([]string, bool) {
					// TODO: check input
					return []string{"/foo/cover.jpg"}, true
				},
				rescan: func(ctx context.Context, t *testing.T, s map[string][]string, id string) error {
					// TODO: check input
					return nil
				}},
		}, {
			label:   "(waiting update event)",
			changed: boolptr(true),
		}, {
			label:      "GET",
			changed:    boolptr(false),
			method:     http.MethodGet,
			want:       `{"updating":false}`,
			wantStatus: http.StatusOK,
		}},
		"error/POST/invalid json": {{
			method:     http.MethodPost,
			body:       strings.NewReader(`invalid json`),
			songs:      songs,
			want:       `{"error":"invalid character 'i' looking for beginning of value"}`,
			wantStatus: http.StatusBadRequest,
		}},
		`error/POST/{"updating":false}`: {{
			method:     http.MethodPost,
			body:       strings.NewReader(`{"updating":false}`),
			songs:      songs,
			want:       `{"error":"requires updating=true"}`,
			wantStatus: http.StatusBadRequest,
		}},
	} {
		t.Run(label, func(t *testing.T) {
			img := &imageProvider{t: t}
			h, err := api.NewImagesHandler([]api.ImageProvider{img})
			if err != nil {
				t.Fatalf("api.NewLibraryHandler(mpd) = %v", err)
			}
			defer h.Close()
			defer h.Shutdown(context.TODO())
			for i := range tt {
				f := func(t *testing.T) {
					if tt[i].changed != nil {
						want := *tt[i].changed
						select {
						case got := <-h.Changed():
							if got != want {
								t.Errorf("got chaned = %v; want %v", got, want)
							}
						case <-time.After(time.Second):
							t.Errorf("no changed event in 1sec")
						}
					}
					if tt[i].songs != nil {
						h.UpdateLibrarySongs(tt[i].songs)
					}
					img.t = t
					if tt[i].img != nil {
						img.rescan = tt[i].img.rescan
						img.update = tt[i].img.update
						img.getURLs = tt[i].img.getURLs
					}
					if tt[i].method != "" {
						r := httptest.NewRequest(tt[i].method, "/", tt[i].body)
						w := httptest.NewRecorder()
						h.ServeHTTP(w, r)
						if status, got := w.Result().StatusCode, w.Body.String(); status != tt[i].wantStatus || got != tt[i].want {
							t.Errorf("ServeHTTP got\n%d %s; want\n%d %s", status, got, tt[i].wantStatus, tt[i].want)
						}
					}
				}
				if len(tt) != 1 {
					if tt[i].label == "" {
						t.Fatalf("test definition error: no test label")
					}
					t.Run(tt[i].label, f)
				} else {
					f(t)
				}
			}
		})
	}
}

func TestImagesHandlerConvSongs(t *testing.T) {
	var m sync.Map
	for label, tt := range map[string][]struct {
		label string
		// wait got := <-handler.Changed() and compare got and *changed before requests, if not nil
		changed *bool
		songs   []map[string][]string
		want    []map[string][]string
		img     *imageProvider
	}{
		"ok": {{
			label: "before updates",
			songs: []map[string][]string{
				{"file": {"/foo/bar.mp3"}},
				{"file": {"/baz/qux.mp3"}},
			},
			want: []map[string][]string{
				{"file": {"/foo/bar.mp3"}},
				{"file": {"/baz/qux.mp3"}},
			},
			img: &imageProvider{
				getURLs: func(t *testing.T, s map[string][]string) ([]string, bool) {
					// TODO: check input
					if _, ok := m.Load("ok"); !ok {
						// before update: return false
						return []string{}, false
					}
					// after update: return false
					return []string{"/foo/cover.jpg"}, true
				},
				update: func(ctx context.Context, t *testing.T, s map[string][]string) error {
					// TODO: check input
					m.Store("ok", struct{}{})
					return nil
				},
			},
		}, {
			changed: boolptr(true),
			label:   "(waiting update event)",
			songs:   []map[string][]string{},
			want:    []map[string][]string{},
		}, {
			changed: boolptr(false),
			label:   "after updates",
			songs: []map[string][]string{
				{"file": {"/foo/bar.mp3"}},
				{"file": {"/baz/qux.mp3"}},
			},
			want: []map[string][]string{
				{"file": {"/foo/bar.mp3"}, "cover": {"/foo/cover.jpg"}},
				{"file": {"/baz/qux.mp3"}, "cover": {"/foo/cover.jpg"}},
			},
		}},
	} {
		t.Run(label, func(t *testing.T) {
			img := &imageProvider{t: t}
			h, err := api.NewImagesHandler([]api.ImageProvider{img})
			if err != nil {
				t.Fatalf("api.NewLibraryHandler(mpd) = %v", err)
			}
			defer h.Close()
			defer h.Shutdown(context.TODO())
			for i := range tt {
				f := func(t *testing.T) {
					if tt[i].changed != nil {
						want := *tt[i].changed
						select {
						case got := <-h.Changed():
							if got != want {
								t.Errorf("got chaned = %v; want %v", got, want)
							}
						case <-time.After(time.Second):
							t.Errorf("no changed event in 1sec")
						}
					}
					img.t = t
					if tt[i].img != nil {
						img.rescan = tt[i].img.rescan
						img.update = tt[i].img.update
						img.getURLs = tt[i].img.getURLs
					}
					if got, want := h.ConvSongs(tt[i].songs), tt[i].want; !reflect.DeepEqual(got, want) {
						t.Errorf("got\n%v; want\n%v", got, want)
					}
				}
				if len(tt) != 1 {
					if tt[i].label == "" {
						t.Fatalf("test definition error: no test label")
					}
					t.Run(tt[i].label, f)
				} else {
					f(t)
				}
			}
		})
	}
}

func TestImagesHandlerConvSong(t *testing.T) {
	for label, tt := range map[string]struct {
		song map[string][]string
		want map[string][]string
		ok   bool
		img  *imageProvider
	}{
		"ok/not indexed": {
			song: map[string][]string{"file": {"/foo/bar.mp3"}},
			want: map[string][]string{"file": {"/foo/bar.mp3"}},
			img: &imageProvider{
				getURLs: func(t *testing.T, s map[string][]string) ([]string, bool) {
					// TODO: check input
					return []string{}, false
				},
			},
		},
		"ok/indexed": {
			song: map[string][]string{"file": {"/foo/bar.mp3"}},
			want: map[string][]string{"file": {"/foo/bar.mp3"}, "cover": {"/foo/cover.jpg"}},
			ok:   true,
			img: &imageProvider{
				getURLs: func(t *testing.T, s map[string][]string) ([]string, bool) {
					// TODO: check input
					return []string{"/foo/cover.jpg"}, true
				},
			},
		},
	} {
		t.Run(label, func(t *testing.T) {
			img := &imageProvider{t: t}
			h, err := api.NewImagesHandler([]api.ImageProvider{img})
			if err != nil {
				t.Fatalf("api.NewLibraryHandler(mpd) = %v", err)
			}
			defer h.Close()
			defer h.Shutdown(context.TODO())
			img.t = t
			if tt.img != nil {
				img.rescan = tt.img.rescan
				img.update = tt.img.update
				img.getURLs = tt.img.getURLs
			}
			if got, ok := h.ConvSong(tt.song); !reflect.DeepEqual(got, tt.want) || ok != tt.ok {
				t.Errorf("got\n%v, %v; want\n%v %v", got, ok, tt.want, tt.ok)
			}
		})
	}
}

type imageProvider struct {
	t       *testing.T
	update  func(context.Context, *testing.T, map[string][]string) error
	rescan  func(context.Context, *testing.T, map[string][]string, string) error
	getURLs func(*testing.T, map[string][]string) ([]string, bool)
}

func (i *imageProvider) Update(ctx context.Context, a map[string][]string) error {
	i.t.Helper()
	if i.update == nil {
		i.t.Fatal("no Update mock function")
	}
	return i.update(ctx, i.t, a)
}
func (i *imageProvider) Rescan(ctx context.Context, a map[string][]string, b string) error {
	i.t.Helper()
	if i.rescan == nil {
		i.t.Fatal("no Rescan mock function")
	}
	return i.rescan(ctx, i.t, a, b)
}
func (i *imageProvider) GetURLs(a map[string][]string) ([]string, bool) {
	i.t.Helper()
	if i.getURLs == nil {
		i.t.Fatal("no GetURLs mock function")
	}
	return i.getURLs(i.t, a)
}
