package analyzer

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

const (
	// DefaultChunkSize is the default size for file chunks (10 MB)
	DefaultChunkSize = 10 * 1024 * 1024

	// LargeFileThreshold defines what is considered a large file (100 MB)
	LargeFileThreshold = 100 * 1024 * 1024

	// StreamBufferSize is the buffer size for streaming operations
	StreamBufferSize = 64 * 1024
)

// StreamProcessor handles streaming processing of large files
type StreamProcessor struct {
	rootDir    string
	chunkSize  int64
	onChunk    ChunkCallback
	onProgress ProgressCallback
}

// ChunkCallback is called for each processed chunk
type ChunkCallback func(file string, chunk []byte, offset int64, total int64) error

// ProgressCallback reports processing progress
type ProgressCallback func(file string, processed int64, total int64)

// NewStreamProcessor creates a new stream processor
func NewStreamProcessor(rootDir string, chunkSize int64) *StreamProcessor {
	if chunkSize <= 0 {
		chunkSize = DefaultChunkSize
	}

	return &StreamProcessor{
		rootDir:   rootDir,
		chunkSize: chunkSize,
	}
}

// SetChunkCallback sets the callback for chunk processing
func (sp *StreamProcessor) SetChunkCallback(cb ChunkCallback) {
	sp.onChunk = cb
}

// SetProgressCallback sets the callback for progress reporting
func (sp *StreamProcessor) SetProgressCallback(cb ProgressCallback) {
	sp.onProgress = cb
}

// ProcessFile processes a file in streaming mode
func (sp *StreamProcessor) ProcessFile(filePath string) error {
	fullPath := filepath.Join(sp.rootDir, filePath)

	// Get file info
	info, err := os.Stat(fullPath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	fileSize := info.Size()

	// For small files, read normally
	if fileSize < sp.chunkSize {
		return sp.processSmallFile(filePath, fullPath, fileSize)
	}

	// For large files, stream in chunks
	return sp.processLargeFile(filePath, fullPath, fileSize)
}

// processSmallFile processes a small file in one go
func (sp *StreamProcessor) processSmallFile(filePath, fullPath string, size int64) error {
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	if sp.onChunk != nil {
		if err := sp.onChunk(filePath, content, 0, size); err != nil {
			return err
		}
	}

	if sp.onProgress != nil {
		sp.onProgress(filePath, size, size)
	}

	return nil
}

// processLargeFile processes a large file in chunks
func (sp *StreamProcessor) processLargeFile(filePath, fullPath string, size int64) error {
	file, err := os.Open(fullPath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := bufio.NewReaderSize(file, StreamBufferSize)
	buffer := make([]byte, sp.chunkSize)
	offset := int64(0)

	for {
		n, err := reader.Read(buffer)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read chunk: %w", err)
		}

		if n == 0 {
			break
		}

		// Process this chunk
		if sp.onChunk != nil {
			chunk := buffer[:n]
			if err := sp.onChunk(filePath, chunk, offset, size); err != nil {
				return err
			}
		}

		offset += int64(n)

		// Report progress
		if sp.onProgress != nil {
			sp.onProgress(filePath, offset, size)
		}

		if err == io.EOF {
			break
		}
	}

	return nil
}

// StreamingAnalyzer performs streaming analysis for large projects
type StreamingAnalyzer struct {
	rootDir     string
	files       []string
	totalSize   int64
	chunkSize   int64
	onFileStart func(file string, size int64)
	onFileEnd   func(file string, duration time.Duration)
	onChunkData func(file string, data []byte, offset int64)
}

// NewStreamingAnalyzer creates a new streaming analyzer
func NewStreamingAnalyzer(rootDir string, files []string, totalSize int64) *StreamingAnalyzer {
	return &StreamingAnalyzer{
		rootDir:   rootDir,
		files:     files,
		totalSize: totalSize,
		chunkSize: DefaultChunkSize,
	}
}

// SetChunkSize sets the chunk size for streaming
func (sa *StreamingAnalyzer) SetChunkSize(size int64) {
	sa.chunkSize = size
}

// SetFileStartCallback sets callback for when file processing starts
func (sa *StreamingAnalyzer) SetFileStartCallback(cb func(string, int64)) {
	sa.onFileStart = cb
}

// SetFileEndCallback sets callback for when file processing ends
func (sa *StreamingAnalyzer) SetFileEndCallback(cb func(string, time.Duration)) {
	sa.onFileEnd = cb
}

// SetChunkDataCallback sets callback for chunk data
func (sa *StreamingAnalyzer) SetChunkDataCallback(cb func(string, []byte, int64)) {
	sa.onChunkData = cb
}

// AnalyzeWithStreaming performs streaming analysis
func (sa *StreamingAnalyzer) AnalyzeWithStreaming() error {
	processor := NewStreamProcessor(sa.rootDir, sa.chunkSize)

	// Set up chunk callback
	processor.SetChunkCallback(func(file string, chunk []byte, offset int64, total int64) error {
		if sa.onChunkData != nil {
			sa.onChunkData(file, chunk, offset)
		}
		return nil
	})

	// Process each file
	for _, file := range sa.files {
		fullPath := filepath.Join(sa.rootDir, file)
		info, err := os.Stat(fullPath)
		if err != nil {
			continue
		}

		startTime := time.Now()

		if sa.onFileStart != nil {
			sa.onFileStart(file, info.Size())
		}

		if err := processor.ProcessFile(file); err != nil {
			return fmt.Errorf("failed to process file %s: %w", file, err)
		}

		if sa.onFileEnd != nil {
			sa.onFileEnd(file, time.Since(startTime))
		}
	}

	return nil
}

// MemoryEfficientCollector collects and processes files with memory constraints
type MemoryEfficientCollector struct {
	rootDir      string
	maxMemory    int64
	currentUsage int64
	buffer       []byte
	writer       io.Writer
}

// NewMemoryEfficientCollector creates a collector with memory limits
func NewMemoryEfficientCollector(rootDir string, maxMemory int64, writer io.Writer) *MemoryEfficientCollector {
	if maxMemory <= 0 {
		maxMemory = 100 * 1024 * 1024 // 100 MB default
	}

	return &MemoryEfficientCollector{
		rootDir:   rootDir,
		maxMemory: maxMemory,
		buffer:    make([]byte, 0, StreamBufferSize),
		writer:    writer,
	}
}

// AddFile adds a file to the collection, streaming if necessary
func (mec *MemoryEfficientCollector) AddFile(filePath string) error {
	fullPath := filepath.Join(mec.rootDir, filePath)

	info, err := os.Stat(fullPath)
	if err != nil {
		return err
	}

	// If adding this file would exceed memory, flush buffer
	if mec.currentUsage+info.Size() > mec.maxMemory {
		if err := mec.Flush(); err != nil {
			return err
		}
	}

	// Stream large files directly
	if info.Size() > LargeFileThreshold {
		return mec.streamFileDirect(filePath, fullPath)
	}

	// Buffer small files
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return err
	}

	mec.buffer = append(mec.buffer, content...)
	mec.currentUsage += int64(len(content))

	return nil
}

// streamFileDirect streams a file directly to writer
func (mec *MemoryEfficientCollector) streamFileDirect(relPath, fullPath string) error {
	file, err := os.Open(fullPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write file header
	header := fmt.Sprintf("\n=== %s ===\n", relPath)
	if _, err := mec.writer.Write([]byte(header)); err != nil {
		return err
	}

	// Stream content
	reader := bufio.NewReaderSize(file, StreamBufferSize)
	buffer := make([]byte, StreamBufferSize)

	for {
		n, err := reader.Read(buffer)
		if err != nil && err != io.EOF {
			return err
		}

		if n == 0 {
			break
		}

		if _, writeErr := mec.writer.Write(buffer[:n]); writeErr != nil {
			return writeErr
		}

		if err == io.EOF {
			break
		}
	}

	return nil
}

// Flush writes buffered content to writer
func (mec *MemoryEfficientCollector) Flush() error {
	if len(mec.buffer) == 0 {
		return nil
	}

	if _, err := mec.writer.Write(mec.buffer); err != nil {
		return err
	}

	mec.buffer = mec.buffer[:0]
	mec.currentUsage = 0

	return nil
}

// StreamingStats tracks statistics during streaming
type StreamingStats struct {
	FilesProcessed  int
	BytesProcessed  int64
	ChunksProcessed int
	StartTime       time.Time
	CurrentFile     string
	LargeFiles      []string
	Errors          []string
}

// NewStreamingStats creates new streaming statistics
func NewStreamingStats() *StreamingStats {
	return &StreamingStats{
		StartTime:  time.Now(),
		LargeFiles: make([]string, 0),
		Errors:     make([]string, 0),
	}
}

// RecordFile records a processed file
func (ss *StreamingStats) RecordFile(file string, size int64, isLarge bool) {
	ss.FilesProcessed++
	ss.BytesProcessed += size
	ss.CurrentFile = file

	if isLarge {
		ss.LargeFiles = append(ss.LargeFiles, file)
	}
}

// RecordChunk records a processed chunk
func (ss *StreamingStats) RecordChunk() {
	ss.ChunksProcessed++
}

// RecordError records an error
func (ss *StreamingStats) RecordError(file string, err error) {
	ss.Errors = append(ss.Errors, fmt.Sprintf("%s: %v", file, err))
}

// GetElapsedTime returns elapsed time
func (ss *StreamingStats) GetElapsedTime() time.Duration {
	return time.Since(ss.StartTime)
}

// GetThroughput returns bytes per second
func (ss *StreamingStats) GetThroughput() float64 {
	elapsed := ss.GetElapsedTime().Seconds()
	if elapsed == 0 {
		return 0
	}
	return float64(ss.BytesProcessed) / elapsed
}

// GetSummary returns a human-readable summary
func (ss *StreamingStats) GetSummary() string {
	elapsed := ss.GetElapsedTime()
	throughput := ss.GetThroughput()

	summary := fmt.Sprintf("Streaming Analysis Summary:\n")
	summary += fmt.Sprintf("  Files Processed: %d\n", ss.FilesProcessed)
	summary += fmt.Sprintf("  Bytes Processed: %d (%.2f MB)\n", ss.BytesProcessed, float64(ss.BytesProcessed)/(1024*1024))
	summary += fmt.Sprintf("  Chunks Processed: %d\n", ss.ChunksProcessed)
	summary += fmt.Sprintf("  Large Files: %d\n", len(ss.LargeFiles))
	summary += fmt.Sprintf("  Elapsed Time: %v\n", elapsed)
	summary += fmt.Sprintf("  Throughput: %.2f MB/s\n", throughput/(1024*1024))

	if len(ss.Errors) > 0 {
		summary += fmt.Sprintf("\nErrors (%d):\n", len(ss.Errors))
		for i, err := range ss.Errors {
			if i >= 5 {
				summary += fmt.Sprintf("  ... and %d more\n", len(ss.Errors)-5)
				break
			}
			summary += fmt.Sprintf("  - %s\n", err)
		}
	}

	return summary
}

// IsLargeProject determines if a project should use streaming
func IsLargeProject(totalSize int64, fileCount int) bool {
	// Use streaming if:
	// - Total size > 500 MB
	// - File count > 10,000
	// - Average file size > 50 KB and count > 5,000
	if totalSize > 500*1024*1024 {
		return true
	}

	if fileCount > 10000 {
		return true
	}

	if fileCount > 5000 && totalSize/int64(fileCount) > 50*1024 {
		return true
	}

	return false
}
