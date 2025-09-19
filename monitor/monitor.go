package monitor

import (
	"context"
	"fmt"
	"time"

	"spotify-playlist-syncer/config"
	"spotify-playlist-syncer/logging"
	"spotify-playlist-syncer/spotify"

	spotifyapi "github.com/zmb3/spotify/v2"
)

// Monitor handles playlist monitoring
type Monitor struct {
	config      *config.Config
	spotify     *spotify.Client
	masterID    spotifyapi.ID
	subIDs      []spotifyapi.ID
	knownTracks map[spotifyapi.ID]map[spotifyapi.ID]bool // playlistID -> trackID -> exists
}

// NewMonitor creates a new monitor instance
func NewMonitor(cfg *config.Config, client *spotify.Client) (*Monitor, error) {
	masterID, err := spotify.ExtractPlaylistID(cfg.MasterPlaylistURL)
	if err != nil {
		return nil, fmt.Errorf("extracting master playlist ID: %w", err)
	}

	var subIDs []spotifyapi.ID
	for _, url := range cfg.SubPlaylistURLs {
		id, err := spotify.ExtractPlaylistID(url)
		if err != nil {
			return nil, fmt.Errorf("extracting sub playlist ID from %s: %w", url, err)
		}
		subIDs = append(subIDs, id)
	}

	return &Monitor{
		config:      cfg,
		spotify:     client,
		masterID:    masterID,
		subIDs:      subIDs,
		knownTracks: make(map[spotifyapi.ID]map[spotifyapi.ID]bool),
	}, nil
}

// Initialize prefetches all existing tracks from master and sub playlists
func (m *Monitor) Initialize(ctx context.Context) error {
	logging.Info("initializing monitor by prefetching existing tracks")

	// Get all tracks from master playlist and mark them as known
	masterItems, err := m.spotify.GetPlaylistTracks(ctx, m.masterID)
	if err != nil {
		return fmt.Errorf("getting master playlist tracks: %w", err)
	}

	masterTrackIDs := spotify.GetTrackIDs(masterItems)
	logging.Info("found tracks in master playlist", "count", len(masterTrackIDs))

	// For each sub-playlist, get all tracks and mark them as known
	for _, subID := range m.subIDs {
		subItems, err := m.spotify.GetPlaylistTracks(ctx, subID)
		if err != nil {
			logging.Warn("failed to get tracks from sub-playlist, skipping", "playlistID", subID, "error", err)
			continue
		}

		subTrackIDs := spotify.GetTrackIDs(subItems)
		knownTracks := make(map[spotifyapi.ID]bool)

		// Mark all tracks from this sub-playlist as known
		for _, trackID := range subTrackIDs {
			knownTracks[trackID] = true
		}

		// Also mark tracks that are already in master as known
		// This prevents re-adding tracks that were added in previous runs
		for _, trackID := range masterTrackIDs {
			knownTracks[trackID] = true
		}

		m.knownTracks[subID] = knownTracks
		logging.Info("initialized sub-playlist", "playlistID", subID, "tracks", len(subTrackIDs))
	}

	logging.Info("monitor initialization complete")
	return nil
}

// Start begins the monitoring loop
func (m *Monitor) Start(ctx context.Context) error {
	logging.Info("starting playlist monitor", "interval", m.config.CheckInterval)

	// initialize by prefetching existing tracks
	if err := m.Initialize(ctx); err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}

	ticker := time.NewTicker(m.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logging.Info("monitor stopped")
			return nil
		case <-ticker.C:
			if err := m.scanAllPlaylists(ctx); err != nil {
				logging.Error("scan failed", err)
				// continue monitoring despite errors
			}
		}
	}
}

// scanAllPlaylists checks all sub playlists for new tracks
func (m *Monitor) scanAllPlaylists(ctx context.Context) error {
	logging.Debug("scanning all playlists")

	for _, playlistID := range m.subIDs {
		if err := m.scanPlaylist(ctx, playlistID); err != nil {
			logging.Error("scanning playlist failed", err, "playlistID", playlistID)
			continue
		}
	}

	return nil
}

// scanPlaylist checks a single playlist for new tracks
func (m *Monitor) scanPlaylist(ctx context.Context, playlistID spotifyapi.ID) error {
	items, err := m.spotify.GetPlaylistTracks(ctx, playlistID)
	if err != nil {
		return fmt.Errorf("getting tracks: %w", err)
	}

	currentTracks := spotify.GetTrackIDs(items)
	knownTracks := m.knownTracks[playlistID]

	// knownTracks should already be initialized by Initialize()
	if knownTracks == nil {
		return fmt.Errorf("knownTracks not initialized for playlist %s", playlistID)
	}

	// find new tracks
	var newTracks []spotifyapi.ID
	for _, trackID := range currentTracks {
		if !knownTracks[trackID] {
			newTracks = append(newTracks, trackID)
			knownTracks[trackID] = true
		}
	}

	// add new tracks to master playlist
	if len(newTracks) > 0 {
		logging.Info("found new tracks", "playlistID", playlistID, "count", len(newTracks))

		if err := m.spotify.AddTracksToPlaylist(ctx, m.masterID, newTracks); err != nil {
			return fmt.Errorf("adding tracks to master: %w", err)
		}

		logging.Info("added tracks to master playlist", "count", len(newTracks))
	} else {
		logging.Debug("no new tracks found", "playlistID", playlistID)
	}

	return nil
}
