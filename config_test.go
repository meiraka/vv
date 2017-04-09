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
