package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestParseConfig(t *testing.T) {
	config, date, err := ParseConfig([]string{"appendix"}, "example.config.yaml")
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	s, err := os.Stat(filepath.Join("appendix", "example.config.yaml"))
	if err != nil {
		t.Fatalf("failed to stat config file: %v", err)
	}
	if !date.Equal(s.ModTime()) {
		t.Errorf("got date %v; want %v", date, s.ModTime())
	}
	want := &Config{}
	want.MPD.Network = "tcp"
	want.MPD.Addr = "localhost:6600"
	want.MPD.MusicDirectory = "/path/to/music/dir"
	want.MPD.Conf = "/etc/mpd.conf"
	want.Server.Addr = ":8080"
	want.Server.CacheDirectory = "/tmp/vv"
	want.Server.Cover.Local = true
	want.Server.Cover.Remote = true
	want.Playlist.Tree = map[string]*ConfigListNode{
		"AlbumArtist": {
			Sort: []string{"AlbumArtist", "Date", "Album", "DiscNumber", "TrackNumber", "Title", "file"},
			Tree: [][2]string{{"AlbumArtist", "plain"}, {"Album", "album"}, {"Title", "song"}},
		},
		"Album": {
			Sort: []string{"Date-Album", "DiscNumber", "TrackNumber", "Title", "file"},
			Tree: [][2]string{{"Date-Album", "album"}, {"Title", "song"}},
		},
		"Artist": {
			Sort: []string{"Artist", "Date", "Album", "DiscNumber", "TrackNumber", "Title", "file"},
			Tree: [][2]string{{"Artist", "plain"}, {"Title", "song"}},
		},
		"Genre": {
			Sort: []string{"Genre", "Album", "DiscNumber", "TrackNumber", "Title", "file"},
			Tree: [][2]string{{"Genre", "plain"}, {"Album", "album"}, {"Title", "song"}},
		},
		"Date": {
			Sort: []string{"Date", "Album", "DiscNumber", "TrackNumber", "Title", "file"},
			Tree: [][2]string{{"Date", "plain"}, {"Album", "album"}, {"Title", "song"}},
		},
		"Composer": {
			Sort: []string{"Composer", "Date", "Album", "DiscNumber", "TrackNumber", "Title", "file"},
			Tree: [][2]string{{"Composer", "plain"}, {"Album", "album"}, {"Title", "song"}},
		},
		"Performer": {
			Sort: []string{"Performer", "Date", "Album", "DiscNumber", "TrackNumber", "Title", "file"},
			Tree: [][2]string{{"Performer", "plain"}, {"Album", "album"}, {"Title", "song"}},
		},
	}
	want.Playlist.TreeOrder = []string{"AlbumArtist", "Album", "Artist", "Genre", "Date", "Composer", "Performer"}
	if !reflect.DeepEqual(config, want) {
		t.Errorf("got %+v; want %+v", config, want)
	}
}

func TestParseConfigDefault(t *testing.T) {
	config, date, err := ParseConfig(nil, "example.config.yaml")
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if !date.IsZero() {
		t.Errorf("got date %v; want zer", date)
	}
	want := &Config{}
	want.MPD.Network = "tcp"
	want.MPD.Addr = "localhost:6600"
	want.MPD.Conf = "/etc/mpd.conf"
	want.Server.Addr = ":8080"
	want.Server.CacheDirectory = "/tmp/vv"
	want.Server.Cover.Local = true
	want.Playlist.Tree = map[string]*ConfigListNode{
		"AlbumArtist": {
			Sort: []string{"AlbumArtist", "Date", "Album", "DiscNumber", "TrackNumber", "Title", "file"},
			Tree: [][2]string{{"AlbumArtist", "plain"}, {"Album", "album"}, {"Title", "song"}},
		},
		"Album": {
			Sort: []string{"AlbumArtist-Date-Album", "DiscNumber", "TrackNumber", "Title", "file"},
			Tree: [][2]string{{"AlbumArtist-Date-Album", "album"}, {"Title", "song"}},
		},
		"Artist": {
			Sort: []string{"Artist", "Date", "Album", "DiscNumber", "TrackNumber", "Title", "file"},
			Tree: [][2]string{{"Artist", "plain"}, {"Title", "song"}},
		},
		"Genre": {
			Sort: []string{"Genre", "Album", "DiscNumber", "TrackNumber", "Title", "file"},
			Tree: [][2]string{{"Genre", "plain"}, {"Album", "album"}, {"Title", "song"}},
		},
		"Date": {
			Sort: []string{"Date", "Album", "DiscNumber", "TrackNumber", "Title", "file"},
			Tree: [][2]string{{"Date", "plain"}, {"Album", "album"}, {"Title", "song"}},
		},
		"Composer": {
			Sort: []string{"Composer", "Date", "Album", "DiscNumber", "TrackNumber", "Title", "file"},
			Tree: [][2]string{{"Composer", "plain"}, {"Album", "album"}, {"Title", "song"}},
		},
		"Performer": {
			Sort: []string{"Performer", "Date", "Album", "DiscNumber", "TrackNumber", "Title", "file"},
			Tree: [][2]string{{"Performer", "plain"}, {"Album", "album"}, {"Title", "song"}},
		},
	}
	want.Playlist.TreeOrder = []string{"AlbumArtist", "Album", "Artist", "Genre", "Date", "Composer", "Performer"}
	if !reflect.DeepEqual(config, want) {
		t.Errorf("got %+v; want %+v", config, want)
	}
}

func TestConfigYAML(t *testing.T) {
	f, err := os.Open("./appendix/example.config.yaml")
	if err != nil {
		t.Fatalf("failed to open test config yaml: %v", err)
	}
	defer f.Close()
	c := Config{}
	if err := yaml.NewDecoder(f).Decode(&c); err != nil {
		t.Fatalf("failed to parse config: %v", err)
	}
	if err := c.Validate(); err != nil {
		t.Errorf("config validate failed: %v", err)
	}
}
