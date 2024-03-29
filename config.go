package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/meiraka/vv/internal/vv"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
)

// Config is vv application config struct.
type Config struct {
	MPD struct {
		Network        string     `yaml:"network"`
		Addr           string     `yaml:"addr"`
		MusicDirectory string     `yaml:"music_directory"`
		Conf           string     `yaml:"conf"`
		BinaryLimit    BinarySize `yaml:"binarylimit"`
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

func DefaultConfig() *Config {
	c := &Config{}
	c.MPD.Conf = "/etc/mpd.conf"
	c.Server.Addr = ":8080"
	c.Server.CacheDirectory = filepath.Join(os.TempDir(), "vv")
	c.Server.Cover.Local = true
	return c
}

func fillConfig(c *Config) {
	if c.MPD.Network == "" {
		if strings.HasPrefix(c.MPD.Addr, "/") || strings.HasPrefix(c.MPD.Addr, "@") {
			c.MPD.Network = "unix"
		} else {
			c.MPD.Network = "tcp"
		}
	}
	if c.MPD.Addr == "" {
		switch c.MPD.Network {
		case "tcp", "tcp4", "tcp6":
			c.MPD.Addr = "localhost:6600"
		case "unix":
			c.MPD.Addr = "/var/run/mpd/socket"
		}
	}
}

// ParseConfig parse yaml config and flags.
func ParseConfig(dir []string, name string, args []string) (*Config, time.Time, error) {
	c := DefaultConfig()
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
	flagset := pflag.NewFlagSet(filepath.Base(args[0]), pflag.ExitOnError)
	mn := flagset.String("mpd.network", "", "mpd server network to connect")
	ma := flagset.String("mpd.addr", "", "mpd server address to connect")
	mm := flagset.String("mpd.music_directory", "", "set music_directory in mpd.conf value to search album cover image")
	mc := flagset.String("mpd.conf", "", "set mpd.conf path to get music_directory and http audio output")
	mb := flagset.String("mpd.binarylimit", "", "set the maximum binary response size of mpd")
	sa := flagset.String("server.addr", "", "this app serving address")
	si := flagset.Bool("server.cover.remote", false, "enable coverart via mpd api")
	d := flagset.BoolP("debug", "d", false, "use local assets if exists")
	flagset.Parse(args)
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
	if len(*mb) != 0 {
		var bl BinarySize
		if err := bl.UnmarshalText([]byte(*mb)); err != nil {
			return nil, date, err
		}
		c.MPD.BinaryLimit = bl
	}
	if len(*sa) != 0 {
		c.Server.Addr = *sa
	}
	if *si {
		c.Server.Cover.Remote = true
	}
	c.debug = *d
	fillConfig(c)
	return c, date, nil
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

// BinarySize represents a number of binary size.
type BinarySize uint64

// MarshalText returns number as IEC prefixed binary size.
func (b BinarySize) MarshalText() ([]byte, error) {
	buf := &bytes.Buffer{}
	if b%1024 != 0 {
		fmt.Fprintf(buf, "%d B", b)
		return buf.Bytes(), nil
	}
	k := b / 1024
	if k%1024 != 0 {
		fmt.Fprintf(buf, "%d KiB", k)
		return buf.Bytes(), nil
	}
	m := k / 1024
	fmt.Fprintf(buf, "%d MiB", m)
	return buf.Bytes(), nil
}

// UnmarshalText parses binary size with suffix like KiB, MB, m.
func (b *BinarySize) UnmarshalText(text []byte) error {
	text = bytes.TrimSpace(text)
	var num []byte
	var suffixB []byte
	f := true
	for _, l := range text {
		if f && '0' <= l && l <= '9' {
			num = append(num, l)
		} else {
			f = false
			suffixB = append(suffixB, l)
		}
	}
	n, err := strconv.ParseUint(string(num), 10, 64)
	if err != nil {
		return err
	}
	suffix := strings.TrimSpace(string(suffixB))
	if suffix == "k" || suffix == "K" || suffix == "KiB" || suffix == "KB" || suffix == "kB" {
		*b = BinarySize(n) * 1024
	} else if suffix == "m" || suffix == "M" || suffix == "MiB" || suffix == "MB" {
		*b = BinarySize(n) * 1024 * 1024
	} else if suffix == "" || suffix == "B" {
		*b = BinarySize(n)
	} else {
		return fmt.Errorf("unknown size suffix: %s", suffix)
	}
	return nil
}

func contains(list []string, item string) bool {
	for _, n := range list {
		if item == n {
			return true
		}
	}
	return false
}
