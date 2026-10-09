package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

const (
	imdbGraphQLEndpoint = "https://api.graphql.imdb.com/"
	imdbTitlesPerPage   = 250
	imdbMaxPages        = 48
	imdbMaxResponseSize = 10 << 20
)

const imdbListQuery = `query PublicList($listId: ID!, $after: String) {
	watchlist: list(id: $listId) {
		titleListItemSearch(first: 250, after: $after) {
			edges { title { id titleText { text } } }
			pageInfo { hasNextPage endCursor }
		}
	}
}`

const imdbWatchlistQuery = `query PublicWatchlist($userId: ID!, $after: String) {
	watchlist: predefinedList(classType: WATCH_LIST, userId: $userId) {
		titleListItemSearch(first: 250, after: $after) {
			edges { title { id titleText { text } } }
			pageInfo { hasNextPage endCursor }
		}
	}
}`

const imdbProfileQuery = `query PublicProfile($profileId: ID!) {
	userProfile(input: {profileId: $profileId}) { userId }
}`

var (
	imdbListPathPattern      = regexp.MustCompile(`^/list/(ls[0-9]+)/?$`)
	imdbWatchlistPathPattern = regexp.MustCompile(`^/user/(ur[0-9]+|p\.[A-Za-z0-9_-]+)/watchlist/?$`)
)

var imdbHTTPClient = &http.Client{Timeout: 20 * time.Second}

type imdbSourceKind int

const (
	imdbPublicList imdbSourceKind = iota
	imdbUserWatchlist
)

type imdbSource struct {
	Kind imdbSourceKind
	ID   string
}

type imdbTitle struct {
	ID   string
	Name string
}

type imdbGraphQLError struct {
	Message string `json:"message"`
}

type imdbTitleConnection struct {
	Edges []struct {
		Title struct {
			ID        string `json:"id"`
			TitleText struct {
				Text string `json:"text"`
			} `json:"titleText"`
		} `json:"title"`
	} `json:"edges"`
	PageInfo struct {
		HasNextPage bool   `json:"hasNextPage"`
		EndCursor   string `json:"endCursor"`
	} `json:"pageInfo"`
}

type imdbTitlesResponse struct {
	Data struct {
		Watchlist *struct {
			Titles imdbTitleConnection `json:"titleListItemSearch"`
		} `json:"watchlist"`
	} `json:"data"`
	Errors []imdbGraphQLError `json:"errors"`
}

type imdbProfileResponse struct {
	Data struct {
		UserProfile *struct {
			UserID string `json:"userId"`
		} `json:"userProfile"`
	} `json:"data"`
	Errors []imdbGraphQLError `json:"errors"`
}

func randomIMDbTitle(ctx context.Context, watchlistURL string, client *http.Client) (imdbTitle, int, error) {
	source, err := parseIMDbSource(watchlistURL)
	if err != nil {
		return imdbTitle{}, 0, err
	}

	titles, err := fetchIMDbTitles(ctx, client, source)
	if err != nil {
		return imdbTitle{}, 0, err
	}
	if len(titles) == 0 {
		return imdbTitle{}, 0, errors.New("no titles found in that public IMDb watchlist")
	}

	index, err := cryptoRandomIndex(len(titles))
	if err != nil {
		return imdbTitle{}, 0, err
	}
	return titles[index], len(titles), nil
}

func parseIMDbSource(rawURL string) (imdbSource, error) {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return imdbSource{}, errors.New("enter a full public IMDb watchlist URL")
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return imdbSource{}, errors.New("IMDb watchlist URL must start with http:// or https://")
	}

	host := strings.ToLower(parsed.Hostname())
	if host != "imdb.com" && host != "www.imdb.com" {
		return imdbSource{}, errors.New("enter a watchlist URL from imdb.com")
	}

	path := parsed.EscapedPath()
	if match := imdbListPathPattern.FindStringSubmatch(path); match != nil {
		return imdbSource{Kind: imdbPublicList, ID: match[1]}, nil
	}
	if match := imdbWatchlistPathPattern.FindStringSubmatch(path); match != nil {
		return imdbSource{Kind: imdbUserWatchlist, ID: match[1]}, nil
	}
	return imdbSource{}, errors.New("enter a public IMDb watchlist link")
}

func imdbWatchlistPath(rawURL string) (string, error) {
	source, err := parseIMDbSource(rawURL)
	if err != nil {
		return "", err
	}
	if source.Kind == imdbPublicList {
		return "/list/" + source.ID + "/", nil
	}
	return "/user/" + source.ID + "/watchlist/", nil
}

func fetchIMDbTitles(ctx context.Context, client *http.Client, source imdbSource) ([]imdbTitle, error) {
	if client == nil {
		client = imdbHTTPClient
	}

	query := imdbListQuery
	idVariable := "listId"
	id := source.ID
	if source.Kind == imdbUserWatchlist {
		query = imdbWatchlistQuery
		idVariable = "userId"
		if strings.HasPrefix(id, "p.") {
			resolvedID, err := resolveIMDbUserID(ctx, client, id)
			if err != nil {
				return nil, err
			}
			id = resolvedID
		}
	}

	seen := make(map[string]bool)
	var titles []imdbTitle
	var after any

	for page := 0; page < imdbMaxPages; page++ {
		var response imdbTitlesResponse
		variables := map[string]any{idVariable: id, "after": after}
		if err := postIMDbGraphQL(ctx, client, query, variables, &response); err != nil {
			return nil, err
		}
		if len(response.Errors) > 0 || response.Data.Watchlist == nil {
			return nil, errors.New("IMDb could not read that watchlist; check that it is public")
		}

		connection := response.Data.Watchlist.Titles
		for _, edge := range connection.Edges {
			title := imdbTitle{
				ID:   strings.TrimSpace(edge.Title.ID),
				Name: strings.TrimSpace(edge.Title.TitleText.Text),
			}
			if title.ID == "" || title.Name == "" || seen[title.ID] {
				continue
			}
			seen[title.ID] = true
			titles = append(titles, title)
		}

		if !connection.PageInfo.HasNextPage {
			return titles, nil
		}
		if connection.PageInfo.EndCursor == "" {
			return nil, errors.New("IMDb returned an incomplete watchlist")
		}
		after = connection.PageInfo.EndCursor
	}

	return nil, errors.New("IMDb watchlist is too large")
}

func resolveIMDbUserID(ctx context.Context, client *http.Client, profileID string) (string, error) {
	var response imdbProfileResponse
	if err := postIMDbGraphQL(ctx, client, imdbProfileQuery, map[string]any{"profileId": profileID}, &response); err != nil {
		return "", err
	}
	if len(response.Errors) > 0 || response.Data.UserProfile == nil || response.Data.UserProfile.UserID == "" {
		return "", errors.New("IMDb could not resolve that public profile")
	}
	return response.Data.UserProfile.UserID, nil
}

func postIMDbGraphQL(ctx context.Context, client *http.Client, query string, variables map[string]any, target any) error {
	payload, err := json.Marshal(map[string]any{
		"query":     query,
		"variables": variables,
	})
	if err != nil {
		return fmt.Errorf("encode IMDb request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, imdbGraphQLEndpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create IMDb request: %w", err)
	}
	request.Header.Set("Accept", "application/graphql+json, application/json")
	request.Header.Set("Accept-Language", "en-US,en;q=0.9")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "https://www.imdb.com")
	request.Header.Set("Referer", "https://www.imdb.com/")
	request.Header.Set("User-Agent", "Mozilla/5.0 (compatible; SpotifyRandomizer/1.0)")
	request.Header.Set("X-Imdb-Client-Name", "imdb-web-next-localized")

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("load IMDb watchlist: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("IMDb returned %s", response.Status)
	}

	body := io.LimitReader(response.Body, imdbMaxResponseSize+1)
	limited, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("read IMDb response: %w", err)
	}
	if len(limited) > imdbMaxResponseSize {
		return errors.New("IMDb response is too large")
	}
	if err := json.Unmarshal(limited, target); err != nil {
		return fmt.Errorf("parse IMDb response: %w", err)
	}
	return nil
}
