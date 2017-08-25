package main

import (
	"fmt"
	"os"
)

//go:generate go-bindata assets
func main() {
	config, err := ReadConfig()
	if err != nil {
		fmt.Printf("faied to load config file: %s\n", err)
		os.Exit(1)
	}
	if len(config.Mpd.MusicDirectory) == 0 && config.Mpd.Host == "localhost" {
		dir, err := getMusicDirectory("/etc/mpd.conf")
		if err == nil {
			config.Mpd.MusicDirectory = dir
		}
	}
	addr := config.Mpd.Host + ":" + config.Mpd.Port
	player, err := Dial("tcp", addr, "", config.Mpd.MusicDirectory)
	defer player.Close()
	if err != nil {
		fmt.Printf("faied to connect/initialize mpd: %s\n", err)
		os.Exit(1)
	}
	App(player, config)
}
