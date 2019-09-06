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

func sortKeys(s map[string][]string, keys []string) []map[string]string {
	sp := []map[string]string{{"all": ""}}
	for _, key := range keys {
		sp = addAllKeys(sp, key, Tags(s, key))
	}
	return sp
}

func addAllKeys(sp []map[string]string, key string, add []string) []map[string]string {
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

type sorter struct {
	song   map[string][]string
	key    map[string]string
	target bool
}

// WeakFilterSort sorts songs by song tag list.
func WeakFilterSort(s []map[string][]string, keys []string, filters [][]string, max, pos int) ([]map[string][]string, [][]string, int) {
	flatten := flat(s, keys)
	sort.Slice(flatten, func(i, j int) bool {
		return flatten[i].key["all"] < flatten[j].key["all"]
	})
	if pos < len(flatten) && pos >= 0 {
		flatten[pos].target = true
	}
	flatten, used := weakFilterSongs(flatten, filters, max)
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
		for _, key := range sortKeys(song, keys) {
			flatten = append(flatten, &sorter{song, key, false})
		}
	}
	return flatten
}

// weakFilterSongs removes songs if not matched by filters until len(songs) over max.
// filters example: [][]string{[]string{"Artist", "foo"}}
func weakFilterSongs(s []*sorter, filters [][]string, max int) ([]*sorter, [][]string) {
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
		nc := make([]*sorter, 0, len(n))
		for _, sorter := range n {
			if value, found := sorter.key[filter[0]]; found && value == filter[1] {
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
