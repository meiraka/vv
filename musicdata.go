package main

import (
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/meiraka/gompd/mpd"
)

// TagAdder add tags to song.
type TagAdder interface {
	AddTags(map[string][]string) map[string][]string
}

// TagAdderFunc converts func to TagAdder
type TagAdderFunc func(map[string][]string) map[string][]string

// AddTags add tags to song.
func (t TagAdderFunc) AddTags(m map[string][]string) map[string][]string {
	return t(m)
}

// LocalCoverSearcher searches song conver art
type LocalCoverSearcher struct {
	dir   string
	glob  string
	cache map[string]string
	image map[string]struct{}
	mu    sync.RWMutex
}

// NewLocalCoverSearcher creates LocalCoverSearcher.
func NewLocalCoverSearcher(dir, glob string) (*LocalCoverSearcher, error) {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	return &LocalCoverSearcher{
		dir:   dir,
		glob:  glob,
		cache: map[string]string{},
		image: map[string]struct{}{},
	}, nil
}

// CachedImage returns true if given path is cached image
func (f *LocalCoverSearcher) CachedImage(path string) (cached bool) {
	f.mu.RLock()
	_, cached = f.image[path]
	f.mu.RUnlock()
	return
}

// AddTags adds cover path to m
func (f *LocalCoverSearcher) AddTags(m map[string][]string) map[string][]string {
	file, ok := m["file"]
	if !ok {
		return m
	}
	if len(file) != 1 {
		return m
	}
	localPath := filepath.Join(filepath.FromSlash(f.dir), filepath.FromSlash(file[0]))
	localGlob := filepath.Join(filepath.Dir(localPath), f.glob)
	f.mu.Lock()
	defer f.mu.Unlock()
	v, ok := f.cache[localGlob]
	if ok {
		if len(v) != 0 {
			m["cover"] = []string{v}
		}
		return m
	}
	p, err := filepath.Glob(localGlob)
	if err != nil || p == nil {
		f.cache[localGlob] = ""
		return m
	}
	cover := strings.TrimPrefix(strings.TrimPrefix(filepath.ToSlash(p[0]), filepath.ToSlash(f.dir)), "/")
	f.cache[localGlob] = cover
	f.image[cover] = struct{}{}
	m["cover"] = []string{cover}
	return m
}

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

func findCovers(dir, file, glob string, cache map[string]string) string {
	dir, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
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
	cover := strings.TrimPrefix(strings.TrimPrefix(m[0], dir), "/")
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
	songs := make([]Song, len(ps))
	for i := range ps {
		songs[i] = MakeSong(ps[i], dir, glob, cache)
	}
	return songs
}

// MakeStatus generates music control status from mpd.Attrs
func MakeStatus(status mpd.Attrs) Status {
	volume, err := strconv.Atoi(status["volume"])
	if err != nil {
		volume = -1
	}
	repeat := status["repeat"] == "1"
	random := status["random"] == "1"
	single := status["single"] == "1"
	consume := status["consume"] == "1"
	state := status["state"]
	if state == "" {
		state = "stopped"
	}
	songpos, err := strconv.Atoi(status["song"])
	if err != nil {
		songpos = 0
	}
	elapsed, err := strconv.ParseFloat(status["elapsed"], 64)
	if err != nil {
		elapsed = 0.0
	}
	_, found := status["updating_db"]
	updateLibrary := found
	return Status{
		volume,
		repeat,
		random,
		single,
		consume,
		state,
		songpos,
		float32(elapsed),
		updateLibrary,
	}
}

// Song represents song metadata
type Song map[string][]string

func songAddAll(sp []map[string]string, key string, add []string) []map[string]string {
	if add == nil || len(add) == 0 {
		for i := range sp {
			sp[i]["all"] = sp[i]["all"] + " "
			sp[i][key] = " "
		}
		return sp
	}
	if len(add) == 1 {
		for i := range sp {
			sp[i]["all"] = sp[i]["all"] + add[0]
			sp[i][key] = add[0]
		}
		return sp
	}
	newsp := make([]map[string]string, len(sp)*len(add))
	index := 0
	for i := range sp {
		for j := range add {
			spd := make(map[string]string, len(sp[i]))
			for k := range sp[i] {
				spd[k] = sp[i][k]
			}
			spd["all"] = spd["all"] + add[j]
			spd[key] = add[j]
			newsp[index] = spd
			index++
		}
	}
	return newsp
}

// SortKeys makes map list for sort key by song tag list.
func (s Song) SortKeys(keys []string) []map[string]string {
	sp := []map[string]string{{"all": ""}}
	for _, key := range keys {
		sp = songAddAll(sp, key, s.Tags(key))
	}
	return sp
}

// Tags returns "-" separated tags values in song.
// returns nil if not found.
func (s Song) Tags(tags string) []string {
	keys := strings.Split(tags, "-")
	var ret []string
	for _, key := range keys {
		t := s.Tag(key)
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
func (s Song) Tag(key string) []string {
	if v, found := s[key]; found {
		return v
	} else if key == "AlbumArtist" {
		return s.Tag("Artist")
	} else if key == "AlbumSort" {
		return s.Tag("Album")
	} else if key == "ArtistSort" {
		return s.Tag("Artist")
	} else if key == "AlbumArtistSort" {
		return s.TagSearch([]string{"AlbumArtist", "Artist"})
	}
	return nil
}

// TagSearch searches tags in song.
// returns nil if not found.
func (s Song) TagSearch(keys []string) []string {
	for i := range keys {
		key := keys[i]
		if _, ok := s[key]; ok {
			return s[key]
		}
	}
	return nil
}

/*Status represents mpd status.*/
type Status struct {
	Volume        int     `json:"volume"`
	Repeat        bool    `json:"repeat"`
	Random        bool    `json:"random"`
	Single        bool    `json:"single"`
	Consume       bool    `json:"consume"`
	State         string  `json:"state"`
	SongPos       int     `json:"song_pos"`
	SongElapsed   float32 `json:"song_elapsed"`
	UpdateLibrary bool    `json:"update_library"`
}

type songSorter struct {
	song   Song
	key    map[string]string
	target bool
}

// SortSongs sorts songs by song tag list.
func SortSongs(s []Song, keys []string, filters [][]string, max, pos int) ([]Song, [][]string, int) {
	flatten := sortSongs(s, keys)
	if pos < len(flatten) && pos >= 0 {
		flatten[pos].target = true
	}
	flatten, used := weakFilterSongs(flatten, filters, max)
	ret := make([]Song, len(flatten))
	newpos := -1
	for i, sorter := range flatten {
		ret[i] = sorter.song
		if sorter.target {
			newpos = i
		}
	}
	return ret, used, newpos
}

func sortSongs(s []Song, keys []string) []*songSorter {
	flatten := make([]*songSorter, 0, len(s))
	for _, song := range s {
		for _, key := range song.SortKeys(keys) {
			flatten = append(flatten, &songSorter{song, key, false})
		}
	}
	sort.Slice(flatten, func(i, j int) bool {
		return flatten[i].key["all"] < flatten[j].key["all"]
	})
	return flatten
}

// weakFilterSongs removes songs if not matched by filters until len(songs) over max.
// filters example: [][]string{[]string{"Artist", "foo"}}
func weakFilterSongs(s []*songSorter, filters [][]string, max int) ([]*songSorter, [][]string) {
	used := [][]string{}
	if len(s) <= max {
		return s, used
	}
	n := s
	for _, filter := range filters {
		if len(n) <= max {
			break
		}
		used = append(used, filter)
		nc := make([]*songSorter, 0, len(n))
		for _, sorter := range n {
			if value, found := sorter.key[filter[0]]; found && value == filter[1] {
				nc = append(nc, sorter)
			}
		}
		n = nc
	}
	if len(n) > max {
		nc := make([]*songSorter, max)
		for i := range n {
			if i < max {
				nc[i] = n[i]
			}
		}
		return nc, used
	}
	return n, used
}
