package httphandler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"movieservice/internal/domain"
)

type fakeUC struct {
	movies []domain.Movie
	err    error
}

func (f fakeUC) LoadMovies(context.Context, int, int) ([]domain.Movie, error) {
	return f.movies, f.err
}

func (f fakeUC) GetLastMovie(context.Context) (domain.Movie, error) {
	return domain.Movie{ID: 7, Title: "Last"}, f.err
}

func get(h http.Handler, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

func TestLoadMoviesBadParams(t *testing.T) {
	h := NewRouter(fakeUC{})

	if rec := get(h, "/load-movies?from=abc&to=3"); rec.Code != http.StatusBadRequest {
		t.Fatalf("from=abc: ожидали 400, получили %d", rec.Code)
	}
	if rec := get(h, "/load-movies?to=3"); rec.Code != http.StatusBadRequest {
		t.Fatalf("нет from: ожидали 400, получили %d", rec.Code)
	}
}

func TestLoadMoviesInvalidRangeFromUseCase(t *testing.T) {
	h := NewRouter(fakeUC{err: domain.ErrInvalidRange})

	if rec := get(h, "/load-movies?from=5&to=1"); rec.Code != http.StatusBadRequest {
		t.Fatalf("ожидали 400, получили %d", rec.Code)
	}
}

func TestLoadMoviesPartial(t *testing.T) {
	h := NewRouter(fakeUC{
		movies: []domain.Movie{{ID: 1, Title: "A"}},
		err: &domain.PartialLoadError{Failures: []domain.LoadFailure{
			{ID: 2, Err: domain.ErrNotFound},
		}},
	})

	rec := get(h, "/load-movies?from=1&to=2")
	if rec.Code != http.StatusOK {
		t.Fatalf("ожидали 200, получили %d", rec.Code)
	}
	if rec.Header().Get("X-Failed-Count") != "1" {
		t.Fatalf("ожидали X-Failed-Count=1, получили %q", rec.Header().Get("X-Failed-Count"))
	}

	var items []map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("ожидали 2 элемента, получили %d", len(items))
	}
	if _, ok := items[1]["error"]; !ok {
		t.Fatalf("у второго элемента должно быть поле error: %v", items[1])
	}
}

func TestLastMovieNotFound(t *testing.T) {
	h := NewRouter(fakeUC{err: domain.ErrNotFound})

	if rec := get(h, "/last-movie"); rec.Code != http.StatusNotFound {
		t.Fatalf("ожидали 404, получили %d", rec.Code)
	}
}

func TestSwaggerAvailable(t *testing.T) {
	h := NewRouter(fakeUC{})

	if rec := get(h, "/swagger/index.html"); rec.Code != http.StatusOK {
		t.Fatalf("ожидали 200, получили %d", rec.Code)
	}
	if rec := get(h, "/swagger/openapi.json"); rec.Code != http.StatusOK || rec.Body.Len() == 0 {
		t.Fatalf("openapi.json не отдаётся: код %d", rec.Code)
	}
}
