# Spotify Randomizer

A small Go webapp that picks a random album from your saved Spotify library or
a random movie from a public IMDb watchlist.

It is designed to be easy to run on a laptop or home server: one Go process serves the UI, handles Spotify OAuth, saves local setup, and exposes the random album action.

## Run

```bash
go run .
```

Then open:

```text
http://127.0.0.1:8080
```

On first run, the app shows a setup screen where you can register your Spotify client credentials. After setup, the app saves your client settings and Spotify token in your user config directory under `spotify-randomizer`.

## Create a Spotify Client

1. Open <https://developer.spotify.com/dashboard>.
2. Sign in with the Spotify account that owns the saved albums you want to randomize.
3. Create an app.
4. Open the app settings.
5. Copy the app's Client ID and Client secret.
6. Add this redirect URI:

```text
http://127.0.0.1:8080/callback
```

7. Save the Spotify app settings.
8. Paste the Client ID, Client secret, and redirect URI into the Spotify Randomizer setup screen.

The redirect URI in Spotify must exactly match the redirect URL configured in Spotify Randomizer.

## In-App Setup

The setup screen is available at:

```text
http://127.0.0.1:8080/setup
```

Use it when:

- You are running the app for the first time.
- You want to change the Spotify client ID or secret.
- You changed the redirect URL in the Spotify Developer Dashboard.

Saving setup stores a local config file with the client ID, client secret, redirect URL, and server address. The client secret is stored plainly in your user config directory, which is intended for this local/home-server app.

## Environment Variables

You can still configure the app with `.env` or shell environment variables. Environment variables override the saved setup config.

```bash
SPOTIFY_CLIENT_ID=your_client_id
SPOTIFY_CLIENT_SECRET=your_client_secret
SPOTIFY_REDIRECT_URL=http://127.0.0.1:8080/callback
SPOTIFY_RANDOMIZER_ADDR=:8080
```

This is useful for command-line runs, deployments, and home-server setups where you prefer not to use the web setup form.

## IMDb Watchlists

The home page also includes an IMDb Watchlist Randomizer. Make the watchlist
public in IMDb, paste its URL into the app, and select **Pick random movie**.
The link should look like:

```text
https://www.imdb.com/user/ur12345678/watchlist/
```

The watchlist link is used only for that request and is not saved. No IMDb
account or API key is required. IMDb's newer `/user/p.…/watchlist/` links and
public `/list/ls…/` links are also accepted.

## Home Server

Bind to all network interfaces:

```bash
SPOTIFY_RANDOMIZER_ADDR=0.0.0.0:8080
SPOTIFY_REDIRECT_URL=http://your-server.local:8080/callback
go run .
```

Then add the same callback URL in the Spotify app settings:

```text
http://your-server.local:8080/callback
```

For internet-facing hosting, put the app behind HTTPS with a reverse proxy and use the public HTTPS callback URL in both `SPOTIFY_REDIRECT_URL` and the Spotify app dashboard.
