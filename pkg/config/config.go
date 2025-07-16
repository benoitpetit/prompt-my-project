package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// Config represents the configuration structure
type Config struct {
	Exclude       []string `json:"exclude,omitempty"`
	Include       []string `json:"include,omitempty"`
	MinSize       string   `json:"minSize,omitempty"`
	MaxSize       string   `json:"maxSize,omitempty"`
	MaxFiles      int      `json:"maxFiles,omitempty"`
	MaxTotalSize  string   `json:"maxTotalSize,omitempty"`
	Format        string   `json:"format,omitempty"`
	OutputDir     string   `json:"outputDir,omitempty"`
	Workers       int      `json:"workers,omitempty"`
	NoGitignore   bool     `json:"noGitignore,omitempty"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		MinSize:      "1KB",
		MaxSize:      "100MB",
		MaxFiles:     500,
		MaxTotalSize: "10MB",
		Format:       "txt",
		OutputDir:    "pmp_output",
		Workers:      runtime.NumCPU(),
		NoGitignore:  false,
	}
}

// LoadConfig loads configuration from .pmprc file if it exists
func LoadConfig(projectPath string) (*Config, error) {
	config := DefaultConfig()
	
	// Look for .pmprc in the project directory
	configPath := filepath.Join(projectPath, ".pmprc")
	
	// Check if .pmprc exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return config, nil // Return default config if no .pmprc found
	}
	
	// Read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		return config, err
	}
	
	// Parse JSON
	var fileConfig Config
	if err := json.Unmarshal(data, &fileConfig); err != nil {
		return config, err
	}
	
	// Merge file config with default config
	if len(fileConfig.Exclude) > 0 {
		config.Exclude = fileConfig.Exclude
	}
	if len(fileConfig.Include) > 0 {
		config.Include = fileConfig.Include
	}
	if fileConfig.MinSize != "" {
		config.MinSize = fileConfig.MinSize
	}
	if fileConfig.MaxSize != "" {
		config.MaxSize = fileConfig.MaxSize
	}
	if fileConfig.MaxFiles > 0 {
		config.MaxFiles = fileConfig.MaxFiles
	}
	if fileConfig.MaxTotalSize != "" {
		config.MaxTotalSize = fileConfig.MaxTotalSize
	}
	if fileConfig.Format != "" {
		config.Format = fileConfig.Format
	}
	if fileConfig.OutputDir != "" {
		config.OutputDir = fileConfig.OutputDir
	}
	if fileConfig.Workers > 0 {
		config.Workers = fileConfig.Workers
	}
	config.NoGitignore = fileConfig.NoGitignore
	
	return config, nil
}

// MergeWithFlags merges configuration with command-line flags
func (c *Config) MergeWithFlags(
	excludePatterns, includePatterns []string,
	minSize, maxSize, maxTotalSize, format, outputDir string,
	maxFiles, workers int,
	noGitignore bool,
) {
	// Command-line flags take precedence over config file
	if len(excludePatterns) > 0 {
		c.Exclude = excludePatterns
	}
	if len(includePatterns) > 0 {
		c.Include = includePatterns
	}
	if minSize != "" {
		c.MinSize = minSize
	}
	if maxSize != "" {
		c.MaxSize = maxSize
	}
	if maxTotalSize != "" {
		c.MaxTotalSize = maxTotalSize
	}
	if format != "" {
		c.Format = format
	}
	if outputDir != "" {
		c.OutputDir = outputDir
	}
	if maxFiles > 0 {
		c.MaxFiles = maxFiles
	}
	if workers > 0 {
		c.Workers = workers
	}
	if noGitignore {
		c.NoGitignore = noGitignore
	}
}

// GetEnvironmentConfig loads configuration from environment variables
func GetEnvironmentConfig() *Config {
	config := DefaultConfig()
	
	if outputDir := os.Getenv("PMP_OUTPUT_DIR"); outputDir != "" {
		config.OutputDir = outputDir
	}
	
	if workersStr := os.Getenv("PMP_WORKERS"); workersStr != "" {
		if workers := parseInt(workersStr); workers > 0 {
			config.Workers = workers
		}
	}
	
	if format := os.Getenv("PMP_FORMAT"); format != "" {
		config.Format = format
	}
	
	if maxFiles := os.Getenv("PMP_MAX_FILES"); maxFiles != "" {
		if files := parseInt(maxFiles); files > 0 {
			config.MaxFiles = files
		}
	}
	
	if maxTotalSize := os.Getenv("PMP_MAX_TOTAL_SIZE"); maxTotalSize != "" {
		config.MaxTotalSize = maxTotalSize
	}
	
	if minSize := os.Getenv("PMP_MIN_SIZE"); minSize != "" {
		config.MinSize = minSize
	}
	
	if maxSize := os.Getenv("PMP_MAX_SIZE"); maxSize != "" {
		config.MaxSize = maxSize
	}
	
	if excludeStr := os.Getenv("PMP_EXCLUDE"); excludeStr != "" {
		config.Exclude = strings.Split(excludeStr, ",")
		for i := range config.Exclude {
			config.Exclude[i] = strings.TrimSpace(config.Exclude[i])
		}
	}
	
	if includeStr := os.Getenv("PMP_INCLUDE"); includeStr != "" {
		config.Include = strings.Split(includeStr, ",")
		for i := range config.Include {
			config.Include[i] = strings.TrimSpace(config.Include[i])
		}
	}
	
	return config
}

// parseInt is a simple integer parser
func parseInt(s string) int {
	var result int
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result = result*10 + int(c-'0')
		} else {
			return 0
		}
	}
	return result
}
