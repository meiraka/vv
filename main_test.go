package main

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/spf13/viper"
)

func init() {
	viper.AddConfigPath("./appendix")
	setupFlag("example.config")
}

func TestReadConfig(t *testing.T) {
	testsets1 := map[string]string{
		"mpd.host":            "",
		"mpd.port":            "",
		"mpd.network":         "tcp",
		"mpd.music_directory": "/path/to/music/dir",
		"server.port":         "",
		"server.addr":         ":8080",
	}
	testsets2 := map[string]bool{
		"server.keepalive": true,
		"debug":            false,
	}
	err := viper.ReadInConfig()
	if err != nil {
		t.Errorf("unexpected err: %s", err.Error())
	}
	for input, expect := range testsets1 {
		actual := viper.GetString(input)
		if actual != expect {
			t.Errorf("unexpected value for %s, actual:%s, expect:%s", input, actual, expect)
		}
	}
	for input, expect := range testsets2 {
		actual := viper.GetBool(input)
		if actual != expect {
			t.Errorf("unexpected value for %s, actual:%t, expect:%t", input, actual, expect)
		}
	}
}

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
