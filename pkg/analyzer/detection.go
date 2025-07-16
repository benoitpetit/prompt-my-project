package analyzer

import (
	"path/filepath"
	"strings"
)

// DetectionUtils provides utility functions for detecting technologies, key files, and issues
type DetectionUtils struct{}

// NewDetectionUtils creates a new DetectionUtils instance
func NewDetectionUtils() *DetectionUtils {
	return &DetectionUtils{}
}

// DetectTechnologies identifies technologies used in the project
func (du *DetectionUtils) DetectTechnologies(files []string) []string {
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
		case strings.HasSuffix(file, ".c") || strings.HasSuffix(file, ".cpp") || strings.HasSuffix(file, ".h"):
			technologies["C/C++"] = true
		case strings.HasSuffix(file, ".html"):
			technologies["HTML"] = true
		case strings.HasSuffix(file, ".css"):
			technologies["CSS"] = true
		case strings.HasSuffix(file, ".rs"):
			technologies["Rust"] = true
		case strings.HasSuffix(file, ".kt"):
			technologies["Kotlin"] = true
		case strings.HasSuffix(file, ".swift"):
			technologies["Swift"] = true
		case strings.HasSuffix(file, ".scala"):
			technologies["Scala"] = true
		case strings.HasSuffix(file, ".r") || strings.HasSuffix(file, ".R"):
			technologies["R"] = true
		case strings.HasSuffix(file, ".dart"):
			technologies["Dart"] = true
		case strings.HasSuffix(file, ".lua"):
			technologies["Lua"] = true
		case strings.HasSuffix(file, ".perl") || strings.HasSuffix(file, ".pl"):
			technologies["Perl"] = true
		}

		// Special files indicating technologies
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
		case "Cargo.toml":
			technologies["Rust"] = true
		case "pom.xml":
			technologies["Java"] = true
		case "build.gradle", "build.gradle.kts":
			technologies["Gradle"] = true
		case "Makefile":
			technologies["Make"] = true
		case "CMakeLists.txt":
			technologies["CMake"] = true
		case "Dockerfile":
			technologies["Docker"] = true
		case "docker-compose.yml", "docker-compose.yaml":
			technologies["Docker Compose"] = true
		}
	}

	// Convert to slice
	result := make([]string, 0, len(technologies))
	for tech := range technologies {
		result = append(result, tech)
	}

	return result
}

// IdentifyKeyFiles identifies important files in the project
func (du *DetectionUtils) IdentifyKeyFiles(files []string) []string {
	keyFiles := make([]string, 0)

	for _, file := range files {
		basename := filepath.Base(file)
		switch basename {
		case "main.go", "app.js", "index.js", "package.json", "go.mod", "requirements.txt",
			"setup.py", "Makefile", "Dockerfile", "docker-compose.yml", "README.md",
			"Cargo.toml", "pom.xml", "build.gradle", "CMakeLists.txt", "Gemfile",
			"composer.json", "pubspec.yaml", "Package.swift":
			keyFiles = append(keyFiles, file)
		}
	}

	return keyFiles
}

// IdentifyPotentialIssues identifies potential issues in the project
func (du *DetectionUtils) IdentifyPotentialIssues(files []string) []string {
	issues := make([]string, 0)

	// Check for missing README
	hasReadme := false
	hasLicense := false
	hasGitignore := false
	hasTests := false

	for _, file := range files {
		basename := strings.ToLower(filepath.Base(file))
		
		if strings.Contains(basename, "readme") {
			hasReadme = true
		}
		
		if strings.Contains(basename, "license") || strings.Contains(basename, "licence") {
			hasLicense = true
		}
		
		if basename == ".gitignore" {
			hasGitignore = true
		}
		
		if strings.Contains(basename, "test") || strings.Contains(basename, "spec") {
			hasTests = true
		}
	}

	if !hasReadme {
		issues = append(issues, "Missing README file")
	}
	
	if !hasLicense {
		issues = append(issues, "Missing LICENSE file")
	}
	
	if !hasGitignore {
		issues = append(issues, "Missing .gitignore file")
	}
	
	if !hasTests {
		issues = append(issues, "No test files found")
	}

	return issues
}

// DetectFileLanguage detects the programming language of a file
func (du *DetectionUtils) DetectFileLanguage(filename string) string {
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
	case ".cpp", ".cc", ".cxx":
		return "C++"
	case ".h", ".hpp":
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
	case ".rs":
		return "Rust"
	case ".kt":
		return "Kotlin"
	case ".swift":
		return "Swift"
	case ".scala":
		return "Scala"
	case ".r":
		return "R"
	case ".dart":
		return "Dart"
	case ".lua":
		return "Lua"
	case ".perl", ".pl":
		return "Perl"
	case ".yaml", ".yml":
		return "YAML"
	case ".toml":
		return "TOML"
	case ".ini":
		return "INI"
	case ".conf":
		return "Configuration"
	default:
		return "Plain Text"
	}
}

// CollectFileExtensions collects file extensions and their counts
func (du *DetectionUtils) CollectFileExtensions(files []string) map[string]int {
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
