package api_test

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/meiraka/vv/internal/vv/api"
)

func TestOutputsStreamHandlerGET(t *testing.T) {
	normal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}))
	defer normal.Close()
	slowconn := make(chan struct{}, 1)
	slow := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		slowconn <- struct{}{}
	}))
	defer slow.Close()
	h, err := api.NewOutputsStreamHandler(map[string]string{
		"normal": normal.URL,
		"slow":   slow.URL,
	})
	if err != nil {
		t.Fatalf("failed to init OutputsStreamHandler: %v", err)
	}
	for _, tt := range []struct {
		label    string
		url      string
		postHook func()
		status   int
		want     string
	}{
		{
			label:  "ok",
			url:    "/?name=normal",
			status: http.StatusOK,
			want:   "",
		},
		{
			label:  "not found",
			url:    "/?name=notfound",
			status: http.StatusNotFound,
			want:   "404 page not found\n",
		},
		{
			label:  "stop",
			url:    "/?name=slow",
			status: http.StatusOK,
			want:   "",
			postHook: func() {
				<-slowconn
				h.Stop()
			},
		},
	} {
		t.Run(tt.label, func(t *testing.T) {
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				r := httptest.NewRequest(http.MethodGet, tt.url, nil)
				w := httptest.NewRecorder()
				h.ServeHTTP(w, r)
				if status, got := w.Result().StatusCode, w.Body.String(); status != tt.status || got != tt.want {
					t.Errorf("ServeHTTP got\n%d %s; want\n%d %s", status, got, tt.status, tt.want)
				}
			}()
			if tt.postHook != nil {
				tt.postHook()
			}
			wg.Wait()

		})
	}

}
