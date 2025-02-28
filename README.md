# Prompt My Project (PMP)

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
        <a href="https://github.com/benoitpetit/prompt-my-project/releases">
        <img src="https://img.shields.io/badge/Version-1.0.0-blue.svg" alt="Version">
    </a>
    <a href="https://golang.org/dl/">
        <img src="https://img.shields.io/badge/Go-%3E%3D%201.21-blue.svg" alt="Go Version">
    </a>
</div>

## ✨ Features

<div align="center">
    <table>
        <tr>
            <td align="center">📂</td>
            <td><strong>Smart Navigation</strong><br/>Recursively scans your project structure</td>
            <td align="center">🔍</td>
            <td><strong>Binary Detection</strong><br/>Intelligently identifies and avoids binary files</td>
        </tr>
        <tr>
            <td align="center">🎯</td>
            <td><strong>Pattern Matching</strong><br/>Supports advanced inclusion/exclusion patterns</td>
            <td align="center">⚡</td>
            <td><strong>Git-Aware</strong><br/>Respects your project's .gitignore rules</td>
        </tr>
        <tr>
            <td align="center">📊</td>
            <td><strong>Size Control</strong><br/>Flexible file size filtering options</td>
            <td align="center">🚀</td>
            <td><strong>High Performance</strong><br/>Concurrent processing with worker pools</td>
        </tr>
        <tr>
            <td align="center">💾</td>
            <td><strong>Smart Caching</strong><br/>Optimized file content caching</td>
            <td align="center">📝</td>
            <td><strong>Detailed Statistics</strong><br/>Comprehensive metrics about your project</td>
        </tr>
    </table>
</div>

## 📂 Output Organization

PMP generates a well-structured prompt file that includes:

- Project information and statistics
- Complete file structure visualization
- Formatted file contents
- Token and character count estimates

Prompts are automatically saved in:

- `./prompts/` (default, automatically added to .gitignore)
- Or in the folder specified by `--output`

Files are named using a timestamp format: `prompt_YYYYMMDD_HHMMSS.txt`

## 🎯 Build Artifacts

Format: `pmp_<version>_<os>_<arch>.<ext>`
Example: `pmp_v1.0.0_linux_amd64.tar.gz`

### Supported Architectures

- amd64 (x86_64)
- arm64 (aarch64)

### Supported Systems

- Linux
- macOS (Darwin)
- Windows

## 🔧 Default Configuration

| Parameter  | Value     | Description                  |
| ---------- | --------- | ---------------------------- |
| Min Size   | 1KB       | Minimum file size            |
| Max Size   | 100MB     | Maximum file size            |
| Output Dir | ./prompts | Output directory for prompts |
| GitIgnore  | true      | Respect .gitignore rules     |

## 🚀 Installation

### macOS & Linux

```bash
curl -fsSL https://raw.githubusercontent.com/benoitpetit/prompt-my-project/refs/heads/master/scripts/install.sh | bash
```

### Windows

```powershell
irm https://raw.githubusercontent.com/benoitpetit/prompt-my-project/refs/heads/master/scripts/install.ps1 | iex
```

## 🛠️ Usage

### Basic Syntax

```bash
pmp [options] [path]
```

### Available Options

| Option           | Description                          |
| ---------------- | ------------------------------------ |
| `--include, -i`  | Include only files matching patterns |
| `--exclude, -e`  | Exclude files matching patterns      |
| `--min-size`     | Minimum file size (default: "1KB")   |
| `--max-size`     | Maximum file size (default: "100MB") |
| `--no-gitignore` | Ignore .gitignore file               |
| `--output, -o`   | Output folder for prompt file        |
| `--help`         | Display help                         |
| `--version`      | Display version                      |

### Quick Examples

```bash
# Analyze current project
pmp

# Filter by extensions
pmp --include "*.go" --include "*.md"

# Exclude directories
pmp --exclude "test/*" --exclude "vendor/*"

# Specify output directory
pmp --output ./prompts
```

## 🚄 Performance

PMP uses an advanced concurrent processing system to optimize performance:

- **Worker Pool**: Parallel file processing with a worker pool
- **Optimized Memory**: Use of reusable buffers
- **Smart Caching**: File content caching to avoid repeated reads
- **Adaptive Concurrency**: Number of workers adapted to available system resources

## 🔧 Building from source

### Prerequisites

- Go 1.21 or higher
- Git

### Build Steps

```bash
# Clone repository
git clone https://github.com/benoitpetit/prompt-my-project.git
cd prompt-my-project

# Install dependencies
go mod tidy

# Build
./scripts/build.sh
```

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
