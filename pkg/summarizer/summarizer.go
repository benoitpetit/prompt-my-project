package summarizer

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"strings"
)

// Summary represents a code summary with important structure information
type Summary struct {
	Path      string
	Language  string
	Exports   []Export
	Imports   []string
	DocString string
}

// Export represents an exported symbol (function, type, interface, etc.)
type Export struct {
	Type      string // "function", "type", "interface", "const", "var"
	Name      string
	Signature string
	DocString string
}

// Summarizer handles code summarization for different languages
type Summarizer struct{}

// NewSummarizer creates a new summarizer
func NewSummarizer() *Summarizer {
	return &Summarizer{}
}

// SummarizeFile generates a summary for a given file
func (s *Summarizer) SummarizeFile(path string, content string) (*Summary, error) {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".go":
		return s.summarizeGo(path, content)
	case ".js", ".jsx", ".ts", ".tsx":
		return s.summarizeJavaScript(path, content)
	case ".py":
		return s.summarizePython(path, content)
	case ".java":
		return s.summarizeJava(path, content)
	case ".c", ".h", ".cpp", ".hpp", ".cc", ".cxx":
		return s.summarizeC(path, content)
	default:
		// For unsupported languages, return a basic summary
		return &Summary{
			Path:      path,
			Language:  ext,
			DocString: fmt.Sprintf("File: %s (unsupported for detailed summarization)", filepath.Base(path)),
		}, nil
	}
}

// summarizeGo uses Go's AST parser to extract structure
func (s *Summarizer) summarizeGo(path string, content string) (*Summary, error) {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, path, content, parser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Go file: %w", err)
	}

	summary := &Summary{
		Path:     path,
		Language: "go",
		Imports:  []string{},
		Exports:  []Export{},
	}

	// Extract package doc
	if node.Doc != nil {
		summary.DocString = node.Doc.Text()
	}

	// Extract imports
	for _, imp := range node.Imports {
		importPath := strings.Trim(imp.Path.Value, `"`)
		summary.Imports = append(summary.Imports, importPath)
	}

	// Extract top-level declarations
	for _, decl := range node.Decls {
		switch d := decl.(type) {
		case *ast.FuncDecl:
			// Only export if it's exported (starts with uppercase)
			if d.Name.IsExported() {
				export := Export{
					Type: "function",
					Name: d.Name.Name,
				}

				if d.Doc != nil {
					export.DocString = d.Doc.Text()
				}

				// Build signature
				receiver := ""
				if d.Recv != nil && len(d.Recv.List) > 0 {
					receiver = formatFieldList(d.Recv) + " "
				}
				params := formatFieldList(d.Type.Params)
				results := ""
				if d.Type.Results != nil {
					results = " " + formatFieldList(d.Type.Results)
				}
				export.Signature = fmt.Sprintf("func %s%s(%s)%s", receiver, d.Name.Name, params, results)

				summary.Exports = append(summary.Exports, export)
			}

		case *ast.GenDecl:
			for _, spec := range d.Specs {
				switch s := spec.(type) {
				case *ast.TypeSpec:
					if s.Name.IsExported() {
						export := Export{
							Name: s.Name.Name,
						}

						if d.Doc != nil {
							export.DocString = d.Doc.Text()
						}

						// Determine type
						switch s.Type.(type) {
						case *ast.InterfaceType:
							export.Type = "interface"
							export.Signature = fmt.Sprintf("type %s interface", s.Name.Name)
						case *ast.StructType:
							export.Type = "type"
							export.Signature = fmt.Sprintf("type %s struct", s.Name.Name)
						default:
							export.Type = "type"
							export.Signature = fmt.Sprintf("type %s", s.Name.Name)
						}

						summary.Exports = append(summary.Exports, export)
					}

				case *ast.ValueSpec:
					for _, name := range s.Names {
						if name.IsExported() {
							export := Export{
								Name: name.Name,
							}

							if d.Doc != nil {
								export.DocString = d.Doc.Text()
							}

							if d.Tok == token.CONST {
								export.Type = "const"
								export.Signature = fmt.Sprintf("const %s", name.Name)
							} else {
								export.Type = "var"
								export.Signature = fmt.Sprintf("var %s", name.Name)
							}

							summary.Exports = append(summary.Exports, export)
						}
					}
				}
			}
		}
	}

	return summary, nil
}

// formatFieldList formats a field list (parameters, returns, etc.)
func formatFieldList(fields *ast.FieldList) string {
	if fields == nil || len(fields.List) == 0 {
		return ""
	}

	var parts []string
	for _, field := range fields.List {
		typeStr := formatExpr(field.Type)
		if len(field.Names) == 0 {
			parts = append(parts, typeStr)
		} else {
			for _, name := range field.Names {
				parts = append(parts, fmt.Sprintf("%s %s", name.Name, typeStr))
			}
		}
	}

	return strings.Join(parts, ", ")
}

// formatExpr formats an expression to a string
func formatExpr(expr ast.Expr) string {
	switch e := expr.(type) {
	case *ast.Ident:
		return e.Name
	case *ast.StarExpr:
		return "*" + formatExpr(e.X)
	case *ast.ArrayType:
		return "[]" + formatExpr(e.Elt)
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", formatExpr(e.Key), formatExpr(e.Value))
	case *ast.SelectorExpr:
		return formatExpr(e.X) + "." + e.Sel.Name
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.FuncType:
		return "func"
	default:
		return "..."
	}
}

// summarizeJavaScript extracts structure from JavaScript/TypeScript using regex (simplified)
func (s *Summarizer) summarizeJavaScript(path string, content string) (*Summary, error) {
	summary := &Summary{
		Path:     path,
		Language: "javascript",
		Imports:  []string{},
		Exports:  []Export{},
	}

	// Extract JSDoc at the top of the file
	docRegex := regexp.MustCompile(`(?s)^/\*\*.*?\*/`)
	if match := docRegex.FindString(content); match != "" {
		summary.DocString = strings.TrimSpace(match)
	}

	// Extract imports
	importRegex := regexp.MustCompile(`import\s+.*?\s+from\s+['"](.+?)['"]`)
	for _, match := range importRegex.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			summary.Imports = append(summary.Imports, match[1])
		}
	}

	// Extract exported functions
	funcRegex := regexp.MustCompile(`export\s+(?:async\s+)?function\s+(\w+)\s*\((.*?)\)`)
	for _, match := range funcRegex.FindAllStringSubmatch(content, -1) {
		if len(match) > 2 {
			export := Export{
				Type:      "function",
				Name:      match[1],
				Signature: fmt.Sprintf("function %s(%s)", match[1], match[2]),
			}
			summary.Exports = append(summary.Exports, export)
		}
	}

	// Extract exported classes
	classRegex := regexp.MustCompile(`export\s+class\s+(\w+)`)
	for _, match := range classRegex.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			export := Export{
				Type:      "class",
				Name:      match[1],
				Signature: fmt.Sprintf("class %s", match[1]),
			}
			summary.Exports = append(summary.Exports, export)
		}
	}

	// Extract exported constants
	constRegex := regexp.MustCompile(`export\s+const\s+(\w+)`)
	for _, match := range constRegex.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			export := Export{
				Type:      "const",
				Name:      match[1],
				Signature: fmt.Sprintf("const %s", match[1]),
			}
			summary.Exports = append(summary.Exports, export)
		}
	}

	return summary, nil
}

// summarizePython extracts structure from Python using regex
func (s *Summarizer) summarizePython(path string, content string) (*Summary, error) {
	summary := &Summary{
		Path:     path,
		Language: "python",
		Imports:  []string{},
		Exports:  []Export{},
	}

	lines := strings.Split(content, "\n")

	// Extract module docstring (first thing in file)
	if len(lines) > 0 {
		docRegex := regexp.MustCompile(`^"""(.*?)"""`)
		if match := docRegex.FindStringSubmatch(strings.Join(lines[0:10], "\n")); len(match) > 1 {
			summary.DocString = strings.TrimSpace(match[1])
		}
	}

	// Extract imports
	importRegex := regexp.MustCompile(`^(?:from\s+(\S+)\s+)?import\s+(.+)`)
	for _, line := range lines {
		if match := importRegex.FindStringSubmatch(strings.TrimSpace(line)); match != nil {
			if match[1] != "" {
				summary.Imports = append(summary.Imports, match[1])
			}
		}
	}

	// Extract functions (only top-level, not indented)
	funcRegex := regexp.MustCompile(`^def\s+(\w+)\s*\((.*?)\)`)
	for _, line := range lines {
		if match := funcRegex.FindStringSubmatch(line); match != nil {
			export := Export{
				Type:      "function",
				Name:      match[1],
				Signature: fmt.Sprintf("def %s(%s)", match[1], match[2]),
			}
			summary.Exports = append(summary.Exports, export)
		}
	}

	// Extract classes (only top-level)
	classRegex := regexp.MustCompile(`^class\s+(\w+)`)
	for _, line := range lines {
		if match := classRegex.FindStringSubmatch(line); match != nil {
			export := Export{
				Type:      "class",
				Name:      match[1],
				Signature: fmt.Sprintf("class %s", match[1]),
			}
			summary.Exports = append(summary.Exports, export)
		}
	}

	return summary, nil
}

// summarizeJava extracts structure from Java using regex
func (s *Summarizer) summarizeJava(path string, content string) (*Summary, error) {
	summary := &Summary{
		Path:     path,
		Language: "java",
		Imports:  []string{},
		Exports:  []Export{},
	}

	// Extract imports
	importRegex := regexp.MustCompile(`import\s+(.+?);`)
	for _, match := range importRegex.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			summary.Imports = append(summary.Imports, match[1])
		}
	}

	// Extract public classes
	classRegex := regexp.MustCompile(`public\s+class\s+(\w+)`)
	for _, match := range classRegex.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			export := Export{
				Type:      "class",
				Name:      match[1],
				Signature: fmt.Sprintf("public class %s", match[1]),
			}
			summary.Exports = append(summary.Exports, export)
		}
	}

	// Extract public interfaces
	interfaceRegex := regexp.MustCompile(`public\s+interface\s+(\w+)`)
	for _, match := range interfaceRegex.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			export := Export{
				Type:      "interface",
				Name:      match[1],
				Signature: fmt.Sprintf("public interface %s", match[1]),
			}
			summary.Exports = append(summary.Exports, export)
		}
	}

	// Extract public methods
	methodRegex := regexp.MustCompile(`public\s+(?:static\s+)?(\w+)\s+(\w+)\s*\((.*?)\)`)
	for _, match := range methodRegex.FindAllStringSubmatch(content, -1) {
		if len(match) > 3 {
			export := Export{
				Type:      "method",
				Name:      match[2],
				Signature: fmt.Sprintf("public %s %s(%s)", match[1], match[2], match[3]),
			}
			summary.Exports = append(summary.Exports, export)
		}
	}

	return summary, nil
}

// summarizeC extracts structure from C/C++ using regex
func (s *Summarizer) summarizeC(path string, content string) (*Summary, error) {
	summary := &Summary{
		Path:     path,
		Language: "c/c++",
		Imports:  []string{},
		Exports:  []Export{},
	}

	// Extract includes
	includeRegex := regexp.MustCompile(`#include\s+[<"](.+?)[>"]`)
	for _, match := range includeRegex.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			summary.Imports = append(summary.Imports, match[1])
		}
	}

	// Extract function declarations/definitions (simplified)
	funcRegex := regexp.MustCompile(`(?m)^[\w\s\*]+\s+(\w+)\s*\((.*?)\)\s*[;{]`)
	for _, match := range funcRegex.FindAllStringSubmatch(content, -1) {
		if len(match) > 2 {
			// Skip common keywords that might match
			if match[1] == "if" || match[1] == "while" || match[1] == "for" || match[1] == "switch" {
				continue
			}
			export := Export{
				Type:      "function",
				Name:      match[1],
				Signature: fmt.Sprintf("%s(%s)", match[1], match[2]),
			}
			summary.Exports = append(summary.Exports, export)
		}
	}

	// Extract struct definitions
	structRegex := regexp.MustCompile(`(?:typedef\s+)?struct\s+(\w+)`)
	for _, match := range structRegex.FindAllStringSubmatch(content, -1) {
		if len(match) > 1 {
			export := Export{
				Type:      "struct",
				Name:      match[1],
				Signature: fmt.Sprintf("struct %s", match[1]),
			}
			summary.Exports = append(summary.Exports, export)
		}
	}

	return summary, nil
}

// FormatSummary formats a summary to a readable string
func FormatSummary(summary *Summary) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n\n", summary.Path))

	if summary.DocString != "" {
		sb.WriteString("## Documentation\n")
		sb.WriteString(summary.DocString)
		sb.WriteString("\n\n")
	}

	if len(summary.Imports) > 0 {
		sb.WriteString("## Imports\n")
		for _, imp := range summary.Imports {
			sb.WriteString(fmt.Sprintf("- %s\n", imp))
		}
		sb.WriteString("\n")
	}

	if len(summary.Exports) > 0 {
		sb.WriteString("## Exports\n")
		for _, exp := range summary.Exports {
			sb.WriteString(fmt.Sprintf("### %s: %s\n", exp.Type, exp.Name))
			sb.WriteString(fmt.Sprintf("```\n%s\n```\n", exp.Signature))
			if exp.DocString != "" {
				sb.WriteString(exp.DocString)
				sb.WriteString("\n")
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
