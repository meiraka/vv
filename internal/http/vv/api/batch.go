package api

import (
	"context"
	"log"
	"path"
	"sync"

	"github.com/meiraka/vv/internal/songs"
)

// ImageProvider represents http cover image image url api.
type ImageProvider interface {
	Rescan(context.Context, map[string][]string) error
	GetURLs(map[string][]string) ([]string, bool)
}

// imgBatch provides background updater for cover image api.
type imgBatch struct {
	apis []ImageProvider
	sem  chan struct{}
	e    chan bool

	shutdownMu sync.Mutex
	shutdownCh chan struct{}
	shutdownB  bool
}

// newImgBatch creates Batch from some cover image api.
func newImgBatch(apis []ImageProvider) *imgBatch {
	ret := &imgBatch{
		apis:       apis,
		sem:        make(chan struct{}, 1),
		e:          make(chan bool, 1),
		shutdownCh: make(chan struct{}),
	}
	ret.sem <- struct{}{}
	return ret
}

// Event returns event chan which returns bool updating or not.
func (b *imgBatch) Event() <-chan bool {
	return b.e
}

// GetURLs returns images url list.
func (b *imgBatch) GetURLs(song map[string][]string) (urls []string, updated bool) {
	allUpdated := true
	for _, api := range b.apis {
		urls, updated = api.GetURLs(song)
		if len(urls) != 0 {
			return
		}
		if !updated {
			allUpdated = false
		}
	}
	return urls, allUpdated
}

var songsTag = songs.Tag

// Update updates image url database.
func (b *imgBatch) Update(songs []map[string][]string) error {
	select {
	case _, ok := <-b.sem:
		if !ok {
			return ErrAlreadyShutdown
		}
	default:
		return errAlreadyUpdating
	}
	go func() {
		select {
		case b.e <- true:
		default:
		}
		defer func() { b.sem <- struct{}{} }()
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go func() {
			select {
			case <-ctx.Done():
			case <-b.shutdownCh:
				cancel()
			}
		}()
		targets := make(map[string]map[string][]string, len(songs))
		for _, song := range songs {
			if len(song["file"]) == 1 {
				targets[path.Dir(song["file"][0])] = song
			}
		}
		for _, song := range targets {
			for _, c := range b.apis {
				if err := c.Rescan(ctx, song); err != nil {
					log.Printf("api: %v: %v", songsTag(song, "file"), err)
					// use previous rescanned result
				}
				urls, _ := c.GetURLs(song)
				if len(urls) > 0 {
					break
				}
			}
		}
		select {
		case <-ctx.Done():
		case b.e <- false:
		default:
		}
	}()
	return nil
}

// Shutdown gracefully shuts down cover image updater.
func (b *imgBatch) Shutdown(ctx context.Context) error {
	b.shutdownMu.Lock()
	if !b.shutdownB {
		close(b.shutdownCh)
		b.shutdownB = true
	}
	b.shutdownMu.Unlock()
	select {
	case _, ok := <-b.sem:
		if ok {
			close(b.sem)
			close(b.e)
		}
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}
