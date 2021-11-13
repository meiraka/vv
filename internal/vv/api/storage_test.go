package api_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/meiraka/vv/internal/mpd"
	"github.com/meiraka/vv/internal/vv/api"
)

func TestStorageHandlerGET(t *testing.T) {
	for label, tt := range map[string][]struct {
		label      string
		listMounts func(*testing.T) ([]map[string]string, error)
		err        error
		want       string
		changed    bool
	}{
		`ok`: {{
			label:      "empty",
			listMounts: func(*testing.T) ([]map[string]string, error) { return []map[string]string{}, nil },
			want:       "{}",
		}, {
			label: "minimal",
			listMounts: func(*testing.T) ([]map[string]string, error) {
				return []map[string]string{
					{"mount": "", "storage": "/home/foo/music"},
					{"mount": "foo", "storage": "nfs://192.168.1.4/export/mp3"},
				}, nil
			},
			want:    `{"":{"uri":"/home/foo/music"},"foo":{"uri":"nfs://192.168.1.4/export/mp3"}}`,
			changed: true,
		}, {
			label:      "removed",
			listMounts: func(*testing.T) ([]map[string]string, error) { return []map[string]string{}, nil },
			want:       "{}",
			changed:    true,
		}},
		`error/other`: {{
			label: "prepare data",
			listMounts: func(*testing.T) ([]map[string]string, error) {
				return []map[string]string{
					{"mount": "", "storage": "/home/foo/music"},
					{"mount": "foo", "storage": "nfs://192.168.1.4/export/mp3"},
				}, nil
			},
			want:    `{"":{"uri":"/home/foo/music"},"foo":{"uri":"nfs://192.168.1.4/export/mp3"}}`,
			changed: true,
		}, {
			label:      "error",
			listMounts: func(*testing.T) ([]map[string]string, error) { return nil, errTest },
			err:        errTest,
			want:       `{"":{"uri":"/home/foo/music"},"foo":{"uri":"nfs://192.168.1.4/export/mp3"}}`,
		}},
		`error/mpd`: {{
			label: "prepare data",
			listMounts: func(*testing.T) ([]map[string]string, error) {
				return []map[string]string{
					{"mount": "", "storage": "/home/foo/music"},
					{"mount": "foo", "storage": "nfs://192.168.1.4/export/mp3"},
				}, nil
			},
			want:    `{"":{"uri":"/home/foo/music"},"foo":{"uri":"nfs://192.168.1.4/export/mp3"}}`,
			changed: true,
		}, {
			label: "error",
			listMounts: func(*testing.T) ([]map[string]string, error) {
				return nil, &mpd.CommandError{ID: 5, Index: 0, Command: "listmounts", Message: "unknown command \"listmounts\""}
			},
			want:    "{}",
			changed: true,
		}},
	} {
		t.Run(label, func(t *testing.T) {
			mpd := &mpdStorage{t: t}
			h, err := api.NewStorageHandler(mpd)
			if err != nil {
				t.Fatalf("failed to init Storage: %v", err)
			}
			for i := range tt {
				f := func(t *testing.T) {
					mpd.listMounts = tt[i].listMounts
					if err := h.Update(context.TODO()); !errors.Is(err, tt[i].err) {
						t.Errorf("Update(ctx) = %v; want %v", err, tt[i].err)
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

func TestStorageHandlerPOST(t *testing.T) {
	for label, tt := range map[string]struct {
		body       string
		status     int
		want       string
		listMounts func(*testing.T) ([]map[string]string, error)
		mount      func(*testing.T, string, string) error
		unmount    func(*testing.T, string) error
		update     func(*testing.T, string) (map[string]string, error)
	}{
		"error/invalid json": {
			body:   `invalid json`,
			status: http.StatusBadRequest,
			want:   `{"error":"invalid character 'i' looking for beginning of value"}`,
		},
		`error/{"":{}}`: {
			body:   `{"":{}}`,
			status: http.StatusBadRequest,
			want:   `{"error":"storage name is empty"}`,
		},
		`ok/{"uri":"nfs://192.168.1.4/export/mp3"}`: {
			body:   `{"foo":{"uri":"nfs://192.168.1.4/export/mp3"}}`,
			status: http.StatusAccepted,
			want:   `{}`,
			mount: func(t *testing.T, name string, uri string) error {
				t.Helper()
				if wantName, wantURI := "foo", "nfs://192.168.1.4/export/mp3"; name != wantName || uri != wantURI {
					t.Errorf("called mpd.Mount(ctx, %q, %q); want mpd.Mount(ctx, %q, %q)", name, uri, wantName, wantURI)
				}
				return nil
			},
			update: func(t *testing.T, path string) (map[string]string, error) {
				t.Helper()
				if want := "foo"; path != want {
					t.Errorf("called mpd.Update(ctx, %q); want mpd.Update(ctx, %q)", path, want)
				}
				return map[string]string{"updating_db": "1"}, nil
			},
		},
		`error/{"uri":"nfs://192.168.1.4/export/mp3"}/update`: {
			body:   `{"foo":{"uri":"nfs://192.168.1.4/export/mp3"}}`,
			status: http.StatusInternalServerError,
			want:   `{"error":"api_test: test error"}`,
			mount: func(t *testing.T, name string, uri string) error {
				t.Helper()
				if wantName, wantURI := "foo", "nfs://192.168.1.4/export/mp3"; name != wantName || uri != wantURI {
					t.Errorf("called mpd.Mount(ctx, %q, %q); want mpd.Mount(ctx, %q, %q)", name, uri, wantName, wantURI)
				}
				return nil
			},
			update: func(t *testing.T, path string) (map[string]string, error) {
				t.Helper()
				if want := "foo"; path != want {
					t.Errorf("called mpd.Update(ctx, %q); want mpd.Update(ctx, %q)", path, want)
				}
				return nil, errTest
			},
		},
		`error/{"uri":"nfs://192.168.1.4/export/mp3"}/mount`: {
			body:   `{"foo":{"uri":"nfs://192.168.1.4/export/mp3"}}`,
			status: http.StatusInternalServerError,
			want:   `{"error":"api_test: test error"}`,
			mount: func(t *testing.T, name string, uri string) error {
				t.Helper()
				if wantName, wantURI := "foo", "nfs://192.168.1.4/export/mp3"; name != wantName || uri != wantURI {
					t.Errorf("called mpd.Mount(ctx, %q, %q); want mpd.Mount(ctx, %q, %q)", name, uri, wantName, wantURI)
				}
				return errTest
			},
		},
		`ok/{"updating":true}`: {
			body:   `{"foo":{"updating":true}}`,
			status: http.StatusAccepted,
			want:   `{}`,
			update: func(t *testing.T, path string) (map[string]string, error) {
				t.Helper()
				if want := "foo"; path != want {
					t.Errorf("called mpd.Update(ctx, %q); want mpd.Update(ctx, %q)", path, want)
				}
				return map[string]string{"updating_db": "1"}, nil
			},
		},
		`error/{"updating":true}`: {
			body:   `{"foo":{"updating":true}}`,
			status: http.StatusInternalServerError,
			want:   `{"error":"api_test: test error"}`,
			update: func(t *testing.T, path string) (map[string]string, error) {
				t.Helper()
				if want := "foo"; path != want {
					t.Errorf("called mpd.Update(ctx, %q); want mpd.Update(ctx, %q)", path, want)
				}
				return nil, errTest
			},
		},
		"ok/null": {
			body:   `{"foo":null}`,
			status: http.StatusAccepted,
			want:   `{}`,
			unmount: func(t *testing.T, name string) error {
				t.Helper()
				if wantName := "foo"; name != wantName {
					t.Errorf("called mpd.Unmount(ctx, %q); want mpd.Unmount(ctx, %q)", name, wantName)
				}
				return nil
			},
			update: func(t *testing.T, path string) (map[string]string, error) {
				t.Helper()
				if want := ""; path != want {
					t.Errorf("called mpd.Update(ctx, %q); want mpd.Update(ctx, %q)", path, want)
				}
				return map[string]string{"updating_db": "1"}, nil
			},
		},
		`ok/{}`: {
			body:   `{"foo":{}}`,
			status: http.StatusAccepted,
			want:   `{}`,
			unmount: func(t *testing.T, name string) error {
				t.Helper()
				if wantName := "foo"; name != wantName {
					t.Errorf("called mpd.Unmount(ctx, %q); want mpd.Unmount(ctx, %q)", name, wantName)
				}
				return nil
			},
			update: func(t *testing.T, path string) (map[string]string, error) {
				t.Helper()
				if want := ""; path != want {
					t.Errorf("called mpd.Update(ctx, %q); want mpd.Update(ctx, %q)", path, want)
				}
				return map[string]string{"updating_db": "1"}, nil
			},
		},
		`ok/{"uri":null}`: {
			body:   `{"foo":{"uri":null}}`,
			status: http.StatusAccepted,
			want:   `{}`,
			unmount: func(t *testing.T, name string) error {
				t.Helper()
				if wantName := "foo"; name != wantName {
					t.Errorf("called mpd.Unmount(ctx, %q); want mpd.Unmount(ctx, %q)", name, wantName)
				}
				return nil
			},
			update: func(t *testing.T, path string) (map[string]string, error) {
				t.Helper()
				if want := ""; path != want {
					t.Errorf("called mpd.Update(ctx, %q); want mpd.Update(ctx, %q)", path, want)
				}
				return map[string]string{"updating_db": "1"}, nil
			},
		},
		`error/{"uri":null}/unmount`: {
			body:   `{"foo":{"uri":null}}`,
			status: http.StatusInternalServerError,
			want:   `{"error":"api_test: test error"}`,
			unmount: func(t *testing.T, name string) error {
				t.Helper()
				if wantName := "foo"; name != wantName {
					t.Errorf("called mpd.Unmount(ctx, %q); want mpd.Unmount(ctx, %q)", name, wantName)
				}
				return errTest
			},
		},
		`error/{"uri":null}/update`: {
			body:   `{"foo":{"uri":null}}`,
			status: http.StatusInternalServerError,
			want:   `{"error":"api_test: test error"}`,
			unmount: func(t *testing.T, name string) error {
				t.Helper()
				if wantName := "foo"; name != wantName {
					t.Errorf("called mpd.Unmount(ctx, %q); want mpd.Unmount(ctx, %q)", name, wantName)
				}
				return nil
			},
			update: func(t *testing.T, path string) (map[string]string, error) {
				t.Helper()
				if want := ""; path != want {
					t.Errorf("called mpd.Update(ctx, %q); want mpd.Update(ctx, %q)", path, want)
				}
				return nil, errTest
			},
		},
	} {
		t.Run(label, func(t *testing.T) {
			mpd := &mpdStorage{t: t, listMounts: tt.listMounts, mount: tt.mount, unmount: tt.unmount, update: tt.update}
			h, err := api.NewStorageHandler(mpd)
			if err != nil {
				t.Fatalf("failed to init Storage: %v", err)
			}
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if got := w.Body.String(); got != tt.want || w.Result().StatusCode != tt.status {
				t.Errorf("ServeHTTP got\n%d %s; want\n%d %s", w.Result().StatusCode, got, tt.status, tt.want)
			}
		})
	}

}

type mpdStorage struct {
	t          *testing.T
	listMounts func(*testing.T) ([]map[string]string, error)
	mount      func(*testing.T, string, string) error
	unmount    func(*testing.T, string) error
	update     func(*testing.T, string) (map[string]string, error)
}

func (a *mpdStorage) ListMounts(context.Context) ([]map[string]string, error) {
	if a.listMounts == nil {
		a.t.Helper()
		a.t.Fatal("no ListMounts mock function")
	}
	return a.listMounts(a.t)
}
func (a *mpdStorage) Mount(ctx context.Context, b string, c string) error {
	if a.mount == nil {
		a.t.Helper()
		a.t.Fatal("no Mount mock function")
	}
	return a.mount(a.t, b, c)
}
func (a *mpdStorage) Unmount(ctx context.Context, b string) error {
	if a.unmount == nil {
		a.t.Helper()
		a.t.Fatal("no Unmount mock function")
	}
	return a.unmount(a.t, b)
}
func (a *mpdStorage) Update(ctx context.Context, b string) (map[string]string, error) {
	if a.update == nil {
		a.t.Helper()
		a.t.Fatal("no Update mock function")
	}
	return a.update(a.t, b)
}
