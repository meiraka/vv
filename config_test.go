package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"gopkg.in/yaml.v2"
)

func TestParseConfigExample(t *testing.T) {
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
	if !reflect.DeepEqual(config, want) {
		t.Errorf("got %+v; want %+v", config, want)
	}
}

func TestParseConfigOptions(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()
	os.Args = []string{origArgs[0],
		"-d",
		"--mpd.conf", "/local/etc/mpd.conf",
		"--mpd.network", "unix",
		"--mpd.addr", "/var/run/mpd/socket",
		"--mpd.music_directory", "/mnt/Music",
		"--server.addr", ":80",
		"--server.cover.remote",
	}
	config, date, err := ParseConfig(nil, "example.config.yaml")
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}
	if !date.IsZero() {
		t.Errorf("got date %v; want zer", date)
	}
	want := &Config{}
	want.debug = true
	want.MPD.Network = "unix"
	want.MPD.Addr = "/var/run/mpd/socket"
	want.MPD.Conf = "/local/etc/mpd.conf"
	want.MPD.MusicDirectory = "/mnt/Music"
	want.Server.Addr = ":80"
	want.Server.CacheDirectory = "/tmp/vv"
	want.Server.Cover.Local = true
	want.Server.Cover.Remote = true
	if !reflect.DeepEqual(config, want) {
		t.Errorf("got \n%+v; want \n%+v", config, want)
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

// TestValidateErrorText tests validation error text for logs readability.
func TestValidateErrorText(t *testing.T) {
	for yamlText, errStr := range map[string]string{
		`{"playlist":{"tree_order":["foo","foo"],"tree":{"foo":{"sort":["file"],"tree":[["file","song"]]}}}}`:                                                  "playlist.tree_order foo is duplicated",
		`{"playlist":{"tree_order":["foo","bar"],"tree":{"foo":{"sort":["file"],"tree":[["file","song"]]}}}}`:                                                  "playlist.tree.bar is not defined in playlist.tree",
		`{"playlist":{"tree_order":["foo","bar"],"tree":{"foo":{"sort":["file"],"tree":[["file","song"]]},"bar":{"sort":["file"],"tree":[["file","song"]]}}}}`: "playlist.tree.*.sort must be unique: playlist.tree.foo.sort and playlist.tree.bar.sort has same value",

		`{"playlist":{"tree_order":["foo"],"tree":{"foo":{"sort":[],"tree":[["file","song"]]}}}}`:        "playlist.tree.foo: sort or tree must not be empty",
		`{"playlist":{"tree_order":["foo"],"tree":{"foo":{"sort":["file"],"tree":[]}}}}`:                 "playlist.tree.foo: sort or tree must not be empty",
		`{"playlist":{"tree_order":["foo"],"tree":{"foo":{"sort":["file"],"tree":[["title","song"]]}}}}`: "playlist.tree.foo: tree: index 0:0: tree tag must be defined in sort: title does not defined in sort: ",     // do not include []string printf representation
		`{"playlist":{"tree_order":["foo"],"tree":{"foo":{"sort":["file"],"tree":[["file","foo"]]}}}}`:   "playlist.tree.foo: tree: index 0:1: unsupported tree view type: got foo; supported tree element views are ", // do not include []string printf representation
	} {
		t.Run(errStr, func(t *testing.T) {
			c := Config{}
			if err := yaml.Unmarshal([]byte(yamlText), &c); err != nil {
				t.Fatalf("failed to parse config: %v", err)
			}
			err := c.Validate()
			if err == nil {
				t.Fatalf("yaml is valid")
			}
			if !strings.HasPrefix(err.Error(), errStr) {
				t.Errorf("got err %v", err)
			}
		})
	}

}
