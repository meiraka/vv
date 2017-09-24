package main

import (
	"encoding/json"
	"github.com/meiraka/gompd/mpd"
	"reflect"
	"testing"
)

func TestSong(t *testing.T) {
	cache := map[string]string{}
	dir := "./"
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

func TestSongSortKey(t *testing.T) {
	song := Song{"Artist": {"foo", "bar"}, "Album": {"baz"}, "Genre": {"Jazz", "Rock"}}
	testsets := []struct {
		input  []string
		expect string
	}{
		{input: []string{"Album"}, expect: "baz"},
		{input: []string{"Not Found"}, expect: " "},
		{input: []string{"Artist", "Album"}, expect: "foo,barbaz"},
		{input: []string{"Artist", "Album", "Genre"}, expect: "foo,barbazJazz,Rock"},
	}
	for _, tt := range testsets {
		actual := song.SortKey(tt.input)
		if !reflect.DeepEqual(tt.expect, actual) {
			t.Errorf("unexpected return for %s. expect %s, actual %s", tt.input, tt.expect, actual)
		}
	}
}

func TestSongSortKeys(t *testing.T) {
	song := Song{"Artist": {"foo", "bar"}, "Album": {"baz"}, "Genre": {"Jazz", "Rock"}}
	testsets := []struct {
		input  []string
		expect []string
	}{
		{input: []string{"Album"}, expect: []string{"baz"}},
		{input: []string{"Not Found"}, expect: []string{" "}},
		{input: []string{"Artist", "Album"}, expect: []string{"foobaz", "barbaz"}},
		{input: []string{"Artist", "Album", "Genre"}, expect: []string{"foobazJazz", "foobazRock", "barbazJazz", "barbazRock"}},
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

func TestSongSortSongs(t *testing.T) {
	a := Song{"Artist": {"foo"}, "Track": {"1"}, "Album": {"baz"}}
	b := Song{"Artist": {"bar"}, "Track": {"2"}, "Album": {"baz"}}
	c := Song{"Artist": {"hoge", "fuga"}, "Album": {"piyo"}}
	songs := []Song{a, b, c}
	testsets := []struct {
		input  []string
		expect []Song
	}{
		{input: []string{"Album", "Track"}, expect: []Song{a, b, c}},
		{input: []string{"Artist"}, expect: []Song{b, a, c, c}},
	}
	for _, tt := range testsets {
		actual := SortSongs(songs, tt.input)
		if !reflect.DeepEqual(tt.expect, actual) {
			t.Errorf("unexpected return for %s. expect %s, actual %s", tt.input, tt.expect, actual)
		}
	}
}

func TestSongSortSongsUniq(t *testing.T) {
	a := Song{"Artist": {"foo"}, "Track": {"1"}, "Album": {"baz"}}
	b := Song{"Artist": {"bar"}, "Track": {"2"}, "Album": {"baz"}}
	c := Song{"Artist": {"hoge", "fuga"}, "Album": {"piyo"}}
	songs := []Song{a, b, c}
	testsets := []struct {
		input  []string
		expect []Song
	}{
		{input: []string{"Album", "Track"}, expect: []Song{a, b, c}},
		{input: []string{"Artist"}, expect: []Song{b, a, c}},
	}
	for _, tt := range testsets {
		actual := SortSongsUniq(songs, tt.input)
		if !reflect.DeepEqual(tt.expect, actual) {
			t.Errorf("unexpected return for %s. expect %s, actual %s", tt.input, tt.expect, actual)
		}
	}
}

func TestSongWeakFilterSongs(t *testing.T) {
	a := Song{"Artist": {"foo"}, "Track": {"1"}, "Album": {"baz"}}
	b := Song{"Artist": {"bar"}, "Track": {"2"}, "Album": {"baz"}}
	c := Song{"Artist": {"hoge", "fuga"}, "Album": {"piyo"}, "Title": {"hogehoge"}}
	songs := []Song{a, a, a, b, b, b, c, c, c}
	testsets := []struct {
		input  [][]string
		max    int
		expect []Song
	}{
		{input: [][]string{{"Album", "baz"}, {"Artist", "foo"}}, max: 6, expect: []Song{a, a, a, b, b, b}},
		{input: [][]string{{"Album", "baz"}, {"Artist", "foo"}}, max: 3, expect: []Song{a, a, a}},
		{input: [][]string{{"Album", "baz"}, {"Artist", "foo"}}, max: 1, expect: []Song{a}},
		{input: [][]string{{"Album", "baz"}, {"Artist", "foo"}}, max: 9999, expect: []Song{a, a, a, b, b, b, c, c, c}},
		{input: [][]string{{"Artist", "fuga"}}, max: 3, expect: []Song{c, c, c}},
		{input: [][]string{{"Title", "hogehoge"}}, max: 3, expect: []Song{c, c, c}},
	}
	for _, tt := range testsets {
		actual := WeakFilterSongs(songs, tt.input, tt.max)
		if !reflect.DeepEqual(tt.expect, actual) {
			t.Errorf("unexpected return for %s, %d. expect %s, actual %s", tt.input, tt.max, tt.expect, actual)
		}
	}
}
