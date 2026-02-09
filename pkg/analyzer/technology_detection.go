package analyzer

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/benoitpetit/prompt-my-project/pkg/utils"
)

// TechnologyInfo represents detected technology with confidence score
type TechnologyInfo struct {
	Name       string  `json:"name"`
	Version    string  `json:"version,omitempty"`
	Confidence float64 `json:"confidence"` // 0.0 to 1.0
	Category   string  `json:"category"`   // language, framework, tool, database, etc.
}

// TechnologyDetector provides advanced technology detection
type TechnologyDetector struct {
	rootDir string
	files   []string
}

// NewTechnologyDetector creates a new technology detector
func NewTechnologyDetector(rootDir string, files []string) *TechnologyDetector {
	return &TechnologyDetector{
		rootDir: rootDir,
		files:   files,
	}
}

// DetectAll performs comprehensive technology detection
func (td *TechnologyDetector) DetectAll() []TechnologyInfo {
	techs := make(map[string]TechnologyInfo)

	// Detect languages
	td.detectLanguages(techs)

	// Detect frameworks
	td.detectFrontendFrameworks(techs)
	td.detectBackendFrameworks(techs)

	// Detect build tools
	td.detectBuildTools(techs)

	// Detect databases
	td.detectDatabases(techs)

	// Detect testing frameworks
	td.detectTestingFrameworks(techs)

	// Detect containerization
	td.detectContainerization(techs)

	// Detect CI/CD
	td.detectCICD(techs)

	// Convert map to slice
	result := make([]TechnologyInfo, 0, len(techs))
	for _, tech := range techs {
		result = append(result, tech)
	}

	// Sort by confidence (highest first) using optimized sort
	if len(result) > 1 {
		techSort := make([]utils.Float64Sortable, len(result))
		for i, tech := range result {
			techSort[i] = utils.Float64Sortable{
				Key:   tech.Confidence,
				Value: tech,
			}
		}
		utils.SortByFloat64(techSort, false) // false = descending

		// Convert back
		for i, sorted := range techSort {
			result[i] = sorted.Value.(TechnologyInfo)
		}
	}

	return result
}

// detectLanguages detects programming languages
func (td *TechnologyDetector) detectLanguages(techs map[string]TechnologyInfo) {
	languageCount := make(map[string]int)
	totalFiles := 0

	for _, file := range td.files {
		ext := strings.ToLower(filepath.Ext(file))
		totalFiles++

		switch ext {
		case ".go":
			languageCount["Go"]++
		case ".js", ".jsx":
			languageCount["JavaScript"]++
		case ".ts", ".tsx":
			languageCount["TypeScript"]++
		case ".py":
			languageCount["Python"]++
		case ".java":
			languageCount["Java"]++
		case ".rb":
			languageCount["Ruby"]++
		case ".php":
			languageCount["PHP"]++
		case ".cs":
			languageCount["C#"]++
		case ".c", ".cpp", ".cc", ".cxx", ".h", ".hpp":
			languageCount["C/C++"]++
		case ".rs":
			languageCount["Rust"]++
		case ".kt", ".kts":
			languageCount["Kotlin"]++
		case ".swift":
			languageCount["Swift"]++
		case ".scala":
			languageCount["Scala"]++
		case ".r":
			languageCount["R"]++
		case ".dart":
			languageCount["Dart"]++
		}
	}

	// Add languages with confidence based on file count
	for lang, count := range languageCount {
		confidence := float64(count) / float64(totalFiles)
		if confidence > 0.01 { // Only include if >1% of files
			version := td.detectLanguageVersion(lang)
			techs[lang] = TechnologyInfo{
				Name:       lang,
				Version:    version,
				Confidence: confidence,
				Category:   "language",
			}
		}
	}
}

// detectLanguageVersion attempts to detect language version
func (td *TechnologyDetector) detectLanguageVersion(language string) string {
	switch language {
	case "Go":
		return td.detectGoVersion()
	case "Node.js", "JavaScript", "TypeScript":
		return td.detectNodeVersion()
	case "Python":
		return td.detectPythonVersion()
	case "Rust":
		return td.detectRustVersion()
	}
	return ""
}

// detectGoVersion reads go.mod for Go version
func (td *TechnologyDetector) detectGoVersion() string {
	goModPath := filepath.Join(td.rootDir, "go.mod")
	content, err := os.ReadFile(goModPath)
	if err != nil {
		return ""
	}

	re := regexp.MustCompile(`go\s+(\d+\.\d+)`)
	if match := re.FindSubmatch(content); match != nil {
		return string(match[1])
	}
	return ""
}

// detectNodeVersion reads package.json for Node version
func (td *TechnologyDetector) detectNodeVersion() string {
	pkgPath := filepath.Join(td.rootDir, "package.json")
	content, err := os.ReadFile(pkgPath)
	if err != nil {
		return ""
	}

	var pkg map[string]interface{}
	if err := json.Unmarshal(content, &pkg); err != nil {
		return ""
	}

	if engines, ok := pkg["engines"].(map[string]interface{}); ok {
		if node, ok := engines["node"].(string); ok {
			return node
		}
	}
	return ""
}

// detectPythonVersion reads setup.py or pyproject.toml
func (td *TechnologyDetector) detectPythonVersion() string {
	// Try pyproject.toml
	pyprojectPath := filepath.Join(td.rootDir, "pyproject.toml")
	if content, err := os.ReadFile(pyprojectPath); err == nil {
		re := regexp.MustCompile(`python\s*=\s*"([^"]+)"`)
		if match := re.FindSubmatch(content); match != nil {
			return string(match[1])
		}
	}

	// Try setup.py
	setupPath := filepath.Join(td.rootDir, "setup.py")
	if content, err := os.ReadFile(setupPath); err == nil {
		re := regexp.MustCompile(`python_requires\s*=\s*['"]([^'"]+)['"]`)
		if match := re.FindSubmatch(content); match != nil {
			return string(match[1])
		}
	}

	return ""
}

// detectRustVersion reads Cargo.toml
func (td *TechnologyDetector) detectRustVersion() string {
	cargoPath := filepath.Join(td.rootDir, "Cargo.toml")
	content, err := os.ReadFile(cargoPath)
	if err != nil {
		return ""
	}

	re := regexp.MustCompile(`rust-version\s*=\s*"([^"]+)"`)
	if match := re.FindSubmatch(content); match != nil {
		return string(match[1])
	}
	return ""
}

// detectFrontendFrameworks detects frontend frameworks
func (td *TechnologyDetector) detectFrontendFrameworks(techs map[string]TechnologyInfo) {
	pkgPath := filepath.Join(td.rootDir, "package.json")
	content, err := os.ReadFile(pkgPath)
	if err != nil {
		return
	}

	var pkg map[string]interface{}
	if err := json.Unmarshal(content, &pkg); err != nil {
		return
	}

	// Check dependencies
	deps := make(map[string]string)
	if d, ok := pkg["dependencies"].(map[string]interface{}); ok {
		for k, v := range d {
			if version, ok := v.(string); ok {
				deps[k] = version
			}
		}
	}
	if d, ok := pkg["devDependencies"].(map[string]interface{}); ok {
		for k, v := range d {
			if version, ok := v.(string); ok {
				deps[k] = version
			}
		}
	}

	// Detect frameworks
	if version, ok := deps["react"]; ok {
		techs["React"] = TechnologyInfo{
			Name:       "React",
			Version:    strings.TrimPrefix(version, "^"),
			Confidence: 1.0,
			Category:   "framework",
		}
	}
	if version, ok := deps["vue"]; ok {
		techs["Vue.js"] = TechnologyInfo{
			Name:       "Vue.js",
			Version:    strings.TrimPrefix(version, "^"),
			Confidence: 1.0,
			Category:   "framework",
		}
	}
	if _, ok := deps["@angular/core"]; ok {
		techs["Angular"] = TechnologyInfo{
			Name:       "Angular",
			Version:    strings.TrimPrefix(deps["@angular/core"], "^"),
			Confidence: 1.0,
			Category:   "framework",
		}
	}
	if version, ok := deps["svelte"]; ok {
		techs["Svelte"] = TechnologyInfo{
			Name:       "Svelte",
			Version:    strings.TrimPrefix(version, "^"),
			Confidence: 1.0,
			Category:   "framework",
		}
	}
	if version, ok := deps["next"]; ok {
		techs["Next.js"] = TechnologyInfo{
			Name:       "Next.js",
			Version:    strings.TrimPrefix(version, "^"),
			Confidence: 1.0,
			Category:   "framework",
		}
	}
	if version, ok := deps["nuxt"]; ok {
		techs["Nuxt.js"] = TechnologyInfo{
			Name:       "Nuxt.js",
			Version:    strings.TrimPrefix(version, "^"),
			Confidence: 1.0,
			Category:   "framework",
		}
	}
}

// detectBackendFrameworks detects backend frameworks
func (td *TechnologyDetector) detectBackendFrameworks(techs map[string]TechnologyInfo) {
	// Node.js frameworks
	pkgPath := filepath.Join(td.rootDir, "package.json")
	if content, err := os.ReadFile(pkgPath); err == nil {
		var pkg map[string]interface{}
		if json.Unmarshal(content, &pkg) == nil {
			deps := make(map[string]string)
			if d, ok := pkg["dependencies"].(map[string]interface{}); ok {
				for k, v := range d {
					if version, ok := v.(string); ok {
						deps[k] = version
					}
				}
			}

			if version, ok := deps["express"]; ok {
				techs["Express"] = TechnologyInfo{
					Name:       "Express",
					Version:    strings.TrimPrefix(version, "^"),
					Confidence: 0.9,
					Category:   "framework",
				}
			}
			if version, ok := deps["fastify"]; ok {
				techs["Fastify"] = TechnologyInfo{
					Name:       "Fastify",
					Version:    strings.TrimPrefix(version, "^"),
					Confidence: 0.9,
					Category:   "framework",
				}
			}
			if version, ok := deps["koa"]; ok {
				techs["Koa"] = TechnologyInfo{
					Name:       "Koa",
					Version:    strings.TrimPrefix(version, "^"),
					Confidence: 0.9,
					Category:   "framework",
				}
			}
		}
	}

	// Python frameworks
	reqPath := filepath.Join(td.rootDir, "requirements.txt")
	if content, err := os.ReadFile(reqPath); err == nil {
		reqs := strings.ToLower(string(content))
		if strings.Contains(reqs, "django") {
			techs["Django"] = TechnologyInfo{
				Name:       "Django",
				Confidence: 0.9,
				Category:   "framework",
			}
		}
		if strings.Contains(reqs, "flask") {
			techs["Flask"] = TechnologyInfo{
				Name:       "Flask",
				Confidence: 0.9,
				Category:   "framework",
			}
		}
		if strings.Contains(reqs, "fastapi") {
			techs["FastAPI"] = TechnologyInfo{
				Name:       "FastAPI",
				Confidence: 0.9,
				Category:   "framework",
			}
		}
	}

	// Go frameworks (check imports in go files)
	for _, file := range td.files {
		if strings.HasSuffix(file, ".go") {
			fullPath := filepath.Join(td.rootDir, file)
			if content, err := os.ReadFile(fullPath); err == nil {
				contentStr := string(content)
				if strings.Contains(contentStr, "github.com/gin-gonic/gin") {
					techs["Gin"] = TechnologyInfo{
						Name:       "Gin",
						Confidence: 0.9,
						Category:   "framework",
					}
				}
				if strings.Contains(contentStr, "github.com/gofiber/fiber") {
					techs["Fiber"] = TechnologyInfo{
						Name:       "Fiber",
						Confidence: 0.9,
						Category:   "framework",
					}
				}
				if strings.Contains(contentStr, "github.com/labstack/echo") {
					techs["Echo"] = TechnologyInfo{
						Name:       "Echo",
						Confidence: 0.9,
						Category:   "framework",
					}
				}
			}
		}
	}
}

// detectBuildTools detects build tools
func (td *TechnologyDetector) detectBuildTools(techs map[string]TechnologyInfo) {
	// Check for config files
	configFiles := map[string]string{
		"webpack.config.js": "Webpack",
		"webpack.config.ts": "Webpack",
		"vite.config.js":    "Vite",
		"vite.config.ts":    "Vite",
		"rollup.config.js":  "Rollup",
		"rollup.config.ts":  "Rollup",
		"esbuild.config.js": "esbuild",
		"tsconfig.json":     "TypeScript Compiler",
		"babel.config.js":   "Babel",
		".babelrc":          "Babel",
		"Makefile":          "Make",
		"CMakeLists.txt":    "CMake",
		"build.gradle":      "Gradle",
		"build.gradle.kts":  "Gradle",
		"pom.xml":           "Maven",
		"Cargo.toml":        "Cargo",
		"mix.exs":           "Mix",
		"Rakefile":          "Rake",
		"gulpfile.js":       "Gulp",
		"Gruntfile.js":      "Grunt",
		"turbo.json":        "Turborepo",
		"nx.json":           "Nx",
	}

	for _, file := range td.files {
		basename := filepath.Base(file)
		if tool, ok := configFiles[basename]; ok {
			techs[tool] = TechnologyInfo{
				Name:       tool,
				Confidence: 1.0,
				Category:   "build-tool",
			}
		}
	}

	// Check package.json scripts
	pkgPath := filepath.Join(td.rootDir, "package.json")
	if content, err := os.ReadFile(pkgPath); err == nil {
		contentStr := string(content)
		if strings.Contains(contentStr, "parcel") {
			techs["Parcel"] = TechnologyInfo{
				Name:       "Parcel",
				Confidence: 0.8,
				Category:   "build-tool",
			}
		}
	}
}

// detectDatabases detects database usage
func (td *TechnologyDetector) detectDatabases(techs map[string]TechnologyInfo) {
	// Check for database config files
	dbFiles := map[string]string{
		"prisma/schema.prisma": "Prisma",
		"drizzle.config.ts":    "Drizzle ORM",
		"typeorm.config.js":    "TypeORM",
		"sequelize.config.js":  "Sequelize",
	}

	for _, file := range td.files {
		for pattern, db := range dbFiles {
			if strings.Contains(file, pattern) {
				techs[db] = TechnologyInfo{
					Name:       db,
					Confidence: 1.0,
					Category:   "database",
				}
			}
		}
	}

	// Check package.json dependencies
	pkgPath := filepath.Join(td.rootDir, "package.json")
	if content, err := os.ReadFile(pkgPath); err == nil {
		var pkg map[string]interface{}
		if json.Unmarshal(content, &pkg) == nil {
			allDeps := make(map[string]bool)

			if deps, ok := pkg["dependencies"].(map[string]interface{}); ok {
				for k := range deps {
					allDeps[k] = true
				}
			}
			if deps, ok := pkg["devDependencies"].(map[string]interface{}); ok {
				for k := range deps {
					allDeps[k] = true
				}
			}

			dbDeps := map[string]string{
				"pg":             "PostgreSQL",
				"mysql":          "MySQL",
				"mysql2":         "MySQL",
				"mongodb":        "MongoDB",
				"mongoose":       "MongoDB",
				"redis":          "Redis",
				"sqlite3":        "SQLite",
				"better-sqlite3": "SQLite",
			}

			for dep, db := range dbDeps {
				if allDeps[dep] {
					techs[db] = TechnologyInfo{
						Name:       db,
						Confidence: 0.9,
						Category:   "database",
					}
				}
			}
		}
	}

	// Check Python requirements
	reqPath := filepath.Join(td.rootDir, "requirements.txt")
	if content, err := os.ReadFile(reqPath); err == nil {
		reqs := strings.ToLower(string(content))
		pyDbLibs := map[string]string{
			"psycopg2":        "PostgreSQL",
			"pymongo":         "MongoDB",
			"redis":           "Redis",
			"mysql-connector": "MySQL",
			"sqlalchemy":      "SQLAlchemy",
		}

		for lib, db := range pyDbLibs {
			if strings.Contains(reqs, lib) {
				techs[db] = TechnologyInfo{
					Name:       db,
					Confidence: 0.9,
					Category:   "database",
				}
			}
		}
	}

	// Check Go dependencies
	goModPath := filepath.Join(td.rootDir, "go.mod")
	if content, err := os.ReadFile(goModPath); err == nil {
		mods := strings.ToLower(string(content))
		goDbLibs := map[string]string{
			"lib/pq":              "PostgreSQL",
			"go-sql-driver/mysql": "MySQL",
			"go-redis/redis":      "Redis",
			"mongo-driver":        "MongoDB",
			"gorm":                "GORM",
		}

		for lib, db := range goDbLibs {
			if strings.Contains(mods, lib) {
				techs[db] = TechnologyInfo{
					Name:       db,
					Confidence: 0.9,
					Category:   "database",
				}
			}
		}
	}
}

// detectTestingFrameworks detects testing frameworks
func (td *TechnologyDetector) detectTestingFrameworks(techs map[string]TechnologyInfo) {
	// Check package.json
	pkgPath := filepath.Join(td.rootDir, "package.json")
	if content, err := os.ReadFile(pkgPath); err == nil {
		var pkg map[string]interface{}
		if json.Unmarshal(content, &pkg) == nil {
			allDeps := make(map[string]bool)

			if deps, ok := pkg["devDependencies"].(map[string]interface{}); ok {
				for k := range deps {
					allDeps[k] = true
				}
			}

			testFrameworks := map[string]string{
				"jest":                   "Jest",
				"mocha":                  "Mocha",
				"jasmine":                "Jasmine",
				"vitest":                 "Vitest",
				"cypress":                "Cypress",
				"playwright":             "Playwright",
				"@testing-library/react": "React Testing Library",
				"chai":                   "Chai",
			}

			for dep, framework := range testFrameworks {
				if allDeps[dep] {
					techs[framework] = TechnologyInfo{
						Name:       framework,
						Confidence: 1.0,
						Category:   "testing",
					}
				}
			}
		}
	}

	// Check Python requirements
	reqPath := filepath.Join(td.rootDir, "requirements.txt")
	if content, err := os.ReadFile(reqPath); err == nil {
		reqs := strings.ToLower(string(content))
		pyTestFrameworks := map[string]string{
			"pytest":   "pytest",
			"unittest": "unittest",
			"nose":     "nose",
		}

		for lib, framework := range pyTestFrameworks {
			if strings.Contains(reqs, lib) {
				techs[framework] = TechnologyInfo{
					Name:       framework,
					Confidence: 0.9,
					Category:   "testing",
				}
			}
		}
	}

	// Check for Go test files
	for _, file := range td.files {
		if strings.HasSuffix(file, "_test.go") {
			techs["Go Testing"] = TechnologyInfo{
				Name:       "Go Testing",
				Confidence: 1.0,
				Category:   "testing",
			}
			break
		}
	}
}

// detectContainerization detects containerization tools
func (td *TechnologyDetector) detectContainerization(techs map[string]TechnologyInfo) {
	for _, file := range td.files {
		basename := filepath.Base(file)

		if basename == "Dockerfile" || strings.HasPrefix(basename, "Dockerfile.") {
			techs["Docker"] = TechnologyInfo{
				Name:       "Docker",
				Confidence: 1.0,
				Category:   "containerization",
			}
		}

		if basename == "docker-compose.yml" || basename == "docker-compose.yaml" {
			techs["Docker Compose"] = TechnologyInfo{
				Name:       "Docker Compose",
				Confidence: 1.0,
				Category:   "containerization",
			}
		}

		if basename == ".dockerignore" {
			if _, ok := techs["Docker"]; !ok {
				techs["Docker"] = TechnologyInfo{
					Name:       "Docker",
					Confidence: 0.8,
					Category:   "containerization",
				}
			}
		}

		if strings.Contains(file, "kubernetes") || strings.HasSuffix(file, ".k8s.yaml") {
			techs["Kubernetes"] = TechnologyInfo{
				Name:       "Kubernetes",
				Confidence: 0.9,
				Category:   "containerization",
			}
		}
	}
}

// detectCICD detects CI/CD tools
func (td *TechnologyDetector) detectCICD(techs map[string]TechnologyInfo) {
	cicdPaths := map[string]string{
		".github/workflows":       "GitHub Actions",
		".gitlab-ci.yml":          "GitLab CI",
		".travis.yml":             "Travis CI",
		"circle.yml":              "CircleCI",
		".circleci":               "CircleCI",
		"Jenkinsfile":             "Jenkins",
		"azure-pipelines.yml":     "Azure Pipelines",
		".drone.yml":              "Drone CI",
		"bitbucket-pipelines.yml": "Bitbucket Pipelines",
	}

	for _, file := range td.files {
		for pattern, cicd := range cicdPaths {
			if strings.Contains(file, pattern) {
				techs[cicd] = TechnologyInfo{
					Name:       cicd,
					Confidence: 1.0,
					Category:   "ci-cd",
				}
			}
		}
	}
}
