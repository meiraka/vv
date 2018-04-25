package main

import (
	"bufio"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"log"
	"os"
	"strings"
	"time"
)

const staticVersion = "v0.6.0+"

var version string

func setupFlag(name string) {
	viper.SetConfigName(name)
	viper.AddConfigPath("/etc/xdg/vv")
	viper.AddConfigPath("$HOME/.config/vv")
	pflag.String("mpd.network", "tcp", "mpd server network to connect")
	pflag.String("mpd.addr", "localhost:6600", "mpd server address to connect")
	pflag.String("mpd.music_directory", "", "set music_directory in mpd.conf value to search album cover image")
	pflag.String("server.addr", ":8080", "this app serving address")
	pflag.Bool("server.keepalive", true, "use HTTP keep-alive")
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
	setupFlag("config")
	err := viper.ReadInConfig()
	if err != nil {
		if _, notfound := err.(viper.ConfigFileNotFoundError); !notfound {
			log.Println("[error]", "faied to load config file:", err)
			os.Exit(1)
		}
	}
	musicDirectory := viper.GetString("mpd.music_directory")
	if len(musicDirectory) == 0 {
		dir, err := getMusicDirectory("/etc/mpd.conf")
		if err == nil {
			musicDirectory = dir
		}
	}
	network := viper.GetString("mpd.network")
	addr := viper.GetString("mpd.addr")
	music, err := Dial(network, addr, "", musicDirectory)
	defer music.Close()
	if err != nil {
		log.Println("[error]", "faied to connect/initialize mpd:", err)
		os.Exit(1)
	}
	serverAddr := viper.GetString("server.addr")
	s := Server{
		Music:          music,
		MusicDirectory: musicDirectory,
		Addr:           serverAddr,
		StartTime:      time.Now().UTC(),
		KeepAlive:      viper.GetBool("server.keepalive"),
		debug:          viper.GetBool("debug"),
	}
	s.Serve()
}
