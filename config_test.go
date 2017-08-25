package main

import (
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
	"testing"
)

func TestReadConfig(t *testing.T) {
	viper.AddConfigPath("./")
	const path = "./config.yaml"
	input := []byte(
		"mpd:\n" +
			"    host: \"hoge.local\"\n" +
			"    port: \"6600\"\n" +
			"    music_directory: \"hoge\"\n" +
			"server:\n" +
			"    port: \"8080\"\n",
	)
	ioutil.WriteFile(path, input, os.ModePerm)
	expected := Config{
		Mpd:    MpdConfig{"hoge.local", "6600", "hoge"},
		Server: ServerConfig{Port: "8080"}}
	actual, err := ReadConfig()
	if err != nil {
		t.Errorf("got unexpected err: %v", err)
	}
	if actual != expected {
		t.Errorf("got %v\nwant %v", actual, expected)
	}
	os.Remove(path)
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
