package api_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/meiraka/vv/internal/vv/api"
)

func TestStatusHandlerGET(t *testing.T) {
	for label, tt := range map[string][]struct {
		label            string
		status           func() (map[string]string, error)
		replayGainStatus func() (map[string]string, error)
		err              error
		want             string
		cache            *api.Status
		changed          bool
		update           string
	}{
		"Update/empty": {{
			status: func() (map[string]string, error) { return map[string]string{}, nil },
			want:   `{"repeat":false,"random":false,"single":false,"oneshot":false,"consume":false,"state":"","song_elapsed":0,"replay_gain":"off","crossfade":0}`,
			cache: &api.Status{
				Repeat:      boolptr(false),
				Random:      boolptr(false),
				Single:      boolptr(false),
				Oneshot:     boolptr(false),
				Consume:     boolptr(false),
				State:       strptr(""),
				SongElapsed: float64ptr(0),
				ReplayGain:  strptr("off"),
				Crossfade:   intptr(0),
			},
			changed: true,
			update:  "Update",
		}},
		"Update/error": {{
			status:  func() (map[string]string, error) { return nil, errTest },
			want:    `{}`,
			err:     errTest,
			cache:   &api.Status{},
			changed: false,
			update:  "Update",
		}},
		"Update/volume": {{
			status: func() (map[string]string, error) { return map[string]string{"volume": "55"}, nil },
			want:   `{"volume":55,"repeat":false,"random":false,"single":false,"oneshot":false,"consume":false,"state":"","song_elapsed":0,"replay_gain":"off","crossfade":0}`,
			cache: &api.Status{
				Volume:      intptr(55),
				Repeat:      boolptr(false),
				Random:      boolptr(false),
				Single:      boolptr(false),
				Oneshot:     boolptr(false),
				Consume:     boolptr(false),
				State:       strptr(""),
				SongElapsed: float64ptr(0),
				ReplayGain:  strptr("off"),
				Crossfade:   intptr(0),
			},
			changed: true,
			update:  "Update",
		}},
		"Update/normal": {{
			status: func() (map[string]string, error) {
				return map[string]string{
					"repeat":         "1",
					"random":         "0",
					"single":         "0",
					"consume":        "0",
					"partition":      "default",
					"playlist":       "4374",
					"playlistlength": "64",
					"mixrampdb":      "0.000000",
					"state":          "pause",
					"song":           "30",
					"songid":         "4337",
					"time":           "250:400",
					"elapsed":        "249.952",
					"bitrate":        "1070",
					"duration":       "399.733",
					"audio":          "44100:16:2",
					"nextsong":       "31",
					"nextsongid":     "4338",
				}, nil
			},
			want: `{"repeat":true,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":249.952,"replay_gain":"off","crossfade":0}`,
			cache: &api.Status{
				Repeat:      boolptr(true),
				Random:      boolptr(false),
				Single:      boolptr(false),
				Oneshot:     boolptr(false),
				Consume:     boolptr(false),
				State:       strptr("pause"),
				SongElapsed: float64ptr(249.952),
				ReplayGain:  strptr("off"),
				Crossfade:   intptr(0),
				Song:        intptr(30),
			},
			changed: true,
			update:  "Update",
		}},
		"UpdateOptions/empty": {{
			status:           func() (map[string]string, error) { return map[string]string{}, nil },
			replayGainStatus: func() (map[string]string, error) { return map[string]string{}, nil },
			want:             `{"repeat":false,"random":false,"single":false,"oneshot":false,"consume":false,"state":"","song_elapsed":0,"replay_gain":"off","crossfade":0}`,
			cache: &api.Status{
				Repeat:      boolptr(false),
				Random:      boolptr(false),
				Single:      boolptr(false),
				Oneshot:     boolptr(false),
				Consume:     boolptr(false),
				State:       strptr(""),
				SongElapsed: float64ptr(0),
				ReplayGain:  strptr("off"),
				Crossfade:   intptr(0),
			},
			changed: true,
			update:  "UpdateOptions",
		}},
		"UpdateOptions/error": {{
			label:            "ReplayGainStatus",
			replayGainStatus: func() (map[string]string, error) { return nil, errTest },
			want:             `{}`,
			err:              errTest,
			cache:            &api.Status{},
			changed:          false,
			update:           "UpdateOptions",
		}, {
			label:            "Status",
			status:           func() (map[string]string, error) { return nil, errTest },
			replayGainStatus: func() (map[string]string, error) { return map[string]string{}, nil },
			want:             `{}`,
			err:              errTest,
			cache:            &api.Status{},
			changed:          false,
			update:           "UpdateOptions",
		}},
		"UpdateOptions/replay_gain_mode": {{
			status:           func() (map[string]string, error) { return map[string]string{}, nil },
			replayGainStatus: func() (map[string]string, error) { return map[string]string{"replay_gain_mode": "track"}, nil },
			want:             `{"repeat":false,"random":false,"single":false,"oneshot":false,"consume":false,"state":"","song_elapsed":0,"replay_gain":"track","crossfade":0}`,
			cache: &api.Status{
				Repeat:      boolptr(false),
				Random:      boolptr(false),
				Single:      boolptr(false),
				Oneshot:     boolptr(false),
				Consume:     boolptr(false),
				State:       strptr(""),
				SongElapsed: float64ptr(0),
				ReplayGain:  strptr("track"),
				Crossfade:   intptr(0),
			},
			changed: true,
			update:  "UpdateOptions",
		}},
		"UpdateOptions/normal/replay_gain_mode": {{
			status: func() (map[string]string, error) {
				return map[string]string{
					"repeat":         "1",
					"random":         "0",
					"single":         "0",
					"consume":        "0",
					"partition":      "default",
					"playlist":       "4374",
					"playlistlength": "64",
					"mixrampdb":      "0.000000",
					"state":          "pause",
					"song":           "30",
					"songid":         "4337",
					"time":           "250:400",
					"elapsed":        "249.952",
					"bitrate":        "1070",
					"duration":       "399.733",
					"audio":          "44100:16:2",
					"nextsong":       "31",
					"nextsongid":     "4338",
				}, nil
			},
			replayGainStatus: func() (map[string]string, error) { return map[string]string{"replay_gain_mode": "track"}, nil },
			want:             `{"repeat":true,"random":false,"single":false,"oneshot":false,"consume":false,"state":"pause","song_elapsed":249.952,"replay_gain":"track","crossfade":0}`,
			cache: &api.Status{
				Repeat:      boolptr(true),
				Random:      boolptr(false),
				Single:      boolptr(false),
				Oneshot:     boolptr(false),
				Consume:     boolptr(false),
				State:       strptr("pause"),
				SongElapsed: float64ptr(249.952),
				ReplayGain:  strptr("track"),
				Crossfade:   intptr(0),
				Song:        intptr(30),
			},
			changed: true,
			update:  "UpdateOptions",
		}},
	} {
		t.Run(label, func(t *testing.T) {
			mpd := &mpdStatus{t: t}
			h, err := api.NewStatusHandler(mpd)
			if err != nil {
				t.Fatalf("api.NewLibrarySongs() = %v, %v", h, err)
			}
			for i := range tt {
				f := func(t *testing.T) {
					mpd.status = tt[i].status
					mpd.replayGainStatus = tt[i].replayGainStatus
					switch tt[i].update {
					case "Update":
						if err := h.Update(context.TODO()); !errors.Is(err, tt[i].err) {
							t.Errorf("handler.Update(context.TODO()) = %v; want %v", err, tt[i].err)
						}
					case "UpdateOptions":
						if err := h.UpdateOptions(context.TODO()); !errors.Is(err, tt[i].err) {
							t.Errorf("handler.UpdateOptions(context.TODO()) = %v; want %v", err, tt[i].err)
						}
					default:
						t.Fatalf("fixme: invalid test case: unsupported update type: %s", tt[i].update)
					}
					r := httptest.NewRequest(http.MethodGet, "/", nil)
					w := httptest.NewRecorder()
					h.ServeHTTP(w, r)
					if status, got := w.Result().StatusCode, w.Body.String(); status != http.StatusOK || got != tt[i].want {
						t.Errorf("ServeHTTP got\n%d %s; want\n%d %s", status, got, http.StatusOK, tt[i].want)
					}
					if cache := h.Cache(); !reflect.DeepEqual(cache, tt[i].cache) {
						t.Errorf("got cache\n%v; want\n%v", cache, tt[i].cache)
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

func TestStatusHandlerPOST(t *testing.T) {
	for label, tt := range map[string]struct {
		body           string
		wantStatus     int
		want           string
		setVol         func(*testing.T, int) error
		repeat         func(*testing.T, bool) error
		random         func(*testing.T, bool) error
		single         func(*testing.T, bool) error
		oneShot        func() error
		consume        func(*testing.T, bool) error
		seekCur        func(*testing.T, float64) error
		replayGainMode func(*testing.T, string) error
		crossfade      func(*testing.T, time.Duration) error
		play           func(*testing.T, int) error
		pause          func(*testing.T, bool) error
		next           func() error
		previous       func() error
	}{
		`ok/{"volume":50}`: {
			body:       `{"volume": 50}`,
			wantStatus: http.StatusAccepted,
			want:       `{}`,
			setVol:     mockIntFunc("mpd.SetVol(ctx, %q)", 50, nil),
		},
		`error/{"volume":50}`: {
			body:       `{"volume": 50}`,
			wantStatus: http.StatusInternalServerError,
			want:       fmt.Sprintf(`{"error":%q}`, errTest.Error()),
			setVol:     mockIntFunc("mpd.SetVol(ctx, %q)", 50, errTest),
		},
		`ok/{"repeat":false}`: {
			body:       `{"repeat":false}`,
			wantStatus: http.StatusAccepted,
			want:       `{}`,
			repeat:     mockBoolFunc("mpd.Repeat(ctx, %v)", false, nil),
		},
		`error/{"repeat":false}`: {
			body:       `{"repeat":false}`,
			wantStatus: http.StatusInternalServerError,
			want:       fmt.Sprintf(`{"error":%q}`, errTest.Error()),
			repeat:     mockBoolFunc("mpd.Repeat(ctx, %v)", false, errTest),
		},
		`ok/{"random":true}`: {
			body:       `{"random":true}`,
			wantStatus: http.StatusAccepted,
			want:       `{}`,
			random:     mockBoolFunc("mpd.Random(ctx, %v)", true, nil),
		},
		`error/{"random":true}`: {
			body:       `{"random":true}`,
			wantStatus: http.StatusInternalServerError,
			want:       fmt.Sprintf(`{"error":%q}`, errTest.Error()),
			random:     mockBoolFunc("mpd.Random(ctx, %v)", true, errTest),
		},
		`ok/{"single":false}`: {
			body:       `{"single":false}`,
			wantStatus: http.StatusAccepted,
			want:       `{}`,
			single:     mockBoolFunc("mpd.Single(ctx, %v)", false, nil),
		},
		`error/{"single":false}`: {
			body:       `{"single":false}`,
			wantStatus: http.StatusInternalServerError,
			want:       fmt.Sprintf(`{"error":%q}`, errTest.Error()),
			single:     mockBoolFunc("mpd.Single(ctx, %v)", false, errTest),
		},
		`ok/{"consume":true}`: {
			body:       `{"consume":true}`,
			wantStatus: http.StatusAccepted,
			want:       `{}`,
			consume:    mockBoolFunc("mpd.Consume(ctx, %v)", true, nil),
		},
		`error/{"consume":true}`: {
			body:       `{"consume":true}`,
			wantStatus: http.StatusInternalServerError,
			want:       fmt.Sprintf(`{"error":%q}`, errTest.Error()),
			consume:    mockBoolFunc("mpd.Consume(ctx, %v)", true, errTest),
		},
		`ok/{"song_elapsed":73.2}`: {
			body:       `{"song_elapsed":73.2}`,
			wantStatus: http.StatusAccepted,
			want:       `{}`,
			seekCur:    mockFloat64Func("mpd.SeekCur(ctx, %v)", 73.2, nil),
		},
		`error/{"song_elapsed":73.2}`: {
			body:       `{"song_elapsed":73.2}`,
			wantStatus: http.StatusInternalServerError,
			want:       fmt.Sprintf(`{"error":%q}`, errTest.Error()),
			seekCur:    mockFloat64Func("mpd.SeekCur(ctx, %v)", 73.2, errTest),
		},
		`ok/{"replay_gain":"album"}`: {
			body:           `{"replay_gain":"album"}`,
			wantStatus:     http.StatusAccepted,
			want:           `{}`,
			replayGainMode: mockStringFunc("mpd.ReplayGainMode(ctx, %q)", "album", nil),
		},
		`error/{"replay_gain":"album"}`: {
			body:           `{"replay_gain":"album"}`,
			wantStatus:     http.StatusInternalServerError,
			want:           fmt.Sprintf(`{"error":%q}`, errTest.Error()),
			replayGainMode: mockStringFunc("mpd.ReplayGainMode(ctx, %q)", "album", errTest),
		},
		`ok/{"crossfade":5}`: {
			body:       `{"crossfade":5}`,
			wantStatus: http.StatusAccepted,
			want:       `{}`,
			crossfade: func(t *testing.T, got time.Duration) error {
				t.Helper()
				if want := 5 * time.Second; got != want {
					t.Errorf("called mpd.Crossfade(ctx, %q); want mpd.Crossfade(ctx, %q)", got, want)
				}
				return nil
			},
		},
		`error/{"crossfade":5}`: {
			body:       `{"crossfade":5}`,
			wantStatus: http.StatusInternalServerError,
			want:       fmt.Sprintf(`{"error":%q}`, errTest.Error()),
			crossfade: func(t *testing.T, got time.Duration) error {
				t.Helper()
				if want := 5 * time.Second; got != want {
					t.Errorf("called mpd.Crossfade(ctx, %q); want mpd.Crossfade(ctx, %q)", got, want)
				}
				return errTest
			},
		},
		`ok/{"state":"play"}`: {
			body:       `{"state":"play"}`,
			wantStatus: http.StatusAccepted,
			want:       `{}`,
			play:       mockIntFunc("mpd.Play(ctx, %q)", -1, nil),
		},
		`error/{"state":"play"}`: {
			body:       `{"state":"play"}`,
			wantStatus: http.StatusInternalServerError,
			want:       fmt.Sprintf(`{"error":%q}`, errTest.Error()),
			play:       mockIntFunc("mpd.Play(ctx, %q)", -1, errTest),
		},
		`ok/{"state":"pause"}`: {
			body:       `{"state":"pause"}`,
			wantStatus: http.StatusAccepted,
			want:       `{}`,
			pause:      mockBoolFunc("mpd.Pause(ctx, %v)", true, nil),
		},
		`error/{"state":"pause"}`: {
			body:       `{"state":"pause"}`,
			wantStatus: http.StatusInternalServerError,
			want:       fmt.Sprintf(`{"error":%q}`, errTest.Error()),
			pause:      mockBoolFunc("mpd.Pause(ctx, %v)", true, errTest),
		},
		`ok/{"state":"next"}`: {
			body:       `{"state":"next"}`,
			wantStatus: http.StatusAccepted,
			want:       `{}`,
			next:       func() error { return nil },
		},
		`error/{"state":"next"}`: {
			body:       `{"state":"next"}`,
			wantStatus: http.StatusInternalServerError,
			want:       fmt.Sprintf(`{"error":%q}`, errTest.Error()),
			next:       func() error { return errTest },
		},
		`ok/{"state":"previous"}`: {
			body:       `{"state":"previous"}`,
			wantStatus: http.StatusAccepted,
			want:       `{}`,
			previous:   func() error { return nil },
		},
		`error/{"state":"previous"}`: {
			body:       `{"state":"previous"}`,
			wantStatus: http.StatusInternalServerError,
			want:       fmt.Sprintf(`{"error":%q}`, errTest.Error()),
			previous:   func() error { return errTest },
		},
		`error/{"state":"unknown"}`: {
			body:       `{"state":"unknown"}`,
			wantStatus: http.StatusBadRequest,
			want:       `{"error":"unknown state: unknown"}`,
		},
	} {
		t.Run(label, func(t *testing.T) {
			mpd := &mpdStatus{
				t:              t,
				setVol:         tt.setVol,
				repeat:         tt.repeat,
				random:         tt.random,
				single:         tt.single,
				oneShot:        tt.oneShot,
				consume:        tt.consume,
				seekCur:        tt.seekCur,
				replayGainMode: tt.replayGainMode,
				crossfade:      tt.crossfade,
				play:           tt.play,
				pause:          tt.pause,
				next:           tt.next,
				previous:       tt.previous,
			}
			h, err := api.NewStatusHandler(mpd)
			if err != nil {
				t.Fatalf("api.NewStatusHandler(mpd) = %v, %v", h, err)
			}
			r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if status, got := w.Result().StatusCode, w.Body.String(); status != tt.wantStatus || got != tt.want {
				t.Errorf("ServeHTTP got\n%d %s; want\n%d %s", status, got, tt.wantStatus, tt.want)
			}
		})
	}
}

func TestStatusHandlerWebSocket(t *testing.T) {
	mpd := &mpdStatus{t: t}
	h, err := api.NewStatusHandler(mpd)
	if err != nil {
		t.Fatalf("api.NewStatusHandler(mpd) = %v, %v", h, err)
	}
	defer h.Close()
	ts := httptest.NewServer(h)
	defer ts.Close()
	ws, _, err := websocket.DefaultDialer.Dial(strings.Replace(ts.URL, "http://", "ws://", 1), nil)
	if err != nil {
		t.Fatalf("failed to connect websocket: %v", err)
	}
	defer ws.Close()
	h.BroadCast("test")
	ws.SetReadDeadline(time.Now().Add(time.Second))
	if _, msg, err := ws.ReadMessage(); string(msg) != "ok" || err != nil {
		t.Fatalf("got message: %s, %v, want: ok <nil>", msg, err)
	}
	if _, msg, err := ws.ReadMessage(); string(msg) != "test" || err != nil {
		t.Fatalf("got message: %s, %v, want: test <nil>", msg, err)
	}
}

type mpdStatus struct {
	t                *testing.T
	status           func() (map[string]string, error)
	replayGainStatus func() (map[string]string, error)
	setVol           func(*testing.T, int) error
	repeat           func(*testing.T, bool) error
	random           func(*testing.T, bool) error
	single           func(*testing.T, bool) error
	oneShot          func() error
	consume          func(*testing.T, bool) error
	seekCur          func(*testing.T, float64) error
	replayGainMode   func(*testing.T, string) error
	crossfade        func(*testing.T, time.Duration) error
	play             func(*testing.T, int) error
	pause            func(*testing.T, bool) error
	next             func() error
	previous         func() error
}

func (m *mpdStatus) Status(context.Context) (map[string]string, error) {
	m.t.Helper()
	if m.status == nil {
		m.t.Fatal("no Status mock function")
	}
	return m.status()
}
func (m *mpdStatus) ReplayGainStatus(context.Context) (map[string]string, error) {
	m.t.Helper()
	if m.replayGainStatus == nil {
		m.t.Fatal("no ReplayGainStatus mock function")
	}
	return m.replayGainStatus()
}
func (m *mpdStatus) SetVol(ctx context.Context, a int) error {
	m.t.Helper()
	if m.setVol == nil {
		m.t.Fatal("no SetVol mock function")
	}
	return m.setVol(m.t, a)
}
func (m *mpdStatus) Repeat(ctx context.Context, a bool) error {
	m.t.Helper()
	if m.repeat == nil {
		m.t.Fatal("no Repeat mock function")
	}
	return m.repeat(m.t, a)
}
func (m *mpdStatus) Random(ctx context.Context, a bool) error {
	m.t.Helper()
	if m.random == nil {
		m.t.Fatal("no Random mock function")
	}
	return m.random(m.t, a)
}
func (m *mpdStatus) Single(ctx context.Context, a bool) error {
	m.t.Helper()
	if m.single == nil {
		m.t.Fatal("no Single mock function")
	}
	return m.single(m.t, a)
}
func (m *mpdStatus) OneShot(context.Context) error {
	m.t.Helper()
	if m.oneShot == nil {
		m.t.Fatal("no OneShot mock function")
	}
	return m.oneShot()
}
func (m *mpdStatus) Consume(ctx context.Context, a bool) error {
	m.t.Helper()
	if m.consume == nil {
		m.t.Fatal("no Consume mock function")
	}
	return m.consume(m.t, a)
}
func (m *mpdStatus) SeekCur(ctx context.Context, a float64) error {
	m.t.Helper()
	if m.seekCur == nil {
		m.t.Fatal("no SeekCur mock function")
	}
	return m.seekCur(m.t, a)
}
func (m *mpdStatus) ReplayGainMode(ctx context.Context, a string) error {
	m.t.Helper()
	if m.replayGainMode == nil {
		m.t.Fatal("no ReplayGainMode mock function")
	}
	return m.replayGainMode(m.t, a)
}
func (m *mpdStatus) Crossfade(ctx context.Context, a time.Duration) error {
	m.t.Helper()
	if m.crossfade == nil {
		m.t.Fatal("no Crossfade mock function")
	}
	return m.crossfade(m.t, a)
}
func (m *mpdStatus) Play(ctx context.Context, a int) error {
	m.t.Helper()
	if m.play == nil {
		m.t.Fatal("no Play mock function")
	}
	return m.play(m.t, a)
}
func (m *mpdStatus) Pause(ctx context.Context, a bool) error {
	m.t.Helper()
	if m.pause == nil {
		m.t.Fatal("no Pause mock function")
	}
	return m.pause(m.t, a)
}
func (m *mpdStatus) Next(context.Context) error {
	m.t.Helper()
	if m.next == nil {
		m.t.Fatal("no Next mock function")
	}
	return m.next()
}
func (m *mpdStatus) Previous(context.Context) error {
	m.t.Helper()
	if m.previous == nil {
		m.t.Fatal("no Previous mock function")
	}
	return m.previous()
}

func intptr(s int) *int {
	return &s
}
func float64ptr(s float64) *float64 {
	return &s
}
func boolptr(s bool) *bool {
	return &s
}
func strptr(s string) *string {
	return &s
}
