package mpd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
)

// Config represents MPD config struct
type Config struct {
	MusicDirectory string
	AudioOutputs   []*ConfigAudioOutput
}

type ConfigAudioOutput struct {
	Type string
	Name string
	Port string
}

// ParseConfig parses mpd.conf
func ParseConfig(file string) (*Config, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	ret := &Config{}
	if err := parseConfig(f, ret); err != nil {
		return nil, err
	}
	return ret, nil
}

type state int

const (
	parseKey state = iota
	parseKeyEnd
	parseValue
	parseBraceKey
	parseBraceKeyEnd
	parseBraceValue
)

func parseConfig(r io.Reader, c *Config) error {
	sc := bufio.NewScanner(r)

	brace := map[string]string{}

	key := []rune{}
	braceKey := []rune{}
	value := []rune{}
	var current state
	for i := 1; sc.Scan(); i++ {
		if err := sc.Err(); err != nil {
			return err
		}
		if current == parseKey && len(key) == 0 {
		} else if current == parseBraceKey && len(braceKey) == 0 {
		} else {
			return fmt.Errorf("parse error unknown state: %d", current)
		}

		l := sc.Text()
	parseLine:
		for _, t := range l {
			if t == ' ' || t == '\t' {
				if current == parseKey && len(key) != 0 {
					current = parseKeyEnd
				}
				if current == parseBraceKey && len(braceKey) != 0 {
					current = parseBraceKeyEnd
				} else if current == parseValue || current == parseBraceValue {
					value = append(value, t)
				}
			} else if t == '"' {
				if current == parseKeyEnd {
					current = parseValue
				} else if current == parseBraceKeyEnd {
					current = parseBraceValue
				} else if current == parseValue {
					c.apply(string(key), string(value))
					key = []rune{}
					value = []rune{}
					current = parseKey
				} else if current == parseBraceValue {
					brace[string(braceKey)] = string(value)
					braceKey = []rune{}
					value = []rune{}
					current = parseBraceKey
				} else {
					return errors.New("parse error")
				}
			} else if t == '#' { // TODO: check original parser
				if current != parseKey {
					break parseLine
				} else if current == parseKey && len(key) == 0 {
					break parseLine
				} else {
					return errors.New("parse error")
				}
			} else if t == '{' {
				current = parseBraceKey
				break parseLine
			} else if t == '}' {
				c.applyMap(string(key), brace)
				key = []rune{}
				brace = map[string]string{}
				current = parseKey
			} else {
				if current == parseKey {
					key = append(key, t)
				} else if current == parseBraceKey {
					braceKey = append(braceKey, t)
				} else {
					value = append(value, t)
				}
			}
		}
	}
	return nil
}

func (c *Config) apply(key, value string) {
	if key == "music_directory" {
		c.MusicDirectory = value
	}
}

func (c *Config) applyMap(key string, value map[string]string) {
	if key == "audio_output" {
		a := &ConfigAudioOutput{}
		a.Type = value["type"]
		a.Name = value["name"]
		a.Port = value["port"]
		c.AudioOutputs = append(c.AudioOutputs, a)
	}
}
