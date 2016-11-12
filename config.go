package vv

import (
	"github.com/BurntSushi/toml"
)

func defaultConfigPath() string {
	return "/etc/vvrc"
}

func readConfig(path string) (Config, error) {
	var config Config
	_, err := toml.DecodeFile(path, &config)
	return config, err
}

// Config represents server properties.
type Config struct {
	Mpd MpdConfig `toml:"mpd"`
}

// MpdConfig represents local mpd information.
type MpdConfig struct {
	Host string `toml:"host"`
	Port string `toml:"port"`
}
