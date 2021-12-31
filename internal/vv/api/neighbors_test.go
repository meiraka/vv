package api_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/meiraka/vv/internal/log"
	"github.com/meiraka/vv/internal/mpd"
	"github.com/meiraka/vv/internal/vv/api"
)

func TestNeighborsHandlerGET(t *testing.T) {
	for label, tt := range map[string][]struct {
		label         string
		listNeighbors func() ([]map[string]string, error)
		err           error
		want          string
		changed       bool
	}{
		"ok": {{
			label: "empty",
			listNeighbors: func() ([]map[string]string, error) {
				return []map[string]string{}, nil
			},
			want: "{}",
		}, {
			label: "some data",
			listNeighbors: func() ([]map[string]string, error) {
				return []map[string]string{
					{
						"neighbor": "smb://FOO",
						"name":     "FOO (Samba 4.1.11-Debian)",
					},
				}, nil
			},
			want:    `{"FOO (Samba 4.1.11-Debian)":{"uri":"smb://FOO"}}`,
			changed: true,
		}, {
			label: "remove",
			listNeighbors: func() ([]map[string]string, error) {
				return []map[string]string{}, nil
			},
			want:    "{}",
			changed: true,
		}},
		"error/network": {{
			label: "prepare data",
			listNeighbors: func() ([]map[string]string, error) {
				return []map[string]string{
					{
						"neighbor": "smb://FOO",
						"name":     "FOO (Samba 4.1.11-Debian)",
					},
				}, nil
			},
			want:    `{"FOO (Samba 4.1.11-Debian)":{"uri":"smb://FOO"}}`,
			changed: true,
		}, {
			label: "error",
			listNeighbors: func() ([]map[string]string, error) {
				return nil, errTest
			},
			err:  errTest,
			want: `{"FOO (Samba 4.1.11-Debian)":{"uri":"smb://FOO"}}`,
		}},
		"error/mpd": {{
			label: "prepare data",
			listNeighbors: func() ([]map[string]string, error) {
				return []map[string]string{
					{
						"neighbor": "smb://FOO",
						"name":     "FOO (Samba 4.1.11-Debian)",
					},
				}, nil
			},
			want:    `{"FOO (Samba 4.1.11-Debian)":{"uri":"smb://FOO"}}`,
			changed: true,
		}, {
			label: "unknown command",
			listNeighbors: func() ([]map[string]string, error) {
				return nil, &mpd.CommandError{ID: 5, Index: 0, Command: "listneighbors", Message: "unknown command \"listneighbors\""}
			},
			want:    "{}",
			changed: true,
		}},
	} {
		t.Run(label, func(t *testing.T) {
			mpd := &mpdNeighbors{t: t}
			h, err := api.NewNeighborsHandler(mpd, log.NewTestLogger(t))
			if err != nil {
				t.Fatalf("failed to init Neighbors: %v", err)
			}
			for i := range tt {
				t.Run(tt[i].label, func(t *testing.T) {
					mpd.t = t
					mpd.listNeighbors = tt[i].listNeighbors
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
				})
			}
		})
	}
}

type mpdNeighbors struct {
	t             *testing.T
	listNeighbors func() ([]map[string]string, error)
}

func (m *mpdNeighbors) ListNeighbors(ctx context.Context) ([]map[string]string, error) {
	m.t.Helper()
	if m.listNeighbors == nil {
		m.t.Fatal("no ListNeighbors mock function")
	}
	return m.listNeighbors()
}
