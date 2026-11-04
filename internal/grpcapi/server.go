
package grpcapi

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"movieservice/gen/moviepb"
	"movieservice/internal/movies"
)

type MovieService interface {
	LoadRange(ctx context.Context, from, to int) ([]movies.Result, error)
	Last(ctx context.Context) (movies.Movie, error)
}

type Server struct {
	moviepb.UnimplementedMovieServiceServer
	svc MovieService
}

func NewServer(svc MovieService) *Server {
	return &Server{svc: svc}
}

func (s *Server) LoadMovies(ctx context.Context, req *moviepb.LoadMoviesRequest) (*moviepb.LoadMoviesResponse, error) {
	results, err := s.svc.LoadRange(ctx, int(req.GetFrom()), int(req.GetTo()))
	if err != nil {
		return nil, toStatus(err)
	}

	resp := &moviepb.LoadMoviesResponse{}
	for _, r := range results {
		if r.Err != nil {
			resp.Errors = append(resp.Errors, &moviepb.LoadError{Id: int32(r.ID), Message: r.Err.Error()})
			continue
		}
		resp.Movies = append(resp.Movies, toProto(r.Movie))
	}

	if len(resp.Movies) == 0 {
		return nil, toStatus(results[0].Err)
	}
	return resp, nil
}

func (s *Server) GetLastMovie(ctx context.Context, _ *moviepb.GetLastMovieRequest) (*moviepb.Movie, error) {
	m, err := s.svc.Last(ctx)
	if err != nil {
		return nil, toStatus(err)
	}
	return toProto(m), nil
}

func toProto(m movies.Movie) *moviepb.Movie {
	return &moviepb.Movie{
		Id:       int32(m.ID),
		Title:    m.Title,
		Year:     int32(m.Year),
		Director: m.Director,
	}
}

func toStatus(err error) error {
	code := codes.Unavailable
	switch {
	case errors.Is(err, movies.ErrInvalidRange):
		code = codes.InvalidArgument
	case errors.Is(err, movies.ErrNotFound):
		code = codes.NotFound
	case errors.Is(err, context.DeadlineExceeded):
		code = codes.DeadlineExceeded
	case errors.Is(err, context.Canceled):
		code = codes.Canceled
	case errors.Is(err, movies.ErrRateLimited):
		code = codes.ResourceExhausted
	}
	return status.Error(code, err.Error())
}
