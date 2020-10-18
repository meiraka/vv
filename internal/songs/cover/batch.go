package cover

import (
	"context"
	"errors"
	"log"
	"path"
	"sync"

	"github.com/meiraka/vv/internal/songs"
)

// Cover represents cover api.
type Cover interface {
	Rescan(context.Context, map[string][]string) error
	GetURLs(map[string][]string) ([]string, bool)
}

var (
	// ErrAlreadyShutdown returns if already Shutdown is called
	ErrAlreadyShutdown = errors.New("cover: already shutdown")
	// ErrAlreadyUpdating returns if already Update is called
	ErrAlreadyUpdating = errors.New("cover: update already started")
)

// Batch provides background updater for cover api.
type Batch struct {
	covers []Cover
	sem    chan struct{}
	e      chan bool

	shutdownMu sync.Mutex
	shutdownCh chan struct{}
	shutdownB  bool
}

// NewBatch creates Batch from some cover api.
func NewBatch(covers []Cover) *Batch {
	ret := &Batch{
		covers:     covers,
		sem:        make(chan struct{}, 1),
		e:          make(chan bool, 1),
		shutdownCh: make(chan struct{}),
	}
	ret.sem <- struct{}{}
	return ret
}

// Event returns event chan which returns bool updating or not.
func (b *Batch) Event() <-chan bool {
	return b.e
}

// GetURLs returns images url list.
func (b *Batch) GetURLs(song map[string][]string) (urls []string, updated bool) {
	allUpdated := true
	for _, cover := range b.covers {
		urls, updated = cover.GetURLs(song)
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
func (b *Batch) Update(songs []map[string][]string) error {
	select {
	case _, ok := <-b.sem:
		if !ok {
			return ErrAlreadyShutdown
		}
	default:
		return ErrAlreadyUpdating
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
			for _, c := range b.covers {
				if err := c.Rescan(ctx, song); err != nil {
					log.Printf("cover: %v: %v", songsTag(song, "file"), err)
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

// Shutdown  gracefully shuts down cover updater.
func (b *Batch) Shutdown(ctx context.Context) error {
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
