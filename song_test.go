package main

import (
	"github.com/fhs/gompd/mpd"
	"testing"
)

func TestSongAddReadableData(t *testing.T) {
	i := songAddReadableData(mpd.Attrs{"Title": "foo", "file": "path"})
	if i["DiscNumber"] != "0001" {
		t.Errorf("unexpected DiscNumber: '%s'", i["DiscNumber"])
	}
	if i["TrackNumber"] != "0000" {
		t.Errorf("unexpected TrackNumber: '%s'", i["TrackNumber"])
	}
	if i["Length"] != "00:00" {
		t.Errorf("unexpected Length: '%s'", i["Length"])
	}
	f := songAddReadableData(mpd.Attrs{
		"Track": "1",
		"Disc":  "2",
		"Time":  "70"})
	if f["DiscNumber"] != "0002" {
		t.Errorf("unexpected DiscNumber: '%s'", f["DiscNumber"])
	}
	if f["TrackNumber"] != "0001" {
		t.Errorf("unexpected TrackNumber: '%s'", f["TrackNumber"])
	}
	if f["Length"] != "01:10" {
		t.Errorf("unexpected Length: '%s'", f["Length"])
	}
}

func TestSongSortKey(t *testing.T) {
	i := songAddReadableData(mpd.Attrs{"Title": "foo", "file": "path"})
	r := songSortKey(i, []string{"TrackNumber", "Title", "file"})
	if r != "0000foopath" {
		t.Errorf("unexpected output for TrackNumber, Title, file: %s", r)
	}
	r = songSortKey(mpd.Attrs{}, []string{"foo"})
	if r != " " {
		t.Errorf("exptects \" \" if key not found but returns: %s", r)
	}
	r = songSortKey(mpd.Attrs{}, []string{"AlbumSort"})
	if r != " " {
		t.Errorf("expects \" \" if key(AlbumSort and Album for AlbumSort) "+
			"but returns: %s", r)
	}
	r = songSortKey(mpd.Attrs{"Artist": "foo"}, []string{"ArtistSort"})
	if r != "foo" {
		t.Errorf("ArtistSort searches ArtistSort and Artist but returns: %s", r)
	}

	r = songSortKey(mpd.Attrs{"Artist": "foo"}, []string{"AlbumArtist"})
	if r != "foo" {
		t.Errorf("AlbumArtist searches AlbumArtist and Artist but returns: %s", r)
	}

	r = songSortKey(mpd.Attrs{"Artist": "foo"}, []string{"AlbumArtistSort"})
	if r != "foo" {
		t.Errorf("AlbumArtist searches AlbumArtistSort, AlbumArtist and Artist but returns: %s", r)
	}
}
