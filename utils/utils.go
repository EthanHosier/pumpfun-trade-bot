package utils

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

func PriceInSol(virtualSolReserves int64, virtualTokenReserves int64) float64 {
	return float64(virtualSolReserves) / float64(virtualTokenReserves)
}
