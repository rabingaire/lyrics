build:
	@go build -o lyrics -ldflags "-X main.spotifyClientID=$(SPOTIFY_ID) -X main.spotifyClientSecret=$(SPOTIFY_SECRET) -X main.geniusClientSecret=$(GENIUS_API_KEY)"
	@sudo cp lyrics /usr/local/bin
	@sudo rm lyrics
