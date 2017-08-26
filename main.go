package main

import (
	"bufio"
	"fmt"
	"github.com/spf13/viper"
	"os"
	"strings"
)

const staticVersion = "v0.0.4+"

var version string

func initConfig() {
	viper.SetConfigName("config")
	viper.AddConfigPath("/etc/xdg/vv")
	viper.AddConfigPath("$HOME/.config/vv")
	viper.SetDefault("mpd.host", "localhost")
	viper.SetDefault("mpd.port", "6600")
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("mpd.music_directory", "")
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

//go:generate go-bindata assets
func main() {
	initConfig()
	err := viper.ReadInConfig()
	if err != nil {
		if _, notfound := err.(viper.ConfigFileNotFoundError); !notfound {
			fmt.Printf("faied to load config file: %s\n", err)
			os.Exit(1)
		}
	}
	musicDirectory := viper.GetString("mpd.music_directory")
	if len(musicDirectory) == 0 && viper.GetString("mpd.host") == "localhost" {
		dir, err := getMusicDirectory("/etc/mpd.conf")
		if err == nil {
			musicDirectory = dir
		}
	}
	addr := viper.GetString("mpd.host") + ":" + viper.GetString("mpd.port")
	player, err := Dial("tcp", addr, "", musicDirectory)
	defer player.Close()
	if err != nil {
		fmt.Printf("faied to connect/initialize mpd: %s\n", err)
		os.Exit(1)
	}
	Serve(player, musicDirectory, viper.GetString("server.port"))
}
