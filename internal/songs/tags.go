package songs

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// AddTags adds tags to song for vv
// TrackNumber, DiscNumber are used for sorting.
// Length is used for displaing time.
func AddTags(m map[string][]string) map[string][]string {
	track := getIntTag(m, "Track", 0)
	m["TrackNumber"] = []string{fmt.Sprintf("%04d", track)}
	disc := getIntTag(m, "Disc", 1)
	m["DiscNumber"] = []string{fmt.Sprintf("%04d", disc)}
	t := getIntTag(m, "Time", 0)
	m["Length"] = []string{fmt.Sprintf("%02d:%02d", t/60, t%60)}
	if l, ok := m["Last-Modified"]; ok && len(l) == 1 {
		if lt, err := time.Parse(time.RFC3339, l[0]); err == nil {
			m["LastModifiedDate"] = []string{lt.Format("2006.01.02")}

		}
	}
	return m
}

func getIntTag(m map[string][]string, k string, e int) int {
	if d, found := m[k]; found {
		ret, err := strconv.Atoi(d[0])
		if err == nil {
			return ret
		}
	}
	return e
}

// Tags returns "-" separated tags values in song.
// returns nil if not found.
func Tags(s map[string][]string, tags string) []string {
	keys := strings.Split(tags, "-")
	var ret []string
	for _, key := range keys {
		t := Tag(s, key)
		if ret == nil {
			ret = t
		} else if t != nil {
			newret := []string{}
			for _, old := range ret {
				for _, new := range t {
					newret = append(newret, old+"-"+new)
				}
			}
			ret = newret
		}
	}
	return ret
}

// Tag returns tag values in song.
// returns nil if not found.
func Tag(s map[string][]string, key string) []string {
	if v, found := s[key]; found {
		return v
	} else if key == "AlbumArtist" {
		return Tag(s, "Artist")
	} else if key == "AlbumSort" {
		return Tag(s, "Album")
	} else if key == "ArtistSort" {
		return Tag(s, "Artist")
	} else if key == "AlbumArtistSort" {
		return TagSearch(s, []string{"AlbumArtist", "Artist"})
	} else if key == "Date" {
		if v, found := s["OriginalDate"]; found {
			return v
		}
	} else if key == "OriginalDate" {
		if v, found := s["Date"]; found {
			return v
		}
	}
	return nil
}

// TagSearch searches tags in song.
// returns nil if not found.
func TagSearch(s map[string][]string, keys []string) []string {
	for i := range keys {
		key := keys[i]
		if _, ok := s[key]; ok {
			return s[key]
		}
	}
	return nil
}
