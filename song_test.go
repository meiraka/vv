package main

import (
	"github.com/fhs/gompd/mpd"
	"strings"
	"testing"
)

func TestSongString(t *testing.T) {
	e := "foo: bar, hoge: fuga"
	r := songString(mpd.Attrs{"hoge": "fuga", "foo": "bar"})
	if r != e {
		t.Errorf(
			"unexpected return\nexpected: %s\nactual:   %s",
			e, r,
		)
	}
}

func TestSongAddReadableData(t *testing.T) {
	candidates := []struct {
		input  mpd.Attrs
		expect mpd.Attrs
	}{
		{
			mpd.Attrs{
				"Title": "foo",
				"file":  "path",
			},
			mpd.Attrs{
				"Title":       "foo",
				"file":        "path",
				"DiscNumber":  "0001",
				"TrackNumber": "0000",
				"Length":      "00:00",
			},
		},
		{
			mpd.Attrs{
				"Title": "foo",
				"file":  "path",
				"Disc":  "2",
				"Track": "1",
				"Time":  "121",
			},
			mpd.Attrs{
				"Title":       "foo",
				"file":        "path",
				"Disc":        "2",
				"DiscNumber":  "0002",
				"Track":       "1",
				"TrackNumber": "0001",
				"Time":        "121",
				"Length":      "02:01",
			},
		},
	}

	for _, c := range candidates {
		r := songAddReadableData(c.input)
		if songString(c.expect) != songString(r) {
			t.Errorf(
				"unexpected return\nexpected: %s\nactual:   %s",
				songString(c.expect),
				songString(r),
			)
		}
	}
}

func TestSongSortKey(t *testing.T) {
	candidates := []struct {
		song    mpd.Attrs
		sortkey []string
		expect  string
	}{
		{
			mpd.Attrs{"Title": "foo", "file": "path"},
			[]string{"TrackNumber", "Title", "file"},
			" foopath",
		},
		{
			mpd.Attrs{"Title": "foo", "file": "path"},
			[]string{"foo"},
			" ",
		},
		{
			mpd.Attrs{},
			[]string{"AlbumSort"},
			" ",
		},
		{
			mpd.Attrs{"Artist": "foo"},
			[]string{"ArtistSort"},
			"foo",
		},
		{
			mpd.Attrs{"Artist": "foo"},
			[]string{"AlbumArtist"},
			"foo",
		},
		{
			mpd.Attrs{"Artist": "foo"},
			[]string{"AlbumArtistSort"},
			"foo",
		},
	}
	for _, c := range candidates {
		r := songSortKey(c.song, c.sortkey)
		if r != c.expect {
			t.Errorf(
				"unexpected output for song: %s sortkey: %s\nexpected: \"%s\"\nactual:   \"%s\"",
				songString(c.song),
				strings.Join(c.sortkey, ","),
				c.expect,
				r,
			)
		}
	}
}
