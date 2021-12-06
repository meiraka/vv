package mpd

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// NewWatcher connects to mpd server
func NewWatcher(proto, addr string, opts *WatcherOptions) (*Watcher, error) {
	if opts == nil {
		opts = &WatcherOptions{}
	}
	args := make([]interface{}, len(opts.SubSystems))
	for i := range opts.SubSystems {
		args[i] = opts.SubSystems[i]
	}
	pool, err := newPool(proto, addr, opts.Timeout, opts.ReconnectionInterval, opts.connectHook)
	if err != nil {
		return nil, err
	}
	event := make(chan string, 10)
	ctx, cancel := context.WithCancel(context.Background())
	closed := make(chan struct{})
	w := &Watcher{
		event:  event,
		closed: closed,
		pool:   pool,
		cancel: cancel,
	}
	go func() {
		defer close(closed)
		defer close(event)
		var err error
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			// TODO: logging
			if err != nil {
				select {
				case event <- "reconnecting":
				default:
				}
			}
			err = w.pool.Exec(context.Background() /* do not use ctx to graceful shutdown */, func(conn *conn) error {
				if err != nil {
					select {
					case event <- "reconnect":
					default:
					}
				}
				if err := request(conn, "idle", args...); err != nil {
					return err
				}
				readCtx, writeCancel := context.WithCancel(context.Background())
				defer writeCancel()
				go func() {
					select {
					case <-ctx.Done():
						// TODO: logging
						select {
						case <-readCtx.Done():
							return
						default:
						}
						_, _ = fmt.Fprintln(conn, "noidle")
						return
					case <-readCtx.Done():
						return
					}

				}()
				for {
					line, err := readln(conn)
					writeCancel()
					if err != nil {
						return err
					}
					if strings.HasPrefix(line, "changed: ") {
						select {
						case event <- strings.TrimPrefix(line, "changed: "):
						default:
						}
					} else if line != "OK" {
						return parseCommandError(line[0 : len(line)-1])
					} else {
						return nil
					}
				}
			})
		}

	}()
	return w, nil
}

// Watcher is the mpd idle command watcher.
type Watcher struct {
	pool   *pool
	closed <-chan struct{}
	event  chan string
	cancel func()
}

// Event returns event channel which sends idle command outputs.
func (w *Watcher) Event() <-chan string {
	return w.event
}

// Close closes connection
func (w *Watcher) Close(ctx context.Context) error {
	w.cancel()
	err := w.pool.Close(ctx)
	select {
	case <-w.closed:
	case <-ctx.Done():
		return ctx.Err()
	}
	return err
}

// WatcherOptions contains options for mpd idle command connection.
type WatcherOptions struct {
	Password             string
	Timeout              time.Duration
	ReconnectionInterval time.Duration
	// SubSystems are list of recieve events. Watcher recieves all events if SubSystems are empty.
	SubSystems []string
}

func (c *WatcherOptions) connectHook(conn *conn) error {
	if len(c.Password) > 0 {
		if err := execOK(conn, "password", c.Password); err != nil {
			return err
		}
	}
	return nil
}
