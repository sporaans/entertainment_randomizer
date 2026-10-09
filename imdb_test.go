package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIMDbWatchlistPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		url     string
		want    string
		wantErr bool
	}{
		{name: "public list", url: "https://www.imdb.com/list/ls123456789/", want: "/list/ls123456789/"},
		{name: "list share query", url: "https://imdb.com/list/ls987654321?ref_=ext_shr_lnk", want: "/list/ls987654321/"},
		{name: "legacy user watchlist", url: "https://www.imdb.com/user/ur12345678/watchlist/", want: "/user/ur12345678/watchlist/"},
		{name: "new user watchlist", url: "https://www.imdb.com/user/p.fl6ssgsolkgcctcuxapgh6chsa/watchlist?ref_=wl_share", want: "/user/p.fl6ssgsolkgcctcuxapgh6chsa/watchlist/"},
		{name: "other host", url: "https://example.com/list/ls123456789/", wantErr: true},
		{name: "title page", url: "https://www.imdb.com/title/tt1234567/", wantErr: true},
		{name: "not a URL", url: "ls123456789", wantErr: true},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := imdbWatchlistPath(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("imdbWatchlistPath() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("imdbWatchlistPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFetchIMDbPublicList(t *testing.T) {
	t.Parallel()

	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.String() != imdbGraphQLEndpoint {
			t.Fatalf("IMDb request URL = %q", req.URL)
		}
		var payload struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if !strings.Contains(payload.Query, "list(id: $listId)") || payload.Variables["listId"] != "ls123456789" {
			t.Fatalf("unexpected GraphQL request: %#v", payload)
		}
		return jsonHTTPResponse(req, `{
			"data":{"watchlist":{"titleListItemSearch":{
				"edges":[{"title":{"id":"tt1234567","titleText":{"text":"A Movie"}}}],
				"pageInfo":{"hasNextPage":false,"endCursor":"cursor"}
			}}}
		}`), nil
	})}

	got, err := fetchIMDbTitles(context.Background(), client, imdbSource{Kind: imdbPublicList, ID: "ls123456789"})
	if err != nil {
		t.Fatalf("fetchIMDbTitles() error = %v", err)
	}
	if len(got) != 1 || got[0].Name != "A Movie" {
		t.Fatalf("fetchIMDbTitles() = %#v", got)
	}
}

func TestFetchIMDbNewProfileWatchlistResolvesIDAndPaginates(t *testing.T) {
	t.Parallel()

	call := 0
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		call++
		var payload struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables"`
		}
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		switch call {
		case 1:
			if payload.Variables["profileId"] != "p.qwdavjczqyzed6wzr27hvaxsxi" {
				t.Fatalf("profile ID = %#v", payload.Variables["profileId"])
			}
			return jsonHTTPResponse(req, `{"data":{"userProfile":{"userId":"ur22633103"}}}`), nil
		case 2:
			if payload.Variables["userId"] != "ur22633103" || payload.Variables["after"] != nil {
				t.Fatalf("first page variables = %#v", payload.Variables)
			}
			return jsonHTTPResponse(req, `{
				"data":{"watchlist":{"titleListItemSearch":{
					"edges":[{"title":{"id":"tt2692904","titleText":{"text":"Locke"}}}],
					"pageInfo":{"hasNextPage":true,"endCursor":"next-page"}
				}}}
			}`), nil
		case 3:
			if payload.Variables["after"] != "next-page" {
				t.Fatalf("second page cursor = %#v", payload.Variables["after"])
			}
			return jsonHTTPResponse(req, `{
				"data":{"watchlist":{"titleListItemSearch":{
					"edges":[{"title":{"id":"tt0093748","titleText":{"text":"Planes, Trains & Automobiles"}}}],
					"pageInfo":{"hasNextPage":false,"endCursor":"done"}
				}}}
			}`), nil
		default:
			t.Fatalf("unexpected request %d", call)
			return nil, nil
		}
	})}

	got, err := fetchIMDbTitles(context.Background(), client, imdbSource{
		Kind: imdbUserWatchlist,
		ID:   "p.qwdavjczqyzed6wzr27hvaxsxi",
	})
	if err != nil {
		t.Fatalf("fetchIMDbTitles() error = %v", err)
	}
	if len(got) != 2 || got[0].Name != "Locke" || got[1].Name != "Planes, Trains & Automobiles" {
		t.Fatalf("fetchIMDbTitles() = %#v", got)
	}
}

func TestRandomMovieHandler(t *testing.T) {
	originalClient := imdbHTTPClient
	t.Cleanup(func() {
		imdbHTTPClient = originalClient
	})
	imdbHTTPClient = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		return jsonHTTPResponse(req, `{
			"data":{"watchlist":{"titleListItemSearch":{
				"edges":[{"title":{"id":"tt1234567","titleText":{"text":"A Movie"}}}],
				"pageInfo":{"hasNextPage":false,"endCursor":"cursor"}
			}}}
		}`), nil
	})}

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/random-movie",
		strings.NewReader(`{"url":"https://www.imdb.com/user/ur12345678/watchlist/"}`),
	)
	response := httptest.NewRecorder()
	(&app{}).handleRandomMovie(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	var payload movieResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload.Name != "A Movie" {
		t.Fatalf("movie name = %q", payload.Name)
	}
}

func TestRandomMovieHandlerRejectsNonIMDbURL(t *testing.T) {
	request := httptest.NewRequest(
		http.MethodPost,
		"/api/random-movie",
		strings.NewReader(`{"url":"https://example.com/list/ls123456789/"}`),
	)
	response := httptest.NewRecorder()
	(&app{}).handleRandomMovie(response, request)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func jsonHTTPResponse(req *http.Request, body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Status:     "200 OK",
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}
