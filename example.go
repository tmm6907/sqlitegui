package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
)

type HTTPStatusCode int

type Numeric interface{ int | uint | float32 | float64 }

func Add[T Numeric](a T, b T) T {
	return a + b
}

const (
	HTTP_INTERNAL_SERVER_ERROR = http.StatusInternalServerError
	HTTP_BAD_GATEWAY           = http.StatusBadGateway
)

type ServerError struct {
	Status HTTPStatusCode
	Err    error
}

func (e *ServerError) Error() string {
	return e.Err.Error()
}

func error_handler(ctx context.Context, errChan <-chan *ServerError) {
	for {
		select {
		case err := <-errChan:
			slog.Error(err.Error())
		case <-ctx.Done():
			slog.Info("Received shutdown signal. Error handler stopping gracefully.")
			return
		}
	}
}

func example_main() {
	errChan := make(chan *ServerError)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		error_handler(ctx, errChan)
	}()
	sendError(errChan)
}

func sendError(errChan chan<- *ServerError) {
	if true {
		errChan <- &ServerError{http.StatusInternalServerError, errors.New("example error")}
	}
}
