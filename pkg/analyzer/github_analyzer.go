package analyzer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
)

// GitHubAnalyzer handles analysis of GitHub repositories
type GitHubAnalyzer struct {
	repoURL string
	branch  string
	tempDir string
	cleanup bool
	silent  bool
}

// NewGitHubAnalyzer creates a new GitHub analyzer
func NewGitHubAnalyzer(repoURL string, branch string) *GitHubAnalyzer {
	return &GitHubAnalyzer{
		repoURL: repoURL,
		branch:  branch,
		cleanup: true,
		silent:  false,
	}
}

// SetCleanup sets whether to clean up the temporary directory
func (ga *GitHubAnalyzer) SetCleanup(cleanup bool) {
	ga.cleanup = cleanup
}

// SetSilent sets whether to suppress output messages
func (ga *GitHubAnalyzer) SetSilent(silent bool) {
	ga.silent = silent
}

// Clone clones the GitHub repository to a temporary directory
func (ga *GitHubAnalyzer) Clone() (string, error) {
	// Extract repo name for better temp directory naming
	repoInfo, err := ga.GetRepoInfo()
	repoName := "pmp-github"
	if err == nil && repoInfo.Name != "" {
		repoName = repoInfo.Name
	}

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", repoName+"-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	ga.tempDir = tempDir

	// Clone options
	cloneOpts := &git.CloneOptions{
		URL:   ga.repoURL,
		Depth: 1, // Shallow clone for speed
	}

	// Only show progress if not silent
	if !ga.silent {
		cloneOpts.Progress = os.Stderr
	}

	// If a specific branch is requested
	if ga.branch != "" && ga.branch != "main" && ga.branch != "master" {
		cloneOpts.ReferenceName = plumbing.NewBranchReferenceName(ga.branch)
		cloneOpts.SingleBranch = true
	}

	if !ga.silent {
		fmt.Fprintf(os.Stderr, "Cloning repository %s...\n", ga.repoURL)
	}

	// Clone the repository
	_, err = git.PlainClone(tempDir, false, cloneOpts)
	if err != nil {
		// Try with master if main didn't work
		if ga.branch == "" || ga.branch == "main" {
			cloneOpts.ReferenceName = plumbing.NewBranchReferenceName("master")
			_, err = git.PlainClone(tempDir, false, cloneOpts)
		}

		if err != nil {
			os.RemoveAll(tempDir)
			return "", fmt.Errorf("failed to clone repository: %w", err)
		}
	}

	if !ga.silent {
		fmt.Fprintf(os.Stderr, "Repository cloned to: %s\n", tempDir)
	}

	return tempDir, nil
}

// Cleanup removes the temporary directory
func (ga *GitHubAnalyzer) Cleanup() error {
	if ga.tempDir == "" || !ga.cleanup {
		return nil
	}

	if !ga.silent {
		fmt.Fprintf(os.Stderr, "Cleaning up temporary directory: %s\n", ga.tempDir)
	}
	return os.RemoveAll(ga.tempDir)
}

// GetRepoInfo extracts information about the GitHub repository
func (ga *GitHubAnalyzer) GetRepoInfo() (*RepoInfo, error) {
	info := &RepoInfo{
		URL: ga.repoURL,
	}

	// Parse owner and repo name from URL
	// Supports:
	// - https://github.com/owner/repo
	// - https://github.com/owner/repo.git
	// - git@github.com:owner/repo.git

	url := ga.repoURL

	// Remove .git suffix
	url = strings.TrimSuffix(url, ".git")

	// Handle SSH URL
	if strings.HasPrefix(url, "git@github.com:") {
		url = strings.TrimPrefix(url, "git@github.com:")
		parts := strings.Split(url, "/")
		if len(parts) >= 2 {
			info.Owner = parts[0]
			info.Name = parts[1]
		}
	} else {
		// Handle HTTPS URL
		url = strings.TrimPrefix(url, "https://")
		url = strings.TrimPrefix(url, "http://")
		url = strings.TrimPrefix(url, "github.com/")

		parts := strings.Split(url, "/")
		if len(parts) >= 2 {
			info.Owner = parts[0]
			info.Name = parts[1]
		}
	}

	if ga.branch != "" {
		info.Branch = ga.branch
	} else {
		info.Branch = "main"
	}

	return info, nil
}

// RepoInfo contains information about a GitHub repository
type RepoInfo struct {
	URL    string `json:"url"`
	Owner  string `json:"owner"`
	Name   string `json:"name"`
	Branch string `json:"branch"`
}

// AnalyzeGitHubRepo clones and analyzes a GitHub repository
func AnalyzeGitHubRepo(repoURL string, branch string, includePatterns, excludePatterns []string, format string, outputDir string) (*StatsResult, error) {
	// Create GitHub analyzer
	ga := NewGitHubAnalyzer(repoURL, branch)

	// Clone repository
	tempDir, err := ga.Clone()
	if err != nil {
		return nil, err
	}

	// Defer cleanup
	defer ga.Cleanup()

	// Get repo info
	repoInfo, _ := ga.GetRepoInfo()

	fmt.Printf("\nAnalyzing repository: %s/%s\n", repoInfo.Owner, repoInfo.Name)
	fmt.Println(strings.Repeat("=", 50))

	// Create analyzer for the cloned repository
	analyzer := New(
		tempDir,
		includePatterns,
		excludePatterns,
		1024,          // min size 1KB
		100*1024*1024, // max size 100MB
		500,           // max files
		10*1024*1024,  // max total size 10MB
		4,             // workers
	)

	// Collect files
	if err := analyzer.CollectFiles(); err != nil {
		return nil, fmt.Errorf("failed to collect files: %w", err)
	}

	fmt.Printf("Found %d files to analyze\n\n", len(analyzer.Files))

	// Process files
	stats, err := analyzer.ProcessFiles(outputDir, format)
	if err != nil {
		return nil, fmt.Errorf("failed to process files: %w", err)
	}

	// Add GitHub info to stats
	fmt.Printf("\nAnalysis complete for GitHub repository!\n")
	fmt.Printf("Repository: %s/%s (branch: %s)\n", repoInfo.Owner, repoInfo.Name, repoInfo.Branch)

	return &stats, nil
}

// ValidateGitHubURL validates that a URL is a valid GitHub repository URL
func ValidateGitHubURL(url string) error {
	if url == "" {
		return fmt.Errorf("repository URL cannot be empty")
	}

	// Check if it's a GitHub URL
	if !strings.Contains(url, "github.com") {
		return fmt.Errorf("URL must be a GitHub repository URL (contains github.com)")
	}

	// Basic validation
	if !strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "git@") {
		return fmt.Errorf("URL must start with https://, http://, or git@")
	}

	return nil
}

// ParseGitHubURL extracts owner and repo name from GitHub URL
func ParseGitHubURL(url string) (owner string, repo string, err error) {
	// Remove .git suffix
	url = strings.TrimSuffix(url, ".git")

	// Handle SSH URL
	if strings.HasPrefix(url, "git@github.com:") {
		url = strings.TrimPrefix(url, "git@github.com:")
		parts := strings.Split(url, "/")
		if len(parts) >= 2 {
			return parts[0], parts[1], nil
		}
	}

	// Handle HTTPS URL
	url = strings.TrimPrefix(url, "https://")
	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimPrefix(url, "github.com/")

	parts := strings.Split(url, "/")
	if len(parts) >= 2 {
		return parts[0], parts[1], nil
	}

	return "", "", fmt.Errorf("invalid GitHub URL format")
}

// GetDefaultBranch attempts to determine the default branch of a repository
func GetDefaultBranch(repoPath string) (string, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return "main", err
	}

	head, err := repo.Head()
	if err != nil {
		return "main", err
	}

	branchName := head.Name().Short()
	return branchName, nil
}

// ListBranches lists all branches in a repository
func ListBranches(repoPath string) ([]string, error) {
	repo, err := git.PlainOpen(repoPath)
	if err != nil {
		return nil, err
	}

	branches := []string{}

	refs, err := repo.Branches()
	if err != nil {
		return nil, err
	}

	err = refs.ForEach(func(ref *plumbing.Reference) error {
		branches = append(branches, ref.Name().Short())
		return nil
	})

	return branches, err
}

// GetRepoSize calculates the size of a cloned repository
func GetRepoSize(repoPath string) (int64, error) {
	var size int64

	err := filepath.Walk(repoPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .git directory
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		if !info.IsDir() {
			size += info.Size()
		}

		return nil
	})

	return size, err
}

// CloneOptions represents options for cloning a repository
type CloneOptions struct {
	URL          string
	Branch       string
	Depth        int
	SingleBranch bool
	OutputDir    string
	Verbose      bool
}

// CloneWithOptions clones a repository with custom options
func CloneWithOptions(opts CloneOptions) (string, error) {
	// Determine output directory
	outputDir := opts.OutputDir
	if outputDir == "" {
		var err error
		outputDir, err = os.MkdirTemp("", "pmp-clone-*")
		if err != nil {
			return "", fmt.Errorf("failed to create temp directory: %w", err)
		}
	}

	// Setup clone options
	cloneOpts := &git.CloneOptions{
		URL: opts.URL,
	}

	if opts.Verbose {
		cloneOpts.Progress = os.Stdout
	}

	if opts.Depth > 0 {
		cloneOpts.Depth = opts.Depth
	}

	if opts.Branch != "" {
		cloneOpts.ReferenceName = plumbing.NewBranchReferenceName(opts.Branch)
		cloneOpts.SingleBranch = opts.SingleBranch
	}

	// Clone
	_, err := git.PlainClone(outputDir, false, cloneOpts)
	if err != nil {
		os.RemoveAll(outputDir)
		return "", fmt.Errorf("failed to clone: %w", err)
	}

	return outputDir, nil
}
