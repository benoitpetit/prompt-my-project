package analyzer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// DependencyType represents the type of dependency
type DependencyType string

const (
	ProductionDep  DependencyType = "production"
	DevelopmentDep DependencyType = "development"
	OptionalDep    DependencyType = "optional"
	PeerDep        DependencyType = "peer"
)

// Dependency represents a single dependency
type Dependency struct {
	Name    string         `json:"name"`
	Version string         `json:"version"`
	Type    DependencyType `json:"type"`
	Direct  bool           `json:"direct"` // true if directly required, false if transitive
}

// DependencyAnalysis represents the complete dependency analysis
type DependencyAnalysis struct {
	Dependencies     []Dependency         `json:"dependencies"`
	TotalCount       int                  `json:"total_count"`
	ProductionCount  int                  `json:"production_count"`
	DevelopmentCount int                  `json:"development_count"`
	OptionalCount    int                  `json:"optional_count"`
	DirectCount      int                  `json:"direct_count"`
	TransitiveCount  int                  `json:"transitive_count"`
	ByLanguage       map[string]int       `json:"by_language"`
	Vulnerabilities  []Vulnerability      `json:"vulnerabilities,omitempty"`
	OutdatedPackages []OutdatedDependency `json:"outdated_packages,omitempty"`
}

// DependencyAnalyzer analyzes project dependencies
type DependencyAnalyzer struct {
	rootDir string
}

// NewDependencyAnalyzer creates a new dependency analyzer
func NewDependencyAnalyzer(rootDir string) *DependencyAnalyzer {
	return &DependencyAnalyzer{
		rootDir: rootDir,
	}
}

// AnalyzeAll performs comprehensive dependency analysis
func (da *DependencyAnalyzer) AnalyzeAll() (*DependencyAnalysis, error) {
	analysis := &DependencyAnalysis{
		Dependencies: []Dependency{},
		ByLanguage:   make(map[string]int),
	}

	// Analyze different package managers
	da.analyzeNodeDependencies(analysis)
	da.analyzeGoDependencies(analysis)
	da.analyzePythonDependencies(analysis)
	da.analyzeRustDependencies(analysis)
	da.analyzeRubyDependencies(analysis)
	da.analyzePHPDependencies(analysis)

	// Calculate totals
	analysis.TotalCount = len(analysis.Dependencies)
	for _, dep := range analysis.Dependencies {
		switch dep.Type {
		case ProductionDep:
			analysis.ProductionCount++
		case DevelopmentDep:
			analysis.DevelopmentCount++
		case OptionalDep:
			analysis.OptionalCount++
		}

		if dep.Direct {
			analysis.DirectCount++
		} else {
			analysis.TransitiveCount++
		}
	}

	// Check for vulnerabilities and outdated packages
	vulnChecker := NewVulnerabilityChecker(da.rootDir)
	analysis.Vulnerabilities, analysis.OutdatedPackages = vulnChecker.CheckDependencies(analysis)

	return analysis, nil
}

// analyzeNodeDependencies analyzes Node.js/npm dependencies
func (da *DependencyAnalyzer) analyzeNodeDependencies(analysis *DependencyAnalysis) {
	pkgPath := filepath.Join(da.rootDir, "package.json")
	content, err := os.ReadFile(pkgPath)
	if err != nil {
		return
	}

	var pkg map[string]interface{}
	if err := json.Unmarshal(content, &pkg); err != nil {
		return
	}

	// Production dependencies
	if deps, ok := pkg["dependencies"].(map[string]interface{}); ok {
		for name, version := range deps {
			if versionStr, ok := version.(string); ok {
				analysis.Dependencies = append(analysis.Dependencies, Dependency{
					Name:    name,
					Version: versionStr,
					Type:    ProductionDep,
					Direct:  true,
				})
				analysis.ByLanguage["Node.js"]++
			}
		}
	}

	// Development dependencies
	if deps, ok := pkg["devDependencies"].(map[string]interface{}); ok {
		for name, version := range deps {
			if versionStr, ok := version.(string); ok {
				analysis.Dependencies = append(analysis.Dependencies, Dependency{
					Name:    name,
					Version: versionStr,
					Type:    DevelopmentDep,
					Direct:  true,
				})
				analysis.ByLanguage["Node.js"]++
			}
		}
	}

	// Optional dependencies
	if deps, ok := pkg["optionalDependencies"].(map[string]interface{}); ok {
		for name, version := range deps {
			if versionStr, ok := version.(string); ok {
				analysis.Dependencies = append(analysis.Dependencies, Dependency{
					Name:    name,
					Version: versionStr,
					Type:    OptionalDep,
					Direct:  true,
				})
				analysis.ByLanguage["Node.js"]++
			}
		}
	}

	// Peer dependencies
	if deps, ok := pkg["peerDependencies"].(map[string]interface{}); ok {
		for name, version := range deps {
			if versionStr, ok := version.(string); ok {
				analysis.Dependencies = append(analysis.Dependencies, Dependency{
					Name:    name,
					Version: versionStr,
					Type:    PeerDep,
					Direct:  true,
				})
				analysis.ByLanguage["Node.js"]++
			}
		}
	}

	// Try to read package-lock.json for transitive dependencies
	lockPath := filepath.Join(da.rootDir, "package-lock.json")
	if lockContent, err := os.ReadFile(lockPath); err == nil {
		var lockData map[string]interface{}
		if json.Unmarshal(lockContent, &lockData) == nil {
			// Mark this as having transitive dependency information
			// In a real implementation, we would parse the full lock file
			// For now, we just note that transitive deps exist
		}
	}
}

// analyzeGoDependencies analyzes Go module dependencies
func (da *DependencyAnalyzer) analyzeGoDependencies(analysis *DependencyAnalysis) {
	goModPath := filepath.Join(da.rootDir, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return
	}

	lines := strings.Split(string(content), "\n")
	inRequireBlock := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Check for require block
		if strings.HasPrefix(line, "require (") {
			inRequireBlock = true
			continue
		}
		if inRequireBlock && line == ")" {
			inRequireBlock = false
			continue
		}

		// Parse require line
		if strings.HasPrefix(line, "require ") || inRequireBlock {
			// Remove "require " prefix if present
			line = strings.TrimPrefix(line, "require ")
			line = strings.TrimSpace(line)

			// Skip comments and empty lines
			if line == "" || strings.HasPrefix(line, "//") {
				continue
			}

			// Parse module and version
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				module := parts[0]
				version := parts[1]

				// Determine if it's an indirect dependency
				isDirect := !strings.Contains(line, "// indirect")

				depType := ProductionDep
				if !isDirect {
					depType = DevelopmentDep
				}

				analysis.Dependencies = append(analysis.Dependencies, Dependency{
					Name:    module,
					Version: version,
					Type:    depType,
					Direct:  isDirect,
				})
				analysis.ByLanguage["Go"]++
			}
		}
	}
}

// analyzePythonDependencies analyzes Python dependencies
func (da *DependencyAnalyzer) analyzePythonDependencies(analysis *DependencyAnalysis) {
	// Try requirements.txt
	reqPath := filepath.Join(da.rootDir, "requirements.txt")
	if content, err := os.ReadFile(reqPath); err == nil {
		da.parsePythonRequirements(string(content), analysis, ProductionDep)
	}

	// Try requirements-dev.txt
	devReqPath := filepath.Join(da.rootDir, "requirements-dev.txt")
	if content, err := os.ReadFile(devReqPath); err == nil {
		da.parsePythonRequirements(string(content), analysis, DevelopmentDep)
	}

	// Try pyproject.toml
	pyprojectPath := filepath.Join(da.rootDir, "pyproject.toml")
	if content, err := os.ReadFile(pyprojectPath); err == nil {
		da.parsePyprojectToml(string(content), analysis)
	}

	// Try Pipfile
	pipfilePath := filepath.Join(da.rootDir, "Pipfile")
	if content, err := os.ReadFile(pipfilePath); err == nil {
		da.parsePipfile(string(content), analysis)
	}
}

// parsePythonRequirements parses a requirements.txt file
func (da *DependencyAnalyzer) parsePythonRequirements(content string, analysis *DependencyAnalysis, depType DependencyType) {
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse package name and version
		re := regexp.MustCompile(`^([a-zA-Z0-9\-_.]+)([><=!~]+)?(.*)$`)
		matches := re.FindStringSubmatch(line)

		if len(matches) > 1 {
			name := matches[1]
			version := ""
			if len(matches) > 3 {
				version = matches[2] + matches[3]
			}

			analysis.Dependencies = append(analysis.Dependencies, Dependency{
				Name:    name,
				Version: version,
				Type:    depType,
				Direct:  true,
			})
			analysis.ByLanguage["Python"]++
		}
	}
}

// parsePyprojectToml parses pyproject.toml
func (da *DependencyAnalyzer) parsePyprojectToml(content string, analysis *DependencyAnalysis) {
	lines := strings.Split(content, "\n")
	inDeps := false
	inDevDeps := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "[tool.poetry.dependencies]") {
			inDeps = true
			inDevDeps = false
			continue
		}
		if strings.HasPrefix(line, "[tool.poetry.dev-dependencies]") ||
			strings.HasPrefix(line, "[tool.poetry.group.dev.dependencies]") {
			inDeps = false
			inDevDeps = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			inDeps = false
			inDevDeps = false
			continue
		}

		if (inDeps || inDevDeps) && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				version := strings.Trim(strings.TrimSpace(parts[1]), "\"'")

				depType := ProductionDep
				if inDevDeps {
					depType = DevelopmentDep
				}

				analysis.Dependencies = append(analysis.Dependencies, Dependency{
					Name:    name,
					Version: version,
					Type:    depType,
					Direct:  true,
				})
				analysis.ByLanguage["Python"]++
			}
		}
	}
}

// parsePipfile parses Pipfile
func (da *DependencyAnalyzer) parsePipfile(content string, analysis *DependencyAnalysis) {
	lines := strings.Split(content, "\n")
	inPackages := false
	inDevPackages := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "[packages]" {
			inPackages = true
			inDevPackages = false
			continue
		}
		if line == "[dev-packages]" {
			inPackages = false
			inDevPackages = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			inPackages = false
			inDevPackages = false
			continue
		}

		if (inPackages || inDevPackages) && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				version := strings.Trim(strings.TrimSpace(parts[1]), "\"'")

				depType := ProductionDep
				if inDevPackages {
					depType = DevelopmentDep
				}

				analysis.Dependencies = append(analysis.Dependencies, Dependency{
					Name:    name,
					Version: version,
					Type:    depType,
					Direct:  true,
				})
				analysis.ByLanguage["Python"]++
			}
		}
	}
}

// analyzeRustDependencies analyzes Rust/Cargo dependencies
func (da *DependencyAnalyzer) analyzeRustDependencies(analysis *DependencyAnalysis) {
	cargoPath := filepath.Join(da.rootDir, "Cargo.toml")
	content, err := os.ReadFile(cargoPath)
	if err != nil {
		return
	}

	lines := strings.Split(string(content), "\n")
	inDeps := false
	inDevDeps := false
	inBuildDeps := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line == "[dependencies]" {
			inDeps = true
			inDevDeps = false
			inBuildDeps = false
			continue
		}
		if line == "[dev-dependencies]" {
			inDeps = false
			inDevDeps = true
			inBuildDeps = false
			continue
		}
		if line == "[build-dependencies]" {
			inDeps = false
			inDevDeps = false
			inBuildDeps = true
			continue
		}
		if strings.HasPrefix(line, "[") {
			inDeps = false
			inDevDeps = false
			inBuildDeps = false
			continue
		}

		if (inDeps || inDevDeps || inBuildDeps) && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				version := strings.Trim(strings.TrimSpace(parts[1]), "\"'")

				depType := ProductionDep
				if inDevDeps || inBuildDeps {
					depType = DevelopmentDep
				}

				analysis.Dependencies = append(analysis.Dependencies, Dependency{
					Name:    name,
					Version: version,
					Type:    depType,
					Direct:  true,
				})
				analysis.ByLanguage["Rust"]++
			}
		}
	}
}

// analyzeRubyDependencies analyzes Ruby/Bundler dependencies
func (da *DependencyAnalyzer) analyzeRubyDependencies(analysis *DependencyAnalysis) {
	gemfilePath := filepath.Join(da.rootDir, "Gemfile")
	content, err := os.ReadFile(gemfilePath)
	if err != nil {
		return
	}

	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "gem ") {
			// Parse gem declaration
			re := regexp.MustCompile(`gem\s+['"]([^'"]+)['"](?:\s*,\s*['"]([^'"]+)['"])?`)
			matches := re.FindStringSubmatch(line)

			if len(matches) > 1 {
				name := matches[1]
				version := ""
				if len(matches) > 2 {
					version = matches[2]
				}

				depType := ProductionDep
				if strings.Contains(line, "group:") && (strings.Contains(line, "development") || strings.Contains(line, "test")) {
					depType = DevelopmentDep
				}

				analysis.Dependencies = append(analysis.Dependencies, Dependency{
					Name:    name,
					Version: version,
					Type:    depType,
					Direct:  true,
				})
				analysis.ByLanguage["Ruby"]++
			}
		}
	}
}

// analyzePHPDependencies analyzes PHP/Composer dependencies
func (da *DependencyAnalyzer) analyzePHPDependencies(analysis *DependencyAnalysis) {
	composerPath := filepath.Join(da.rootDir, "composer.json")
	content, err := os.ReadFile(composerPath)
	if err != nil {
		return
	}

	var composer map[string]interface{}
	if err := json.Unmarshal(content, &composer); err != nil {
		return
	}

	// Production dependencies
	if deps, ok := composer["require"].(map[string]interface{}); ok {
		for name, version := range deps {
			// Skip PHP version constraint
			if name == "php" {
				continue
			}

			if versionStr, ok := version.(string); ok {
				analysis.Dependencies = append(analysis.Dependencies, Dependency{
					Name:    name,
					Version: versionStr,
					Type:    ProductionDep,
					Direct:  true,
				})
				analysis.ByLanguage["PHP"]++
			}
		}
	}

	// Development dependencies
	if deps, ok := composer["require-dev"].(map[string]interface{}); ok {
		for name, version := range deps {
			if versionStr, ok := version.(string); ok {
				analysis.Dependencies = append(analysis.Dependencies, Dependency{
					Name:    name,
					Version: versionStr,
					Type:    DevelopmentDep,
					Direct:  true,
				})
				analysis.ByLanguage["PHP"]++
			}
		}
	}
}

// GetDependencySummary returns a human-readable summary
func (da *DependencyAnalysis) GetDependencySummary() string {
	var sb strings.Builder

	sb.WriteString("Dependency Analysis:\n")
	sb.WriteString("===================\n\n")

	sb.WriteString("Total Dependencies: ")
	sb.WriteString(intToString(da.TotalCount))
	sb.WriteString("\n")

	sb.WriteString("  - Production: ")
	sb.WriteString(intToString(da.ProductionCount))
	sb.WriteString("\n")

	sb.WriteString("  - Development: ")
	sb.WriteString(intToString(da.DevelopmentCount))
	sb.WriteString("\n")

	if da.OptionalCount > 0 {
		sb.WriteString("  - Optional: ")
		sb.WriteString(intToString(da.OptionalCount))
		sb.WriteString("\n")
	}

	sb.WriteString("\n")
	sb.WriteString("Direct Dependencies: ")
	sb.WriteString(intToString(da.DirectCount))
	sb.WriteString("\n")

	if da.TransitiveCount > 0 {
		sb.WriteString("Transitive Dependencies: ")
		sb.WriteString(intToString(da.TransitiveCount))
		sb.WriteString("\n")
	}

	if len(da.ByLanguage) > 0 {
		sb.WriteString("\nBy Ecosystem:\n")
		for lang, count := range da.ByLanguage {
			sb.WriteString("  - ")
			sb.WriteString(lang)
			sb.WriteString(": ")
			sb.WriteString(intToString(count))
			sb.WriteString("\n")
		}
	}

	// Add security summary
	if len(da.Vulnerabilities) > 0 || len(da.OutdatedPackages) > 0 {
		sb.WriteString("\n")
		sb.WriteString(GetVulnerabilitySummary(da.Vulnerabilities, da.OutdatedPackages))
	}

	return sb.String()
}

// Helper function to convert int to string without fmt
func intToString(n int) string {
	if n == 0 {
		return "0"
	}

	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}

	return string(digits)
}
