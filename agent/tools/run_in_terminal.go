package tools

import (
	"context"
	"errors"
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

	stdout, stderr := &strings.Builder{}, &strings.Builder{}
	command.Stdout = stdout
	command.Stderr = stderr

	err := command.Run()
	if err != nil && !errors.Is(err, &exec.ExitError{}) {
		return nil, fmt.Errorf("error running command: %w", err)
	}

	return map[string]any{
		"status":    "completed",
		"stdout":    stdout.String(),
		"stderr":    stderr.String(),
		"exit_code": command.ProcessState.ExitCode(),
	}, nil
}
