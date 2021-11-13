package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Status struct {
	Volume      *int     `json:"volume,omitempty"`
	Repeat      *bool    `json:"repeat,omitempty"`
	Random      *bool    `json:"random,omitempty"`
	Single      *bool    `json:"single,omitempty"`
	Oneshot     *bool    `json:"oneshot,omitempty"`
	Consume     *bool    `json:"consume,omitempty"`
	State       *string  `json:"state,omitempty"`
	SongElapsed *float64 `json:"song_elapsed,omitempty"`
	ReplayGain  *string  `json:"replay_gain,omitempty"`
	Crossfade   *int     `json:"crossfade,omitempty"`

	Updating bool    `json:"-"`
	Error    *string `json:"-"`
	Song     *int    `json:"-"`
}

type MPDStatus interface {
	Status(context.Context) (map[string]string, error)
	ReplayGainStatus(context.Context) (map[string]string, error)
	SetVol(context.Context, int) error
	Repeat(context.Context, bool) error
	Random(context.Context, bool) error
	Single(context.Context, bool) error
	OneShot(context.Context) error
	Consume(context.Context, bool) error
	SeekCur(context.Context, float64) error
	ReplayGainMode(context.Context, string) error
	Crossfade(context.Context, time.Duration) error
	Play(context.Context, int) error
	Pause(context.Context, bool) error
	Next(context.Context) error
	Previous(context.Context) error
}

type StatusHandler struct {
	mpd        MPDStatus
	cache      *cache
	data       *Status
	replayGain map[string]string
	changed    chan struct{}

	upgrader websocket.Upgrader
	mu       sync.RWMutex
	subs     []chan string
}

func NewStatusHandler(mpd MPDStatus) (*StatusHandler, error) {
	data := &Status{}
	c, err := newCache(data)
	if err != nil {
		return nil, err
	}
	return &StatusHandler{
		mpd:     mpd,
		cache:   c,
		data:    data,
		changed: make(chan struct{}, cap(c.Changed())),
		subs:    make([]chan string, 0, 10),
	}, nil
}

func (a *StatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("Upgrade") == "websocket" {
		a.websocket(w, r)
		return
	}
	if r.Method == http.MethodPost {
		a.post(w, r)
		return
	}
	a.cache.ServeHTTP(w, r)
}

// Broadcast broadcasts messages to websocket mpds.
func (a *StatusHandler) BroadCast(s string) {
	a.mu.Lock()
	for _, c := range a.subs {
		select {
		case c <- s:
		default:
		}
	}
	a.mu.Unlock()
}

func (a *StatusHandler) Update(ctx context.Context) error {
	s, err := a.mpd.Status(ctx)
	if err != nil {
		return err
	}
	var volume *int
	v, err := strconv.Atoi(s["volume"])
	if err == nil && v >= 0 {
		volume = &v
	}
	var pos *int
	p, err := strconv.Atoi(s["song"])
	if err == nil {
		pos = &p
	}
	elapsed, err := strconv.ParseFloat(s["elapsed"], 64)
	if err != nil {
		elapsed = 0
		// return fmt.Errorf("elapsed: %v", err)
	}
	// TODO: add millisec update time to JSON
	a.mu.Lock()
	replayGain, ok := a.replayGain["replay_gain_mode"]
	a.mu.Unlock()
	if !ok {
		replayGain = "off"
	}
	crossfade, err := strconv.Atoi(s["xfade"])
	if err != nil {
		crossfade = 0
	}
	_, updating := s["updating_db"]
	var errstr *string
	if err, ok := s["error"]; ok {
		errstr = &err
	}
	data := &Status{
		Volume:      volume,
		Repeat:      boolPtr(s["repeat"] == "1"),
		Random:      boolPtr(s["random"] == "1"),
		Single:      boolPtr(s["single"] == "1"),
		Oneshot:     boolPtr(s["single"] == "oneshot"),
		Consume:     boolPtr(s["consume"] == "1"),
		State:       stringPtr(s["state"]),
		SongElapsed: &elapsed,
		ReplayGain:  &replayGain,
		Crossfade:   &crossfade,

		Song:     pos,
		Updating: updating,
		Error:    errstr,
	}
	// force update to update Last-Modified header to calc current SongElapsed
	a.mu.Lock()
	defer a.mu.Unlock()
	if err := a.cache.Set(data); err != nil {
		return err
	}
	a.data = data
	select {
	case a.changed <- struct{}{}:
	default:
	}
	return nil
}

func (a *StatusHandler) UpdateOptions(ctx context.Context) error {
	s, err := a.mpd.ReplayGainStatus(ctx)
	if err != nil {
		return err
	}
	a.mu.Lock()
	a.replayGain = s
	a.mu.Unlock()
	return a.Update(ctx)
}

func (a *StatusHandler) Cache() *Status {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.data
}

// Changed returns status update event chan.
func (a *StatusHandler) Changed() <-chan struct{} {
	return a.changed
}

// Close closes update event chan.
func (a *StatusHandler) Close() {
	a.cache.Close()
	close(a.changed)
}

func (a *StatusHandler) post(w http.ResponseWriter, r *http.Request) {
	var s Status
	if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
		writeHTTPError(w, http.StatusBadRequest, err)
		return
	}
	ctx := r.Context()
	now := time.Now().UTC()
	changed := false
	if s.Volume != nil {
		if err := a.mpd.SetVol(ctx, *s.Volume); err != nil {
			writeHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		changed = true
	}
	if s.Repeat != nil {
		if err := a.mpd.Repeat(ctx, *s.Repeat); err != nil {
			writeHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		changed = true
	}
	if s.Random != nil {
		if err := a.mpd.Random(ctx, *s.Random); err != nil {
			writeHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		changed = true
	}
	if s.Single != nil {
		if err := a.mpd.Single(ctx, *s.Single); err != nil {
			writeHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		changed = true
	}
	if s.Oneshot != nil {
		if err := a.mpd.OneShot(ctx); err != nil {
			writeHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		changed = true
	}
	if s.Consume != nil {
		if err := a.mpd.Consume(ctx, *s.Consume); err != nil {
			writeHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		changed = true
	}
	if s.SongElapsed != nil {
		if err := a.mpd.SeekCur(ctx, *s.SongElapsed); err != nil {
			writeHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		changed = true
	}
	if s.ReplayGain != nil {
		if err := a.mpd.ReplayGainMode(ctx, *s.ReplayGain); err != nil {
			writeHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		changed = true
	}
	if s.Crossfade != nil {
		if err := a.mpd.Crossfade(ctx, time.Duration(*s.Crossfade)*time.Second); err != nil {
			writeHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		changed = true
	}
	if s.State != nil {
		var err error
		switch *s.State {
		case "play":
			err = a.mpd.Play(ctx, -1)
		case "pause":
			err = a.mpd.Pause(ctx, true)
		case "next":
			err = a.mpd.Next(ctx)
		case "previous":
			err = a.mpd.Previous(ctx)
		default:
			writeHTTPError(w, http.StatusBadRequest, fmt.Errorf("unknown state: %s", *s.State))
			return
		}
		if err != nil {
			writeHTTPError(w, http.StatusInternalServerError, err)
			return
		}
		changed = true
	}
	r.Method = "GET"
	if changed {
		r = setUpdateTime(r, now)
	}
	a.cache.ServeHTTP(w, r)
}

func (a *StatusHandler) websocket(w http.ResponseWriter, r *http.Request) {
	ws, err := a.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	c := make(chan string, 100)
	a.mu.Lock()
	a.subs = append(a.subs, c)
	a.mu.Unlock()
	defer func() {
		a.mu.Lock()
		n := make([]chan string, len(a.subs)-1, len(a.subs)+10)
		diff := 0
		for i, ec := range a.subs {
			if ec == c {
				diff = -1
			} else {
				n[i+diff] = ec
			}
		}
		a.subs = n
		close(c)
		ws.Close()
		a.mu.Unlock()
	}()
	if err := ws.WriteMessage(websocket.TextMessage, []byte("ok")); err != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		defer cancel()
		for {
			_, _, err := ws.ReadMessage()
			if err != nil {
				return
			}
		}
	}()
	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-c:
			if !ok {
				return
			}
			if err := ws.WriteMessage(websocket.TextMessage, []byte(e)); err != nil {
				return
			}
		case <-time.After(time.Second * 5):
			if err := ws.WriteMessage(websocket.TextMessage, []byte("ping")); err != nil {
				return
			}
		}

	}

}
