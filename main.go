package main

import (
	"bufio"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/meiraka/vv/internal/mpd"
)

const staticVersion = "v0.6.2+"

var version = "v0.7.0+"

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
		l = strings.TrimSpace(l)
		if strings.HasPrefix(l, "music_directory") {
			q := strings.TrimSpace(strings.TrimPrefix(l, "music_directory"))
			if strings.HasPrefix(q, "\"") && strings.HasSuffix(q, "\"") {
				return strings.Trim(q, "\""), nil
			}
		}
	}
	return "", nil
}

//go:generate go run internal/cmd/fix-assets/main.go
func main() {
	v2()
}

func v2() {
	ctx := context.TODO()
	config, date, err := ParseConfig([]string{"/etc/xdg/vv"})
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
	// get music dir from local mpd connection
	if config.MPD.Network == "unix" && config.MPD.MusicDirectory == "" {
		if c, err := cl.Config(ctx); err == nil {
			if dir, ok := c["music_directory"]; ok {
				config.MPD.MusicDirectory = dir
			}
		}
	}

	// get music dir from local mpd config
	if config.MPD.MusicDirectory == "" {
		dir, err := getMusicDirectory("/etc/mpd.conf")
		if err == nil {
			config.MPD.MusicDirectory = dir
		}
	}
	if !strings.HasPrefix(config.MPD.MusicDirectory, "/") {
		config.MPD.MusicDirectory = ""
	}
	assets := AssetsConfig{
		LocalAssets: config.debug,
		Extra:       map[string]string{"TREE": string(tree), "TREE_ORDER": string(treeOrder)},
		ExtraDate:   date,
	}.NewAssetsHandler()
	api, err := APIConfig{
		MusicDirectory: config.MPD.MusicDirectory,
	}.NewAPIHandler(ctx, cl, w)
	if err != nil {
		log.Fatalf("failed to initialize api handler: %v", err)
	}
	m := http.NewServeMux()
	m.Handle("/", assets)
	m.Handle("/api/", api)

	if err != nil {
		log.Fatalf("failed to initialize app: %v", err)
	}
	s := http.Server{
		Handler: m,
		Addr:    config.Server.Addr,
	}
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
}
