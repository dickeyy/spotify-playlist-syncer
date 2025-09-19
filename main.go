package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"spotify-playlist-syncer/config"
	"spotify-playlist-syncer/logging"
	"spotify-playlist-syncer/monitor"
	"spotify-playlist-syncer/spotify"

	"github.com/joho/godotenv"
)

func init() {
	if err := godotenv.Load(".env"); err != nil {
		panic(err)
	}
}

func main() {
	var dev = flag.Bool("dev", false, "enable development mode with pretty logging")
	flag.Parse()

	// setup logging first
	logging.Setup(*dev)

	logging.Info("starting spotify playlist syncer")

	// load configuration
	cfg, err := config.Load(*dev)
	if err != nil {
		logging.Error("failed to load config", err)
		os.Exit(1)
	}

	logging.Info("loaded configuration",
		"sub_playlists", len(cfg.SubPlaylistURLs),
		"master_playlist", cfg.MasterPlaylistURL,
		"check_interval", cfg.CheckInterval)

	// get spotify credentials from environment
	clientID := os.Getenv("SPOTIFY_CLIENT_ID")
	clientSecret := os.Getenv("SPOTIFY_CLIENT_SECRET")

	logging.Debug("spotify credentials", "client_id", clientID, "client_secret", fmt.Sprintf("%s********", clientSecret[:4]))

	if clientID == "" || clientSecret == "" {
		logging.Error("missing spotify credentials", fmt.Errorf("SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET must be set"))
		os.Exit(1)
	}

	// create spotify client
	client, err := spotify.NewClient(clientID, clientSecret)
	if err != nil {
		logging.Error("failed to create spotify client", err)
		os.Exit(1)
	}

	// create monitor
	mon, err := monitor.NewMonitor(cfg, client)
	if err != nil {
		logging.Error("failed to create monitor", err)
		os.Exit(1)
	}

	// setup signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		logging.Info("received shutdown signal")
		cancel()
	}()

	// start monitoring
	if err := mon.Start(ctx); err != nil {
		logging.Error("monitor failed", err)
		os.Exit(1)
	}

	logging.Info("spotify playlist syncer stopped")
}
