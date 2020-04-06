package main

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestGetMusicDirectory(t *testing.T) {
	inputPath := "mpd.conf"
	testsets := []struct {
		input       []byte
		readPath    string
		expect      string
		expectError bool
	}{
		{
			input:       []byte("music_directory \t\"hoge\"\n"),
			readPath:    inputPath,
			expect:      "hoge",
			expectError: false,
		},
		{
			input:       []byte("MUSIC_DIRECTORY \t\"hoge\"\n"),
			readPath:    inputPath,
			expect:      "",
			expectError: false,
		},
		{
			input:       []byte("music_directory hoge"),
			readPath:    inputPath,
			expect:      "",
			expectError: false,
		},
		{
			input:       []byte("music_directory \"hoge\""),
			readPath:    "not found",
			expect:      "",
			expectError: true,
		},
	}
	for _, tt := range testsets {
		ioutil.WriteFile(inputPath, tt.input, os.ModePerm)
		actual, err := getMusicDirectory(tt.readPath)
		if actual != tt.expect {
			t.Errorf("got unexpected result. expect: %s, actual: %s", tt.expect, actual)
		}
		if !tt.expectError && err != nil {
			t.Errorf("got unexpected err: %v", err)
		}
		if tt.expectError && err == nil {
			t.Errorf("got unexpected nil")
		}
	}
	os.Remove(inputPath)

}
