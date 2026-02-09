# Prompt My Project (PMP)

<div align="center">
  <img src="logo.png" alt="Prompt My Project Logo" height="200">
</div>

<div align="center">
  <a href="https://github.com/benoitpetit/prompt-my-project/blob/master/LICENSE">
    <img src="https://img.shields.io/badge/license-MIT-blue.svg" alt="MIT License"/>
  </a>
  <img src="https://img.shields.io/badge/go-1.21+-00ADD8?logo=go&logoColor=white" alt="Go Version"/>
  <img src="https://img.shields.io/github/v/release/benoitpetit/prompt-my-project?label=release" alt="Latest Release"/>
  <img src="https://img.shields.io/badge/PRs-welcome-brightgreen.svg" alt="PRs Welcome"/>
  <a href="https://liberapay.com/devbyben/donate">
    <img src="https://img.shields.io/badge/Liberapay-Donate-yellow.svg" alt="Donate on Liberapay"/>
  </a>
</div>

<p align="center">
  <strong>Transform your codebase into AI-ready prompts</strong><br>
  Prompt My Project (PMP) is a powerful command-line tool that analyzes your source code<br>
  and generates structured prompts for AI assistants like ChatGPT, Claude, and Gemini.
</p>

<p align="center">
  <a href="#-quick-start">Quick Start</a> •
  <a href="#-installation">Installation</a> •
  <a href="#-core-features">Features</a> •
  <a href="#-usage-guide">Usage</a> •
  <a href="#-examples">Examples</a>
</p>

---

## 🚀 Quick Start

```bash
# Install PMP
go install github.com/benoitpetit/prompt-my-project@latest

# Generate a prompt for your current project
pmp prompt .

# Analyze a GitHub repository
pmp github prompt https://github.com/username/project

# Generate a dependency graph
pmp graph . --format stdout:dot | dot -Tpng > graph.png

# Optional: compress context for large projects
pmp prompt . --summary-only           # Architecture overview
pmp prompt . --focus-changes          # Git-aware filtering
```

---

## 🎯 Why PMP?

### The Problem

When working with AI assistants, you need to share your codebase context efficiently. Manually copying files is tedious and error-prone.

### The Solution

PMP automatically generates structured prompts from your codebase:

- **🤖 AI-Ready Prompts**: Generate formatted prompts optimized for ChatGPT, Claude, Gemini, and other LLMs
- **📊 Project Analysis**: Technology detection, dependencies, code quality metrics
- **🛡️ Security Scanning**: Detect secrets, vulnerabilities, dangerous permissions
- **🐙 GitHub Integration**: Analyze any repository directly by URL
- **📈 Dependency Graphs**: Visual representations of your project structure

---

## 📦 Installation

### Option 1: Go Install (Recommended)

```bash
go install github.com/benoitpetit/prompt-my-project@latest
```

### Option 2: Installation Script

**Linux/macOS:**

```bash
curl -fsSL https://raw.githubusercontent.com/benoitpetit/prompt-my-project/master/scripts/install.sh | bash
```

**Windows (PowerShell):**

```powershell
iwr -useb https://raw.githubusercontent.com/benoitpetit/prompt-my-project/master/scripts/install.ps1 | iex
```

### Option 3: Download Binary

Download the latest binary from the [releases page](https://github.com/benoitpetit/prompt-my-project/releases)

### Verify Installation

```bash
pmp --help
```

---

## ✨ Core Features

### 🤖 AI-Ready Prompts

Generate structured prompts optimized for ChatGPT, Claude, Gemini, and other LLMs. Output in TXT, JSON, or XML formats.

### 📊 Visual Dependency Graphs

Create beautiful dependency graphs in multiple formats (DOT, JSON, XML, TXT) for documentation and analysis.

### 🔍 Project Analysis

- **Technology Detection**: Automatically identifies 50+ languages, frameworks, and tools
- **Dependency Analysis**: Multi-language support (Node.js, Go, Python, Rust, Ruby, PHP)
- **Code Quality Metrics**: Complexity analysis, maintainability index, code smell detection

### 🛡️ Security Scanning

- **Secret Detection**: Find exposed API keys, tokens, passwords (AWS, GitHub, JWT, SSH keys)
- **Vulnerability Scanning**: Built-in CVE database for known vulnerable dependencies
- **Permission Analysis**: Detect dangerous file permissions and configurations
- **Configuration Security**: Check Docker, CORS, Nginx configs for security issues

### 🐙 GitHub Integration

Analyze any GitHub repository directly without manual cloning. Supports HTTPS/SSH URLs and branch selection.

### ⚡ High Performance

- Parallel processing with configurable workers
- Intelligent file filtering and size limits
- Streaming support for massive projects (handles multi-GB codebases)
- Optimized sorting algorithms (50-90% performance improvement)

### 🔧 Smart Filtering

- Automatically excludes binary files
- Respects `.gitignore` rules
- Custom include/exclude patterns with glob support
- File size filtering (min/max)

---

## 📚 Usage Guide

### Basic Commands

| Command             | Description                | Example                                          |
| ------------------- | -------------------------- | ------------------------------------------------ |
| `pmp prompt`        | Generate AI-ready prompts  | `pmp prompt .`                                   |
| `pmp github prompt` | Analyze GitHub repo        | `pmp github prompt https://github.com/user/repo` |
| `pmp graph`         | Generate dependency graphs | `pmp graph .`                                    |
| `pmp github graph`  | GitHub repo graph          | `pmp github graph https://github.com/user/repo`  |
| `pmp completion`    | Shell completions          | `pmp completion bash`                            |

### Generate Prompts

#### Basic Usage

```bash
# Current directory
pmp prompt .

# Specific project
pmp prompt /path/to/project

# Output to stdout (for piping)
pmp prompt . --format stdout:txt
```

#### Advanced Filtering

```bash
# Include specific file types
pmp prompt . --include "*.go" --include "*.md"

# Exclude patterns
pmp prompt . --exclude "test/*" --exclude "*.log"

# Combine include/exclude
pmp prompt . --include "*.go" --exclude "*_test.go"

# Ignore .gitignore
pmp prompt . --no-gitignore
```

#### Size and Performance Controls

```bash
# Limit file sizes
pmp prompt . --min-size 100B --max-size 1MB

# Limit total processing
pmp prompt . --max-files 100 --max-total-size 5MB

# Adjust workers (default: number of CPUs)
pmp prompt . --workers 8
```

#### Output Formats

```bash
# JSON output
pmp prompt . --format json

# XML output
pmp prompt . --format xml

# Stdout for piping
pmp prompt . --format stdout:json | jq .

# Custom output directory
pmp prompt . --output /custom/path
```

### Generate Dependency Graphs

```bash
# DOT format (default)
pmp graph .

# JSON structure
pmp graph . --format json

# Text tree
pmp graph . --format txt

# Create visualization
pmp graph . --format stdout:dot | dot -Tpng > graph.png
pmp graph . --format stdout:dot | dot -Tsvg > graph.svg
pmp graph . --format stdout:dot | dot -Tpdf > graph.pdf
```

### GitHub Repository Analysis

```bash
# Basic analysis
pmp github prompt https://github.com/username/repo

# Specific branch
pmp github prompt https://github.com/username/repo --branch develop

# Generate graph
pmp github graph https://github.com/username/repo --format dot

# With filtering
pmp github prompt https://github.com/username/repo --include "*.go" --exclude "*test*"

# Multiple formats
pmp github prompt https://github.com/username/repo --format json
pmp github graph https://github.com/username/repo --format stdout:dot | dot -Tpng > graph.png
```

**Supported URL Formats:**

- `https://github.com/user/repo`
- `https://github.com/user/repo.git`
- `git@github.com:user/repo.git`

**Features:**

- Shallow cloning for speed (depth=1)
- Automatic cleanup of temporary files
- Full support for all PMP features
- Works with private repos (via SSH keys or credentials)

### Context Compression

For large projects, PMP provides optional flags to compress the context:

#### AST-Based Summarization (`--summary-only`)

Generate only function signatures and interfaces instead of full source code:

```bash
# Generate architecture overview
pmp prompt . --summary-only

# Combine with filters
pmp prompt . --summary-only --include "*.go" --max-files 100
```

**Supported Languages:** Go, JavaScript/TypeScript, Python, Java, C/C++

#### Git-Aware Context (`--focus-changes`)

Prioritize recently modified files:

```bash
# Focus on modified files
pmp prompt . --focus-changes

# Include last N commits
pmp prompt . --focus-changes --recent-commits 5
```

#### Custom Patterns (`--summary-patterns`)

Manually specify files to summarize:

```bash
# Summarize vendor/dependencies
pmp prompt . --summary-patterns "vendor/**" --summary-patterns "node_modules/**"
```

**When to use:**

- Very large projects (>100 files)
- Architecture documentation needs
- Pull request focused analysis
- Active development workflows

---

## 🔬 Examples

### For AI Assistants

```bash
# Quick prompt for ChatGPT/Claude
pmp prompt . --format stdout:txt | pbcopy                      # macOS
pmp prompt . --format stdout:txt | xclip -selection clipboard  # Linux

# Structured JSON for API integration
pmp prompt . --format stdout:json | curl -X POST https://api.example.com/analyze -d @-
```

### Code Analysis Workflows

```bash
# Analyze only source code (no tests, no docs)
pmp prompt . --include "*.go" --include "*.js" --include "*.py" \
  --exclude "*test*" --exclude "*.md"

# Large project with limits
pmp prompt . --max-files 200 --max-total-size 15MB --workers 8

# Get project statistics
pmp prompt . --format stdout:json | jq '.statistics'

# List detected technologies
pmp prompt . --format stdout:json | jq '.technologies[]'
```

### Documentation Generation

```bash
# Project structure
pmp graph . --format txt > STRUCTURE.md

# Visual dependency graph
pmp graph . --format stdout:dot | dot -Tsvg > dependencies.svg

# Multiple documentation formats
pmp graph . --format json > structure.json
pmp graph . --format xml > structure.xml
```

### Integration Examples

#### With jq (JSON processing)

```bash
# Get project statistics
pmp prompt . --format stdout:json | jq '.statistics'

# List detected technologies
pmp prompt . --format stdout:json | jq '.technologies[]'

# List key files
pmp prompt . --format stdout:json | jq '.key_files[]'

# Get file types distribution
pmp prompt . --format stdout:json | jq '.file_types[]'

# Extract file paths and sizes
pmp prompt . --format stdout:json | jq '.files[] | {path: .path, size: .size}'

# Count files by extension
pmp prompt . --format stdout:json | jq '.file_types[] | "\(.extension): \(.count)"'
```

#### CI/CD Integration

```bash
# Generate project context for automated reviews
pmp prompt . --format json --output build/context.json

# Check minimum file count (fail if too few files)
pmp prompt . --format stdout:json | jq -e '.statistics.file_count > 0'

# Verify technologies detected
pmp prompt . --format stdout:json | jq -e '.technologies | length > 0'

# Create documentation automatically
pmp graph . --format txt --output docs/STRUCTURE.md
```

#### External Repository Analysis

```bash
# Audit open source project
pmp github prompt https://github.com/vendor/project --format stdout:json | jq '.statistics'

# Compare branches
pmp github prompt https://github.com/user/repo --branch main --format stdout:json > main.json
pmp github prompt https://github.com/user/repo --branch dev --format stdout:json > dev.json
diff <(jq -S . main.json) <(jq -S . dev.json)

# List technologies in external library
pmp github prompt https://github.com/vendor/library --format stdout:json | jq '.technologies[]'

# Visual graph of external project
pmp github graph https://github.com/user/project --format stdout:dot | dot -Tsvg > graph.svg

# Get project statistics
pmp github prompt https://github.com/user/project --format stdout:json | jq '.statistics'
```

---

## 🛡️ Security Features

PMP includes comprehensive security scanning for your codebase.

### Secret Detection

Detects 13 types of secrets in your code:

- **AWS Credentials**: Access Keys, Secret Keys
- **GitHub Tokens**: Personal Access Tokens, OAuth tokens
- **JWT Tokens**: JSON Web Tokens
- **SSH Keys**: Private keys in code
- **API Keys**: Generic API keys, Stripe, Slack, etc.
- **OAuth Secrets**: Client secrets
- **Database URLs**: Connection strings with credentials
- **Passwords**: Hard-coded passwords
- **Webhooks**: Slack webhooks and similar

### File Permission Analysis

- World-writable files (chmod 777)
- Executable configuration files
- Private keys without proper permissions (should be 600)
- Sensitive files with insecure permissions

### Configuration Security

- **Dockerfile**: Missing USER directive, exposed SSH ports, running as root
- **Nginx**: Missing security headers, insecure configurations
- **CORS**: Wildcard misuse, overly permissive settings
- **Environment Files**: Exposed .env files in repository

### Vulnerability Scanning

Built-in CVE database for detecting vulnerable dependencies:

- **Node.js**: lodash, axios, express, and more
- **Python**: django, flask, requests, and more
- **Ruby**: rails, devise, and more
- **PHP**: symfony, laravel components, and more

### Dangerous Files Detection

- `.env` files in repository
- Private key files (`.pem`, `.key`, `id_rsa`)
- Credential files
- Database dumps
- Configuration files with secrets

### Usage Examples

```bash
# Analyze codebase and output to file
pmp prompt . --format txt

# Get project overview in JSON
pmp prompt . --format stdout:json | jq '.'

# Check which technologies are detected
pmp prompt . --format stdout:json | jq '.technologies[]'

# List all key files in the project
pmp prompt . --format stdout:json | jq '.key_files[]'

# Get file count and size statistics
pmp prompt . --format stdout:json | jq '.statistics'

# Note: Security scanning, dependency analysis, and code quality metrics
# are performed during analysis but detailed results are shown in TXT format.
# For full analysis, use: pmp prompt . --format txt
```

---

## 🎓 Advanced Features

### Technology Detection

Automatically detects 50+ technologies with confidence scoring:

**Languages:**

- Go, JavaScript, TypeScript, Python, Java, Ruby, PHP
- C#, C, C++, Rust, Kotlin, Swift, Scala
- R, Dart, Lua, Perl, Elixir, Haskell

**Frameworks:**

- React, Vue, Angular, Svelte
- Django, Flask, FastAPI
- Rails, Sinatra
- Laravel, Symfony
- Express, Koa, NestJS
- Spring, Spring Boot

**Build Tools:**

- Make, CMake, Gradle, Maven
- npm, yarn, pnpm
- pip, poetry, pipenv
- cargo, bundle, composer

**Databases:**

- PostgreSQL, MySQL, MongoDB, Redis
- SQLite, Elasticsearch, Cassandra

**Testing:**

- Jest, Mocha, Pytest, RSpec, PHPUnit
- JUnit, TestNG, Go testing

**CI/CD:**

- GitHub Actions, GitLab CI, Jenkins
- Travis CI, CircleCI

**Containers:**

- Docker, Docker Compose, Kubernetes

### Dependency Analysis

Multi-language dependency analysis with detailed insights:

**Supported Ecosystems:**

- **Node.js**: package.json, package-lock.json, yarn.lock, pnpm-lock.yaml
- **Go**: go.mod, go.sum
- **Python**: requirements.txt, Pipfile, Pipfile.lock, pyproject.toml, poetry.lock
- **Rust**: Cargo.toml, Cargo.lock
- **Ruby**: Gemfile, Gemfile.lock
- **PHP**: composer.json, composer.lock

**Features:**

- Dependency classification (production, dev, optional)
- Version information and constraints
- Vulnerability detection via CVE database
- Obsolete package detection
- Dependency count and metrics

**Usage:**

```bash
# View dependency analysis (shown in TXT format)
pmp prompt . --format txt

# Export full analysis including dependencies
pmp prompt . --format json

# Note: Dependency details are analyzed and shown in the TXT output.
# The JSON format includes basic project information and file contents.
```

### Code Quality Metrics

Comprehensive code quality analysis:

**Metrics Provided:**

- **Cyclomatic Complexity**: Per-file complexity analysis
- **Maintainability Index**: 0-100 score based on complexity, code size, and structure
- **Code Smells**: Automatic detection of common anti-patterns
- **Test Coverage Estimate**: Based on test file patterns

**Code Smells Detected:**

- Long functions (>50 lines)
- Long files (>500 lines)
- High complexity functions
- Deeply nested code
- Too many parameters
- Duplicate code patterns

**Usage:**

```bash
# View quality metrics (shown in TXT format)
pmp prompt . --format txt

# Export full analysis including quality metrics
pmp prompt . --format json

# Note: Code quality analysis is performed and details are shown in TXT output.
# Use the TXT format to see complexity, maintainability, and code smells.
```

### Linter Integration

Integrates with popular linters when available:

**Supported Linters:**

- **JavaScript/TypeScript**: ESLint
- **Go**: golint, staticcheck, go vet
- **Python**: pylint
- **Ruby**: RuboCop

**Features:**

- Automatic linter detection
- Formatted issue summary
- Severity classification (error, warning, info)
- Fall back to basic linting when external tools unavailable

**Usage:**

```bash
# View linter results (shown in TXT format)
pmp prompt . --format txt

# Note: Linter integration output is included in the TXT format.
# External linters are automatically detected and run when available.
```

### Streaming Support

For massive projects, PMP automatically uses streaming mode:

**Features:**

- Memory-efficient processing for >100MB projects
- Configurable chunk sizes
- Progress tracking
- Handles projects of any size
- Automatic detection and activation

**When Streaming Activates:**

- Projects with >100MB total size
- More than 10,000 files
- Available system memory below threshold
- Manual activation via configuration

---

## ⚙️ Configuration

### Environment Variables

```bash
# Output directory
export PMP_OUTPUT_DIR="./analysis"

# Worker count
export PMP_WORKERS=8

# Output format
export PMP_FORMAT="json"

# File limits
export PMP_MAX_FILES=500
export PMP_MAX_TOTAL_SIZE="10MB"
export PMP_MIN_SIZE="1KB"
export PMP_MAX_SIZE="100MB"

# Patterns
export PMP_EXCLUDE="vendor/**,node_modules/**"
export PMP_INCLUDE="*.go,*.js,*.py"
```

### Project Configuration (.pmprc)

Create `.pmprc` in your project root:

```json
{
  "exclude": ["vendor/**", "node_modules/**", "dist/**", "build/**", ".git/**"],
  "include": ["*.go", "*.js", "*.ts", "*.py", "*.java", "*.md"],
  "minSize": "1KB",
  "maxSize": "100MB",
  "maxFiles": 500,
  "maxTotalSize": "10MB",
  "format": "txt",
  "outputDir": "pmp_output",
  "workers": 8,
  "noGitignore": false,
  "summaryOnly": false,
  "focusChanges": false,
  "recentCommits": 3,
  "summaryPatterns": [
    "vendor/**",
    "node_modules/**",
    "**/generated/**",
    "**/*.pb.go"
  ]
}
```

### Shell Autocompletion

Enable autocompletion for your shell:

**Bash:**

```bash
source <(pmp completion bash)

# For all sessions (Linux):
pmp completion bash > /etc/bash_completion.d/pmp

# For all sessions (macOS):
pmp completion bash > /usr/local/etc/bash_completion.d/pmp
```

**Zsh:**

```bash
echo 'autoload -U compinit; compinit' >> ~/.zshrc
pmp completion zsh > "${fpath[1]}/_pmp"
```

**Fish:**

```bash
pmp completion fish | source

# For all sessions:
pmp completion fish > ~/.config/fish/completions/pmp.fish
```

**PowerShell:**

```powershell
pmp completion powershell | Out-String | Invoke-Expression

# Add to $PROFILE for all sessions
```

---

## 🔧 Troubleshooting

### Permission Denied

```bash
# Make binary executable
chmod +x pmp

# Or install via go install
go install github.com/benoitpetit/prompt-my-project@latest
```

### Command Not Found

```bash
# Check if Go bin is in PATH
echo $PATH | grep -q "$(go env GOPATH)/bin"

# If not, add to PATH
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.bashrc
source ~/.bashrc
```

### Large Project Performance

```bash
# Limit processing
pmp prompt . --max-files 200 --max-total-size 15MB

# Adjust workers
pmp prompt . --workers 4

# Use specific patterns
pmp prompt . --include "*.go" --include "*.js"
```

### Output Too Large

```bash
# Focus on specific files
pmp prompt . --include "*.go" --exclude "*test*" --max-files 50

# Size limits
pmp prompt . --max-size 50KB --max-total-size 5MB

# Optional: use context compression
pmp prompt . --summary-only                          # Architecture overview
pmp prompt . --focus-changes                         # Git-aware filtering
pmp prompt . --summary-patterns "vendor/**"          # Custom patterns
```

### Missing Small Files

```bash
# Include very small files
pmp prompt . --min-size 0
```

### Git Context Not Working

```bash
# Ensure you're in a git repository
git status

# If not initialized, initialize git
git init
git add .
git commit -m "Initial commit"

# Now use focus-changes
pmp prompt . --focus-changes
```

### Binary Detection Issues

```bash
# Force include files if wrongly detected as binary
pmp prompt . --no-gitignore --min-size 0

# Or add to .pmprc:
{
  "noGitignore": true,
  "minSize": "0"
}
```

---

## 📊 Project Statistics

View comprehensive project statistics:

```bash
# Project statistics
pmp prompt . --format stdout:json | jq '.statistics'

# Technology stack
pmp prompt . --format stdout:json | jq '.technologies'

# File types distribution
pmp prompt . --format stdout:json | jq '.file_types'

# Key files
pmp prompt . --format stdout:json | jq '.key_files'

# Potential issues
pmp prompt . --format stdout:json | jq '.issues'

# Get complete analysis (includes security, dependencies, quality)
pmp prompt . --format txt

# Export everything for processing
pmp prompt . --format json
```

---

## 🤝 Contributing

Contributions are welcome! Here's how you can help:

1. **Fork** the repository
2. **Create** your feature branch (`git checkout -b feature/amazing-feature`)
3. **Commit** your changes (`git commit -m 'Add amazing feature'`)
4. **Push** to the branch (`git push origin feature/amazing-feature`)
5. **Open** a Pull Request

### Development Setup

```bash
# Clone the repository
git clone https://github.com/benoitpetit/prompt-my-project.git
cd prompt-my-project

# Build
go build -o pmp .

# Run tests
go test ./...

# Run benchmarks
go test -bench=. ./pkg/analyzer/
```

---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## 💖 Support

If you find PMP useful, consider supporting its development:

- ⭐ **Star** the repository
- 🐛 **Report** bugs and request features via [Issues](https://github.com/benoitpetit/prompt-my-project/issues)
- 💰 **Donate** on [Liberapay](https://liberapay.com/devbyben/donate)
- 📢 **Share** with your network

---

## 🔗 Links

- **Repository**: [github.com/benoitpetit/prompt-my-project](https://github.com/benoitpetit/prompt-my-project)
- **Issues**: [Report bugs or request features](https://github.com/benoitpetit/prompt-my-project/issues)
- **Releases**: [Download latest version](https://github.com/benoitpetit/prompt-my-project/releases)
- **Donate**: [Support on Liberapay](https://liberapay.com/devbyben/donate)

---

<div align="center">
  <p>Made with ❤️ by <a href="https://github.com/benoitpetit">Benoit Petit</a></p>
  <p>Transform your codebase • Optimize your prompts • Build better with AI</p>
</div>
