package main

import (
	"testing"
)

func TestFindCover(t *testing.T) {
	t.Run("found", func(t *testing.T) {
		ret := findCover("./hoge", "cover.*")
		if ret != "cover.go" {
			t.Errorf("unexpected result: %s", ret)
		}
	})
	t.Run("not found", func(t *testing.T) {
		ret := findCover("./hoge", "cover_not_found.*")
		if ret != "" {
			t.Errorf("unexpected result: %s", ret)
		}
	})
}
