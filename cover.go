package main

import (
	"path"
	"path/filepath"
)

func findCover(addr, glob string) string {
	d := path.Dir(addr)
	m, err := filepath.Glob(path.Join(d, glob))
	if err != nil || m == nil {
		return ""
	}
	return m[0]
}
