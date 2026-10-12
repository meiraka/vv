package request

import (
	"net/http"
	"testing"
	"time"
)

func TestModifiedSince(t *testing.T) {
	base := time.Date(2024, 1, 15, 12, 0, 0, 0, time.UTC)
	for _, tt := range []struct {
		name   string
		header string
		last   time.Time
		want   bool
	}{
		{
			name:   "no header returns true",
			header: "",
			last:   base,
			want:   true,
		},
		{
			name:   "malformed header returns true",
			header: "not-a-date",
			last:   base,
			want:   true,
		},
		{
			name:   "resource newer than header returns true",
			header: base.Add(-time.Hour).Format(http.TimeFormat),
			last:   base,
			want:   true,
		},
		{
			name:   "resource older than header returns false",
			header: base.Add(time.Hour).Format(http.TimeFormat),
			last:   base,
			want:   false,
		},
		{
			name:   "resource equal to header within one second returns false",
			header: base.Format(http.TimeFormat),
			last:   base,
			want:   false,
		},
		{
			name:   "resource one second after header returns true",
			header: base.Format(http.TimeFormat),
			last:   base.Add(time.Second),
			want:   true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := &http.Request{Header: http.Header{}}
			if tt.header != "" {
				r.Header.Set("If-Modified-Since", tt.header)
			}
			if got := ModifiedSince(r, tt.last); got != tt.want {
				t.Errorf("ModifiedSince = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNoneMatch(t *testing.T) {
	for _, tt := range []struct {
		name   string
		header string
		etag   string
		want   bool
	}{
		{
			name:   "matching etag returns true",
			header: `"abc123"`,
			etag:   `"abc123"`,
			want:   true,
		},
		{
			name:   "non-matching etag returns false",
			header: `"abc123"`,
			etag:   `"def456"`,
			want:   false,
		},
		{
			name:   "empty header returns false",
			header: "",
			etag:   `"abc123"`,
			want:   false,
		},
		{
			name:   "empty etag with empty header returns true",
			header: "",
			etag:   "",
			want:   true,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := &http.Request{Header: http.Header{}}
			if tt.header != "" {
				r.Header.Set("If-None-Match", tt.header)
			}
			if got := NoneMatch(r, tt.etag); got != tt.want {
				t.Errorf("NoneMatch = %v, want %v", got, tt.want)
			}
		})
	}
}
