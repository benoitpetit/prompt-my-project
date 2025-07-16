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
	var rootCmd = &cobra.Command{
		Use:   "pmp",
		Short: "Generate a project prompt or dependency graph for AI",
		Long: `Prompt My Project (PMP) analyzes your project and generates a formatted prompt for AI assistants, or a dependency graph.

Usage examples:
   pmp prompt ./path/to/project [options]          # Generate prompt
   pmp graph ./path/to/project [options]           # Generate dependency graph in chosen format`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// PROMPT COMMAND
	var promptCmd = &cobra.Command{
		Use:   "prompt [project path]",
		Short: "Generate a project prompt for AI",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Flags
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

			// Path
			dir := args[0]
			if !filepath.IsAbs(dir) {
				absDir, err := filepath.Abs(dir)
				if err == nil {
					dir = absDir
				}
			}

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
				printStatistics(stats)
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

	// GRAPH COMMAND
	var graphCmd = &cobra.Command{
		Use:   "graph [project path]",
		Short: "Generate a project dependency graph/arborescence",
		Args:  cobra.ExactArgs(1),
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
			printStatistics(stats)
			fmt.Printf("Internal dependencies: %d\n", internalDeps)
			fmt.Printf("External dependencies: %d\n", externalDeps)
			fmt.Println()
			return nil
		},
	}
	graphCmd.Flags().StringP("format", "f", "dot", "Output format for the graph (dot, json, xml, txt, or stdout[:dot|json|xml|txt])")
	graphCmd.Flags().StringP("output", "o", "", "Output directory or file for the graph (default: pmp_output/)")

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
