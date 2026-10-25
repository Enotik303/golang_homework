
package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"movieservice/internal/movies"
)

type MovieService interface {
	LoadRange(ctx context.Context, from, to int) ([]movies.Result, error)
	Last(ctx context.Context) (movies.Movie, error)
}

type Handler struct {
	svc MovieService
}

type movieDTO struct {
	ID       int    `json:"id"`
	Title    string `json:"title,omitempty"`
	Year     int    `json:"year,omitempty"`
	Director string `json:"director,omitempty"`
	Error    string `json:"error,omitempty"`
}

type errorBody struct {
	Error string `json:"error"`
}

func NewRouter(svc MovieService) http.Handler {
	h := &Handler{svc: svc}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /load-movies", h.loadMovies)
	mux.HandleFunc("GET /last-movie", h.lastMovie)
	registerSwagger(mux)
	return mux
}

func (h *Handler) loadMovies(w http.ResponseWriter, r *http.Request) {
	from, ok := intParam(w, r, "from")
	if !ok {
		return
	}
	to, ok := intParam(w, r, "to")
	if !ok {
		return
	}

	results, err := h.svc.LoadRange(r.Context(), from, to)
	if err != nil {
		writeError(w, err)
		return
	}

	items := make([]movieDTO, 0, len(results))
	failed := 0
	for _, res := range results {
		if res.Err != nil {
			failed++
			items = append(items, movieDTO{ID: res.ID, Error: res.Err.Error()})
			continue
		}
		items = append(items, toDTO(res.Movie))
	}

	if failed == len(results) {
		writeError(w, results[0].Err)
		return
	}
	if failed > 0 {
		w.Header().Set("X-Failed-Count", strconv.Itoa(failed))
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) lastMovie(w http.ResponseWriter, r *http.Request) {
	m, err := h.svc.Last(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toDTO(m))
}

func toDTO(m movies.Movie) movieDTO {
	return movieDTO{ID: m.ID, Title: m.Title, Year: m.Year, Director: m.Director}
}

func intParam(w http.ResponseWriter, r *http.Request, name string) (int, bool) {
	v, err := strconv.Atoi(r.URL.Query().Get(name))
	if err != nil {
		writeJSON(w, http.StatusBadRequest,
			errorBody{Error: "параметр " + name + " обязателен и должен быть целым числом"})
		return 0, false
	}
	return v, true
}

func statusFor(err error) int {
	switch {
	case errors.Is(err, movies.ErrInvalidRange):
		return http.StatusBadRequest
	case errors.Is(err, movies.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, context.DeadlineExceeded):
		return http.StatusGatewayTimeout
	case errors.Is(err, context.Canceled):
		return 499 // клиент сам оборвал запрос
	case errors.Is(err, movies.ErrRateLimited):
		return http.StatusServiceUnavailable
	default:
		return http.StatusBadGateway
	}
}

func writeError(w http.ResponseWriter, err error) {
	writeJSON(w, statusFor(err), errorBody{Error: err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("не удалось записать ответ", "err", err)
	}
}
