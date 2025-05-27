package tools

import (
	"github.com/iancoleman/strcase"
	"github.com/jtarchie/outrageous/agent"
	"github.com/samber/lo"
)

// availableTools defines all possible tools with their descriptions and implementations
var availableTools = []agent.Tool{
	agent.MustWrapStruct(
		"Read specific lines from a file in the codebase. Use this tool when you know the file path and want to inspect only a section of the file to avoid loading large files in full. This is useful for reviewing implementations, extracting function or class definitions, or confirming assumptions about code structure.",
		ReadFile{},
	),
	agent.MustWrapStruct(
		"Run a command in the terminal. Use this tool when you need to execute a command that is not directly related to the codebase, such as running tests, building the project, or executing scripts.",
		RunInTerminal{},
	),
	agent.MustWrapStruct(
		"Insert or edit a file in the codebase. Use this tool when you need to apply changes to a file based on the provided unified diff. This is useful for making code modifications, applying patches, or updating configurations.",
		InsertEditIntoFile{},
	),
	agent.MustWrapStruct(
		"Search for text content across files in a directory. Performs case-insensitive search and returns the first occurrence found in each matching file along with metadata like line number, file size, and modification time. Supports file type filtering and uses efficient goroutines for concurrent processing.",
		SearchFiles{},
	),
}

// selectTools determines which tools to include based on CLI input
func Select(requestedTools []string) []agent.Tool {
	// If no specific tools requested, include all available tools
	if len(requestedTools) == 0 {
		return availableTools
	}

	requestedTools = lo.Map(requestedTools, func(s string, _ int) string {
		return strcase.ToSnake(s)
	})

	// Otherwise, include only the specified tools
	toolsToInclude := []agent.Tool{}

	for _, tool := range availableTools {
		if lo.Contains(requestedTools, tool.Name) {
			toolsToInclude = append(toolsToInclude, tool)
		}
	}

	return toolsToInclude
}
