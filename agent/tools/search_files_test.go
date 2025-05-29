package tools_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jtarchie/agent/agent/tools"
	. "github.com/onsi/gomega"
)

func TestSearchFiles(t *testing.T) {
	assert := NewGomegaWithT(t)

	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "search_test")
	assert.Expect(err).NotTo(HaveOccurred())
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create test files with various content
	testFiles := map[string]string{
		"file1.txt": "This is a test file\nWith SOME content\nAnd more lines",
		"file2.go":  "package main\n// This is a Go file\nfunc main() {\n\tfmt.Println(\"Hello World\")\n}",
		"file3.md":  "# Documentation\nThis contains **important** information\nAnd examples",
		"file4.txt": "No matching content here\nJust random text\nNothing special",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(tmpDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		assert.Expect(err).NotTo(HaveOccurred())
	}

	// Test case-insensitive search
	searcher := tools.SearchFiles{
		Query:     "SOME",
		Directory: tmpDir,
	}

	result, err := searcher.Call(context.Background())
	assert.Expect(err).NotTo(HaveOccurred())

	response, ok := result.(tools.SearchResponse)
	assert.Expect(ok).To(BeTrue())
	assert.Expect(response.TotalFiles).To(Equal(4))
	assert.Expect(response.FilesMatched).To(Equal(1))
	assert.Expect(len(response.Results)).To(Equal(1))

	// Check the result details - should find "SOME" in file1.txt
	searchResult := response.Results[0]
	assert.Expect(searchResult.FilePath).To(ContainSubstring("file1.txt"))
	assert.Expect(searchResult.LineNumber).To(Equal(2))
	assert.Expect(searchResult.LineContent).To(Equal("With SOME content"))
	assert.Expect(searchResult.FoundAt).To(Equal(5)) // Position of "SOME" in the line
}

func TestSearchFilesWithFileTypeFilter(t *testing.T) {
	assert := NewGomegaWithT(t)

	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "search_test")
	assert.Expect(err).NotTo(HaveOccurred())
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create test files with various extensions
	testFiles := map[string]string{
		"test1.go":  "package main\nfunc Test() {}",
		"test2.txt": "package main\nsome text content",
		"test3.go":  "package utils\nfunc Helper() {}",
		"test4.md":  "# Title\npackage description",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(tmpDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		assert.Expect(err).NotTo(HaveOccurred())
	}

	// Search only in .go files
	searcher := tools.SearchFiles{
		Query:     "package",
		Directory: tmpDir,
		Files:     []string{"*.go"}, // Filter to only .go files
	}

	result, err := searcher.Call(context.Background())
	assert.Expect(err).NotTo(HaveOccurred())

	response, ok := result.(tools.SearchResponse)
	assert.Expect(ok).To(BeTrue())
	assert.Expect(response.TotalFiles).To(Equal(2)) // Only .go files
	assert.Expect(response.FilesMatched).To(Equal(2))
	assert.Expect(len(response.Results)).To(Equal(2))

	// Verify all results are from .go files
	for _, result := range response.Results {
		assert.Expect(result.FilePath).To(HaveSuffix(".go"))
		assert.Expect(result.LineContent).To(ContainSubstring("package"))
	}
}

func TestSearchFilesWithGlobFilter(t *testing.T) {
	assert := NewGomegaWithT(t)

	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "search_test")
	assert.Expect(err).NotTo(HaveOccurred())
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create test files with various extensions
	testFiles := map[string]string{
		"test1.go":  "package main\nfunc Test() {}",
		"test2.txt": "package main\nsome text content",
		"test3.go":  "package utils\nfunc Helper() {}",
		"test4.md":  "# Title\npackage description",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(tmpDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		assert.Expect(err).NotTo(HaveOccurred())
	}

	// Search only in .go files using glob pattern
	searcher := tools.SearchFiles{
		Query:     "package",
		Directory: tmpDir,
		Files:     []string{"*.go"},
	}

	result, err := searcher.Call(context.Background())
	assert.Expect(err).NotTo(HaveOccurred())

	response, ok := result.(tools.SearchResponse)
	assert.Expect(ok).To(BeTrue())
	assert.Expect(response.TotalFiles).To(Equal(2)) // Only .go files
	assert.Expect(response.FilesMatched).To(Equal(2))
	assert.Expect(len(response.Results)).To(Equal(2))

	// Verify all results are from .go files
	for _, result := range response.Results {
		assert.Expect(result.FilePath).To(HaveSuffix(".go"))
		assert.Expect(result.LineContent).To(ContainSubstring("package"))
	}
}

func TestSearchFilesMultipleMatches(t *testing.T) {
	assert := NewGomegaWithT(t)

	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "search_test")
	assert.Expect(err).NotTo(HaveOccurred())
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create files where multiple files contain the search term
	testFiles := map[string]string{
		"file1.txt": "error occurred here\nsecond line",
		"file2.txt": "no match in this file\njust regular content",
		"file3.txt": "first line\nERROR in uppercase\nthird line",
		"file4.txt": "another file with error\nmore content",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(tmpDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		assert.Expect(err).NotTo(HaveOccurred())
	}

	searcher := tools.SearchFiles{
		Query:     "error",
		Directory: tmpDir,
	}

	result, err := searcher.Call(context.Background())
	assert.Expect(err).NotTo(HaveOccurred())

	response, ok := result.(tools.SearchResponse)
	assert.Expect(ok).To(BeTrue())
	assert.Expect(response.TotalFiles).To(Equal(4))
	assert.Expect(response.FilesMatched).To(Equal(3)) // files 1, 3, and 4 have matches
	assert.Expect(len(response.Results)).To(Equal(3))

	// Verify we only get first occurrence per file
	fileResults := make(map[string]tools.SearchResult)
	for _, result := range response.Results {
		baseFile := filepath.Base(result.FilePath)
		fileResults[baseFile] = result
	}

	// Check specific results
	assert.Expect(fileResults["file1.txt"].LineNumber).To(Equal(1))
	assert.Expect(fileResults["file1.txt"].LineContent).To(Equal("error occurred here"))

	assert.Expect(fileResults["file3.txt"].LineNumber).To(Equal(2))
	assert.Expect(fileResults["file3.txt"].LineContent).To(Equal("ERROR in uppercase"))

	assert.Expect(fileResults["file4.txt"].LineNumber).To(Equal(1))
	assert.Expect(fileResults["file4.txt"].LineContent).To(Equal("another file with error"))
}

func TestSearchFilesEmptyQuery(t *testing.T) {
	assert := NewGomegaWithT(t)

	searcher := tools.SearchFiles{
		Query:     "",
		Directory: ".",
	}

	_, err := searcher.Call(context.Background())
	assert.Expect(err).To(HaveOccurred())
	assert.Expect(err.Error()).To(ContainSubstring("query cannot be empty"))
}

func TestSearchFilesNonExistentDirectory(t *testing.T) {
	assert := NewGomegaWithT(t)

	searcher := tools.SearchFiles{
		Query:     "test",
		Directory: "/non/existent/directory",
	}

	_, err := searcher.Call(context.Background())
	assert.Expect(err).To(HaveOccurred())
	assert.Expect(err.Error()).To(ContainSubstring("error getting files to search"))
}

func TestSearchFilesDefaultDirectory(t *testing.T) {
	assert := NewGomegaWithT(t)

	// Test with empty directory (should default to current directory)
	searcher := tools.SearchFiles{
		Query: "package",
		// Directory not set, should default to "."
	}

	result, err := searcher.Call(context.Background())
	assert.Expect(err).NotTo(HaveOccurred())

	response, ok := result.(tools.SearchResponse)
	assert.Expect(ok).To(BeTrue())
	assert.Expect(response.TotalFiles).To(BeNumerically(">", 0))
}

func TestSearchFilesMetadata(t *testing.T) {
	assert := NewGomegaWithT(t)

	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "search_test")
	assert.Expect(err).NotTo(HaveOccurred())
	defer func() { _ = os.RemoveAll(tmpDir) }()

	content := "test content for metadata check"
	filePath := filepath.Join(tmpDir, "metadata_test.txt")
	err = os.WriteFile(filePath, []byte(content), 0644)
	assert.Expect(err).NotTo(HaveOccurred())

	searcher := tools.SearchFiles{
		Query:     "metadata",
		Directory: tmpDir,
	}

	result, err := searcher.Call(context.Background())
	assert.Expect(err).NotTo(HaveOccurred())

	response, ok := result.(tools.SearchResponse)
	assert.Expect(ok).To(BeTrue())
	assert.Expect(len(response.Results)).To(Equal(1))

	searchResult := response.Results[0]
	assert.Expect(searchResult.FileSize).To(BeNumerically(">", 0))
	assert.Expect(searchResult.ModifiedTime).NotTo(BeEmpty())
	assert.Expect(response.Duration).NotTo(BeEmpty())
}

func TestSearchFilesWithRootPathSecurity(t *testing.T) {
	assert := NewGomegaWithT(t)

	// Create a temporary directory as root
	tmpDir, err := os.MkdirTemp("", "rootdir")
	assert.Expect(err).NotTo(HaveOccurred())
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a file inside the root directory
	testFile := filepath.Join(tmpDir, "testfile.txt")
	err = os.WriteFile(testFile, []byte("This is a test file with content"), 0644)
	assert.Expect(err).NotTo(HaveOccurred())

	// Test searching within root path - should succeed
	searcher := tools.SearchFiles{
		Query:     "test",
		Directory: tmpDir,
		RootPath:  tmpDir,
	}

	result, err := searcher.Call(context.Background())
	assert.Expect(err).NotTo(HaveOccurred())

	response, ok := result.(tools.SearchResponse)
	assert.Expect(ok).To(BeTrue())
	assert.Expect(response.TotalFiles).To(Equal(1))
	assert.Expect(response.FilesMatched).To(Equal(1))

	// Test searching outside root path - should fail
	searcher = tools.SearchFiles{
		Query:     "test",
		Directory: "/tmp",
		RootPath:  tmpDir,
	}

	_, err = searcher.Call(context.Background())
	assert.Expect(err).To(HaveOccurred())
	assert.Expect(err.Error()).To(ContainSubstring("security error"))
}

func TestSearchFilesPathTraversalAttack(t *testing.T) {
	assert := NewGomegaWithT(t)

	// Create a temporary directory as root
	tmpDir, err := os.MkdirTemp("", "rootdir")
	assert.Expect(err).NotTo(HaveOccurred())
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Try to search outside using path traversal
	searcher := tools.SearchFiles{
		Query:     "test",
		Directory: filepath.Join(tmpDir, "../../../etc"),
		RootPath:  tmpDir,
	}

	_, err = searcher.Call(context.Background())
	assert.Expect(err).To(HaveOccurred())
	assert.Expect(err.Error()).To(ContainSubstring("security error"))
}

func TestSearchFilesWithSpecificFilesAndGlobs(t *testing.T) {
	assert := NewGomegaWithT(t)

	// Create a temporary directory structure for testing
	tmpDir, err := os.MkdirTemp("", "search_glob_test")
	assert.Expect(err).NotTo(HaveOccurred())
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create subdirectories
	srcDir := filepath.Join(tmpDir, "src")
	testsDir := filepath.Join(tmpDir, "tests")
	docsDir := filepath.Join(tmpDir, "docs")
	err = os.MkdirAll(srcDir, 0755)
	assert.Expect(err).NotTo(HaveOccurred())
	err = os.MkdirAll(testsDir, 0755)
	assert.Expect(err).NotTo(HaveOccurred())
	err = os.MkdirAll(docsDir, 0755)
	assert.Expect(err).NotTo(HaveOccurred())

	// Create test files with search content
	testFiles := map[string]string{
		"main.go":             "package main\n// TODO: implement feature\nfunc main() {}",
		"config.json":         `{"name": "test", "TODO": "update config"}`,
		"src/utils.go":        "package utils\n// TODO: add validation\nfunc Helper() {}",
		"src/handler.go":      "package handlers\nfunc Handle() {}\n// No todos here",
		"tests/main_test.go":  "package main\n// TODO: write more tests\nfunc TestMain() {}",
		"tests/utils_test.go": "package utils\nfunc TestUtils() {}\n// Complete test",
		"docs/README.md":      "# Project\nTODO: write documentation\n## Features",
		"docs/API.md":         "# API\n## Endpoints\nAll documented",
	}

	for filename, content := range testFiles {
		filePath := filepath.Join(tmpDir, filename)
		err := os.WriteFile(filePath, []byte(content), 0644)
		assert.Expect(err).NotTo(HaveOccurred())
	}

	t.Run("SearchSpecificFiles", func(t *testing.T) {
		// Test searching specific files only
		searcher := tools.SearchFiles{
			Query:     "TODO",
			Directory: tmpDir,
			Files:     []string{"main.go", "config.json"},
		}

		result, err := searcher.Call(context.Background())
		assert.Expect(err).NotTo(HaveOccurred())

		response, ok := result.(tools.SearchResponse)
		assert.Expect(ok).To(BeTrue())
		assert.Expect(response.TotalFiles).To(Equal(2))
		assert.Expect(response.FilesMatched).To(Equal(2))
		assert.Expect(len(response.Results)).To(Equal(2))

		// Verify only specified files were searched
		foundFiles := make(map[string]bool)
		for _, result := range response.Results {
			fileName := filepath.Base(result.FilePath)
			foundFiles[fileName] = true
			assert.Expect(result.LineContent).To(ContainSubstring("TODO"))
		}
		assert.Expect(foundFiles["main.go"]).To(BeTrue())
		assert.Expect(foundFiles["config.json"]).To(BeTrue())
	})

	t.Run("SearchWithSimpleGlob", func(t *testing.T) {
		// Test simple glob pattern for Go files
		searcher := tools.SearchFiles{
			Query:     "TODO",
			Directory: tmpDir,
			Files:     []string{"*.go"},
		}

		result, err := searcher.Call(context.Background())
		assert.Expect(err).NotTo(HaveOccurred())

		response, ok := result.(tools.SearchResponse)
		assert.Expect(ok).To(BeTrue())
		assert.Expect(response.TotalFiles).To(Equal(1)) // Only main.go in root
		assert.Expect(response.FilesMatched).To(Equal(1))
		assert.Expect(len(response.Results)).To(Equal(1))

		fileName := filepath.Base(response.Results[0].FilePath)
		assert.Expect(fileName).To(Equal("main.go"))
	})

	t.Run("SearchWithDoubleStarGlob", func(t *testing.T) {
		// Test doublestar pattern for all Go files recursively
		searcher := tools.SearchFiles{
			Query:     "// TODO",
			Directory: tmpDir,
			Files:     []string{"**/*.go"},
		}

		result, err := searcher.Call(context.Background())
		assert.Expect(err).NotTo(HaveOccurred())

		response, ok := result.(tools.SearchResponse)
		assert.Expect(ok).To(BeTrue())
		assert.Expect(response.TotalFiles).To(Equal(5))   // main.go, src/utils.go, tests/main_test.go, tests/utils_test.go
		assert.Expect(response.FilesMatched).To(Equal(3)) // Only 3 have TODO comments

		// Verify expected files were found
		foundFiles := make(map[string]bool)
		for _, result := range response.Results {
			fileName := filepath.Base(result.FilePath)
			foundFiles[fileName] = true
		}
		assert.Expect(foundFiles["main.go"]).To(BeTrue())
		assert.Expect(foundFiles["utils.go"]).To(BeTrue())
		assert.Expect(foundFiles["main_test.go"]).To(BeTrue())
		assert.Expect(foundFiles["utils_test.go"]).To(BeFalse()) // No TODO in this file
	})

	t.Run("SearchWithMixedPatternsAndFiles", func(t *testing.T) {
		// Test combination of specific files and glob patterns
		searcher := tools.SearchFiles{
			Query:     "TODO",
			Directory: tmpDir,
			Files:     []string{"config.json", "src/*.go", "docs/**/*.md"},
		}

		result, err := searcher.Call(context.Background())
		assert.Expect(err).NotTo(HaveOccurred())

		response, ok := result.(tools.SearchResponse)
		assert.Expect(ok).To(BeTrue())
		assert.Expect(response.TotalFiles).To(Equal(5))   // config.json, src/utils.go, src/handler.go, docs/README.md, docs/API.md
		assert.Expect(response.FilesMatched).To(Equal(4)) // Only 3 have TODO

		// Check that we got results from different file types
		foundExtensions := make(map[string]bool)
		for _, result := range response.Results {
			ext := filepath.Ext(result.FilePath)
			foundExtensions[ext] = true
		}
		assert.Expect(foundExtensions[".json"]).To(BeTrue())
		assert.Expect(foundExtensions[".go"]).To(BeTrue())
		assert.Expect(foundExtensions[".md"]).To(BeTrue())
	})

	t.Run("SearchWithNonExistentFile", func(t *testing.T) {
		// Test with a mix of existing and non-existent files
		searcher := tools.SearchFiles{
			Query:     "TODO",
			Directory: tmpDir,
			Files:     []string{"main.go", "nonexistent.txt", "src/utils.go"},
		}

		result, err := searcher.Call(context.Background())
		assert.Expect(err).NotTo(HaveOccurred())

		response, ok := result.(tools.SearchResponse)
		assert.Expect(ok).To(BeTrue())
		assert.Expect(response.TotalFiles).To(Equal(2)) // Only existing files
		assert.Expect(response.FilesMatched).To(Equal(2))
	})

	t.Run("SearchWithAbsolutePaths", func(t *testing.T) {
		// Test with absolute file paths
		mainGoPath := filepath.Join(tmpDir, "main.go")
		utilsGoPath := filepath.Join(tmpDir, "src", "utils.go")

		searcher := tools.SearchFiles{
			Query:     "TODO",
			Directory: tmpDir,
			Files:     []string{mainGoPath, utilsGoPath},
		}

		result, err := searcher.Call(context.Background())
		assert.Expect(err).NotTo(HaveOccurred())

		response, ok := result.(tools.SearchResponse)
		assert.Expect(ok).To(BeTrue())
		assert.Expect(response.TotalFiles).To(Equal(2))
		assert.Expect(response.FilesMatched).To(Equal(2))
	})

	t.Run("SearchWithMultipleFileTypeGlobs", func(t *testing.T) {
		// Test that multiple glob patterns work for different file types
		searcher := tools.SearchFiles{
			Query:     "package",
			Directory: tmpDir,
			Files:     []string{"**/*.go", "**/*.json"}, // Go and JSON files
		}

		result, err := searcher.Call(context.Background())
		assert.Expect(err).NotTo(HaveOccurred())

		response, ok := result.(tools.SearchResponse)
		assert.Expect(ok).To(BeTrue())
		assert.Expect(response.TotalFiles).To(Equal(6))   // 4 .go files + 1 .json file
		assert.Expect(response.FilesMatched).To(Equal(5)) // 4 .go files + 1 .json file

		// Verify results contain both .go and .json files
		foundExtensions := make(map[string]bool)
		for _, result := range response.Results {
			ext := filepath.Ext(result.FilePath)
			foundExtensions[ext] = true
		}
		assert.Expect(foundExtensions[".go"]).To(BeTrue())
		assert.Expect(foundExtensions[".json"]).To(BeFalse())
	})
}

func TestSearchFilesWithRootPathFileFiltering(t *testing.T) {
	assert := NewGomegaWithT(t)

	// Create a temporary directory as root
	tmpDir, err := os.MkdirTemp("", "rootdir")
	assert.Expect(err).NotTo(HaveOccurred())
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create an outside directory
	outsideDir, err := os.MkdirTemp("", "outsidedir")
	assert.Expect(err).NotTo(HaveOccurred())
	defer func() { _ = os.RemoveAll(outsideDir) }()

	// Create files inside root directory
	insideFile := filepath.Join(tmpDir, "inside.txt")
	err = os.WriteFile(insideFile, []byte("content inside root"), 0644)
	assert.Expect(err).NotTo(HaveOccurred())

	// Create file outside root directory
	outsideFile := filepath.Join(outsideDir, "outside.txt")
	err = os.WriteFile(outsideFile, []byte("content outside root"), 0644)
	assert.Expect(err).NotTo(HaveOccurred())

	// Test with specific files that include outside file - should filter it out
	searcher := tools.SearchFiles{
		Query:     "content",
		Directory: tmpDir,
		Files:     []string{insideFile, outsideFile},
		RootPath:  tmpDir,
	}

	result, err := searcher.Call(context.Background())
	assert.Expect(err).NotTo(HaveOccurred())

	response, ok := result.(tools.SearchResponse)
	assert.Expect(ok).To(BeTrue())
	assert.Expect(response.TotalFiles).To(Equal(1)) // Only the inside file should be processed
	assert.Expect(response.FilesMatched).To(Equal(1))
	assert.Expect(len(response.Results)).To(Equal(1))

	// Verify only the inside file was found
	assert.Expect(response.Results[0].FilePath).To(Equal(insideFile))
	assert.Expect(response.Results[0].LineContent).To(ContainSubstring("inside"))
}
