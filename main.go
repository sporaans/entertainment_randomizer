package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	_ = godotenv.Load()

	paths, err := appFilePaths()
	if err != nil {
		return err
	}

	settings, err := loadAppSettings(paths.configPath)
	if err != nil {
		return err
	}

	token, err := initialToken(context.Background(), settings, paths.tokenPath)
	if err != nil {
		return err
	}

	spotifyApp := newApp(settings, paths, token)
	server := &http.Server{
		Addr:              settings.ServerAddr,
		Handler:           spotifyApp.routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		log.Printf("Spotify Randomizer webapp listening on %s", displayServerURL(settings.ServerAddr))
		if !settings.IsConfigured() {
			log.Printf("Open %s to register your Spotify app client.", displayServerURL(settings.ServerAddr))
		}
		errCh <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
