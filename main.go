package main

import (
	"fmt"
	"io"
	"io/fs"
	"math"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bmatcuk/doublestar/v4"
	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/go-git/go-git/v5/plumbing/format/gitignore"
	"github.com/urfave/cli/v2"
)

// Global configuration
const (
	Version   = "1.0.0"
	GoVersion = "1.21"
)

// Default configuration
var DefaultConfig = struct {
	MinSize   string
	MaxSize   string
	OutputDir string
	GitIgnore bool
}{
	MinSize:   "1KB",
	MaxSize:   "100MB",
	OutputDir: "prompts",
	GitIgnore: true,
}

// List of extensions to exclude
var binaryExtensions = map[string]bool{
	".exe": true, ".dll": true, ".so": true, ".dylib": true,
	".bin": true, ".obj": true, ".o": true, ".a": true,
	".lib": true, ".pyc": true, ".pyo": true, ".pyd": true,
	".class": true, ".jar": true, ".war": true, ".ear": true,
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".bmp": true, ".ico": true, ".zip": true, ".tar": true,
	".gz": true, ".rar": true, ".7z": true, ".pdf": true,
}

func isBinaryFile(filepath string) bool {
	// Check extension
	ext := strings.ToLower(path.Ext(filepath))
	if binaryExtensions[ext] {
		return true
	}

	// Check MIME type
	mimeType := mime.TypeByExtension(ext)
	if mimeType != "" {
		return !strings.HasPrefix(mimeType, "text/") &&
			!strings.Contains(mimeType, "application/json") &&
			!strings.Contains(mimeType, "application/xml")
	}

	// Read first bytes to detect null characters
	file, err := os.Open(filepath)
	if err != nil {
		return true // When in doubt, consider as binary
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil {
		return true
	}

	return http.DetectContentType(buffer[:n]) != "text/plain; charset=utf-8"
}

type Directory struct {
	Name    string
	SubDirs map[string]*Directory
	Files   []string
}

// SimpleProgress represents a simple progress bar with mutex
type SimpleProgress struct {
	sync.Mutex
	total       int
	current     int
	lastWidth   int
	width       int
	startTime   time.Time
	lastUpdate  time.Time
	completed   bool
	description string
}

func NewSimpleProgress(total int) *SimpleProgress {
	return &SimpleProgress{
		total:       total,
		current:     0,
		lastWidth:   0,
		width:       40, // Largeur fixe de la barre à 40 caractères
		startTime:   time.Now(),
		lastUpdate:  time.Now(),
		completed:   false,
		description: "Processing files",
	}
}

// SetDescription permet de définir le texte descriptif de la progression
func (p *SimpleProgress) SetDescription(desc string) {
	p.Lock()
	defer p.Unlock()
	p.description = desc
}

func (p *SimpleProgress) Update(current int) {
	p.Lock()
	defer p.Unlock()

	// Ne pas mettre à jour trop souvent pour les gros fichiers (limiter à 10 rafraîchissements/sec)
	now := time.Now()
	if current < p.total && now.Sub(p.lastUpdate).Milliseconds() < 100 {
		return
	}
	p.lastUpdate = now

	// Mettre à jour le compteur
	p.current = current
	percent := float64(current) / float64(p.total)
	if current >= p.total && !p.completed {
		p.completed = true
	}

	// Effacer entièrement la ligne précédente
	if p.lastWidth > 0 {
		fmt.Print("\r")
		fmt.Print(strings.Repeat(" ", p.lastWidth))
		fmt.Print("\r")
	}

	// Couleurs et styles
	boxColor := color.New(color.FgHiBlue)
	fillColor := color.New(color.FgGreen)
	emptyColor := color.New(color.FgHiBlack)
	textColor := color.New(color.FgHiWhite)

	// Construire l'affichage de la barre
	filled := int(float64(p.width) * percent)

	// Barre avec bordures améliorées
	bar := boxColor.Sprint("┃")
	bar += fillColor.Sprint(strings.Repeat("█", filled))
	if filled < p.width {
		bar += emptyColor.Sprint(strings.Repeat("░", p.width-filled))
	}
	bar += boxColor.Sprint("┃")

	// Informations de progression
	progressText := textColor.Sprintf(" %d/%d (%d%%)", current, p.total, int(percent*100))

	// Afficher la barre et le texte
	fmt.Print("\r" + bar + progressText)
	p.lastWidth = len(bar + progressText) + 20 // Marge pour les codes ANSI

	// Ajouter un saut de ligne quand terminé
	if p.completed {
		fmt.Println()
	}
}

// formatDuration permet un affichage lisible des durées
func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)

	if d.Hours() >= 1 {
		h := int(d.Hours())
		m := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh%dm", h, m)
	} else if d.Minutes() >= 1 {
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", m, s)
	} else {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
}

// Add a constant for default exclude patterns
var defaultExcludes = []string{
	".git/",
	".git/*",
	"node_modules/",
	"vendor/",
	"*.exe",
	"*.dll",
	"*.so",
	"*.dylib",
}

// Cache for already processed files
type FileCache struct {
	sync.RWMutex
	cache map[string]string
}

func NewFileCache() *FileCache {
	return &FileCache{
		cache: make(map[string]string),
	}
}

func (fc *FileCache) Get(key string) (string, bool) {
	fc.RLock()
	defer fc.RUnlock()
	val, ok := fc.cache[key]
	return val, ok
}

func (fc *FileCache) Set(key, value string) {
	fc.Lock()
	defer fc.Unlock()
	fc.cache[key] = value
}

var fileCache = NewFileCache()

// WorkerPool manages a pool of workers for concurrent processing
type WorkerPool struct {
	workerCount int
	jobs        chan WorkerJob
	results     chan WorkerResult
	wg          sync.WaitGroup
}

type WorkerJob struct {
	index    int
	filePath string
	rootDir  string
}

type WorkerResult struct {
	index   int
	content string
	err     error
}

func NewWorkerPool(workerCount int) *WorkerPool {
	return &WorkerPool{
		workerCount: workerCount,
		jobs:        make(chan WorkerJob, workerCount*2),
		results:     make(chan WorkerResult, workerCount*2),
	}
}

func (wp *WorkerPool) Start() {
	for i := 0; i < wp.workerCount; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}
}

func (wp *WorkerPool) worker() {
	defer wp.wg.Done()

	// Buffer reuse to optimize memory
	buffer := make([]byte, 32*1024)

	for job := range wp.jobs {
		content, err := wp.processFile(job.rootDir, job.filePath, buffer)
		wp.results <- WorkerResult{
			index:   job.index,
			content: content,
			err:     err,
		}
	}
}

func (wp *WorkerPool) processFile(rootDir, relPath string, buffer []byte) (string, error) {
	// Check cache
	if cached, ok := fileCache.Get(relPath); ok {
		return cached, nil
	}

	absPath := filepath.Join(rootDir, relPath)
	file, err := os.Open(absPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var output strings.Builder
	output.WriteString("\n================================================\n")
	output.WriteString(fmt.Sprintf("File: %s\n", relPath))
	output.WriteString("================================================\n")

	// Using reusable buffer for reading
	for {
		n, err := file.Read(buffer)
		if n > 0 {
			output.Write(buffer[:n])
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return "", err
		}
	}
	output.WriteString("\n")

	result := output.String()
	fileCache.Set(relPath, result)
	return result, nil
}

func (wp *WorkerPool) Stop() {
	close(wp.jobs)
	wp.wg.Wait()
	close(wp.results)
}

var (
	successColor = color.New(color.FgGreen, color.Bold)
	errorColor   = color.New(color.FgRed, color.Bold)
	infoColor    = color.New(color.FgCyan)
	headerColor  = color.New(color.FgBlue, color.Bold)
)

type ProcessStats struct {
	startTime  time.Time
	duration   time.Duration
	fileCount  int
	totalSize  int64
	tokenCount int
	charCount  int
	outputPath string
}

func main() {
	app := &cli.App{
		Name:    "pmp",
		Usage:   "Generate a project prompt for AI",
		Version: Version,
		Description: `Prompt My Project (PMP) analyzes your project and generates a formatted
prompt for AI assistants. It allows excluding binary files, respecting .gitignore rules,
and offers advanced filtering options.

Usage examples:
   pmp /path/to/project          # Analyze the specified project
   pmp . --include "*.go"        # Analyze only .go files in the current project
   pmp /path/project --exclude "test/*"  # Exclude files in the test folder
   pmp /path/project --output ~/prompts  # Specify the output directory`,
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:    "exclude",
				Aliases: []string{"e"},
				Usage:   "Exclude files matching these patterns (e.g., *.md, src/)",
			},
			&cli.StringSliceFlag{
				Name:    "include",
				Aliases: []string{"i"},
				Usage:   "Include only files matching these patterns",
			},
			&cli.StringFlag{
				Name:  "min-size",
				Usage: "Minimum file size (e.g., 1KB, 500B)",
				Value: DefaultConfig.MinSize,
			},
			&cli.StringFlag{
				Name:  "max-size",
				Usage: "Maximum file size (e.g., 100MB, 1GB)",
				Value: DefaultConfig.MaxSize,
			},
			&cli.BoolFlag{
				Name:  "no-gitignore",
				Usage: "Ignore .gitignore file",
				Value: !DefaultConfig.GitIgnore,
			},
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output directory for the prompt file",
				Value:   DefaultConfig.OutputDir,
			},
		},
		Action: func(c *cli.Context) error {
			// If no argument is provided, display help and exit
			if !c.Args().Present() {
				return cli.ShowAppHelp(c)
			}

			// If an argument is provided, proceed with the analysis
			startTime := time.Now()
			var totalSize int64

			dir := c.Args().First()

			// Convertir en chemin absolu pour garantir la cohérence
			if !filepath.IsAbs(dir) {
				absDir, err := filepath.Abs(dir)
				if err == nil {
					dir = absDir
				}
			}

			minSize, err := parseSize(c.String("min-size"))
			if err != nil {
				return fmt.Errorf("invalid min-size: %w", err)
			}

			maxSize, err := parseSize(c.String("max-size"))
			if err != nil {
				return fmt.Errorf("invalid max-size: %w", err)
			}

			// Merge default excludes with user excludes and gitignore patterns
			excludePatterns := c.StringSlice("exclude")
			excludePatterns = append(excludePatterns, defaultExcludes...)

			if !c.Bool("no-gitignore") {
				patterns, err := loadGitignorePatterns(dir)
				if err != nil {
					fmt.Printf("Warning: error loading .gitignore: %v\n", err)
				} else if patterns != nil {
					excludePatterns = append(excludePatterns, patterns...)
				}
			}

			files, err := collectFiles(dir, c.StringSlice("include"), excludePatterns, minSize, maxSize)
			if err != nil {
				return fmt.Errorf("error collecting files: %w", err)
			}

			// Calculate the total size of files
			for _, file := range files {
				if info, err := os.Stat(filepath.Join(dir, file)); err == nil {
					totalSize += info.Size()
				}
			}

			// Create tree and structure
			rootName := filepath.Base(dir)
			tree := buildTree(files, rootName)
			structure := generateTreeOutput(tree)

			// contentBuilder variable to store the content of files
			var contentBuilder strings.Builder

			// Initialize the worker pool
			numWorkers := runtime.GOMAXPROCS(0) * 2
			pool := NewWorkerPool(numWorkers)
			pool.Start()

			// Send jobs
			go func() {
				for i, file := range files {
					pool.jobs <- WorkerJob{
						index:    i,
						filePath: file,
						rootDir:  dir,
					}
				}
				pool.Stop()
			}()

			// Collect results
			results := make([]string, len(files))
			progress := NewSimpleProgress(len(files))
			progress.SetDescription("Processing files")
			completed := 0

			for result := range pool.results {
				if result.err != nil {
					// Sauvegarder la position actuelle
					if progress.lastWidth > 0 {
						fmt.Print("\r")
						fmt.Print(strings.Repeat(" ", progress.lastWidth))
						fmt.Print("\r")
					}
					fmt.Printf("Warning: error processing file: %v\n", result.err)
				}
				results[result.index] = result.content
				completed++
				progress.Update(completed)
			}

			// Le saut de ligne est déjà géré par la barre de progression quand elle est terminée

			// Write results in order
			for _, content := range results {
				if content != "" {
					contentBuilder.WriteString(content)
				}
			}

			// File contents
			promptContent := contentBuilder.String()

			// Calculate tokens
			tokenCount := estimateTokenCount(promptContent)

			// Building the enhanced final prompt
			var output strings.Builder

			// Generate the header with stats at the top
			header := generatePromptHeader(files, totalSize, tokenCount, len(promptContent), dir)
			output.WriteString(header)

			// Add project structure
			output.WriteString("PROJECT STRUCTURE:\n")
			output.WriteString("-----------------------------------------------------\n\n")
			output.WriteString(structure)

			// Add file contents
			output.WriteString("\n\nFILE CONTENTS:\n")
			output.WriteString("-----------------------------------------------------\n")
			output.WriteString(promptContent)

			finalPrompt := output.String()

			// Determine output directory
			outputDir := c.String("output")
			// fmt.Printf("Debug - Project directory: %s\n", dir)
			if outputDir == "" {
				// By default, create a "prompts" folder in the analyzed project
				outputDir = filepath.Join(dir, "prompts")
				// fmt.Printf("Debug - Output directory (default): %s\n", outputDir)
			} else {
				// Si un chemin relatif est fourni, le rendre absolu par rapport au projet analysé
				if !filepath.IsAbs(outputDir) {
					outputDir = filepath.Join(dir, outputDir)
				}
				// fmt.Printf("Debug - Output directory (specified): %s\n", outputDir)
			}

			// Check if output directory is inside the analyzed project
			// If so, add it to .gitignore
			relPath, err := filepath.Rel(dir, outputDir)
			if err == nil && !strings.HasPrefix(relPath, "..") {
				// Output dir is inside the project, add to .gitignore
				gitignoreEntry := strings.TrimPrefix(relPath, string(filepath.Separator))
				if err := ensureGitignoreEntry(dir, gitignoreEntry); err != nil {
					fmt.Printf("Warning: unable to update .gitignore: %v\n", err)
				}
			}

			// Create output directory if it doesn't exist
			if err := os.MkdirAll(outputDir, 0755); err != nil {
				return fmt.Errorf("error creating output directory: %w", err)
			}

			timestamp := time.Now().Format("20060102_150405")
			fileName := fmt.Sprintf("prompt_%s.txt", timestamp)
			outputPath := filepath.Join(outputDir, fileName)

			if err := os.WriteFile(outputPath, []byte(finalPrompt), 0644); err != nil {
				return fmt.Errorf("error writing output file: %w", err)
			}

			// fmt.Printf("Final prompt output path: %s\n", outputPath)

			stats := ProcessStats{
				startTime:  startTime,
				duration:   time.Since(startTime),
				fileCount:  len(files),
				totalSize:  totalSize,
				tokenCount: tokenCount,
				charCount:  len(promptContent),
				outputPath: outputPath,
			}

			printStatistics(stats)
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		errorColor.Fprintf(os.Stderr, "❌ Error: %v\n", err)
		os.Exit(1)
	}
}

func parseSize(sizeStr string) (int64, error) {
	sizeStr = strings.TrimSpace(sizeStr)
	if len(sizeStr) == 0 {
		return 0, fmt.Errorf("empty size string")
	}

	i := 0
	for ; i < len(sizeStr); i++ {
		c := sizeStr[i]
		if c < '0' || c > '9' {
			break
		}
	}

	if i == 0 {
		return 0, fmt.Errorf("no numeric part in size: %s", sizeStr)
	}

	numericPart := sizeStr[:i]
	unitPart := strings.ToUpper(sizeStr[i:])

	size, err := strconv.ParseInt(numericPart, 10, 64)
	if err != nil {
		return 0, err
	}

	switch unitPart {
	case "", "B":
		return size, nil
	case "KB":
		return size * 1024, nil
	case "MB":
		return size * 1024 * 1024, nil
	case "GB":
		return size * 1024 * 1024 * 1024, nil
	case "TB":
		return size * 1024 * 1024 * 1024 * 1024, nil
	case "PB":
		return size * 1024 * 1024 * 1024 * 1024 * 1024, nil
	default:
		return 0, fmt.Errorf("invalid unit: %s", unitPart)
	}
}

// collectFiles recursively traverses the project to collect files
// respecting inclusion/exclusion patterns and size limits.
func collectFiles(rootDir string, includePatterns, excludePatterns []string, minSize, maxSize int64) ([]string, error) {
	var files []string

	// Create gitignore matcher
	ps := make([]gitignore.Pattern, 0, len(excludePatterns))
	for _, p := range excludePatterns {
		if strings.TrimSpace(p) == "" {
			continue
		}
		ps = append(ps, gitignore.ParsePattern(p, nil))
	}
	matcher := gitignore.NewMatcher(ps)

	err := filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}

		// Check gitignore patterns
		if matcher.Match(strings.Split(filepath.ToSlash(relPath), "/"), d.IsDir()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		excluded := false
		for _, pattern := range excludePatterns {
			matched, err := doublestar.Match(pattern, relPath)
			if err != nil {
				return err
			}
			if matched {
				excluded = true
				break
			}
		}
		if excluded {
			return nil
		}

		if len(includePatterns) > 0 {
			included := false
			for _, pattern := range includePatterns {
				matched, err := doublestar.Match(pattern, relPath)
				if err != nil {
					return err
				}
				if matched {
					included = true
					break
				}
			}
			if !included {
				return nil
			}
		}

		if isBinaryFile(path) {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}
		size := info.Size()
		if size < minSize || size > maxSize {
			return nil
		}

		files = append(files, relPath)
		return nil
	})

	return files, err
}

// buildTree builds a tree representation of the project
// from the collected files list.
func buildTree(files []string, rootName string) *Directory {
	root := &Directory{
		Name:    rootName,
		SubDirs: make(map[string]*Directory),
		Files:   []string{},
	}

	for _, file := range files {
		components := strings.Split(filepath.ToSlash(file), "/")
		current := root

		for i, comp := range components {
			if i == len(components)-1 {
				current.Files = append(current.Files, comp)
			} else {
				if _, exists := current.SubDirs[comp]; !exists {
					current.SubDirs[comp] = &Directory{
						Name:    comp,
						SubDirs: make(map[string]*Directory),
						Files:   []string{},
					}
				}
				current = current.SubDirs[comp]
			}
		}
	}

	return root
}

// generateTreeOutput generates a textual representation of the tree
// structure of the project in a readable format.
func generateTreeOutput(root *Directory) string {
	var output strings.Builder
	output.WriteString(fmt.Sprintf("└── %s/\n", root.Name))
	printDir(root, "    ", &output)
	return output.String()
}

func printDir(dir *Directory, prefix string, output *strings.Builder) {
	subDirs := make([]*Directory, 0, len(dir.SubDirs))
	for _, d := range dir.SubDirs {
		subDirs = append(subDirs, d)
	}
	sort.Slice(subDirs, func(i, j int) bool {
		return subDirs[i].Name < subDirs[j].Name
	})

	sort.Strings(dir.Files)

	for i, d := range subDirs {
		isLast := i == len(subDirs)-1 && len(dir.Files) == 0
		if isLast {
			output.WriteString(fmt.Sprintf("%s└── %s/\n", prefix, d.Name))
			printDir(d, prefix+"    ", output)
		} else {
			output.WriteString(fmt.Sprintf("%s├── %s/\n", prefix, d.Name))
			printDir(d, prefix+"│   ", output)
		}
	}

	for i, f := range dir.Files {
		isLast := i == len(dir.Files)-1
		connector := "├── "
		if isLast && len(subDirs) == 0 {
			connector = "└── "
		}
		output.WriteString(fmt.Sprintf("%s%s%s\n", prefix, connector, f))
	}
}

func loadGitignorePatterns(rootDir string) ([]string, error) {
	gitignorePath := filepath.Join(rootDir, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		return nil, nil
	}

	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		return nil, err
	}

	var patterns []string
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			patterns = append(patterns, line)
		}
	}

	return patterns, nil
}

func ensureGitignoreEntry(projectDir, entry string) error {
	gitignorePath := filepath.Join(projectDir, ".gitignore")

	// Normalize entry - remove trailing slash if present
	entry = strings.TrimSuffix(strings.TrimPrefix(entry, "/"), "/")

	// Variable pour stocker le contenu
	var content string

	// Lire le contenu existant s'il existe
	data, err := os.ReadFile(gitignorePath)
	if err == nil {
		// Le fichier existe, lire son contenu
		content = string(data)

		// Vérifier si l'entrée existe déjà (avec ou sans slash)
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			trimmedLine := strings.TrimSpace(line)
			// Skip empty lines and comments
			if trimmedLine == "" || strings.HasPrefix(trimmedLine, "#") {
				continue
			}

			// Normalize line for comparison
			normalizedLine := strings.TrimSuffix(strings.TrimPrefix(trimmedLine, "/"), "/")

			// Check if entry already exists (with or without trailing slash)
			if normalizedLine == entry ||
				normalizedLine == entry+"/" ||
				entry == normalizedLine+"/" {
				return nil // Déjà présent
			}
		}

		// Ajouter une nouvelle ligne si nécessaire
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
	} else if os.IsNotExist(err) {
		// Le fichier n'existe pas, créer un nouveau contenu avec un en-tête
		content = "# Generated by Prompt My Project\n" +
			"# Ignore output directory\n\n"
	} else {
		// Une autre erreur s'est produite
		return fmt.Errorf("error reading .gitignore: %w", err)
	}

	// Ajouter la nouvelle entrée avec slash à la fin pour indiquer un dossier
	content += entry + "/\n"

	// Écrire le fichier
	err = os.WriteFile(gitignorePath, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("error writing .gitignore: %w", err)
	}

	fmt.Printf("Updated %s: added '%s/'\n", gitignorePath, entry)
	return nil
}

func estimateTokenCount(text string) int {
	// Simple estimation: 1 token ≈ 4 characters
	return int(math.Ceil(float64(len(text)) / 4.0))
}

func printStatistics(stats ProcessStats) {
	fmt.Println()
	headerColor.Println("\n📊 Generation Report")
	fmt.Println(strings.Repeat("─", 50))

	successColor.Print("✓ ")
	fmt.Printf("File saved: %s\n", infoColor.Sprint(stats.outputPath))

	fmt.Println("\n📈 Statistics:")
	fmt.Printf("  • Duration: %s\n", infoColor.Sprintf("%.5f seconds", stats.duration.Seconds()))
	fmt.Printf("  • Files processed: %s\n", successColor.Sprintf("%d", stats.fileCount))
	fmt.Printf("  • Total size: %s\n", infoColor.Sprintf("%s", humanize.Bytes(uint64(stats.totalSize))))
	fmt.Printf("  • Estimated tokens: %s\n", successColor.Sprintf("%d", stats.tokenCount))
	fmt.Printf("  • Characters: %s\n", infoColor.Sprintf("%d", stats.charCount))
	fmt.Printf("  • Average: %s per file\n", infoColor.Sprintf("%s", humanize.Bytes(uint64(stats.totalSize/int64(stats.fileCount)))))
	fmt.Printf("  • Speed: %s\n", successColor.Sprintf("%.2f files/sec", float64(stats.fileCount)/stats.duration.Seconds()))

	fmt.Println(strings.Repeat("─", 50))
}

// New function to collect file extensions
func collectFileExtensions(files []string) map[string]int {
	extensions := make(map[string]int)
	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file))
		if ext != "" {
			extensions[ext]++
		} else {
			extensions["<no-extension>"]++
		}
	}
	return extensions
}

// New function to generate the prompt header
func generatePromptHeader(files []string, totalSize int64, tokenCount int, charCount int, projectDir string) string {
	var header strings.Builder

	// Project information
	projectName := filepath.Base(filepath.Clean(projectDir)) // INCORRECT: Using dir which is undefined here
	hostname, _ := os.Hostname()
	currentTime := time.Now()

	// General information
	header.WriteString("PROJECT INFORMATION\n")
	header.WriteString("-----------------------------------------------------\n")
	header.WriteString(fmt.Sprintf("• Project Name: %s\n", projectName))
	header.WriteString(fmt.Sprintf("• Generated On: %s\n", currentTime.Format("2006-01-02 15:04:05")))
	header.WriteString(fmt.Sprintf("• Generated with: Prompt My Project (PMP) v%s\n", Version))
	header.WriteString(fmt.Sprintf("• Host: %s\n", hostname))
	header.WriteString(fmt.Sprintf("• OS: %s/%s\n", runtime.GOOS, runtime.GOARCH))
	header.WriteString("\n")

	// File statistics
	header.WriteString("FILE STATISTICS\n")
	header.WriteString("-----------------------------------------------------\n")
	header.WriteString(fmt.Sprintf("• Total Files: %d\n", len(files)))
	header.WriteString(fmt.Sprintf("• Total Size: %s\n", humanize.Bytes(uint64(totalSize))))
	header.WriteString(fmt.Sprintf("• Avg. File Size: %s\n", humanize.Bytes(uint64(totalSize/int64(max(1, len(files)))))))

	// File extensions
	extensions := collectFileExtensions(files)
	if len(extensions) > 0 {
		header.WriteString("• File Types:\n")
		extList := make([]string, 0, len(extensions))
		for ext := range extensions {
			extList = append(extList, ext)
		}
		sort.Slice(extList, func(i, j int) bool {
			return extensions[extList[i]] > extensions[extList[j]]
		})
		for i, ext := range extList {
			if i < 10 { // Limit to 10 extensions for readability
				header.WriteString(fmt.Sprintf("  - %s: %d files\n", ext, extensions[ext]))
			} else {
				break
			}
		}
		if len(extList) > 10 {
			header.WriteString(fmt.Sprintf("  - ...and %d other types\n", len(extList)-10))
		}
	}
	header.WriteString("\n")

	// Token statistics
	header.WriteString("TOKEN STATISTICS\n")
	header.WriteString("-----------------------------------------------------\n")
	header.WriteString(fmt.Sprintf("• Estimated Token Count: %d\n", tokenCount))
	header.WriteString(fmt.Sprintf("• Character Count: %d\n", charCount))
	header.WriteString("\n")
	header.WriteString("=====================================================\n\n")
	return header.String()
}

// Helper function for max
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
