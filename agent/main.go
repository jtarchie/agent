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
	Filenames      []string `arg:"" optional:"" type:"existingfile" help:"List of filenames to process."`
	Message        string   `help:"Message to send to the planning agent." required:""`
	PlanningModel  string   `help:"Model to use for the planning agent." default:"phi4-mini-reasoning:latest"`
	ExecutingModel string   `help:"Model to use for the executing agent." default:"qwen3:8b"`
	Tools          []string `help:"List of tools to allow the executing agent to use. Default is all." optional:"" enum:"ReadFile,RunInTerminal,InsertEditIntoFile"`
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

	// Process files
	fileInfos, err := processFiles(cli.Filenames, pwd)
	if err != nil {
		return err
	}

	// Create and run the planning agent
	plan, err := runPlanningPhase(cli, fileInfos)
	if err != nil {
		return err
	}

	// Create and run the executing agent
	return runExecutionPhase(cli, plan, fileInfos)
}

// processFiles reads and analyzes the files provided as CLI arguments
func processFiles(filenames []string, pwd string) ([]map[string]interface{}, error) {
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
func runPlanningPhase(cli *CLI, fileInfos []map[string]interface{}) (string, error) {
	// Load planning prompt template
	planningTmpl, err := loadPromptTemplate("planning.md")
	if err != nil {
		return "", fmt.Errorf("failed to load planning prompt: %w", err)
	}

	// Execute planning template
	var planningPromptBuf strings.Builder
	err = planningTmpl.Execute(&planningPromptBuf, map[string]interface{}{
		"Message": cli.Message,
		"Files":   fileInfos,
	})
	if err != nil {
		return "", fmt.Errorf("failed to execute planning prompt template: %w", err)
	}

	// Create planning agent
	planningAgent := agent.New(
		"Planning Agent",
		planningPromptBuf.String(),
		agent.WithClient(client.NewOllamaClient(cli.PlanningModel)),
	)

	// Create user message for planning agent
	userMessage := createPlanningUserMessage(cli.Message, fileInfos)

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
	slog.Debug("planning.agent", "plan", plan)

	return plan, nil
}

// createPlanningUserMessage creates the user message for the planning agent
func createPlanningUserMessage(message string, fileInfos []map[string]interface{}) string {
	filesList := "Files: \n"
	for _, file := range fileInfos {
		filesList += fmt.Sprintf("- %s: language %q, size %d\n",
			file["filename"], file["language"], file["size"])
	}

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
func runExecutionPhase(cli *CLI, plan string, fileInfos []map[string]interface{}) error {
	// Load execution prompt template
	executeTmpl, err := loadPromptTemplate("execute.md")
	if err != nil {
		return fmt.Errorf("failed to load execute prompt: %w", err)
	}

	// Select tools to include
	toolsToInclude := tools.Select(cli.Tools)

	// Execute template for execution agent
	var executePromptBuf strings.Builder
	err = executeTmpl.Execute(&executePromptBuf, map[string]interface{}{
		"Plan":  plan,
		"Files": fileInfos,
		"Tools": toolsToInclude,
	})
	if err != nil {
		return fmt.Errorf("failed to execute execute prompt template: %w", err)
	}

	// Create executing agent
	executingAgent := createExecutingAgent(executePromptBuf.String(), cli.ExecutingModel, toolsToInclude)

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
func createExecutingAgent(prompt string, modelName string, tools []tools.Tool) *agent.Agent {
	executingAgent := agent.New(
		"Executing Agent",
		prompt,
		agent.WithClient(client.NewOllamaClient(modelName)),
	)

	// Add the selected tools to the executing agent
	for _, tool := range tools {
		executingAgent.Tools.Add(agent.MustWrapStruct(tool.Description, tool.Implementation))
	}

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
