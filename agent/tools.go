package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/jtarchie/outrageous/agent"
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

// Tool represents an available tool for the executing agent
type Tool struct {
	Name           string
	Description    string
	Implementation agent.Caller
}

// availableTools defines all possible tools with their descriptions and implementations
var availableTools = []Tool{
	{
		Name:           "ReadFile",
		Description:    "Read specific lines from a file in the codebase. Use this tool when you know the file path and want to inspect only a section of the file to avoid loading large files in full. This is useful for reviewing implementations, extracting function or class definitions, or confirming assumptions about code structure.",
		Implementation: ReadFile{},
	},
	{
		Name:           "RunInTerminal",
		Description:    "Run a command in the terminal. Use this tool when you need to execute a command that is not directly related to the codebase, such as running tests, building the project, or executing scripts.",
		Implementation: RunInTerminal{},
	},
	{
		Name:           "InsertEditIntoFile",
		Description:    "Insert or edit a file in the codebase. Use this tool when you need to apply changes to a file based on the provided unified diff. This is useful for making code modifications, applying patches, or updating configurations.",
		Implementation: InsertEditIntoFile{},
	},
}

// selectTools determines which tools to include based on CLI input
func selectTools(requestedTools []string) []Tool {
	// If no specific tools requested, include all available tools
	if len(requestedTools) == 0 {
		return availableTools
	}

	// Otherwise, include only the specified tools
	toolsToInclude := make([]Tool, 0)
	toolNames := make(map[string]bool)

	for _, name := range requestedTools {
		toolNames[name] = true
	}

	for _, tool := range availableTools {
		if toolNames[tool.Name] {
			toolsToInclude = append(toolsToInclude, tool)
		}
	}

	return toolsToInclude
}

// createToolDescriptions creates a list of tool descriptions for templates
func createToolDescriptions(tools []Tool) []map[string]string {
	descriptions := make([]map[string]string, 0, len(tools))

	for _, tool := range tools {
		descriptions = append(descriptions, map[string]string{
			"name":        tool.Name,
			"description": tool.Description,
		})
	}

	return descriptions
}
