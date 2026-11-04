package main

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	defaultMovieBaseURL   = "https://homeworksite.site"
	defaultHTTPAddr       = ":8080"
	defaultGRPCAddr       = ":50051"
	defaultRequestTimeout = 5 * time.Second
	defaultWorkers        = 10
)

type config struct {
	movieBaseURL   string        // MOVIE_BASE_URL
	httpAddr       string        // HTTP_ADDR
	grpcAddr       string        // GRPC_ADDR (необязательная)
	requestTimeout time.Duration // REQUEST_TIMEOUT (необязательная)
	workers        int
}

func loadConfig() (config, error) {
	cfg := config{
		movieBaseURL:   strings.TrimRight(getenv("MOVIE_BASE_URL", defaultMovieBaseURL), "/"),
		httpAddr:       getenv("HTTP_ADDR", defaultHTTPAddr),
		grpcAddr:       getenv("GRPC_ADDR", defaultGRPCAddr),
		requestTimeout: defaultRequestTimeout,
		workers:        defaultWorkers,
	}

	if v := os.Getenv("REQUEST_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 {
			return config{}, fmt.Errorf("REQUEST_TIMEOUT: некорректная длительность %q (пример: 5s)", v)
		}
		cfg.requestTimeout = d
	}
	return cfg, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
