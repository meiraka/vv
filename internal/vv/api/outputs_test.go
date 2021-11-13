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

func TestOutputsHandlerGET(t *testing.T) {
	proxy := map[string]string{"Ogg Stream": "localhost:8080/"}
	for label, tt := range map[string][]struct {
		label   string
		outputs func() ([]*mpd.Output, error)
		err     error
		want    string
		changed bool
	}{
		"ok": {{
			label:   "empty",
			outputs: func() ([]*mpd.Output, error) { return []*mpd.Output{}, nil },
			want:    `{}`,
		}, {
			label: "minimal",
			outputs: func() ([]*mpd.Output, error) {
				return []*mpd.Output{{
					ID:      "0",
					Name:    "My ALSA Device",
					Plugin:  "alsa",
					Enabled: true,
				}}, nil
			},
			want:    `{"0":{"name":"My ALSA Device","plugin":"alsa","enabled":true}}`,
			changed: true,
		}, {
			label:   "remove",
			outputs: func() ([]*mpd.Output, error) { return []*mpd.Output{}, nil },
			want:    `{}`,
			changed: true,
		}},
		"ok/dop": {{
			outputs: func() ([]*mpd.Output, error) {
				return []*mpd.Output{{
					ID:         "0",
					Name:       "My ALSA Device",
					Plugin:     "alsa",
					Enabled:    true,
					Attributes: map[string]string{"dop": "0"},
				}}, nil
			},
			want:    `{"0":{"name":"My ALSA Device","plugin":"alsa","enabled":true,"attributes":{"dop":false}}}`,
			changed: true,
		}},
		"ok/allowed formats": {{
			outputs: func() ([]*mpd.Output, error) {
				return []*mpd.Output{{
					ID:         "0",
					Name:       "My ALSA Device",
					Plugin:     "alsa",
					Enabled:    true,
					Attributes: map[string]string{"allowed_formats": "96000:16:* 192000:24:* dsd64:=dop *:dsd:"},
				}}, nil
			},
			want:    `{"0":{"name":"My ALSA Device","plugin":"alsa","enabled":true,"attributes":{"allowed_formats":["96000:16:*","192000:24:*","dsd64:=dop","*:dsd:"]}}}`,
			changed: true,
		}},
		"ok/stream": {{
			outputs: func() ([]*mpd.Output, error) {
				return []*mpd.Output{{
					ID:         "0",
					Name:       "My ALSA Device",
					Plugin:     "alsa",
					Enabled:    true,
					Attributes: map[string]string{"dop": "0"},
				}, {
					ID:      "1",
					Name:    "Ogg Stream",
					Plugin:  "http",
					Enabled: true,
				}}, nil
			},
			want:    `{"0":{"name":"My ALSA Device","plugin":"alsa","enabled":true,"attributes":{"dop":false}},"1":{"name":"Ogg Stream","plugin":"http","enabled":true,"stream":"/api/music/outputs/stream?name=Ogg+Stream"}}`,
			changed: true,
		}},
		"error": {{
			label: "prepare data",
			outputs: func() ([]*mpd.Output, error) {
				return []*mpd.Output{{
					ID:         "0",
					Name:       "My ALSA Device",
					Plugin:     "alsa",
					Enabled:    true,
					Attributes: map[string]string{"dop": "0"},
				}}, nil
			},
			want:    `{"0":{"name":"My ALSA Device","plugin":"alsa","enabled":true,"attributes":{"dop":false}}}`,
			changed: true,
		}, {
			label:   "error",
			outputs: func() ([]*mpd.Output, error) { return nil, errTest },
			err:     errTest,
			want:    `{"0":{"name":"My ALSA Device","plugin":"alsa","enabled":true,"attributes":{"dop":false}}}`,
		}},
	} {
		t.Run(label, func(t *testing.T) {
			mpd := &mpdOutputs{t: t}
			h, err := api.NewOutputsHandler(mpd, proxy)
			if err != nil {
				t.Fatalf("api.NewOutputsHandler(mpd) = %v", err)
			}
			defer h.Close()
			for i := range tt {
				f := func(t *testing.T) {
					mpd.t = t
					mpd.outputs = tt[i].outputs
					if err := h.Update(context.TODO()); !errors.Is(err, tt[i].err) {
						t.Errorf("h.Update(context.TODO()) = %v; want %v", err, tt[i].err)
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
					t.Run(tt[i].label, f)
				} else {
					f(t)
				}
			}
		})
	}
}
func TestOutputsHandlerPOST(t *testing.T) {
	for label, tt := range map[string]struct {
		body          string
		wantStatus    int
		want          string
		enableOutput  func(*testing.T, string) error
		disableOutput func(*testing.T, string) error
		outputSet     func(*testing.T, string, string, string) error
	}{
		`error/invalid json`: {
			body:       `invalid json`,
			want:       `{"error":"invalid character 'i' looking for beginning of value"}`,
			wantStatus: http.StatusBadRequest,
		},
		`ok/{"enabled":true}`: {
			body:         `{"0":{"enabled":true}}`,
			wantStatus:   http.StatusAccepted,
			want:         `{}`,
			enableOutput: mockStringFunc("mpd.EnableOutput(ctx, %q)", "0", nil),
		},
		`error/{"enabled":true}`: {
			body:         `{"0":{"enabled":true}}`,
			wantStatus:   http.StatusInternalServerError,
			want:         `{"error":"api_test: test error"}`,
			enableOutput: mockStringFunc("mpd.EnableOutput(ctx, %q)", "0", errTest),
		},
		`ok/{"enabled":false}`: {
			body:          `{"1":{"enabled":false}}`,
			wantStatus:    http.StatusAccepted,
			want:          `{}`,
			disableOutput: mockStringFunc("mpd.DisableOutput(ctx, %q)", "1", nil),
		},
		`error/{"enabled":false}`: {
			body:          `{"1":{"enabled":false}}`,
			wantStatus:    http.StatusInternalServerError,
			want:          `{"error":"api_test: test error"}`,
			disableOutput: mockStringFunc("mpd.DisableOutput(ctx, %q)", "1", errTest),
		},
		`ok/{"attributes":{"dop":true}}`: {
			body:       `{"1000":{"attributes":{"dop":true}}}`,
			wantStatus: http.StatusAccepted,
			want:       `{}`,
			outputSet: func(t *testing.T, a, b, c string) error {
				t.Helper()
				if wa, wb, wc := "1000", "dop", "1"; a != wa || b != wb || c != wc {
					t.Errorf(`called mpd.OutputSet(ctx, %q, %q, %q); want mpd.OutputSet(ctx, %q, %q, %q)`, a, b, c, wa, wb, wc)
				}
				return nil
			},
		},
		`ok/{"attributes":{"dop":false}}`: {
			body:       `{"1001":{"attributes":{"dop":false}}}`,
			wantStatus: http.StatusAccepted,
			want:       `{}`,
			outputSet: func(t *testing.T, a, b, c string) error {
				t.Helper()
				if wa, wb, wc := "1001", "dop", "0"; a != wa || b != wb || c != wc {
					t.Errorf(`called mpd.OutputSet(ctx, %q, %q, %q); want mpd.OutputSet(ctx, %q, %q, %q)`, a, b, c, wa, wb, wc)
				}
				return nil
			},
		},
		`error/{"attributes":{"dop":false}}`: {
			body:       `{"1002":{"attributes":{"dop":false}}}`,
			wantStatus: http.StatusInternalServerError,
			want:       `{"error":"api_test: test error"}`,
			outputSet: func(t *testing.T, a, b, c string) error {
				t.Helper()
				if wa, wb, wc := "1002", "dop", "0"; a != wa || b != wb || c != wc {
					t.Errorf(`called mpd.OutputSet(ctx, %q, %q, %q); want mpd.OutputSet(ctx, %q, %q, %q)`, a, b, c, wa, wb, wc)
				}
				return errTest
			},
		},
		`ok/{"attributes":{"allowed_formats":["dsd64:2","dsd128:2"]}}`: {
			body:       `{"1003":{"attributes":{"allowed_formats":["dsd64:2","dsd128:2"]}}}`,
			wantStatus: http.StatusAccepted,
			want:       `{}`,
			outputSet: func(t *testing.T, a, b, c string) error {
				t.Helper()
				if wa, wb, wc := "1003", "allowed_formats", "dsd64:2 dsd128:2"; a != wa || b != wb || c != wc {
					t.Errorf(`called mpd.OutputSet(ctx, %q, %q, %q); want mpd.OutputSet(ctx, %q, %q, %q)`, a, b, c, wa, wb, wc)
				}
				return nil
			},
		},
		`error/{"attributes":{"allowed_formats":["invalid allowed formats"]}}`: {
			body:       `{"1003":{"attributes":{"allowed_formats":["invalid allowed formats"]}}}`,
			wantStatus: http.StatusBadRequest,
			want:       `{"error":"api: invalid allowed formats: #0: \"invalid allowed formats\""}`,
		},
		`ok/{"attributes":{"allowed_formats":[]}}`: {
			body:       `{"1004":{"attributes":{"allowed_formats":[]}}}`,
			wantStatus: http.StatusAccepted,
			want:       `{}`,
			outputSet: func(t *testing.T, a, b, c string) error {
				t.Helper()
				if wa, wb, wc := "1004", "allowed_formats", ""; a != wa || b != wb || c != wc {
					t.Errorf(`called mpd.OutputSet(ctx, %q, %q, %q); want mpd.OutputSet(ctx, %q, %q, %q)`, a, b, c, wa, wb, wc)
				}
				return nil
			},
		},
	} {
		t.Run(label, func(t *testing.T) {
			mpd := &mpdOutputs{t: t, enableOutput: tt.enableOutput, disableOutput: tt.disableOutput, outputSet: tt.outputSet}
			h, err := api.NewOutputsHandler(mpd, map[string]string{})
			if err != nil {
				t.Fatalf("api.NewOutputsHandler(mpd) = %v, %v", h, err)
			}
			defer h.Close()
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if status, got := w.Result().StatusCode, w.Body.String(); status != tt.wantStatus || got != tt.want {
				t.Errorf("ServeHTTP got\n%d %s; want\n%d %s", status, got, tt.wantStatus, tt.want)
			}
		})

	}
}

type mpdOutputs struct {
	t             *testing.T
	enableOutput  func(*testing.T, string) error
	disableOutput func(*testing.T, string) error
	outputSet     func(*testing.T, string, string, string) error
	outputs       func() ([]*mpd.Output, error)
}

func (m *mpdOutputs) EnableOutput(ctx context.Context, a string) error {
	m.t.Helper()
	if m.enableOutput == nil {
		m.t.Fatal("no EnableOutput mock function")
	}
	return m.enableOutput(m.t, a)
}
func (m *mpdOutputs) DisableOutput(ctx context.Context, a string) error {
	m.t.Helper()
	if m.disableOutput == nil {
		m.t.Fatal("no DisableOutput mock function")
	}
	return m.disableOutput(m.t, a)
}
func (m *mpdOutputs) OutputSet(ctx context.Context, a string, b string, c string) error {
	m.t.Helper()
	if m.outputSet == nil {
		m.t.Fatal("no OutputSet mock function")
	}
	return m.outputSet(m.t, a, b, c)
}
func (m *mpdOutputs) Outputs(context.Context) ([]*mpd.Output, error) {
	m.t.Helper()
	if m.outputs == nil {
		m.t.Fatal("no Outputs mock function")
	}
	return m.outputs()
}
