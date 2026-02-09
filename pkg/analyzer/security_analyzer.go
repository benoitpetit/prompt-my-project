package analyzer

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// SecurityIssue represents a security concern
type SecurityIssue struct {
	Type        string `json:"type"`     // secret, permission, config, etc.
	Severity    string `json:"severity"` // critical, high, medium, low
	File        string `json:"file"`
	Line        int    `json:"line"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
	Matched     string `json:"matched,omitempty"`
}

// SecurityReport contains all security findings
type SecurityReport struct {
	TotalIssues    int             `json:"total_issues"`
	CriticalCount  int             `json:"critical_count"`
	HighCount      int             `json:"high_count"`
	MediumCount    int             `json:"medium_count"`
	LowCount       int             `json:"low_count"`
	ByType         map[string]int  `json:"by_type"`
	Issues         []SecurityIssue `json:"issues"`
	DangerousFiles []string        `json:"dangerous_files"`
}

// SecurityAnalyzer analyzes security aspects of the codebase
type SecurityAnalyzer struct {
	rootDir        string
	files          []string
	secretPatterns map[string]*regexp.Regexp
	configChecks   []ConfigCheck
}

// ConfigCheck represents a security configuration check
type ConfigCheck struct {
	Name        string
	FilePattern string
	CheckFunc   func(content string) []SecurityIssue
}

// NewSecurityAnalyzer creates a new security analyzer
func NewSecurityAnalyzer(rootDir string, files []string) *SecurityAnalyzer {
	sa := &SecurityAnalyzer{
		rootDir: rootDir,
		files:   files,
	}

	sa.initSecretPatterns()
	sa.initConfigChecks()

	return sa
}

// initSecretPatterns initializes regex patterns for secret detection
func (sa *SecurityAnalyzer) initSecretPatterns() {
	sa.secretPatterns = map[string]*regexp.Regexp{
		// API Keys
		"AWS Access Key":  regexp.MustCompile(`(?i)(AWS|aws)(_|-)?ACCESS(_|-)?KEY(_|-)?ID['"\s]*[:=]\s*['"]?[A-Z0-9]{20}['"]?`),
		"AWS Secret Key":  regexp.MustCompile(`(?i)(AWS|aws)(_|-)?SECRET(_|-)?ACCESS(_|-)?KEY['"\s]*[:=]\s*['"]?[A-Za-z0-9/+=]{40}['"]?`),
		"GitHub Token":    regexp.MustCompile(`(?i)gh[pousr]_[A-Za-z0-9_]{36,}`),
		"Generic API Key": regexp.MustCompile(`(?i)api[_-]?key['"\s]*[:=]\s*['"][A-Za-z0-9_\-]{20,}['"]`),

		// Private Keys
		"Private Key": regexp.MustCompile(`-----BEGIN\s+(RSA|DSA|EC|OPENSSH|PGP)\s+PRIVATE\s+KEY-----`),

		// Passwords
		"Password in Code": regexp.MustCompile(`(?i)(password|passwd|pwd)['"\s]*[:=]\s*['"][^'"]{8,}['"]`),

		// Tokens
		"JWT Token":    regexp.MustCompile(`eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`),
		"Bearer Token": regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9_\-\.=]{20,}`),
		"Slack Token":  regexp.MustCompile(`xox[baprs]-[0-9]{12}-[0-9]{12}-[A-Za-z0-9]{24}`),
		"Stripe Key":   regexp.MustCompile(`(?i)(sk|pk)_(test|live)_[A-Za-z0-9]{24,}`),

		// Database Credentials
		"Database URL": regexp.MustCompile(`(?i)(postgres|mysql|mongodb)://[^:]+:[^@]+@[^/]+`),

		// OAuth
		"OAuth Client Secret": regexp.MustCompile(`(?i)client[_-]?secret['"\s]*[:=]\s*['"][A-Za-z0-9_\-]{20,}['"]`),

		// Generic Secrets
		"Secret in Code": regexp.MustCompile(`(?i)secret['"\s]*[:=]\s*['"][A-Za-z0-9_\-]{16,}['"]`),
	}
}

// initConfigChecks initializes configuration security checks
func (sa *SecurityAnalyzer) initConfigChecks() {
	sa.configChecks = []ConfigCheck{
		{
			Name:        "Docker Exposed Ports",
			FilePattern: "Dockerfile",
			CheckFunc:   sa.checkDockerfile,
		},
		{
			Name:        "Nginx Security Headers",
			FilePattern: "nginx.conf",
			CheckFunc:   sa.checkNginxConfig,
		},
		{
			Name:        "CORS Configuration",
			FilePattern: "*.js|*.ts|*.go|*.py",
			CheckFunc:   sa.checkCORSConfig,
		},
	}
}

// Analyze performs comprehensive security analysis
func (sa *SecurityAnalyzer) Analyze() (*SecurityReport, error) {
	report := &SecurityReport{
		ByType:         make(map[string]int),
		Issues:         []SecurityIssue{},
		DangerousFiles: []string{},
	}

	// Check file permissions
	sa.checkFilePermissions(report)

	// Scan for secrets
	sa.scanForSecrets(report)

	// Check configurations
	sa.checkConfigurations(report)

	// Check for dangerous files
	sa.checkDangerousFiles(report)

	// Calculate totals
	sa.calculateTotals(report)

	return report, nil
}

// checkFilePermissions checks for dangerous file permissions
func (sa *SecurityAnalyzer) checkFilePermissions(report *SecurityReport) {
	for _, file := range sa.files {
		fullPath := filepath.Join(sa.rootDir, file)
		info, err := os.Stat(fullPath)
		if err != nil {
			continue
		}

		mode := info.Mode()
		perm := mode.Perm()

		// Check for world-writable files
		if perm&0002 != 0 {
			report.Issues = append(report.Issues, SecurityIssue{
				Type:        "permission",
				Severity:    "high",
				File:        file,
				Description: "File is world-writable",
				Suggestion:  "Remove write permissions for others: chmod o-w " + file,
			})
		}

		// Check for executable configuration files
		if isConfigFile(file) && mode&0111 != 0 {
			report.Issues = append(report.Issues, SecurityIssue{
				Type:        "permission",
				Severity:    "medium",
				File:        file,
				Description: "Configuration file should not be executable",
				Suggestion:  "Remove executable bit: chmod -x " + file,
			})
		}

		// Check for overly permissive private keys
		if strings.Contains(file, "id_rsa") || strings.Contains(file, ".pem") {
			if perm&0077 != 0 {
				report.Issues = append(report.Issues, SecurityIssue{
					Type:        "permission",
					Severity:    "critical",
					File:        file,
					Description: "Private key has overly permissive permissions",
					Suggestion:  "Restrict permissions: chmod 600 " + file,
				})
			}
		}
	}
}

// isConfigFile checks if a file is a configuration file
func isConfigFile(file string) bool {
	configExts := []string{".json", ".yaml", ".yml", ".toml", ".ini", ".conf", ".config"}
	configFiles := []string{".env", ".npmrc", ".gitconfig"}

	for _, ext := range configExts {
		if strings.HasSuffix(file, ext) {
			return true
		}
	}

	baseName := filepath.Base(file)
	for _, name := range configFiles {
		if baseName == name || strings.HasPrefix(baseName, name) {
			return true
		}
	}

	return false
}

// scanForSecrets scans files for potential secrets
func (sa *SecurityAnalyzer) scanForSecrets(report *SecurityReport) {
	// Skip binary files and large files
	maxSize := int64(10 * 1024 * 1024) // 10MB

	for _, file := range sa.files {
		fullPath := filepath.Join(sa.rootDir, file)

		// Skip certain directories
		if shouldSkipForSecrets(file) {
			continue
		}

		info, err := os.Stat(fullPath)
		if err != nil || info.Size() > maxSize {
			continue
		}

		f, err := os.Open(fullPath)
		if err != nil {
			continue
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		lineNum := 0

		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			// Skip comments (basic detection)
			trimmed := strings.TrimSpace(line)
			if strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "/*") {
				continue
			}

			// Check each pattern
			for secretType, pattern := range sa.secretPatterns {
				if pattern.MatchString(line) {
					// Extract matched portion
					matched := pattern.FindString(line)
					if len(matched) > 50 {
						matched = matched[:50] + "..."
					}

					severity := "critical"
					if secretType == "Password in Code" || secretType == "Secret in Code" {
						severity = "high"
					}

					report.Issues = append(report.Issues, SecurityIssue{
						Type:        "secret",
						Severity:    severity,
						File:        file,
						Line:        lineNum,
						Description: secretType + " detected in code",
						Suggestion:  "Move sensitive data to environment variables or secure vault",
						Matched:     matched,
					})
				}
			}
		}
	}
}

// shouldSkipForSecrets checks if file should be skipped for secret scanning
func shouldSkipForSecrets(file string) bool {
	skipDirs := []string{"node_modules/", "vendor/", ".git/", "dist/", "build/", "target/"}
	skipExts := []string{".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".woff", ".ttf", ".eot"}

	for _, dir := range skipDirs {
		if strings.Contains(file, dir) {
			return true
		}
	}

	for _, ext := range skipExts {
		if strings.HasSuffix(file, ext) {
			return true
		}
	}

	return false
}

// checkConfigurations checks security configurations
func (sa *SecurityAnalyzer) checkConfigurations(report *SecurityReport) {
	for _, file := range sa.files {
		for _, check := range sa.configChecks {
			if matchesPattern(file, check.FilePattern) {
				fullPath := filepath.Join(sa.rootDir, file)
				content, err := os.ReadFile(fullPath)
				if err != nil {
					continue
				}

				issues := check.CheckFunc(string(content))
				for _, issue := range issues {
					issue.File = file
					report.Issues = append(report.Issues, issue)
				}
			}
		}
	}
}

// matchesPattern checks if filename matches pattern
func matchesPattern(filename, pattern string) bool {
	if strings.Contains(pattern, "|") {
		patterns := strings.Split(pattern, "|")
		for _, p := range patterns {
			if matchesPattern(filename, p) {
				return true
			}
		}
		return false
	}

	if pattern == "*" {
		return true
	}

	if strings.Contains(pattern, "*") {
		ext := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(filename, ext)
	}

	return strings.Contains(filename, pattern)
}

// checkDockerfile checks Docker configuration security
func (sa *SecurityAnalyzer) checkDockerfile(content string) []SecurityIssue {
	issues := []SecurityIssue{}
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check for running as root
		if strings.HasPrefix(trimmed, "USER root") {
			issues = append(issues, SecurityIssue{
				Type:        "config",
				Severity:    "high",
				Line:        i + 1,
				Description: "Container runs as root user",
				Suggestion:  "Create and use a non-root user in Dockerfile",
			})
		}

		// Check for EXPOSE with dangerous ports
		if strings.HasPrefix(trimmed, "EXPOSE") {
			if strings.Contains(trimmed, "22") {
				issues = append(issues, SecurityIssue{
					Type:        "config",
					Severity:    "high",
					Line:        i + 1,
					Description: "SSH port exposed in container",
					Suggestion:  "Avoid exposing SSH in containers",
				})
			}
		}
	}

	// Check for missing USER instruction
	hasUser := false
	for _, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "USER ") {
			hasUser = true
			break
		}
	}

	if !hasUser {
		issues = append(issues, SecurityIssue{
			Type:        "config",
			Severity:    "medium",
			Line:        0,
			Description: "No USER instruction found - container will run as root",
			Suggestion:  "Add USER instruction to run as non-root",
		})
	}

	return issues
}

// checkNginxConfig checks Nginx security headers
func (sa *SecurityAnalyzer) checkNginxConfig(content string) []SecurityIssue {
	issues := []SecurityIssue{}

	securityHeaders := map[string]string{
		"X-Frame-Options":           "DENY or SAMEORIGIN",
		"X-Content-Type-Options":    "nosniff",
		"X-XSS-Protection":          "1; mode=block",
		"Strict-Transport-Security": "max-age=31536000",
	}

	for header, value := range securityHeaders {
		if !strings.Contains(content, header) {
			issues = append(issues, SecurityIssue{
				Type:        "config",
				Severity:    "medium",
				Line:        0,
				Description: "Missing security header: " + header,
				Suggestion:  "Add header: add_header " + header + " " + value + ";",
			})
		}
	}

	return issues
}

// checkCORSConfig checks CORS configuration
func (sa *SecurityAnalyzer) checkCORSConfig(content string) []SecurityIssue {
	issues := []SecurityIssue{}
	lines := strings.Split(content, "\n")

	for i, line := range lines {
		// Check for wildcard CORS
		if strings.Contains(line, "Access-Control-Allow-Origin") && strings.Contains(line, "*") {
			issues = append(issues, SecurityIssue{
				Type:        "config",
				Severity:    "high",
				Line:        i + 1,
				Description: "CORS allows all origins (*)",
				Suggestion:  "Restrict CORS to specific trusted domains",
			})
		}

		// Check for insecure CORS with credentials
		if strings.Contains(line, "credentials: true") || strings.Contains(line, "AllowCredentials") {
			if strings.Contains(content, "Access-Control-Allow-Origin") && strings.Contains(content, "*") {
				issues = append(issues, SecurityIssue{
					Type:        "config",
					Severity:    "critical",
					Line:        i + 1,
					Description: "CORS allows credentials with wildcard origin",
					Suggestion:  "Never use credentials: true with origin: *",
				})
			}
		}
	}

	return issues
}

// checkDangerousFiles checks for presence of dangerous files
func (sa *SecurityAnalyzer) checkDangerousFiles(report *SecurityReport) {
	dangerousPatterns := []struct {
		pattern     string
		severity    string
		description string
	}{
		{".env", "critical", "Environment file with potential secrets"},
		{".pem", "critical", "Private key file"},
		{"id_rsa", "critical", "SSH private key"},
		{".key", "critical", "Private key file"},
		{"credentials", "high", "File containing credentials"},
		{"secrets", "high", "File containing secrets"},
		{".htpasswd", "medium", "Password file"},
		{"database.yml", "medium", "Database configuration with potential credentials"},
	}

	for _, file := range sa.files {
		baseName := filepath.Base(file)

		for _, dp := range dangerousPatterns {
			if strings.Contains(baseName, dp.pattern) {
				report.DangerousFiles = append(report.DangerousFiles, file)
				report.Issues = append(report.Issues, SecurityIssue{
					Type:        "file",
					Severity:    dp.severity,
					File:        file,
					Description: dp.description,
					Suggestion:  "Ensure this file is in .gitignore and not committed",
				})
			}
		}
	}
}

// calculateTotals calculates summary statistics
func (sa *SecurityAnalyzer) calculateTotals(report *SecurityReport) {
	report.TotalIssues = len(report.Issues)

	for _, issue := range report.Issues {
		report.ByType[issue.Type]++

		switch issue.Severity {
		case "critical":
			report.CriticalCount++
		case "high":
			report.HighCount++
		case "medium":
			report.MediumCount++
		case "low":
			report.LowCount++
		}
	}
}

// GetSecuritySummary returns a human-readable security summary
func GetSecuritySummary(report *SecurityReport) string {
	if report == nil || report.TotalIssues == 0 {
		return "✅ No security issues detected.\n"
	}

	var sb strings.Builder

	sb.WriteString("Security Analysis Report:\n")
	sb.WriteString("========================\n\n")

	sb.WriteString("Total Issues: ")
	sb.WriteString(intToString(report.TotalIssues))
	sb.WriteString("\n\n")

	if report.CriticalCount > 0 {
		sb.WriteString("🔴 Critical: ")
		sb.WriteString(intToString(report.CriticalCount))
		sb.WriteString("\n")
	}
	if report.HighCount > 0 {
		sb.WriteString("🟠 High: ")
		sb.WriteString(intToString(report.HighCount))
		sb.WriteString("\n")
	}
	if report.MediumCount > 0 {
		sb.WriteString("🟡 Medium: ")
		sb.WriteString(intToString(report.MediumCount))
		sb.WriteString("\n")
	}
	if report.LowCount > 0 {
		sb.WriteString("🟢 Low: ")
		sb.WriteString(intToString(report.LowCount))
		sb.WriteString("\n")
	}

	if len(report.ByType) > 0 {
		sb.WriteString("\nBy Type:\n")
		for issueType, count := range report.ByType {
			sb.WriteString("  - ")
			sb.WriteString(issueType)
			sb.WriteString(": ")
			sb.WriteString(intToString(count))
			sb.WriteString("\n")
		}
	}

	if len(report.DangerousFiles) > 0 {
		sb.WriteString("\n⚠️  Dangerous Files Found: ")
		sb.WriteString(intToString(len(report.DangerousFiles)))
		sb.WriteString("\n")
	}

	// Show top 5 critical/high issues
	criticalIssues := []SecurityIssue{}
	for _, issue := range report.Issues {
		if issue.Severity == "critical" || issue.Severity == "high" {
			criticalIssues = append(criticalIssues, issue)
			if len(criticalIssues) >= 5 {
				break
			}
		}
	}

	if len(criticalIssues) > 0 {
		sb.WriteString("\nTop Issues:\n")
		for i, issue := range criticalIssues {
			sb.WriteString(intToString(i + 1))
			sb.WriteString(". [")
			sb.WriteString(strings.ToUpper(issue.Severity))
			sb.WriteString("] ")
			sb.WriteString(issue.Description)
			sb.WriteString("\n   File: ")
			sb.WriteString(issue.File)
			if issue.Line > 0 {
				sb.WriteString(":")
				sb.WriteString(intToString(issue.Line))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
