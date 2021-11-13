package songs

// Copy shallow copies songs.
func Copy(s []map[string][]string) []map[string][]string {
	n := make([]map[string][]string, len(s))
	for i := range s {
		n[i] = s[i]
	}
	return n
}
