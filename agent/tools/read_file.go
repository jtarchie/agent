package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ReadFile struct {
	FilePath            string `json:"filePath" description:"Path to the file to read."`
	StartLineNumberZero int    `json:"startLineNumberBaseZero" description:"Start line number (0-based) to read from the file."`
	EndLineNumberZero   int    `json:"endLineNumberBaseZero" description:"End line number (0-based) to read from the file. If not specified, reads until the end of the file."`

	RootPath string `json:"-"`
}

func (r ReadFile) Call(ctx context.Context) (any, error) {
	filePath, err := filepath.Abs(r.FilePath)
	if err != nil {
		return nil, fmt.Errorf("error getting absolute path for %s: %w", r.FilePath, err)
	}

	// Security check - ensure file is inside rootPath if provided
	if r.RootPath != "" {
		rootPath, err := filepath.Abs(r.RootPath)
		if err != nil {
			return nil, fmt.Errorf("error getting absolute path for rootPath %s: %w", r.RootPath, err)
		}

		// Make sure paths have trailing slashes for proper prefix checking
		rootPathWithSlash := ensureTrailingSlash(rootPath)
		filePathWithDir := ensureTrailingSlash(filepath.Dir(filePath))

		if !strings.HasPrefix(filePathWithDir, rootPathWithSlash) && !strings.HasPrefix(filePath, rootPath) {
			return nil, fmt.Errorf("security error: cannot read %s outside of root path %s", filePath, rootPath)
		}
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	lines := strings.Split(string(data), "\n")

	if r.StartLineNumberZero >= len(lines) {
		return nil, fmt.Errorf("start line out of range")
	}

	end := r.EndLineNumberZero + 1
	if end > len(lines) {
		end = len(lines)
	}

	return strings.Join(lines[r.StartLineNumberZero:end], "\n"), nil
}
