package cover

import (
	"fmt"
	"reflect"
	"testing"
)

func TestLocalCover(t *testing.T) {
	cover, err := NewLocalSearcher("/foo", "./../../../", []string{"app.png"})
	if err != nil {
		t.Fatalf("failed to initialize LocalSearcher: %v", err)
	}
	for _, tt := range []struct {
		in   map[string][]string
		want map[string][]string
	}{
		{in: map[string][]string{"file": {"assets/test.flac"}},
			want: map[string][]string{"file": {"assets/test.flac"}, "cover": {"/foo/assets/app.png"}}},
		{in: map[string][]string{"file": {"assets/app.js"}},
			want: map[string][]string{"file": {"assets/app.js"}, "cover": {"/foo/assets/app.png"}}},
		{in: map[string][]string{"file": {"assets/app.html"}},
			want: map[string][]string{"file": {"assets/app.html"}, "cover": {"/foo/assets/app.png"}}},
		{in: map[string][]string{"file": {"appendix/example.config.yaml"}},
			want: map[string][]string{"file": {"appendix/example.config.yaml"}, "cover": {}}},
	} {
		t.Run(fmt.Sprint(tt.in), func(t *testing.T) {
			if got := cover.AddTags(tt.in); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got AddTags=%v; want %v", got, tt.want)
			}
		})
	}
}
