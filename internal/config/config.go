
package config

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	DefaultMovieBaseURL = "https://homeworksite.site"
	DefaultHTTPAddr     = ":8080"
	DefaultGRPCAddr     = ":50051"
	DefaultTimeout      = 5 * time.Second
	DefaultWorkers      = 10
)

type Config struct {
	MovieBaseURL   string        
	HTTPAddr       string        
	GRPCAddr       string        
	RequestTimeout time.Duration 
	Workers        int           
}

func Load() (Config, error) {
	cfg := Config{
		MovieBaseURL:   strings.TrimRight(getenv("MOVIE_BASE_URL", DefaultMovieBaseURL), "/"),
		HTTPAddr:       getenv("HTTP_ADDR", DefaultHTTPAddr),
		GRPCAddr:       getenv("GRPC_ADDR", DefaultGRPCAddr),
		RequestTimeout: DefaultTimeout,
		Workers:        DefaultWorkers,
	}

	if v := os.Getenv("REQUEST_TIMEOUT"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 {
			return Config{}, fmt.Errorf("REQUEST_TIMEOUT: некорректная длительность %q (пример: 5s)", v)
		}
		cfg.RequestTimeout = d
	}
	return cfg, nil
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
