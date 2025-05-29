package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

type InsertEditIntoFile struct {
	Explanation string `json:"explanation" description:"A short explanation of the edit being made."`
	FilePath    string `json:"filePath" description:"An absolute path to the file to edit."`
	Content     string `json:"content" description:"The new content that will replace the entire file."`
}

func (i InsertEditIntoFile) Call(ctx context.Context) (any, error) {
	filePath, err := filepath.Abs(i.FilePath)
	if err != nil {
		return nil, fmt.Errorf("error getting absolute path for %s: %w", i.FilePath, err)
	}

	err = os.MkdirAll(filepath.Dir(filePath), 0755) // Ensure the directory exists
	if err != nil {
		return nil, fmt.Errorf("error creating directories for %s: %w", i.FilePath, err)
	}

	err = os.WriteFile(filePath, []byte(i.Content), 0644)
	if err != nil {
		return nil, fmt.Errorf("error writing to file %s: %w", i.FilePath, err)
	}

	return map[string]any{
		"status": "completed",
	}, nil
}
