package main

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"movieservice/gen/moviepb"
	"movieservice/internal/adapter/moviestand"
	"movieservice/internal/handler/grpchandler"
	"movieservice/internal/handler/httphandler"
	"movieservice/internal/usecase"
)

func main() {
	if err := run(); err != nil {
		slog.Error("сервис завершился с ошибкой", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := loadConfig()
	if err != nil {
		return err
	}

	provider := moviestand.NewClient(cfg.movieBaseURL, cfg.requestTimeout)
	uc := usecase.NewService(provider, cfg.workers)

	httpSrv := &http.Server{
		Addr:              cfg.httpAddr,
		Handler:           httphandler.NewRouter(uc),
		ReadHeaderTimeout: 5 * time.Second,
	}

	lis, err := net.Listen("tcp", cfg.grpcAddr)
	if err != nil {
		return err
	}
	grpcSrv := grpc.NewServer()
	moviepb.RegisterMovieServiceServer(grpcSrv, grpchandler.NewServer(uc))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 2)
	go func() {
		slog.Info("HTTP-сервер запущен", "addr", cfg.httpAddr)
		if err := httpSrv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()
	go func() {
		slog.Info("gRPC-сервер запущен", "addr", cfg.grpcAddr)
		if err := grpcSrv.Serve(lis); err != nil {
			errCh <- err
		}
	}()

	var runErr error
	select {
	case <-ctx.Done():
		slog.Info("получен сигнал, останавливаем сервис")
	case runErr = <-errCh:
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = httpSrv.Shutdown(shutdownCtx)

	stopped := make(chan struct{})
	go func() {
		grpcSrv.GracefulStop()
		close(stopped)
	}()
	select {
	case <-stopped:
	case <-shutdownCtx.Done():
		grpcSrv.Stop()
	}
	return runErr
}
