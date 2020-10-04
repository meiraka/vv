package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/meiraka/vv/internal/mpd"
	"github.com/meiraka/vv/internal/songs/cover"
)

const (
	defaultConfigDir = "/etc/xdg/vv"
)

var version = "v0.10.2+"

//go:generate go run internal/cmd/fix-assets/main.go
func main() {
	v2()
}

func configDirs() []string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return []string{defaultConfigDir}
	}
	return []string{filepath.Join(dir, "vv"), defaultConfigDir}
}

func v2() {
	ctx := context.TODO()
	config, date, err := ParseConfig(configDirs(), "config.yaml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	dialer := mpd.Dialer{
		Timeout:              10 * time.Second,
		HealthCheckInterval:  time.Second,
		ReconnectionInterval: 5 * time.Second,
	}
	tree, err := json.Marshal(config.Playlist.Tree)
	if err != nil {
		log.Fatalf("failed to create playlist tree: %v", err)
	}
	treeOrder, err := json.Marshal(config.Playlist.TreeOrder)
	if err != nil {
		log.Fatalf("failed to create playlist tree order: %v", err)
	}
	cl, err := dialer.Dial(config.MPD.Network, config.MPD.Addr, "")
	if err != nil {
		log.Fatalf("failed to dial mpd: %v", err)
	}
	w, err := dialer.NewWatcher(config.MPD.Network, config.MPD.Addr, "")
	if err != nil {
		log.Fatalf("failed to dial mpd: %v", err)
	}
	commands, err := cl.Commands(ctx)
	if err != nil {
		log.Fatalf("failed to check mpd supported functions: %v", err)
	}
	// get music dir from local mpd connection
	if config.MPD.Network == "unix" && config.MPD.MusicDirectory == "" {
		if c, err := cl.Config(ctx); err == nil {
			if dir, ok := c["music_directory"]; ok {
				config.MPD.MusicDirectory = dir
				log.Printf("apply mpd.music_directory from mpd connection: %s", dir)
			}
		}
	}

	// get music dir from local mpd config
	mpdConf, _ := mpd.ParseConfig(config.MPD.Conf)
	if config.MPD.MusicDirectory == "" {
		if mpdConf != nil && len(config.MPD.Conf) != 0 {
			config.MPD.MusicDirectory = mpdConf.MusicDirectory
			log.Printf("apply mpd.music_directory from %s: %s", config.MPD.Conf, mpdConf.MusicDirectory)
		}
	}
	proxy := map[string]string{}
	if mpdConf != nil {
		host := "localhost"
		if config.MPD.Network == "tcp" {
			h := strings.Split(config.MPD.Addr, ":")[0]
			if len(h) != 0 {
				host = h
			}
		}
		for _, dev := range mpdConf.AudioOutputs {
			if len(dev.Port) != 0 {
				proxy[dev.Name] = "http://" + host + ":" + dev.Port
			}
		}
	}
	m := http.NewServeMux()
	covers := make([]cover.Cover, 0, 2)
	if config.Server.Cover.Local {
		if len(config.MPD.MusicDirectory) == 0 {
			log.Println("config.server.cover.local is disabled: mpd.music_directory is empty")
		} else if !strings.HasPrefix(config.MPD.MusicDirectory, "/") {
			log.Printf("config.server.cover.local is disabled: mpd.music_directory is not absolute local directory path: %v", config.MPD.MusicDirectory)
		} else {
			c, err := cover.NewLocal("/api/music/images/local/", config.MPD.MusicDirectory, []string{"cover.jpg", "cover.jpeg", "cover.png", "cover.gif", "cover.bmp"})
			if err != nil {
				log.Fatalf("failed to initialize coverart: %v", err)
			}
			m.Handle("/api/music/images/local/", c)
			covers = append(covers, c)

		}
	}
	if config.Server.Cover.Remote {
		if !contains(commands, "albumart") {
			log.Println("config.server.cover.remote is disabled: mpd does not support albumart command")
		} else {
			c, err := cover.NewRemote("/api/music/images/remote/", cl, filepath.Join(config.Server.CacheDirectory, "imgcache"))
			if err != nil {
				log.Fatalf("failed to initialize coverart: %v", err)
			}
			m.Handle("/api/music/images/remote/", c)
			covers = append(covers, c)
			defer c.Close()
		}
	}
	batch := cover.NewBatch(covers)
	assets := AssetsConfig{
		LocalAssets: config.debug,
		Extra: map[string]string{
			"AssetsAppCSSHash": string(AssetsAppCSSHash),
			"AssetsAppJSHash":  string(AssetsAppJSHash),
			"TREE":             string(tree),
			"TREE_ORDER":       string(treeOrder),
		},
		ExtraDate: date,
	}.NewAssetsHandler()
	api, stopAPI, err := APIConfig{
		AudioProxy: proxy,
	}.NewAPIHandler(ctx, cl, w, batch)
	if err != nil {
		log.Fatalf("failed to initialize api handler: %v", err)
	}
	m.Handle("/", assets)
	m.Handle("/api/", api)

	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}
	s := http.Server{
		Handler: m,
		Addr:    config.Server.Addr,
	}
	s.RegisterOnShutdown(stopAPI)
	errs := make(chan error, 1)
	go func() {
		errs <- s.ListenAndServe()
	}()
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGTERM, syscall.SIGINT)
	select {
	case <-sc:
	case err := <-errs:
		if err != http.ErrServerClosed {
			log.Fatalf("server stopped with error: %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		log.Printf("failed to stop http server: %v", err)
	}
	if err := cl.Close(ctx); err != nil {
		log.Printf("failed to close mpd connection(main): %v", err)
	}
	if err := w.Close(ctx); err != nil {
		log.Printf("failed to close mpd connection(event): %v", err)
	}
	if err := batch.Shutdown(ctx); err != nil {
		log.Printf("failed to stop image api: %v", err)
	}
}

func contains(list []string, item string) bool {
	for _, n := range list {
		if item == n {
			return true
		}
	}
	return false
}
