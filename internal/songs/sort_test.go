package songs

import (
	"reflect"
	"testing"
)

func TestWeakFilterSort(t *testing.T) {
	a := map[string][]string{"Artist": {"foo", "bar"}, "Track": {"1"}, "Album": {"baz"}, "Genre": {"qux"}}
	b := map[string][]string{"Artist": {"bar"}, "Track": {"2"}, "Album": {"baz"}}
	c := map[string][]string{"Artist": {"hoge", "fuga"}, "Album": {"piyo"}}
	songs := []map[string][]string{a, b, c}
	testsets := map[string]struct {
		desc    string
		keys    []string
		filters [][2]*string
		must    int
		max     int
		pos     int
		want    []map[string][]string
		wantPos int
	}{
		"": {
			keys: []string{"Album", "Track"},
			max:  100, filters: [][2]*string{},
			pos:     0,
			want:    []map[string][]string{a, b, c},
			wantPos: 0,
		},
		"invalid pos returns -1": {
			keys: []string{"Album", "Track"},
			max:  100, filters: [][2]*string{},
			pos:     -1,
			want:    []map[string][]string{a, b, c},
			wantPos: -1,
		},
		"filter 1st item": {
			keys: []string{"Album", "Track"},
			max:  2, filters: [][2]*string{{strPtr("Album"), strPtr("baz")}, {strPtr("Track"), strPtr("1")}},
			pos:     0,
			want:    []map[string][]string{a, b},
			wantPos: 0,
		},
		"filter 1st item(must)": {
			keys: []string{"Album", "Track"},
			max:  100, filters: [][2]*string{{strPtr("Album"), strPtr("piyo")}, {strPtr("Track"), nil}},
			must:    1,
			pos:     2,
			want:    []map[string][]string{c},
			wantPos: 0,
		},
		"filter 2nd item": {
			keys: []string{"Album", "Track"},
			max:  1, filters: [][2]*string{{strPtr("Album"), strPtr("baz")}, {strPtr("Track"), strPtr("1")}},
			pos:     0,
			want:    []map[string][]string{a},
			wantPos: 0,
		},
		"filter 2nd item(must)": {
			keys: []string{"Album", "Track"},
			max:  100, filters: [][2]*string{{strPtr("Album"), strPtr("baz")}, {strPtr("Track"), strPtr("1")}},
			must:    2,
			pos:     0,
			want:    []map[string][]string{a},
			wantPos: 0,
		},
		"filter by max value": {
			keys: []string{"Album", "Track"},
			max:  1, filters: [][2]*string{{strPtr("Album"), strPtr("baz")}},
			pos:     0,
			want:    []map[string][]string{a},
			wantPos: 0,
		},
		"multi tags": {
			keys: []string{"Artist", "Album"},
			max:  100, filters: [][2]*string{{strPtr("Artist"), strPtr("fuga")}},
			pos:     3,
			want:    []map[string][]string{a, b, a, c, c},
			wantPos: 3,
		},
		"wantPos changed {removed(a), removed(b), removed(a), selected(c), removed(c)}": {
			keys: []string{"Artist", "Album"},
			max:  1, filters: [][2]*string{{strPtr("Artist"), strPtr("fuga")}},
			pos:     3,
			want:    []map[string][]string{c},
			wantPos: 0,
		},
		"selected pos was removed {selected(removed(a)), removed(b), removed(a), c, removed(c)}": {
			keys: []string{"Artist", "Album"},
			max:  1, filters: [][2]*string{{strPtr("Artist"), strPtr("fuga")}},
			pos:     0,
			want:    []map[string][]string{c},
			wantPos: -1,
		},
	}
	for label, tt := range testsets {
		t.Run(label, func(t *testing.T) {
			got, _, pos := WeakFilterSort(songs, tt.keys, tt.filters, tt.must, tt.max, tt.pos)
			if !reflect.DeepEqual(got, tt.want) || pos != tt.wantPos {
				t.Errorf("got WeakFilterSort(%v, %v, %v, %v, %v) =\n%v, _, %v; want\n%v, _, %v", songs, tt.keys, tt.filters, tt.max, tt.pos, got, pos, tt.want, tt.wantPos)
			}
		})
	}
}
