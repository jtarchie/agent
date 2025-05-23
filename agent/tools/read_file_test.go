package tools_test

import (
	"context"
	"fmt"
	"os"
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
