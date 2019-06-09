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
	connK := &connKeeper{
		proto:                proto,
		addr:                 addr,
		password:             password,
		ReconnectionTimeout:  d.ReconnectionTimeout,
		ReconnectionInterval: d.ReconnectionInterval,
		connC:                make(chan *conn, 1),
	}
	if err := connK.connectOnce(); err != nil {
		return nil, err
	}
	c := make(chan string, 10)
	ctx, cancel := context.WithCancel(context.Background())
	w := &Watcher{
		C:      c,
		closed: make(chan struct{}),
		conn:   connK,
		cancel: cancel,
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
			}
			_ = w.conn.Exec(ctx, func(conn *conn) error {
				if _, err := conn.Writeln(cmd...); err != nil {
					return err
				}
				pctx, pcancel := context.WithCancel(context.Background())
				defer pcancel()
				go func() {
					select {
					case <-ctx.Done():
						conn.Writeln("noidle")
						return
					case <-pctx.Done():
						return
					}

				}()
				for {
					line, err := conn.Readln()
					pcancel() // cancel noidle
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
					}
					break
				}
				return nil
			})
		}

	}()
	return w, nil
}

// Watcher is the mpd idle command wather
type Watcher struct {
	conn   *connKeeper
	closed chan struct{}
	C      <-chan string
	cancel func()
}

// Close closes connection
func (w *Watcher) Close(ctx context.Context) error {
	w.cancel()
	return w.conn.Close(ctx)

}
