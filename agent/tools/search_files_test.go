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
		FileTypes: []string{".go"},
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
