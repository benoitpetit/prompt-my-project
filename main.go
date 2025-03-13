package main

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
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
	MinSize         string
	MaxSize         string
	OutputDir       string
	GitIgnore       bool
	WorkerCount     int
	MaxFiles        int
	MaxTotalSize    string
	ProgressBarSize int
	OutputFormat    string
}{
	MinSize:         "1KB",
	MaxSize:         "100MB",
	OutputDir:       "prompts",
	GitIgnore:       true,
	WorkerCount:     runtime.NumCPU(),
	MaxFiles:        500,    // Limite par défaut de fichiers
	MaxTotalSize:    "10MB", // Taille totale maximale par défaut
	ProgressBarSize: 40,
	OutputFormat:    "txt", // Format par défaut
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

// BinaryCache gère un cache persistant des résultats de détection de fichiers binaires
type BinaryCache struct {
	sync.RWMutex
	cache     map[string]bool
	cacheDir  string
	cacheFile string
}

// NewBinaryCache crée un nouveau cache pour les fichiers binaires
func NewBinaryCache() *BinaryCache {
	homeDir, err := os.UserHomeDir()
	cacheDir := filepath.Join(homeDir, ".pmp", "cache")
	if err != nil {
		// En cas d'erreur, utiliser un répertoire temporaire
		cacheDir = filepath.Join(os.TempDir(), "pmp-cache")
	}

	return &BinaryCache{
		cache:     make(map[string]bool),
		cacheDir:  cacheDir,
		cacheFile: filepath.Join(cacheDir, "binary_cache.json"),
	}
}

// Load charge le cache depuis le disque
func (bc *BinaryCache) Load() error {
	bc.Lock()
	defer bc.Unlock()

	// Vérifier si le fichier de cache existe
	if _, err := os.Stat(bc.cacheFile); os.IsNotExist(err) {
		// Créer le répertoire de cache si nécessaire
		if err := os.MkdirAll(bc.cacheDir, 0755); err != nil {
			return fmt.Errorf("error creating cache directory: %w", err)
		}
		// Pas de cache existant, rien à charger
		return nil
	}

	// Lire le fichier de cache
	data, err := os.ReadFile(bc.cacheFile)
	if err != nil {
		return fmt.Errorf("error reading cache file: %w", err)
	}

	// Désérialiser le cache
	if err := json.Unmarshal(data, &bc.cache); err != nil {
		// En cas d'erreur, réinitialiser le cache
		bc.cache = make(map[string]bool)
		return fmt.Errorf("error parsing cache file: %w", err)
	}

	return nil
}

// Save sauvegarde le cache sur le disque
func (bc *BinaryCache) Save() error {
	bc.RLock()
	defer bc.RUnlock()

	// Créer le répertoire de cache si nécessaire
	if err := os.MkdirAll(bc.cacheDir, 0755); err != nil {
		return fmt.Errorf("error creating cache directory: %w", err)
	}

	// Sérialiser le cache
	data, err := json.MarshalIndent(bc.cache, "", "  ")
	if err != nil {
		return fmt.Errorf("error serializing cache: %w", err)
	}

	// Écrire le fichier de cache
	if err := os.WriteFile(bc.cacheFile, data, 0644); err != nil {
		return fmt.Errorf("error writing cache file: %w", err)
	}

	return nil
}

// Get récupère une valeur du cache
func (bc *BinaryCache) Get(key string) (bool, bool) {
	bc.RLock()
	defer bc.RUnlock()
	val, ok := bc.cache[key]
	return val, ok
}

// Set ajoute une valeur au cache
func (bc *BinaryCache) Set(key string, value bool) {
	bc.Lock()
	defer bc.Unlock()
	bc.cache[key] = value
}

// Variable globale pour le cache des fichiers binaires
var binaryCache = NewBinaryCache()

func isBinaryFile(filepath string) bool {
	// Calculer un hash du chemin et de la taille/date de modification pour l'unicité
	fileInfo, err := os.Stat(filepath)
	if err != nil {
		return true // En cas de doute, considérer comme binaire
	}

	// Créer une clé de cache unique basée sur le chemin et les méta-informations
	cacheKey := fmt.Sprintf("%s:%d:%d", filepath, fileInfo.Size(), fileInfo.ModTime().UnixNano())

	// Vérifier dans le cache
	if isBinary, found := binaryCache.Get(cacheKey); found {
		return isBinary
	}

	// Si non trouvé dans le cache, effectuer la détection
	// Check extension
	ext := strings.ToLower(path.Ext(filepath))
	if binaryExtensions[ext] {
		binaryCache.Set(cacheKey, true)
		return true
	}

	// Check MIME type
	mimeType := mime.TypeByExtension(ext)
	if mimeType != "" {
		isBinary := !strings.HasPrefix(mimeType, "text/") &&
			!strings.Contains(mimeType, "application/json") &&
			!strings.Contains(mimeType, "application/xml")

		// Si le MIME type est définitif, mettre en cache et retourner
		if isBinary || strings.HasPrefix(mimeType, "text/") {
			binaryCache.Set(cacheKey, isBinary)
			return isBinary
		}
	}

	// Read first bytes to detect null characters
	file, err := os.Open(filepath)
	if err != nil {
		binaryCache.Set(cacheKey, true) // When in doubt, consider as binary
		return true
	}
	defer file.Close()

	buffer := make([]byte, 512)
	n, err := file.Read(buffer)
	if err != nil {
		binaryCache.Set(cacheKey, true)
		return true
	}

	isBinary := http.DetectContentType(buffer[:n]) != "text/plain; charset=utf-8"
	binaryCache.Set(cacheKey, isBinary)
	return isBinary
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
	p.lastWidth = len(bar+progressText) + 20 // Marge pour les codes ANSI

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

// RetryConfig définit la configuration pour les retries
type RetryConfig struct {
	MaxRetries  int
	RetryDelay  time.Duration
	MaxFileSize int64 // Taille maximale pour les retries
}

// DefaultRetryConfig est la configuration par défaut pour les retries
var DefaultRetryConfig = RetryConfig{
	MaxRetries:  3,
	RetryDelay:  100 * time.Millisecond,
	MaxFileSize: 1024 * 1024, // 1MB
}

func (wp *WorkerPool) processFileWithRetry(rootDir, relPath string, buffer []byte, retryConfig RetryConfig) (string, error) {
	// Vérifier d'abord si le fichier n'est pas trop gros pour les retries
	absPath := filepath.Join(rootDir, relPath)
	info, err := os.Stat(absPath)
	if err != nil {
		return "", fmt.Errorf("error stat file: %w", err)
	}

	// Si le fichier est trop grand, ne pas utiliser de retry
	if info.Size() > retryConfig.MaxFileSize {
		return wp.processFile(rootDir, relPath, buffer)
	}

	var lastErr error
	for attempt := 0; attempt <= retryConfig.MaxRetries; attempt++ {
		if attempt > 0 {
			// Attendre avant de réessayer
			time.Sleep(retryConfig.RetryDelay)
		}

		content, err := wp.processFile(rootDir, relPath, buffer)
		if err == nil {
			return content, nil
		}

		lastErr = err
	}

	return "", fmt.Errorf("failed after %d attempts: %w", retryConfig.MaxRetries, lastErr)
}

// ProjectAnalyzer encapsule la logique d'analyse d'un projet
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
}

// NewProjectAnalyzer crée un nouvel analyseur de projet avec les paramètres spécifiés
func NewProjectAnalyzer(dir string, includePatterns, excludePatterns []string, minSize, maxSize int64, maxFiles int, maxTotalSize int64, workerCount int) *ProjectAnalyzer {
	return &ProjectAnalyzer{
		Dir:             dir,
		IncludePatterns: includePatterns,
		ExcludePatterns: excludePatterns,
		MinSize:         minSize,
		MaxSize:         maxSize,
		MaxFiles:        maxFiles,
		MaxTotalSize:    maxTotalSize,
		WorkerCount:     workerCount,
	}
}

// CollectFiles collecte les fichiers selon les critères spécifiés
func (pa *ProjectAnalyzer) CollectFiles() error {
	files, err := collectFiles(pa.Dir, pa.IncludePatterns, pa.ExcludePatterns, pa.MinSize, pa.MaxSize)
	if err != nil {
		return fmt.Errorf("error collecting files: %w", err)
	}

	// Appliquer la limite du nombre de fichiers
	if pa.MaxFiles > 0 && len(files) > pa.MaxFiles {
		fmt.Printf("Limiting to %d files out of %d found\n", pa.MaxFiles, len(files))
		files = files[:pa.MaxFiles]
	}

	// Calculer et appliquer la limite de taille totale
	var fileInfos []struct {
		path string
		size int64
	}

	for _, file := range files {
		if info, err := os.Stat(filepath.Join(pa.Dir, file)); err == nil {
			fileInfos = append(fileInfos, struct {
				path string
				size int64
			}{file, info.Size()})
			pa.TotalSize += info.Size()
		}
	}

	// Appliquer la limite de taille totale si nécessaire
	if pa.MaxTotalSize > 0 && pa.TotalSize > pa.MaxTotalSize {
		fmt.Printf("Total size exceeds limit (%s > %s), filtering files...\n",
			humanize.Bytes(uint64(pa.TotalSize)), humanize.Bytes(uint64(pa.MaxTotalSize)))

		// Trier les fichiers par taille
		sort.Slice(fileInfos, func(i, j int) bool {
			return fileInfos[i].size < fileInfos[j].size
		})

		// Reconstruire la liste des fichiers
		files = []string{}
		pa.TotalSize = 0
		for _, fi := range fileInfos {
			if pa.TotalSize+fi.size <= pa.MaxTotalSize {
				files = append(files, fi.path)
				pa.TotalSize += fi.size
			} else {
				break
			}
		}

		fmt.Printf("Reduced to %d files, total size: %s\n",
			len(files), humanize.Bytes(uint64(pa.TotalSize)))
	}

	pa.Files = files
	return nil
}

// GenerateProjectStructure génère une représentation structurée du projet
func (pa *ProjectAnalyzer) GenerateProjectStructure() (string, error) {
	rootName := filepath.Base(pa.Dir)
	tree := buildTreeImproved(pa.Files, rootName)
	return generateTreeOutput(tree), nil
}

// ProcessFiles traite les fichiers et retourne le chemin du fichier de sortie
func (pa *ProjectAnalyzer) ProcessFiles(outputDir string, format string) (string, error) {
	// Créer le répertoire de sortie
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("error creating output directory: %w", err)
	}

	// Si le format n'est pas texte, on ne génère pas de fichier texte
	if format != "txt" {
		// Pour les formats JSON et XML, on se contente de collecter les informations
		// sans générer de fichier texte
		tokenEstimator := NewTokenEstimator()
		for _, file := range pa.Files {
			absPath := filepath.Join(pa.Dir, file)
			if info, err := os.Stat(absPath); err == nil {
				pa.TotalSize += info.Size()
			}

			// Estimer les tokens pour chaque fichier
			tokens, err := tokenEstimator.EstimateFileTokens(absPath, true)
			if err == nil {
				pa.TokenCount += tokens
			}

			// Compter les caractères
			if content, err := os.ReadFile(absPath); err == nil {
				pa.CharCount += len(content)
			}
		}

		// Retourner un chemin vide, le fichier sera créé plus tard
		return "", nil
	}

	// Pour le format texte, continuer avec le comportement actuel
	// Créer les fichiers de sortie
	timestamp := time.Now().Format("20060102_150405")
	fileName := fmt.Sprintf("prompt_%s.txt", timestamp)
	outputPath := filepath.Join(outputDir, fileName)

	// Fichier temporaire pour le contenu
	tmpFile, err := os.CreateTemp(outputDir, "tmp_content_*.txt")
	if err != nil {
		return "", fmt.Errorf("error creating temporary file: %w", err)
	}
	tmpFilePath := tmpFile.Name()

	// Utiliser une fonction defer immédiate pour garantir que le nettoyage est effectué
	defer func(file *os.File, path string) {
		file.Close()
		os.Remove(path)
	}(tmpFile, tmpFilePath)

	// Traiter les fichiers avec un pool de workers
	pool := NewWorkerPool(pa.WorkerCount)
	pool.Start()

	// Canal pour suivre les goroutines
	done := make(chan struct{})

	// Envoyer les jobs
	go func() {
		defer close(done) // Signaler que l'envoi des jobs est terminé
		for i, file := range pa.Files {
			select {
			case <-done: // Au cas où nous devons annuler prématurément
				return
			default:
				pool.jobs <- WorkerJob{
					index:    i,
					filePath: file,
					rootDir:  pa.Dir,
				}
			}
		}
		pool.Stop()
	}()

	// Collecter les résultats
	results := make(map[int]bool, len(pa.Files))
	progress := NewSimpleProgress(len(pa.Files))
	progress.SetDescription("Processing files")
	completed := 0
	var failedFiles int

	// Configuration pour les retries
	retryConfig := DefaultRetryConfig

	for result := range pool.results {
		if result.err != nil {
			// Gestion des erreurs avec retry
			if progress.lastWidth > 0 {
				fmt.Print("\r")
				fmt.Print(strings.Repeat(" ", progress.lastWidth))
				fmt.Print("\r")
			}
			fmt.Printf("Warning: error processing file: %v\n", result.err)
			failedFiles++

			// Retry pour les fichiers problématiques
			if failedFiles < 10 {
				filePath := pa.Files[result.index]
				fmt.Printf("Retrying file: %s\n", filePath)

				content, retryErr := pool.processFileWithRetry(pa.Dir, filePath, nil, retryConfig)
				if retryErr == nil {
					n, writeErr := tmpFile.WriteString(content)
					if writeErr != nil {
						fmt.Printf("Warning: error writing to temporary file (retry): %v\n", writeErr)
					} else {
						pa.CharCount += n
						failedFiles--
					}
				} else {
					fmt.Printf("Warning: retry failed for file: %v\n", retryErr)
				}
			}
		} else if result.content != "" {
			// Écrire le contenu
			n, writeErr := tmpFile.WriteString(result.content)
			if writeErr != nil {
				fmt.Printf("Warning: error writing to temporary file: %v\n", writeErr)
			} else {
				pa.CharCount += n
			}
		}
		results[result.index] = true
		completed++
		progress.Update(completed)
	}

	// S'assurer que tout est écrit
	tmpFile.Sync()
	tmpFile.Seek(0, 0)

	// Estimer les tokens
	tokenEstimator := NewTokenEstimator()
	tokenCount, err := tokenEstimator.EstimateFileTokens(tmpFilePath, true)
	if err != nil {
		tokenCount = estimateTokenFromChars(pa.CharCount)
		fmt.Printf("Warning: error estimating tokens: %v, using basic estimation\n", err)
	}
	pa.TokenCount = tokenCount

	// Générer la structure du projet
	structure, err := pa.GenerateProjectStructure()
	if err != nil {
		return "", fmt.Errorf("error generating project structure: %w", err)
	}

	// Construire le prompt final
	var output strings.Builder
	header := generatePromptHeader(pa.Files, pa.TotalSize, pa.TokenCount, pa.CharCount, pa.Dir)
	output.WriteString(header)

	// Ajouter la structure
	output.WriteString("PROJECT STRUCTURE:\n")
	output.WriteString("-----------------------------------------------------\n\n")
	output.WriteString(structure)

	// Ajouter l'en-tête du contenu
	output.WriteString("\n\nFILE CONTENTS:\n")
	output.WriteString("-----------------------------------------------------\n")

	// Écrire dans le fichier final
	finalFile, err := os.Create(outputPath)
	if err != nil {
		return "", fmt.Errorf("error creating output file: %w", err)
	}
	defer finalFile.Close()

	// Écrire l'en-tête
	if _, err := finalFile.WriteString(output.String()); err != nil {
		return "", fmt.Errorf("error writing header to output file: %w", err)
	}

	// Copier le contenu
	if _, err := io.Copy(finalFile, tmpFile); err != nil {
		return "", fmt.Errorf("error copying content to output file: %w", err)
	}

	return outputPath, nil
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
			&cli.IntFlag{
				Name:  "workers",
				Usage: "Number of parallel workers (default: number of CPUs)",
				Value: DefaultConfig.WorkerCount,
			},
			&cli.IntFlag{
				Name:  "max-files",
				Usage: "Maximum number of files to process (default: 500, 0 = unlimited)",
				Value: DefaultConfig.MaxFiles,
			},
			&cli.StringFlag{
				Name:  "max-total-size",
				Usage: "Maximum total size of all files (e.g., 10MB, 0 = unlimited)",
				Value: DefaultConfig.MaxTotalSize,
			},
			&cli.StringFlag{
				Name:    "format",
				Aliases: []string{"f"},
				Usage:   "Output format (txt, json, or xml)",
				Value:   DefaultConfig.OutputFormat,
			},
		},
		Action: func(c *cli.Context) error {
			// If no argument is provided, display help and exit
			if !c.Args().Present() {
				return cli.ShowAppHelp(c)
			}

			// Charger le cache des fichiers binaires
			if err := binaryCache.Load(); err != nil {
				fmt.Printf("Warning: error loading binary cache: %v\n", err)
			}
			// Sauvegarder le cache à la fin
			defer func() {
				if err := binaryCache.Save(); err != nil {
					fmt.Printf("Warning: error saving binary cache: %v\n", err)
				}
			}()

			// If an argument is provided, proceed with the analysis
			startTime := time.Now()

			// Récupérer le chemin du projet
			dir := c.Args().First()
			if !filepath.IsAbs(dir) {
				absDir, err := filepath.Abs(dir)
				if err == nil {
					dir = absDir
				}
			}

			// Parser les limites de taille
			minSize, err := parseSize(c.String("min-size"))
			if err != nil {
				return fmt.Errorf("invalid min-size: %w", err)
			}

			maxSize, err := parseSize(c.String("max-size"))
			if err != nil {
				return fmt.Errorf("invalid max-size: %w", err)
			}

			// Parser la taille totale maximale
			maxTotalSizeStr := c.String("max-total-size")
			var maxTotalSize int64
			if maxTotalSizeStr != "0" && maxTotalSizeStr != "" {
				maxTotalSize, err = parseSize(maxTotalSizeStr)
				if err != nil {
					return fmt.Errorf("invalid max-total-size: %w", err)
				}
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

			// Créer l'analyseur de projet
			analyzer := NewProjectAnalyzer(
				dir,
				c.StringSlice("include"),
				excludePatterns,
				minSize,
				maxSize,
				c.Int("max-files"),
				maxTotalSize,
				c.Int("workers"),
			)

			// Collecter les fichiers
			if err := analyzer.CollectFiles(); err != nil {
				return err
			}

			// Determiner le répertoire de sortie
			outputDir := c.String("output")
			if outputDir == "" {
				outputDir = filepath.Join(dir, "prompts")
			} else if !filepath.IsAbs(outputDir) {
				outputDir = filepath.Join(dir, outputDir)
			}

			// Vérifier si le répertoire de sortie est dans le projet
			relPath, err := filepath.Rel(dir, outputDir)
			if err == nil && !strings.HasPrefix(relPath, "..") {
				gitignoreEntry := strings.TrimPrefix(relPath, string(filepath.Separator))
				if err := ensureGitignoreEntry(dir, gitignoreEntry); err != nil {
					fmt.Printf("Warning: unable to update .gitignore: %v\n", err)
				}
			}

			// Récupérer les informations système
			hostname, err := os.Hostname()
			if err != nil {
				hostname = "unknown"
			}

			// Traiter les fichiers
			outputPath, err := analyzer.ProcessFiles(outputDir, c.String("format"))
			if err != nil {
				return err
			}

			// Afficher les statistiques
			stats := ProcessStats{
				startTime:  startTime,
				duration:   time.Since(startTime),
				fileCount:  len(analyzer.Files),
				totalSize:  analyzer.TotalSize,
				tokenCount: analyzer.TokenCount,
				charCount:  analyzer.CharCount,
				outputPath: outputPath,
			}

			// Créer le rapport
			report := &ProjectReport{}
			report.ProjectInfo.Name = filepath.Base(filepath.Clean(analyzer.Dir))
			report.ProjectInfo.GeneratedAt = time.Now()
			report.ProjectInfo.Generator = fmt.Sprintf("Prompt My Project (PMP) v%s", Version)
			report.ProjectInfo.Host = hostname
			report.ProjectInfo.OS = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
			report.Technologies = detectTechnologies(analyzer.Files, analyzer.Dir)
			report.KeyFiles = identifyKeyFiles(analyzer.Files)
			report.Issues = identifyPotentialIssues(analyzer.Files, analyzer.Dir)
			report.Statistics.FileCount = len(analyzer.Files)
			report.Statistics.TotalSize = analyzer.TotalSize
			report.Statistics.TotalSizeHuman = humanize.Bytes(uint64(analyzer.TotalSize))
			report.Statistics.AvgFileSize = analyzer.TotalSize / int64(max(1, len(analyzer.Files)))
			report.Statistics.TokenCount = analyzer.TokenCount
			report.Statistics.CharCount = analyzer.CharCount
			report.Statistics.FilesPerSecond = float64(len(analyzer.Files)) / stats.duration.Seconds()
			report.FileTypes = make([]struct {
				Extension string `json:"extension" xml:"extension,attr"`
				Count     int    `json:"count" xml:"count"`
			}, 0)
			for ext, count := range collectFileExtensions(analyzer.Files) {
				report.FileTypes = append(report.FileTypes, struct {
					Extension string `json:"extension" xml:"extension,attr"`
					Count     int    `json:"count" xml:"count"`
				}{
					Extension: ext,
					Count:     count,
				})
			}
			// Initialiser la liste des fichiers vide
			report.Files = []struct {
				Path     string `json:"path" xml:"path"`
				Size     int64  `json:"size" xml:"size"`
				Content  string `json:"content,omitempty" xml:"content,omitempty"`
				Language string `json:"language" xml:"language"`
			}{}

			// Générer le fichier de sortie selon le format demandé
			format := c.String("format")
			var outputExt string
			var content []byte

			switch format {
			case "json":
				// Vider la liste des fichiers pour éviter les doublons
				report.Files = []struct {
					Path     string `json:"path" xml:"path"`
					Size     int64  `json:"size" xml:"size"`
					Content  string `json:"content,omitempty" xml:"content,omitempty"`
					Language string `json:"language" xml:"language"`
				}{}

				// Ajouter le contenu des fichiers au rapport avec barre de progression
				progress := NewSimpleProgress(len(analyzer.Files))
				progress.SetDescription("Processing JSON files")
				for i, file := range analyzer.Files {
					absPath := filepath.Join(analyzer.Dir, file)
					fileContent, err := os.ReadFile(absPath)
					if err == nil {
						info, err := os.Stat(absPath)
						if err == nil {
							report.Files = append(report.Files, struct {
								Path     string `json:"path" xml:"path"`
								Size     int64  `json:"size" xml:"size"`
								Content  string `json:"content,omitempty" xml:"content,omitempty"`
								Language string `json:"language" xml:"language"`
							}{
								Path:     file,
								Size:     info.Size(),
								Content:  string(fileContent),
								Language: detectFileLanguage(file),
							})
						}
					}
					progress.Update(i + 1)
				}
				content, err = json.MarshalIndent(report, "", "  ")
				outputExt = "json"
			case "xml":
				// Vider la liste des fichiers pour éviter les doublons
				report.Files = []struct {
					Path     string `json:"path" xml:"path"`
					Size     int64  `json:"size" xml:"size"`
					Content  string `json:"content,omitempty" xml:"content,omitempty"`
					Language string `json:"language" xml:"language"`
				}{}

				// Ajouter le contenu des fichiers au rapport avec barre de progression
				progress := NewSimpleProgress(len(analyzer.Files))
				progress.SetDescription("Processing XML files")
				for i, file := range analyzer.Files {
					absPath := filepath.Join(analyzer.Dir, file)
					fileContent, err := os.ReadFile(absPath)
					if err == nil {
						info, err := os.Stat(absPath)
						if err == nil {
							report.Files = append(report.Files, struct {
								Path     string `json:"path" xml:"path"`
								Size     int64  `json:"size" xml:"size"`
								Content  string `json:"content,omitempty" xml:"content,omitempty"`
								Language string `json:"language" xml:"language"`
							}{
								Path:     file,
								Size:     info.Size(),
								Content:  string(fileContent),
								Language: detectFileLanguage(file),
							})
						}
					}
					progress.Update(i + 1)
				}
				content, err = xml.MarshalIndent(report, "", "  ")
				outputExt = "xml"
			default:
				// Format texte par défaut
				var textContent strings.Builder
				textContent.WriteString(generatePromptHeader(analyzer.Files, analyzer.TotalSize, analyzer.TokenCount, analyzer.CharCount, analyzer.Dir))
				textContent.WriteString("\nPROJECT STRUCTURE:\n")
				textContent.WriteString("-----------------------------------------------------\n\n")
				structure, _ := analyzer.GenerateProjectStructure()
				textContent.WriteString(structure)
				textContent.WriteString("\nFILE CONTENTS:\n")
				textContent.WriteString("-----------------------------------------------------\n")

				// Ajouter le contenu des fichiers
				for _, file := range analyzer.Files {
					absPath := filepath.Join(analyzer.Dir, file)
					fileContent, err := os.ReadFile(absPath)
					if err == nil {
						textContent.WriteString("\n================================================\n")
						textContent.WriteString(fmt.Sprintf("File: %s\n", file))
						textContent.WriteString("================================================\n")
						textContent.WriteString(string(fileContent))
						textContent.WriteString("\n")
					}
				}

				content = []byte(textContent.String())
				outputExt = "txt"
			}

			if err != nil {
				return fmt.Errorf("error generating %s output: %w", format, err)
			}

			// Créer le fichier de sortie avec la bonne extension
			timestamp := time.Now().Format("20060102_150405")
			fileName := fmt.Sprintf("prompt_%s.%s", timestamp, outputExt)
			outputPath = filepath.Join(outputDir, fileName)

			if err = os.WriteFile(outputPath, content, 0644); err != nil {
				return fmt.Errorf("error writing output file: %w", err)
			}

			// Afficher les statistiques
			stats.outputPath = outputPath
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

		// Si c'est un dossier, vérifier s'il est explicitement exclu par les patterns
		if d.IsDir() {
			for _, pattern := range excludePatterns {
				// Ajouter un slash final si nécessaire pour matcher les dossiers
				checkPattern := pattern
				if !strings.HasSuffix(pattern, "/") && !strings.HasSuffix(pattern, "/*") {
					checkPattern = pattern + "/"
				}
				matched, err := doublestar.Match(checkPattern, relPath+"/")
				if err != nil {
					return err
				}
				if matched {
					return filepath.SkipDir
				}
			}
			return nil
		}

		excluded := false
		for _, pattern := range excludePatterns {
			matched, err := doublestar.Match(pattern, filepath.ToSlash(relPath))
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
				matched, err := doublestar.Match(pattern, filepath.ToSlash(relPath))
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

// buildTreeImproved construit une représentation en arbre du projet
// à partir de la liste de fichiers, en utilisant des fonctions utilitaires pour plus de clarté
func buildTreeImproved(files []string, rootName string) *Directory {
	root := &Directory{
		Name:    rootName,
		SubDirs: make(map[string]*Directory),
		Files:   []string{},
	}

	// Ajouter chaque fichier à l'arborescence
	for _, file := range files {
		addFileToDirectory(root, file)
	}

	return root
}

// getOrCreateDirectory obtient ou crée un sous-répertoire dans le répertoire parent spécifié
func getOrCreateDirectory(parent *Directory, name string) *Directory {
	if dir, exists := parent.SubDirs[name]; exists {
		return dir
	}

	dir := &Directory{
		Name:    name,
		SubDirs: make(map[string]*Directory),
		Files:   []string{},
	}
	parent.SubDirs[name] = dir
	return dir
}

// addFileToDirectory ajoute un fichier à un répertoire en suivant le chemin spécifié
func addFileToDirectory(root *Directory, filePath string) {
	components := strings.Split(filepath.ToSlash(filePath), "/")
	current := root

	// Naviguer à travers les composants du chemin
	for i, comp := range components {
		if i == len(components)-1 {
			// Dernier composant = nom du fichier
			current.Files = append(current.Files, comp)
		} else {
			// Composant intermédiaire = nom de répertoire
			current = getOrCreateDirectory(current, comp)
		}
	}
}

// generateTreeOutput generates a textual representation of the tree
// structure of the project in a readable format.
func generateTreeOutput(root *Directory) string {
	var output strings.Builder
	output.WriteString("```\n") // Encadrer avec des délimiteurs de code pour une meilleure lisibilité
	printDirectory(root, &output, "", true)
	output.WriteString("```\n")
	return output.String()
}

// Fonction printDir remplacée par printDirectory définie précédemment

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

func estimateTokenFromChars(charCount int) int {
	// Simple estimation: 1 token ≈ 4 caractères
	return int(math.Ceil(float64(charCount) / 4.0))
}

// TokenEstimator fournit des estimations de tokens plus précises selon le contenu
type TokenEstimator struct {
	// Facteurs d'ajustement pour différentes catégories de texte
	CodeFactor   float64
	TextFactor   float64
	SpecialChars map[rune]float64 // Caractères spéciaux et leur poids en tokens
}

// NewTokenEstimator crée un nouvel estimateur avec des valeurs par défaut
func NewTokenEstimator() *TokenEstimator {
	return &TokenEstimator{
		CodeFactor: 0.25, // ~4 caractères par token pour le code
		TextFactor: 0.20, // ~5 caractères par token pour le texte normal
		SpecialChars: map[rune]float64{
			'\n': 0.5,           // Les sauts de ligne comptent plus
			'\t': 0.5,           // Les tabulations comptent plus
			' ':  0.1,           // Les espaces comptent moins
			'{':  0.5, '}': 0.5, // Les accolades comptent plus
			'[': 0.5, ']': 0.5, // Les crochets comptent plus
			'(': 0.5, ')': 0.5, // Les parenthèses comptent plus
			'<': 0.5, '>': 0.5, // Les chevrons comptent plus
		},
	}
}

// EstimateTokens retourne une estimation plus précise des tokens pour un texte
func (te *TokenEstimator) EstimateTokens(text string, isCode bool) int {
	if text == "" {
		return 0
	}

	factor := te.TextFactor
	if isCode {
		factor = te.CodeFactor
	}

	// Compter les caractères spéciaux
	specialCharsWeight := 0.0
	for _, r := range text {
		if weight, exists := te.SpecialChars[r]; exists {
			specialCharsWeight += weight
		}
	}

	// Estimation de base par caractères + ajustement pour caractères spéciaux
	baseEstimate := float64(len(text)) * factor
	return int(math.Ceil(baseEstimate + specialCharsWeight))
}

// EstimateFileTokens estime les tokens pour un fichier sans le charger entièrement en mémoire
func (te *TokenEstimator) EstimateFileTokens(filePath string, isCode bool) (int, error) {
	// Vérifier d'abord que le fichier existe
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return 0, fmt.Errorf("file does not exist: %s", filePath)
	}

	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	// Vérifier la taille du fichier pour éviter les problèmes avec des fichiers trop grands
	info, err := file.Stat()
	if err != nil {
		return 0, err
	}

	// Si le fichier est très grand, utiliser une estimation approximative
	if info.Size() > 50*1024*1024 { // 50MB
		return estimateTokenFromChars(int(info.Size())), nil
	}

	var totalTokens int
	buffer := make([]byte, 8192) // Lire par blocs de 8KB

	for {
		n, err := file.Read(buffer)
		if n > 0 {
			chunk := string(buffer[:n])
			totalTokens += te.EstimateTokens(chunk, isCode)
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return 0, err
		}
	}

	return totalTokens, nil
}

func printStatistics(stats ProcessStats) {
	fmt.Println()
	headerColor.Println("\n📊 Generation Report")
	fmt.Println(strings.Repeat("─", 50))

	successColor.Print("✓ ")
	fmt.Printf("File saved: %s\n", infoColor.Sprint(stats.outputPath))

	fmt.Println("\n📈 Statistics:")
	fmt.Printf("  • Duration: %s\n", infoColor.Sprintf("%s (%0.2f seconds)", formatDuration(stats.duration), stats.duration.Seconds()))
	fmt.Printf("  • Files processed: %s\n", successColor.Sprintf("%d", stats.fileCount))
	fmt.Printf("  • Total size: %s\n", infoColor.Sprintf("%s", humanize.Bytes(uint64(stats.totalSize))))
	fmt.Printf("  • Estimated tokens: %s\n", successColor.Sprintf("%d", stats.tokenCount))
	fmt.Printf("  • Characters: %s\n", infoColor.Sprintf("%d", stats.charCount))

	// Éviter les divisions par zéro
	if stats.fileCount > 0 {
		fmt.Printf("  • Average: %s per file\n", infoColor.Sprintf("%s", humanize.Bytes(uint64(stats.totalSize/int64(stats.fileCount)))))
	} else {
		fmt.Printf("  • Average: %s per file\n", infoColor.Sprintf("N/A"))
	}

	if stats.duration.Seconds() > 0 {
		fmt.Printf("  • Speed: %s\n", successColor.Sprintf("%.2f files/sec", float64(stats.fileCount)/stats.duration.Seconds()))
	} else {
		fmt.Printf("  • Speed: %s\n", successColor.Sprintf("N/A"))
	}

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

// detectTechnologies tente de détecter les technologies utilisées dans le projet
// en se basant sur les extensions de fichiers et certains fichiers spécifiques
func detectTechnologies(files []string, rootDir string) []string {
	extensions := collectFileExtensions(files)
	detectedTechs := make(map[string]bool)

	// Détecter les technologies par extension
	techByExt := map[string]string{
		".go":     "Go",
		".py":     "Python",
		".js":     "JavaScript",
		".jsx":    "React",
		".ts":     "TypeScript",
		".tsx":    "React with TypeScript",
		".html":   "HTML",
		".css":    "CSS",
		".scss":   "SCSS/Sass",
		".java":   "Java",
		".c":      "C",
		".cpp":    "C++",
		".cs":     "C#",
		".rb":     "Ruby",
		".php":    "PHP",
		".rs":     "Rust",
		".swift":  "Swift",
		".kt":     "Kotlin",
		".dart":   "Dart",
		".vue":    "Vue.js",
		".svelte": "Svelte",
	}

	for ext, count := range extensions {
		if tech, exists := techByExt[ext]; exists && count > 0 {
			detectedTechs[tech] = true
		}
	}

	// Détecter les frameworks et outils basés sur des fichiers spécifiques
	fileIndicators := map[string]string{
		"package.json":       "Node.js",
		"go.mod":             "Go Modules",
		"Cargo.toml":         "Rust/Cargo",
		"requirements.txt":   "Python",
		"Gemfile":            "Ruby",
		"composer.json":      "PHP/Composer",
		"webpack.config.js":  "Webpack",
		"next.config.js":     "Next.js",
		"angular.json":       "Angular",
		"docker-compose.yml": "Docker",
		"Dockerfile":         "Docker",
		"Jenkinsfile":        "Jenkins",
		".github/workflows":  "GitHub Actions",
		"tailwind.config.js": "Tailwind CSS",
		"jest.config.js":     "Jest",
		"cypress.json":       "Cypress",
	}

	// Vérifier l'existence de fichiers spécifiques dans le projet
	for indicator, tech := range fileIndicators {
		indicatorPath := filepath.Join(rootDir, indicator)
		if _, err := os.Stat(indicatorPath); err == nil {
			// Le fichier existe directement à la racine
			detectedTechs[tech] = true
			continue
		}

		// Chercher si le fichier est mentionné dans la liste des fichiers
		for _, file := range files {
			if strings.HasSuffix(file, indicator) || strings.Contains(file, indicator) {
				detectedTechs[tech] = true
				break
			}
		}
	}

	// Convertir en slice et trier
	var result []string
	for tech := range detectedTechs {
		result = append(result, tech)
	}
	sort.Strings(result)

	return result
}

// identifyKeyFiles identifie les fichiers les plus importants du projet
func identifyKeyFiles(files []string) []string {
	// Fichiers importants par leur nom
	keyFileNames := map[string]bool{
		"main.go":            true,
		"app.js":             true,
		"index.js":           true,
		"server.js":          true,
		"index.html":         true,
		"package.json":       true,
		"go.mod":             true,
		"requirements.txt":   true,
		"Dockerfile":         true,
		"docker-compose.yml": true,
		"Makefile":           true,
		"readme.md":          true,
		"README.md":          true,
		"config.json":        true,
		"settings.json":      true,
		"schema.sql":         true,
		"app.py":             true,
		"main.py":            true,
	}

	var keyFiles []string

	// Ajouter les fichiers par nom
	for _, file := range files {
		baseName := filepath.Base(file)
		if keyFileNames[baseName] {
			keyFiles = append(keyFiles, file)
		}
	}

	// Si moins de 5 fichiers clés trouvés, ajouter les fichiers qui semblent importants
	// basés sur leur emplacement (racine du projet, dossiers src/core/main)
	if len(keyFiles) < 5 {
		for _, file := range files {
			// Éviter les doublons
			alreadyAdded := false
			for _, kf := range keyFiles {
				if kf == file {
					alreadyAdded = true
					break
				}
			}

			if alreadyAdded {
				continue
			}

			// Fichiers à la racine du projet (pas de séparateur dans le chemin)
			if !strings.Contains(file, "/") && !strings.Contains(file, "\\") {
				keyFiles = append(keyFiles, file)
				continue
			}

			// Fichiers dans des dossiers importants
			importantDirs := []string{"/src/", "/core/", "/main/", "/app/", "/config/", "/api/"}
			for _, dir := range importantDirs {
				if strings.Contains(filepath.ToSlash(file), dir) {
					keyFiles = append(keyFiles, file)
					break
				}
			}

			// Limiter à 10 fichiers clés
			if len(keyFiles) >= 10 {
				break
			}
		}
	}

	// Trier les fichiers pour une présentation cohérente
	sort.Strings(keyFiles)

	return keyFiles
}

// identifyPotentialIssues analyse les fichiers pour détecter des problèmes potentiels
func identifyPotentialIssues(files []string, projectDir string) []string {
	var issues []string

	// Compteurs pour différents types de problèmes
	totalFiles := len(files)
	largeFiles := 0
	longFiles := 0
	deeplyNestedFiles := 0

	// Seuils pour considérer qu'un fichier présente un problème
	const (
		largeFileSizeThreshold  = 100 * 1024 // 100 KB
		longFileLineThreshold   = 500        // 500 lignes
		deepNestedPathThreshold = 5          // 5 niveaux de profondeur
	)

	// Structure pour stocker temporairement les informations sur les fichiers
	type fileInfo struct {
		path      string
		size      int64
		lineCount int
		depth     int
	}

	// Pré-allouer la slice avec la capacité appropriée pour optimiser les performances
	filesInfo := make([]fileInfo, 0, len(files))

	// Collecter les informations sur les fichiers
	for _, file := range files {
		absPath := file
		if !filepath.IsAbs(file) {
			absPath = filepath.Join(projectDir, file)
		}

		// Vérifier l'existence du fichier
		stat, err := os.Stat(absPath)
		if err != nil {
			continue
		}

		// Calculer la profondeur du fichier
		relPath, err := filepath.Rel(projectDir, absPath)
		if err != nil {
			continue
		}
		depth := len(strings.Split(filepath.ToSlash(relPath), "/")) - 1

		// Compter les lignes dans le fichier
		lineCount := 0
		if stat.Size() < 10*1024*1024 { // Ne pas essayer de lire les fichiers > 10MB
			f, err := os.Open(absPath)
			if err == nil {
				defer f.Close()
				scanner := bufio.NewScanner(f)
				for scanner.Scan() {
					lineCount++
				}
			}
		}

		// Ajouter le fichier à la slice (en Go, l'affectation du résultat de append est obligatoire)
		filesInfo = append(filesInfo, fileInfo{
			path:      relPath,
			size:      stat.Size(),
			lineCount: lineCount,
			depth:     depth,
		})

		// Compteurs pour les problèmes
		if stat.Size() > largeFileSizeThreshold {
			largeFiles++
		}
		if lineCount > longFileLineThreshold {
			longFiles++
		}
		if depth > deepNestedPathThreshold {
			deeplyNestedFiles++
		}
	}

	// Vérifier s'il y a des problèmes au niveau global du projet
	if largeFiles > 0 {
		percentage := float64(largeFiles) / float64(totalFiles) * 100
		issues = append(issues, fmt.Sprintf("%.1f%% of files (%d) are large (>100KB), which may indicate poor modularity", percentage, largeFiles))
	}

	if longFiles > 0 {
		percentage := float64(longFiles) / float64(totalFiles) * 100
		issues = append(issues, fmt.Sprintf("%.1f%% of files (%d) contain more than 500 lines, which can make the code difficult to maintain", percentage, longFiles))
	}

	if deeplyNestedFiles > 0 {
		percentage := float64(deeplyNestedFiles) / float64(totalFiles) * 100
		issues = append(issues, fmt.Sprintf("%.1f%% of files (%d) are at more than 5 levels of depth, suggesting a potentially overly complex structure", percentage, deeplyNestedFiles))
	}

	// Limiter à 5 problèmes pour ne pas surcharger le prompt
	if len(issues) > 5 {
		issues = issues[:5]
	}

	return issues
}

// calculateCodeComplexityMetrics calcule des métriques avancées sur la complexité du code
func calculateCodeComplexityMetrics(files []string, projectDir string) map[string]interface{} {
	metrics := make(map[string]interface{})
	var totalLines int
	var lineCountByFile []int
	// Fusionner la déclaration et l'initialisation en une seule ligne
	sizeByExtension := make(map[string]int64)

	// Parcourir les fichiers pour collecter les métriques
	for _, file := range files {
		absPath := file
		if !filepath.IsAbs(file) {
			absPath = filepath.Join(projectDir, file)
		}

		// Vérifier l'existence du fichier
		stat, err := os.Stat(absPath)
		if err != nil {
			continue
		}

		// Taille par extension
		ext := filepath.Ext(file)
		if ext == "" {
			ext = "[no extension]"
		}
		sizeByExtension[ext] += stat.Size()

		// Compter les lignes
		if stat.Size() < 5*1024*1024 { // Ignorer les fichiers > 5MB
			lineCount := 0
			f, err := os.Open(absPath)
			if err == nil {
				defer f.Close()
				scanner := bufio.NewScanner(f)
				for scanner.Scan() {
					lineCount++
				}
				lineCountByFile = append(lineCountByFile, lineCount)
				totalLines += lineCount
			}
		}
	}

	// Calculer les statistiques
	metrics["total_lines"] = totalLines

	// Nombre moyen de lignes par fichier
	avgLines := 0
	if len(lineCountByFile) > 0 {
		avgLines = totalLines / len(lineCountByFile)
	}
	metrics["avg_lines_per_file"] = avgLines

	// Distribution des tailles de fichiers
	if len(lineCountByFile) > 0 {
		sort.Ints(lineCountByFile)
		metrics["median_lines"] = lineCountByFile[len(lineCountByFile)/2]

		// Calculer les percentiles
		percentiles := map[string]int{
			"p90": len(lineCountByFile) * 90 / 100,
			"p95": len(lineCountByFile) * 95 / 100,
			"p99": len(lineCountByFile) * 99 / 100,
		}

		percentileValues := make(map[string]int)
		for name, index := range percentiles {
			if index < len(lineCountByFile) {
				percentileValues[name] = lineCountByFile[index]
			}
		}
		metrics["percentiles"] = percentileValues
	}

	// Top 3 extensions par taille totale
	type extSize struct {
		ext  string
		size int64
	}

	extSizes := make([]extSize, 0, len(sizeByExtension))
	for ext, size := range sizeByExtension {
		extSizes = append(extSizes, extSize{ext, size})
	}

	sort.Slice(extSizes, func(i, j int) bool {
		return extSizes[i].size > extSizes[j].size
	})

	topExtensions := make([]map[string]interface{}, 0)
	for i, es := range extSizes {
		if i < 3 { // Top 3
			topExtensions = append(topExtensions, map[string]interface{}{
				"extension":  es.ext,
				"size":       es.size,
				"size_human": humanize.Bytes(uint64(es.size)),
			})
		}
	}
	metrics["top_extensions_by_size"] = topExtensions

	return metrics
}

// New function to generate the prompt header
func generatePromptHeader(files []string, totalSize int64, tokenCount int, charCount int, projectDir string) string {
	var header strings.Builder

	// Récupérer les informations du projet
	projectName := filepath.Base(filepath.Clean(projectDir))
	hostname, _ := os.Hostname()
	currentTime := time.Now()

	// Project information
	header.WriteString("PROJECT INFORMATION\n")
	header.WriteString("-----------------------------------------------------\n")
	header.WriteString(fmt.Sprintf("• Project Name: %s\n", projectName))
	header.WriteString(fmt.Sprintf("• Generated On: %s\n", currentTime.Format("2006-01-02 15:04:05")))
	header.WriteString(fmt.Sprintf("• Generated with: Prompt My Project (PMP) v%s\n", Version))
	header.WriteString(fmt.Sprintf("• Host: %s\n", hostname))
	header.WriteString(fmt.Sprintf("• OS: %s/%s\n", runtime.GOOS, runtime.GOARCH))
	header.WriteString("\n")

	// Technologies détectées
	technologies := detectTechnologies(files, projectDir)
	if len(technologies) > 0 {
		header.WriteString("DETECTED TECHNOLOGIES\n")
		header.WriteString("-----------------------------------------------------\n")
		for _, tech := range technologies {
			header.WriteString(fmt.Sprintf("• %s\n", tech))
		}
		header.WriteString("\n")
	}

	// Fichiers clés
	keyFiles := identifyKeyFiles(files)
	if len(keyFiles) > 0 {
		header.WriteString("KEY FILES\n")
		header.WriteString("-----------------------------------------------------\n")
		header.WriteString("These files are likely the most important for understanding the project:\n")
		for _, file := range keyFiles {
			header.WriteString(fmt.Sprintf("• %s\n", file))
		}
		header.WriteString("\n")
	}

	// Points d'attention potentiels
	issues := identifyPotentialIssues(files, projectDir)
	if len(issues) > 0 {
		header.WriteString("POINTS OF INTEREST\n")
		header.WriteString("-----------------------------------------------------\n")
		header.WriteString("These elements may deserve special attention during analysis:\n")
		for _, issue := range issues {
			header.WriteString(fmt.Sprintf("• %s\n", issue))
		}
		header.WriteString("\n")
	}

	// Calculer des métriques avancées
	complexityMetrics := calculateCodeComplexityMetrics(files, projectDir)

	// File statistics
	header.WriteString("FILE STATISTICS\n")
	header.WriteString("-----------------------------------------------------\n")
	header.WriteString(fmt.Sprintf("• Total Files: %d\n", len(files)))
	header.WriteString(fmt.Sprintf("• Total Size: %s\n", humanize.Bytes(uint64(totalSize))))
	header.WriteString(fmt.Sprintf("• Avg. File Size: %s\n", humanize.Bytes(uint64(totalSize/int64(max(1, len(files)))))))

	// Ajouter les métriques de complexité
	totalLines, hasLines := complexityMetrics["total_lines"].(int)
	if hasLines {
		header.WriteString(fmt.Sprintf("• Total Lines of Code: %d\n", totalLines))
		if avgLines, ok := complexityMetrics["avg_lines_per_file"].(int); ok {
			header.WriteString(fmt.Sprintf("• Avg. Lines per File: %d\n", avgLines))
		}
		if medianLines, ok := complexityMetrics["median_lines"].(int); ok {
			header.WriteString(fmt.Sprintf("• Median Lines per File: %d\n", medianLines))
		}

		// Ajouter les percentiles de lignes par fichier
		if percentiles, ok := complexityMetrics["percentiles"].(map[string]int); ok {
			if p90, exists := percentiles["p90"]; exists {
				header.WriteString(fmt.Sprintf("• 90%% of files have fewer than %d lines\n", p90))
			}
		}
	}

	// Afficher les extensions principales par taille
	if topExt, ok := complexityMetrics["top_extensions_by_size"].([]map[string]interface{}); ok && len(topExt) > 0 {
		header.WriteString("• Top Extensions by Size:\n")
		for _, ext := range topExt {
			extension := ext["extension"].(string)
			sizeHuman := ext["size_human"].(string)
			header.WriteString(fmt.Sprintf("  - %s: %s\n", extension, sizeHuman))
		}
	}

	// Amélioration des extensions de fichiers avec pourcentages
	extensions := collectFileExtensions(files)
	if len(extensions) > 0 {
		header.WriteString("• File Types:\n")

		// Convertir en slice pour le tri
		type ExtCount struct {
			Ext   string
			Count int
		}
		extList := make([]ExtCount, 0, len(extensions))
		for ext, count := range extensions {
			extList = append(extList, ExtCount{ext, count})
		}

		// Trier par count décroissant
		sort.Slice(extList, func(i, j int) bool {
			return extList[i].Count > extList[j].Count
		})

		total := len(files)
		for i, ext := range extList {
			if i < 10 { // Limiter à 10 extensions pour lisibilité
				percentage := float64(ext.Count) / float64(total) * 100
				header.WriteString(fmt.Sprintf("  - %s: %d files (%.1f%%)\n", ext.Ext, ext.Count, percentage))
			} else {
				// Compter le nombre total de fichiers restants
				remainingFiles := 0
				for j := i; j < len(extList); j++ {
					remainingFiles += extList[j].Count
				}
				remainingPercentage := float64(remainingFiles) / float64(total) * 100
				header.WriteString(fmt.Sprintf("  - ...and %d other types (%.1f%%)\n", len(extList)-i, remainingPercentage))
				break
			}
		}
	}
	header.WriteString("\n")

	// Ajouter des suggestions basées sur l'analyse
	header.WriteString("ANALYSIS SUGGESTIONS\n")
	header.WriteString("-----------------------------------------------------\n")
	header.WriteString("When analyzing this project, consider the following approaches:\n")

	// Suggérer des approches d'analyse basées sur les résultats
	if len(technologies) > 0 {
		techsToMention := technologies
		if len(technologies) > 3 {
			techsToMention = technologies[:3]
		}
		header.WriteString(fmt.Sprintf("• For a project using %s, examine the typical patterns and practices of these technologies\n", strings.Join(techsToMention, ", ")))
	}

	if len(keyFiles) > 0 {
		header.WriteString("• Start by analyzing the identified key files, which likely contain the main logic\n")
	}

	if len(issues) > 0 {
		header.WriteString("• Pay special attention to the identified points of interest, which may reveal problems or opportunities for improvement\n")
	}

	// Suggestions basées sur les statistiques
	if avgLines, ok := complexityMetrics["avg_lines_per_file"].(int); ok {
		if avgLines > 300 {
			header.WriteString("• The project contains large files. Look for opportunities for modularization and separation of responsibilities\n")
		} else if avgLines < 50 {
			header.WriteString("• The project is highly modularized. Focus on the interfaces between modules\n")
		}
	}

	// Suggestion générique pour compléter
	header.WriteString("• Look for design patterns used and evaluate if they are implemented effectively\n")
	header.WriteString("• Identify potential areas of technical debt or optimization\n")
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

func printDirectory(dir *Directory, builder *strings.Builder, prefix string, isLast bool) {
	// Détermine les préfixes à utiliser pour l'affichage
	var currentPrefix, childPrefix string
	if isLast {
		currentPrefix = prefix + "└── "
		childPrefix = prefix + "    "
	} else {
		currentPrefix = prefix + "├── "
		childPrefix = prefix + "│   "
	}

	// Ajoute le nom du répertoire
	builder.WriteString(currentPrefix + dir.Name)

	// Ajoute le nombre de fichiers entre parenthèses si > 0
	if len(dir.Files) > 0 {
		builder.WriteString(fmt.Sprintf(" (%d files)", len(dir.Files)))
	}
	builder.WriteString("\n")

	// Affiche les fichiers avant les sous-répertoires
	fileCount := len(dir.Files)

	// Trier les fichiers par nom
	sort.Slice(dir.Files, func(i, j int) bool {
		return dir.Files[i] < dir.Files[j]
	})

	// Limiter le nombre de fichiers à afficher pour éviter un prompt trop long
	maxFilesToShow := 10
	if fileCount > maxFilesToShow && fileCount < 20 {
		maxFilesToShow = 5 // Réduire pour les répertoires moyens
	} else if fileCount >= 20 {
		maxFilesToShow = 3 // Encore moins pour les gros répertoires
	}

	shownFiles := 0
	for i, file := range dir.Files {
		if shownFiles >= maxFilesToShow {
			// Afficher un message pour les fichiers restants
			remainingFiles := fileCount - shownFiles
			isLastItem := i == len(dir.Files)-1 && len(dir.SubDirs) == 0

			var remainingPrefix string
			if isLastItem {
				remainingPrefix = childPrefix + "└── "
			} else {
				remainingPrefix = childPrefix + "├── "
			}

			builder.WriteString(remainingPrefix + fmt.Sprintf("... %d other files\n", remainingFiles))
			break
		}

		isLastItem := i == len(dir.Files)-1 && len(dir.SubDirs) == 0
		if isLastItem {
			builder.WriteString(childPrefix + "└── " + file + "\n")
		} else {
			builder.WriteString(childPrefix + "├── " + file + "\n")
		}
		shownFiles++
	}

	// Convertir la map de sous-répertoires en slice pour pouvoir la trier
	subDirs := make([]*Directory, 0, len(dir.SubDirs))
	for _, subdir := range dir.SubDirs {
		subDirs = append(subDirs, subdir)
	}

	// Trier les sous-répertoires par nombre de fichiers (décroissant)
	sort.Slice(subDirs, func(i, j int) bool {
		// Calculer le nombre total de fichiers dans chaque sous-répertoire
		countI := len(subDirs[i].Files)
		countJ := len(subDirs[j].Files)

		// Ajouter les fichiers des sous-répertoires récursivement
		for _, subdir := range subDirs[i].SubDirs {
			countI += countFilesRecursive(subdir)
		}
		for _, subdir := range subDirs[j].SubDirs {
			countJ += countFilesRecursive(subdir)
		}

		return countI > countJ
	})

	// Affiche les sous-répertoires
	for i, subdir := range subDirs {
		isLastSubdir := i == len(subDirs)-1
		printDirectory(subdir, builder, childPrefix, isLastSubdir)
	}
}

// Fonction utilitaire pour compter récursivement les fichiers
func countFilesRecursive(dir *Directory) int {
	count := len(dir.Files)
	for _, subdir := range dir.SubDirs {
		count += countFilesRecursive(subdir)
	}
	return count
}

// ProjectReport représente la structure du rapport généré
type ProjectReport struct {
	ProjectInfo struct {
		Name        string    `json:"name" xml:"name"`
		GeneratedAt time.Time `json:"generated_at" xml:"generated_at"`
		Generator   string    `json:"generator" xml:"generator"`
		Host        string    `json:"host" xml:"host"`
		OS          string    `json:"os" xml:"os"`
	} `json:"project_info" xml:"project_info"`
	Technologies []string `json:"technologies" xml:"technologies>technology"`
	KeyFiles     []string `json:"key_files" xml:"key_files>file"`
	Issues       []string `json:"issues" xml:"issues>issue"`
	Statistics   struct {
		FileCount      int     `json:"file_count" xml:"file_count"`
		TotalSize      int64   `json:"total_size" xml:"total_size"`
		TotalSizeHuman string  `json:"total_size_human" xml:"total_size_human"`
		AvgFileSize    int64   `json:"avg_file_size" xml:"avg_file_size"`
		TokenCount     int     `json:"token_count" xml:"token_count"`
		CharCount      int     `json:"char_count" xml:"char_count"`
		FilesPerSecond float64 `json:"files_per_second" xml:"files_per_second"`
	} `json:"statistics" xml:"statistics"`
	FileTypes []struct {
		Extension string `json:"extension" xml:"extension,attr"`
		Count     int    `json:"count" xml:"count"`
	} `json:"file_types" xml:"file_types>type"`
	Files []struct {
		Path     string `json:"path" xml:"path"`
		Size     int64  `json:"size" xml:"size"`
		Content  string `json:"content,omitempty" xml:"content,omitempty"`
		Language string `json:"language" xml:"language"`
	} `json:"files" xml:"files>file"`
}

// Fonction utilitaire pour détecter le langage d'un fichier
func detectFileLanguage(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
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
	case ".c":
		return "C"
	case ".cpp":
		return "C++"
	case ".cs":
		return "C#"
	case ".rb":
		return "Ruby"
	case ".php":
		return "PHP"
	case ".html":
		return "HTML"
	case ".css":
		return "CSS"
	case ".md":
		return "Markdown"
	default:
		return "Unknown"
	}
}
