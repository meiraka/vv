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
		Conf           string `yaml:"conf"`
	} `yaml:"mpd"`
	Server struct {
		Addr           string `yaml:"addr"`
		CacheDirectory string `yaml:"cache_directory"`
		Cover          struct {
			Local  bool `yaml:"local"`
			Remote bool `yaml:"remote"`
		} `yaml:"cover"`
	} `yaml:"server"`
	Playlist struct {
		Tree      map[string]*ConfigListNode `yaml:"tree"`
		TreeOrder []string                   `yaml:"tree_order"`
	}
	debug bool
}

var (
	// FIXME
	mn = pflag.String("mpd.network", "", "mpd server network to connect")
	ma = pflag.String("mpd.addr", "", "mpd server address to connect")
	mm = pflag.String("mpd.music_directory", "", "set music_directory in mpd.conf value to search album cover image")
	mc = pflag.String("mpd.conf", "", "set mpd.conf path to get music_directory and http audio output")
	sa = pflag.String("server.addr", "", "this app serving address")
	si = pflag.Bool("server.cover.remote", false, "enable coverart via mpd api")
	d  = pflag.BoolP("debug", "d", false, "use local assets if exists")
)

// ParseConfig parse yaml config and flags.
func ParseConfig(dir []string, name string) (*Config, time.Time, error) {
	c := &Config{}
	c.Server.Cover.Local = true
	c.Server.CacheDirectory = filepath.Join(os.TempDir(), "vv")
	c.MPD.Conf = "/etc/mpd.conf"
	date := time.Time{}
	for _, d := range dir {
		path := filepath.Join(d, name)
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
			if err := yaml.NewDecoder(f).Decode(&c); err != nil {
				return nil, date, err
			}
		}
	}

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
	if len(*mc) != 0 {
		c.MPD.Conf = *mc
	}
	if len(*sa) != 0 {
		c.Server.Addr = *sa
	}
	if *si {
		c.Server.Cover.Remote = true
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
		c.MPD.Addr = "localhost:6600"
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
	defaultTree = map[string]*ConfigListNode{
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

// ConfigListNode represents smart playlist node.
type ConfigListNode struct {
	Sort []string    `json:"sort"`
	Tree [][2]string `json:"tree"`
}

// Validate ConfigListNode data struct.
func (l *ConfigListNode) Validate() error {
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
