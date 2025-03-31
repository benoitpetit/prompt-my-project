package analyzer

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/benoitpetit/prompt-my-project/pkg/binary"
	"github.com/benoitpetit/prompt-my-project/pkg/formatter"
	"github.com/benoitpetit/prompt-my-project/pkg/progress"
	"github.com/benoitpetit/prompt-my-project/pkg/utils"
	"github.com/benoitpetit/prompt-my-project/pkg/worker"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/dustin/go-humanize"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
)

// ProjectAnalyzer analyzes a project directory
type ProjectAnalyzer struct {
	Dir             string
	IncludePatterns []string
	ExcludePatterns []string
	MinSize         int64
	MaxSize         int64
	MaxFiles        int
	MaxTotalSize    int64
	WorkerCount     int
	Files           []string
	TotalSize       int64
	TokenCount      int
	CharCount       int
	BinaryCache     *binary.Cache
}

// StatsResult represents statistics from project analysis
type StatsResult struct {
	FileCount    int
	TotalSize    int64
	TokenCount   int
	CharCount    int
	ProcessTime  time.Duration
	FilesPerSec  float64
	OutputPath   string
	Technologies []string
	KeyFiles     []string
	Issues       []string
	FileTypes    map[string]int
}

// New creates a new project analyzer
func New(
	dir string,
	includePatterns, excludePatterns []string,
	minSize, maxSize int64,
	maxFiles int,
	maxTotalSize int64,
	workerCount int,
) *ProjectAnalyzer {
	return &ProjectAnalyzer{
		Dir:             dir,
		IncludePatterns: includePatterns,
		ExcludePatterns: excludePatterns,
		MinSize:         minSize,
		MaxSize:         maxSize,
		MaxFiles:        maxFiles,
		MaxTotalSize:    maxTotalSize,
		WorkerCount:     workerCount,
		Files:           []string{},
		TotalSize:       0,
		TokenCount:      0,
		CharCount:       0,
		BinaryCache:     binary.NewCache(),
	}
}

// CollectFiles gathers files to be processed
func (pa *ProjectAnalyzer) CollectFiles() error {
	// Load binary cache
	if err := pa.BinaryCache.Load(); err != nil {
		fmt.Printf("Warning: error loading binary cache: %v\n", err)
	}

	// Collect files
	files, err := pa.collectFiles()
	if err != nil {
		return fmt.Errorf("error collecting files: %w", err)
	}

	// Limit the number of files
	if pa.MaxFiles > 0 && len(files) > pa.MaxFiles {
		fmt.Printf("Limiting to %d files (from %d total)\n", pa.MaxFiles, len(files))
		files = files[:pa.MaxFiles]
	}

	// Calculate total size
	var totalSize int64
	for _, file := range files {
		info, err := os.Stat(filepath.Join(pa.Dir, file))
		if err == nil {
			totalSize += info.Size()
		}
	}

	// Check total size limit
	if pa.MaxTotalSize > 0 && totalSize > pa.MaxTotalSize {
		fmt.Printf("Warning: Total size exceeds limit: %s > %s\n",
			humanize.Bytes(uint64(totalSize)),
			humanize.Bytes(uint64(pa.MaxTotalSize)))

		// Sort files by size and keep only those that fit the limit
		// This would require more implementation details to properly sort
		// and select files based on size
	}

	pa.Files = files
	return nil
}

// collectFiles collects files that match criteria
func (pa *ProjectAnalyzer) collectFiles() ([]string, error) {
	var result []string
	var totalSize int64
	var matcher gitignore.Matcher

	// Setup gitignore matcher if needed
	if len(pa.ExcludePatterns) > 0 {
		patterns := make([]gitignore.Pattern, 0, len(pa.ExcludePatterns))
		for _, pattern := range pa.ExcludePatterns {
			patterns = append(patterns, gitignore.ParsePattern(pattern, nil))
		}
		matcher = gitignore.NewMatcher(patterns)
	}

	// Create a progress bar
	pg := progress.New(0)
	pg.SetDescription("Collecting files")
	fileCount := 0

	// Walk the directory tree
	err := filepath.WalkDir(pa.Dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Get relative path for matching
		relPath, err := filepath.Rel(pa.Dir, path)
		if err != nil {
			return nil
		}

		// Skip directories
		if d.IsDir() {
			fileCount += 10 // Rough estimate for progress bar
			pg.Update(fileCount)

			// Check exclude patterns for directories
			if matcher != nil && matcher.Match(strings.Split(relPath, string(filepath.Separator)), d.IsDir()) {
				return filepath.SkipDir
			}
			return nil
		}

		// Update progress
		fileCount++
		if fileCount%100 == 0 {
			pg.Update(fileCount)
		}

		// Skip files that don't match include patterns
		if len(pa.IncludePatterns) > 0 {
			isIncluded := false
			for _, pattern := range pa.IncludePatterns {
				match, err := doublestar.Match(pattern, relPath)
				if err == nil && match {
					isIncluded = true
					break
				}
			}
			if !isIncluded {
				return nil
			}
		}

		// Skip files that match exclude patterns
		if matcher != nil && matcher.Match(strings.Split(relPath, string(filepath.Separator)), false) {
			return nil
		}

		// Check if file is binary
		if binary.IsBinaryFile(path, pa.BinaryCache) {
			return nil
		}

		// Check file size
		info, err := d.Info()
		if err != nil {
			return nil
		}

		if (pa.MinSize > 0 && info.Size() < pa.MinSize) ||
			(pa.MaxSize > 0 && info.Size() > pa.MaxSize) {
			return nil
		}

		// Add to results
		result = append(result, relPath)
		totalSize += info.Size()

		return nil
	})

	// Complete the progress bar
	pg.Update(fileCount)

	// Save binary cache
	if err := pa.BinaryCache.Save(); err != nil {
		fmt.Printf("Warning: error saving binary cache: %v\n", err)
	}

	return result, err
}

// GenerateProjectStructure creates a string representation of the project structure
func (pa *ProjectAnalyzer) GenerateProjectStructure() (string, error) {
	// Build tree
	root := utils.BuildTree(pa.Files, filepath.Base(pa.Dir))
	return utils.GenerateTreeOutput(root), nil
}

// ProcessFiles processes the files and generates output in the specified format
func (pa *ProjectAnalyzer) ProcessFiles(outputDir string, format string) (StatsResult, error) {
	startTime := time.Now()
	stats := StatsResult{}

	// Create formatter
	fmtr := formatter.NewFormatter(format, outputDir, pa.Dir)

	// Process files with worker pool
	pool := worker.NewPool(pa.WorkerCount)
	pool.Start()

	// Detect technologies and key files
	technologies := detectTechnologies(pa.Files)
	keyFiles := identifyKeyFiles(pa.Files)
	issues := identifyPotentialIssues(pa.Files)
	fileTypes := collectFileExtensions(pa.Files)

	// Setup progress tracking
	pg := progress.New(len(pa.Files))
	pg.SetDescription(fmt.Sprintf("Processing %s files", format))

	// Channel for tracking completion
	done := make(chan struct{})

	// Send jobs
	go func() {
		defer close(done)
		for i, file := range pa.Files {
			pool.GetJobs() <- worker.Job{
				Index:    i,
				FilePath: file,
				RootDir:  pa.Dir,
			}
		}
		pool.Stop()
	}()

	// Collect results
	completed := 0
	var failedFiles int
	tokenEstimator := utils.NewTokenEstimator()

	// Process each file
	for result := range pool.GetResults() {
		if result.Err != nil {
			fmt.Printf("Warning: error processing file: %v\n", result.Err)
			failedFiles++
		} else if result.Content != "" {
			// Add file to format
			filePath := pa.Files[result.Index]
			absPath := filepath.Join(pa.Dir, filePath)
			fileInfo, err := os.Stat(absPath)

			if err == nil {
				content, _ := os.ReadFile(absPath)
				pa.CharCount += len(content)
				tokens, _ := tokenEstimator.EstimateFileTokens(absPath, true)
				pa.TokenCount += tokens

				fmtr.AddFile(formatter.FileInfo{
					Path:     filePath,
					Size:     fileInfo.Size(),
					Content:  string(content),
					Language: detectFileLanguage(filePath),
				})
			}
		}

		completed++
		pg.Update(completed)
	}

	// Set statistics and metadata
	fmtr.SetStatistics(
		len(pa.Files),
		pa.TotalSize,
		pa.TokenCount,
		pa.CharCount,
		time.Since(startTime),
	)
	fmtr.SetTechnologies(technologies)
	fmtr.SetKeyFiles(keyFiles)
	fmtr.SetIssues(issues)
	fmtr.SetFileTypes(fileTypes)

	// Generate project structure
	structure, err := pa.GenerateProjectStructure()
	if err != nil {
		return stats, fmt.Errorf("error generating project structure: %w", err)
	}
	fmtr.SetProjectStructure(structure)

	// Write the formatted output to file
	outputPath, err := fmtr.WriteToFile()
	if err != nil {
		return stats, fmt.Errorf("error writing output file: %w", err)
	}

	// Populate stats result
	stats.FileCount = len(pa.Files)
	stats.TotalSize = pa.TotalSize
	stats.TokenCount = pa.TokenCount
	stats.CharCount = pa.CharCount
	stats.ProcessTime = time.Since(startTime)
	stats.FilesPerSec = float64(len(pa.Files)) / stats.ProcessTime.Seconds()
	stats.OutputPath = outputPath
	stats.Technologies = technologies
	stats.KeyFiles = keyFiles
	stats.Issues = issues
	stats.FileTypes = fileTypes

	return stats, nil
}

// collectFileExtensions collects file extensions and their counts
func collectFileExtensions(files []string) map[string]int {
	extensions := make(map[string]int)

	for _, file := range files {
		ext := filepath.Ext(file)
		if ext == "" {
			ext = "(no extension)"
		}
		extensions[ext]++
	}

	return extensions
}

// detectTechnologies identifies technologies used in the project
func detectTechnologies(files []string) []string {
	// Implementation would be more extensive in a real system
	technologies := make(map[string]bool)

	for _, file := range files {
		switch {
		case strings.HasSuffix(file, ".go"):
			technologies["Go"] = true
		case strings.HasSuffix(file, ".js"):
			technologies["JavaScript"] = true
		case strings.HasSuffix(file, ".ts"):
			technologies["TypeScript"] = true
		case strings.HasSuffix(file, ".py"):
			technologies["Python"] = true
		case strings.HasSuffix(file, ".java"):
			technologies["Java"] = true
		case strings.HasSuffix(file, ".rb"):
			technologies["Ruby"] = true
		case strings.HasSuffix(file, ".php"):
			technologies["PHP"] = true
		case strings.HasSuffix(file, ".cs"):
			technologies["C#"] = true
		case strings.HasSuffix(file, ".c") || strings.HasSuffix(file, ".cpp") || strings.HasSuffix(file, ".h"):
			technologies["C/C++"] = true
		case strings.HasSuffix(file, ".html"):
			technologies["HTML"] = true
		case strings.HasSuffix(file, ".css"):
			technologies["CSS"] = true
		}

		// Special files indicating technologies
		switch filepath.Base(file) {
		case "package.json":
			technologies["Node.js"] = true
		case "go.mod":
			technologies["Go"] = true
		case "requirements.txt":
			technologies["Python"] = true
		case "Gemfile":
			technologies["Ruby"] = true
		case "composer.json":
			technologies["PHP"] = true
		}
	}

	// Convert to slice
	result := make([]string, 0, len(technologies))
	for tech := range technologies {
		result = append(result, tech)
	}

	return result
}

// identifyKeyFiles identifies important files in the project
func identifyKeyFiles(files []string) []string {
	keyFiles := make([]string, 0)

	for _, file := range files {
		basename := filepath.Base(file)
		switch basename {
		case "main.go", "app.js", "index.js", "package.json", "go.mod", "requirements.txt",
			"setup.py", "Makefile", "Dockerfile", "docker-compose.yml", "README.md":
			keyFiles = append(keyFiles, file)
		}
	}

	return keyFiles
}

// identifyPotentialIssues identifies potential issues in the project
func identifyPotentialIssues(files []string) []string {
	// This would be more extensive in a real system
	issues := make([]string, 0)

	// Check for missing README
	hasReadme := false
	for _, file := range files {
		if strings.Contains(strings.ToLower(filepath.Base(file)), "readme") {
			hasReadme = true
			break
		}
	}

	if !hasReadme {
		issues = append(issues, "Missing README file")
	}

	return issues
}

// detectFileLanguage detects the programming language of a file
func detectFileLanguage(filename string) string {
	ext := filepath.Ext(filename)

	switch strings.ToLower(ext) {
	case ".go":
		return "Go"
	case ".js":
		return "JavaScript"
	case ".ts":
		return "TypeScript"
	case ".py":
		return "Python"
	case ".java":
		return "Java"
	case ".rb":
		return "Ruby"
	case ".php":
		return "PHP"
	case ".cs":
		return "C#"
	case ".c":
		return "C"
	case ".cpp", ".cc":
		return "C++"
	case ".h":
		return "C/C++ Header"
	case ".html", ".htm":
		return "HTML"
	case ".css":
		return "CSS"
	case ".json":
		return "JSON"
	case ".xml":
		return "XML"
	case ".md":
		return "Markdown"
	case ".sh":
		return "Shell"
	case ".bat", ".cmd":
		return "Batch"
	case ".sql":
		return "SQL"
	default:
		return "Plain Text"
	}
}
