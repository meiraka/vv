package api

import (
	"context"
	"net/http"
	"time"

	"github.com/meiraka/vv/internal/mpd"
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
	api                          *api
	apiVersion                   http.Handler
	apiMusic                     http.Handler
	apiMusicStats                http.Handler
	apiMusicPlaylist             http.Handler
	apiMusicPlaylistSongs        http.Handler
	apiMusicPlaylistSongsCurrent http.Handler
	apiMusicLibrary              http.Handler
	apiMusicLibrarySongs         http.Handler
	apiMusicOutputs              http.Handler
	apiMusicOutputsStream        http.Handler
	apiMusicImages               http.Handler
	apiMusicStorage              http.Handler
}

// NewHandler creates json api handler.
func NewHandler(ctx context.Context, cl *mpd.Client, w *mpd.Watcher, c *Config) (*Handler, error) {
	a, err := newAPI(ctx, cl, w, c)
	if err != nil {
		return nil, err
	}
	h := &Handler{
		api:                          a,
		apiVersion:                   a.VersionHandler(),
		apiMusic:                     a.StatusHandler(),
		apiMusicStats:                a.StatsHandler(),
		apiMusicPlaylist:             a.PlaylistHandler(),
		apiMusicPlaylistSongs:        a.PlaylistSongsHandler(),
		apiMusicPlaylistSongsCurrent: a.PlaylistSongsCurrentHandler(),
		apiMusicLibrary:              a.LibraryHandler(),
		apiMusicLibrarySongs:         a.LibrarySongsHandler(),
		apiMusicOutputs:              a.OutputsHandler(),
		apiMusicOutputsStream:        a.OutputsStreamHandler(),
		apiMusicImages:               a.ImagesHandler(),
		apiMusicStorage:              a.StorageHandler(),
	}
	a.ClearEvent()

	return h, nil
}

// ServeHTTP implements http.Handler.
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
	default:
		http.NotFound(w, r)
	}
}

// Stop stops internal goroutine and html audio connection.
func (h *Handler) Stop() {
	h.api.Stop()
}

// Shutdown stops background api.
func (h *Handler) Shutdown(ctx context.Context) error {
	return h.api.imgBatch.Shutdown(ctx)
}
