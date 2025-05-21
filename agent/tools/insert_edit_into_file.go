package tools

import (
	"context"
	"fmt"
	"os"
)

type InsertEditIntoFile struct {
	Explanation string `json:"explanation" description:"A short explanation of the edit being made."`
	FilePath    string `json:"filePath" description:"An absolute path to the file to edit."`
	Content     string `json:"content" description:"The new content that will replace the entire file."`
}

func (i InsertEditIntoFile) Call(ctx context.Context) (any, error) {
	err := os.WriteFile(i.FilePath, []byte(i.Content), 0644)
	if err != nil {
		return nil, fmt.Errorf("error writing to file %s: %w", i.FilePath, err)
	}

	return map[string]any{
		"status":      "completed",
		"explanation": i.Explanation,
		"filePath":    i.FilePath,
	}, nil
}
