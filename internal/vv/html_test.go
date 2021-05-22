package vv

import (
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func TestHTMLHandler(t *testing.T) {
	for label, tt := range map[string]struct {
		config *HTMLConfig
		hasErr bool
	}{
		"nil config":   {},
		"empty config": {config: &HTMLConfig{}},
		"local dir": {
			config: &HTMLConfig{Local: true, LocalDir: filepath.Join("..", "..", "assets"), LocalDate: time.Now()},
		},
		"tree config only": {
			config: &HTMLConfig{Tree: Tree{"AlbumArtist": DefaultTree["AlbumArtist"]}},
			hasErr: true,
		},
		"tree order config only": {
			config: &HTMLConfig{TreeOrder: []string{"AlbumArtist"}},
			hasErr: true,
		},
		"custom order": {
			config: &HTMLConfig{
				Tree:      Tree{"AlbumArtist": DefaultTree["AlbumArtist"]},
				TreeOrder: []string{"AlbumArtist"},
			},
		},
	} {
		t.Run(label, func(t *testing.T) {
			// test constructor
			h, err := NewHTMLHander(tt.config)
			if tt.hasErr {
				if h != nil || err == nil {
					t.Errorf("got %v, %v; want <nil>, non-nil error", h, err)
				}
				return
			}
			if h == nil || err != nil {
				t.Fatalf("got %v, %v; want non-nil handler, <nil>", h, err)
			}
			// test handler resp status is 200
			req := httptest.NewRequest("GET", "http://localhost:8080/", nil)
			w := httptest.NewRecorder()
			h.ServeHTTP(w, req)

			resp := w.Result()
			if resp.StatusCode != 200 {
				t.Errorf("got status %d; want 200", resp.StatusCode)
			}
		})
	}

}
