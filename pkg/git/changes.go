package git

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
)

// ChangeType represents the type of change in a file
type ChangeType string

const (
	ChangeTypeAdded    ChangeType = "added"
	ChangeTypeModified ChangeType = "modified"
	ChangeTypeDeleted  ChangeType = "deleted"
)

// FileChange represents a file that has been changed
type FileChange struct {
	Path       string
	ChangeType ChangeType
	IsStaged   bool
}

// ChangesAnalyzer analyzes git changes in a repository
type ChangesAnalyzer struct {
	repo     *git.Repository
	repoPath string
	worktree *git.Worktree
}

// NewChangesAnalyzer creates a new git changes analyzer
func NewChangesAnalyzer(projectPath string) (*ChangesAnalyzer, error) {
	repo, err := git.PlainOpen(projectPath)
	if err != nil {
		return nil, fmt.Errorf("not a git repository: %w", err)
	}

	worktree, err := repo.Worktree()
	if err != nil {
		return nil, fmt.Errorf("failed to get worktree: %w", err)
	}

	return &ChangesAnalyzer{
		repo:     repo,
		repoPath: projectPath,
		worktree: worktree,
	}, nil
}

// GetChangedFiles returns all files that have been modified, added, or staged
func (ca *ChangesAnalyzer) GetChangedFiles() ([]FileChange, error) {
	status, err := ca.worktree.Status()
	if err != nil {
		return nil, fmt.Errorf("failed to get git status: %w", err)
	}

	var changes []FileChange

	for filePath, fileStatus := range status {
		absPath := filepath.Join(ca.repoPath, filePath)

		change := FileChange{
			Path: absPath,
		}

		// Staging area status
		switch fileStatus.Staging {
		case git.Added:
			change.ChangeType = ChangeTypeAdded
			change.IsStaged = true
			changes = append(changes, change)
			continue
		case git.Modified:
			change.ChangeType = ChangeTypeModified
			change.IsStaged = true
			changes = append(changes, change)
			continue
		case git.Deleted:
			change.ChangeType = ChangeTypeDeleted
			change.IsStaged = true
			changes = append(changes, change)
			continue
		}

		// Worktree status (unstaged)
		switch fileStatus.Worktree {
		case git.Modified:
			change.ChangeType = ChangeTypeModified
			change.IsStaged = false
			changes = append(changes, change)
		case git.Deleted:
			change.ChangeType = ChangeTypeDeleted
			change.IsStaged = false
			changes = append(changes, change)
		case git.Untracked:
			change.ChangeType = ChangeTypeAdded
			change.IsStaged = false
			changes = append(changes, change)
		}
	}

	return changes, nil
}

// GetRecentCommitsFiles returns files changed in the last N commits
func (ca *ChangesAnalyzer) GetRecentCommitsFiles(numCommits int) ([]FileChange, error) {
	if numCommits <= 0 {
		numCommits = 1
	}

	ref, err := ca.repo.Head()
	if err != nil {
		return nil, fmt.Errorf("failed to get HEAD: %w", err)
	}

	commitIter, err := ca.repo.Log(&git.LogOptions{
		From: ref.Hash(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get commit log: %w", err)
	}

	var commits []*object.Commit
	count := 0
	err = commitIter.ForEach(func(c *object.Commit) error {
		if count < numCommits {
			commits = append(commits, c)
			count++
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to iterate commits: %w", err)
	}

	changedFilesMap := make(map[string]bool)

	for i, commit := range commits {
		var parentTree *object.Tree

		if i < len(commits)-1 {
			parentTree, _ = commits[i+1].Tree()
		}

		currentTree, err := commit.Tree()
		if err != nil {
			continue
		}

		if parentTree == nil {
			err = currentTree.Files().ForEach(func(f *object.File) error {
				absPath := filepath.Join(ca.repoPath, f.Name)
				changedFilesMap[absPath] = true
				return nil
			})
			if err != nil {
				continue
			}
		} else {
			changes, err := parentTree.Diff(currentTree)
			if err != nil {
				continue
			}

			for _, change := range changes {
				var path string
				if change.To.Name != "" {
					path = change.To.Name
				} else if change.From.Name != "" {
					path = change.From.Name
				}

				if path != "" {
					absPath := filepath.Join(ca.repoPath, path)
					changedFilesMap[absPath] = true
				}
			}
		}
	}

	var changes []FileChange
	for path := range changedFilesMap {
		changes = append(changes, FileChange{
			Path:       path,
			ChangeType: ChangeTypeModified,
			IsStaged:   false,
		})
	}

	return changes, nil
}

// GetAllChangedFiles returns all changed files including unstaged, staged, and recent commits
func (ca *ChangesAnalyzer) GetAllChangedFiles(includeRecentCommits bool, numCommits int) ([]FileChange, error) {
	changedFilesMap := make(map[string]FileChange)

	currentChanges, err := ca.GetChangedFiles()
	if err != nil {
		return nil, err
	}

	for _, change := range currentChanges {
		changedFilesMap[change.Path] = change
	}

	if includeRecentCommits && numCommits > 0 {
		recentChanges, err := ca.GetRecentCommitsFiles(numCommits)
		if err != nil {
			fmt.Printf("Warning: failed to get recent commits: %v\n", err)
		} else {
			for _, change := range recentChanges {
				if _, exists := changedFilesMap[change.Path]; !exists {
					changedFilesMap[change.Path] = change
				}
			}
		}
	}

	var allChanges []FileChange
	for _, change := range changedFilesMap {
		allChanges = append(allChanges, change)
	}

	return allChanges, nil
}

// IsGitRepository checks if the given path is a git repository
func IsGitRepository(projectPath string) bool {
	_, err := git.PlainOpen(projectPath)
	return err == nil
}

// GetBranchName returns the current branch name
func (ca *ChangesAnalyzer) GetBranchName() (string, error) {
	head, err := ca.repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	if head.Name().IsBranch() {
		return head.Name().Short(), nil
	}

	return head.Hash().String()[:7], nil
}

// HasChanges returns true if there are any staged or unstaged changes
func (ca *ChangesAnalyzer) HasChanges() (bool, error) {
	status, err := ca.worktree.Status()
	if err != nil {
		return false, err
	}

	return !status.IsClean(), nil
}

// GetLastCommitMessage returns the message of the last commit
func (ca *ChangesAnalyzer) GetLastCommitMessage() (string, error) {
	ref, err := ca.repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	commit, err := ca.repo.CommitObject(ref.Hash())
	if err != nil {
		return "", fmt.Errorf("failed to get commit: %w", err)
	}

	return strings.TrimSpace(commit.Message), nil
}

// GetCommitHash returns the hash of the current commit
func (ca *ChangesAnalyzer) GetCommitHash() (string, error) {
	ref, err := ca.repo.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	return ref.Hash().String()[:7], nil
}

// FormatChangeSummary returns a formatted summary of changes for display
func FormatChangeSummary(changes []FileChange, repoPath string) string {
	if len(changes) == 0 {
		return "No changes detected"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d changed file(s):\n", len(changes)))

	staged := 0
	unstaged := 0

	for _, change := range changes {
		relPath, err := filepath.Rel(repoPath, change.Path)
		if err != nil {
			relPath = change.Path
		}

		status := ""
		if change.IsStaged {
			status = "[staged]"
			staged++
		} else {
			status = "[unstaged]"
			unstaged++
		}

		sb.WriteString(fmt.Sprintf("  %s %s (%s)\n", status, relPath, change.ChangeType))
	}

	sb.WriteString(fmt.Sprintf("\nSummary: %d staged, %d unstaged\n", staged, unstaged))

	return sb.String()
}
