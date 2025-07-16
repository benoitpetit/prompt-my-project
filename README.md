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


<p align="center">Prompt My Project (PMP) is a command-line tool to generate structured prompts<br>and dependency graphs from source code, optimized for AI assistants.</p>

## Installation

You can install PMP with go install:

```
go install github.com/benoitpetit/prompt-my-project@latest
```

Or with one of the installation scripts:

```
curl -fsSL https://raw.githubusercontent.com/benoitpetit/prompt-my-project/master/scripts/install.sh | bash
```

For more information, see https://github.com/benoitpetit/prompt-my-project

## Usage

### Show help

```
pmp
```
Shows the help message and available subcommands.

---

### Generate a project prompt

```
pmp prompt /path/to/project [options]
```

Generates a prompt for the specified project, with options for filtering files and output format.

- By default, the prompt file is saved in `/path/to/project/pmp_output/` with a timestamped filename (e.g. `prompt_YYYYMMDD_HHMMSS.txt`).
- The `pmp_output` directory is automatically created and added to the project's `.gitignore` if not present.

#### Options for `prompt`

| Option                | Description                                                      | Default                |
|-----------------------|------------------------------------------------------------------|------------------------|
| `--exclude, -e`       | Exclude files matching these patterns (e.g., *.md, src/)         | *(NONE)*               |
| `--include, -i`       | Include only files matching these patterns                       | *(ALL)*               |
| `--min-size`          | Minimum file size (e.g., 1KB, 500B)                              | `1KB`                  |
| `--max-size`          | Maximum file size (e.g., 100MB, 1GB)                             | `100MB`                |
| `--no-gitignore`      | Ignore .gitignore file                                           | `false`                |
| `--output, -o`        | Output directory for the prompt file                             | `<project>/pmp_output/`   |
| `--workers`           | Number of parallel workers                                       | Number of CPUs         |
| `--max-files`         | Maximum number of files to process (0 = unlimited)               | `500`                  |
| `--max-total-size`    | Maximum total size of all files (e.g., 10MB, 0 = unlimited)      | `10MB`                 |
| `--format, -f`        | Output format (`txt`, `json`, `xml`, or `stdout[:txt|json|xml]`) | `txt`                  |

#### Example
```
pmp prompt ./myproject --include "*.go" --exclude "test/*" --format json
# Output: ./myproject/pmp_output/prompt_YYYYMMDD_HHMMSS.json
```

---

### Generate a project dependency graph

```
pmp graph /path/to/project --format <dot|json|xml|txt|stdout[:dot|json|xml|txt]>
```

Generates a dependency tree (arborescence) of the project in the specified format.

- By default, the graph file is saved in `/path/to/project/pmp_output/` with a timestamped filename (e.g. `graph_YYYYMMDD_HHMMSS.dot`).
- The `pmp_output` directory is automatically created and added to the project's `.gitignore` if not present.

#### Options for `graph`
- `--format, -f`    Output format for the graph (`dot`, `json`, `xml`, `txt`, or `stdout[:dot|json|xml|txt]`). Default: `dot`.
- `--output, -o`    Output directory or file for the graph (default: `<project>/pmp_output/`)

#### Example
```
pmp graph ./myproject --format dot
# Output: ./myproject/pmp_output/graph_YYYYMMDD_HHMMSS.dot

pmp graph ./myproject --format json
# Output: ./myproject/pmp_output/graph_YYYYMMDD_HHMMSS.json

pmp graph ./myproject --format txt
# Output: ./myproject/pmp_output/graph_YYYYMMDD_HHMMSS.txt
```

- `dot`: Outputs a Graphviz DOT file representing the directory and file structure.
- `json`: Outputs the tree as JSON.
- `xml`: Outputs the tree as XML.
- `txt`: Outputs a human-readable tree (like the Unix `tree` command).
- `stdout[:format]`: Outputs the result directly to stdout, for piping to other tools. Example: `--format stdout:json`.

---

## Machine-Readable Output & Piping

You can output directly to stdout for use with other tools:

- **Prompt as JSON**
  ```sh
  pmp prompt . --format stdout:json | jq .
  ```
- **Prompt as XML**
  ```sh
  pmp prompt . --format stdout:xml | xmllint --format -
  ```
- **Graph as JSON**
  ```sh
  pmp graph . --format stdout:json | jq .
  ```
- **Graph as DOT (for Graphviz)**
  ```sh
  pmp graph . --format stdout | dot -Tpng > graph.png
  ```

---

## Output Files & Directory

- All generated files (prompts and graphs) are saved in a `pmp_output` directory inside your project by default.
- This directory is automatically added to your project's `.gitignore` to avoid committing large or generated files.
- You can override the output location with the `--output` option.

## Supported Languages

PMP supports dependency and structure analysis for projects in Go, JavaScript, TypeScript, Python, Java, Ruby, PHP, C#, C, C++, HTML, CSS, JSON, XML, Markdown, Shell, Batch, SQL, and more.

## Example

```
pmp prompt ./myproject --format txt
# Output: ./myproject/pmp_output/prompt_YYYYMMDD_HHMMSS.txt

pmp graph ./myproject --format dot
# Output: ./myproject/pmp_output/graph_YYYYMMDD_HHMMSS.dot
```

---

## Example: Dependency Graph for This Project

Below is an example of a dependency graph generated for this project using the `pmp graph . --format dot` command:

```
digraph G {
  dir_0 [label="prompt-my-project/", shape=folder];
  file_1 [label=".gitignore", shape=note];
  dir_0 -> file_1;
  file_2 [label="LICENSE", shape=note];
  dir_0 -> file_2;
  file_3 [label="README.md", shape=note];
  dir_0 -> file_3;
  file_4 [label="go.sum", shape=note];
  dir_0 -> file_4;
  file_5 [label="main.go", shape=note];
  dir_0 -> file_5;
  dir_6 [label="pkg/", shape=folder];
  dir_0 -> dir_6;
  dir_7 [label="binary/", shape=folder];
  dir_6 -> dir_7;
  file_8 [label="cache.go", shape=note];
  dir_7 -> file_8;
  file_9 [label="detector.go", shape=note];
  dir_7 -> file_9;
  dir_10 [label="formatter/", shape=folder];
  dir_6 -> dir_10;
  file_11 [label="formatter.go", shape=note];
  dir_10 -> file_11;
  dir_12 [label="utils/", shape=folder];
  dir_6 -> dir_12;
  file_13 [label="directory.go", shape=note];
  dir_12 -> file_13;
  file_14 [label="token_estimator.go", shape=note];
  dir_12 -> file_14;
  dir_15 [label="worker/", shape=folder];
  dir_6 -> dir_15;
  file_16 [label="worker.go", shape=note];
  dir_15 -> file_16;
  dir_17 [label="analyzer/", shape=folder];
  dir_6 -> dir_17;
  file_18 [label="analyzer.go", shape=note];
  dir_17 -> file_18;
}
```

Visual representation:

<img src="screen_graph.png" alt="Dependency Graph Screenshot" width="600"/>

## Shell Autocompletion

PMP supports shell autocompletion for Bash, Zsh, Fish, and PowerShell.

To enable autocompletion, run the following command for your shell:

> **Note:** If you use `./pmp` (relative path) instead of `pmp` in your PATH, the installer configures autocompletion for both forms. You may need to open a new terminal or re-source your completion scripts for the change to take effect.

> **Tip:** If you want to use autocompletion for `./pmp` (without installing it globally), you can enable it for your current session with:
```sh
source <(./pmp completion bash)
complete -o default -F __start_pmp ./pmp
```


### Bash
```sh
source <(pmp completion bash)
# To enable for all sessions:
# Linux:
pmp completion bash > /etc/bash_completion.d/pmp
# macOS:
pmp completion bash > /usr/local/etc/bash_completion.d/pmp
```

### Zsh
```sh
echo 'autoload -U compinit; compinit' >> ~/.zshrc
pmp completion zsh > "${fpath[1]}/_pmp"
```

### Fish
```sh
pmp completion fish | source
pmp completion fish > ~/.config/fish/completions/pmp.fish
```

### PowerShell
```powershell
pmp completion powershell | Out-String | Invoke-Expression
# To enable for all sessions, add the above to your $PROFILE
```

## License

MIT License
