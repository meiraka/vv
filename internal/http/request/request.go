package request

import (
	"net/http"
	"time"
)

/*ModifiedSince compares request If-Modified-Since header and l.*/
func ModifiedSince(r *http.Request, l time.Time) bool {
	t, err := time.Parse(http.TimeFormat, r.Header.Get("If-Modified-Since"))
	if err != nil {
		return true
	}
	return !l.Before(t.Add(time.Second))
}

// NoneMatch compares request If-None-Match header and etag
func NoneMatch(r *http.Request, etag string) bool {
	return r.Header.Get("If-None-Match") == etag
}
