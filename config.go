package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultRedirectURL = "http://127.0.0.1:8080/callback"
	defaultServerAddr  = ":8080"
	configFileName     = "spotify_randomizer_config.json"
	tokenFileName      = ".spotify_randomizer_token.json"
)

type appPaths struct {
	configDir  string
	configPath string
	tokenPath  string
}

type appSettings struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	RedirectURL  string `json:"redirectUrl"`
	ServerAddr   string `json:"serverAddr"`
}

func (s appSettings) IsConfigured() bool {
	return strings.TrimSpace(s.ClientID) != "" && strings.TrimSpace(s.ClientSecret) != ""
}

func (s appSettings) WithDefaults() appSettings {
	s.ClientID = strings.TrimSpace(s.ClientID)
	s.ClientSecret = strings.TrimSpace(s.ClientSecret)
	s.RedirectURL = strings.TrimSpace(s.RedirectURL)
	s.ServerAddr = strings.TrimSpace(s.ServerAddr)
	if s.RedirectURL == "" {
		s.RedirectURL = defaultRedirectURL
	}
	if s.ServerAddr == "" {
		s.ServerAddr = defaultServerAddr
	}
	return s
}

func appFilePaths() (appPaths, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return appPaths{}, fmt.Errorf("resolve user config dir: %w", err)
	}

	dir := filepath.Join(configDir, "spotify-randomizer")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return appPaths{}, fmt.Errorf("create config dir: %w", err)
	}

	return appPaths{
		configDir:  dir,
		configPath: filepath.Join(dir, configFileName),
		tokenPath:  filepath.Join(dir, tokenFileName),
	}, nil
}

func loadAppSettings(path string) (appSettings, error) {
	settings, err := loadStoredSettings(path)
	if err != nil {
		return appSettings{}, err
	}

	settings = applyEnvSettings(settings)
	return settings.WithDefaults(), nil
}

func loadStoredSettings(path string) (appSettings, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return appSettings{}, nil
	}
	if err != nil {
		return appSettings{}, fmt.Errorf("read app config: %w", err)
	}

	var settings appSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return appSettings{}, fmt.Errorf("parse app config: %w", err)
	}

	return settings, nil
}

func applyEnvSettings(settings appSettings) appSettings {
	if value := strings.TrimSpace(os.Getenv("SPOTIFY_CLIENT_ID")); value != "" {
		settings.ClientID = value
	}
	if value := strings.TrimSpace(os.Getenv("SPOTIFY_CLIENT_SECRET")); value != "" {
		settings.ClientSecret = value
	}
	if value := strings.TrimSpace(os.Getenv("SPOTIFY_REDIRECT_URL")); value != "" {
		settings.RedirectURL = value
	}
	if value := strings.TrimSpace(os.Getenv("SPOTIFY_RANDOMIZER_ADDR")); value != "" {
		settings.ServerAddr = value
	}
	return settings
}

func saveStoredSettings(path string, settings appSettings) error {
	settings = settings.WithDefaults()
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("encode app config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write app config: %w", err)
	}

	return nil
}

func validateSetupSettings(settings appSettings) error {
	settings = settings.WithDefaults()
	if settings.ClientID == "" {
		return errors.New("client ID is required")
	}
	if settings.ClientSecret == "" {
		return errors.New("client secret is required")
	}

	parsed, err := url.Parse(settings.RedirectURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return errors.New("redirect URL must be a full URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("redirect URL must start with http:// or https://")
	}
	if parsed.Path != "/callback" {
		return errors.New("redirect URL path must be /callback")
	}

	return nil
}

func displayServerURL(addr string) string {
	if strings.HasPrefix(addr, ":") {
		return "http://127.0.0.1" + addr
	}
	return "http://" + addr
}
