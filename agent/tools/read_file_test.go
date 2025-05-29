package tools_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jtarchie/agent/agent/tools"
	. "github.com/onsi/gomega"
)

func TestReadFile(t *testing.T) {
	assert := NewGomegaWithT(t)

	tmpFile, err := os.CreateTemp("", "testfile")
	assert.Expect(err).NotTo(HaveOccurred())

	for i := range 100 {
		_, err := fmt.Fprintf(tmpFile, "This is line %d\n", i)
		assert.Expect(err).NotTo(HaveOccurred())
	}

	err = tmpFile.Close()
	assert.Expect(err).NotTo(HaveOccurred())

	reader := tools.ReadFile{
		FilePath:            tmpFile.Name(),
		StartLineNumberZero: 10,
		EndLineNumberZero:   21,
	}

	payload, err := reader.Call(context.Background())
	assert.Expect(err).NotTo(HaveOccurred())

	lines, ok := payload.(string)
	assert.Expect(ok).To(BeTrue())

	assert.Expect(lines).NotTo(ContainSubstring("This is line 1\n"))
	for i := 10; i < 21; i++ {
		assert.Expect(lines).To(ContainSubstring(fmt.Sprintf("This is line %d\n", i)))
	}
	assert.Expect(lines).NotTo(ContainSubstring("This is line 21\n"))
}

func TestReadFileMissingFile(t *testing.T) {
	assert := NewGomegaWithT(t)

	reader := tools.ReadFile{
		FilePath:            "asdfASDFasdfasdfasdfasdf.txt.dat",
		StartLineNumberZero: 10,
		EndLineNumberZero:   21,
	}

	_, err := reader.Call(context.Background())
	assert.Expect(err).To(HaveOccurred())
}

func TestReadFileLineOutOfRange(t *testing.T) {
	assert := NewGomegaWithT(t)

	tmpFile, err := os.CreateTemp("", "testfile")
	assert.Expect(err).NotTo(HaveOccurred())

	for i := range 100 {
		_, err := fmt.Fprintf(tmpFile, "This is line %d\n", i)
		assert.Expect(err).NotTo(HaveOccurred())
	}

	err = tmpFile.Close()
	assert.Expect(err).NotTo(HaveOccurred())

	reader := tools.ReadFile{
		FilePath:            tmpFile.Name(),
		StartLineNumberZero: 101,
		EndLineNumberZero:   21,
	}

	_, err = reader.Call(context.Background())
	assert.Expect(err).To(HaveOccurred())

	reader = tools.ReadFile{
		FilePath:            tmpFile.Name(),
		StartLineNumberZero: 10,
		EndLineNumberZero:   1000,
	}

	payload, err := reader.Call(context.Background())
	assert.Expect(err).NotTo(HaveOccurred())

	lines, ok := payload.(string)
	assert.Expect(ok).To(BeTrue())

	assert.Expect(lines).NotTo(ContainSubstring("This is line 1\n"))
	for i := 10; i < 100; i++ {
		assert.Expect(lines).To(ContainSubstring(fmt.Sprintf("This is line %d\n", i)))
	}
	assert.Expect(lines).NotTo(ContainSubstring("This is line 101\n"))
}

func TestReadFileWithRootPathSecurity(t *testing.T) {
	assert := NewGomegaWithT(t)

	// Create a temporary directory as root
	tmpDir, err := os.MkdirTemp("", "rootdir")
	assert.Expect(err).NotTo(HaveOccurred())
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a file inside the root directory
	tmpFile, err := os.CreateTemp(tmpDir, "testfile")
	assert.Expect(err).NotTo(HaveOccurred())

	_, err = fmt.Fprintf(tmpFile, "This is a test file\n")
	assert.Expect(err).NotTo(HaveOccurred())
	err = tmpFile.Close()
	assert.Expect(err).NotTo(HaveOccurred())

	// Test reading file within root path - should succeed
	reader := tools.ReadFile{
		FilePath:            tmpFile.Name(),
		StartLineNumberZero: 0,
		EndLineNumberZero:   0,
		RootPath:            tmpDir,
	}

	payload, err := reader.Call(context.Background())
	assert.Expect(err).NotTo(HaveOccurred())

	content, ok := payload.(string)
	assert.Expect(ok).To(BeTrue())
	assert.Expect(content).To(ContainSubstring("This is a test file"))

	// Test reading file outside root path - should fail
	outsideFile, err := os.CreateTemp("", "outsidefile")
	assert.Expect(err).NotTo(HaveOccurred())
	defer func() { _ = os.Remove(outsideFile.Name()) }()

	_, err = fmt.Fprintf(outsideFile, "This should not be accessible\n")
	assert.Expect(err).NotTo(HaveOccurred())
	err = outsideFile.Close()
	assert.Expect(err).NotTo(HaveOccurred())

	reader = tools.ReadFile{
		FilePath:            outsideFile.Name(),
		StartLineNumberZero: 0,
		EndLineNumberZero:   0,
		RootPath:            tmpDir,
	}

	_, err = reader.Call(context.Background())
	assert.Expect(err).To(HaveOccurred())
	assert.Expect(err.Error()).To(ContainSubstring("security error"))
}

func TestReadFilePathTraversalAttack(t *testing.T) {
	assert := NewGomegaWithT(t)

	// Create a temporary directory as root
	tmpDir, err := os.MkdirTemp("", "rootdir")
	assert.Expect(err).NotTo(HaveOccurred())
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Try to read a file outside using path traversal
	reader := tools.ReadFile{
		FilePath:            filepath.Join(tmpDir, "../../../etc/passwd"),
		StartLineNumberZero: 0,
		EndLineNumberZero:   0,
		RootPath:            tmpDir,
	}

	_, err = reader.Call(context.Background())
	assert.Expect(err).To(HaveOccurred())
	assert.Expect(err.Error()).To(ContainSubstring("security error"))
}
