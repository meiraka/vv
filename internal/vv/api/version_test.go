package api_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"

	"github.com/meiraka/vv/internal/vv/api"
)

func TestVersionHandlerGET(t *testing.T) {
	goVersion := fmt.Sprintf("%s %s %s", runtime.Version(), runtime.GOOS, runtime.GOARCH)
	appVersion := "baz"
	for label, tt := range map[string]struct {
		version func() string
		err     error
		want    string
		changed bool
		update  string
	}{
		"update": {
			version: func() string { return "foobar" },
			want:    fmt.Sprintf(`{"app":"%s","go":"%s","mpd":"foobar"}`, appVersion, goVersion),
			changed: true,
			update:  "Update",
		},
		"update/unknown": {
			version: func() string { return "" },
			want:    fmt.Sprintf(`{"app":"%s","go":"%s","mpd":"unknown"}`, appVersion, goVersion),
			changed: true,
			update:  "Update",
		},
		"update no mpd": {
			want:    fmt.Sprintf(`{"app":"%s","go":"%s","mpd":""}`, appVersion, goVersion),
			changed: true,
			update:  "UpdateNoMPD",
		},
	} {
		t.Run(label, func(t *testing.T) {
			mpd := &mpdVersion{t: t}
			h, err := api.NewVersionHandler(mpd, appVersion)
			if err != nil {
				t.Fatalf("failed to init Neighbors: %v", err)
			}
			defer h.Close()
			mpd.t = t
			mpd.version = tt.version
			switch tt.update {
			case "Update":
				if err := h.Update(); !errors.Is(err, tt.err) {
					t.Errorf("handler.Update() = %v; want %v", err, tt.err)
				}
			case "UpdateNoMPD":
				if err := h.UpdateNoMPD(); !errors.Is(err, tt.err) {
					t.Errorf("handler.UpdateNoMPD() = %v; want %v", err, tt.err)
				}
			}
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if got, status, wantStatus := w.Body.String(), w.Result().StatusCode, http.StatusOK; got != tt.want || status != wantStatus {
				t.Errorf("ServeHTTP got\n%d %s; want\n%d %s", status, got, wantStatus, tt.want)
			}
			if changed := recieveMsg(h.Changed()); changed != tt.changed {
				t.Errorf("changed = %v; want %v", changed, tt.changed)
			}
		})
	}
}

type mpdVersion struct {
	t       *testing.T
	version func() string
}

func (m *mpdVersion) Version() string {
	m.t.Helper()
	if m.version == nil {
		m.t.Fatal("no Version mock function")
	}
	return m.version()
}
