package songs

// Copy shallow copies songs.
func Copy(s []map[string][]string) []map[string][]string {
	n := make([]map[string][]string, len(s))
	copy(n, s)
	return n
}
