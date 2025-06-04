package main

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/jtarchie/agent/agent/tools"
	"github.com/jtarchie/outrageous/agent"
	"github.com/jtarchie/outrageous/client"
)

// Executor orchestrates the execution phase of the agent.
type Executor struct {
	cli       *CLI
	pwd       string
	promptsFS embed.FS
}

// NewExecutor creates a new Executor.
func NewExecutor(cli *CLI, pwd string, promptsFS embed.FS) *Executor {
	return &Executor{
		cli:       cli,
		pwd:       pwd,
		promptsFS: promptsFS,
	}
}

// Run executes the execution phase for a set of files.
func (e *Executor) Run(plan string, fileInfos []map[string]interface{}) error {
	// Load execution prompt template using the shared loadPromptTemplate function
	executeTmpl, err := loadPromptTemplate(e.promptsFS, "execute.md")
	if err != nil {
		return fmt.Errorf("failed to load execute prompt: %w", err)
	}

	var customPrompt []byte
	customPromptPath := filepath.Join(e.pwd, ".prompts", "execute.md")
	if _, err := os.Stat(customPromptPath); err == nil {
		customPrompt, err = os.ReadFile(customPromptPath)
		if err != nil {
			return fmt.Errorf("failed to read custom execute prompt: %w", err)
		}
	}

	toolsToInclude := tools.Select(e.pwd, e.cli.Tools)

	isBatchSingleFile := e.cli.Batch && len(fileInfos) == 1

	var currentFile interface{}
	if len(fileInfos) > 0 {
		currentFile = fileInfos[0]["filename"]
	} else {
		currentFile = "" // Explicitly set to empty string if no files
	}

	// Execute template for execution agent
	var executePromptBuf strings.Builder
	err = executeTmpl.Execute(&executePromptBuf, map[string]interface{}{
		"Plan":             plan,
		"Files":            fileInfos,
		"Tools":            toolsToInclude,
		"CustomPrompt":     string(customPrompt),
		"BatchMode":        isBatchSingleFile,
		"CurrentFile":      currentFile,
		"WorkingDirectory": e.pwd,
	})
	if err != nil {
		return fmt.Errorf("failed to execute execute prompt template: %w", err)
	}

	executingAgent := e.createExecutingAgent(executePromptBuf.String(), toolsToInclude)

	response, err := executingAgent.Run(
		context.Background(),
		agent.Messages{
			agent.Message{
				Role:    "user",
				Content: plan,
			},
		},
	)
	if err != nil {
		return fmt.Errorf("failed to run executing agent: %w", err)
	}

	slog.Debug("execution.agent", "response", response.Messages[len(response.Messages)-1].Content)
	return nil
}

// RunBatch executes the plan for each file individually.
func (e *Executor) RunBatch(plan string, allFileInfos []map[string]interface{}) error {
	slog.Info("batch.start", "plan", plan)

	if len(allFileInfos) == 0 {
		slog.Info("batch.iter", "working_directory", e.pwd, "index", 1, "total", 1)
		err := e.Run(plan, allFileInfos) // Use empty slice for fileInfos
		if err != nil {
			return fmt.Errorf("execution failed for current directory: %w", err)
		}
		slog.Info("completed processing current directory in batch mode")
		return nil
	}

	for i, fileInfo := range allFileInfos {
		singleFileInfo := []map[string]interface{}{fileInfo}
		fileName := fileInfo["filename"].(string)
		slog.Info("batch.iter", "file", fileName, "index", i+1, "total", len(allFileInfos))
		err := e.Run(plan, singleFileInfo)
		if err != nil {
			return fmt.Errorf("execution failed for file %s: %w", fileName, err)
		}
		slog.Info("batch.completed", "file", fileName)
	}

	slog.Info("batch.done", "total_files", len(allFileInfos))
	return nil
}

// createExecutingAgent creates and configures the executing agent.
func (e *Executor) createExecutingAgent(prompt string, toolsToUse []agent.Tool) *agent.Agent {
	executingAgent := agent.New(
		"Executing Agent",
		prompt,
		agent.WithClient(client.New(
			e.cli.ExecutingApiEndpoint,
			e.cli.ExecutingApiToken,
			e.cli.ExecutingModel,
		)),
	)

	toolNames := []string{}
	for _, tool := range toolsToUse {
		executingAgent.Tools.Add(tool)
		toolNames = append(toolNames, tool.Name)
	}

	slog.Debug("executing.agent", "prompt", prompt, "tools", toolNames, "batch_mode", e.cli.Batch)
	return executingAgent
}
