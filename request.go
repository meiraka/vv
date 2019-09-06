package main

import (
	"context"
	"net/http"
	"time"
)

/*modifiedSince compares If-Modified-Since header given time.Time.*/
func modifiedSince(r *http.Request, l time.Time) bool {
	t, err := time.Parse(http.TimeFormat, r.Header.Get("If-Modified-Since"))
	if err != nil {
		return true
	}
	return !l.Before(t.Add(time.Second))
}

func noneMatch(r *http.Request, etag string) bool {
	return r.Header.Get("If-None-Match") == etag
}

type httpContextKey string

const httpUpdateTime = httpContextKey("updateTime")

func getUpdateTime(r *http.Request) time.Time {
	if v := r.Context().Value(httpUpdateTime); v != nil {
		if i, ok := v.(time.Time); ok {
			return i
		}
	}
	return time.Time{}
}

func setUpdateTime(r *http.Request, u time.Time) *http.Request {
	ctx := context.WithValue(r.Context(), httpUpdateTime, u)
	return r.WithContext(ctx)
}
