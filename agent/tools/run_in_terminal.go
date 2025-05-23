package tools

import (
	"context"
	"fmt"
	"os/exec"
)

type RunInTerminal struct {
	Command     []string `json:"command" description:"Command with args to run in the terminal."`
	Explanation string   `json:"explanation" description:"Please provide a brief explanation of why this command needs to run."`
}

func (r RunInTerminal) Call(ctx context.Context) (any, error) {
	var (
		out []byte
		err error
	)

	if len(r.Command) == 0 {
		return nil, fmt.Errorf("command is required")
	}

	if len(r.Command) == 1 {
		out, err = exec.CommandContext(ctx, r.Command[0]).CombinedOutput()
	} else {
		out, err = exec.CommandContext(ctx, r.Command[0], r.Command[1:]...).CombinedOutput()
	}

	if err != nil {
		return nil, fmt.Errorf("error running command: %w\nOutput: %s", err, out)
	}

	return map[string]any{
		"status": "completed",
		"output": string(out),
	}, nil
}
