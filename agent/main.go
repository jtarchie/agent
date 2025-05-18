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
	"github.com/jtarchie/outrageous/agent"
	"github.com/jtarchie/outrageous/client"
)

//go:embed prompts
var promptsFS embed.FS

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})))

	cli := &CLI{}
	ctx := kong.Parse(cli)
	// Call the Run() method of the selected parsed command.
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}

type CLI struct {
	Filenames      []string `arg:"" optional:"" type:"existingfile" help:"List of filenames to process."`
	Message        string   `help:"Message to send to the planning agent." required:""`
	PlanningModel  string   `help:"Model to use for the planning agent." default:"phi4-mini-reasoning:latest"`
	ExecutingModel string   `help:"Model to use for the executing agent." default:"qwen3:8b"`
}

// loadPromptTemplate loads a prompt from the embedded filesystem and parses it as a template
func loadPromptTemplate(name string) (*template.Template, error) {
	content, err := promptsFS.ReadFile(filepath.Join("prompts", name))
	if err != nil {
		return nil, fmt.Errorf("failed to read prompt %s: %w", name, err)
	}

	return template.New(name).Funcs(sprig.FuncMap()).Parse(string(content))
}

func (cli *CLI) Run() error {
	pwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	// Prepare file information for templates
	fileInfo := make([]map[string]interface{}, 0, len(cli.Filenames))
	for _, filename := range cli.Filenames {
		contents, err := os.ReadFile(filename)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", filename, err)
		}

		absFilename, err := filepath.Abs(filename)
		if err != nil {
			return fmt.Errorf("failed to get absolute path for file %s: %w", filename, err)
		}

		if !strings.HasPrefix(absFilename, pwd) {
			return fmt.Errorf("file %s is not within the current working directory %s", filename, pwd)
		}

		relativeFilename := strings.TrimPrefix(absFilename, pwd+"/")
		lang := enry.GetLanguage(filepath.Base(relativeFilename), contents)

		fileInfo = append(fileInfo, map[string]interface{}{
			"filename": relativeFilename,
			"language": lang,
			"size":     len(contents),
		})
	}

	// Load and execute planning prompt template
	planningTmpl, err := loadPromptTemplate("planning.md")
	if err != nil {
		return fmt.Errorf("failed to load planning prompt: %w", err)
	}

	var planningPromptBuf strings.Builder
	err = planningTmpl.Execute(&planningPromptBuf, map[string]interface{}{
		"Message": cli.Message,
		"Files":   fileInfo,
	})
	if err != nil {
		return fmt.Errorf("failed to execute planning prompt template: %w", err)
	}

	planningAgent := agent.New(
		"Planning Agent",
		planningPromptBuf.String(),
		agent.WithClient(client.NewOllamaClient(cli.PlanningModel)),
	)

	// Create user message for planning agent
	filesList := "Files: \n"
	for _, file := range fileInfo {
		filesList += fmt.Sprintf("- %s: language %q, size %d\n",
			file["filename"], file["language"], file["size"])
	}

	userMessage := "User Messages:\n" + cli.Message + "\n\n" + filesList

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
		return fmt.Errorf("failed to run planning agent: %w", err)
	}

	plan := response.Messages[len(response.Messages)-1].Content

	cleanupPlan := regexp.MustCompile(`</(?:\w+)>`)
	if loc := cleanupPlan.FindStringIndex(plan); loc != nil {
		plan = plan[loc[1]:]
	}

	plan = strings.TrimSpace(plan)

	slog.Debug("planning.agent", "plan", plan)

	// Load and execute execution prompt template
	executeTmpl, err := loadPromptTemplate("execute.md")
	if err != nil {
		return fmt.Errorf("failed to load execute prompt: %w", err)
	}

	var executePromptBuf strings.Builder
	err = executeTmpl.Execute(&executePromptBuf, map[string]interface{}{
		"Plan":  plan,
		"Files": fileInfo,
	})
	if err != nil {
		return fmt.Errorf("failed to execute execute prompt template: %w", err)
	}

	executingAgent := agent.New(
		"Executing Agent",
		executePromptBuf.String(),
		agent.WithClient(client.NewOllamaClient(cli.ExecutingModel)),
	)

	executingAgent.Tools.Add(
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
	)

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
