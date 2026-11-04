package moviestand

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"movieservice/internal/domain"
)

func TestGetMovie(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/1/info.0.json":
			fmt.Fprint(w, `{"id":1,"title":"To Rome with Love","year":2012,"director":"Woody Allen"}`)
		case "/2/info.0.json":
			http.NotFound(w, r)
		case "/3/info.0.json":
			w.WriteHeader(http.StatusTooManyRequests)
		case "/4/info.0.json":
			w.WriteHeader(http.StatusInternalServerError)
		case "/5/info.0.json":
			fmt.Fprint(w, `{это не json`)
		}
	}))
	defer srv.Close()

	c := NewClient(srv.URL, time.Second)

	m, err := c.GetMovie(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	want := domain.Movie{ID: 1, Title: "To Rome with Love", Year: 2012, Director: "Woody Allen"}
	if m != want {
		t.Fatalf("ожидали %+v, получили %+v", want, m)
	}

	cases := []struct {
		id   int
		want error
	}{
		{2, domain.ErrNotFound},
		{3, domain.ErrRateLimited},
		{4, domain.ErrUpstream},
		{5, domain.ErrUpstream},
	}
	for _, tc := range cases {
		_, err := c.GetMovie(context.Background(), tc.id)
		if !errors.Is(err, tc.want) {
			t.Errorf("id %d: ожидали %v, получили %v", tc.id, tc.want, err)
		}
	}
}

func TestGetMovieTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(300 * time.Millisecond)
	}))
	defer srv.Close()

	c := NewClient(srv.URL, 50*time.Millisecond)

	_, err := c.GetMovie(context.Background(), 1)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("ожидали DeadlineExceeded, получили %v", err)
	}
}
