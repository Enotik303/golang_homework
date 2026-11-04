package httphandler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sort"
	"strconv"

	"movieservice/internal/domain"
	"movieservice/internal/usecase"
)

type Handler struct {
	uc usecase.MovieUseCase
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

func NewRouter(uc usecase.MovieUseCase) http.Handler {
	h := &Handler{uc: uc}

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

	movies, err := h.uc.LoadMovies(r.Context(), from, to)

	var partial *domain.PartialLoadError
	if err != nil && !errors.As(err, &partial) {
		writeError(w, err)
		return
	}

	items := make([]movieDTO, 0, len(movies))
	for _, m := range movies {
		items = append(items, toDTO(m))
	}
	if partial != nil {
		for _, f := range partial.Failures {
			items = append(items, movieDTO{ID: f.ID, Error: f.Err.Error()})
		}
		sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
		w.Header().Set("X-Failed-Count", strconv.Itoa(len(partial.Failures)))
	}
	writeJSON(w, http.StatusOK, items)
}

func (h *Handler) lastMovie(w http.ResponseWriter, r *http.Request) {
	m, err := h.uc.GetLastMovie(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toDTO(m))
}

func toDTO(m domain.Movie) movieDTO {
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
	case errors.Is(err, domain.ErrInvalidRange):
		return http.StatusBadRequest
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, context.DeadlineExceeded):
		return http.StatusGatewayTimeout
	case errors.Is(err, context.Canceled):
		return 499
	case errors.Is(err, domain.ErrRateLimited):
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
