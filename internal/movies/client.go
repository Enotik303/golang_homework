
package movies

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

var (
	ErrNotFound    = errors.New("фильм не найден")
	ErrRateLimited = errors.New("стенд ограничил частоту запросов")
	ErrUpstream    = errors.New("ошибка внешнего стенда")
)

type Movie struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Year     int    `json:"year"`
	Director string `json:"director"`
}

type Client struct {
	baseURL string
	http    *http.Client
	timeout time.Duration
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{baseURL: baseURL, http: &http.Client{}, timeout: timeout}
}

func (c *Client) Get(ctx context.Context, id int) (Movie, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	url := fmt.Sprintf("%s/%d/info.0.json", c.baseURL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Movie{}, fmt.Errorf("фильм %d: %w: %v", id, ErrUpstream, err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return Movie{}, fmt.Errorf("фильм %d: запрос не выполнен: %w", id, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return Movie{}, fmt.Errorf("фильм %d: %w", id, ErrNotFound)
	case http.StatusTooManyRequests:
		return Movie{}, fmt.Errorf("фильм %d: %w", id, ErrRateLimited)
	default:
		return Movie{}, fmt.Errorf("фильм %d: %w (статус %d)", id, ErrUpstream, resp.StatusCode)
	}

	var m Movie
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return Movie{}, fmt.Errorf("фильм %d: %w: битый JSON: %v", id, ErrUpstream, err)
	}
	return m, nil
}
