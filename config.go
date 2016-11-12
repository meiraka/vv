package main

import (
	"github.com/BurntSushi/toml"
)

// DefaultConfigPath returns this app default config path.
func DefaultConfigPath() string {
	return "/etc/vvrc"
}

// ReadConfig returns filled Config struct.
func ReadConfig(path string) (Config, error) {
	var config Config
	_, err := toml.DecodeFile(path, &config)
	return config, err
}

// Config represents app properties.
type Config struct {
	Server ServerConfig `toml:"server"`
	Mpd    MpdConfig    `toml:"mpd"`
}

// MpdConfig represents local mpd information.
type MpdConfig struct {
	Host string `toml:"host"`
	Port string `toml:"port"`
}

// ServerConfig represents HTTP server information.
type ServerConfig struct {
	Port string `toml:"port"`
}
