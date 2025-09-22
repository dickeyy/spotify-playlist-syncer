package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

// Config holds the application configuration
type Config struct {
	SubPlaylistURLs   []string
	MasterPlaylistURL string
	CheckInterval     time.Duration
	DevMode           bool
}

// Load reads configuration from files and environment
func Load(dev bool) (*Config, error) {
	interval := 15 * time.Minute // default 15 minutes
	if dev {
		interval = 10 * time.Second
	}
	config := &Config{
		CheckInterval: interval, // default 15 minutes
		DevMode:       dev,
	}

	// read sub playlists
	subURLs, err := readPlaylistFile("data/subplaylists.txt")
	if err != nil {
		return nil, fmt.Errorf("reading subplaylists.txt: %w", err)
	}
	config.SubPlaylistURLs = subURLs

	// read master playlist
	masterURLs, err := readPlaylistFile("data/masterplaylist.txt")
	if err != nil {
		return nil, fmt.Errorf("reading masterplaylist.txt: %w", err)
	}
	if len(masterURLs) != 1 {
		return nil, fmt.Errorf("masterplaylist.txt must contain exactly one playlist URL")
	}
	config.MasterPlaylistURL = masterURLs[0]

	return config, nil
}

// readPlaylistFile reads playlist URLs from a file, one per line
func readPlaylistFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var urls []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			urls = append(urls, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return urls, nil
}
