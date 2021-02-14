package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/meiraka/vv/internal/http/vv"
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
		BinaryLimit    int    `yaml:"binarylimit"`
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
	flagset := pflag.NewFlagSet(filepath.Base(os.Args[0]), pflag.ExitOnError)
	mn := flagset.String("mpd.network", "", "mpd server network to connect")
	ma := flagset.String("mpd.addr", "", "mpd server address to connect")
	mm := flagset.String("mpd.music_directory", "", "set music_directory in mpd.conf value to search album cover image")
	mc := flagset.String("mpd.conf", "", "set mpd.conf path to get music_directory and http audio output")
	mb := flagset.Int("mpd.binarylimit", 0, "set the maximum binary response size of mpd")
	sa := flagset.String("server.addr", "", "this app serving address")
	si := flagset.Bool("server.cover.remote", false, "enable coverart via mpd api")
	d := flagset.BoolP("debug", "d", false, "use local assets if exists")
	flagset.Parse(os.Args)
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
	if *mb != 0 {
		c.MPD.BinaryLimit = *mb
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
			return fmt.Errorf("playlist.tree.%s is not defined in playlist.tree", label)
		}
		if err := node.Validate(); err != nil {
			return fmt.Errorf("playlist.tree.%s: %w", label, err)
		}
	}
	for i, label := range c.Playlist.TreeOrder {
		// check playlist sort is uniq
		if i != len(c.Playlist.TreeOrder)-1 {
			for _, compare := range c.Playlist.TreeOrder[i+1:] {
				if sameStrSlice(c.Playlist.Tree[label].Sort, c.Playlist.Tree[compare].Sort) {
					return fmt.Errorf("playlist.tree.*.sort must be unique: playlist.tree.%s.sort and playlist.tree.%s.sort has same value", label, compare)
				}
			}
		}
	}
	if t, o := len(c.Playlist.Tree), len(c.Playlist.TreeOrder); o != t {
		return fmt.Errorf("playlist.tree length (%d) and playlist.tree_order length (%d) mismatch", t, o)
	}
	return nil
}

func sameStrSlice(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i, s := range a {
		if b[i] != s {
			return false
		}
	}
	return true
}

var (
	supportTreeViews = []string{"plain", "album", "song"}
)

// ConfigListNode represents smart playlist node.
type ConfigListNode struct {
	Sort []string    `yaml:"sort"`
	Tree [][2]string `yaml:"tree"`
}

// Validate ConfigListNode data struct.
func (l *ConfigListNode) Validate() error {
	if len(l.Tree) == 0 || len(l.Sort) == 0 {
		return errors.New("sort or tree must not be empty")
	}
	if len(l.Tree) > 4 {
		return fmt.Errorf("maximum tree length is 4; got %d", len(l.Tree))
	}
	for i, leef := range l.Tree {
		if !contains(l.Sort, leef[0]) {
			return fmt.Errorf("tree: index %d:0: tree tag must be defined in sort: %s does not defined in sort: %v", i, leef[0], l.Sort)
		}
		if !contains(supportTreeViews, leef[1]) {
			return fmt.Errorf("tree: index %d:1: unsupported tree view type: got %s; supported tree element views are %v", i, leef[1], supportTreeViews)
		}
	}
	return nil
}

// toTree shallow copies config tree to vv.Tree.
func toTree(t map[string]*ConfigListNode) vv.Tree {
	if t == nil {
		return nil
	}
	ret := make(vv.Tree, len(t))
	for k, v := range t {
		ret[k] = &vv.TreeNode{
			Sort: v.Sort,
			Tree: v.Tree,
		}
	}
	return ret
}
