package main

import (
	"bufio"
	"fmt"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"os"
	"strings"
	"time"
)

const staticVersion = "v0.1.1+"

var version string

func setupFlag() {
	viper.SetConfigName("config")
	viper.AddConfigPath("/etc/xdg/vv")
	viper.AddConfigPath("$HOME/.config/vv")
	pflag.String("mpd.host", "localhost", "mpd server hostname to connect")
	pflag.String("mpd.port", "6600", "mpd server TCP port to connect")
	pflag.String("mpd.music_directory", "", "set music_directory in mpd.conf value to search album cover image")
	pflag.String("server.port", "8080", "this app serving TCP port")
	pflag.BoolP("debug", "d", false, "use local assets if exists")
	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)
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
	setupFlag()
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
	s := Server{
		Music:          player,
		MusicDirectory: musicDirectory,
		Port:           viper.GetString("server.port"),
		StartTime:      time.Now().UTC(),
		debug:          viper.GetBool("debug"),
	}
	s.Serve()
}
