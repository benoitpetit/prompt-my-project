package analyzer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/benoitpetit/prompt-my-project/pkg/binary"
	"github.com/benoitpetit/prompt-my-project/pkg/utils"
)

// Benchmark technology detection
func BenchmarkTechnologyDetection(b *testing.B) {
	// Create test files
	tmpDir := b.TempDir()
	testFiles := []string{
		"package.json",
		"go.mod",
		"requirements.txt",
		"Cargo.toml",
		"Gemfile",
		"composer.json",
		"main.go",
		"app.js",
		"index.tsx",
	}

	for _, file := range testFiles {
		path := filepath.Join(tmpDir, file)
		os.WriteFile(path, []byte("test content"), 0644)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector := NewTechnologyDetector(tmpDir, testFiles)
		detector.DetectAll()
	}
}

// Benchmark dependency analysis
func BenchmarkDependencyAnalysis(b *testing.B) {
	tmpDir := b.TempDir()

	// Create package.json
	packageJSON := `{
		"name": "test-project",
		"dependencies": {
			"react": "^18.0.0",
			"lodash": "^4.17.21"
		},
		"devDependencies": {
			"jest": "^29.0.0"
		}
	}`
	os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)

	// Create go.mod
	goMod := `module test-project

go 1.21

require (
	github.com/gin-gonic/gin v1.9.0
	github.com/stretchr/testify v1.8.4
)
`
	os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer := NewDependencyAnalyzer(tmpDir)
		analyzer.AnalyzeAll()
	}
}

// Benchmark code quality analysis
func BenchmarkCodeQualityAnalysis(b *testing.B) {
	tmpDir := b.TempDir()

	// Create test Go file
	goCode := `package main

import "fmt"

func complexFunction(a, b, c int) int {
	if a > 0 {
		if b > 0 {
			if c > 0 {
				return a + b + c
			} else {
				return a + b
			}
		} else {
			return a
		}
	}
	return 0
}

func main() {
	fmt.Println(complexFunction(1, 2, 3))
}
`
	goFile := filepath.Join(tmpDir, "main.go")
	os.WriteFile(goFile, []byte(goCode), 0644)

	files := []string{"main.go"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer := NewCodeQualityAnalyzer(tmpDir, files)
		analyzer.Analyze()
	}
}

// Benchmark vulnerability checking
func BenchmarkVulnerabilityChecking(b *testing.B) {
	tmpDir := b.TempDir()

	// Create package.json with known vulnerable dependencies
	packageJSON := `{
		"dependencies": {
			"lodash": "4.17.15",
			"axios": "0.19.0",
			"jquery": "2.1.4"
		}
	}`
	os.WriteFile(filepath.Join(tmpDir, "package.json"), []byte(packageJSON), 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checker := NewVulnerabilityChecker(tmpDir)
		analyzer := NewDependencyAnalyzer(tmpDir)
		analysis, _ := analyzer.AnalyzeAll()

		if analysis != nil {
			checker.CheckDependencies(analysis)
		}
	}
}

// Benchmark linter integration
func BenchmarkLinterIntegration(b *testing.B) {
	tmpDir := b.TempDir()

	// Create test files
	jsCode := `console.log("test");
var x = 10;
debugger;
`
	jsFile := filepath.Join(tmpDir, "test.js")
	os.WriteFile(jsFile, []byte(jsCode), 0644)

	goCode := `package main

// func oldFunction() {}

func main() {
	println("test")
}
`
	goFile := filepath.Join(tmpDir, "test.go")
	os.WriteFile(goFile, []byte(goCode), 0644)

	files := []string{"test.js", "test.go"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		linter := NewLinterIntegration(tmpDir, files)
		linter.RunAllLinters()
	}
}

// Benchmark streaming processor
func BenchmarkStreamingProcessor(b *testing.B) {
	tmpDir := b.TempDir()

	// Create a large file
	largeContent := make([]byte, 50*1024*1024) // 50 MB
	for i := range largeContent {
		largeContent[i] = byte('a' + (i % 26))
	}

	largeFile := filepath.Join(tmpDir, "large.txt")
	os.WriteFile(largeFile, largeContent, 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		processor := NewStreamProcessor(tmpDir, DefaultChunkSize)
		processor.ProcessFile("large.txt")
	}
}

// Benchmark file collection
func BenchmarkFileCollection(b *testing.B) {
	tmpDir := b.TempDir()

	// Create multiple files
	for i := 0; i < 100; i++ {
		dir := filepath.Join(tmpDir, "dir"+string(rune(i)))
		os.MkdirAll(dir, 0755)

		for j := 0; j < 10; j++ {
			file := filepath.Join(dir, "file"+string(rune(j))+".go")
			os.WriteFile(file, []byte("package main\n\nfunc main() {}\n"), 0644)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer := New(
			tmpDir,
			[]string{"**/*.go"},
			[]string{"node_modules/**"},
			0,
			10*1024*1024,
			1000,
			100*1024*1024,
			4,
		)
		analyzer.CollectFiles()
	}
}

// Benchmark project structure generation
func BenchmarkProjectStructure(b *testing.B) {
	tmpDir := b.TempDir()

	// Create directory structure
	dirs := []string{
		"src/components",
		"src/utils",
		"src/services",
		"tests/unit",
		"tests/integration",
	}

	for _, dir := range dirs {
		os.MkdirAll(filepath.Join(tmpDir, dir), 0755)

		// Add some files
		for i := 0; i < 5; i++ {
			file := filepath.Join(tmpDir, dir, "file"+string(rune(i))+".js")
			os.WriteFile(file, []byte("// test"), 0644)
		}
	}

	analyzer := New(
		tmpDir,
		[]string{"**/*"},
		[]string{},
		0,
		10*1024*1024,
		1000,
		100*1024*1024,
		4,
	)
	analyzer.CollectFiles()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer.GenerateProjectStructure()
	}
}

// Benchmark full analysis pipeline
func BenchmarkFullAnalysisPipeline(b *testing.B) {
	tmpDir := b.TempDir()

	// Create a realistic project structure
	files := map[string]string{
		"package.json": `{
			"name": "test-app",
			"dependencies": {
				"react": "^18.0.0",
				"express": "^4.18.0"
			}
		}`,
		"src/index.js": `
			import React from 'react';
			console.log("Starting app");
			export default function App() {
				return <div>Hello World</div>;
			}
		`,
		"src/utils.js": `
			export function helper() {
				var x = 10;
				return x * 2;
			}
		`,
		"README.md":  `# Test Project\n\nThis is a test.`,
		".gitignore": `node_modules\ndist\n`,
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		os.MkdirAll(filepath.Dir(fullPath), 0755)
		os.WriteFile(fullPath, []byte(content), 0644)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		analyzer := New(
			tmpDir,
			[]string{"**/*"},
			[]string{"node_modules/**"},
			0,
			10*1024*1024,
			1000,
			100*1024*1024,
			4,
		)

		analyzer.CollectFiles()

		// Run all analysis
		detector := NewTechnologyDetector(tmpDir, analyzer.Files)
		detector.DetectAll()

		depAnalyzer := NewDependencyAnalyzer(tmpDir)
		depAnalyzer.AnalyzeAll()

		qualityAnalyzer := NewCodeQualityAnalyzer(tmpDir, analyzer.Files)
		qualityAnalyzer.Analyze()

		linter := NewLinterIntegration(tmpDir, analyzer.Files)
		linter.RunAllLinters()
	}
}

// Benchmark sorting algorithms
func BenchmarkQuickSortStrings(b *testing.B) {
	// Create test data
	data := make([]string, 1000)
	for i := range data {
		data[i] = "file_" + string(rune(1000-i)) + ".go"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testData := make([]string, len(data))
		copy(testData, data)
		utils.SortStrings(testData)
	}
}

func BenchmarkHeapSortStrings(b *testing.B) {
	// Create test data
	data := make([]string, 1000)
	for i := range data {
		data[i] = "file_" + string(rune(1000-i)) + ".go"
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testData := make([]string, len(data))
		copy(testData, data)
		utils.SortStringsHeap(testData)
	}
}

// Benchmark token estimation
func BenchmarkTokenEstimation(b *testing.B) {
	tmpDir := b.TempDir()

	// Create test file with various content
	content := `package main

import (
	"fmt"
	"os"
	"time"
)

// Main function that does complex operations
func main() {
	data := make([]string, 100)
	for i := 0; i < 100; i++ {
		data[i] = fmt.Sprintf("item_%d", i)
	}
	
	processData(data)
}

func processData(items []string) {
	for _, item := range items {
		fmt.Println(item)
		time.Sleep(10 * time.Millisecond)
	}
}
`
	testFile := filepath.Join(tmpDir, "test.go")
	os.WriteFile(testFile, []byte(content), 0644)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate token estimation
		_ = len(content) / 4 // Rough estimate: 4 chars per token
	}
}

// Benchmark binary detection
func BenchmarkBinaryDetection(b *testing.B) {
	tmpDir := b.TempDir()

	// Create text file
	textFile := filepath.Join(tmpDir, "text.txt")
	os.WriteFile(textFile, []byte("This is plain text content"), 0644)

	// Create binary file
	binaryFile := filepath.Join(tmpDir, "binary.bin")
	binaryData := make([]byte, 1024)
	for i := range binaryData {
		binaryData[i] = byte(i % 256)
	}
	os.WriteFile(binaryFile, binaryData, 0644)

	cache := binary.NewCache()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		binary.IsBinaryFile(textFile, cache)
		binary.IsBinaryFile(binaryFile, cache)
	}
}

// Table-driven benchmarks for different project sizes
func BenchmarkProjectSizes(b *testing.B) {
	sizes := []struct {
		name  string
		files int
		dirs  int
	}{
		{"Small", 10, 2},
		{"Medium", 100, 10},
		{"Large", 1000, 50},
	}

	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			tmpDir := b.TempDir()

			// Create structure
			for i := 0; i < size.dirs; i++ {
				dir := filepath.Join(tmpDir, "dir"+string(rune(i)))
				os.MkdirAll(dir, 0755)

				filesPerDir := size.files / size.dirs
				for j := 0; j < filesPerDir; j++ {
					file := filepath.Join(dir, "file"+string(rune(j))+".go")
					os.WriteFile(file, []byte("package main\nfunc main() {}\n"), 0644)
				}
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				analyzer := New(
					tmpDir,
					[]string{"**/*.go"},
					[]string{},
					0,
					10*1024*1024,
					10000,
					1024*1024*1024,
					4,
				)
				analyzer.CollectFiles()
			}
		})
	}
}

// Benchmark memory usage for different operations
func BenchmarkMemoryUsage(b *testing.B) {
	b.Run("SmallFiles", func(b *testing.B) {
		tmpDir := b.TempDir()
		content := []byte("small content")

		for i := 0; i < 100; i++ {
			file := filepath.Join(tmpDir, "file"+string(rune(i))+".txt")
			os.WriteFile(file, content, 0644)
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			files, _ := os.ReadDir(tmpDir)
			for _, f := range files {
				path := filepath.Join(tmpDir, f.Name())
				os.ReadFile(path)
			}
		}
	})

	b.Run("LargeFiles", func(b *testing.B) {
		tmpDir := b.TempDir()
		content := make([]byte, 1024*1024) // 1 MB

		file := filepath.Join(tmpDir, "large.bin")
		os.WriteFile(file, content, 0644)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			os.ReadFile(file)
		}
	})
}

// Helper function to benchmark concurrent operations
func BenchmarkConcurrentAnalysis(b *testing.B) {
	tmpDir := b.TempDir()

	// Create files
	for i := 0; i < 50; i++ {
		file := filepath.Join(tmpDir, "file"+string(rune(i))+".go")
		os.WriteFile(file, []byte("package main\nfunc main() {}\n"), 0644)
	}

	workers := []int{1, 2, 4, 8}

	for _, w := range workers {
		b.Run("Workers"+string(rune(w+'0')), func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				analyzer := New(
					tmpDir,
					[]string{"**/*.go"},
					[]string{},
					0,
					10*1024*1024,
					1000,
					100*1024*1024,
					w,
				)
				analyzer.CollectFiles()
			}
		})
	}
}

// Benchmark comparison: old vs new sorting
func BenchmarkSortingComparison(b *testing.B) {
	data := make([]string, 1000)
	for i := range data {
		data[i] = "file_" + string(rune(1000-i)) + ".go"
	}

	b.Run("Old Bubble Sort", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			testData := make([]string, len(data))
			copy(testData, data)
			// Simulate old O(n²) bubble sort
			for j := 0; j < len(testData); j++ {
				for k := j + 1; k < len(testData); k++ {
					if testData[j] > testData[k] {
						testData[j], testData[k] = testData[k], testData[j]
					}
				}
			}
		}
	})

	b.Run("New QuickSort", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			testData := make([]string, len(data))
			copy(testData, data)
			utils.SortStrings(testData)
		}
	})
}
