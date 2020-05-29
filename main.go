package main

import (
	"bufio"
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/meiraka/vv/internal/mpd"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const staticVersion = "v0.6.2+"

var version = "v0.7.0+"

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

//go:generate go run internal/cmd/fix-assets/main.go
func main() {
	setupFlag("config")
	err := viper.ReadInConfig()
	if err != nil {
		if _, notfound := err.(viper.ConfigFileNotFoundError); !notfound {
			log.Println("[error]", "faied to load config file:", err)
			os.Exit(1)
		}
	}
	v2()
}

func v2() {
	ctx := context.TODO()
	network := viper.GetString("mpd.network")
	addr := viper.GetString("mpd.addr")
	musicDirectory := viper.GetString("mpd.music_directory")
	dialer := mpd.Dialer{
		Timeout:              10 * time.Second,
		HealthCheckInterval:  time.Second,
		ReconnectionInterval: 5 * time.Second,
	}
	cl, err := dialer.Dial(network, addr, "")
	if err != nil {
		log.Fatalf("failed to dial mpd: %v", err)
	}
	w, err := dialer.NewWatcher(network, addr, "")
	if err != nil {
		log.Fatalf("failed to dial mpd: %v", err)
	}
	// get music dir from local mpd connection
	if network == "unix" {
		if c, err := cl.Config(ctx); err == nil {
			if dir, ok := c["music_directory"]; ok {
				musicDirectory = dir
			}
		}
	}

	// get music dir from local mpd config
	if len(musicDirectory) == 0 {
		dir, err := getMusicDirectory("/etc/mpd.conf")
		if err == nil {
			musicDirectory = dir
		}
	}
	assets := AssetsConfig{
		LocalAssets: viper.GetBool("debug"),
	}.NewAssetsHandler()
	api, err := APIConfig{
		MusicDirectory: musicDirectory,
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
		Addr:    viper.GetString("server.addr"),
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
