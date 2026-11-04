package movies

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
)

const (
	MaxID = 99999
	MaxRange = 10000
)

var ErrInvalidRange = errors.New("некорректный диапазон")

type Result struct {
	ID    int
	Movie Movie
	Err   error
}

type Service struct {
	client  *Client
	workers int
}

func NewService(client *Client, workers int) *Service {
	if workers <= 0 {
		workers = 1
	}
	return &Service{client: client, workers: workers}
}

func (s *Service) LoadRange(ctx context.Context, from, to int) ([]Result, error) {
	if from < 1 || to < from || to-from+1 > MaxRange {
		return nil, fmt.Errorf("%w: from=%d, to=%d (нужно 1 <= from <= to, не более %d фильмов)",
			ErrInvalidRange, from, to, MaxRange)
	}

	total := to - from + 1
	jobs := make(chan int, total)
	for id := from; id <= to; id++ {
		jobs <- id
	}
	close(jobs)

	results := make(chan Result, total)

	var wg sync.WaitGroup
	for w := 0; w < min(s.workers, total); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for id := range jobs {
				if ctx.Err() != nil {
					return 
				}
				m, err := s.client.Get(ctx, id)
				results <- Result{ID: id, Movie: m, Err: err}
			}
		}()
	}
	wg.Wait()
	close(results)

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	out := make([]Result, 0, total)
	for r := range results {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (s *Service) Last(ctx context.Context) (Movie, error) {
	lo, hi := 0, MaxID
	var last Movie

	for lo < hi {
		mid := (lo + hi + 1) / 2
		m, err := s.client.Get(ctx, mid)
		switch {
		case err == nil:
			lo, last = mid, m
		case errors.Is(err, ErrNotFound):
			hi = mid - 1
		default:
			return Movie{}, err
		}
	}

	if lo == 0 {
		return Movie{}, fmt.Errorf("на стенде нет ни одного фильма: %w", ErrNotFound)
	}
	return last, nil
}
