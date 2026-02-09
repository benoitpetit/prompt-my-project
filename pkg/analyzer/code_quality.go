package analyzer

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// CodeQualityMetrics represents code quality metrics for a project
type CodeQualityMetrics struct {
	TotalFiles           int                         `json:"total_files"`
	TotalLines           int                         `json:"total_lines"`
	CodeLines            int                         `json:"code_lines"`
	CommentLines         int                         `json:"comment_lines"`
	BlankLines           int                         `json:"blank_lines"`
	AverageComplexity    float64                     `json:"average_complexity"`
	TestCoverage         float64                     `json:"test_coverage"`
	TestFileCount        int                         `json:"test_file_count"`
	CodeSmells           []CodeSmell                 `json:"code_smells,omitempty"`
	MaintainabilityIndex float64                     `json:"maintainability_index"`
	FileMetrics          map[string]*FileMetrics     `json:"file_metrics,omitempty"`
	LanguageMetrics      map[string]*LanguageMetrics `json:"language_metrics"`
}

// FileMetrics represents metrics for a single file
type FileMetrics struct {
	FilePath             string   `json:"file_path"`
	Lines                int      `json:"lines"`
	CodeLines            int      `json:"code_lines"`
	CommentLines         int      `json:"comment_lines"`
	BlankLines           int      `json:"blank_lines"`
	CyclomaticComplexity int      `json:"cyclomatic_complexity"`
	Functions            int      `json:"functions"`
	Classes              int      `json:"classes"`
	MaxNestingDepth      int      `json:"max_nesting_depth"`
	CodeSmells           []string `json:"code_smells,omitempty"`
}

// LanguageMetrics represents metrics aggregated by language
type LanguageMetrics struct {
	Language      string  `json:"language"`
	FileCount     int     `json:"file_count"`
	TotalLines    int     `json:"total_lines"`
	CodeLines     int     `json:"code_lines"`
	CommentLines  int     `json:"comment_lines"`
	BlankLines    int     `json:"blank_lines"`
	AvgComplexity float64 `json:"avg_complexity"`
}

// CodeSmell represents a detected code smell
type CodeSmell struct {
	File        string `json:"file"`
	Line        int    `json:"line"`
	Type        string `json:"type"`
	Severity    string `json:"severity"` // low, medium, high
	Description string `json:"description"`
}

// CodeQualityAnalyzer analyzes code quality
type CodeQualityAnalyzer struct {
	rootDir string
	files   []string
}

// NewCodeQualityAnalyzer creates a new code quality analyzer
func NewCodeQualityAnalyzer(rootDir string, files []string) *CodeQualityAnalyzer {
	return &CodeQualityAnalyzer{
		rootDir: rootDir,
		files:   files,
	}
}

// Analyze performs comprehensive code quality analysis
func (cqa *CodeQualityAnalyzer) Analyze() (*CodeQualityMetrics, error) {
	metrics := &CodeQualityMetrics{
		FileMetrics:     make(map[string]*FileMetrics),
		LanguageMetrics: make(map[string]*LanguageMetrics),
		CodeSmells:      []CodeSmell{},
	}

	// Analyze each file
	for _, file := range cqa.files {
		fullPath := filepath.Join(cqa.rootDir, file)
		fileMetrics, err := cqa.analyzeFile(fullPath, file)
		if err != nil {
			continue
		}

		metrics.FileMetrics[file] = fileMetrics
		metrics.TotalFiles++
		metrics.TotalLines += fileMetrics.Lines
		metrics.CodeLines += fileMetrics.CodeLines
		metrics.CommentLines += fileMetrics.CommentLines
		metrics.BlankLines += fileMetrics.BlankLines

		// Aggregate by language
		lang := detectFileLanguage(file)
		if langMetrics, ok := metrics.LanguageMetrics[lang]; ok {
			langMetrics.FileCount++
			langMetrics.TotalLines += fileMetrics.Lines
			langMetrics.CodeLines += fileMetrics.CodeLines
			langMetrics.CommentLines += fileMetrics.CommentLines
			langMetrics.BlankLines += fileMetrics.BlankLines
		} else {
			metrics.LanguageMetrics[lang] = &LanguageMetrics{
				Language:     lang,
				FileCount:    1,
				TotalLines:   fileMetrics.Lines,
				CodeLines:    fileMetrics.CodeLines,
				CommentLines: fileMetrics.CommentLines,
				BlankLines:   fileMetrics.BlankLines,
			}
		}

		// Collect code smells
		for _, smell := range fileMetrics.CodeSmells {
			metrics.CodeSmells = append(metrics.CodeSmells, CodeSmell{
				File:        file,
				Type:        smell,
				Severity:    "medium",
				Description: smell,
			})
		}
	}

	// Calculate averages
	if metrics.TotalFiles > 0 {
		totalComplexity := 0
		for _, fm := range metrics.FileMetrics {
			totalComplexity += fm.CyclomaticComplexity
		}
		metrics.AverageComplexity = float64(totalComplexity) / float64(metrics.TotalFiles)
	}

	// Calculate test coverage estimate
	metrics.TestFileCount = cqa.countTestFiles()
	if metrics.TotalFiles > 0 {
		metrics.TestCoverage = float64(metrics.TestFileCount) / float64(metrics.TotalFiles) * 100
	}

	// Calculate maintainability index (simplified)
	metrics.MaintainabilityIndex = cqa.calculateMaintainabilityIndex(metrics)

	return metrics, nil
}

// analyzeFile analyzes a single file
func (cqa *CodeQualityAnalyzer) analyzeFile(fullPath, relPath string) (*FileMetrics, error) {
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	metrics := &FileMetrics{
		FilePath:   relPath,
		CodeSmells: []string{},
	}

	scanner := bufio.NewScanner(file)
	lineNum := 0
	inBlockComment := false
	currentNestingDepth := 0
	lang := detectFileLanguage(relPath)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		// Count lines
		metrics.Lines++

		// Blank lines
		if trimmedLine == "" {
			metrics.BlankLines++
			continue
		}

		// Comment lines (language-specific)
		isComment := false
		switch lang {
		case "Go", "JavaScript", "TypeScript", "Java", "C", "C++", "C#", "Rust", "Kotlin", "Swift", "Scala":
			if strings.HasPrefix(trimmedLine, "//") {
				isComment = true
			}
			if strings.HasPrefix(trimmedLine, "/*") {
				inBlockComment = true
				isComment = true
			}
			if inBlockComment {
				isComment = true
			}
			if strings.HasSuffix(trimmedLine, "*/") {
				inBlockComment = false
			}
		case "Python", "Ruby", "Shell":
			if strings.HasPrefix(trimmedLine, "#") {
				isComment = true
			}
		case "HTML", "XML":
			if strings.HasPrefix(trimmedLine, "<!--") {
				isComment = true
			}
		case "CSS":
			if strings.HasPrefix(trimmedLine, "/*") {
				isComment = true
			}
		}

		if isComment {
			metrics.CommentLines++
		} else {
			metrics.CodeLines++
		}

		// Calculate cyclomatic complexity (simplified)
		complexity := cqa.calculateLineComplexity(line, lang)
		metrics.CyclomaticComplexity += complexity

		// Detect functions
		if cqa.isFunctionDeclaration(line, lang) {
			metrics.Functions++
		}

		// Detect classes
		if cqa.isClassDeclaration(line, lang) {
			metrics.Classes++
		}

		// Track nesting depth
		nestingChange := cqa.calculateNestingChange(line, lang)
		currentNestingDepth += nestingChange
		if currentNestingDepth > metrics.MaxNestingDepth {
			metrics.MaxNestingDepth = currentNestingDepth
		}

		// Detect code smells
		smells := cqa.detectLineCodeSmells(line, lineNum, lang)
		metrics.CodeSmells = append(metrics.CodeSmells, smells...)
	}

	return metrics, scanner.Err()
}

// calculateLineComplexity calculates cyclomatic complexity contribution of a line
func (cqa *CodeQualityAnalyzer) calculateLineComplexity(line, lang string) int {
	complexity := 0

	// Keywords that increase complexity
	complexityKeywords := []string{
		"if", "else", "for", "while", "switch", "case", "catch",
		"&&", "||", "?", "break", "continue", "return",
	}

	for _, keyword := range complexityKeywords {
		// Use word boundaries for keywords
		pattern := `\b` + regexp.QuoteMeta(keyword) + `\b`
		re := regexp.MustCompile(pattern)
		matches := re.FindAllString(line, -1)
		complexity += len(matches)
	}

	return complexity
}

// isFunctionDeclaration checks if a line declares a function
func (cqa *CodeQualityAnalyzer) isFunctionDeclaration(line, lang string) bool {
	trimmed := strings.TrimSpace(line)

	switch lang {
	case "Go":
		return strings.HasPrefix(trimmed, "func ")
	case "JavaScript", "TypeScript":
		return strings.Contains(line, "function ") ||
			regexp.MustCompile(`=>\s*{`).MatchString(line) ||
			regexp.MustCompile(`\w+\s*\([^)]*\)\s*{`).MatchString(line)
	case "Python":
		return strings.HasPrefix(trimmed, "def ")
	case "Java", "C", "C++", "C#":
		return regexp.MustCompile(`\w+\s+\w+\s*\([^)]*\)\s*{`).MatchString(line)
	case "Ruby":
		return strings.HasPrefix(trimmed, "def ")
	case "Rust":
		return strings.HasPrefix(trimmed, "fn ")
	}

	return false
}

// isClassDeclaration checks if a line declares a class
func (cqa *CodeQualityAnalyzer) isClassDeclaration(line, lang string) bool {
	trimmed := strings.TrimSpace(line)

	switch lang {
	case "Go":
		return strings.HasPrefix(trimmed, "type ") && strings.Contains(trimmed, "struct")
	case "JavaScript", "TypeScript":
		return strings.HasPrefix(trimmed, "class ")
	case "Python":
		return strings.HasPrefix(trimmed, "class ")
	case "Java", "C++", "C#":
		return strings.HasPrefix(trimmed, "class ") ||
			strings.HasPrefix(trimmed, "public class ") ||
			strings.HasPrefix(trimmed, "private class ")
	case "Ruby":
		return strings.HasPrefix(trimmed, "class ")
	case "Rust":
		return strings.HasPrefix(trimmed, "struct ") || strings.HasPrefix(trimmed, "impl ")
	}

	return false
}

// calculateNestingChange calculates how nesting depth changes on this line
func (cqa *CodeQualityAnalyzer) calculateNestingChange(line, lang string) int {
	change := 0

	// Count opening braces
	change += strings.Count(line, "{")

	// Count closing braces
	change -= strings.Count(line, "}")

	// For Python, Ruby: indentation-based
	if lang == "Python" || lang == "Ruby" {
		trimmed := strings.TrimSpace(line)
		if strings.HasSuffix(trimmed, ":") {
			change++
		}
		// This is simplified; real implementation would track indentation
	}

	return change
}

// detectLineCodeSmells detects code smells on a line
func (cqa *CodeQualityAnalyzer) detectLineCodeSmells(line string, lineNum int, lang string) []string {
	smells := []string{}

	trimmed := strings.TrimSpace(line)

	// Long line
	if len(line) > 120 {
		smells = append(smells, "Long line (>120 characters)")
	}

	// Multiple statements on one line (for languages with semicolons)
	if lang != "Python" && lang != "Ruby" {
		semicolonCount := strings.Count(line, ";")
		if semicolonCount > 1 {
			smells = append(smells, "Multiple statements on one line")
		}
	}

	// TODOs and FIXMEs
	if strings.Contains(strings.ToUpper(line), "TODO") {
		smells = append(smells, "TODO comment")
	}
	if strings.Contains(strings.ToUpper(line), "FIXME") {
		smells = append(smells, "FIXME comment")
	}
	if strings.Contains(strings.ToUpper(line), "HACK") {
		smells = append(smells, "HACK comment")
	}

	// Console/debug statements (should be removed in production)
	debugPatterns := []string{
		"console.log", "console.error", "console.warn",
		"print(", "println(", "fmt.Println", "System.out.println",
		"debugger", "pdb.set_trace", "binding.pry",
	}
	for _, pattern := range debugPatterns {
		if strings.Contains(line, pattern) {
			smells = append(smells, "Debug/Console statement")
			break
		}
	}

	// Magic numbers (numbers that aren't 0, 1, or assigned to a constant)
	if !strings.Contains(trimmed, "const") && !strings.Contains(trimmed, "final") {
		re := regexp.MustCompile(`[^a-zA-Z0-9_]\d{2,}[^a-zA-Z0-9_]`)
		if re.MatchString(line) {
			smells = append(smells, "Magic number")
		}
	}

	// Deeply nested code (excessive indentation)
	leadingSpaces := len(line) - len(strings.TrimLeft(line, " \t"))
	if leadingSpaces > 32 { // More than 8 levels with 4-space indent
		smells = append(smells, "Deeply nested code")
	}

	return smells
}

// countTestFiles counts the number of test files
func (cqa *CodeQualityAnalyzer) countTestFiles() int {
	count := 0
	for _, file := range cqa.files {
		basename := strings.ToLower(filepath.Base(file))
		if strings.Contains(basename, "test") ||
			strings.Contains(basename, "spec") ||
			strings.HasSuffix(basename, "_test.go") ||
			strings.HasSuffix(basename, ".test.js") ||
			strings.HasSuffix(basename, ".test.ts") ||
			strings.HasSuffix(basename, ".spec.js") ||
			strings.HasSuffix(basename, ".spec.ts") {
			count++
		}
	}
	return count
}

// calculateMaintainabilityIndex calculates a maintainability index (0-100)
func (cqa *CodeQualityAnalyzer) calculateMaintainabilityIndex(metrics *CodeQualityMetrics) float64 {
	// Simplified maintainability index calculation
	// Based on Halstead Volume, Cyclomatic Complexity, and Lines of Code

	if metrics.CodeLines == 0 {
		return 100.0
	}

	// Calculate volume (simplified)
	volume := float64(metrics.CodeLines) * 1.2

	// Calculate complexity factor
	complexityFactor := metrics.AverageComplexity * 2.0

	// Calculate comment ratio (higher is better)
	commentRatio := 0.0
	if metrics.TotalLines > 0 {
		commentRatio = float64(metrics.CommentLines) / float64(metrics.TotalLines)
	}

	// Base index
	index := 171.0 - 5.2*logBase2(volume) - 0.23*complexityFactor - 16.2*logBase2(float64(metrics.CodeLines))

	// Adjust for comments (bonus for good documentation)
	index += commentRatio * 20.0

	// Penalty for code smells
	smellPenalty := float64(len(metrics.CodeSmells)) * 0.5
	index -= smellPenalty

	// Clamp to 0-100
	if index < 0 {
		index = 0
	}
	if index > 100 {
		index = 100
	}

	return index
}

// logBase2 calculates log base 2
func logBase2(x float64) float64 {
	if x <= 0 {
		return 0
	}
	// Simple log2 approximation
	result := 0.0
	for x >= 2 {
		x /= 2
		result++
	}
	return result
}

// GetQualitySummary returns a human-readable summary
func (cqm *CodeQualityMetrics) GetQualitySummary() string {
	var sb strings.Builder

	sb.WriteString("Code Quality Metrics:\n")
	sb.WriteString("=====================\n\n")

	sb.WriteString("Total Files: ")
	sb.WriteString(intToString(cqm.TotalFiles))
	sb.WriteString("\n")

	sb.WriteString("Total Lines: ")
	sb.WriteString(intToString(cqm.TotalLines))
	sb.WriteString("\n")

	sb.WriteString("  - Code Lines: ")
	sb.WriteString(intToString(cqm.CodeLines))
	if cqm.TotalLines > 0 {
		percentage := float64(cqm.CodeLines) / float64(cqm.TotalLines) * 100
		sb.WriteString(" (")
		sb.WriteString(floatToString(percentage, 1))
		sb.WriteString("%)")
	}
	sb.WriteString("\n")

	sb.WriteString("  - Comment Lines: ")
	sb.WriteString(intToString(cqm.CommentLines))
	if cqm.TotalLines > 0 {
		percentage := float64(cqm.CommentLines) / float64(cqm.TotalLines) * 100
		sb.WriteString(" (")
		sb.WriteString(floatToString(percentage, 1))
		sb.WriteString("%)")
	}
	sb.WriteString("\n")

	sb.WriteString("  - Blank Lines: ")
	sb.WriteString(intToString(cqm.BlankLines))
	sb.WriteString("\n\n")

	sb.WriteString("Average Cyclomatic Complexity: ")
	sb.WriteString(floatToString(cqm.AverageComplexity, 2))
	sb.WriteString("\n")

	sb.WriteString("Test Coverage Estimate: ")
	sb.WriteString(floatToString(cqm.TestCoverage, 1))
	sb.WriteString("% (")
	sb.WriteString(intToString(cqm.TestFileCount))
	sb.WriteString(" test files)\n")

	sb.WriteString("Maintainability Index: ")
	sb.WriteString(floatToString(cqm.MaintainabilityIndex, 1))
	sb.WriteString("/100\n")

	if len(cqm.CodeSmells) > 0 {
		sb.WriteString("\nCode Smells Detected: ")
		sb.WriteString(intToString(len(cqm.CodeSmells)))
		sb.WriteString("\n")

		// Count by type
		smellTypes := make(map[string]int)
		for _, smell := range cqm.CodeSmells {
			smellTypes[smell.Type]++
		}

		for smellType, count := range smellTypes {
			sb.WriteString("  - ")
			sb.WriteString(smellType)
			sb.WriteString(": ")
			sb.WriteString(intToString(count))
			sb.WriteString("\n")
		}
	}

	if len(cqm.LanguageMetrics) > 0 {
		sb.WriteString("\nMetrics by Language:\n")
		for lang, metrics := range cqm.LanguageMetrics {
			sb.WriteString("  ")
			sb.WriteString(lang)
			sb.WriteString(": ")
			sb.WriteString(intToString(metrics.FileCount))
			sb.WriteString(" files, ")
			sb.WriteString(intToString(metrics.TotalLines))
			sb.WriteString(" lines\n")
		}
	}

	return sb.String()
}

// floatToString converts float to string with precision
func floatToString(f float64, precision int) string {
	// Simple float to string conversion
	intPart := int(f)
	fracPart := f - float64(intPart)

	result := intToString(intPart)

	if precision > 0 {
		result += "."
		for i := 0; i < precision; i++ {
			fracPart *= 10
			digit := int(fracPart) % 10
			result += string(byte('0' + digit))
		}
	}

	return result
}
