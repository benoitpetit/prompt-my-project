# Prompt My Project (PMP)

[README EN FRANCAIS](/README_FR.md) 

<p align="center">
    <img src="./logo.png" alt="Prompt My Project" width="800">
    <p align="center">Command-line tool to generate structured prompts from your source code, optimized for AI assistants.</p>
</p>

<div align="center">
    <a href="https://github.com/benoitpetit/prompt-my-project/blob/main/LICENSE">
        <img src="https://img.shields.io/badge/License-MIT-yellow.svg" alt="License: MIT">
    </a>
    <a href="https://github.com/benoitpetit/prompt-my-project/releases">
        <img src="https://img.shields.io/github/v/release/benoitpetit/prompt-my-project" alt="Latest Release">
    </a>
    <a href="https://opensource.org">
        <img src="https://img.shields.io/badge/Open%20Source-%E2%9D%A4-brightgreen" alt="Open Source Love">
    </a>
    <a href="https://github.com/benoitpetit/prompt-my-project/stargazers">
        <img src="https://img.shields.io/github/stars/benoitpetit/prompt-my-project" alt="GitHub Stars">
    </a>
    <a href="https://golang.org/dl/">
        <img src="https://img.shields.io/badge/Go-%3E%3D%201.21-blue.svg" alt="Go Version">
    </a>
</div>

## Overview

PMP analyzes your codebase and generates comprehensive, structured prompts optimized for AI assistants like ChatGPT, Claude, or Gemini. It extracts key information, detects technologies, and formats output to maximize the context provided to AI tools.

## ✨ Key Features

- 📂 **Smart Project Analysis**: Recursively scans your project structure with binary detection and .gitignore support
- 🎯 **Flexible Filtering**: Advanced pattern matching for including or excluding specific files and directories
- 📊 **Comprehensive Statistics**: File counts, size distribution, and token estimation for AI models
- 🔬 **Technology Detection**: Automatically identifies programming languages and frameworks used
- 📝 **Multiple Output Formats**: Export as TXT, JSON, or XML with detailed project information
- 🚀 **High Performance**: Concurrent processing with smart caching and memory management

## 🚀 Installation

### Using Go Install (New!)

The simplest method if you have Go installed:

```bash
go install github.com/benoitpetit/prompt-my-project@latest
```

### Script Installation

#### macOS & Linux
```bash
curl -fsSL https://raw.githubusercontent.com/benoitpetit/prompt-my-project/master/scripts/install.sh | bash
```

#### Windows
```powershell
irm https://raw.githubusercontent.com/benoitpetit/prompt-my-project/master/scripts/install.ps1 | iex
```

### Manual Installation

1. Download the latest release from [GitHub Releases](https://github.com/benoitpetit/prompt-my-project/releases)
2. Extract the archive
3. Move the binary to a location in your PATH

## 🛠️ Usage

### Basic Syntax

```bash
pmp [options] [path]
```

### Common Commands

```bash
# Analyze current directory
pmp .

# Analyze specific project
pmp /path/to/project

# Include only specific file types
pmp . -i "*.go" -i "*.md"

# Exclude test files and vendor directory
pmp . -e "test/*" -e "vendor/*"

# Generate JSON output
pmp . --format json

# Specify output directory
pmp . -o ~/prompts
```

### Available Options

| Option           | Alias | Description                          | Default |
| ---------------- | ----- | ------------------------------------ | ------- |
| `--include`      | `-i`  | Include only files matching patterns | - |
| `--exclude`      | `-e`  | Exclude files matching patterns      | - |
| `--min-size`     | -     | Minimum file size                    | 1KB |
| `--max-size`     | -     | Maximum file size                    | 100MB |
| `--no-gitignore` | -     | Ignore .gitignore file              | false |
| `--output`       | `-o`  | Output folder for prompt file        | ./prompts |
| `--workers`      | -     | Number of parallel workers           | CPU cores |
| `--max-files`    | -     | Maximum number of files              | 500 |
| `--max-total-size` | -   | Maximum total size                   | 10MB |
| `--format`       | `-f`  | Output format (txt, json, or xml)    | txt |
| `--help`         | -     | Display help                         | - |
| `--version`      | -     | Display version                      | - |

## 📋 Output Formats

PMP supports three output formats, each designed for different use cases:

### Text Format (Default)
Human-readable, formatted text optimized for direct use with AI assistants. Includes project structure, file contents, and comprehensive statistics.

### JSON Format
Structured data format for programmatic processing and integration with other tools. Perfect for CI/CD pipelines and custom analysis tools.

```bash
pmp . --format json
```

### XML Format
Hierarchical format for integration with enterprise systems and XML-based tools.

```bash
pmp . --format xml
```

## 📊 Output Content

The generated prompt includes:

- Project information and statistics
- Detected technologies and frameworks
- Key files for understanding the project
- Complete file structure visualization
- Formatted file contents
- Token and character count estimates
- Code quality metrics and suggestions

## 🧠 Advanced Features

- **Binary Detection**: Automatically identifies and excludes binary files
- **Smart Token Estimation**: Accurate prediction of token usage for AI models
- **Technology Detection**: Identifies programming languages and frameworks
- **Code Complexity Analysis**: Flags potential maintenance issues
- **Intelligent Caching**: Improves performance with smart file content caching

## 🛠️ Building from Source

```bash
# Clone repository
git clone https://github.com/benoitpetit/prompt-my-project.git
cd prompt-my-project

# Install dependencies
go mod tidy

# Build
./scripts/build.sh

# Or build with go directly
go build -o pmp
```

## 🗑️ Uninstallation

### macOS & Linux
```bash
curl -fsSL https://raw.githubusercontent.com/benoitpetit/prompt-my-project/master/scripts/remove.sh | bash
```

### Windows
```powershell
irm https://raw.githubusercontent.com/benoitpetit/prompt-my-project/master/scripts/remove.ps1 | iex
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
