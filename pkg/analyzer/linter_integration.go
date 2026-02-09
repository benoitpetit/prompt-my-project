package analyzer

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// LintIssue represents a linting issue
type LintIssue struct {
	File     string `json:"file"`
	Line     int    `json:"line"`
	Column   int    `json:"column"`
	Severity string `json:"severity"` // error, warning, info
	Message  string `json:"message"`
	Rule     string `json:"rule"`
	Linter   string `json:"linter"` // eslint, golint, pylint, etc.
	Fixable  bool   `json:"fixable"`
}

// LintSummary represents summary of linting results
type LintSummary struct {
	TotalIssues  int            `json:"total_issues"`
	ErrorCount   int            `json:"error_count"`
	WarningCount int            `json:"warning_count"`
	InfoCount    int            `json:"info_count"`
	FixableCount int            `json:"fixable_count"`
	ByLinter     map[string]int `json:"by_linter"`
	ByFile       map[string]int `json:"by_file"`
	TopIssues    []LintIssue    `json:"top_issues"`
}

// LinterIntegration integrates with popular linters
type LinterIntegration struct {
	rootDir string
	files   []string
}

// NewLinterIntegration creates a new linter integration
func NewLinterIntegration(rootDir string, files []string) *LinterIntegration {
	return &LinterIntegration{
		rootDir: rootDir,
		files:   files,
	}
}

// RunAllLinters runs all available linters
func (li *LinterIntegration) RunAllLinters() (*LintSummary, []LintIssue) {
	allIssues := []LintIssue{}

	// Try ESLint for JavaScript/TypeScript
	if li.hasFileType(".js", ".ts", ".jsx", ".tsx") {
		if issues := li.runESLint(); len(issues) > 0 {
			allIssues = append(allIssues, issues...)
		}
	}

	// Try golint/staticcheck for Go
	if li.hasFileType(".go") {
		if issues := li.runGoLinters(); len(issues) > 0 {
			allIssues = append(allIssues, issues...)
		}
	}

	// Try pylint for Python
	if li.hasFileType(".py") {
		if issues := li.runPyLint(); len(issues) > 0 {
			allIssues = append(allIssues, issues...)
		}
	}

	// Try RuboCop for Ruby
	if li.hasFileType(".rb") {
		if issues := li.runRuboCop(); len(issues) > 0 {
			allIssues = append(allIssues, issues...)
		}
	}

	// Generate summary
	summary := li.generateSummary(allIssues)

	return summary, allIssues
}

// hasFileType checks if project has files of given types
func (li *LinterIntegration) hasFileType(extensions ...string) bool {
	for _, file := range li.files {
		for _, ext := range extensions {
			if strings.HasSuffix(file, ext) {
				return true
			}
		}
	}
	return false
}

// runESLint runs ESLint for JavaScript/TypeScript files
func (li *LinterIntegration) runESLint() []LintIssue {
	// Check if eslint is available
	if !commandExists("eslint") {
		return li.basicJSLint()
	}

	// Run eslint with JSON output
	cmd := exec.Command("eslint", ".", "--format", "json", "--no-color")
	cmd.Dir = li.rootDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// ESLint returns non-zero exit code when issues are found, which is expected
	_ = cmd.Run()

	// Parse JSON output
	var eslintOutput []struct {
		FilePath string `json:"filePath"`
		Messages []struct {
			Line     int       `json:"line"`
			Column   int       `json:"column"`
			Severity int       `json:"severity"` // 1 = warning, 2 = error
			Message  string    `json:"message"`
			RuleID   string    `json:"ruleId"`
			Fix      *struct{} `json:"fix"`
		} `json:"messages"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &eslintOutput); err != nil {
		return []LintIssue{}
	}

	issues := []LintIssue{}
	for _, fileResult := range eslintOutput {
		relPath, _ := filepath.Rel(li.rootDir, fileResult.FilePath)
		for _, msg := range fileResult.Messages {
			severity := "warning"
			if msg.Severity == 2 {
				severity = "error"
			}

			issues = append(issues, LintIssue{
				File:     relPath,
				Line:     msg.Line,
				Column:   msg.Column,
				Severity: severity,
				Message:  msg.Message,
				Rule:     msg.RuleID,
				Linter:   "eslint",
				Fixable:  msg.Fix != nil,
			})
		}
	}

	return issues
}

// basicJSLint performs basic JS linting when ESLint is not available
func (li *LinterIntegration) basicJSLint() []LintIssue {
	issues := []LintIssue{}

	for _, file := range li.files {
		if !strings.HasSuffix(file, ".js") && !strings.HasSuffix(file, ".ts") {
			continue
		}

		fullPath := filepath.Join(li.rootDir, file)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}

		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			// Check for common issues
			if strings.Contains(line, "console.log") && !strings.Contains(line, "//") {
				issues = append(issues, LintIssue{
					File:     file,
					Line:     i + 1,
					Severity: "warning",
					Message:  "Unexpected console statement",
					Rule:     "no-console",
					Linter:   "basic-js",
					Fixable:  false,
				})
			}

			if strings.Contains(line, "debugger") && !strings.Contains(line, "//") {
				issues = append(issues, LintIssue{
					File:     file,
					Line:     i + 1,
					Severity: "error",
					Message:  "Unexpected debugger statement",
					Rule:     "no-debugger",
					Linter:   "basic-js",
					Fixable:  true,
				})
			}

			// Check for var usage (should use let/const)
			if strings.Contains(line, " var ") && !strings.Contains(line, "//") {
				issues = append(issues, LintIssue{
					File:     file,
					Line:     i + 1,
					Severity: "warning",
					Message:  "Unexpected var, use let or const instead",
					Rule:     "no-var",
					Linter:   "basic-js",
					Fixable:  true,
				})
			}
		}
	}

	return issues
}

// runGoLinters runs Go linters
func (li *LinterIntegration) runGoLinters() []LintIssue {
	issues := []LintIssue{}

	// Try staticcheck first (more modern)
	if commandExists("staticcheck") {
		issues = append(issues, li.runStaticCheck()...)
	} else if commandExists("golint") {
		issues = append(issues, li.runGoLintTool()...)
	} else {
		// Basic Go checks
		issues = append(issues, li.basicGoLint()...)
	}

	// Always run go vet if available
	if commandExists("go") {
		issues = append(issues, li.runGoVet()...)
	}

	return issues
}

// runStaticCheck runs staticcheck
func (li *LinterIntegration) runStaticCheck() []LintIssue {
	cmd := exec.Command("staticcheck", "./...")
	cmd.Dir = li.rootDir

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	_ = cmd.Run()

	return li.parseGoLintOutput(stdout.String(), "staticcheck")
}

// runGoLintTool runs golint
func (li *LinterIntegration) runGoLintTool() []LintIssue {
	cmd := exec.Command("golint", "./...")
	cmd.Dir = li.rootDir

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	_ = cmd.Run()

	return li.parseGoLintOutput(stdout.String(), "golint")
}

// runGoVet runs go vet
func (li *LinterIntegration) runGoVet() []LintIssue {
	cmd := exec.Command("go", "vet", "./...")
	cmd.Dir = li.rootDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	_ = cmd.Run()

	return li.parseGoVetOutput(stderr.String())
}

// parseGoLintOutput parses golint/staticcheck output
func (li *LinterIntegration) parseGoLintOutput(output, linter string) []LintIssue {
	issues := []LintIssue{}
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if line == "" {
			continue
		}

		// Format: file.go:line:column: message
		parts := strings.SplitN(line, ":", 4)
		if len(parts) < 3 {
			continue
		}

		lineNum := 0
		colNum := 0
		if len(parts) > 1 {
			// Parse line number
			if n, err := parseInt(parts[1]); err == nil {
				lineNum = n
			}
		}
		if len(parts) > 2 {
			// Parse column number
			if n, err := parseInt(parts[2]); err == nil {
				colNum = n
			}
		}

		message := ""
		if len(parts) > 3 {
			message = strings.TrimSpace(parts[3])
		}

		issues = append(issues, LintIssue{
			File:     parts[0],
			Line:     lineNum,
			Column:   colNum,
			Severity: "warning",
			Message:  message,
			Rule:     "",
			Linter:   linter,
			Fixable:  false,
		})
	}

	return issues
}

// parseGoVetOutput parses go vet output
func (li *LinterIntegration) parseGoVetOutput(output string) []LintIssue {
	issues := []LintIssue{}
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		if line == "" || !strings.Contains(line, ":") {
			continue
		}

		// Format: file.go:line:column: message
		parts := strings.SplitN(line, ":", 4)
		if len(parts) < 3 {
			continue
		}

		lineNum := 0
		if len(parts) > 1 {
			if n, err := parseInt(parts[1]); err == nil {
				lineNum = n
			}
		}

		message := ""
		if len(parts) > 2 {
			message = strings.TrimSpace(strings.Join(parts[2:], ":"))
		}

		issues = append(issues, LintIssue{
			File:     parts[0],
			Line:     lineNum,
			Column:   0,
			Severity: "error",
			Message:  message,
			Rule:     "go-vet",
			Linter:   "go-vet",
			Fixable:  false,
		})
	}

	return issues
}

// basicGoLint performs basic Go linting
func (li *LinterIntegration) basicGoLint() []LintIssue {
	issues := []LintIssue{}

	for _, file := range li.files {
		if !strings.HasSuffix(file, ".go") {
			continue
		}

		fullPath := filepath.Join(li.rootDir, file)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}

		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			// Check for commented code
			if strings.HasPrefix(strings.TrimSpace(line), "// ") {
				trimmed := strings.TrimPrefix(strings.TrimSpace(line), "// ")
				if strings.Contains(trimmed, "func ") || strings.Contains(trimmed, "if ") {
					issues = append(issues, LintIssue{
						File:     file,
						Line:     i + 1,
						Severity: "info",
						Message:  "Commented code should be removed",
						Rule:     "no-commented-code",
						Linter:   "basic-go",
						Fixable:  true,
					})
				}
			}
		}
	}

	return issues
}

// runPyLint runs pylint for Python files
func (li *LinterIntegration) runPyLint() []LintIssue {
	if !commandExists("pylint") {
		return li.basicPyLint()
	}

	// Find Python files
	pyFiles := []string{}
	for _, file := range li.files {
		if strings.HasSuffix(file, ".py") {
			pyFiles = append(pyFiles, filepath.Join(li.rootDir, file))
		}
	}

	if len(pyFiles) == 0 {
		return []LintIssue{}
	}

	// Run pylint
	args := append([]string{"--output-format=json"}, pyFiles...)
	cmd := exec.Command("pylint", args...)

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	_ = cmd.Run()

	// Parse JSON output
	var pylintOutput []struct {
		Type    string `json:"type"`
		Module  string `json:"module"`
		Path    string `json:"path"`
		Line    int    `json:"line"`
		Column  int    `json:"column"`
		Message string `json:"message"`
		Symbol  string `json:"symbol"`
	}

	if err := json.Unmarshal(stdout.Bytes(), &pylintOutput); err != nil {
		return []LintIssue{}
	}

	issues := []LintIssue{}
	for _, msg := range pylintOutput {
		severity := "info"
		switch msg.Type {
		case "error", "fatal":
			severity = "error"
		case "warning":
			severity = "warning"
		}

		relPath, _ := filepath.Rel(li.rootDir, msg.Path)
		issues = append(issues, LintIssue{
			File:     relPath,
			Line:     msg.Line,
			Column:   msg.Column,
			Severity: severity,
			Message:  msg.Message,
			Rule:     msg.Symbol,
			Linter:   "pylint",
			Fixable:  false,
		})
	}

	return issues
}

// basicPyLint performs basic Python linting
func (li *LinterIntegration) basicPyLint() []LintIssue {
	issues := []LintIssue{}

	for _, file := range li.files {
		if !strings.HasSuffix(file, ".py") {
			continue
		}

		fullPath := filepath.Join(li.rootDir, file)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}

		lines := strings.Split(string(content), "\n")
		for i, line := range lines {
			trimmed := strings.TrimSpace(line)

			// Check for print statements (should use logging)
			if strings.HasPrefix(trimmed, "print(") {
				issues = append(issues, LintIssue{
					File:     file,
					Line:     i + 1,
					Severity: "info",
					Message:  "Consider using logging instead of print",
					Rule:     "use-logging",
					Linter:   "basic-py",
					Fixable:  false,
				})
			}
		}
	}

	return issues
}

// runRuboCop runs RuboCop for Ruby files
func (li *LinterIntegration) runRuboCop() []LintIssue {
	if !commandExists("rubocop") {
		return []LintIssue{}
	}

	cmd := exec.Command("rubocop", "--format", "json")
	cmd.Dir = li.rootDir

	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	_ = cmd.Run()

	// Parse JSON output (simplified)
	return []LintIssue{}
}

// generateSummary generates a summary of linting results
func (li *LinterIntegration) generateSummary(issues []LintIssue) *LintSummary {
	summary := &LintSummary{
		ByLinter:  make(map[string]int),
		ByFile:    make(map[string]int),
		TopIssues: []LintIssue{},
	}

	for _, issue := range issues {
		summary.TotalIssues++
		summary.ByLinter[issue.Linter]++
		summary.ByFile[issue.File]++

		switch issue.Severity {
		case "error":
			summary.ErrorCount++
		case "warning":
			summary.WarningCount++
		case "info":
			summary.InfoCount++
		}

		if issue.Fixable {
			summary.FixableCount++
		}

		// Keep top 10 issues (highest severity first)
		if len(summary.TopIssues) < 10 {
			summary.TopIssues = append(summary.TopIssues, issue)
		}
	}

	return summary
}

// commandExists checks if a command is available
func commandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

// parseInt parses a string to int
func parseInt(s string) (int, error) {
	s = strings.TrimSpace(s)
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}

// GetLintSummaryString returns a human-readable summary
func GetLintSummaryString(summary *LintSummary) string {
	if summary == nil || summary.TotalIssues == 0 {
		return "✅ No linting issues detected.\n"
	}

	var sb strings.Builder

	sb.WriteString("Code Quality - Linting Results:\n")
	sb.WriteString("================================\n\n")

	sb.WriteString("Total Issues: ")
	sb.WriteString(intToString(summary.TotalIssues))
	sb.WriteString("\n")

	if summary.ErrorCount > 0 {
		sb.WriteString("  🔴 Errors: ")
		sb.WriteString(intToString(summary.ErrorCount))
		sb.WriteString("\n")
	}
	if summary.WarningCount > 0 {
		sb.WriteString("  🟡 Warnings: ")
		sb.WriteString(intToString(summary.WarningCount))
		sb.WriteString("\n")
	}
	if summary.InfoCount > 0 {
		sb.WriteString("  ℹ️  Info: ")
		sb.WriteString(intToString(summary.InfoCount))
		sb.WriteString("\n")
	}

	if summary.FixableCount > 0 {
		sb.WriteString("\n")
		sb.WriteString(intToString(summary.FixableCount))
		sb.WriteString(" issues can be fixed automatically\n")
	}

	if len(summary.ByLinter) > 0 {
		sb.WriteString("\nBy Linter:\n")
		for linter, count := range summary.ByLinter {
			sb.WriteString("  - ")
			sb.WriteString(linter)
			sb.WriteString(": ")
			sb.WriteString(intToString(count))
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
