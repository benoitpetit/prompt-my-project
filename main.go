package main

// Prompt My Project (PMP) is a command-line tool to generate structured prompts
// from source code, optimized for AI assistants.
// It analyzes project files, generates statistics, and creates formatted output
// that can be used with AI tools like ChatGPT, Claude or Gemini.
//
// Installation:
//
// You can install PMP with go install:
//
//	go install github.com/benoitpetit/prompt-my-project@latest
//
// Or with one of the installation scripts:
//
//	curl -fsSL https://raw.githubusercontent.com/benoitpetit/prompt-my-project/master/scripts/install.sh | bash
//
// For more information, see https://github.com/benoitpetit/prompt-my-project
import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/benoitpetit/prompt-my-project/pkg/analyzer"
	"github.com/benoitpetit/prompt-my-project/pkg/binary"
	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
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
	MaxFiles:        500,    // Default file limit
	MaxTotalSize:    "10MB", // Default total size limit
	ProgressBarSize: 40,
	OutputFormat:    "txt", // Default format
}

// DefaultExcludes are patterns to exclude by default
var defaultExcludes = []string{
	"node_modules/**", "vendor/**", ".git/**", "**/.git/**", ".svn/**",
	"**/.DS_Store", ".idea/**", ".vscode/**", "dist/**", "build/**",
	"**/__pycache__/**", "**/*.pyc", "**/*.pyo", "**/*.so", "**/*.dll",
	"**/*.exe", "**/*.bin", "**/*.obj", "**/*.o", "**/*.a", "**/*.lib",
}

// Helper function to parse a size string
func parseSize(sizeStr string) (int64, error) {
	sizeStr = strings.TrimSpace(sizeStr)
	if len(sizeStr) == 0 {
		return 0, fmt.Errorf("empty size string")
	}

	// Find the numeric part
	i := 0
	for ; i < len(sizeStr); i++ {
		c := sizeStr[i]
		if c < '0' || c > '9' {
			break
		}
	}

	var multiplier int64 = 1
	var value int64 = 0

	// Parse the numeric part
	if i > 0 {
		var err error
		value, err = parseInt64(sizeStr[:i])
		if err != nil {
			return 0, fmt.Errorf("invalid size value: %w", err)
		}
	}

	// Parse the unit part
	if i < len(sizeStr) {
		unitStr := strings.ToUpper(strings.TrimSpace(sizeStr[i:]))

		switch unitStr {
		case "B", "":
			multiplier = 1
		case "KB", "K":
			multiplier = 1024
		case "MB", "M":
			multiplier = 1024 * 1024
		case "GB", "G":
			multiplier = 1024 * 1024 * 1024
		case "TB", "T":
			multiplier = 1024 * 1024 * 1024 * 1024
		default:
			return 0, fmt.Errorf("unknown size unit: %s", unitStr)
		}
	}

	return value * multiplier, nil
}

// Helper function to parse int64
func parseInt64(s string) (int64, error) {
	var n int64
	var err error

	// Simple parsing without strconv
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid character in integer: %c", c)
		}
		n = n*10 + int64(c-'0')
	}

	return n, err
}

// Load .gitignore patterns
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

// Make sure the output directory is in .gitignore
func ensureGitignoreEntry(projectDir, entry string) error {
	gitignorePath := filepath.Join(projectDir, ".gitignore")

	// Check if .gitignore already contains the entry
	if _, err := os.Stat(gitignorePath); err == nil {
		content, err := os.ReadFile(gitignorePath)
		if err != nil {
			return err
		}

		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) == entry {
				// Entry already exists
				return nil
			}
		}

		// Entry doesn't exist, append it
		file, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer file.Close()

		if _, err := file.WriteString("\n" + entry + "\n"); err != nil {
			return err
		}
	} else {
		// .gitignore doesn't exist, create it
		if err := os.WriteFile(gitignorePath, []byte(entry+"\n"), 0644); err != nil {
			return err
		}
	}

	return nil
}

// Print statistics to the console
func printStatistics(stats analyzer.StatsResult) {
	bold := color.New(color.Bold).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	fmt.Println()
	fmt.Println(bold("=== Project Analysis Complete ==="))
	fmt.Printf("Files processed: %s\n", green(stats.FileCount))
	fmt.Printf("Total file size: %s\n", green(humanize.Bytes(uint64(stats.TotalSize))))
	fmt.Printf("Estimated tokens: %s\n", green(stats.TokenCount))
	fmt.Printf("Characters: %s\n", green(stats.CharCount))
	fmt.Printf("Processing time: %s\n", green(stats.ProcessTime.Round(time.Millisecond)))
	fmt.Printf("Files per second: %s\n", green(fmt.Sprintf("%.1f", stats.FilesPerSec)))
	fmt.Printf("Output file: %s\n", green(stats.OutputPath))
	fmt.Println()
}

func main() {
	// Initialize binary cache
	binaryCache := binary.NewCache()

	// Load binary cache
	if err := binaryCache.Load(); err != nil {
		fmt.Printf("Warning: error loading binary cache: %v\n", err)
	}

	// Save binary cache when done
	defer func() {
		if err := binaryCache.Save(); err != nil {
			fmt.Printf("Warning: error saving binary cache: %v\n", err)
		}
	}()

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

			// Check for specific format in args
			var formatOverride string
			for i, arg := range os.Args {
				if arg == "--format" || arg == "-f" {
					if i+1 < len(os.Args) {
						formatOverride = os.Args[i+1]
					}
				}
			}

			// Check for specific output in args
			var outputOverride string
			for i, arg := range os.Args {
				if arg == "--output" || arg == "-o" {
					if i+1 < len(os.Args) {
						outputOverride = os.Args[i+1]
					}
				}
			}

			// Use the format override if available
			format := c.String("format")
			if formatOverride != "" {
				format = formatOverride
			}

			// If an argument is provided, proceed with the analysis
			// Get the project path
			dir := c.Args().First()
			if !filepath.IsAbs(dir) {
				absDir, err := filepath.Abs(dir)
				if err == nil {
					dir = absDir
				}
			}

			// Parse size limits
			minSize, err := parseSize(c.String("min-size"))
			if err != nil {
				return fmt.Errorf("invalid min-size: %w", err)
			}

			maxSize, err := parseSize(c.String("max-size"))
			if err != nil {
				return fmt.Errorf("invalid max-size: %w", err)
			}

			// Parse total size limit
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

			// Determine output directory
			outputDir := c.String("output")
			if outputOverride != "" {
				outputDir = outputOverride
			}

			if outputDir == "" {
				outputDir = filepath.Join(dir, DefaultConfig.OutputDir)
			} else if !filepath.IsAbs(outputDir) {
				outputDir = filepath.Join(dir, outputDir)
			}

			// Check if output directory is in the project
			relPath, err := filepath.Rel(dir, outputDir)
			if err == nil && !strings.HasPrefix(relPath, "..") {
				gitignoreEntry := strings.TrimPrefix(relPath, string(filepath.Separator))
				if err := ensureGitignoreEntry(dir, gitignoreEntry); err != nil {
					fmt.Printf("Warning: unable to update .gitignore: %v\n", err)
				}
			}

			// Create the project analyzer
			projectAnalyzer := analyzer.New(
				dir,
				c.StringSlice("include"),
				excludePatterns,
				minSize,
				maxSize,
				c.Int("max-files"),
				maxTotalSize,
				c.Int("workers"),
			)

			// Collect files
			if err := projectAnalyzer.CollectFiles(); err != nil {
				return err
			}

			// Process files
			stats, err := projectAnalyzer.ProcessFiles(outputDir, format)
			if err != nil {
				return err
			}

			// Display statistics
			printStatistics(stats)
			return nil
		},
	}

	// Run the application
	if err := app.Run(os.Args); err != nil {
		color.Red("❌ Error: %v", err)
		os.Exit(1)
	}
}
