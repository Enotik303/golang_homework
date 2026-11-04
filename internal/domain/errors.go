package domain

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidRange = errors.New("некорректный диапазон")
	ErrNotFound     = errors.New("фильм не найден")
	ErrRateLimited  = errors.New("стенд ограничил частоту запросов")
	ErrUpstream     = errors.New("ошибка внешнего стенда")
)

type LoadFailure struct {
	ID  int
	Err error
}

type PartialLoadError struct {
	Failures []LoadFailure
}

func (e *PartialLoadError) Error() string {
	return fmt.Sprintf("часть фильмов не загружена: %d", len(e.Failures))
}
