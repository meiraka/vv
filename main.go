package main

import (
	"fmt"
	"os"
)

func main() {
	config, err := ReadConfig("./vvrc")
	if err != nil {
		fmt.Printf("faied to load config file: %s", err)
		os.Exit(1)
	}
	App(config.Server)
}
