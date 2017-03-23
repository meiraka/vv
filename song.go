package main

import (
	"fmt"
	"github.com/fhs/gompd/mpd"
	"strconv"
	"strings"
)

/*songAddReadableData adds tags.
 * TrackNumber: 0 filled Track
 * DiscNumber: 0 filled Disc
 * Length: MM:SS styled Time
 */
func songAddReadableData(m mpd.Attrs) mpd.Attrs {
	track, err := strconv.Atoi(m["Track"])
	if err != nil {
		track = 0
	}
	m["TrackNumber"] = fmt.Sprintf("%04d", track)
	disc, err := strconv.Atoi(m["Disc"])
	if err != nil {
		disc = 1
	}
	m["DiscNumber"] = fmt.Sprintf("%04d", disc)

	t, err := strconv.Atoi(m["Time"])
	if err != nil {
		t = 0
	}
	m["Length"] = fmt.Sprintf("%02d:%02d", t/60, t%60)
	return m
}

// songTag searches tags in song.
// returns empty string if not found.
func songTag(s mpd.Attrs, keys []string) string {
	for i := range keys {
		key := keys[i]
		if _, ok := s[key]; ok {
			return s[key]
		}
	}
	return " "
}

// songSortKey makes string for sort key by song tag list.
func songSortKey(s mpd.Attrs, keys []string) string {
	sp := make([]string, len(keys))
	for i := range keys {
		key := keys[i]
		if _, ok := s[key]; ok {
			sp = append(sp, s[key])
		} else if key == "AlbumSort" {
			sp = append(sp, songTag(s, []string{"Album"}))
		} else if key == "ArtistSort" {
			sp = append(sp, songTag(s, []string{"Artist"}))
		} else if key == "AlbumArtist" {
			sp = append(sp, songTag(s, []string{"Artist"}))
		} else if key == "AlbumArtistSort" {
			sp = append(sp, songTag(s, []string{"AlbumArtist", "Artist"}))
		} else {
			sp = append(sp, " ")
		}
	}
	return strings.Join(sp, "")
}

func songsAddReadableData(ps []mpd.Attrs) []mpd.Attrs {
	for i := range ps {
		ps[i] = songAddReadableData(ps[i])
	}
	return ps
}
