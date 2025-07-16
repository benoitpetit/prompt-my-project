package utils

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"path/filepath"
	"strings"
)

// Directory represents a directory in the file system
type Directory struct {
	Name    string
	SubDirs map[string]*Directory
	Files   []string
}

// NewDirectory creates a new directory structure
func NewDirectory(name string) *Directory {
	return &Directory{
		Name:    name,
		SubDirs: make(map[string]*Directory),
		Files:   []string{},
	}
}

// BuildTree builds a directory tree from a list of files
func BuildTree(files []string, rootName string) *Directory {
	root := NewDirectory(rootName)

	for _, file := range files {
		AddFileToDirectory(root, file)
	}

	return root
}

// GetOrCreateDirectory gets or creates a directory at the specified path
func GetOrCreateDirectory(parent *Directory, name string) *Directory {
	if dir, ok := parent.SubDirs[name]; ok {
		return dir
	}

	dir := NewDirectory(name)
	parent.SubDirs[name] = dir
	return dir
}

// AddFileToDirectory adds a file to the directory structure
func AddFileToDirectory(root *Directory, filePath string) {
	dir := filepath.Dir(filePath)

	// If it's in the root directory
	if dir == "." || dir == "/" {
		root.Files = append(root.Files, filePath)
		return
	}

	// Split the directory path into components
	parts := strings.Split(dir, string(filepath.Separator))
	current := root

	// Navigate/create the directory structure
	for _, part := range parts {
		if part == "" {
			continue
		}
		current = GetOrCreateDirectory(current, part)
	}

	// Add the file to the final directory
	fileName := filepath.Base(filePath)
	current.Files = append(current.Files, fileName)
}

// GenerateTreeOutput generates a string representation of the directory tree
func GenerateTreeOutput(root *Directory) string {
	var builder strings.Builder
	printDirectory(root, &builder, "", true)
	return builder.String()
}

// printDirectory prints a directory and its contents to the builder
func printDirectory(dir *Directory, builder *strings.Builder, prefix string, isLast bool) {
	// Print current directory
	if dir.Name != "" {
		if isLast {
			fmt.Fprintf(builder, "%s└── %s/\n", prefix, dir.Name)
			prefix += "    "
		} else {
			fmt.Fprintf(builder, "%s├── %s/\n", prefix, dir.Name)
			prefix += "│   "
		}
	}

	// Sort files and directories for consistent output
	sortedFiles := make([]string, len(dir.Files))
	copy(sortedFiles, dir.Files)
	// strings.Sort(sortedFiles) // Would need to implement

	// Print files
	for i, file := range sortedFiles {
		isLastFile := i == len(sortedFiles)-1 && len(dir.SubDirs) == 0
		if isLastFile {
			fmt.Fprintf(builder, "%s└── %s\n", prefix, file)
		} else {
			fmt.Fprintf(builder, "%s├── %s\n", prefix, file)
		}
	}

	// Get sorted directory names
	dirNames := make([]string, 0, len(dir.SubDirs))
	for name := range dir.SubDirs {
		dirNames = append(dirNames, name)
	}
	// strings.Sort(dirNames) // Would need to implement

	// Print subdirectories
	for i, name := range dirNames {
		isLastDir := i == len(dirNames)-1
		printDirectory(dir.SubDirs[name], builder, prefix, isLastDir)
	}
}

// CountFilesRecursive counts the number of files in a directory and its subdirectories
func CountFilesRecursive(dir *Directory) int {
	count := len(dir.Files)

	for _, subdir := range dir.SubDirs {
		count += CountFilesRecursive(subdir)
	}

	return count
}

// GenerateDotOutput generates a DOT (Graphviz) representation of the directory tree
func GenerateDotOutput(root *Directory) string {
	var builder strings.Builder
	builder.WriteString("digraph G {\n")
	id := 0
	var walk func(dir *Directory, parentID int) int
	walk = func(dir *Directory, parentID int) int {
		myID := id
		id++
		dirLabel := fmt.Sprintf("dir_%d", myID)
		builder.WriteString(fmt.Sprintf("  %s [label=\"%s/\", shape=folder];\n", dirLabel, dir.Name))
		if parentID >= 0 {
			parentLabel := fmt.Sprintf("dir_%d", parentID)
			builder.WriteString(fmt.Sprintf("  %s -> %s;\n", parentLabel, dirLabel))
		}
		for _, file := range dir.Files {
			fileID := id
			id++
			fileLabel := fmt.Sprintf("file_%d", fileID)
			builder.WriteString(fmt.Sprintf("  %s [label=\"%s\", shape=note];\n", fileLabel, file))
			builder.WriteString(fmt.Sprintf("  %s -> %s;\n", dirLabel, fileLabel))
		}
		for _, sub := range dir.SubDirs {
			walk(sub, myID)
		}
		return myID
	}
	walk(root, -1)
	builder.WriteString("}\n")
	return builder.String()
}

// GenerateJSONTreeOutput generates a JSON representation of the directory tree
func GenerateJSONTreeOutput(root *Directory) string {
	data, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return "{}"
	}
	return string(data)
}

// GenerateXMLTreeOutput generates an XML representation of the directory tree
func GenerateXMLTreeOutput(root *Directory) string {
	type xmlDir Directory
	data, err := xml.MarshalIndent((*xmlDir)(root), "", "  ")
	if err != nil {
		return "<Directory/>"
	}
	return xml.Header + string(data)
}
