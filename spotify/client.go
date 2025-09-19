package spotify

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
)

// Client wraps the Spotify client
type Client struct {
	client *spotify.Client
}

// generateRandomState generates a random state string for OAuth2 security
func generateRandomState() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

// NewClient creates a new Spotify client with user authentication
func NewClient(clientID, clientSecret string) (*Client, error) {
	callbackURL := os.Getenv("CALLBACK_URL")
	if callbackURL == "" {
		callbackURL = "http://localhost:8080/callback"
	}
	// create OAuth2 config
	auth := spotifyauth.New(
		spotifyauth.WithClientID(clientID),
		spotifyauth.WithClientSecret(clientSecret),
		spotifyauth.WithRedirectURL(callbackURL),
		spotifyauth.WithScopes(spotifyauth.ScopePlaylistReadPrivate, spotifyauth.ScopePlaylistModifyPrivate, spotifyauth.ScopePlaylistModifyPublic))

	state, err := generateRandomState()
	if err != nil {
		return nil, fmt.Errorf("generating state: %w", err)
	}

	// create a channel to receive the authenticated client
	clientChan := make(chan *spotify.Client, 1)
	errorChan := make(chan error, 1)

	// start local server to handle callback
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		token, err := auth.Token(r.Context(), state, r)
		if err != nil {
			errorChan <- fmt.Errorf("getting token: %w", err)
			http.Error(w, "Authentication failed", http.StatusInternalServerError)
			return
		}

		// create authenticated client
		httpClient := spotifyauth.New().Client(r.Context(), token)
		client := spotify.New(httpClient)
		clientChan <- client

		// show success page
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `
			<html>
			<body>
			<h1>Authentication Successful!</h1>
			<p>You can now close this window and return to the application.</p>
			<script>window.close();</script>
			</body>
			</html>
		`)
	})

	// start server in background
	server := &http.Server{Addr: ":8080"}
	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			errorChan <- fmt.Errorf("server error: %w", err)
		}
	}()

	// get the authorization URL
	authURL := auth.AuthURL(state)
	fmt.Printf("\n🔗 Please visit this URL to authenticate:\n%s\n\n", authURL)

	// wait for either client or error
	select {
	case client := <-clientChan:
		// stop server
		server.Close()
		return &Client{client: client}, nil
	case err := <-errorChan:
		server.Close()
		return nil, err
	}
}

// ExtractPlaylistID extracts playlist ID from Spotify URL
func ExtractPlaylistID(playlistURL string) (spotify.ID, error) {
	u, err := url.Parse(playlistURL)
	if err != nil {
		return "", fmt.Errorf("parsing URL: %w", err)
	}

	pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(pathParts) < 2 || pathParts[0] != "playlist" {
		return "", fmt.Errorf("invalid playlist URL format")
	}

	return spotify.ID(pathParts[1]), nil
}

// GetPlaylistTracks gets all tracks from a playlist
func (c *Client) GetPlaylistTracks(ctx context.Context, playlistID spotify.ID) ([]spotify.PlaylistItem, error) {
	var allTracks []spotify.PlaylistItem

	// get first page
	tracks, err := c.client.GetPlaylistItems(ctx, playlistID)
	if err != nil {
		return nil, fmt.Errorf("getting playlist items: %w", err)
	}

	allTracks = append(allTracks, tracks.Items...)

	// get remaining pages
	for tracks.Next != "" {
		err = c.client.NextPage(ctx, tracks)
		if err != nil {
			return nil, fmt.Errorf("getting next page: %w", err)
		}
		allTracks = append(allTracks, tracks.Items...)
	}

	return allTracks, nil
}

// AddTracksToPlaylist adds tracks to a playlist
func (c *Client) AddTracksToPlaylist(ctx context.Context, playlistID spotify.ID, trackIDs []spotify.ID) error {
	// spotify api allows max 100 tracks per request
	const batchSize = 100

	for i := 0; i < len(trackIDs); i += batchSize {
		end := i + batchSize
		if end > len(trackIDs) {
			end = len(trackIDs)
		}

		batch := trackIDs[i:end]
		_, err := c.client.AddTracksToPlaylist(ctx, playlistID, batch...)
		if err != nil {
			return fmt.Errorf("adding tracks batch %d-%d: %w", i, end-1, err)
		}
	}

	return nil
}

// Track represents a simplified track structure
type Track struct {
	ID   spotify.ID
	Name string
}

// GetTrackIDs extracts track IDs from playlist items
func GetTrackIDs(items []spotify.PlaylistItem) []spotify.ID {
	var ids []spotify.ID
	for _, item := range items {
		if item.Track.Track != nil {
			ids = append(ids, item.Track.Track.ID)
		}
	}
	return ids
}
