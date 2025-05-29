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

func TestRootPathSecurity(t *testing.T) {
	assert := NewGomegaWithT(t)

	// Create a root directory
	rootDir, err := os.MkdirTemp("", "rootdir")
	assert.Expect(err).NotTo(HaveOccurred())
	defer func() { _ = os.RemoveAll(rootDir) }()

	// Test 1: File inside root path - should succeed
	inserter := tools.InsertEditIntoFile{
		FilePath: filepath.Join(rootDir, "allowed", "file.txt"),
		Content:  "This is allowed",
		RootPath: rootDir,
	}

	payload, err := inserter.Call(context.Background())
	assert.Expect(err).NotTo(HaveOccurred())

	status, ok := payload.(map[string]any)
	assert.Expect(ok).To(BeTrue())
	assert.Expect(status).To(Equal(map[string]any{
		"status": "completed",
	}))

	// Verify file was created
	_, err = os.Stat(inserter.FilePath)
	assert.Expect(err).NotTo(HaveOccurred())

	// Test 2: File outside root path - should fail
	outsideDir, err := os.MkdirTemp("", "outsidedir")
	assert.Expect(err).NotTo(HaveOccurred())
	defer func() { _ = os.RemoveAll(outsideDir) }()

	inserterOutside := tools.InsertEditIntoFile{
		FilePath: filepath.Join(outsideDir, "notallowed.txt"),
		Content:  "This should fail",
		RootPath: rootDir,
	}

	_, err = inserterOutside.Call(context.Background())
	assert.Expect(err).To(HaveOccurred())
	assert.Expect(err.Error()).To(ContainSubstring("security error"))

	// Verify file was not created
	_, err = os.Stat(inserterOutside.FilePath)
	assert.Expect(err).To(HaveOccurred())
	assert.Expect(os.IsNotExist(err)).To(BeTrue())
}
