package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	spotifyauth "github.com/zmb3/spotify/v2/auth"
	"golang.org/x/oauth2"
)

type app struct {
	auth     *spotifyauth.Authenticator
	settings appSettings
	paths    appPaths
	token    *oauth2.Token
	configMu sync.RWMutex
	tokenMu  sync.RWMutex
}

type pageData struct {
	IsConfigured bool
	IsConnected  bool
	Settings     appSettings
	ConfigPath   string
}

type setupPageData struct {
	Settings      appSettings
	Error         string
	ConfigPath    string
	RestartNotice bool
}

type albumResponse struct {
	Name        string `json:"name"`
	Artists     string `json:"artists"`
	ReleaseDate string `json:"releaseDate"`
	SpotifyURL  string `json:"spotifyUrl"`
	Total       int    `json:"total"`
}

type imdbRequest struct {
	URL string `json:"url"`
}

type movieResponse struct {
	Name string `json:"name"`
}

type errorResponse struct {
	Error string `json:"error"`
}

func newApp(settings appSettings, paths appPaths, token *oauth2.Token) *app {
	return &app{
		auth:     newSpotifyAuth(settings),
		settings: settings,
		paths:    paths,
		token:    token,
	}
}

func (a *app) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/", a.handleHome)
	mux.HandleFunc("/setup", a.handleSetup)
	mux.HandleFunc("/login", a.handleLogin)
	mux.HandleFunc("/callback", a.handleCallback)
	mux.HandleFunc("/api/random-album", a.handleRandomAlbum)
	mux.HandleFunc("/api/random-movie", a.handleRandomMovie)
	return mux
}

func (a *app) handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	settings := a.currentSettings()
	if !settings.IsConfigured() {
		a.renderSetup(w, setupPageData{
			Settings:   settings,
			ConfigPath: a.paths.configPath,
		})
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := homeTemplate.Execute(w, pageData{
		IsConfigured: true,
		IsConnected:  a.hasValidToken(),
		Settings:     settings,
		ConfigPath:   a.paths.configPath,
	}); err != nil {
		log.Printf("render home: %v", err)
	}
}

func (a *app) handleSetup(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		a.renderSetup(w, setupPageData{
			Settings:   a.currentSettings(),
			ConfigPath: a.paths.configPath,
		})
	case http.MethodPost:
		a.saveSetup(w, r)
	default:
		w.Header().Set("Allow", "GET, POST")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (a *app) saveSetup(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		a.renderSetup(w, setupPageData{
			Settings:   a.currentSettings(),
			Error:      "Could not read the setup form.",
			ConfigPath: a.paths.configPath,
		})
		return
	}

	current := a.currentSettings()
	next := appSettings{
		ClientID:     strings.TrimSpace(r.FormValue("client_id")),
		ClientSecret: strings.TrimSpace(r.FormValue("client_secret")),
		RedirectURL:  strings.TrimSpace(r.FormValue("redirect_url")),
		ServerAddr:   current.ServerAddr,
	}.WithDefaults()

	if err := validateSetupSettings(next); err != nil {
		a.renderSetup(w, setupPageData{
			Settings:   next,
			Error:      err.Error(),
			ConfigPath: a.paths.configPath,
		})
		return
	}

	if err := saveStoredSettings(a.paths.configPath, next); err != nil {
		a.renderSetup(w, setupPageData{
			Settings:   next,
			Error:      "Could not save setup config.",
			ConfigPath: a.paths.configPath,
		})
		log.Printf("save setup config: %v", err)
		return
	}

	a.replaceSettings(next)
	a.setToken(nil)
	if err := os.Remove(a.paths.tokenPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Printf("clear token after setup update: %v", err)
	}

	http.Redirect(w, r, "/login?force=1", http.StatusFound)
}

func (a *app) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	auth := a.currentAuth()
	if auth == nil {
		http.Redirect(w, r, "/setup", http.StatusFound)
		return
	}

	state, err := randomState()
	if err != nil {
		http.Error(w, "could not start spotify login", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "spotify_auth_state",
		Value:    state,
		Path:     "/",
		MaxAge:   600,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	authOpts := []oauth2.AuthCodeOption{}
	if r.URL.Query().Get("force") == "1" {
		authOpts = append(authOpts, oauth2.SetAuthURLParam("show_dialog", "true"))
	}

	http.Redirect(w, r, auth.AuthURL(state, authOpts...), http.StatusFound)
}

func (a *app) handleCallback(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	auth := a.currentAuth()
	if auth == nil {
		http.Redirect(w, r, "/setup", http.StatusFound)
		return
	}

	stateCookie, err := r.Cookie("spotify_auth_state")
	if err != nil || stateCookie.Value == "" || r.URL.Query().Get("state") != stateCookie.Value {
		http.Error(w, "spotify login state mismatch", http.StatusBadRequest)
		return
	}

	token, err := auth.Token(r.Context(), stateCookie.Value, r)
	if err != nil {
		http.Error(w, "spotify authorization failed", http.StatusBadRequest)
		return
	}

	if err := saveToken(a.paths.tokenPath, token); err != nil {
		http.Error(w, "could not save spotify token", http.StatusInternalServerError)
		return
	}

	a.setToken(token)
	http.SetCookie(w, &http.Cookie{Name: "spotify_auth_state", Value: "", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/", http.StatusFound)
}

func (a *app) handleRandomAlbum(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	client, err := a.spotifyClient(r.Context())
	if err != nil {
		writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "Connect Spotify first."})
		return
	}

	album, total, err := randomSavedAlbum(r.Context(), client)
	if err != nil {
		if errors.Is(handleSpotifyError(a.paths.tokenPath, err), errReauthorize) {
			a.setToken(nil)
			writeJSON(w, http.StatusUnauthorized, errorResponse{Error: "Spotify needs a fresh login."})
			return
		}
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, albumResponse{
		Name:        album.Name,
		Artists:     joinArtistNames(album.Artists),
		ReleaseDate: album.ReleaseDate,
		SpotifyURL:  album.ExternalURLs["spotify"],
		Total:       total,
	})
}

func (a *app) handleRandomMovie(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, errorResponse{Error: "method not allowed"})
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	var request imdbRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "Could not read the IMDb watchlist link."})
		return
	}
	if _, err := imdbWatchlistPath(request.URL); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
		return
	}

	title, _, err := randomIMDbTitle(r.Context(), request.URL, imdbHTTPClient)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, errorResponse{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, movieResponse{Name: title.Name})
}

func (a *app) renderSetup(w http.ResponseWriter, data setupPageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := setupTemplate.Execute(w, data); err != nil {
		log.Printf("render setup: %v", err)
	}
}

func (a *app) currentAuth() *spotifyauth.Authenticator {
	a.configMu.RLock()
	defer a.configMu.RUnlock()
	return a.auth
}

func (a *app) currentSettings() appSettings {
	a.configMu.RLock()
	defer a.configMu.RUnlock()
	return a.settings
}

func (a *app) replaceSettings(settings appSettings) {
	a.configMu.Lock()
	defer a.configMu.Unlock()
	a.settings = settings.WithDefaults()
	a.auth = newSpotifyAuth(a.settings)
}

func (a *app) hasValidToken() bool {
	a.tokenMu.RLock()
	defer a.tokenMu.RUnlock()
	return a.token != nil
}

func (a *app) setToken(token *oauth2.Token) {
	a.tokenMu.Lock()
	defer a.tokenMu.Unlock()
	a.token = token
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write json: %v", err)
	}
}
