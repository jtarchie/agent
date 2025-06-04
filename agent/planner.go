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

	"github.com/jtarchie/outrageous/agent"
	"github.com/jtarchie/outrageous/client"
)

// Planner orchestrates the planning phase of the agent.
type Planner struct {
	cli       *CLI
	pwd       string
	promptsFS embed.FS
}

// NewPlanner creates a new Planner.
func NewPlanner(cli *CLI, pwd string, promptsFS embed.FS) *Planner {
	return &Planner{
		cli:       cli,
		pwd:       pwd,
		promptsFS: promptsFS,
	}
}

// Run executes the planning phase.
func (p *Planner) Run(fileInfos []map[string]interface{}) (string, error) {
	// Load planning prompt template using the shared loadPromptTemplate function
	planningTmpl, err := loadPromptTemplate(p.promptsFS, "planning.md")
	if err != nil {
		return "", fmt.Errorf("failed to load planning prompt: %w", err)
	}

	var customPrompt []byte
	customPromptPath := filepath.Join(p.pwd, ".prompts", "planning.md")
	if _, err := os.Stat(customPromptPath); err == nil {
		customPrompt, err = os.ReadFile(customPromptPath)
		if err != nil {
			return "", fmt.Errorf("failed to read custom planning prompt: %w", err)
		}
	}

	// Execute planning template
	var planningPromptBuf strings.Builder
	err = planningTmpl.Execute(&planningPromptBuf, map[string]interface{}{
		"Message":      p.cli.Message,
		"Files":        fileInfos,
		"CustomPrompt": string(customPrompt),
		"BatchMode":    p.cli.Batch,
	})
	if err != nil {
		return "", fmt.Errorf("failed to execute planning prompt template: %w", err)
	}

	// Create planning agent
	planningAgent := agent.New(
		"Planning Agent",
		planningPromptBuf.String(),
		agent.WithClient(client.New(
			p.cli.PlanningApiEndpoint,
			p.cli.PlanningApiToken,
			p.cli.PlanningModel,
		)),
	)

	// Create user message for planning agent
	userMessage := p.createPlanningUserMessage(fileInfos)
	if p.cli.Batch {
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
	plan := p.extractAndCleanPlanFromResponse(response)

	slog.Debug("planning.agent", "plan", plan, "batch_mode", p.cli.Batch)
	return plan, nil
}

// createPlanningUserMessage creates the user message for the planning agent.
func (p *Planner) createPlanningUserMessage(fileInfos []map[string]interface{}) string {
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
	return "User Messages:\n" + p.cli.Message + "\n\n" + filesList
}

// extractAndCleanPlanFromResponse extracts and cleans the plan from the agent's response.
func (p *Planner) extractAndCleanPlanFromResponse(response *agent.Response) string {
	plan := response.Messages[len(response.Messages)-1].Content

	cleanupPlan := regexp.MustCompile(`</(?:\w+)>`)
	if loc := cleanupPlan.FindStringIndex(plan); loc != nil {
		plan = plan[loc[1]:]
	}
	return strings.TrimSpace(plan)
}
