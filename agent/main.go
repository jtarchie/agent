package main

import (
	"embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/alecthomas/kong"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/go-enry/go-enry/v2"
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
	Message  string   `help:"Message to send to the planning agent." required:"" env:"AGENT_MESSAGE"`
	Batch    bool     `help:"Enable batch mode for the executing agent." default:"false" env:"AGENT_BATCH"`

	Tools []string `help:"List of tools to allow the executing agent to use. Default is all." optional:"" env:"AGENT_TOOLS"`

	PlanningApiToken    string `help:"API token for OpenAI compatible endpoint" env:"AGENT_PLANNING_API_TOKEN"`
	PlanningApiEndpoint string `help:"API endpoint for OpenAI compatible endpoint" default:"http://localhost:11434/v1" env:"AGENT_PLANNING_API_ENDPOINT"`
	PlanningModel       string `help:"Model to use for the planning agent." default:"phi4-reasoning:latest" env:"AGENT_PLANNING_MODEL"`

	ExecutingApiToken    string `help:"API token for OpenAI compatible endpoint" env:"AGENT_EXECUTING_API_TOKEN"`
	ExecutingApiEndpoint string `help:"API endpoint for OpenAI compatible endpoint" default:"http://localhost:11434/v1" env:"AGENT_EXECUTING_API_ENDPOINT"`
	ExecutingModel       string `help:"Model to use for the executing agent." default:"qwen3:32b" env:"AGENT_EXECUTING_MODEL"`
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

	// Create and run the planning phase using Planner
	planner := NewPlanner(cli, pwd, promptsFS)
	plan, err := planner.Run(fileInfos)
	if err != nil {
		return err // Error is already contextualized by planner.Run
	}

	// Create and run the execution phase using Executor
	executor := NewExecutor(cli, pwd, promptsFS)
	if cli.Batch {
		return executor.RunBatch(plan, fileInfos) // Error is already contextualized
	}

	return executor.Run(plan, fileInfos) // Error is already contextualized
}

// expandPatterns expands glob patterns into actual file paths
func expandPatterns(patterns []string, pwd string) ([]string, error) {
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

// loadPromptTemplate loads a prompt from the provided embedded filesystem and parses it as a template
func loadPromptTemplate(fs embed.FS, name string) (*template.Template, error) {
	content, err := fs.ReadFile(filepath.Join("prompts", name))
	if err != nil {
		return nil, fmt.Errorf("failed to read prompt %s: %w", name, err)
	}

	return template.New(name).Funcs(sprig.FuncMap()).Parse(string(content))
}
