package usecase

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"movieservice/internal/domain"
)

const (
	MaxID    = 99999
	MaxRange = 10000
)

type MovieProvider interface {
	GetMovie(ctx context.Context, id int) (domain.Movie, error)
}

type MovieUseCase interface {
	LoadMovies(ctx context.Context, from, to int) ([]domain.Movie, error)
	GetLastMovie(ctx context.Context) (domain.Movie, error)
}

type Service struct {
	provider MovieProvider
	workers  int
}

var _ MovieUseCase = (*Service)(nil)

func NewService(provider MovieProvider, workers int) *Service {
	if workers <= 0 {
		workers = 1
	}
	return &Service{provider: provider, workers: workers}
}

type result struct {
	id    int
	movie domain.Movie
	err   error
}

func (s *Service) LoadMovies(ctx context.Context, from, to int) ([]domain.Movie, error) {
	if from < 1 || to < from || to-from+1 > MaxRange {
		return nil, fmt.Errorf("%w: from=%d, to=%d (нужно 1 <= from <= to, не более %d фильмов)",
			domain.ErrInvalidRange, from, to, MaxRange)
	}

	total := to - from + 1
	jobs := make(chan int, total)
	for id := from; id <= to; id++ {
		jobs <- id
	}
	close(jobs)

	results := make(chan result, total)

	var wg sync.WaitGroup
	for w := 0; w < min(s.workers, total); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for id := range jobs {
				if ctx.Err() != nil {
					return
				}
				m, err := s.provider.GetMovie(ctx, id)
				results <- result{id: id, movie: m, err: err}
			}
		}()
	}
	wg.Wait()
	close(results)

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	all := make([]result, 0, total)
	for r := range results {
		all = append(all, r)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].id < all[j].id })

	movies := make([]domain.Movie, 0, len(all))
	var failures []domain.LoadFailure
	for _, r := range all {
		if r.err != nil {
			failures = append(failures, domain.LoadFailure{ID: r.id, Err: r.err})
			continue
		}
		movies = append(movies, r.movie)
	}

	switch {
	case len(failures) == 0:
		return movies, nil
	case len(movies) == 0:
		return nil, failures[0].Err
	default:
		return movies, &domain.PartialLoadError{Failures: failures}
	}
}

func (s *Service) GetLastMovie(ctx context.Context) (domain.Movie, error) {
	lo, hi := 0, MaxID
	var last domain.Movie

	for lo < hi {
		mid := (lo + hi + 1) / 2
		m, err := s.provider.GetMovie(ctx, mid)
		switch {
		case err == nil:
			lo, last = mid, m
		case errors.Is(err, domain.ErrNotFound):
			hi = mid - 1
		default:
			return domain.Movie{}, err
		}
	}

	if lo == 0 {
		return domain.Movie{}, fmt.Errorf("на стенде нет ни одного фильма: %w", domain.ErrNotFound)
	}
	return last, nil
}
