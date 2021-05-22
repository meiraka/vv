package api

import (
	"context"
	"net/http"
	"time"
)

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
