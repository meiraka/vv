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

const (
	pathAPIMusicStatus = "/api/music"
)

// status

type httpMusicStatus struct {
	Volume      *int     `json:"volume,omitempty"`
	Repeat      *bool    `json:"repeat,omitempty"`
	Random      *bool    `json:"random,omitempty"`
	Single      *bool    `json:"single,omitempty"`
	Oneshot     *bool    `json:"oneshot,omitempty"`
	Consume     *bool    `json:"consume,omitempty"`
	State       *string  `json:"state,omitempty"`
	Pos         *int     `json:"pos,omitempty"`
	SongElapsed *float64 `json:"song_elapsed,omitempty"`
	ReplayGain  *string  `json:"replay_gain"`
	Crossfade   *int     `json:"crossfade"`
}

func (a *api) StatusHandler() http.HandlerFunc {
	rest := a.statusHandlerHTTP()
	subs := make([]chan string, 0, 10)
	var mu sync.Mutex

	go func() {
		for e := range a.jsonCache.Event() {
			mu.Lock()
			for _, c := range subs {
				select {
				case c <- e:
				default:
				}
			}
			mu.Unlock()
		}
	}()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Upgrade") != "websocket" {
			rest.ServeHTTP(w, r)
			return
		}
		ws, err := a.upgrader.Upgrade(w, r, nil)
		if err != nil {
			rest.ServeHTTP(w, r)
			return
		}
		c := make(chan string, 100)
		mu.Lock()
		subs = append(subs, c)
		mu.Unlock()
		defer func() {
			mu.Lock()
			n := make([]chan string, len(subs)-1, len(subs)+10)
			diff := 0
			for i, ec := range subs {
				if ec == c {
					diff = -1
				} else {
					n[i+diff] = ec
				}
			}
			subs = n
			close(c)
			ws.Close()
			mu.Unlock()
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
}

func (a *api) statusHandlerHTTP() http.HandlerFunc {
	get := a.jsonCache.Handler(pathAPIMusicStatus)
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			get.ServeHTTP(w, r)
			return
		}
		var s httpMusicStatus
		if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
			writeHTTPError(w, http.StatusBadRequest, err)
			return
		}
		ctx := r.Context()
		now := time.Now().UTC()
		changed := false
		if s.Volume != nil {
			if err := a.client.SetVol(ctx, *s.Volume); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			changed = true
		}
		if s.Repeat != nil {
			if err := a.client.Repeat(ctx, *s.Repeat); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			changed = true
		}
		if s.Random != nil {
			if err := a.client.Random(ctx, *s.Random); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			changed = true
		}
		if s.Single != nil {
			if err := a.client.Single(ctx, *s.Single); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			changed = true
		}
		if s.Oneshot != nil {
			if err := a.client.OneShot(ctx); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			changed = true
		}
		if s.Consume != nil {
			if err := a.client.Consume(ctx, *s.Consume); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			changed = true
		}
		if s.SongElapsed != nil {
			if err := a.client.SeekCur(ctx, *s.SongElapsed); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			changed = true
		}
		if s.ReplayGain != nil {
			if err := a.client.ReplayGainMode(ctx, *s.ReplayGain); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			changed = true
		}
		if s.Crossfade != nil {
			if err := a.client.Crossfade(ctx, time.Duration(*s.Crossfade)*time.Second); err != nil {
				writeHTTPError(w, http.StatusInternalServerError, err)
				return
			}
			changed = true
		}
		if s.State != nil {
			var err error
			switch *s.State {
			case "play":
				err = a.client.Play(ctx, -1)
			case "pause":
				err = a.client.Pause(ctx, true)
			case "next":
				err = a.client.Next(ctx)
			case "previous":
				err = a.client.Previous(ctx)
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
		get.ServeHTTP(w, r)
	}
}

func (a *api) updateStatus(ctx context.Context) error {
	s, err := a.client.Status(ctx)
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
	// force update to Last-Modified header to calc current SongElapsed
	// TODO: add millisec update time to JSON
	a.mu.Lock()
	replayGain := a.replayGain["replay_gain_mode"]
	a.mu.Unlock()
	crossfade, err := strconv.Atoi(s["xfade"])
	if err != nil {
		crossfade = 0
	}
	if err := a.jsonCache.Set(pathAPIMusicStatus, &httpMusicStatus{
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
	}); err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	a.playlistInfo.Current = pos
	if err := a.updatePlaylist(); err != nil {
		return err
	}
	_, updating := s["updating_db"]
	return a.jsonCache.SetIfModified(pathAPIMusicLibrary, &httpLibraryInfo{
		Updating: updating,
	})
}

func (a *api) updateOptions(ctx context.Context) error {
	s, err := a.client.ReplayGainStatus(ctx)
	if err != nil {
		return err
	}
	a.mu.Lock()
	a.replayGain = s
	a.mu.Unlock()
	return nil
}
