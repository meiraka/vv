package main

import (
	"bufio"
	"github.com/spf13/viper"
	"os"
	"strings"
)

// ReadConfig returns filled Config struct.
func ReadConfig() (Config, error) {
	viper.SetConfigName("config")
	viper.AddConfigPath("/etc/xdg/vv")
	viper.AddConfigPath("$HOME/.config/vv")
	viper.SetDefault("mpd.host", "localhost")
	viper.SetDefault("mpd.port", "6600")
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("mpd.music_directory", "")
	err := viper.ReadInConfig()
	if err != nil {
		if _, notfound := err.(viper.ConfigFileNotFoundError); !notfound {
			return Config{}, err
		}
	}
	return Config{
		ServerConfig{viper.GetString("server.port")},
		MpdConfig{
			viper.GetString("mpd.host"),
			viper.GetString("mpd.port"),
			viper.GetString("mpd.music_directory"),
		},
	}, nil
}

func getMusicDirectory(confpath string) (string, error) {
	f, err := os.Open(confpath)
	if err != nil {
		return "", err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for i := 1; sc.Scan(); i++ {
		if err := sc.Err(); err != nil {
			return "", err
		}
		l := sc.Text()
		if strings.HasPrefix(l, "music_directory") {
			q := strings.TrimSpace(strings.TrimPrefix(l, "music_directory"))
			if strings.HasPrefix(q, "\"") && strings.HasSuffix(q, "\"") {
				return strings.Trim(q, "\""), nil
			}
		}
	}
	return "", nil
}

// Config represents app properties.
type Config struct {
	Server ServerConfig `toml:"server"`
	Mpd    MpdConfig    `toml:"mpd"`
}

// MpdConfig represents local mpd information.
type MpdConfig struct {
	Host           string `toml:"host"`
	Port           string `toml:"port"`
	MusicDirectory string
}

// ServerConfig represents HTTP server information.
type ServerConfig struct {
	Port string `toml:"port"`
}
