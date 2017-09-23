package main

import (
	"github.com/meiraka/gompd/mpd"
	"reflect"
	"testing"
)

func TestSong(t *testing.T) {
	cache := map[string]string{}
	dir := "./"
	glob := "main.*"

	input := []mpd.Tags{
		mpd.Tags{"file": []string{"hoge"}},
		mpd.Tags{"file": []string{"appendix/hoge"}},
		mpd.Tags{"file": []string{"appendix/hoge"}, "Track": []string{"1"}, "Disc": []string{"2"}, "Time": []string{"121"}},
	}
	expect := []Song{
		Song{"file": []string{"hoge"}, "cover": []string{"main.go"}, "TrackNumber": []string{"0000"}, "DiscNumber": []string{"0001"}, "Length": []string{"00:00"}},
		Song{"file": []string{"appendix/hoge"}, "TrackNumber": []string{"0000"}, "DiscNumber": []string{"0001"}, "Length": []string{"00:00"}},
		Song{"file": []string{"appendix/hoge"}, "Track": []string{"1"}, "Disc": []string{"2"}, "Time": []string{"121"}, "TrackNumber": []string{"0001"}, "DiscNumber": []string{"0002"}, "Length": []string{"02:01"}},
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
	song := Song{"Artist": []string{"foobar"}}
	testsets := []struct {
		input  []string
		expect []string
	}{
		{input: []string{"ArtistSort"}, expect: nil},
		{input: []string{"ArtistSort", "Artist"}, expect: []string{"foobar"}},
		{input: []string{"Artist"}, expect: []string{"foobar"}},
	}
	for _, tt := range testsets {
		actual := song.Tag(tt.input)
		if !reflect.DeepEqual(tt.expect, actual) {
			t.Errorf("unexpected return for %s. expect %s, actual %s", tt.input, tt.expect, actual)
		}
	}
}

func TestSongSortKey(t *testing.T) {
	song := Song{"Artist": []string{"foo", "bar"}, "Album": []string{"baz"}, "Genre": []string{"Jazz", "Rock"}}
	testsets := []struct {
		input  []string
		expect string
	}{
		{input: []string{"Album"}, expect: "baz"},
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
	song := Song{"Artist": []string{"foo", "bar"}, "Album": []string{"baz"}, "Genre": []string{"Jazz", "Rock"}}
	testsets := []struct {
		input  []string
		expect []string
	}{
		{input: []string{"Album"}, expect: []string{"baz"}},
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

func TestSongSortSongs(t *testing.T) {
	a := Song{"Artist": []string{"foo"}, "Track": []string{"1"}, "Album": {"baz"}}
	b := Song{"Artist": []string{"bar"}, "Track": []string{"2"}, "Album": {"baz"}}
	c := Song{"Artist": []string{"hoge", "fuga"}, "Album": {"piyo"}}
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

func TestSongWeakFilterSongs(t *testing.T) {
	a := Song{"Artist": []string{"foo"}, "Track": []string{"1"}, "Album": {"baz"}}
	b := Song{"Artist": []string{"bar"}, "Track": []string{"2"}, "Album": {"baz"}}
	c := Song{"Artist": []string{"hoge", "fuga"}, "Album": {"piyo"}}
	songs := []Song{a, a, a, b, b, b, c, c, c}
	testsets := []struct {
		input  [][]string
		max    int
		expect []Song
	}{
		{input: [][]string{[]string{"Album", "baz"}, []string{"Artist", "foo"}}, max: 6, expect: []Song{a, a, a, b, b, b}},
		{input: [][]string{[]string{"Album", "baz"}, []string{"Artist", "foo"}}, max: 3, expect: []Song{a, a, a}},
		{input: [][]string{[]string{"Album", "baz"}, []string{"Artist", "foo"}}, max: 1, expect: []Song{a}},
		{input: [][]string{[]string{"Artist", "fuga"}}, max: 3, expect: []Song{c, c, c}},
	}
	for _, tt := range testsets {
		actual := WeakFilterSongs(songs, tt.input, tt.max)
		if !reflect.DeepEqual(tt.expect, actual) {
			t.Errorf("unexpected return for %s, %d. expect %s, actual %s", tt.input, tt.max, tt.expect, actual)
		}
	}
}
