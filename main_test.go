package main

import (
	"context"
	"errors"
	"os"
	"sync"
	"testing"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

var test_app *App

type DialogResult struct {
	Result string
	Err    error
}

// MockDialogService now holds a queue of results.
type MockDialogService struct {
	DirResults  []DialogResult // Dedicated queue for OpenDirectory calls
	FileResults []DialogResult // Dedicated queue for OpenFile calls
	mu          sync.Mutex     // Protects access to Results, important if tests were parallelized
}

func (m *MockDialogService) QueueDirResult(result string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.DirResults = append(m.DirResults, DialogResult{Result: result, Err: err})
}

// QueueFileResult is a setup helper for OpenFile.
func (m *MockDialogService) QueueFileResult(result string, err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.FileResults = append(m.FileResults, DialogResult{Result: result, Err: err})
}

func (m *MockDialogService) OpenDirectory(ctx context.Context, opts runtime.OpenDialogOptions) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.DirResults) == 0 {
		return "", errors.New("mock error: no result was queued for OpenDirectory call")
	}

	// Pop the next result from the queue
	result := m.DirResults[0]
	m.DirResults = m.DirResults[1:]

	return result.Result, result.Err
}

func (m *MockDialogService) OpenFile(ctx context.Context, opts runtime.OpenDialogOptions) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.FileResults) == 0 {
		return "", errors.New("mock error: no result was queued for OpenFile call")
	}

	// Pop the next result from the queue
	result := m.FileResults[0]
	m.FileResults = m.FileResults[1:]

	return result.Result, result.Err
}

func TestMain(m *testing.M) {
	uniqueDBIdentifier := "test_app"
	test_app = NewApp(&CustomAppConfig{
		Logger:        NewSLogger(),
		RootDBName:    uniqueDBIdentifier,
		DialogService: &MockDialogService{},
	})
	test_app.startup(context.Background())

	exitCode := m.Run()

	test_app.shutdown(test_app.ctx)

	os.Exit(exitCode)
}
