package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
)

// Config is vv application config struct.
type Config struct {
	MPD struct {
		Network        string `yaml:"network"`
		Addr           string `yaml:"addr"`
		MusicDirectory string `yaml:"music_directory"`
	} `yaml:"mpd"`
	Server struct {
		Addr string `yaml:"addr"`
	} `yaml:"server"`
	Playlist struct {
		Tree      map[string]*ListNode `yaml:"tree"`
		TreeOrder []string             `yaml:"tree_order"`
	}
	debug bool
}

// ParseConfig parse yaml config and flags.
func ParseConfig(dir []string) (*Config, time.Time, error) {
	c := &Config{}
	date := time.Time{}
	for _, d := range dir {
		path := filepath.Join(d, "config.yaml")
		_, err := os.Stat(path)
		if err == nil {
			f, err := os.Open(path)
			if err != nil {
				return nil, date, err
			}
			s, err := f.Stat()
			if err != nil {
				return nil, date, err
			}
			date = s.ModTime()
			defer f.Close()
			c := Config{}
			if err := yaml.NewDecoder(f).Decode(&c); err != nil {
				return nil, date, err
			}
		}
	}

	mn := pflag.String("mpd.network", "", "mpd server network to connect")
	ma := pflag.String("mpd.addr", "", "mpd server address to connect")
	mm := pflag.String("mpd.music_directory", "", "set music_directory in mpd.conf value to search album cover image")
	sa := pflag.String("server.addr", "", "this app serving address")
	d := pflag.BoolP("debug", "d", false, "use local assets if exists")
	pflag.Parse()
	if len(*mn) != 0 {
		c.MPD.Network = *mn
	}
	if len(*ma) != 0 {
		c.MPD.Addr = *ma
	}
	if len(*mm) != 0 {
		c.MPD.MusicDirectory = *mm
	}
	if len(*sa) != 0 {
		c.Server.Addr = *sa
	}
	c.debug = *d
	c.setDefault()
	return c, date, nil
}

func (c *Config) setDefault() {
	if c.MPD.Network == "" {
		c.MPD.Network = "tcp"
	}
	if c.MPD.Addr == "" {
		c.MPD.Addr = ":6600"
	}
	if c.Server.Addr == "" {
		c.Server.Addr = ":8080"
	}
	if c.Playlist.Tree == nil && c.Playlist.TreeOrder == nil {
		c.Playlist.Tree = defaultTree
		c.Playlist.TreeOrder = defaultTreeOrder
	}
}

// Validate validates config data.
func (c *Config) Validate() error {
	set := make(map[string]struct{}, len(c.Playlist.TreeOrder))
	for _, label := range c.Playlist.TreeOrder {
		if _, ok := set[label]; ok {
			return fmt.Errorf("playlist.tree_order %s is duplicated", label)
		}
		set[label] = struct{}{}
		node, ok := c.Playlist.Tree[label]
		if !ok {
			return fmt.Errorf("playlist.tree_order %s is not defined in tree", label)
		}
		if err := node.Validate(); err != nil {
			return fmt.Errorf("playlist.tree label %s: %w", label, err)
		}
	}
	if t, o := len(c.Playlist.Tree), len(c.Playlist.TreeOrder); o != t {
		return fmt.Errorf("playlist.tree length (%d) and playlist.tree_order length (%d) mismatch", t, o)
	}
	return nil
}

var (
	defaultTree = map[string]*ListNode{
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
	defaultTreeOrder = []string{"AlbumArtist", "Album", "Artist", "Genre", "Date", "Composer", "Performer"}
	supportTreeViews = []string{"plain", "album", "song"}
)

type ListNode struct {
	Sort []string    `json:"sort"`
	Tree [][2]string `json:"tree"`
}

// Validate ListNode data struct.
func (l *ListNode) Validate() error {
	if len(l.Tree) > 4 {
		return fmt.Errorf("maximum tree length is 4; got %d", len(l.Tree))
	}
	for i, leef := range l.Tree {
		for _, view := range supportTreeViews {
			if view == leef[1] {
				goto OK
			}
		}
		return fmt.Errorf("index %d, supported tree element views are %v; got %s", i, supportTreeViews, leef[1])
	OK:
	}
	return nil

}
