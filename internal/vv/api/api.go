package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/meiraka/vv/internal/mpd"
	"github.com/meiraka/vv/internal/songs"
)

// api caches mpd data and generate http.Handler for mpd.
type api struct {
	config    *Config
	client    *mpd.Client
	watcher   *mpd.Watcher
	jsonCache *jsonCache
	upgrader  websocket.Upgrader
	imgBatch  *imgBatch

	neighbors *neighbors

	playlist     []map[string][]string
	library      []map[string][]string
	librarySort  []map[string][]string
	replayGain   map[string]string
	playlistInfo *httpPlaylistInfo

	mu     sync.Mutex
	stopCh chan struct{}
	stopB  bool
}

// newAPI creates api.
func newAPI(ctx context.Context, cl *mpd.Client, w *mpd.Watcher, c *Config) (*api, error) {
	if c.BackgroundTimeout == 0 {
		c.BackgroundTimeout = 30 * time.Second
	}
	cache := newJSONCache()
	a := &api{
		config:    c,
		client:    cl,
		watcher:   w,
		imgBatch:  newImgBatch(c.ImageProviders),
		jsonCache: cache,
		neighbors: newNeighbors(cl, cache),

		playlistInfo: &httpPlaylistInfo{},
		stopCh:       make(chan struct{}),
	}
	if err := a.runCacheUpdater(ctx); err != nil {
		return nil, err
	}
	return a, nil
}

// runCacheUpdater initializes mpd caches and launches mpd/cover image cache updater.
func (a *api) runCacheUpdater(ctx context.Context) error {
	if err := a.updateVersion(); err != nil {
		return err
	}
	all := []func(context.Context) error{a.updateLibrarySongs, a.updatePlaylistSongs, a.updateOptions, a.updateStatus, a.updatePlaylistSongsCurrent, a.updateOutputs, a.updateStats, a.updateStorage, a.neighbors.Update}
	if !a.config.skipInit {
		for _, v := range all {
			if err := v(ctx); err != nil {
				return err
			}
		}
	}
	var wg sync.WaitGroup
	wg.Add(1)
	a.jsonCache.SetIfNone(pathAPIMusicImages, &httpImages{})
	go func() {
		defer wg.Done()
		for updating := range a.imgBatch.Event() {
			a.jsonCache.SetIfModified(pathAPIMusicImages, &httpImages{Updating: updating})
			if !updating {
				ctx, cancel := context.WithTimeout(context.Background(), a.config.BackgroundTimeout)
				a.updatePlaylistSongsCurrent(ctx)
				a.updateLibrarySongs(ctx)
				// h.updatePlaylistSongs(ctx) // client does not use this api
				cancel()
			}
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for e := range a.watcher.Event() {
			ctx, cancel := context.WithTimeout(context.Background(), a.config.BackgroundTimeout)
			switch e {
			case "reconnecting":
				a.updateVersionNoMPD()
			case "reconnect":
				a.updateVersion()
				for _, v := range all {
					v(ctx)
				}
			case "database":
				a.updateLibrarySongs(ctx)
				a.updateStatus(ctx)
				// h.updateCurrentSong(ctx) // "currentsong" metadata did not updated until song changes
				// h.updatePlaylistSongs(ctx) // client does not use this api
				a.updateStats(ctx)
			case "playlist":
				a.updatePlaylistSongs(ctx)
			case "player":
				a.updateStatus(ctx)
				a.updatePlaylistSongsCurrent(ctx)
				a.updateStats(ctx)
			case "mixer":
				a.updateStatus(ctx)
			case "options":
				a.updateOptions(ctx)
				a.updateStatus(ctx)
			case "update":
				a.updateStatus(ctx)
			case "output":
				a.updateOutputs(ctx)
			case "mount":
				a.updateStorage(ctx)
			case "neighbor":
				a.neighbors.Update(ctx)
			}
			cancel()
		}
	}()
	go func() {
		wg.Wait()
		a.jsonCache.Close()
	}()
	return nil
}

// ClearEvent clears current websocket api event list.
func (a *api) ClearEvent() {
LOOP:
	for {
		select {
		case <-a.jsonCache.Event():
		default:
			break LOOP
		}
	}
}

// Stop stops cache updater and html audio stream.
func (a *api) Stop() {
	a.mu.Lock()
	if !a.stopB {
		a.stopB = true
		close(a.stopCh)
	}
	a.mu.Unlock()
}

func (a *api) convSong(s map[string][]string) (map[string][]string, bool) {
	s = songs.AddTags(s)
	delete(s, "cover")
	cover, updated := a.imgBatch.GetURLs(s)
	if len(cover) != 0 {
		s["cover"] = cover
	}
	return s, updated
}

func (a *api) convSongs(s []map[string][]string) []map[string][]string {
	ret := make([]map[string][]string, len(s))
	needUpdates := make([]map[string][]string, 0, len(s))
	for i := range s {
		song, ok := a.convSong(s[i])
		ret[i] = song
		if !ok {
			needUpdates = append(needUpdates, song)
		}
	}
	if len(needUpdates) != 0 {
		a.imgBatch.Update(needUpdates)
	}
	return ret
}

func writeHTTPError(w http.ResponseWriter, status int, err error) {
	w.Header().Add("Content-Type", "application/json; charset=utf-8")
	b, _ := json.Marshal(map[string]string{"error": err.Error()})
	w.Header().Add("Content-Length", strconv.Itoa(len(b)))
	w.WriteHeader(status)
	w.Write(b)
}

func boolPtr(b bool) *bool                { return &b }
func stringPtr(s string) *string          { return &s }
func stringSlicePtr(s []string) *[]string { return &s }
