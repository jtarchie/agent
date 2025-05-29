package tools_test

import (
	"context"
	"os"
	"path/filepath"
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

func TestCreateMissingFile(t *testing.T) {
	assert := NewGomegaWithT(t)

	tmpDir, err := os.MkdirTemp("", "testdir")
	assert.Expect(err).NotTo(HaveOccurred())
	defer func() { _ = os.RemoveAll(tmpDir) }() // Clean up the temporary directory

	inserter := tools.InsertEditIntoFile{
		FilePath: filepath.Join(tmpDir, "some", "dir", "missingfile.txt"),
		Content:  "This is a line",
	}

	payload, err := inserter.Call(context.Background())
	assert.Expect(err).NotTo(HaveOccurred())

	status, ok := payload.(map[string]any)
	assert.Expect(ok).To(BeTrue())
	assert.Expect(status).To(Equal(map[string]any{
		"status": "completed",
	}))

	_, err = os.Stat(inserter.FilePath)
	assert.Expect(err).NotTo(HaveOccurred())
}