package tools

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type RunInTerminal struct {
	Command     []string `json:"command" description:"Command with args to run in the terminal."`
	Explanation string   `json:"explanation" description:"Please provide a brief explanation of why this command needs to run."`
}

func (r RunInTerminal) Call(ctx context.Context) (any, error) {
	if len(r.Command) == 0 {
		return nil, fmt.Errorf("command is required")
	}

	command := exec.CommandContext(ctx, r.Command[0])

	if len(r.Command) > 1 {
		command = exec.CommandContext(ctx, r.Command[0], r.Command[1:]...)
	}

	output := &strings.Builder{}
	command.Stdout = output
	command.Stderr = output

	err := command.Run()
	if err != nil {
		return nil, fmt.Errorf("error running command: %w", err)
	}

	return map[string]any{
		"status":    "completed",
		"output":    output.String(),
		"exit_code": command.ProcessState.ExitCode(),
	}, nil
}
