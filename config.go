package main

import (
	"github.com/spf13/viper"
)

// ReadConfig returns filled Config struct.
func ReadConfig() (Config, error) {
	viper.SetConfigName("config")
	viper.AddConfigPath("/etc/xdg/vv")
	viper.AddConfigPath("$HOME/.config/vv")
	viper.SetDefault("mpd.host", "localhost")
	viper.SetDefault("mpd.port", "6600")
	viper.SetDefault("server.port", "8080")
	err := viper.ReadInConfig()
	if err != nil {
		return Config{}, err
	}
	return Config{
		ServerConfig{viper.GetString("server.port")},
		MpdConfig{
			viper.GetString("mpd.host"),
			viper.GetString("mpd.port"),
		},
	}, nil
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
