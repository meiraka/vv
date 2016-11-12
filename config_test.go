package main

import (
	"io/ioutil"
	"os"
	"testing"
)

func TestReadConfig(t *testing.T) {
	const path = "./config_test_rc"
	input := []byte(
		"[mpd]\n" +
			"host = \"localhost\"\n" +
			"port = \"6600\"\n" +
			"[server]\n" +
			"port = \"8080\"\n",
	)
	ioutil.WriteFile(path, input, os.ModePerm)
	expected := Config{Mpd: MpdConfig{Host: "localhost", Port: "6600"}, Server: ServerConfig{Port: "8080"}}
	actual, err := ReadConfig(path)
	if err != nil {
		t.Errorf("got unexpected err: %v", err)
	}
	if actual != expected {
		t.Errorf("got %v\nwant %v", actual, expected)
	}
	os.Remove(path)
}
