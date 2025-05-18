package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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

type RunInTerminal struct {
	Command     []string `json:"command" description:"Command with args to run in the terminal."`
	Explanation string   `json:"explanation" description:"Please provide a brief explanation of why this command needs to run."`
}

func (r RunInTerminal) Call(ctx context.Context) (any, error) {
	out, err := exec.CommandContext(ctx, r.Command[0], r.Command[1:]...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("error running command: %w\nOutput: %s", err, out)
	}

	return map[string]any{
		"status":      "completed",
		"output":      string(out),
		"explanation": r.Explanation,
	}, nil
}

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
