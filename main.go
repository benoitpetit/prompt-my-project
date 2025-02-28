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

// Configuration globale
const (
	Version   = "1.0.0"
	GoVersion = "1.21"
)

// Configuration par défaut
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

// Liste des extensions à exclure
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
	// Vérifier l'extension
	ext := strings.ToLower(path.Ext(filepath))
	if binaryExtensions[ext] {
		return true
	}

	// Vérifier le type MIME
	mimeType := mime.TypeByExtension(ext)
	if mimeType != "" {
		return !strings.HasPrefix(mimeType, "text/") &&
			!strings.Contains(mimeType, "application/json") &&
			!strings.Contains(mimeType, "application/xml")
	}

	// Lire les premiers octets pour détecter les caractères nuls
	file, err := os.Open(filepath)
	if err != nil {
		return true // En cas de doute, considérer comme binaire
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

// SimpleProgress représente une barre de progression simple avec mutex
type SimpleProgress struct {
	sync.Mutex
	total     int
	current   int
	lastWidth int
}

func NewSimpleProgress(total int) *SimpleProgress {
	return &SimpleProgress{
		total:     total,
		current:   0,
		lastWidth: 0,
	}
}

func (p *SimpleProgress) Update(current int) {
	p.Lock()
	defer p.Unlock()

	p.current = current
	width := 40
	progress := float64(current) / float64(p.total)
	filled := int(float64(width) * progress)

	if p.lastWidth > 0 {
		fmt.Printf("\r%s\r", strings.Repeat(" ", p.lastWidth))
	}

	bar := color.HiBlueString("[")
	bar += color.GreenString(strings.Repeat("█", filled))
	if filled < width {
		bar += color.HiBlackString(strings.Repeat("░", width-filled))
	}
	bar += color.HiBlueString("]")

	output := fmt.Sprintf("%s %s",
		bar,
		color.HiWhiteString("%d/%d (%d%%)", current, p.total, int(progress*100)),
	)
	fmt.Print(output)
	p.lastWidth = len(output) + 20 // Compensation pour les codes ANSI
}

// Ajouter une constante pour les patterns à exclure par défaut
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

// Cache pour les fichiers déjà traités
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

// WorkerPool gère un pool de workers pour le traitement concurrent
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

	// Réutilisation du buffer pour optimiser la mémoire
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
	// Vérifier le cache
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

	// Utilisation du buffer réutilisable pour la lecture
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
		Description: `Prompt My Project (PMP) analyse votre projet et génère un prompt 
formaté pour les assistants AI. Il permet d'exclure les fichiers binaires, 
de respecter les règles .gitignore, et offre des options de filtrage avancées.

Exemple d'utilisation:
   pmp                            # Analyse le projet courant
   pmp --include "*.go"           # Analyse uniquement les fichiers .go
   pmp --exclude "test/*"         # Exclut les fichiers du dossier test
   pmp --output ~/prompts         # Spécifie le dossier de sortie`,
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
			startTime := time.Now()
			var totalSize int64

			dir := "."
			if c.Args().Present() {
				dir = c.Args().First()
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

			// Calculer la taille totale des fichiers
			for _, file := range files {
				if info, err := os.Stat(filepath.Join(dir, file)); err == nil {
					totalSize += info.Size()
				}
			}

			// Création de l'arbre et de la structure
			rootName := filepath.Base(dir)
			tree := buildTree(files, rootName)
			structure := generateTreeOutput(tree)

			// Variable contentBuilder pour stocker le contenu des fichiers
			var contentBuilder strings.Builder

			// Initialiser le pool de workers
			numWorkers := runtime.GOMAXPROCS(0) * 2
			pool := NewWorkerPool(numWorkers)
			pool.Start()

			// Envoyer les jobs
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

			// Collecter les résultats
			results := make([]string, len(files))
			progress := NewSimpleProgress(len(files))
			completed := 0

			for result := range pool.results {
				if result.err != nil {
					fmt.Printf("\nWarning: error processing file: %v\n", result.err)
					continue
				}
				results[result.index] = result.content
				completed++
				progress.Update(completed)
			}

			fmt.Println() // Nouvelle ligne après la barre de progression

			// Écriture des résultats dans l'ordre
			for _, content := range results {
				if content != "" {
					contentBuilder.WriteString(content)
				}
			}

			// Contenu des fichiers
			promptContent := contentBuilder.String()

			// Calculer les tokens
			tokenCount := estimateTokenCount(promptContent)

			// Construction du prompt final amélioré
			var output strings.Builder

			// Générer l'en-tête avec les statistiques en haut
			header := generatePromptHeader(files, totalSize, tokenCount, len(promptContent))
			output.WriteString(header)

			// Ajouter la structure du projet
			output.WriteString("PROJECT STRUCTURE:\n")
			output.WriteString("-----------------------------------------------------\n\n")
			output.WriteString(structure)

			// Ajouter le contenu des fichiers
			output.WriteString("\n\nFILE CONTENTS:\n")
			output.WriteString("-----------------------------------------------------\n")
			output.WriteString(promptContent)

			finalPrompt := output.String()

			// Determine output directory
			outputDir := c.String("output")
			if outputDir == "" {
				// Par défaut, créer un dossier "prompts" dans le projet analysé
				outputDir = filepath.Join(dir, "prompts")

				// Ajouter "prompts/" au .gitignore s'il n'y est pas déjà
				if err := ensureGitignoreEntry(dir, "prompts/"); err != nil {
					fmt.Printf("Warning: impossible de mettre à jour .gitignore: %v\n", err)
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

// collectFiles parcourt récursivement le projet pour collecter les fichiers
// en respectant les patterns d'inclusion/exclusion et les limites de taille.
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

// buildTree construit une représentation arborescente du projet
// à partir de la liste des fichiers collectés.
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

// generateTreeOutput génère une représentation textuelle de l'arborescence
// du projet dans un format lisible.
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

	// Lire le contenu existant ou créer un nouveau fichier
	var content string
	if data, err := os.ReadFile(gitignorePath); err == nil {
		content = string(data)

		// Vérifier si l'entrée existe déjà
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) == strings.TrimSpace(entry) {
				return nil // Déjà présent
			}
		}

		// Ajouter une nouvelle ligne si nécessaire
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
	}

	// Ajouter la nouvelle entrée
	content += entry + "\n"

	// Écrire le fichier
	return os.WriteFile(gitignorePath, []byte(content), 0644)
}

func estimateTokenCount(text string) int {
	// Estimation simple : 1 token ≈ 4 caractères
	return int(math.Ceil(float64(len(text)) / 4.0))
}

func printStatistics(stats ProcessStats) {
	fmt.Println()
	headerColor.Println("\n📊 Rapport de génération")
	fmt.Println(strings.Repeat("─", 50))

	successColor.Print("✓ ")
	fmt.Printf("Fichier sauvegardé: %s\n", infoColor.Sprint(stats.outputPath))

	fmt.Println("\n📈 Statistiques:")
	fmt.Printf("  • Durée: %s\n", infoColor.Sprintf("%.5f secondes", stats.duration.Seconds()))
	fmt.Printf("  • Fichiers traités: %s\n", successColor.Sprintf("%d", stats.fileCount))
	fmt.Printf("  • Taille totale: %s\n", infoColor.Sprintf("%s", humanize.Bytes(uint64(stats.totalSize))))
	fmt.Printf("  • Tokens estimés: %s\n", successColor.Sprintf("%d", stats.tokenCount))
	fmt.Printf("  • Caractères: %s\n", infoColor.Sprintf("%d", stats.charCount))
	fmt.Printf("  • Moyenne: %s par fichier\n", infoColor.Sprintf("%s", humanize.Bytes(uint64(stats.totalSize/int64(stats.fileCount)))))
	fmt.Printf("  • Vitesse: %s\n", successColor.Sprintf("%.2f fichiers/sec", float64(stats.fileCount)/stats.duration.Seconds()))

	fmt.Println(strings.Repeat("─", 50))
}

// Nouvelle fonction pour collecter les extensions de fichiers
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

// Nouvelle fonction pour générer l'en-tête du prompt
func generatePromptHeader(files []string, totalSize int64, tokenCount int, charCount int) string {
	var header strings.Builder

	// Information sur le projet
	projectName := filepath.Base(filepath.Clean("."))
	hostname, _ := os.Hostname()
	currentTime := time.Now()

	// Informations générales
	header.WriteString("PROJECT INFORMATION\n")
	header.WriteString("-----------------------------------------------------\n")
	header.WriteString(fmt.Sprintf("• Project Name: %s\n", projectName))
	header.WriteString(fmt.Sprintf("• Generated On: %s\n", currentTime.Format("2006-01-02 15:04:05")))
	header.WriteString(fmt.Sprintf("• Generated with: Prompt My Project (PMP) v%s\n", Version))
	header.WriteString(fmt.Sprintf("• Host: %s\n", hostname))
	header.WriteString(fmt.Sprintf("• OS: %s/%s\n", runtime.GOOS, runtime.GOARCH))
	header.WriteString("\n")

	// Statistiques des fichiers
	header.WriteString("FILE STATISTICS\n")
	header.WriteString("-----------------------------------------------------\n")
	header.WriteString(fmt.Sprintf("• Total Files: %d\n", len(files)))
	header.WriteString(fmt.Sprintf("• Total Size: %s\n", humanize.Bytes(uint64(totalSize))))
	header.WriteString(fmt.Sprintf("• Avg. File Size: %s\n", humanize.Bytes(uint64(totalSize/int64(max(1, len(files)))))))

	// Extensions de fichiers
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
			if i < 10 { // Limiter à 10 extensions pour la lisibilité
				header.WriteString(fmt.Sprintf("  - %s: %d files\n", ext, extensions[ext]))
			} else {
				break
			}
		}
		if len(extList) > 10 {
			header.WriteString(fmt.Sprintf("  - ...et %d autres types\n", len(extList)-10))
		}
	}
	header.WriteString("\n")

	// Statistiques de tokens
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
