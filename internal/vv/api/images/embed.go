package images

import (
	"context"
	"net/http"
	"path"
	"strings"

	"github.com/meiraka/vv/internal/mpd"
	"github.com/meiraka/vv/internal/songs"
)

// Embed provides http album art server from mpd readpicture api.
type Embed struct {
	httpPrefix string
	cache      *cache
	client     *mpd.Client
}

// NewEmbed initializes Embed Cover Art provider with cacheDir.
func NewEmbed(httpPrefix string, client *mpd.Client, cacheDir string) (*Embed, error) {
	cache, err := newCache(cacheDir)
	if err != nil {
		return nil, err
	}
	s := &Embed{
		httpPrefix: httpPrefix,
		cache:      cache,
		client:     client,
	}
	return s, nil
}

// Update rescans song images if not indexed.
func (s *Embed) Update(ctx context.Context, song map[string][]string) error {
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
func (s *Embed) Rescan(ctx context.Context, song map[string][]string, reqid string) error {
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
func (s *Embed) Close() error {
	return s.cache.Close()
}

// ServeHTTP serves local cover art with httpPrefix
func (s *Embed) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
func (s *Embed) GetURLs(song map[string][]string) ([]string, bool) {
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

func (s *Embed) isSupported(ctx context.Context) (bool, error) {
	cmds, err := s.client.Commands(ctx)
	if err != nil {
		return false, err
	}
	for _, cmd := range cmds {
		if cmd == "readpicture" {
			return true, nil
		}
	}
	return false, nil
}

func (s *Embed) key(song map[string][]string) (key string, path string, ok bool) {
	file, ok := song["file"]
	if !ok || len(file) != 1 {
		return "", "", false
	}
	path = file[0]
	key = strings.Join(songs.Tags(song, "AlbumArtist-Album-Date-Label"), ",")
	if len(key) == 0 {
		return "", "", false
	}
	return key, path, true
}

func (s *Embed) updateCache(ctx context.Context, key, file, reqid string) error {
	b, err := s.client.ReadPicture(ctx, file)
	if err != nil {
		return err
	}
	if b == nil {
		return s.cache.SetEmpty(key, reqid)
	}
	return s.cache.Set(key, reqid, b)
}
