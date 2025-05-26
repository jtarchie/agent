package tools

import (
	"github.com/jtarchie/outrageous/agent"
)

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
	{
		Name:           "SearchFiles",
		Description:    "Search for text content across files in a directory. Performs case-insensitive search and returns the first occurrence found in each matching file along with metadata like line number, file size, and modification time. Supports file type filtering and uses efficient goroutines for concurrent processing.",
		Implementation: SearchFiles{},
	},
}

// selectTools determines which tools to include based on CLI input
func Select(requestedTools []string) []Tool {
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
