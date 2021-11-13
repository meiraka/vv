package api

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/meiraka/vv/internal/mpd"
)

func TestNeighbors(t *testing.T) {
	t.Run("Get", func(t *testing.T) {
		for label, tt := range map[string][]struct {
			mpd  mpdNeighbors
			err  error
			want string
		}{
			"empty": {{
				mpd: mpdNeighborsFunc(func(context.Context) ([]map[string]string, error) {
					return []map[string]string{}, nil
				}),
				want: "{}",
			}},
			"exists": {{
				mpd: mpdNeighborsFunc(func(context.Context) ([]map[string]string, error) {
					return []map[string]string{
						{
							"neighbor": "smb://FOO",
							"name":     "FOO (Samba 4.1.11-Debian)",
						},
					}, nil
				}),
				want: `{"FOO (Samba 4.1.11-Debian)":{"uri":"smb://FOO"}}`,
			}},
			"removed": {{
				mpd: mpdNeighborsFunc(func(context.Context) ([]map[string]string, error) {
					return []map[string]string{
						{
							"neighbor": "smb://FOO",
							"name":     "FOO (Samba 4.1.11-Debian)",
						},
					}, nil
				}),
				want: `{"FOO (Samba 4.1.11-Debian)":{"uri":"smb://FOO"}}`,
			}, {
				mpd: mpdNeighborsFunc(func(context.Context) ([]map[string]string, error) {
					return []map[string]string{}, nil
				}),
				want: "{}",
			}},
			"err": {{
				mpd: mpdNeighborsFunc(func(context.Context) ([]map[string]string, error) {
					return nil, context.DeadlineExceeded
				}),
				err:  context.DeadlineExceeded,
				want: "",
			}},
			"err after exists": {{
				mpd: mpdNeighborsFunc(func(context.Context) ([]map[string]string, error) {
					return []map[string]string{
						{
							"neighbor": "smb://FOO",
							"name":     "FOO (Samba 4.1.11-Debian)",
						},
					}, nil
				}),
				want: `{"FOO (Samba 4.1.11-Debian)":{"uri":"smb://FOO"}}`,
			}, {
				mpd: mpdNeighborsFunc(func(context.Context) ([]map[string]string, error) {
					return nil, context.DeadlineExceeded
				}),
				err:  context.DeadlineExceeded,
				want: `{"FOO (Samba 4.1.11-Debian)":{"uri":"smb://FOO"}}`,
			}},
			"mpd err": {{
				mpd: mpdNeighborsFunc(func(context.Context) ([]map[string]string, error) {
					return nil, &mpd.CommandError{ID: 5, Index: 0, Command: "listneighbors", Message: "unknown command \"listneighbors\""}
				}),
				want: "{}",
			}},
			"mpd err after exists": {{
				mpd: mpdNeighborsFunc(func(context.Context) ([]map[string]string, error) {
					return []map[string]string{
						{
							"neighbor": "smb://FOO",
							"name":     "FOO (Samba 4.1.11-Debian)",
						},
					}, nil
				}),
				want: `{"FOO (Samba 4.1.11-Debian)":{"uri":"smb://FOO"}}`,
			}, {
				mpd: mpdNeighborsFunc(func(context.Context) ([]map[string]string, error) {
					return nil, &mpd.CommandError{ID: 5, Index: 0, Command: "listneighbors", Message: "unknown command \"listneighbors\""}
				}),
				want: "{}",
			}},
		} {
			t.Run(label, func(t *testing.T) {
				cache := newJSONCache()
				defer cache.Close()
				for i := range tt {
					t.Run(fmt.Sprint(i), func(t *testing.T) {
						h := newNeighbors(tt[i].mpd, cache)
						if err := h.Update(context.TODO()); !errors.Is(err, tt[i].err) {
							t.Errorf("Update(ctx) = %v; want %v", err, tt[i].err)
						}
						r := httptest.NewRequest(http.MethodGet, "/", nil)
						w := httptest.NewRecorder()
						h.ServeHTTP(w, r)
						if got := w.Body.String(); got != tt[i].want {
							t.Errorf("ServeHTTP(updated) got %q; want %q", got, tt[i].want)
						}
					})
				}
			})
		}
	})

}

type mpdNeighborsFunc func(context.Context) ([]map[string]string, error)

func (m mpdNeighborsFunc) ListNeighbors(ctx context.Context) ([]map[string]string, error) {
	return m(ctx)
}
