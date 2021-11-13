package images

import (
	"context"
	"errors"
	"net/http"
	"path"
	"strings"

	"github.com/meiraka/vv/internal/mpd"
)

// Remote provides http album art server from mpd albumart api.
type Remote struct {
	httpPrefix string
	cache      *cache
	client     *mpd.Client
}

// NewRemote initializes Remote with cacheDir.
func NewRemote(httpPrefix string, client *mpd.Client, cacheDir string) (*Remote, error) {
	cache, err := newCache(cacheDir)
	if err != nil {
		return nil, err
	}
	s := &Remote{
		httpPrefix: httpPrefix,
		cache:      cache,
		client:     client,
	}
	return s, nil
}

// Update rescans song images if not indexed.
func (s *Remote) Update(ctx context.Context, song map[string][]string) error {
	if ok, err := s.isSupported(ctx); err != nil {
		return err
	} else if !ok {
		return nil
	}
	key, file, ok := s.key(song)
	if !ok {
		return nil
	}
	if _, ok := s.cache.GetLastRequestID(key); ok {
		return nil
	}
	if err := s.updateCache(ctx, key, file, ""); err != nil {
		// reduce errors for same key
		s.cache.SetEmpty(key, "")
		return err
	}
	return nil
}

// Rescan rescans song images.
func (s *Remote) Rescan(ctx context.Context, song map[string][]string, reqid string) error {
	if ok, err := s.isSupported(ctx); err != nil {
		return err
	} else if !ok {
		return nil
	}

	key, file, ok := s.key(song)
	if !ok {
		return nil
	}
	id, ok := s.cache.GetLastRequestID(key)
	if ok && id == reqid {
		return nil
	}
	if err := s.updateCache(ctx, key, file, reqid); err != nil {
		return err
	}
	return nil
}

// Close finalizes cache db, coroutines.
func (s *Remote) Close() error {
	return s.cache.Close()
}

// ServeHTTP serves local cover art with httpPrefix
func (s *Remote) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// strip httpPrefix
	p := s.httpPrefix
	if p[len(p)-1] != '/' {
		p += "/"
	}
	if !strings.HasPrefix(r.URL.Path, p) {
		http.NotFound(w, r)
		return
	}
	urlName := r.URL.Path[len(p):]

	// get local filename
	path, ok := s.cache.GetLocalPath(urlName)
	if !ok {
		http.NotFound(w, r)
		return
	}
	serveImage(path, w, r)
}

// GetURLs returns cover path for song
func (s *Remote) GetURLs(song map[string][]string) ([]string, bool) {
	if s == nil {
		return nil, true
	}
	key, _, ok := s.key(song)
	if !ok {
		return nil, true
	}
	url, ok := s.cache.GetURL(key)
	if !ok {
		return nil, false
	}
	if len(url) == 0 {
		return nil, true
	}
	return []string{path.Join(s.httpPrefix, url)}, true
}

func (s *Remote) isSupported(ctx context.Context) (bool, error) {
	cmds, err := s.client.Commands(ctx)
	if err != nil {
		return false, err
	}
	for _, cmd := range cmds {
		if cmd == "albumart" {
			return true, nil
		}
	}
	return false, nil
}

// key returns cache key and albumart command file path.
func (s *Remote) key(song map[string][]string) (key, file string, ok bool) {
	f, ok := song["file"]
	if !ok {
		return "", "", false
	}
	if len(f) != 1 {
		return "", "", false
	}
	return path.Dir(f[0]), f[0], true
}

func (s *Remote) updateCache(ctx context.Context, key, file, reqid string) error {
	b, err := s.client.AlbumArt(ctx, file)
	if err != nil {
		// set zero value for not found
		if errors.Is(err, mpd.ErrNoExist) {
			return s.cache.SetEmpty(key, reqid)
		}
		return err
	}
	return s.cache.Set(key, reqid, b)
}
