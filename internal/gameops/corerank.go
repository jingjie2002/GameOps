package gameops

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"
)

type CoreRankClient struct {
	baseURL string
	client  *http.Client
}

func NewCoreRankClient(baseURL string) CoreRankClient {
	return CoreRankClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 3 * time.Second},
	}
}

func (c CoreRankClient) Health(ctx context.Context) (map[string]any, error) {
	var result map[string]any
	err := c.getJSON(ctx, "/health", &result)
	return result, err
}

func (c CoreRankClient) Leaderboard(ctx context.Context, leaderboardType string, limit string) ([]map[string]any, error) {
	query := url.Values{}
	if limit != "" {
		query.Set("n", limit)
	}
	if leaderboardType != "" {
		query.Set("leaderboard_type", leaderboardType)
	}
	var result []map[string]any
	err := c.getJSON(ctx, "/api/rank/top?"+query.Encode(), &result)
	return result, err
}

func (c CoreRankClient) PlayerRank(ctx context.Context, playerID string, leaderboardType string) (map[string]any, error) {
	query := url.Values{}
	if leaderboardType != "" {
		query.Set("leaderboard_type", leaderboardType)
	}
	path := "/api/rank/player/" + url.PathEscape(playerID)
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	var result map[string]any
	err := c.getJSON(ctx, path, &result)
	return result, err
}

func (c CoreRankClient) getJSON(ctx context.Context, path string, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return err
	}
	response, err := c.client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode >= 300 {
		return ErrNotFound
	}
	return json.NewDecoder(response.Body).Decode(target)
}
