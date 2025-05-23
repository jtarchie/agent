package tools_test

import (
	"context"
	"os"
	"testing"

	"github.com/jtarchie/agent/agent/tools"
	. "github.com/onsi/gomega"
)

func TestInsertEditIntoFile(t *testing.T) {
	assert := NewGomegaWithT(t)

	tmpFile, err := os.CreateTemp("", "testfile")
	assert.Expect(err).NotTo(HaveOccurred())

	err = tmpFile.Close()
	assert.Expect(err).NotTo(HaveOccurred())

	inserter := tools.InsertEditIntoFile{
		FilePath: tmpFile.Name(),
		Content:  "This is a line",
	}

	payload, err := inserter.Call(context.Background())
	assert.Expect(err).NotTo(HaveOccurred())

	status, ok := payload.(map[string]any)
	assert.Expect(ok).To(BeTrue())
	assert.Expect(status).To(Equal(map[string]any{
		"status": "completed",
	}))
}
