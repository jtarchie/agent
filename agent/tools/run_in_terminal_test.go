package tools_test

import (
	"context"
	"os"
	"testing"

	"github.com/jtarchie/agent/agent/tools"
	. "github.com/onsi/gomega"
)

func TestRunInTerminal(t *testing.T) {
	assert := NewGomegaWithT(t)

	tmpFile, err := os.CreateTemp("", "testfile")
	assert.Expect(err).NotTo(HaveOccurred())

	err = tmpFile.Close()
	assert.Expect(err).NotTo(HaveOccurred())

	runner := tools.RunInTerminal{
		Command: []string{"echo", "Hello, World!"},
	}

	payload, err := runner.Call(context.Background())
	assert.Expect(err).NotTo(HaveOccurred())

	status, ok := payload.(map[string]any)
	assert.Expect(ok).To(BeTrue())
	assert.Expect(status).To(Equal(map[string]any{
		"status": "completed",
		"output": "Hello, World!\n",
	}))
}

func TestRunInTerminalErroredCommand(t *testing.T) {
	assert := NewGomegaWithT(t)

	runner := tools.RunInTerminal{
		Command: []string{"nonexistentcommand"},
	}

	_, err := runner.Call(context.Background())
	assert.Expect(err).To(HaveOccurred())
}

func TestRunInTerminalMissingCommand(t *testing.T) {
	assert := NewGomegaWithT(t)

	runner := tools.RunInTerminal{
		Command: nil,
	}

	_, err := runner.Call(context.Background())
	assert.Expect(err).To(HaveOccurred())
}
