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
	addr := config.Mpd.Host + ":" + config.Mpd.Port
	player, err := Dial("tcp", addr, "", config.Mpd.MusicDirectory, 1000)
	defer player.Close()
	if err != nil {
		fmt.Printf("faied to connect/initialize mpd: %s\n", err)
		os.Exit(1)
	}
	App(player, config)
}
