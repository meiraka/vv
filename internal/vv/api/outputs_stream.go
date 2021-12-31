package api

import (
	"context"
	"io"
	"net/http"
	"sync"
)

// OutputsStreamHandler is a MPD HTTP audio proxy.
type OutputsStreamHandler struct {
	proxy  map[string]string
	stopCh chan struct{}
	stopMu sync.Mutex
	stopB  bool
	logger Logger
}

// NewOutputsStreamHandler initilize OutputsStreamHandler cache with mpd connection.
func NewOutputsStreamHandler(proxy map[string]string, logger Logger) (*OutputsStreamHandler, error) {
	return &OutputsStreamHandler{
		proxy:  proxy,
		stopCh: make(chan struct{}),
		logger: logger,
	}, nil
}

// ServeHTTP responses audio stream.
func (a *OutputsStreamHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	dev := r.URL.Query().Get("name")
	url, ok := a.proxy[dev]
	if !ok {
		http.NotFound(w, r)
		return
	}
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	pr, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		a.logger.Println("vv/api: stream:", url, err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	resp, err := http.DefaultClient.Do(pr)
	if err != nil {
		a.logger.Println("vv/api: stream:", url, err)
		w.WriteHeader(http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	for k, v := range resp.Header {
		for i := range v {
			w.Header().Add(k, v[i])
		}
	}
	go func() {
		select {
		case <-ctx.Done():
		case <-a.stopCh:
			// disconnect audio stream by stop()
			cancel()
		}
	}()
	io.Copy(w, resp.Body)
}

// Stop closes audio streams.
func (a *OutputsStreamHandler) Stop() {
	a.stopMu.Lock()
	if !a.stopB {
		a.stopB = true
		close(a.stopCh)
	}
	a.stopMu.Unlock()
}
