package local

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sherpa/pkg/models"
)

func TestPathTraversalProtection(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "sherpa-security-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create a test file in the temp directory
	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0644)
	require.NoError(t, err)

	// Create client
	client, err := NewClient(tempDir)
	require.NoError(t, err)

	// Test cases that should fail due to path traversal protection
	maliciousPaths := []string{
		"../../../etc/passwd",
		"/etc/passwd",
		"..\\..\\windows\\system32",
		"normal/../../../etc/passwd",
		"./../../etc/passwd",
		"../outside.txt",
	}

	for _, path := range maliciousPaths {
		t.Run("should_reject_"+path, func(t *testing.T) {
			_, err := client.GetFileContent(context.Background(), "", path, "")
			assert.Error(t, err, "Expected error for malicious path: %s", path)
			assert.Contains(t, err.Error(), "invalid file path", "Error should mention invalid file path")
		})
	}

	// Test valid paths that should work
	validPaths := []string{
		"test.txt",
		"./test.txt",
	}

	for _, path := range validPaths {
		t.Run("should_allow_"+path, func(t *testing.T) {
			content, err := client.GetFileContent(context.Background(), "", path, "")
			assert.NoError(t, err, "Valid path should work: %s", path)
			assert.Equal(t, "test content", content)
		})
	}
}

func TestResourceLimits(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "sherpa-resource-test")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create client
	client, err := NewClient(tempDir)
	require.NoError(t, err)

	// Test with too many files
	tooManyFiles := make([]string, 1001) // Exceeds maxFiles = 1000
	for i := range tooManyFiles {
		tooManyFiles[i] = "file" + string(rune(i)) + ".txt"
	}

	config := &models.ProcessingConfig{
		MaxFiles:         1000,
		MaxMemoryPerFile: 10 * 1024 * 1024,
		MaxTotalMemory:   100 * 1024 * 1024,
	}
	_, err = client.GetMultipleFiles(context.Background(), "", tooManyFiles, "", 5, config)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "too many files to process safely")
}
