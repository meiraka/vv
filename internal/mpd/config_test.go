package mpd

import (
	"reflect"
	"testing"
)

func TestParseConfig(t *testing.T) {
	c, err := ParseConfig("mpd.conf")
	if err != nil {
		t.Fatalf("got parse err %v", err)
	}
	want := &Config{
		MusicDirectory: "/mnt/Music/NAS/storage",
		AudioOutputs: []*ConfigAudioOutput{
			{Name: "My ALSA Device", Type: "alsa"},
			{Name: "My HTTP Stream", Type: "httpd", Port: "8000"},
		},
	}
	if !reflect.DeepEqual(c, want) {
		t.Errorf("got %+v, want %+v", c, want)
	}
}
