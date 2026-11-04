package grpchandler

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"movieservice/gen/moviepb"
	"movieservice/internal/domain"
	"movieservice/internal/usecase"
)

type Server struct {
	moviepb.UnimplementedMovieServiceServer
	uc usecase.MovieUseCase
}

func NewServer(uc usecase.MovieUseCase) *Server {
	return &Server{uc: uc}
}

func (s *Server) LoadMovies(ctx context.Context, req *moviepb.LoadMoviesRequest) (*moviepb.LoadMoviesResponse, error) {
	movies, err := s.uc.LoadMovies(ctx, int(req.GetFrom()), int(req.GetTo()))

	var partial *domain.PartialLoadError
	if err != nil && !errors.As(err, &partial) {
		return nil, toStatus(err)
	}

	resp := &moviepb.LoadMoviesResponse{}
	for _, m := range movies {
		resp.Movies = append(resp.Movies, toProto(m))
	}
	if partial != nil {
		for _, f := range partial.Failures {
			resp.Errors = append(resp.Errors, &moviepb.LoadError{Id: int32(f.ID), Message: f.Err.Error()})
		}
	}
	return resp, nil
}

func (s *Server) GetLastMovie(ctx context.Context, _ *moviepb.GetLastMovieRequest) (*moviepb.Movie, error) {
	m, err := s.uc.GetLastMovie(ctx)
	if err != nil {
		return nil, toStatus(err)
	}
	return toProto(m), nil
}

func toProto(m domain.Movie) *moviepb.Movie {
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
	case errors.Is(err, domain.ErrInvalidRange):
		code = codes.InvalidArgument
	case errors.Is(err, domain.ErrNotFound):
		code = codes.NotFound
	case errors.Is(err, context.DeadlineExceeded):
		code = codes.DeadlineExceeded
	case errors.Is(err, context.Canceled):
		code = codes.Canceled
	case errors.Is(err, domain.ErrRateLimited):
		code = codes.ResourceExhausted
	}
	return status.Error(code, err.Error())
}
