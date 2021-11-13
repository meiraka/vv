package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/meiraka/vv/internal/mpd"
	"github.com/meiraka/vv/internal/songs"
)

const (
	pathAPIMusicStatus               = "/api/music"
	pathAPIMusicImages               = "/api/music/images"
	pathAPIMusicLibrary              = "/api/music/library"
	pathAPIMusicLibrarySongs         = "/api/music/library/songs"
	pathAPIMusicOutputs              = "/api/music/outputs"
	pathAPIMusicOutputsStream        = "/api/music/outputs/stream"
	pathAPIMusicPlaylist             = "/api/music/playlist"
	pathAPIMusicPlaylistSongs        = "/api/music/playlist/songs"
	pathAPIMusicPlaylistSongsCurrent = "/api/music/playlist/songs/current"
	pathAPIMusicStats                = "/api/music/stats"
	pathAPIMusicStorage              = "/api/music/storage"
	pathAPIMusicStorageNeighbors     = "/api/music/storage/neighbors"
	pathAPIVersion                   = "/api/version"
)

// Config is options for api Handler.
type Config struct {
	AppVersion        string            // app version string for info
	BackgroundTimeout time.Duration     // timeout for background mpd cache updating jobs
	AudioProxy        map[string]string // audio device - mpd http server addr pair to proxy
	skipInit          bool              // do not initialize mpd cache(for test)
	ImageProviders    []ImageProvider
}

// Handler implements http.Handler for vv json api.
type Handler struct {
	apiMusic                     *StatusHandler
	apiMusicImages               *ImagesHandler
	apiMusicLibrary              *LibraryHandler
	apiMusicLibrarySongs         *LibrarySongsHandler
	apiMusicOutputs              *OutputsHandler
	apiMusicOutputsStream        *OutputsStreamHandler
	apiMusicPlaylist             *PlaylistHandler
	apiMusicPlaylistSongs        *PlaylistSongsHandler
	apiMusicPlaylistSongsCurrent *CurrentSongHandler
	apiMusicStats                *StatsHandler
	apiMusicStorage              *StorageHandler
	apiMusicStorageNeighbors     *NeighborsHandler
	apiVersion                   *VersionHandler
	songHooks                    []func(s map[string][]string) map[string][]string
	songsHooks                   []func(s []map[string][]string) []map[string][]string
	closable                     []interface{ Close() }
	stoppable                    []interface{ Stop() }
	shutdownable                 []interface{ Shutdown(context.Context) error }
}

// NewHandler creates Handler and initialize mpd cache data.
func NewHandler(ctx context.Context, cl *mpd.Client, w *mpd.Watcher, c *Config) (*Handler, error) {
	if c == nil {
		c = &Config{}
	}
	if c.BackgroundTimeout == 0 {
		c.BackgroundTimeout = 30 * time.Second
	}
	h := &Handler{}
	var err error
	if h.apiMusic, err = NewStatusHandler(cl); err != nil {
		return nil, err
	}
	h.closable = append(h.closable, h.apiMusic)

	if h.apiMusicImages, err = NewImagesHandler(c.ImageProviders); err != nil {
		return nil, err
	}
	h.songHooks = append(h.songHooks, func(s map[string][]string) map[string][]string { s, _ = h.apiMusicImages.ConvSong(s); return s })
	h.songsHooks = append(h.songsHooks, h.apiMusicImages.ConvSongs)
	h.closable = append(h.closable, h.apiMusicImages)
	h.shutdownable = append(h.shutdownable, h.apiMusicImages)

	if h.apiMusicLibrary, err = NewLibraryHandler(cl); err != nil {
		return nil, err
	}
	h.closable = append(h.closable, h.apiMusicLibrary)

	if h.apiMusicLibrarySongs, err = NewLibrarySongsHandler(cl, h.songsHook); err != nil {
		return nil, err
	}
	h.closable = append(h.closable, h.apiMusicLibrarySongs)

	if h.apiMusicOutputs, err = NewOutputsHandler(cl, c.AudioProxy); err != nil {
		return nil, err
	}
	h.closable = append(h.closable, h.apiMusicOutputs)

	if h.apiMusicOutputsStream, err = NewOutputsStreamHandler(c.AudioProxy); err != nil {
		return nil, err
	}
	h.stoppable = append(h.stoppable, h.apiMusicOutputsStream)

	if h.apiMusicPlaylist, err = NewPlaylistHandler(cl, c); err != nil {
		return nil, err
	}
	h.closable = append(h.closable, h.apiMusicPlaylist)
	h.shutdownable = append(h.shutdownable, h.apiMusicPlaylist)

	if h.apiMusicPlaylistSongs, err = NewPlaylistSongsHandler(cl, h.songsHook); err != nil {
		return nil, err
	}
	h.closable = append(h.closable, h.apiMusicPlaylistSongs)

	if h.apiMusicPlaylistSongsCurrent, err = NewCurrentSongHandler(cl, h.songHook); err != nil {
		return nil, err
	}
	h.closable = append(h.closable, h.apiMusicPlaylistSongsCurrent)

	if h.apiMusicStats, err = NewStatsHandler(cl); err != nil {
		return nil, err
	}
	h.closable = append(h.closable, h.apiMusicStats)

	if h.apiMusicStorage, err = NewStorageHandler(cl); err != nil {
		return nil, err
	}
	h.closable = append(h.closable, h.apiMusicStorage)
	if h.apiMusicStorageNeighbors, err = NewNeighborsHandler(cl); err != nil {
		return nil, err
	}
	h.closable = append(h.closable, h.apiMusicStorageNeighbors)

	if h.apiVersion, err = NewVersionHandler(cl, c.AppVersion); err != nil {
		return nil, err
	}
	h.closable = append(h.closable, h.apiVersion)
	if err := h.apiVersion.Update(); err != nil {
		return nil, err
	}
	// remove changed event for test stability
	clearChan(h.apiVersion.Changed())
	if err := h.hookEvent(ctx, w, c); err != nil {
		return nil, err
	}
	return h, nil
}

// ServeHTTP serves vv json api.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case pathAPIVersion:
		h.apiVersion.ServeHTTP(w, r)
	case pathAPIMusicStatus:
		h.apiMusic.ServeHTTP(w, r)
	case pathAPIMusicStats:
		h.apiMusicStats.ServeHTTP(w, r)
	case pathAPIMusicPlaylist:
		h.apiMusicPlaylist.ServeHTTP(w, r)
	case pathAPIMusicPlaylistSongs:
		h.apiMusicPlaylistSongs.ServeHTTP(w, r)
	case pathAPIMusicPlaylistSongsCurrent:
		h.apiMusicPlaylistSongsCurrent.ServeHTTP(w, r)
	case pathAPIMusicLibrary:
		h.apiMusicLibrary.ServeHTTP(w, r)
	case pathAPIMusicLibrarySongs:
		h.apiMusicLibrarySongs.ServeHTTP(w, r)
	case pathAPIMusicOutputs:
		h.apiMusicOutputs.ServeHTTP(w, r)
	case pathAPIMusicOutputsStream:
		h.apiMusicOutputsStream.ServeHTTP(w, r)
	case pathAPIMusicImages:
		h.apiMusicImages.ServeHTTP(w, r)
	case pathAPIMusicStorage:
		h.apiMusicStorage.ServeHTTP(w, r)
	case pathAPIMusicStorageNeighbors:
		h.apiMusicStorageNeighbors.ServeHTTP(w, r)
	default:
		http.NotFound(w, r)
	}
}

// Stop stops handlers which cannot stop by (*http.Server) Shutdown.
func (h *Handler) Stop() {
	for i := range h.stoppable {
		h.stoppable[i].Stop()
	}
}

// Shutdown stops background api.
func (h *Handler) Shutdown(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	errs := make(chan error, len(h.shutdownable))
	var wg sync.WaitGroup
	for i := range h.shutdownable {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			if err := h.shutdownable[i].Shutdown(ctx); err != nil {
				errs <- err
				cancel()
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	return <-errs
}

func clearChan(c <-chan struct{}) {
	for {
		select {
		case <-c:
		default:
			return
		}
	}
}

func (h *Handler) hookEvent(ctx context.Context, w *mpd.Watcher, c *Config) error {
	go func() {
		for range h.apiMusic.Changed() {
			h.apiMusic.BroadCast(pathAPIMusicStatus)
			h.apiMusicLibrary.UpdateStatus(h.apiMusic.Cache().Updating)
			if pos := h.apiMusic.Cache().Song; pos != nil {
				h.apiMusicPlaylist.UpdateCurrent(*pos)
			}
		}
	}()
	go func() {
		for updating := range h.apiMusicImages.Changed() {
			h.apiMusic.BroadCast(pathAPIMusicImages)
			if !updating {
				ctx, cancel := context.WithTimeout(context.Background(), c.BackgroundTimeout)
				h.apiMusicPlaylistSongsCurrent.Update(ctx)
				h.apiMusicLibrarySongs.Update(ctx)
				cancel()
			}
		}
	}()
	go func() {
		for range h.apiMusicLibrary.Changed() {
			h.apiMusic.BroadCast(pathAPIMusicLibrary)
		}
	}()
	go func() {
		for range h.apiMusicLibrarySongs.Changed() {
			h.apiMusic.BroadCast(pathAPIMusicLibrarySongs)
			h.apiMusicPlaylist.UpdateLibrarySongs(h.apiMusicLibrarySongs.Cache())
		}
	}()
	go func() {
		for range h.apiMusicOutputs.Changed() {
			h.apiMusic.BroadCast(pathAPIMusicOutputs)
		}
	}()
	go func() {
		for range h.apiMusicPlaylist.Changed() {
			h.apiMusic.BroadCast(pathAPIMusicPlaylist)
		}
	}()
	go func() {
		for range h.apiMusicPlaylistSongs.Changed() {
			h.apiMusic.BroadCast(pathAPIMusicPlaylistSongs)
			h.apiMusicPlaylist.UpdatePlaylistSongs(h.apiMusicPlaylistSongs.Cache())
		}
	}()
	go func() {
		for range h.apiMusicPlaylistSongsCurrent.Changed() {
			h.apiMusic.BroadCast(pathAPIMusicPlaylistSongsCurrent)
		}
	}()
	go func() {
		for range h.apiMusicStats.Changed() {
			h.apiMusic.BroadCast(pathAPIMusicStats)
		}
	}()
	go func() {
		for range h.apiMusicStorage.Changed() {
			h.apiMusic.BroadCast(pathAPIMusicStorage)
		}
	}()
	go func() {
		for range h.apiMusicStorageNeighbors.Changed() {
			h.apiMusic.BroadCast(pathAPIMusicStorageNeighbors)
		}
	}()
	go func() {
		for range h.apiVersion.Changed() {
			h.apiMusic.BroadCast(pathAPIVersion)
		}
	}()

	all := []func(context.Context) error{
		h.apiMusicLibrarySongs.Update,
		h.apiMusicPlaylistSongs.Update,
		h.apiMusic.UpdateOptions,
		h.apiMusicPlaylistSongsCurrent.Update,
		h.apiMusicOutputs.Update,
		h.apiMusicStats.Update,
		h.apiMusicStorage.Update,
		h.apiMusicStorageNeighbors.Update,
	}
	go func() {
		for e := range w.Event() {
			ctx, cancel := context.WithTimeout(context.Background(), c.BackgroundTimeout)
			switch e {
			case "reconnecting":
				h.apiVersion.UpdateNoMPD()
			case "reconnect":
				h.apiVersion.Update()
				for _, v := range all {
					v(ctx)
				}
			case "database":
				h.apiMusicLibrarySongs.Update(ctx)
				h.apiMusic.Update(ctx)
				// h.apiMusicPlaylistSongsCurrent.Update(ctx) // "currentsong" metadata did not updated until song changes
				// h.apiMusicPlaylistSongs.Update(ctx) // client does not use this api
				h.apiMusicStats.Update(ctx)
			case "playlist":
				h.apiMusicPlaylistSongs.Update(ctx)
			case "player":
				h.apiMusic.Update(ctx)
				h.apiMusicPlaylistSongsCurrent.Update(ctx)
				h.apiMusicStats.Update(ctx)
			case "mixer":
				h.apiMusic.Update(ctx)
			case "options":
				h.apiMusic.UpdateOptions(ctx)
			case "update":
				h.apiMusic.Update(ctx)
			case "output":
				h.apiMusicOutputs.Update(ctx)
			case "mount":
				h.apiMusicStorage.Update(ctx)
			case "neighbor":
				h.apiMusicStorageNeighbors.Update(ctx)
			default:
			}
			cancel()
		}
		for i := range h.closable {
			h.closable[i].Close()
		}
	}()
	if c.skipInit {
		return nil
	}
	for i := range all {
		if err := all[i](ctx); err != nil {
			return err
		}
	}
	// update handler cache before return.
	// for test stability only
	if pos := h.apiMusic.Cache().Song; pos != nil {
		h.apiMusicPlaylist.UpdateCurrent(*pos)
		clearChan(h.apiMusicPlaylist.Changed())
	}
	return nil
}

func (h *Handler) songHook(s map[string][]string) map[string][]string {
	s = songs.AddTags(s)
	for i := range h.songHooks {
		s = h.songHooks[i](s)
	}
	return s
}

func (h *Handler) songsHook(s []map[string][]string) []map[string][]string {
	n := make([]map[string][]string, len(s))
	for i := range s {
		n[i] = songs.AddTags(s[i])
	}
	for i := range h.songsHooks {
		n = h.songsHooks[i](n)
	}
	return n
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

// btoa convert bool to string.
func btoa(b bool, t, f string) string {
	if b {
		return t
	}
	return f
}
