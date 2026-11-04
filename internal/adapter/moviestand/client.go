package moviestand

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"movieservice/internal/domain"
)

type Client struct {
	baseURL string
	http    *http.Client
	timeout time.Duration
}

func NewClient(baseURL string, timeout time.Duration) *Client {
	return &Client{baseURL: baseURL, http: &http.Client{}, timeout: timeout}
}

type movieResponse struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Year     int    `json:"year"`
	Director string `json:"director"`
}

func (r movieResponse) toDomain() domain.Movie {
	return domain.Movie{ID: r.ID, Title: r.Title, Year: r.Year, Director: r.Director}
}

func (c *Client) GetMovie(ctx context.Context, id int) (domain.Movie, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	url := fmt.Sprintf("%s/%d/info.0.json", c.baseURL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return domain.Movie{}, fmt.Errorf("фильм %d: %w: %v", id, domain.ErrUpstream, err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return domain.Movie{}, fmt.Errorf("фильм %d: запрос не выполнен: %w", id, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return domain.Movie{}, fmt.Errorf("фильм %d: %w", id, domain.ErrNotFound)
	case http.StatusTooManyRequests:
		return domain.Movie{}, fmt.Errorf("фильм %d: %w", id, domain.ErrRateLimited)
	default:
		return domain.Movie{}, fmt.Errorf("фильм %d: %w (статус %d)", id, domain.ErrUpstream, resp.StatusCode)
	}

	var body movieResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return domain.Movie{}, fmt.Errorf("фильм %d: %w: битый JSON: %v", id, domain.ErrUpstream, err)
	}
	return body.toDomain(), nil
}
