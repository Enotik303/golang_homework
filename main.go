package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"
	"time"
)

type Movie struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Year     int    `json:"year"`
	Director string `json:"director"`
}

type result struct {
	id    int
	movie *Movie
	err   error
}

func main() {
	from := flag.Int("from", -1, "id первого фильма")
	to := flag.Int("to", -1, "id последнего фильма")
	workers := flag.Int("workers", 10, "количество воркеров в пуле")
	timeout := flag.Duration("timeout", 5*time.Second, "таймаут одного HTTP-запроса")
	flag.Parse()

	if *from < 0 {
		fatalf("флаг --from обязателен и должен быть >= 0")
	}
	if *to < 0 {
		fatalf("флаг --to обязателен и должен быть >= 0")
	}
	if *from > *to {
		fatalf("--from (%d) не может быть больше --to (%d)", *from, *to)
	}
	if *workers <= 0 {
		fatalf("--workers должен быть положительным числом, получено %d", *workers)
	}
	if *timeout <= 0 {
		fatalf("--timeout должен быть положительным числом, получено %s", *timeout)
	}

	total := *to - *from + 1

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	jobs := make(chan int, total)
	for id := *from; id <= *to; id++ {
		jobs <- id
	}
	close(jobs)

	results := make(chan result, total)

	client := &http.Client{}

	var wg sync.WaitGroup
	for w := 0; w < *workers; w++ {
		wg.Add(1)
		go worker(ctx, client, *timeout, jobs, results, &wg)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	var movies []Movie
	var errCount int
	for r := range results {
		if r.err != nil {
			fmt.Fprintf(os.Stderr, "фильм %d: ошибка: %v\n", r.id, r.err)
			errCount++
			continue
		}
		movies = append(movies, *r.movie)
	}

	sort.Slice(movies, func(i, j int) bool { return movies[i].ID < movies[j].ID })

	fmt.Println("---")
	for _, m := range movies {
		fmt.Printf("%d — %s — %d — %s\n", m.ID, m.Title, m.Year, m.Director)
	}
	fmt.Printf("---\nЗагружено: %d, ошибок: %d, всего запрошено: %d\n", len(movies), errCount, total)
}

func worker(ctx context.Context, client *http.Client, timeout time.Duration, jobs <-chan int, results chan<- result, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case id, ok := <-jobs:
			if !ok {
				return
			}
			movie, err := fetchMovie(ctx, client, id, timeout)
			results <- result{id: id, movie: movie, err: err}
		}
	}
}

func fetchMovie(ctx context.Context, client *http.Client, id int, timeout time.Duration) (*Movie, error) {
	reqCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	url := fmt.Sprintf("https://homeworksite.site/%d/info.0.json", id)
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("не удалось собрать запрос: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("запрос не выполнен: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("неожиданный статус ответа: %d", resp.StatusCode)
	}

	var m Movie
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, fmt.Errorf("битый JSON: %w", err)
	}
	return &m, nil
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "ошибка: "+format+"\n", args...)
	os.Exit(1)
}
