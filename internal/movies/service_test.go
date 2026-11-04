package movies

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func newStand(t *testing.T, last int) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/") // /{id}/info.0.json
		id, err := strconv.Atoi(parts[0])
		if err != nil || id < 1 || id > last {
			http.NotFound(w, r)
			return
		}
		fmt.Fprintf(w, `{"id":%d,"title":"Movie %d","year":2000,"director":"Dir"}`, id, id)
	}))
	t.Cleanup(srv.Close)
	return srv
}

func TestLast(t *testing.T) {
	stand := newStand(t, 37)
	svc := NewService(NewClient(stand.URL, time.Second), 4)

	m, err := svc.Last(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if m.ID != 37 {
		t.Fatalf("ожидали последний фильм 37, получили %d", m.ID)
	}
}

func TestLoadRangePartial(t *testing.T) {
	stand := newStand(t, 3)
	svc := NewService(NewClient(stand.URL, time.Second), 2)

	res, err := svc.LoadRange(context.Background(), 2, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 4 {
		t.Fatalf("ожидали 4 результата, получили %d", len(res))
	}
	if res[0].Err != nil || res[1].Err != nil {
		t.Fatalf("фильмы 2 и 3 должны загрузиться: %v %v", res[0].Err, res[1].Err)
	}
	if !errors.Is(res[2].Err, ErrNotFound) || !errors.Is(res[3].Err, ErrNotFound) {
		t.Fatalf("фильмы 4 и 5 должны дать ErrNotFound: %v %v", res[2].Err, res[3].Err)
	}
}

func TestLoadRangeInvalid(t *testing.T) {
	svc := NewService(NewClient("http://127.0.0.1:0", time.Second), 2)

	_, err := svc.LoadRange(context.Background(), 5, 2)
	if !errors.Is(err, ErrInvalidRange) {
		t.Fatalf("ожидали ErrInvalidRange, получили %v", err)
	}
}
