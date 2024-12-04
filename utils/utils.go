package utils

import (
	"fmt"
	"reflect"
)

type Task[T any] struct {
	ch      chan T
	errorCh chan error
}

func DoAsync[T any](fn func() (T, error)) *Task[T] {
	ch := make(chan T)
	errorCh := make(chan error)

	go func() {
		result, err := fn()
		if err != nil {
			errorCh <- err
			return
		}
		ch <- result
	}()

	return &Task[T]{ch, errorCh}
}

func GetAsync[T any](task *Task[T]) (T, error) {
	var zero T // This will initialize `zero` to the zero value for type T
	select {
	case result := <-task.ch:
		return result, nil
	case err := <-task.errorCh:
		return zero, err
	}
}

func DoAsyncList[T any, U any](items []T, fn func(T) (U, error)) []*Task[U] {
	tasks := make([]*Task[U], len(items))

	for i, item := range items {
		tasks[i] = DoAsync(func() (U, error) {
			return fn(item)
		})
	}

	return tasks
}

func GetAsyncList[T any](tasks []*Task[T]) ([]T, error) {
	results := make([]T, len(tasks))

	for i, task := range tasks {
		result, err := GetAsync(task)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}

	return results, nil
}

func PriceInSol(virtualSolReserves int64, virtualTokenReserves int64) float64 {
	return float64(virtualSolReserves) / float64(virtualTokenReserves)
}

func RemoveDuplicates[T comparable](slice []T, hash func(T) string) []T {
	encountered := map[string]bool{}
	result := []T{}

	for _, v := range slice {
		if !encountered[hash(v)] {
			encountered[hash(v)] = true
			result = append(result, v)
		}
	}
	return result
}

func Required[T any](value T, name string) T {
	if reflect.ValueOf(value).IsZero() {
		panic(fmt.Sprintf("%s is required", name))
	}
	return value
}
