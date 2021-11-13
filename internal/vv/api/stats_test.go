package api_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/meiraka/vv/internal/vv/api"
)

func TestStatsHandlerGET(t *testing.T) {
	for label, tt := range map[string][]struct {
		label   string
		stats   func() (map[string]string, error)
		err     error
		want    string
		changed bool
	}{
		"ok": {{
			label: "empty",
			stats: func() (map[string]string, error) {
				return map[string]string{}, nil
			},
			want:    `{"uptime":0,"playtime":0,"artists":0,"albums":0,"songs":0,"library_playtime":0,"library_update":0}`,
			changed: true,
		}, {
			label: "all",
			stats: func() (map[string]string, error) {
				return map[string]string{
					"artists":     "6",
					"albums":      "5",
					"songs":       "4",
					"uptime":      "3",
					"db_playtime": "2",
					"db_update":   "1",
					"playtime":    "10",
				}, nil
			},
			want:    `{"uptime":3,"playtime":10,"artists":6,"albums":5,"songs":4,"library_playtime":2,"library_update":1}`,
			changed: true,
		}, {
			label: "remove",
			stats: func() (map[string]string, error) {
				return map[string]string{}, nil
			},
			want:    `{"uptime":0,"playtime":0,"artists":0,"albums":0,"songs":0,"library_playtime":0,"library_update":0}`,
			changed: true,
		}},
		"error": {{
			label: "prepare data",
			stats: func() (map[string]string, error) {
				return map[string]string{"uptime": "100"}, nil
			},
			want:    `{"uptime":100,"playtime":0,"artists":0,"albums":0,"songs":0,"library_playtime":0,"library_update":0}`,
			changed: true,
		}, {
			label: "error",
			stats: func() (map[string]string, error) {
				return nil, errTest
			},
			err:     errTest,
			want:    `{"uptime":100,"playtime":0,"artists":0,"albums":0,"songs":0,"library_playtime":0,"library_update":0}`,
			changed: false,
		}},
	} {
		t.Run(label, func(t *testing.T) {
			mpd := &mpdStats{t: t}
			h, err := api.NewStatsHandler(mpd)
			if err != nil {
				t.Fatalf("api.NewStatsHandler(mpd) = %v", err)
			}
			defer h.Close()
			for i := range tt {
				f := func(t *testing.T) {
					mpd.t = t
					mpd.stats = tt[i].stats
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

type mpdStats struct {
	t     *testing.T
	stats func() (map[string]string, error)
}

func (m *mpdStats) Stats(context.Context) (map[string]string, error) {
	m.t.Helper()
	if m.stats == nil {
		m.t.Fatal("no Stats mock function")
	}
	return m.stats()
}
