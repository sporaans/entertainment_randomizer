package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"

	"github.com/zmb3/spotify/v2"
	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
)

var errReauthorize = errors.New("spotify re-authorization required")

func newSpotifyAuth(settings appSettings) *spotifyauth.Authenticator {
	settings = settings.WithDefaults()
	if !settings.IsConfigured() {
		return nil
	}

	return spotifyauth.New(
		spotifyauth.WithClientID(settings.ClientID),
		spotifyauth.WithClientSecret(settings.ClientSecret),
		spotifyauth.WithRedirectURL(settings.RedirectURL),
		spotifyauth.WithScopes(
			spotifyauth.ScopeUserLibraryRead,
		),
	)
}

func initialToken(ctx context.Context, settings appSettings, tokenPath string) (*oauth2.Token, error) {
	auth := newSpotifyAuth(settings)
	if auth == nil {
		return nil, nil
	}

	token, err := loadToken(tokenPath)
	if err != nil {
		return nil, err
	}

	if token != nil && !token.Valid() {
		refreshedToken, refreshErr := auth.RefreshToken(ctx, token)
		if refreshErr == nil {
			token = refreshedToken
			if err := saveToken(tokenPath, token); err != nil {
				return nil, err
			}
		} else {
			token = nil
		}
	}

	return token, nil
}

func (a *app) spotifyClient(ctx context.Context) (*spotify.Client, error) {
	auth := a.currentAuth()
	if auth == nil {
		return nil, errReauthorize
	}

	a.tokenMu.RLock()
	token := a.token
	a.tokenMu.RUnlock()

	if token == nil {
		return nil, errReauthorize
	}

	if !token.Valid() {
		refreshedToken, err := auth.RefreshToken(ctx, token)
		if err != nil {
			_ = os.Remove(a.paths.tokenPath)
			a.setToken(nil)
			return nil, errReauthorize
		}
		if err := saveToken(a.paths.tokenPath, refreshedToken); err != nil {
			return nil, err
		}
		a.setToken(refreshedToken)
		token = refreshedToken
	}

	return spotify.New(auth.Client(ctx, token)), nil
}

func randomSavedAlbum(ctx context.Context, client *spotify.Client) (*spotify.SimpleAlbum, int, error) {
	limit := 50
	offset := 0
	var albums []spotify.SavedAlbum

	for {
		page, err := client.CurrentUsersAlbums(ctx, spotify.Limit(limit), spotify.Offset(offset))
		if err != nil {
			return nil, 0, err
		}

		albums = append(albums, page.Albums...)
		offset += len(page.Albums)

		if len(page.Albums) == 0 || offset >= int(page.Total) {
			break
		}
	}

	if len(albums) == 0 {
		return nil, 0, errors.New("no saved albums found in your Spotify library")
	}

	index, err := cryptoRandomIndex(len(albums))
	if err != nil {
		return nil, 0, err
	}

	return &albums[index].SimpleAlbum, len(albums), nil
}

func cryptoRandomIndex(length int) (int, error) {
	if length <= 0 {
		return 0, errors.New("length must be positive")
	}

	n, err := rand.Int(rand.Reader, big.NewInt(int64(length)))
	if err != nil {
		return 0, fmt.Errorf("generate random number: %w", err)
	}

	return int(n.Int64()), nil
}

func randomState() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1<<62))
	if err != nil {
		return "", fmt.Errorf("generate oauth state: %w", err)
	}

	return fmt.Sprintf("%x", n.Uint64()), nil
}

func joinArtistNames(artists []spotify.SimpleArtist) string {
	names := make([]string, 0, len(artists))
	for _, artist := range artists {
		names = append(names, artist.Name)
	}

	return strings.Join(names, ", ")
}

func handleSpotifyError(tokenPath string, err error) error {
	var retrieveErr spotify.Error
	if errors.As(err, &retrieveErr) {
		if retrieveErr.Status == http.StatusUnauthorized {
			if removeErr := os.Remove(tokenPath); removeErr == nil || errors.Is(removeErr, os.ErrNotExist) {
				return errReauthorize
			}
		}

		if retrieveErr.Status == http.StatusForbidden {
			removeErr := os.Remove(tokenPath)
			if removeErr == nil || errors.Is(removeErr, os.ErrNotExist) {
				return errReauthorize
			}

			return fmt.Errorf("clear token cache after spotify denied access: %w", removeErr)
		}
	}

	return err
}

func loadToken(path string) (*oauth2.Token, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read token cache: %w", err)
	}

	var token oauth2.Token
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("parse token cache: %w", err)
	}

	return &token, nil
}

func saveToken(path string, token *oauth2.Token) error {
	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return fmt.Errorf("encode token cache: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write token cache: %w", err)
	}

	return nil
}
