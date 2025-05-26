package tools

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// SearchFiles represents a tool for searching through files in a directory
type SearchFiles struct {
	Query     string   `json:"query" description:"The search term to look for in files."`
	Directory string   `json:"directory" description:"The directory to search in. Defaults to current directory if not specified."`
	FileTypes []string `json:"fileTypes,omitempty" description:"Optional file extensions to filter by (e.g., ['.go', '.txt']). If not specified, searches all files."`
}

// SearchResult represents the result of a search operation
type SearchResult struct {
	FilePath     string `json:"filePath"`
	LineNumber   int    `json:"lineNumber"`
	LineContent  string `json:"lineContent"`
	FoundAt      int    `json:"foundAt"` // Position in the line where the match was found
	FileSize     int64  `json:"fileSize"`
	ModifiedTime string `json:"modifiedTime"`
}

// SearchResponse represents the complete response from a search operation
type SearchResponse struct {
	Results      []SearchResult `json:"results"`
	TotalFiles   int            `json:"totalFiles"`
	FilesMatched int            `json:"filesMatched"`
	Duration     string         `json:"duration"`
}

func (s SearchFiles) Call(ctx context.Context) (any, error) {
	startTime := time.Now()

	if s.Query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	directory := s.Directory
	if directory == "" {
		directory = "."
	}

	// Convert query to lowercase for case-insensitive search
	queryLower := strings.ToLower(s.Query)

	// Get all files to search
	files, err := s.getFilesToSearch(directory)
	if err != nil {
		return nil, fmt.Errorf("error getting files to search: %w", err)
	}

	if len(files) == 0 {
		return SearchResponse{
			Results:      []SearchResult{},
			TotalFiles:   0,
			FilesMatched: 0,
			Duration:     time.Since(startTime).String(),
		}, nil
	}

	// Limit goroutines to number of CPUs
	numWorkers := runtime.NumCPU()
	if numWorkers > len(files) {
		numWorkers = len(files)
	}

	// Create channels for work distribution
	filesChan := make(chan string, len(files))
	resultsChan := make(chan SearchResult, len(files))

	// Send all files to the channel
	for _, file := range files {
		filesChan <- file
	}
	close(filesChan)

	// Start worker goroutines
	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.searchWorker(ctx, queryLower, filesChan, resultsChan)
		}()
	}

	// Close results channel when all workers are done
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	var results []SearchResult
	filesWithMatches := make(map[string]bool)

	for result := range resultsChan {
		results = append(results, result)
		filesWithMatches[result.FilePath] = true
	}

	return SearchResponse{
		Results:      results,
		TotalFiles:   len(files),
		FilesMatched: len(filesWithMatches),
		Duration:     time.Since(startTime).String(),
	}, nil
}

// getFilesToSearch returns a list of files to search based on directory and file type filters
func (s SearchFiles) getFilesToSearch(directory string) ([]string, error) {
	// Check if directory exists
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory does not exist: %s", directory)
	}

	var files []string

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err // Return errors to be handled by caller
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Skip hidden files and directories
		if strings.HasPrefix(filepath.Base(path), ".") {
			return nil
		}

		// Apply file type filter if specified
		if len(s.FileTypes) > 0 {
			ext := filepath.Ext(path)
			matched := false
			for _, allowedExt := range s.FileTypes {
				if strings.EqualFold(ext, allowedExt) {
					matched = true
					break
				}
			}
			if !matched {
				return nil
			}
		}

		files = append(files, path)
		return nil
	})

	return files, err
}

// searchWorker processes files from the channel and searches for the query
func (s SearchFiles) searchWorker(ctx context.Context, queryLower string, filesChan <-chan string, resultsChan chan<- SearchResult) {
	for filePath := range filesChan {
		select {
		case <-ctx.Done():
			return
		default:
			s.searchInFile(filePath, queryLower, resultsChan)
		}
	}
}

// searchInFile searches for the query in a single file, reading line by line
func (s SearchFiles) searchInFile(filePath, queryLower string, resultsChan chan<- SearchResult) {
	file, err := os.Open(filePath)
	if err != nil {
		return // Skip files that can't be opened
	}
	defer func() { _ = file.Close() }()

	// Get file info for metadata
	fileInfo, err := file.Stat()
	if err != nil {
		return
	}

	scanner := bufio.NewScanner(file)
	lineNumber := 1

	// Search line by line until we find the first occurrence
	for scanner.Scan() {
		line := scanner.Text()
		lineLower := strings.ToLower(line)

		if foundAt := strings.Index(lineLower, queryLower); foundAt != -1 {
			// Found the query in this line, send result and return
			result := SearchResult{
				FilePath:     filePath,
				LineNumber:   lineNumber,
				LineContent:  line,
				FoundAt:      foundAt,
				FileSize:     fileInfo.Size(),
				ModifiedTime: fileInfo.ModTime().Format(time.RFC3339),
			}
			resultsChan <- result
			return // Only care about first occurrence per file
		}

		lineNumber++
	}
}
