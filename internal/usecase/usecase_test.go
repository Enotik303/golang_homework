package usecase

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"movieservice/internal/domain"
)

type fakeProvider struct {
	last   int
	failID map[int]error
}

func (f fakeProvider) GetMovie(_ context.Context, id int) (domain.Movie, error) {
	if err, ok := f.failID[id]; ok {
		return domain.Movie{}, err
	}
	if id < 1 || id > f.last {
		return domain.Movie{}, domain.ErrNotFound
	}
	return domain.Movie{ID: id, Title: fmt.Sprintf("Movie %d", id)}, nil
}

func TestGetLastMovie(t *testing.T) {
	uc := NewService(fakeProvider{last: 37}, 4)

	m, err := uc.GetLastMovie(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != 37 {
		t.Fatalf("ожидали 37, получили %d", m.ID)
	}
}

func TestLoadMoviesSortedByID(t *testing.T) {
	uc := NewService(fakeProvider{last: 10}, 3)

	movies, err := uc.LoadMovies(context.Background(), 3, 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(movies) != 5 {
		t.Fatalf("ожидали 5 фильмов, получили %d", len(movies))
	}
	for i, m := range movies {
		if m.ID != 3+i {
			t.Fatalf("фильмы не по порядку: позиция %d, id %d", i, m.ID)
		}
	}
}

func TestLoadMoviesPartial(t *testing.T) {
	uc := NewService(fakeProvider{last: 3}, 2)

	movies, err := uc.LoadMovies(context.Background(), 2, 5)

	var partial *domain.PartialLoadError
	if !errors.As(err, &partial) {
		t.Fatalf("ожидали *PartialLoadError, получили %v", err)
	}
	if len(movies) != 2 || len(partial.Failures) != 2 {
		t.Fatalf("ожидали 2 фильма и 2 ошибки, получили %d и %d", len(movies), len(partial.Failures))
	}
	if !errors.Is(partial.Failures[0].Err, domain.ErrNotFound) {
		t.Fatalf("ожидали ErrNotFound, получили %v", partial.Failures[0].Err)
	}
}

func TestLoadMoviesAllFailed(t *testing.T) {
	uc := NewService(fakeProvider{last: 3}, 2)

	movies, err := uc.LoadMovies(context.Background(), 10, 12)
	if movies != nil || !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("ожидали (nil, ErrNotFound), получили (%v, %v)", movies, err)
	}
}

func TestLoadMoviesInvalidRange(t *testing.T) {
	uc := NewService(fakeProvider{last: 3}, 2)

	for _, tc := range [][2]int{{5, 2}, {0, 3}, {-1, 3}, {1, MaxRange + 1}} {
		_, err := uc.LoadMovies(context.Background(), tc[0], tc[1])
		if !errors.Is(err, domain.ErrInvalidRange) {
			t.Errorf("from=%d to=%d: ожидали ErrInvalidRange, получили %v", tc[0], tc[1], err)
		}
	}
}
