package songs

import (
	"reflect"
	"testing"
)

func TestAddTags(t *testing.T) {
	for _, tt := range []struct {
		in   map[string][]string
		want map[string][]string
	}{
		{
			in:   map[string][]string{"file": {"hoge"}},
			want: map[string][]string{"file": {"hoge"}, "TrackNumber": {"0000"}, "DiscNumber": {"0001"}, "Length": {"00:00"}},
		},
		{
			in:   map[string][]string{"file": {"appendix/hoge"}, "Track": {"1"}, "Disc": {"2"}, "Time": {"121"}, "Last-Modified": {"2008-09-28T20:04:57Z"}},
			want: map[string][]string{"file": {"appendix/hoge"}, "Track": {"1"}, "Disc": {"2"}, "Time": {"121"}, "Last-Modified": {"2008-09-28T20:04:57Z"}, "TrackNumber": {"0001"}, "DiscNumber": {"0002"}, "Length": {"02:01"}, "LastModifiedDate": {"2008.09.28"}},
		},
	} {
		if got := AddTags(tt.in); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("got AddTags(%v) = %v; want %v", tt.in, got, tt.want)
		}

	}
}

func TestSongTags(t *testing.T) {
	testsets := []struct {
		song  map[string][]string
		input string
		want  []string
	}{
		{song: map[string][]string{"Artist": {"baz"}, "Album": {"foobar"}}, input: "Date", want: nil},
		{song: map[string][]string{"Artist": {"baz"}, "Album": {"foobar"}}, input: "Date-Artist", want: []string{"baz"}},
		{song: map[string][]string{"Artist": {"baz"}, "Album": {"foobar"}}, input: "Artist-Date", want: []string{"baz"}},
		{song: map[string][]string{"Artist": {"baz"}, "Album": {"foobar"}}, input: "Artist-Date-Album", want: []string{"baz-foobar"}},
		{song: map[string][]string{"Artist": {"baz", "qux"}, "Album": {"foo", "bar"}}, input: "Artist-Album", want: []string{"baz-foo", "baz-bar", "qux-foo", "qux-bar"}},
	}
	for _, tt := range testsets {
		if got := Tags(tt.song, tt.input); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("got Tags(%v, %v) = %v; want %v", tt.song, tt.input, got, tt.want)
		}
	}
}

func TestSongTag(t *testing.T) {
	testsets := []struct {
		song  map[string][]string
		input string
		want  []string
	}{
		{song: map[string][]string{"Album": {"foobar"}}, input: "Artist", want: nil},
		{song: map[string][]string{"Album": {"foobar"}}, input: "ArtistSort", want: nil},
		{song: map[string][]string{"Album": {"foobar"}}, input: "AlbumArtist", want: nil},
		{song: map[string][]string{"Album": {"foobar"}}, input: "AlbumArtistSort", want: nil},
		{song: map[string][]string{"Artist": {"foobar"}}, input: "AlbumArtistSort", want: []string{"foobar"}},
		{song: map[string][]string{"Album": {"foobar"}}, input: "AlbumSort", want: []string{"foobar"}},
		{song: map[string][]string{"Album": {"foobar"}}, input: "Album", want: []string{"foobar"}},
	}
	for _, tt := range testsets {
		if got := Tag(tt.song, tt.input); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("got Tag(%v, %v) = %v; want %v", tt.song, tt.input, got, tt.want)
		}
	}
}
