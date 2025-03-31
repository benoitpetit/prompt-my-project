package worker

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Pool represents a pool of workers processing files
type Pool struct {
	workerCount int
	jobs        chan Job
	results     chan Result
	wg          sync.WaitGroup
}

// Job represents a single job to be processed by a worker
type Job struct {
	Index    int
	FilePath string
	RootDir  string
}

// Result represents the result of processing a single job
type Result struct {
	Index   int
	Content string
	Err     error
}

// RetryConfig defines configuration for retry attempts
type RetryConfig struct {
	MaxRetries  int
	RetryDelay  time.Duration
	MaxFileSize int64 // Maximum file size for retries
}

// DefaultRetryConfig provides sensible default retry settings
var DefaultRetryConfig = RetryConfig{
	MaxRetries:  3,
	RetryDelay:  500 * time.Millisecond,
	MaxFileSize: 10 * 1024 * 1024, // 10MB max for retries
}

// NewPool creates a new worker pool
func NewPool(workerCount int) *Pool {
	return &Pool{
		workerCount: workerCount,
		jobs:        make(chan Job, workerCount*2),
		results:     make(chan Result, workerCount*2),
	}
}

// Start starts the worker pool
func (wp *Pool) Start() {
	for i := 0; i < wp.workerCount; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}
}

// worker processes jobs
func (wp *Pool) worker() {
	defer wp.wg.Done()
	for job := range wp.jobs {
		content, err := wp.processFile(job.RootDir, job.FilePath, nil)
		wp.results <- Result{
			Index:   job.Index,
			Content: content,
			Err:     err,
		}
	}
}

// processFile processes a single file and returns its formatted content
func (wp *Pool) processFile(rootDir, relPath string, buffer []byte) (string, error) {
	absPath := filepath.Join(rootDir, relPath)

	var content []byte
	var err error

	if buffer != nil {
		content = buffer
	} else {
		content, err = os.ReadFile(absPath)
		if err != nil {
			return "", fmt.Errorf("error reading file %s: %w", relPath, err)
		}
	}

	// Format the output
	builder := fmt.Sprintf("File: %s\n", relPath)
	builder += "================================================\n"
	builder += string(content)
	builder += "\n\n"

	return builder, nil
}

// Stop stops the worker pool and waits for all workers to finish
func (wp *Pool) Stop() {
	close(wp.jobs)
	wp.wg.Wait()
	close(wp.results)
}

// GetJobs returns the jobs channel
func (wp *Pool) GetJobs() chan<- Job {
	return wp.jobs
}

// GetResults returns the results channel
func (wp *Pool) GetResults() <-chan Result {
	return wp.results
}

// ProcessFileWithRetry attempts to process a file with multiple retries if needed
func (wp *Pool) ProcessFileWithRetry(rootDir, relPath string, buffer []byte, retryConfig RetryConfig) (string, error) {
	var lastError error

	// Check file size before proceeding
	absPath := filepath.Join(rootDir, relPath)
	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("error accessing file %s: %w", relPath, err)
	}

	// Skip large files for retry
	if info.Size() > retryConfig.MaxFileSize {
		return "", fmt.Errorf("file too large for retry: %s (size: %d)", relPath, info.Size())
	}

	// Try processing with retries
	for attempt := 0; attempt <= retryConfig.MaxRetries; attempt++ {
		content, err := wp.processFile(rootDir, relPath, buffer)
		if err == nil {
			return content, nil
		}

		lastError = err

		// Wait before retrying
		if attempt < retryConfig.MaxRetries {
			time.Sleep(retryConfig.RetryDelay * time.Duration(attempt+1))
		}
	}

	return "", fmt.Errorf("failed after %d attempts: %w", retryConfig.MaxRetries, lastError)
}
