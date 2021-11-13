package songs

import "sort"

// SortEqual compares song filepath is equal
func SortEqual(o, n []map[string][]string) bool {
	if len(o) != len(n) {
		return false
	}
	for i := range n {
		f1 := o[i]["file"][0]
		f2 := n[i]["file"][0]
		if f1 != f2 {
			return false
		}
	}
	return true
}

type sorter struct {
	song    map[string][]string
	keys    map[string]*string
	sortkey string
	target  bool
}

func sortKeys(s map[string][]string, keys []string) []*sorter {
	sp := []*sorter{{song: s, keys: make(map[string]*string, len(keys))}}
	for _, key := range keys {
		sp = addAllKeys(sp, key, Tags(s, key))
	}
	return sp
}

func addAllKeys(sp []*sorter, key string, add []string) []*sorter {
	if len(add) == 0 {
		for i := range sp {
			sp[i].sortkey = sp[i].sortkey + " "
			sp[i].keys[key] = nil
		}
		return sp
	}
	if len(add) == 1 {
		for i := range sp {
			sp[i].sortkey = sp[i].sortkey + add[0]
			sp[i].keys[key] = &add[0]
		}
		return sp
	}
	newsp := make([]*sorter, len(sp)*len(add))
	index := 0
	for i := range sp {
		for j := range add {
			s := &sorter{song: sp[i].song, keys: make(map[string]*string, len(sp[i].keys))}
			for k := range sp[i].keys {
				s.keys[k] = sp[i].keys[k]
			}
			s.sortkey = s.sortkey + add[j]
			s.keys[key] = &add[j]
			newsp[index] = s
			index++
		}
	}
	return newsp
}

// WeakFilterSort sorts songs by song tag list.
func WeakFilterSort(s []map[string][]string, keys []string, filters [][2]*string, must, max, pos int) ([]map[string][]string, [][2]*string, int) {
	flatten := flat(s, keys)
	sort.Slice(flatten, func(i, j int) bool {
		return flatten[i].sortkey < flatten[j].sortkey
	})
	if pos < len(flatten) && pos >= 0 {
		flatten[pos].target = true
	}
	flatten, used := weakFilterSongs(flatten, filters, must, max)
	ret := make([]map[string][]string, len(flatten))
	newpos := -1
	for i, sorter := range flatten {
		ret[i] = sorter.song
		if sorter.target {
			newpos = i
		}
	}
	return ret, used, newpos
}

func flat(s []map[string][]string, keys []string) []*sorter {
	flatten := make([]*sorter, 0, len(s))
	for _, song := range s {
		flatten = append(flatten, sortKeys(song, keys)...)
	}
	return flatten
}

// weakFilterSongs removes songs if not matched by filters until len(songs) over max.
// filters example: [][]string{[]string{"Artist", "foo"}}
func weakFilterSongs(s []*sorter, filters [][2]*string, must, max int) ([]*sorter, [][2]*string) {
	used := [][2]*string{}
	if len(s) <= max && must == 0 {
		return s, used
	}
	n := s
	for i, filter := range filters {
		if len(n) <= max && must <= i {
			break
		}
		used = append(used, filter)
		nc := make([]*sorter, 0, len(n))
		for _, sorter := range n {
			key, want := filter[0], filter[1]
			if key == nil {
				nc = append(nc, sorter)
			} else if value := sorter.keys[*key]; (value != nil && want != nil && *value == *want) || (value == nil && want == nil) {
				nc = append(nc, sorter)
			}
		}
		n = nc
	}
	if len(n) > max {
		nc := make([]*sorter, max)
		for i := range n {
			if i < max {
				nc[i] = n[i]
			}
		}
		return nc, used
	}
	return n, used
}

func strPtr(s string) *string { return &s }
