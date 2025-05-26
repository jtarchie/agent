package main

import (
	"context"
	"embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/alecthomas/kong"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/go-enry/go-enry/v2"
	"github.com/jtarchie/agent/agent/tools"
	"github.com/jtarchie/outrageous/agent"
	"github.com/jtarchie/outrageous/client"
)

//go:embed prompts
var promptsFS embed.FS

func main() {
	// Set up logging
	setupLogging()

	// Parse CLI arguments
	cli := &CLI{}
	ctx := kong.Parse(cli)

	// Run the command
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}

func setupLogging() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))
}

// CLI defines the command-line interface structure
type CLI struct {
	Patterns []string `arg:"" optional:"" help:"List of file patterns (globs) or filenames to process. Supports doublestar (**) patterns. If empty, works from current directory."`
	Message  string   `help:"Message to send to the planning agent." required:""`
	Batch    bool     `help:"Enable batch mode for the executing agent." default:"false"`

	Tools []string `help:"List of tools to allow the executing agent to use. Default is all." optional:""`

	PlanningApiToken    string `help:"API token for OpenAI compatible endpoint"`
	PlanningApiEndpoint string `help:"API endpoint for OpenAI compatible endpoint" default:"http://localhost:11434/v1"`
	PlanningModel       string `help:"Model to use for the planning agent." default:"phi4-mini-reasoning:latest"`

	ExecutingApiToken    string `help:"API token for OpenAI compatible endpoint"`
	ExecutingApiEndpoint string `help:"API endpoint for OpenAI compatible endpoint" default:"http://localhost:11434/v1"`
	ExecutingModel       string `help:"Model to use for the executing agent." default:"qwen3:8b"`
}

// FileInfo represents information about a file in the codebase
type FileInfo struct {
	Filename string
	Language string
	Size     int
}

// Run executes the main CLI workflow
func (cli *CLI) Run() error {
	// Get current working directory
	pwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Process patterns to get actual files
	filenames, err := expandPatterns(cli.Patterns, pwd)
	if err != nil {
		return err
	}

	// Process files
	fileInfos, err := processFiles(filenames, pwd)
	if err != nil {
		return err
	}

	// Create and run the planning agent (once for all files)
	plan, err := runPlanningPhase(cli, pwd, fileInfos)
	if err != nil {
		return err
	}

	// In batch mode, execute plan for each file separately
	if cli.Batch {
		return runBatchExecution(cli, plan, pwd, fileInfos)
	}

	// Create and run the executing agent normally (all files at once)
	return runExecutionPhase(cli, plan, pwd, fileInfos)
}

// expandPatterns expands glob patterns into actual file paths
func expandPatterns(patterns []string, pwd string) ([]string, error) {
	// If no patterns provided, return empty slice to indicate working from current directory
	if len(patterns) == 0 {
		slog.Debug("no patterns provided, working from current directory", "pwd", pwd)
		return []string{}, nil
	}

	var filenames []string
	seenFiles := make(map[string]bool)

	for _, pattern := range patterns {
		// Check if pattern contains glob characters
		if strings.ContainsAny(pattern, "*?[{") {
			// It's a glob pattern - use doublestar to expand it
			matches, err := doublestar.FilepathGlob(pattern, doublestar.WithFilesOnly())
			if err != nil {
				return nil, fmt.Errorf("failed to expand pattern %s: %w", pattern, err)
			}

			if len(matches) == 0 {
				slog.Warn("pattern matched no files", "pattern", pattern)
				continue
			}

			for _, match := range matches {
				// Convert to absolute path for consistency
				absMatch, err := filepath.Abs(match)
				if err != nil {
					return nil, fmt.Errorf("failed to get absolute path for %s: %w", match, err)
				}

				// Ensure file is within current working directory
				if !strings.HasPrefix(absMatch, pwd) {
					slog.Warn("file outside working directory, skipping", "file", match, "pwd", pwd)
					continue
				}

				// Avoid duplicates
				if !seenFiles[absMatch] {
					filenames = append(filenames, match)
					seenFiles[absMatch] = true
				}
			}
		} else {
			// It's a regular filename - check if it exists
			if _, err := os.Stat(pattern); err != nil {
				return nil, fmt.Errorf("file %s does not exist: %w", pattern, err)
			}

			// Convert to absolute path for consistency check
			absPattern, err := filepath.Abs(pattern)
			if err != nil {
				return nil, fmt.Errorf("failed to get absolute path for %s: %w", pattern, err)
			}

			// Ensure file is within current working directory
			if !strings.HasPrefix(absPattern, pwd) {
				return nil, fmt.Errorf("file %s is not within the current working directory %s", pattern, pwd)
			}

			// Avoid duplicates
			if !seenFiles[absPattern] {
				filenames = append(filenames, pattern)
				seenFiles[absPattern] = true
			}
		}
	}

	// For empty patterns, we return empty slice (handled elsewhere)
	if len(patterns) == 0 {
		return filenames, nil
	}

	if len(filenames) == 0 {
		return nil, fmt.Errorf("no files found matching the provided patterns")
	}

	slog.Debug("expanded patterns", "patterns", patterns, "files", filenames, "count", len(filenames))
	return filenames, nil
}

// processFiles reads and analyzes the files provided as CLI arguments
func processFiles(filenames []string, pwd string) ([]map[string]interface{}, error) {
	// If no files specified, return empty slice to indicate working from current directory
	if len(filenames) == 0 {
		slog.Debug("no files specified, working from current directory", "pwd", pwd)
		return []map[string]interface{}{}, nil
	}

	fileInfo := make([]map[string]interface{}, 0, len(filenames))

	for _, filename := range filenames {
		// Read file content
		contents, err := os.ReadFile(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", filename, err)
		}

		// Get absolute path
		absFilename, err := filepath.Abs(filename)
		if err != nil {
			return nil, fmt.Errorf("failed to get absolute path for file %s: %w", filename, err)
		}

		// Ensure file is within current working directory
		if !strings.HasPrefix(absFilename, pwd) {
			return nil, fmt.Errorf("file %s is not within the current working directory %s", filename, pwd)
		}

		// Create file info entry
		relativeFilename := strings.TrimPrefix(absFilename, pwd+"/")
		lang := enry.GetLanguage(filepath.Base(relativeFilename), contents)

		fileInfo = append(fileInfo, map[string]interface{}{
			"filename": relativeFilename,
			"language": lang,
			"size":     len(contents),
		})
	}

	return fileInfo, nil
}

// runPlanningPhase sets up and executes the planning agent
func runPlanningPhase(cli *CLI, pwd string, fileInfos []map[string]interface{}) (string, error) {
	// Load planning prompt template
	planningTmpl, err := loadPromptTemplate("planning.md")
	if err != nil {
		return "", fmt.Errorf("failed to load planning prompt: %w", err)
	}

	var customPrompt []byte
	customPromptPath := filepath.Join(pwd, ".prompts", "planning.md")
	if _, err := os.Stat(customPromptPath); err == nil {
		customPrompt, err = os.ReadFile(customPromptPath)
		if err != nil {
			return "", fmt.Errorf("failed to read custom planning prompt: %w", err)
		}
	}

	// Execute planning template
	var planningPromptBuf strings.Builder
	err = planningTmpl.Execute(&planningPromptBuf, map[string]interface{}{
		"Message":      cli.Message,
		"Files":        fileInfos,
		"CustomPrompt": string(customPrompt),
		"BatchMode":    cli.Batch, // Pass batch mode flag to planning template
	})
	if err != nil {
		return "", fmt.Errorf("failed to execute planning prompt template: %w", err)
	}

	// Create planning agent
	planningAgent := agent.New(
		"Planning Agent",
		planningPromptBuf.String(),
		agent.WithClient(client.New(
			cli.PlanningApiEndpoint,
			cli.PlanningApiToken,
			cli.PlanningModel,
		)),
	)

	// Create user message for planning agent
	userMessage := createPlanningUserMessage(cli.Message, fileInfos)
	if cli.Batch {
		userMessage += "\n\nNote: Your plan will be executed in batch mode, processing each file individually."
	}

	// Run planning agent
	response, err := planningAgent.Run(
		context.Background(),
		agent.Messages{
			agent.Message{
				Role:    "user",
				Content: userMessage,
			},
		},
	)
	if err != nil {
		return "", fmt.Errorf("failed to run planning agent: %w", err)
	}

	// Process the plan
	plan := extractAndCleanPlanFromResponse(response)

	// Log the plan
	slog.Debug("planning.agent", "plan", plan, "batch_mode", cli.Batch)

	return plan, nil
}

// runBatchExecution processes each file individually in execution phase
func runBatchExecution(cli *CLI, plan string, pwd string, allFileInfos []map[string]interface{}) error {
	slog.Info("batch.start", "plan", plan)

	// If no files specified, run once for current directory
	if len(allFileInfos) == 0 {
		slog.Info("batch.iter", "working_directory", pwd, "index", 1, "total", 1)
		err := runExecutionPhase(cli, plan, pwd, allFileInfos)
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

		// Execute plan for single file
		err := runExecutionPhase(cli, plan, pwd, singleFileInfo)
		if err != nil {
			return fmt.Errorf("execution failed for file %s: %w", fileName, err)
		}

		slog.Info("batch.completed", "file", fileName)
	}

	slog.Info("batch.done", "total_files", len(allFileInfos))
	return nil
}

// createPlanningUserMessage creates the user message for the planning agent
func createPlanningUserMessage(message string, fileInfos []map[string]interface{}) string {
	var filesList string
	if len(fileInfos) == 0 {
		filesList = "Files: Working from current directory (no specific files provided)\n"
	} else {
		filesList = "Files: \n"
		for _, file := range fileInfos {
			filesList += fmt.Sprintf("- %s: language %q, size %d\n",
				file["filename"], file["language"], file["size"])
		}
	}

	// Add batch mode context if needed
	return "User Messages:\n" + message + "\n\n" + filesList
}

// extractAndCleanPlanFromResponse extracts and cleans the plan from the agent's response
func extractAndCleanPlanFromResponse(response *agent.Response) string {
	plan := response.Messages[len(response.Messages)-1].Content

	// Remove any HTML tags at the beginning
	cleanupPlan := regexp.MustCompile(`</(?:\w+)>`)
	if loc := cleanupPlan.FindStringIndex(plan); loc != nil {
		plan = plan[loc[1]:]
	}

	return strings.TrimSpace(plan)
}

// runExecutionPhase sets up and executes the execution agent
func runExecutionPhase(cli *CLI, plan string, pwd string, fileInfos []map[string]interface{}) error {
	// Load execution prompt template
	executeTmpl, err := loadPromptTemplate("execute.md")
	if err != nil {
		return fmt.Errorf("failed to load execute prompt: %w", err)
	}

	var customPrompt []byte
	customPromptPath := filepath.Join(pwd, ".prompts", "execute.md")
	if _, err := os.Stat(customPromptPath); err == nil {
		customPrompt, err = os.ReadFile(customPromptPath)
		if err != nil {
			return fmt.Errorf("failed to read custom execute prompt: %w", err)
		}
	}

	// Select tools to include
	toolsToInclude := tools.Select(cli.Tools)

	// Check if we're in batch mode with a single file
	isBatchSingleFile := cli.Batch && len(fileInfos) == 1

	// Handle current file for template (may be empty if no files specified)
	var currentFile interface{}
	if len(fileInfos) > 0 {
		currentFile = fileInfos[0]["filename"]
	} else {
		currentFile = ""
	}

	// Execute template for execution agent
	var executePromptBuf strings.Builder
	err = executeTmpl.Execute(&executePromptBuf, map[string]interface{}{
		"Plan":         plan,
		"Files":        fileInfos,
		"Tools":        toolsToInclude,
		"CustomPrompt": string(customPrompt),
		"BatchMode":    isBatchSingleFile,
		"CurrentFile":  currentFile,
	})
	if err != nil {
		return fmt.Errorf("failed to execute execute prompt template: %w", err)
	}

	// Create executing agent
	executingAgent := createExecutingAgent(cli, executePromptBuf.String(), toolsToInclude)

	// Run executing agent
	_, err = executingAgent.Run(
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

	return nil
}

// createExecutingAgent creates and configures the executing agent
func createExecutingAgent(cli *CLI, prompt string, tools []tools.Tool) *agent.Agent {
	executingAgent := agent.New(
		"Executing Agent",
		prompt,
		agent.WithClient(client.New(
			cli.ExecutingApiEndpoint,
			cli.ExecutingApiToken,
			cli.ExecutingModel,
		)),
	)

	// Add the selected tools to the executing agent
	for _, tool := range tools {
		executingAgent.Tools.Add(agent.MustWrapStruct(tool.Description, tool.Implementation))
	}

	slog.Debug("executing.agent", "prompt", prompt)

	return executingAgent
}

// loadPromptTemplate loads a prompt from the embedded filesystem and parses it as a template
func loadPromptTemplate(name string) (*template.Template, error) {
	content, err := promptsFS.ReadFile(filepath.Join("prompts", name))
	if err != nil {
		return nil, fmt.Errorf("failed to read prompt %s: %w", name, err)
	}

	return template.New(name).Funcs(sprig.FuncMap()).Parse(string(content))
}
