package formatter

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
)

// OutputFormat represents the format of the output file
type OutputFormat string

const (
	FormatTXT    OutputFormat = "txt"
	FormatJSON   OutputFormat = "json"
	FormatXML    OutputFormat = "xml"
	FormatSTDOUT OutputFormat = "stdout" // Format pour sortie directe sur stdout
)

// ProjectReport represents the structure for formatted output
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

// FileInfo represents information about a file
type FileInfo struct {
	Path     string
	Size     int64
	Content  string
	Language string
}

// Formatter handles formatting output in different formats
type Formatter struct {
	format        OutputFormat
	report        *ProjectReport
	outputDir     string
	projectDir    string
	headerContent string
	structure     string
	filePrefix    string // Optional custom prefix for output filename
	projectName   string // Optional custom project name
}

// NewFormatter creates a new formatter for the specified format
func NewFormatter(format string, outputDir, projectDir string) *Formatter {
	f := &Formatter{
		format:     OutputFormat(strings.ToLower(format)),
		report:     &ProjectReport{},
		outputDir:  outputDir,
		projectDir: projectDir,
	}

	// Initialize report with default values
	f.report.ProjectInfo.Name = filepath.Base(filepath.Clean(projectDir))
	f.report.ProjectInfo.GeneratedAt = time.Now()
	f.report.ProjectInfo.Generator = "Prompt My Project (PMP)"

	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	f.report.ProjectInfo.Host = hostname

	// OS info
	f.report.ProjectInfo.OS = fmt.Sprintf("%s/%s", os.Getenv("GOOS"), os.Getenv("GOARCH"))

	return f
}

// SetHeaderContent sets the header content for the report
func (f *Formatter) SetHeaderContent(content string) {
	f.headerContent = content
}

// SetProjectStructure sets the project structure for the report
func (f *Formatter) SetProjectStructure(structure string) {
	f.structure = structure
}

// SetStatistics sets the statistics for the report
func (f *Formatter) SetStatistics(fileCount int, totalSize int64, tokenCount, charCount int, duration time.Duration) {
	f.report.Statistics.FileCount = fileCount
	f.report.Statistics.TotalSize = totalSize
	f.report.Statistics.TotalSizeHuman = humanize.Bytes(uint64(totalSize))
	f.report.Statistics.AvgFileSize = totalSize / int64(max(1, fileCount))
	f.report.Statistics.TokenCount = tokenCount
	f.report.Statistics.CharCount = charCount
	f.report.Statistics.FilesPerSecond = float64(fileCount) / duration.Seconds()
}

// SetFileTypes sets the file types in the report
func (f *Formatter) SetFileTypes(fileTypes map[string]int) {
	f.report.FileTypes = make([]struct {
		Extension string `json:"extension" xml:"extension,attr"`
		Count     int    `json:"count" xml:"count"`
	}, 0)

	for ext, count := range fileTypes {
		f.report.FileTypes = append(f.report.FileTypes, struct {
			Extension string `json:"extension" xml:"extension,attr"`
			Count     int    `json:"count" xml:"count"`
		}{
			Extension: ext,
			Count:     count,
		})
	}
}

// SetTechnologies sets the technologies in the report
func (f *Formatter) SetTechnologies(technologies []string) {
	f.report.Technologies = technologies
}

// SetKeyFiles sets the key files in the report
func (f *Formatter) SetKeyFiles(keyFiles []string) {
	f.report.KeyFiles = keyFiles
}

// SetIssues sets the issues in the report
func (f *Formatter) SetIssues(issues []string) {
	f.report.Issues = issues
}

// SetFilePrefix sets a custom prefix for the output filename
func (f *Formatter) SetFilePrefix(prefix string) {
	f.filePrefix = prefix
}

// SetProjectName sets a custom project name
func (f *Formatter) SetProjectName(name string) {
	f.projectName = name
	if name != "" {
		f.report.ProjectInfo.Name = name
	}
}

// AddFile adds a file to the report
func (f *Formatter) AddFile(fileInfo FileInfo) {
	f.report.Files = append(f.report.Files, struct {
		Path     string `json:"path" xml:"path"`
		Size     int64  `json:"size" xml:"size"`
		Content  string `json:"content,omitempty" xml:"content,omitempty"`
		Language string `json:"language" xml:"language"`
	}{
		Path:     fileInfo.Path,
		Size:     fileInfo.Size,
		Content:  fileInfo.Content,
		Language: fileInfo.Language,
	})
}

// WriteToFile writes the formatted output to a file
func (f *Formatter) WriteToFile() (string, error) {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(f.outputDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Build filename with optional custom prefix
	var filename string
	timestamp := time.Now().Format("20060102_150405")

	if f.filePrefix != "" {
		// Custom prefix: repoName_prompt_timestamp
		filename = fmt.Sprintf("%s_prompt_%s.%s", f.filePrefix, timestamp, f.format)
	} else {
		// Default: prompt_timestamp
		filename = fmt.Sprintf("prompt_%s.%s", timestamp, f.format)
	}
	outputPath := filepath.Join(f.outputDir, filename)

	var content []byte
	var err error

	switch f.format {
	case FormatJSON:
		content, err = json.MarshalIndent(f.report, "", "  ")
	case FormatXML:
		content, err = xml.MarshalIndent(f.report, "", "  ")
		// Add XML header
		content = append([]byte(xml.Header), content...)
	default: // FormatTXT
		var textContent strings.Builder
		textContent.WriteString(f.headerContent)
		textContent.WriteString("\nPROJECT STRUCTURE:\n")
		textContent.WriteString("-----------------------------------------------------\n\n")
		textContent.WriteString(f.structure)
		textContent.WriteString("\nFILE CONTENTS:\n")
		textContent.WriteString("-----------------------------------------------------\n")

		// Add file contents
		for _, file := range f.report.Files {
			textContent.WriteString("\n================================================\n")
			textContent.WriteString(fmt.Sprintf("File: %s\n", file.Path))
			textContent.WriteString("================================================\n")
			textContent.WriteString(file.Content)
			textContent.WriteString("\n")
		}

		content = []byte(textContent.String())
	}

	if err != nil {
		return "", fmt.Errorf("error formatting output: %w", err)
	}

	// Write to file
	if err := os.WriteFile(outputPath, content, 0644); err != nil {
		return "", fmt.Errorf("failed to write output file: %w", err)
	}

	return outputPath, nil
}

// WriteToStdout writes the formatted output to stdout
func (f *Formatter) WriteToStdout() error {
	var content []byte
	var err error

	switch f.format {
	case FormatJSON:
		content, err = json.MarshalIndent(f.report, "", "  ")
	case FormatXML:
		content, err = xml.MarshalIndent(f.report, "", "  ")
		// Add XML header
		content = append([]byte(xml.Header), content...)
	default: // FormatTXT
		var textContent strings.Builder
		textContent.WriteString(f.headerContent)
		textContent.WriteString("\nPROJECT STRUCTURE:\n")
		textContent.WriteString("-----------------------------------------------------\n\n")
		textContent.WriteString(f.structure)
		textContent.WriteString("\nFILE CONTENTS:\n")
		textContent.WriteString("-----------------------------------------------------\n")

		// Add file contents
		for _, file := range f.report.Files {
			textContent.WriteString("\n================================================\n")
			textContent.WriteString(fmt.Sprintf("File: %s\n", file.Path))
			textContent.WriteString("================================================\n")
			textContent.WriteString(file.Content)
			textContent.WriteString("\n")
		}

		content = []byte(textContent.String())
	}

	if err != nil {
		return fmt.Errorf("error formatting output: %w", err)
	}

	// Write to stdout
	_, err = fmt.Print(string(content))
	if err != nil {
		return fmt.Errorf("failed to write to stdout: %w", err)
	}

	return nil
}

// GetFormattedContent returns the formatted content as a string
func (f *Formatter) GetFormattedContent() (string, error) {
	var content []byte
	var err error

	switch f.format {
	case FormatJSON:
		content, err = json.MarshalIndent(f.report, "", "  ")
	case FormatXML:
		content, err = xml.MarshalIndent(f.report, "", "  ")
		// Add XML header
		content = append([]byte(xml.Header), content...)
	default: // FormatTXT
		var textContent strings.Builder
		textContent.WriteString(f.headerContent)
		textContent.WriteString("\nPROJECT STRUCTURE:\n")
		textContent.WriteString("-----------------------------------------------------\n\n")
		textContent.WriteString(f.structure)
		textContent.WriteString("\nFILE CONTENTS:\n")
		textContent.WriteString("-----------------------------------------------------\n")

		// Add file contents
		for _, file := range f.report.Files {
			textContent.WriteString("\n================================================\n")
			textContent.WriteString(fmt.Sprintf("File: %s\n", file.Path))
			textContent.WriteString("================================================\n")
			textContent.WriteString(file.Content)
			textContent.WriteString("\n")
		}

		content = []byte(textContent.String())
	}

	if err != nil {
		return "", fmt.Errorf("error formatting output: %w", err)
	}

	return string(content), nil
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
