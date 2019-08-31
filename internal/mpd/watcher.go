package mpd

import (
	"context"
	"strings"
)

// NewWatcher connects to mpd server
func (d Dialer) NewWatcher(proto, addr, password string, subsystems ...string) (*Watcher, error) {
	cmd := make([]interface{}, len(subsystems)+1)
	cmd[0] = "idle"
	for i := range subsystems {
		cmd[i+1] = subsystems[i]
	}
	pool, err := newPool(proto, addr, password, d.Timeout, d.ReconnectionInterval)
	if err != nil {
		return nil, err
	}
	c := make(chan string, 10)
	ctx, cancel := context.WithCancel(context.Background())
	closed := make(chan struct{})
	w := &Watcher{
		C:      c,
		closed: closed,
		pool:   pool,
		cancel: cancel,
	}
	go func() {
		defer close(closed)
		var err error
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			// TODO: logging
			err = w.pool.Exec(context.Background(), func(conn *conn) error {
				if err != nil {
					select {
					case c <- "reconnect":
					default:
					}
				}
				if _, err := conn.Writeln(cmd...); err != nil {
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
						_, _ = conn.Writeln("noidle")
						return
					case <-readCtx.Done():
						return
					}

				}()
				for {
					line, err := conn.Readln()
					writeCancel()
					if err != nil {
						return err
					}
					if strings.HasPrefix(line, "changed: ") {
						select {
						case c <- strings.TrimPrefix(line, "changed: "):
						default:
						}
					} else if line != "OK" {
						return newCommandError(line[0 : len(line)-1])
					} else {
						return nil
					}
				}
			})
		}

	}()
	return w, nil
}

// Watcher is the mpd idle command wather
type Watcher struct {
	pool   *pool
	closed <-chan struct{}
	C      <-chan string
	cancel func()
}

// Close closes connection
func (w *Watcher) Close(ctx context.Context) error {
	w.cancel()
	select {
	case <-w.closed:
	case <-ctx.Done():
		return ctx.Err()
	}
	return w.pool.Close(ctx)
}
