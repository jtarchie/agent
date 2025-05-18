package main

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/go-enry/go-enry/v2"
	"github.com/jtarchie/outrageous/agent"
	"github.com/jtarchie/outrageous/client"
)

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

//go:embed prompts/planning.md
var planningPrompt string

//go:embed prompts/execute.md
var executePrompt string

func (cli *CLI) Run() error {
	planningAgent := agent.New(
		"Planning Agent",
		planningPrompt,
		agent.WithClient(client.NewOllamaClient(cli.PlanningModel)),
	)

	pwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current working directory: %w", err)
	}

	message := "User Messages:\n" + cli.Message + "\n\nFiles: \n"
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
		message += fmt.Sprintf("- %s: language %q, size %d\n", relativeFilename, lang, len(contents))
	}

	response, err := planningAgent.Run(
		context.Background(),
		agent.Messages{
			agent.Message{
				Role:    "user",
				Content: message,
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

	executingAgent := agent.New(
		"Executing Agent",
		executePrompt,
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
