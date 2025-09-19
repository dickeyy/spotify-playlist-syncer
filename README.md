# Spotify Playlist Syncer

A Go program that monitors multiple Spotify playlists and automatically adds new songs to a master playlist.

## What it does

I have playlists organized by different "feelings" or vibes, but sometimes I just want to listen to everything. This program monitors my sub-playlists and whenever a song gets added to any of them, it automatically adds that song to my master playlist that contains everything.

## Setup

### 1. Spotify App

Create a Spotify app at [Spotify Developer Dashboard](https://developer.spotify.com/dashboard):

1. Click "Create an App"
2. Give it a name and description
3. **Important**: Set the Redirect URI to `http://localhost:8080/callback`
4. Copy the Client ID and Client Secret

### 2. Environment Variables

Set your Spotify credentials:

```bash
export SPOTIFY_CLIENT_ID="your_client_id_here"
export SPOTIFY_CLIENT_SECRET="your_client_secret_here"
```

### 3. Playlist Files

Create two files in the `data/` directory as the program:

#### `data/subplaylists.txt`

Contains the URLs of playlists you want to monitor, one per line:

```
https://open.spotify.com/playlist/37i9dQZF1DX0XUsuxWHRQd
https://open.spotify.com/playlist/37i9dQZF1DX4JAvHpjipBk
https://open.spotify.com/playlist/37i9dQZF1DWZd79rJ6a7lp
```

#### `data/masterplaylist.txt`

Contains exactly one URL - your master playlist:

```
https://open.spotify.com/playlist/1Bxi8MJuX2PgkLcYDwcUGg
```

### 4. Permissions

The app will request the following permissions when you authenticate:

- **Read your private playlists** - to monitor your sub-playlists
- **Modify your private playlists** - to add songs to your master playlist
- **Modify your public playlists** - in case your master playlist is public

### 5. Build and Run

```bash
go build -o syncer
./syncer              # production mode (json logs)
./syncer --dev        # development mode (pretty logs)
```

## First Run

When you start the program for the first time:

1. The program will display an authentication URL
2. Visit the URL in your web browser
3. Sign in to your Spotify account (if not already signed in)
4. Click "Agree" to grant the requested permissions
5. The browser will redirect you to a success page
6. Return to the terminal - the program will initialize by fetching all existing tracks
7. The program will begin monitoring for new additions

## How it works

- **Authentication**: Uses OAuth2 Authorization Code flow with user consent
- **Initialization**: On startup, fetches all existing tracks from master and sub-playlists to establish baseline
- **Monitoring**: The program scans all your sub-playlists every 5 minutes (or 30 seconds in dev mode)
- **Syncing**: When it finds new songs in any sub-playlist, it adds them to your master playlist
- **Deduplication**: Remembers all tracks seen during initialization and runtime to avoid duplicates
- **Continuous**: Runs continuously until you stop it with Ctrl+C

## Configuration

### Check Interval

The default check interval is 5 minutes. In development mode (`--dev`), it's 30 seconds for faster testing. You can modify this in `config/config.go` if needed.

### Logging

- **Production**: JSON format logs (default)
- **Development**: Pretty, human-readable logs with `--dev` flag

## Project Structure

```
spotify-playlist-syncer/
├── main.go              # entry point and CLI
├── config/              # configuration loading
├── spotify/             # spotify api client
├── monitor/             # monitoring logic
├── logging/             # logging setup
├── go.mod
└── README.md
```

## Requirements

- Go 1.19+
- Spotify Premium account (for playlist modification)
- Spotify Developer App credentials

## Notes

- The program uses Spotify's Web API with OAuth2 Authorization Code flow for user authentication
- It only adds songs to playlists, never removes them
- Make sure your Spotify app has the redirect URI set to `http://localhost:8080/callback`
- The master playlist should be writable by your account
- The program requires an internet connection for authentication and API calls
