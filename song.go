package main

import (
	"fmt"
	"github.com/meiraka/gompd/mpd"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func findCovers(dir, file, glob string, cache map[string]string) string {
	addr := path.Join(dir, file)
	d := path.Dir(addr)
	k := path.Join(d, glob)
	v, ok := cache[k]
	if ok {
		return v
	}
	m, err := filepath.Glob(k)
	if err != nil || m == nil {
		cache[k] = ""
		return ""
	}
	cover := strings.Replace(m[0], dir, "", -1)
	cache[k] = cover
	return cover
}

func getInt(m mpd.Tags, k string, e int) int {
	if d, found := m[k]; found {
		ret, err := strconv.Atoi(d[0])
		if err == nil {
			return ret
		}
	}
	return e
}

// MakeSong generate song metadata from mpd.Tags
func MakeSong(m mpd.Tags, dir, glob string, cache map[string]string) Song {
	track := getInt(m, "Track", 0)
	m["TrackNumber"] = []string{fmt.Sprintf("%04d", track)}
	disc := getInt(m, "Disc", 1)
	m["DiscNumber"] = []string{fmt.Sprintf("%04d", disc)}
	t := getInt(m, "Time", 0)
	m["Length"] = []string{fmt.Sprintf("%02d:%02d", t/60, t%60)}

	if _, found := m["file"]; found {
		if len(m["file"]) > 0 {
			cover := findCovers(dir, m["file"][0], glob, cache)
			if len(cover) > 0 {
				m["cover"] = []string{cover}
			}
		}
	}
	return Song(m)
}

// MakeSongs generate song metadata from []mpd.Tags
func MakeSongs(ps []mpd.Tags, dir, glob string, cache map[string]string) []Song {
	songs := make([]Song, 0, len(ps))
	for i := range ps {
		songs = append(songs, MakeSong(ps[i], dir, glob, cache))
	}
	return songs
}

// Song represents song metadata
type Song map[string][]string

// SortKey makes string for sort key by song tag list.
func (s Song) SortKey(keys []string) string {
	sp := []string{}
	for i := range keys {
		key := keys[i]
		if _, ok := s[key]; ok {
			sp = append(sp, strings.Join(s[key], ","))
		} else if key == "AlbumSort" {
			sp = append(sp, strings.Join(s.Tag([]string{"Album"}), ","))
		} else if key == "ArtistSort" {
			sp = append(sp, strings.Join(s.Tag([]string{"Artist"}), ","))
		} else if key == "AlbumArtist" {
			sp = append(sp, strings.Join(s.Tag([]string{"Artist"}), ","))
		} else if key == "AlbumArtistSort" {
			sp = append(sp, strings.Join(s.Tag([]string{"AlbumArtist", "Artist"}), ","))
		} else {
			sp = append(sp, " ")
		}
	}
	return strings.Join(sp, "")
}

func songAddAll(sp, add []string) []string {
	if add == nil || len(add) == 0 {
		return sp
	}
	if len(add) == 1 {
		for i := range sp {
			sp[i] = sp[i] + add[0]
		}
		return sp
	}
	newsp := make([]string, 0, len(sp)*len(add))
	for i := range sp {
		for j := range add {
			newsp = append(newsp, sp[i]+add[j])
		}
	}
	return newsp
}

// SortKeys makes string list for sort key by song tag list.
func (s Song) SortKeys(keys []string) []string {
	sp := []string{""}
	for i := range keys {
		key := keys[i]
		if _, ok := s[key]; ok {
			sp = songAddAll(sp, s[key])
		} else if key == "AlbumSort" {
			sp = songAddAll(sp, s.Tag([]string{"Album"}))
		} else if key == "ArtistSort" {
			sp = songAddAll(sp, s.Tag([]string{"Artist"}))
		} else if key == "AlbumArtist" {
			sp = songAddAll(sp, s.Tag([]string{"Artist"}))
		} else if key == "AlbumArtistSort" {
			sp = songAddAll(sp, s.Tag([]string{"AlbumArtist", "Artist"}))
		} else {
			sp = songAddAll(sp, []string{" "})
		}
	}
	return sp
}

func (s Song) String() string {
	kv := []string{}
	for k, v := range s {
		kv = append(kv, k+": "+strings.Join(v, " "))
	}
	sort.Strings(kv)
	return strings.Join(kv, ", ")
}

// Tag searches tags in song.
// returns nil if not found.
func (s Song) Tag(keys []string) []string {
	for i := range keys {
		key := keys[i]
		if _, ok := s[key]; ok {
			return s[key]
		}
	}
	return nil
}

type songSorter struct {
	song Song
	key  string
}

// SortSongs sorts songs by song tag list.
func SortSongs(s []Song, keys []string) []Song {
	flatten := make([]*songSorter, 0, len(s))
	for i := range s {
		for _, key := range s[i].SortKeys(keys) {
			flatten = append(flatten, &songSorter{s[i], key})
		}
	}
	sort.Slice(flatten, func(i, j int) bool {
		return flatten[i].key < flatten[j].key
	})
	ret := make([]Song, len(flatten))
	for i := range flatten {
		ret[i] = flatten[i].song
	}
	return ret
}

// SortSongsUniq sorts songs by song tag list.
func SortSongsUniq(s []Song, keys []string) []Song {
	n := make([]Song, len(s))
	copy(n, s)
	sort.Slice(n, func(i, j int) bool {
		return n[i].SortKey(keys) < n[j].SortKey(keys)
	})
	return n
}

// WeakFilterSongs removes songs if not matched by filters until len(songs) over max.
// filters example: [][]string{[]string{"Artist", "foo"}}
func WeakFilterSongs(s []Song, filters [][]string, max int) []Song {
	n := make([]Song, len(s))
	copy(n, s)
	for _, filter := range filters {
		if len(n) <= max {
			break
		}
		nc := []Song{}
		for _, song := range n {
			if _, found := song[filter[0]]; !found {
				continue
			}
			for _, value := range song[filter[0]] {
				if value == filter[1] {
					nc = append(nc, song)
					break
				}
			}
		}
		n = nc
	}
	if len(n) > max {
		nc := make([]Song, 0, max)
		for i := range n {
			if i < max {
				nc = append(nc, n[i])
			}
		}
		return nc
	}
	return n
}
