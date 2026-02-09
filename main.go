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
	"github.com/benoitpetit/prompt-my-project/pkg/config"
	"github.com/benoitpetit/prompt-my-project/pkg/formatter"
	"github.com/benoitpetit/prompt-my-project/pkg/utils"
	"github.com/dustin/go-humanize"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// Global configuration
const (
	Version   = "1.0.2"
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
	OutputDir:       "pmp_output",
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

	// Handle explicit '0' value
	if sizeStr == "0" {
		return 0, nil
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

	comment := "# Added by Prompt My Project (PMP) - https://github.com/benoitpetit/prompt-my-project"

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

		// Entry doesn't exist, append it with a comment
		file, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			return err
		}
		defer file.Close()

		if _, err := file.WriteString("\n" + comment + "\n" + entry + "\n"); err != nil {
			return err
		}
	} else {
		// .gitignore doesn't exist, create it with a comment
		if err := os.WriteFile(gitignorePath, []byte(comment+"\n"+entry+"\n"), 0644); err != nil {
			return err
		}
	}

	return nil
}

// Print statistics to the console (on stderr to not interfere with stdout output)
func printStatistics(stats analyzer.StatsResult, format string) {
	// Don't print statistics for stdout formats (used for piping)
	if strings.HasPrefix(format, "stdout:") {
		return
	}

	bold := color.New(color.Bold).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, bold("=== Project Analysis Complete ==="))
	fmt.Fprintf(os.Stderr, "Files processed: %s\n", green(stats.FileCount))
	fmt.Fprintf(os.Stderr, "Total file size: %s\n", green(humanize.Bytes(uint64(stats.TotalSize))))
	fmt.Fprintf(os.Stderr, "Estimated tokens: %s\n", green(stats.TokenCount))
	fmt.Fprintf(os.Stderr, "Characters: %s\n", green(stats.CharCount))
	fmt.Fprintf(os.Stderr, "Processing time: %s\n", green(stats.ProcessTime.Round(time.Millisecond)))
	fmt.Fprintf(os.Stderr, "Files per second: %s\n", green(fmt.Sprintf("%.1f", stats.FilesPerSec)))
	fmt.Fprintf(os.Stderr, "Output file: %s\n", green(stats.OutputPath))
	fmt.Fprintln(os.Stderr)
}

func main() {
	var rootCmd = &cobra.Command{
		Use:   "pmp",
		Short: "Transform your codebase into AI-ready prompts and visual dependency graphs",
		Long: `Prompt My Project (PMP) is a powerful command-line tool that analyzes your source code
and generates structured prompts optimized for AI assistants like ChatGPT, Claude, and Gemini.

🤖 AI-Ready Prompts: Converts your codebase into structured prompts
📊 Visual Dependency Graphs: Creates beautiful dependency graphs in multiple formats
🔧 Smart Filtering: Automatically excludes binary files and respects .gitignore
⚡ High Performance: Parallel processing with configurable workers

Quick Start:
   pmp prompt .                                    # Generate prompt for current project
   pmp prompt . --format stdout:txt               # Output to stdout for piping
   pmp graph . --format dot                       # Generate dependency graph
   pmp graph . --format stdout:dot | dot -Tpng > graph.png  # Create visual graph

For more examples and documentation, visit:
   https://github.com/benoitpetit/prompt-my-project`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// PROMPT COMMAND
	var promptCmd = &cobra.Command{
		Use:   "prompt [project path]",
		Short: "Generate AI-ready prompts from your codebase",
		Long: `Generate structured prompts from your project's source code, optimized for AI assistants like ChatGPT, Claude, and Gemini.

This command analyzes your project files, extracts meaningful content, and formats it into
a comprehensive prompt that provides AI assistants with complete context about your codebase.

Key features:
  • Smart filtering: Automatically excludes binary files and respects .gitignore
  • Multiple formats: Output as TXT, JSON, XML, or directly to stdout
  • Performance controls: Configurable limits for files, sizes, and parallel workers
  • Technology detection: Automatically identifies languages, frameworks, and tools

Examples:
  pmp prompt .                                    # Generate prompt for current project
  pmp prompt . --format stdout:txt               # Output to stdout for piping
  pmp prompt . --include "*.go" --exclude "test" # Focus on Go files, exclude tests
  pmp prompt . --format json --output /tmp       # Save as JSON in custom location
  pmp prompt . --max-files 100 --workers 4       # Limit files and workers for large projects`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Path
			dir := args[0]
			if !filepath.IsAbs(dir) {
				absDir, err := filepath.Abs(dir)
				if err == nil {
					dir = absDir
				}
			}

			// Load configuration from .pmprc, environment variables, and command-line flags
			cfg, err := config.LoadConfig(dir)
			if err != nil {
				fmt.Printf("Warning: error loading .pmprc: %v\n", err)
				cfg = config.DefaultConfig()
			}

			// Merge with environment variables
			envCfg := config.GetEnvironmentConfig()
			if len(envCfg.Exclude) > 0 {
				cfg.Exclude = envCfg.Exclude
			}
			if len(envCfg.Include) > 0 {
				cfg.Include = envCfg.Include
			}
			if envCfg.MinSize != "" {
				cfg.MinSize = envCfg.MinSize
			}
			if envCfg.MaxSize != "" {
				cfg.MaxSize = envCfg.MaxSize
			}
			if envCfg.MaxFiles > 0 {
				cfg.MaxFiles = envCfg.MaxFiles
			}
			if envCfg.MaxTotalSize != "" {
				cfg.MaxTotalSize = envCfg.MaxTotalSize
			}
			if envCfg.Format != "" {
				cfg.Format = envCfg.Format
			}
			if envCfg.OutputDir != "" {
				cfg.OutputDir = envCfg.OutputDir
			}
			if envCfg.Workers > 0 {
				cfg.Workers = envCfg.Workers
			}
			if envCfg.NoGitignore {
				cfg.NoGitignore = envCfg.NoGitignore
			}

			// Get command-line flags
			excludePatterns, _ := cmd.Flags().GetStringSlice("exclude")
			includePatterns, _ := cmd.Flags().GetStringSlice("include")
			minSizeStr, _ := cmd.Flags().GetString("min-size")
			maxSizeStr, _ := cmd.Flags().GetString("max-size")
			noGitignore, _ := cmd.Flags().GetBool("no-gitignore")
			outputDir, _ := cmd.Flags().GetString("output")
			workers, _ := cmd.Flags().GetInt("workers")
			maxFiles, _ := cmd.Flags().GetInt("max-files")
			maxTotalSizeStr, _ := cmd.Flags().GetString("max-total-size")
			format, _ := cmd.Flags().GetString("format")
			// Smart Context flags
			summaryOnly, _ := cmd.Flags().GetBool("summary-only")
			focusChanges, _ := cmd.Flags().GetBool("focus-changes")
			recentCommits, _ := cmd.Flags().GetInt("recent-commits")
			summaryPatterns, _ := cmd.Flags().GetStringSlice("summary-patterns")

			// Merge command-line flags with configuration (flags take precedence)
			cfg.MergeWithFlags(
				excludePatterns, includePatterns, summaryPatterns,
				minSizeStr, maxSizeStr, maxTotalSizeStr, format, outputDir,
				maxFiles, workers, recentCommits,
				noGitignore, summaryOnly, focusChanges,
			)

			// Use final configuration values
			excludePatterns = cfg.Exclude
			includePatterns = cfg.Include
			minSizeStr = cfg.MinSize
			maxSizeStr = cfg.MaxSize
			maxTotalSizeStr = cfg.MaxTotalSize
			format = cfg.Format
			outputDir = cfg.OutputDir
			maxFiles = cfg.MaxFiles
			workers = cfg.Workers
			noGitignore = cfg.NoGitignore

			// Parse sizes
			minSize, err := parseSize(minSizeStr)
			if err != nil {
				return fmt.Errorf("invalid min-size: %w", err)
			}
			maxSize, err := parseSize(maxSizeStr)
			if err != nil {
				return fmt.Errorf("invalid max-size: %w", err)
			}
			var maxTotalSize int64
			if maxTotalSizeStr != "0" && maxTotalSizeStr != "" {
				maxTotalSize, err = parseSize(maxTotalSizeStr)
				if err != nil {
					return fmt.Errorf("invalid max-total-size: %w", err)
				}
			}

			// Merge explicit excludes with default excludes
			excludePatterns = append(excludePatterns, defaultExcludes...)
			if !noGitignore {
				patterns, err := loadGitignorePatterns(dir)
				if err != nil {
					fmt.Printf("Warning: error loading .gitignore: %v\n", err)
				} else if patterns != nil {
					excludePatterns = append(excludePatterns, patterns...)
				}
			}

			// Create the project analyzer
			projectAnalyzer := analyzer.New(
				dir,
				includePatterns,
				excludePatterns,
				minSize,
				maxSize,
				maxFiles,
				maxTotalSize,
				workers,
			)

			if err := projectAnalyzer.CollectFiles(); err != nil {
				return err
			}

			// Output directory
			if format != "stdout" {
				if outputDir == "" {
					outputDir = filepath.Join(dir, DefaultConfig.OutputDir)
				} else if !filepath.IsAbs(outputDir) {
					outputDir = filepath.Join(dir, outputDir)
				}
				// Check if outputDir exists and is not a directory
				if info, err := os.Stat(outputDir); err == nil {
					if !info.IsDir() {
						return fmt.Errorf("output path exists and is not a directory: %s", outputDir)
					}
				} else if !os.IsNotExist(err) {
					return fmt.Errorf("failed to stat output directory: %w", err)
				}
				if err := os.MkdirAll(outputDir, 0755); err != nil {
					return fmt.Errorf("failed to create output directory: %w", err)
				}
				relPath, err := filepath.Rel(dir, outputDir)
				if err == nil && !strings.HasPrefix(relPath, "..") {
					gitignoreEntry := strings.TrimPrefix(relPath, string(filepath.Separator))
					if err := ensureGitignoreEntry(dir, gitignoreEntry); err != nil {
						fmt.Printf("Warning: unable to update .gitignore: %v\n", err)
					}
				}
			}

			// Process files and generate output
			if strings.HasPrefix(format, "stdout") {
				stdoutFormat := "txt"
				if strings.Contains(format, ":") {
					parts := strings.SplitN(format, ":", 2)
					if len(parts) == 2 && parts[1] != "" {
						stdoutFormat = parts[1]
					}
				}
				t0 := time.Now()
				// Build formatter and fill it as in ProcessFiles
				fmtr := formatter.NewFormatter(stdoutFormat, "", dir)
				// Detect technologies, key files, issues, file types
				technologies := detectTechnologiesLocal(projectAnalyzer.Files)
				keyFiles := identifyKeyFilesLocal(projectAnalyzer.Files)
				issues := identifyPotentialIssuesLocal(projectAnalyzer.Files)
				fileTypes := collectFileExtensionsLocal(projectAnalyzer.Files)
				// Add files
				for _, file := range projectAnalyzer.Files {
					absPath := filepath.Join(dir, file)
					fileInfo, err := os.Stat(absPath)
					if err != nil {
						continue
					}
					content, _ := os.ReadFile(absPath)
					fmtr.AddFile(formatter.FileInfo{
						Path:     file,
						Size:     fileInfo.Size(),
						Content:  string(content),
						Language: detectFileLanguageLocal(file),
					})
				}
				// Set stats and structure
				structure, _ := projectAnalyzer.GenerateProjectStructure()
				duration := time.Since(t0)
				if duration == 0 {
					duration = time.Second
				}
				fmtr.SetStatistics(len(projectAnalyzer.Files), projectAnalyzer.TotalSize, projectAnalyzer.TokenCount, projectAnalyzer.CharCount, duration)
				fmtr.SetTechnologies(technologies)
				fmtr.SetKeyFiles(keyFiles)
				fmtr.SetIssues(issues)
				fmtr.SetFileTypes(fileTypes)
				fmtr.SetProjectStructure(structure)
				output, err := fmtr.GetFormattedContent()
				if err != nil {
					return err
				}
				fmt.Print(output)
				return nil
			} else {
				stats, err := projectAnalyzer.ProcessFiles(outputDir, format)
				if err != nil {
					return err
				}
				printStatistics(stats, format)
				return nil
			}
		},
	}

	promptCmd.Flags().StringSliceP("exclude", "e", nil, "Exclude files matching these patterns (e.g., *.md, src/)")
	promptCmd.Flags().StringSliceP("include", "i", nil, "Include only files matching these patterns")
	promptCmd.Flags().String("min-size", DefaultConfig.MinSize, "Minimum file size (e.g., 1KB, 500B)")
	promptCmd.Flags().String("max-size", DefaultConfig.MaxSize, "Maximum file size (e.g., 100MB, 1GB)")
	promptCmd.Flags().Bool("no-gitignore", !DefaultConfig.GitIgnore, "Ignore .gitignore file")
	promptCmd.Flags().StringP("output", "o", DefaultConfig.OutputDir, "Output directory for the prompt file")
	promptCmd.Flags().Int("workers", DefaultConfig.WorkerCount, "Number of parallel workers (default: number of CPUs)")
	promptCmd.Flags().Int("max-files", DefaultConfig.MaxFiles, "Maximum number of files to process (default: 500, 0 = unlimited)")
	promptCmd.Flags().String("max-total-size", DefaultConfig.MaxTotalSize, "Maximum total size of all files (e.g., 10MB, 0 = unlimited)")
	promptCmd.Flags().StringP("format", "f", DefaultConfig.OutputFormat, "Output format (txt, json, xml, or stdout[:txt|json|xml])")
	// Smart Context flags
	promptCmd.Flags().Bool("summary-only", false, "Generate only function signatures and interfaces (AST-based summarization)")
	promptCmd.Flags().Bool("focus-changes", false, "Prioritize recently modified files (Git-aware context)")
	promptCmd.Flags().Int("recent-commits", 0, "Include files from last N commits (use with --focus-changes)")
	promptCmd.Flags().StringSlice("summary-patterns", nil, "File patterns to summarize (e.g., vendor/**, node_modules/**)")

	// GRAPH COMMAND
	var graphCmd = &cobra.Command{
		Use:   "graph [project path]",
		Short: "Generate visual dependency graphs and project structure",
		Long: `Generate visual dependency graphs and project structure representations in multiple formats.

This command analyzes your project's file and directory structure to create visual representations
that help understand the project's organization and dependencies.

Supported formats:
  • DOT: Graphviz format for creating visual diagrams (default)
  • JSON: Machine-readable structure data
  • XML: Structured markup representation
  • TXT: Human-readable tree format (like Unix tree command)

Output options:
  • File: Save to timestamped files in pmp_output/ directory
  • Stdout: Pipe output to other tools for processing

Examples:
  pmp graph .                                     # Generate DOT graph
  pmp graph . --format json                      # Generate JSON structure
  pmp graph . --format stdout:dot | dot -Tpng > graph.png  # Create PNG image
  pmp graph . --format txt > STRUCTURE.md        # Save as documentation
  pmp graph . --output /tmp/graph.json           # Custom output location`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			format, _ := cmd.Flags().GetString("format")
			outputArg, _ := cmd.Flags().GetString("output")
			projectPath := args[0]
			if !filepath.IsAbs(projectPath) {
				absDir, err := filepath.Abs(projectPath)
				if err == nil {
					projectPath = absDir
				}
			}
			an := analyzer.New(
				projectPath,
				[]string{},
				append([]string{}, defaultExcludes...),
				0, 0, 0, 0, 1,
			)
			if err := an.CollectFiles(); err != nil {
				return err
			}
			structure, err := an.GenerateProjectStructure()
			if err != nil {
				return err
			}
			var ext, content string
			var internalDeps, externalDeps int
			t0 := time.Now()
			// Handle stdout and subformats
			if strings.HasPrefix(format, "stdout") {
				stdoutFormat := "dot"
				if strings.Contains(format, ":") {
					parts := strings.SplitN(format, ":", 2)
					if len(parts) == 2 && parts[1] != "" {
						stdoutFormat = parts[1]
					}
				}
				switch stdoutFormat {
				case "dot":
					tree := utils.BuildTree(an.Files, filepath.Base(projectPath))
					content = utils.GenerateDotOutput(tree)
				case "json":
					tree := utils.BuildTree(an.Files, filepath.Base(projectPath))
					content = utils.GenerateJSONTreeOutput(tree)
				case "xml":
					tree := utils.BuildTree(an.Files, filepath.Base(projectPath))
					content = utils.GenerateXMLTreeOutput(tree)
				case "txt":
					content = structure
				default:
					return fmt.Errorf("unsupported stdout subformat: %s", stdoutFormat)
				}
				fmt.Print(content)
				return nil
			}
			switch format {
			case "dot":
				ext = "dot"
				tree := utils.BuildTree(an.Files, filepath.Base(projectPath))
				content = utils.GenerateDotOutput(tree)
				internalDeps = countDotEdges(content)
			case "json":
				ext = "json"
				tree := utils.BuildTree(an.Files, filepath.Base(projectPath))
				content = utils.GenerateJSONTreeOutput(tree)
				internalDeps = countInternalDepsFromTree(tree)
			case "xml":
				ext = "xml"
				tree := utils.BuildTree(an.Files, filepath.Base(projectPath))
				content = utils.GenerateXMLTreeOutput(tree)
				internalDeps = countInternalDepsFromTree(tree)
			case "txt":
				ext = "txt"
				content = structure
				internalDeps = countInternalDepsFromTree(utils.BuildTree(an.Files, filepath.Base(projectPath)))
			default:
				return fmt.Errorf("unsupported format: %s", format)
			}
			timestamp := time.Now().Format("20060102_150405")
			var outputPath string
			if outputArg == "" {
				outputDir := filepath.Join(projectPath, DefaultConfig.OutputDir)
				if info, err := os.Stat(outputDir); err == nil {
					if !info.IsDir() {
						return fmt.Errorf("output path exists and is not a directory: %s", outputDir)
					}
				} else if !os.IsNotExist(err) {
					return fmt.Errorf("failed to stat output directory: %w", err)
				}
				if err := os.MkdirAll(outputDir, 0755); err != nil {
					return fmt.Errorf("failed to create output directory: %w", err)
				}
				if err := ensureGitignoreEntry(projectPath, DefaultConfig.OutputDir); err != nil {
					fmt.Printf("Warning: unable to update .gitignore: %v\n", err)
				}
				outputPath = filepath.Join(outputDir, fmt.Sprintf("graph_%s.%s", timestamp, ext))
			} else {
				info, err := os.Stat(outputArg)
				if err == nil && info.IsDir() {
					outputPath = filepath.Join(outputArg, fmt.Sprintf("graph_%s.%s", timestamp, ext))
				} else if err == nil && !info.IsDir() {
					outputPath = outputArg // treat as file
				} else if os.IsNotExist(err) {
					if strings.HasSuffix(outputArg, "."+ext) {
						outputPath = outputArg
					} else {
						// Check if outputArg exists as a file (should not happen, but for safety)
						if info, err := os.Stat(outputArg); err == nil && !info.IsDir() {
							return fmt.Errorf("output path exists and is not a directory: %s", outputArg)
						}
						if err := os.MkdirAll(outputArg, 0755); err != nil {
							return fmt.Errorf("failed to create output directory: %w", err)
						}
						outputPath = filepath.Join(outputArg, fmt.Sprintf("graph_%s.%s", timestamp, ext))
					}
				} else {
					return fmt.Errorf("invalid output path: %w", err)
				}
			}
			if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to write output file: %w", err)
			}
			// Dépendances externes : simple heuristique sur fichiers connus
			externalDeps = countExternalDeps(projectPath)
			// Statistiques
			elapsed := time.Since(t0)
			stats := analyzer.StatsResult{
				FileCount:   len(an.Files),
				TotalSize:   an.TotalSize,
				ProcessTime: elapsed,
				OutputPath:  outputPath,
				FilesPerSec: float64(len(an.Files)) / elapsed.Seconds(),
				TokenCount:  an.TokenCount,
				CharCount:   an.CharCount,
				KeyFiles:    []string{},
				Issues:      []string{},
				FileTypes:   map[string]int{},
			}
			fmt.Println()
			printStatistics(stats, format)
			fmt.Printf("Internal dependencies: %d\n", internalDeps)
			fmt.Printf("External dependencies: %d\n", externalDeps)
			fmt.Println()
			return nil
		},
	}
	graphCmd.Flags().StringP("format", "f", "dot", "Output format for the graph (dot, json, xml, txt, or stdout[:dot|json|xml|txt])")
	graphCmd.Flags().StringP("output", "o", "", "Output directory or file for the graph (default: pmp_output/)")

	// GITHUB PROMPT COMMAND
	var githubPromptCmd = &cobra.Command{
		Use:   "prompt [github-url]",
		Short: "Generate AI-ready prompts from a GitHub repository",
		Long: `Clone and analyze a GitHub repository to generate structured prompts.

This command clones a GitHub repository (using shallow clone for speed) and then
generates a comprehensive prompt from its source code, exactly like the 'pmp prompt' command.

Supported URL formats:
  • https://github.com/user/repo
  • https://github.com/user/repo.git
  • git@github.com:user/repo.git

Examples:
  pmp github prompt https://github.com/user/repo
  pmp github prompt https://github.com/user/repo --branch develop
  pmp github prompt https://github.com/user/repo --format json
  pmp github prompt https://github.com/user/repo --include "*.go" --exclude "*test*"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoURL := args[0]
			branch, _ := cmd.Flags().GetString("branch")

			// Create GitHub analyzer and clone
			ghAnalyzer := analyzer.NewGitHubAnalyzer(repoURL, branch)
			dir, err := ghAnalyzer.Clone()
			if err != nil {
				return fmt.Errorf("failed to clone repository: %w", err)
			}
			defer ghAnalyzer.Cleanup()

			// Get repository info for naming
			repoInfo, _ := ghAnalyzer.GetRepoInfo()
			repoName := repoInfo.Name
			if repoName == "" {
				repoName = "github-repo"
			}

			// Get current working directory for output
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}

			// === Copy exact logic from promptCmd ===
			cfg, err := config.LoadConfig(dir)
			if err != nil {
				fmt.Printf("Warning: error loading .pmprc: %v\n", err)
				cfg = config.DefaultConfig()
			}

			// Merge with environment variables
			envCfg := config.GetEnvironmentConfig()
			if len(envCfg.Exclude) > 0 {
				cfg.Exclude = envCfg.Exclude
			}
			if len(envCfg.Include) > 0 {
				cfg.Include = envCfg.Include
			}
			if envCfg.MinSize != "" {
				cfg.MinSize = envCfg.MinSize
			}
			if envCfg.MaxSize != "" {
				cfg.MaxSize = envCfg.MaxSize
			}
			if envCfg.MaxFiles > 0 {
				cfg.MaxFiles = envCfg.MaxFiles
			}
			if envCfg.MaxTotalSize != "" {
				cfg.MaxTotalSize = envCfg.MaxTotalSize
			}
			if envCfg.Format != "" {
				cfg.Format = envCfg.Format
			}
			if envCfg.OutputDir != "" {
				cfg.OutputDir = envCfg.OutputDir
			}
			if envCfg.Workers > 0 {
				cfg.Workers = envCfg.Workers
			}
			if envCfg.NoGitignore {
				cfg.NoGitignore = envCfg.NoGitignore
			}

			excludePatterns, _ := cmd.Flags().GetStringSlice("exclude")
			includePatterns, _ := cmd.Flags().GetStringSlice("include")
			minSizeStr, _ := cmd.Flags().GetString("min-size")
			maxSizeStr, _ := cmd.Flags().GetString("max-size")
			noGitignore, _ := cmd.Flags().GetBool("no-gitignore")
			outputDir, _ := cmd.Flags().GetString("output")
			workers, _ := cmd.Flags().GetInt("workers")
			maxFiles, _ := cmd.Flags().GetInt("max-files")
			maxTotalSizeStr, _ := cmd.Flags().GetString("max-total-size")
			format, _ := cmd.Flags().GetString("format")
			summaryOnly, _ := cmd.Flags().GetBool("summary-only")
			focusChanges, _ := cmd.Flags().GetBool("focus-changes")
			recentCommits, _ := cmd.Flags().GetInt("recent-commits")
			summaryPatterns, _ := cmd.Flags().GetStringSlice("summary-patterns")

			cfg.MergeWithFlags(
				excludePatterns, includePatterns, summaryPatterns,
				minSizeStr, maxSizeStr, maxTotalSizeStr, format, outputDir,
				maxFiles, workers, recentCommits,
				noGitignore, summaryOnly, focusChanges,
			)

			excludePatterns = cfg.Exclude
			includePatterns = cfg.Include
			minSizeStr = cfg.MinSize
			maxSizeStr = cfg.MaxSize
			maxTotalSizeStr = cfg.MaxTotalSize
			format = cfg.Format
			outputDir = cfg.OutputDir
			maxFiles = cfg.MaxFiles
			workers = cfg.Workers
			noGitignore = cfg.NoGitignore

			minSize, err := parseSize(minSizeStr)
			if err != nil {
				return fmt.Errorf("invalid min-size: %w", err)
			}
			maxSize, err := parseSize(maxSizeStr)
			if err != nil {
				return fmt.Errorf("invalid max-size: %w", err)
			}
			var maxTotalSize int64
			if maxTotalSizeStr != "0" && maxTotalSizeStr != "" {
				maxTotalSize, err = parseSize(maxTotalSizeStr)
				if err != nil {
					return fmt.Errorf("invalid max-total-size: %w", err)
				}
			}

			excludePatterns = append(excludePatterns, defaultExcludes...)
			if !noGitignore {
				patterns, err := loadGitignorePatterns(dir)
				if err != nil {
					fmt.Printf("Warning: error loading .gitignore: %v\n", err)
				} else if patterns != nil {
					excludePatterns = append(excludePatterns, patterns...)
				}
			}

			projectAnalyzer := analyzer.New(
				dir,
				includePatterns,
				excludePatterns,
				minSize,
				maxSize,
				maxFiles,
				maxTotalSize,
				workers,
			)

			// Set custom project name and file prefix for GitHub repos
			projectAnalyzer.ProjectName = repoName
			projectAnalyzer.FilePrefix = repoName

			if err := projectAnalyzer.CollectFiles(); err != nil {
				return err
			}

			if format != "stdout" {
				// Use current working directory for output (not the temp clone directory)
				if outputDir == "" {
					outputDir = filepath.Join(cwd, DefaultConfig.OutputDir)
				} else if !filepath.IsAbs(outputDir) {
					outputDir = filepath.Join(cwd, outputDir)
				}
				if info, err := os.Stat(outputDir); err == nil {
					if !info.IsDir() {
						return fmt.Errorf("output path exists and is not a directory: %s", outputDir)
					}
				} else if !os.IsNotExist(err) {
					return fmt.Errorf("failed to stat output directory: %w", err)
				}
				if err := os.MkdirAll(outputDir, 0755); err != nil {
					return fmt.Errorf("failed to create output directory: %w", err)
				}
				// Note: We don't add to .gitignore in the temp directory
			}

			if strings.HasPrefix(format, "stdout") {
				stdoutFormat := "txt"
				if strings.Contains(format, ":") {
					parts := strings.SplitN(format, ":", 2)
					if len(parts) == 2 && parts[1] != "" {
						stdoutFormat = parts[1]
					}
				}
				t0 := time.Now()
				fmtr := formatter.NewFormatter(stdoutFormat, "", dir)
				fmtr.SetProjectName(repoName) // Use GitHub repo name instead of temp directory name
				technologies := detectTechnologiesLocal(projectAnalyzer.Files)
				keyFiles := identifyKeyFilesLocal(projectAnalyzer.Files)
				issues := identifyPotentialIssuesLocal(projectAnalyzer.Files)
				fileTypes := collectFileExtensionsLocal(projectAnalyzer.Files)
				for _, file := range projectAnalyzer.Files {
					absPath := filepath.Join(dir, file)
					fileInfo, err := os.Stat(absPath)
					if err != nil {
						continue
					}
					content, _ := os.ReadFile(absPath)
					fmtr.AddFile(formatter.FileInfo{
						Path:     file,
						Size:     fileInfo.Size(),
						Content:  string(content),
						Language: detectFileLanguageLocal(file),
					})
				}
				structure, _ := projectAnalyzer.GenerateProjectStructure()
				duration := time.Since(t0)
				if duration == 0 {
					duration = time.Second
				}
				fmtr.SetStatistics(len(projectAnalyzer.Files), projectAnalyzer.TotalSize, projectAnalyzer.TokenCount, projectAnalyzer.CharCount, duration)
				fmtr.SetTechnologies(technologies)
				fmtr.SetKeyFiles(keyFiles)
				fmtr.SetIssues(issues)
				fmtr.SetFileTypes(fileTypes)
				fmtr.SetProjectStructure(structure)
				output, err := fmtr.GetFormattedContent()
				if err != nil {
					return err
				}
				fmt.Print(output)
				return nil
			} else {
				stats, err := projectAnalyzer.ProcessFiles(outputDir, format)
				if err != nil {
					return err
				}
				printStatistics(stats, format)
				return nil
			}
		},
	}

	githubPromptCmd.Flags().String("branch", "", "Branch to clone (default: main/master)")
	githubPromptCmd.Flags().StringSliceP("exclude", "e", nil, "Exclude files matching these patterns (e.g., *.md, src/)")
	githubPromptCmd.Flags().StringSliceP("include", "i", nil, "Include only files matching these patterns")
	githubPromptCmd.Flags().String("min-size", DefaultConfig.MinSize, "Minimum file size (e.g., 1KB, 500B)")
	githubPromptCmd.Flags().String("max-size", DefaultConfig.MaxSize, "Maximum file size (e.g., 100MB, 1GB)")
	githubPromptCmd.Flags().Bool("no-gitignore", !DefaultConfig.GitIgnore, "Ignore .gitignore file")
	githubPromptCmd.Flags().StringP("output", "o", DefaultConfig.OutputDir, "Output directory for the prompt file")
	githubPromptCmd.Flags().Int("workers", DefaultConfig.WorkerCount, "Number of parallel workers (default: number of CPUs)")
	githubPromptCmd.Flags().Int("max-files", DefaultConfig.MaxFiles, "Maximum number of files to process (default: 500, 0 = unlimited)")
	githubPromptCmd.Flags().String("max-total-size", DefaultConfig.MaxTotalSize, "Maximum total size of all files (e.g., 10MB, 0 = unlimited)")
	githubPromptCmd.Flags().StringP("format", "f", DefaultConfig.OutputFormat, "Output format (txt, json, xml, or stdout[:txt|json|xml])")
	githubPromptCmd.Flags().Bool("summary-only", false, "Generate only function signatures and interfaces (AST-based summarization)")
	githubPromptCmd.Flags().Bool("focus-changes", false, "Prioritize recently modified files (Git-aware context)")
	githubPromptCmd.Flags().Int("recent-commits", 0, "Include files from last N commits (use with --focus-changes)")
	githubPromptCmd.Flags().StringSlice("summary-patterns", nil, "File patterns to summarize (e.g., vendor/**, node_modules/**)")

	// GITHUB GRAPH COMMAND
	var githubGraphCmd = &cobra.Command{
		Use:   "graph [github-url]",
		Short: "Generate dependency graphs from a GitHub repository",
		Long: `Clone and analyze a GitHub repository to generate visual dependency graphs.

This command clones a GitHub repository (using shallow clone for speed) and then
generates dependency graphs, exactly like the 'pmp graph' command.

Supported URL formats:
  • https://github.com/user/repo
  • https://github.com/user/repo.git
  • git@github.com:user/repo.git

Examples:
  pmp github graph https://github.com/user/repo
  pmp github graph https://github.com/user/repo --branch develop
  pmp github graph https://github.com/user/repo --format json
  pmp github graph https://github.com/user/repo --format stdout:dot | dot -Tpng > graph.png`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoURL := args[0]
			branch, _ := cmd.Flags().GetString("branch")

			// Create GitHub analyzer and clone
			ghAnalyzer := analyzer.NewGitHubAnalyzer(repoURL, branch)
			projectPath, err := ghAnalyzer.Clone()
			if err != nil {
				return fmt.Errorf("failed to clone repository: %w", err)
			}
			defer ghAnalyzer.Cleanup()

			// Get repository info for naming
			repoInfo, _ := ghAnalyzer.GetRepoInfo()
			repoName := repoInfo.Name
			if repoName == "" {
				repoName = "github-repo"
			}

			// Get current working directory for output
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}

			// === Copy exact logic from graphCmd ===
			format, _ := cmd.Flags().GetString("format")
			outputArg, _ := cmd.Flags().GetString("output")

			an := analyzer.New(
				projectPath,
				[]string{},
				append([]string{}, defaultExcludes...),
				0, 0, 0, 0, 1,
			)

			// Set custom project name for GitHub repos
			an.ProjectName = repoName

			if err := an.CollectFiles(); err != nil {
				return err
			}
			structure, err := an.GenerateProjectStructure()
			if err != nil {
				return err
			}
			var ext, content string
			var internalDeps, externalDeps int
			t0 := time.Now()
			if strings.HasPrefix(format, "stdout") {
				stdoutFormat := "dot"
				if strings.Contains(format, ":") {
					parts := strings.SplitN(format, ":", 2)
					if len(parts) == 2 && parts[1] != "" {
						stdoutFormat = parts[1]
					}
				}
				switch stdoutFormat {
				case "dot":
					tree := utils.BuildTree(an.Files, repoName)
					content = utils.GenerateDotOutput(tree)
				case "json":
					tree := utils.BuildTree(an.Files, repoName)
					content = utils.GenerateJSONTreeOutput(tree)
				case "xml":
					tree := utils.BuildTree(an.Files, repoName)
					content = utils.GenerateXMLTreeOutput(tree)
				case "txt":
					content = structure
				default:
					return fmt.Errorf("unsupported stdout subformat: %s", stdoutFormat)
				}
				fmt.Print(content)
				return nil
			}
			switch format {
			case "dot":
				ext = "dot"
				tree := utils.BuildTree(an.Files, repoName)
				content = utils.GenerateDotOutput(tree)
				internalDeps = countDotEdges(content)
			case "json":
				ext = "json"
				tree := utils.BuildTree(an.Files, repoName)
				content = utils.GenerateJSONTreeOutput(tree)
				internalDeps = countInternalDepsFromTree(tree)
			case "xml":
				ext = "xml"
				tree := utils.BuildTree(an.Files, repoName)
				content = utils.GenerateXMLTreeOutput(tree)
				internalDeps = countInternalDepsFromTree(tree)
			case "txt":
				ext = "txt"
				content = structure
				internalDeps = countInternalDepsFromTree(utils.BuildTree(an.Files, repoName))
			default:
				return fmt.Errorf("unsupported format: %s", format)
			}
			timestamp := time.Now().Format("20060102_150405")
			var outputPath string
			if outputArg == "" {
				// Use current working directory for output
				outputDir := filepath.Join(cwd, DefaultConfig.OutputDir)
				if info, err := os.Stat(outputDir); err == nil {
					if !info.IsDir() {
						return fmt.Errorf("output path exists and is not a directory: %s", outputDir)
					}
				} else if !os.IsNotExist(err) {
					return fmt.Errorf("failed to stat output directory: %w", err)
				}
				if err := os.MkdirAll(outputDir, 0755); err != nil {
					return fmt.Errorf("failed to create output directory: %w", err)
				}
				// Use repo name in filename
				outputPath = filepath.Join(outputDir, fmt.Sprintf("%s_graph_%s.%s", repoName, timestamp, ext))
			} else {
				info, err := os.Stat(outputArg)
				if err == nil && info.IsDir() {
					outputPath = filepath.Join(outputArg, fmt.Sprintf("%s_graph_%s.%s", repoName, timestamp, ext))
				} else if err == nil && !info.IsDir() {
					outputPath = outputArg
				} else if os.IsNotExist(err) {
					if strings.HasSuffix(outputArg, "."+ext) {
						outputPath = outputArg
					} else {
						if info, err := os.Stat(outputArg); err == nil && !info.IsDir() {
							return fmt.Errorf("output path exists and is not a directory: %s", outputArg)
						}
						if err := os.MkdirAll(outputArg, 0755); err != nil {
							return fmt.Errorf("failed to create output directory: %w", err)
						}
						outputPath = filepath.Join(outputArg, fmt.Sprintf("%s_graph_%s.%s", repoName, timestamp, ext))
					}
				} else {
					return fmt.Errorf("invalid output path: %w", err)
				}
			}
			if err := os.WriteFile(outputPath, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to write output file: %w", err)
			}
			externalDeps = countExternalDeps(projectPath)
			elapsed := time.Since(t0)
			stats := analyzer.StatsResult{
				FileCount:   len(an.Files),
				TotalSize:   an.TotalSize,
				ProcessTime: elapsed,
				OutputPath:  outputPath,
				FilesPerSec: float64(len(an.Files)) / elapsed.Seconds(),
				TokenCount:  an.TokenCount,
				CharCount:   an.CharCount,
				KeyFiles:    []string{},
				Issues:      []string{},
				FileTypes:   map[string]int{},
			}
			fmt.Println()
			printStatistics(stats, format)
			fmt.Printf("Internal dependencies: %d\n", internalDeps)
			fmt.Printf("External dependencies: %d\n", externalDeps)
			fmt.Println()
			return nil
		},
	}

	githubGraphCmd.Flags().String("branch", "", "Branch to clone (default: main/master)")
	githubGraphCmd.Flags().StringP("format", "f", "dot", "Output format for the graph (dot, json, xml, txt, or stdout[:dot|json|xml|txt])")
	githubGraphCmd.Flags().StringP("output", "o", "", "Output directory or file for the graph (default: pmp_output/)")

	// Créer une commande parente "github" pour grouper les sous-commandes
	var githubCmd = &cobra.Command{
		Use:   "github",
		Short: "Analyze GitHub repositories",
		Long: `Clone and analyze GitHub repositories directly without manual cloning.

Available commands:
  prompt - Generate AI-ready prompts from a GitHub repository
  graph  - Generate dependency graphs from a GitHub repository

These commands use shallow cloning for speed and automatically clean up temporary files.`,
	}

	githubCmd.AddCommand(githubPromptCmd)
	githubCmd.AddCommand(githubGraphCmd)

	// Ajout de la commande d'autocompletion
	var completionCmd = &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Long: `To load completions:

Bash:
  $ source <(pmp completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ pmp completion bash > /etc/bash_completion.d/pmp
  # macOS:
  $ pmp completion bash > /usr/local/etc/bash_completion.d/pmp

Zsh:
  $ echo 'autoload -U compinit; compinit' >> ~/.zshrc
  $ pmp completion zsh > "${fpath[1]}/_pmp"

Fish:
  $ pmp completion fish | source
  $ pmp completion fish > ~/.config/fish/completions/pmp.fish

PowerShell:
  PS> pmp completion powershell | Out-String | Invoke-Expression
  # To load for every session, add above to $PROFILE
`,
		Args:      cobra.ExactValidArgs(1),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Run: func(cmd *cobra.Command, args []string) {
			switch args[0] {
			case "bash":
				cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
		},
	}

	rootCmd.AddCommand(promptCmd)
	rootCmd.AddCommand(graphCmd)
	rootCmd.AddCommand(githubCmd)
	rootCmd.AddCommand(completionCmd)

	// Binary cache init/save
	binaryCache := binary.NewCache()
	if err := binaryCache.Load(); err != nil {
		fmt.Printf("Warning: error loading binary cache: %v\n", err)
	}
	defer func() {
		if err := binaryCache.Save(); err != nil {
			fmt.Printf("Warning: error saving binary cache: %v\n", err)
		}
	}()

	if err := rootCmd.Execute(); err != nil {
		color.Red("❌ Error: %v", err)
		os.Exit(1)
	}
}

// Fonctions utilitaires pour compter les dépendances internes/externes
// (à placer en bas du fichier)

// Compte les arêtes dans le DOT (dépendances internes)
func countDotEdges(dot string) int {
	count := 0
	for _, line := range strings.Split(dot, "\n") {
		if strings.Contains(line, "->") {
			count++
		}
	}
	return count
}

// Compte les dépendances internes à partir de l'arbre
func countInternalDepsFromTree(tree *utils.Directory) int {
	count := 0
	for _, sub := range tree.SubDirs {
		count += countInternalDepsFromTree(sub)
	}
	count += len(tree.Files)
	return count
}

// Compte les dépendances externes (heuristique simple)
func countExternalDeps(projectPath string) int {
	count := 0
	depFiles := []string{"go.mod", "package.json", "requirements.txt"}
	for _, depFile := range depFiles {
		path := filepath.Join(projectPath, depFile)
		if _, err := os.Stat(path); err == nil {
			// Compter les dépendances dans le fichier
			lines, err := os.ReadFile(path)
			if err == nil {
				for _, line := range strings.Split(string(lines), "\n") {
					line = strings.TrimSpace(line)
					if depFile == "go.mod" && strings.HasPrefix(line, "require ") {
						count++
					}
					if depFile == "package.json" && (strings.Contains(line, "dependencies") || strings.Contains(line, "devDependencies")) {
						count++
					}
					if depFile == "requirements.txt" && line != "" && !strings.HasPrefix(line, "#") {
						count++
					}
				}
			}
		}
	}
	return count
}

// Local copies of analyzer helpers for stdout output (since originals are not exported)
func detectTechnologiesLocal(files []string) []string {
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
		case strings.HasSuffix(file, ".c"), strings.HasSuffix(file, ".cpp"), strings.HasSuffix(file, ".h"):
			technologies["C/C++"] = true
		case strings.HasSuffix(file, ".html"):
			technologies["HTML"] = true
		case strings.HasSuffix(file, ".css"):
			technologies["CSS"] = true
		}
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
	result := make([]string, 0, len(technologies))
	for tech := range technologies {
		result = append(result, tech)
	}
	return result
}

func identifyKeyFilesLocal(files []string) []string {
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

func identifyPotentialIssuesLocal(files []string) []string {
	issues := make([]string, 0)
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

func collectFileExtensionsLocal(files []string) map[string]int {
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

func detectFileLanguageLocal(filename string) string {
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
