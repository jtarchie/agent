package tools

import (
	"context"
	"fmt"
	"os"
	"strings"
)

type ReadFile struct {
	FilePath            string `json:"filePath" description:"Path to the file to read."`
	StartLineNumberZero int    `json:"startLineNumberBaseZero" description:"Start line number (0-based) to read from the file."`
	EndLineNumberZero   int    `json:"endLineNumberBaseZero" description:"End line number (0-based) to read from the file. If not specified, reads until the end of the file."`
}

func (r ReadFile) Call(ctx context.Context) (any, error) {
	data, err := os.ReadFile(r.FilePath)
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
