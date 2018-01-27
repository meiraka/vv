package main

import (
	"encoding/json"
	"github.com/meiraka/gompd/mpd"
	"reflect"
	"testing"
)

func TestSong(t *testing.T) {
	cache := map[string]string{}
	dir := "."
	glob := "main.*"

	input := []mpd.Tags{
		mpd.Tags{"file": {"hoge"}},
		mpd.Tags{"file": {"appendix/hoge"}},
		mpd.Tags{"file": {"appendix/hoge"}, "Track": {"1"}, "Disc": {"2"}, "Time": {"121"}},
	}
	expect := []Song{
		Song{"file": {"hoge"}, "cover": {"main.go"}, "TrackNumber": {"0000"}, "DiscNumber": {"0001"}, "Length": {"00:00"}},
		Song{"file": {"appendix/hoge"}, "TrackNumber": {"0000"}, "DiscNumber": {"0001"}, "Length": {"00:00"}},
		Song{"file": {"appendix/hoge"}, "Track": {"1"}, "Disc": {"2"}, "Time": {"121"}, "TrackNumber": {"0001"}, "DiscNumber": {"0002"}, "Length": {"02:01"}},
	}
	actual := MakeSongs(input, dir, glob, cache)
	if len(expect) != len(actual) {
		t.Fatalf("unexpected MakeSongs return length. expect: %d, actual: %d", len(expect), len(actual))
	}
	for i := range expect {
		if !reflect.DeepEqual(expect[i], actual[i]) {
			t.Errorf("unexpected MakeSongs return index %d. expect: %s, actual: %s", i, expect[i], actual[i])
		}
	}
}

func TestSongTag(t *testing.T) {
	testsets := []struct {
		song   Song
		input  string
		expect []string
	}{
		{song: Song{"Album": {"foobar"}}, input: "Artist", expect: nil},
		{song: Song{"Album": {"foobar"}}, input: "ArtistSort", expect: nil},
		{song: Song{"Album": {"foobar"}}, input: "AlbumArtist", expect: nil},
		{song: Song{"Album": {"foobar"}}, input: "AlbumArtistSort", expect: nil},
		{song: Song{"Artist": {"foobar"}}, input: "AlbumArtistSort", expect: []string{"foobar"}},
		{song: Song{"Album": {"foobar"}}, input: "AlbumSort", expect: []string{"foobar"}},
		{song: Song{"Album": {"foobar"}}, input: "Album", expect: []string{"foobar"}},
	}
	for _, tt := range testsets {
		actual := tt.song.Tag(tt.input)
		if !reflect.DeepEqual(tt.expect, actual) {
			t.Errorf("unexpected return for %s. expect %s, actual %s", tt.input, tt.expect, actual)
		}
	}
}

func TestSongSortKeys(t *testing.T) {
	song := Song{"Artist": {"foo", "bar"}, "Album": {"baz"}, "Genre": {"Jazz", "Rock"}}
	testsets := []struct {
		input  []string
		expect []map[string]string
	}{
		{input: []string{"Album"}, expect: []map[string]string{{"Album": "baz", "all": "baz"}}},
		{input: []string{"Not Found"}, expect: []map[string]string{{"Not Found": " ", "all": " "}}},
		{input: []string{"Artist", "Album"}, expect: []map[string]string{{"Artist": "foo", "Album": "baz", "all": "foobaz"}, {"Artist": "bar", "Album": "baz", "all": "barbaz"}}},
		{input: []string{"Artist", "Album", "Genre"}, expect: []map[string]string{
			{"Artist": "foo", "Album": "baz", "Genre": "Jazz", "all": "foobazJazz"},
			{"Artist": "foo", "Album": "baz", "Genre": "Rock", "all": "foobazRock"},
			{"Artist": "bar", "Album": "baz", "Genre": "Jazz", "all": "barbazJazz"},
			{"Artist": "bar", "Album": "baz", "Genre": "Rock", "all": "barbazRock"}}},
	}
	for _, tt := range testsets {
		actual := song.SortKeys(tt.input)
		if !reflect.DeepEqual(tt.expect, actual) {
			t.Errorf("unexpected return for %s. expect %s, actual %s", tt.input, tt.expect, actual)
		}
	}
}

func TestStatus(t *testing.T) {
	candidates := []struct {
		status mpd.Attrs
		expect Status
	}{
		{
			mpd.Attrs{},
			Status{
				-1, false, false, false, false,
				"stopped", 0, 0.0, false,
			},
		},
		{
			mpd.Attrs{
				"volume":      "100",
				"repeat":      "1",
				"random":      "0",
				"single":      "1",
				"consume":     "0",
				"state":       "playing",
				"song":        "1",
				"elapsed":     "10.1",
				"updating_db": "1",
			},
			Status{
				100, true, false, true, false,
				"playing", 1, 10.1, true,
			},
		},
	}
	for _, c := range candidates {
		r := MakeStatus(c.status)
		if !reflect.DeepEqual(c.expect, r) {
			jr, _ := json.Marshal(r)
			je, _ := json.Marshal(c.expect)
			t.Errorf(
				"unexpected. input: %s\nexpected: %s\nactual:   %s",
				c.status,
				je, jr,
			)
		}
	}
}

func TestSortSongs(t *testing.T) {
	a := Song{"Artist": {"foo", "bar"}, "Track": {"1"}, "Album": {"baz"}}
	b := Song{"Artist": {"bar"}, "Track": {"2"}, "Album": {"baz"}}
	c := Song{"Artist": {"hoge", "fuga"}, "Album": {"piyo"}}
	songs := []Song{a, b, c}
	testsets := []struct {
		desc       string
		keys       []string
		filters    [][]string
		max        int
		pos        int
		expectSong []Song
		expectPos  int
	}{
		{
			keys: []string{"Album", "Track"},
			max:  100, filters: [][]string{},
			pos:        0,
			expectSong: []Song{a, b, c},
			expectPos:  0,
		},
		{
			desc: "invalid pos returns -1",
			keys: []string{"Album", "Track"},
			max:  100, filters: [][]string{},
			pos:        -1,
			expectSong: []Song{a, b, c},
			expectPos:  -1,
		},
		{
			desc: "filter 1st item",
			keys: []string{"Album", "Track"},
			max:  2, filters: [][]string{{"Album", "baz"}, {"Track", "1"}},
			pos:        0,
			expectSong: []Song{a, b},
			expectPos:  0,
		},
		{
			desc: "filter 2nd item",
			keys: []string{"Album", "Track"},
			max:  1, filters: [][]string{{"Album", "baz"}, {"Track", "1"}},
			pos:        0,
			expectSong: []Song{a},
			expectPos:  0,
		},
		{
			desc: "filter by max value",
			keys: []string{"Album", "Track"},
			max:  1, filters: [][]string{{"Album", "baz"}},
			pos:        0,
			expectSong: []Song{a},
			expectPos:  0,
		},
		{
			desc: "multi tags",
			keys: []string{"Artist", "Album"},
			max:  100, filters: [][]string{{"Artist", "fuga"}},
			pos:        3,
			expectSong: []Song{a, b, a, c, c},
			expectPos:  3,
		},
		{
			desc: "expectPos changed {removed(a), removed(b), removed(a), selected(c), removed(c)}",
			keys: []string{"Artist", "Album"},
			max:  1, filters: [][]string{{"Artist", "fuga"}},
			pos:        3,
			expectSong: []Song{c},
			expectPos:  0,
		},
		{
			desc: "selected pos was removed {selected(removed(a)), removed(b), removed(a), c, removed(c)}",
			keys: []string{"Artist", "Album"},
			max:  1, filters: [][]string{{"Artist", "fuga"}},
			pos:        0,
			expectSong: []Song{c},
			expectPos:  -1,
		},
	}
	for _, tt := range testsets {
		actualSong, actualPos := SortSongs(songs, tt.keys, tt.filters, tt.max, tt.pos)
		if !reflect.DeepEqual(tt.expectSong, actualSong) || tt.expectPos != actualPos {
			t.Errorf("[%s] unexpected return for SortSongs(%s, %s, %d, %d).\nexpectSong: %s expectPos: %d\nactualSong: %s actualPos: %d", tt.desc, tt.keys, tt.filters, tt.max, tt.pos, tt.expectSong, tt.expectPos, actualSong, actualPos)
		}
	}
}
