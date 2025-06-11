package tools

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/jtarchie/outrageous/agent"
)

type RuntimeInfo struct {
	Name    string
	Path    string
	Version string
}

func MustScript() agent.Tool {
	availableRuntimes := detectAvailableRuntimes()

	description := "This tool lets you execute source code directly by providing both the code and the command to run it. It's useful when precise control over execution is needed. Only features from the language's standard library (for the specified version) should be used—external dependencies are not installed or supported."

	if len(availableRuntimes) > 0 {
		runtimeList := make([]string, 0, len(availableRuntimes))
		for _, runtime := range availableRuntimes {
			runtimeList = append(runtimeList, fmt.Sprintf("%s (%s)", runtime.Name, runtime.Version))
		}
		description += "\n\nAvailable Runtimes (you MUST use one of these exact names): " + strings.Join(runtimeList, ", ")
	} else {
		description += "\n\nNo supported CLIs found on the system."
	}

	return agent.MustWrapStruct(
		description,
		RunInTerminal{},
	)
}

func detectAvailableRuntimes() []RuntimeInfo {
	runtimeConfigs := []struct {
		name       string
		versionCmd []string
	}{
		{"ruby", []string{"--version"}},
		{"python", []string{"--version"}},
		{"python3", []string{"--version"}},
		{"node", []string{"--version"}},
		{"bash", []string{"--version"}},
		{"sh", []string{"--version"}},
	}

	var availableRuntimes []RuntimeInfo

	for _, config := range runtimeConfigs {
		path, err := exec.LookPath(config.name)
		if err != nil {
			continue
		}

		// Try to get version
		cmd := exec.Command(path, config.versionCmd...)
		output, err := cmd.Output()
		if err != nil {
			continue
		}

		version := strings.TrimSpace(string(output))
		// Take only the first line for cleaner output
		if lines := strings.Split(version, "\n"); len(lines) > 0 {
			version = lines[0]
		}

		availableRuntimes = append(availableRuntimes, RuntimeInfo{
			Name:    config.name,
			Path:    path,
			Version: version,
		})
	}

	return availableRuntimes
}
