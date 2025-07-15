package local

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"sherpa/pkg/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestDir creates a temporary directory with test files
func setupTestDir(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "sherpa-test-*")
	require.NoError(t, err)

	// Create test files
	testFiles := map[string]string{
		"main.go":        "package main\n\nfunc main() {\n\tprintln(\"Hello, World!\")\n}",
		"config.json":    `{"name": "test", "version": "1.0.0"}`,
		"README.md":      "# Test Project\n\nThis is a test project.",
		"subdir/test.go": "package subdir\n\nfunc Test() {\n\t// test function\n}",
	}

	for filePath, content := range testFiles {
		fullPath := filepath.Join(tmpDir, filePath)
		dir := filepath.Dir(fullPath)

		// Create directory if it doesn't exist
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}

		// Write file
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file %s: %v", fullPath, err)
		}
	}

	// Create a binary file for testing
	binaryFile := filepath.Join(tmpDir, "binary.bin")
	binaryContent := []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE}
	require.NoError(t, os.WriteFile(binaryFile, binaryContent, 0644))

	return tmpDir
}

func TestNewClient(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name      string
		path      string
		wantError bool
	}{
		{
			name:      "valid directory",
			path:      tmpDir,
			wantError: false,
		},
		{
			name:      "non-existent directory",
			path:      "/non/existent/path",
			wantError: true,
		},
		{
			name:      "file instead of directory",
			path:      filepath.Join(tmpDir, "main.go"),
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.path)
			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, client)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, client)
				assert.Equal(t, tt.path, client.basePath)
			}
		})
	}
}

func TestClient_GetRepository(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	client, err := NewClient(tmpDir)
	require.NoError(t, err)

	ctx := context.Background()
	repo, err := client.GetRepository(ctx, "test")
	require.NoError(t, err)

	assert.Equal(t, models.PlatformLocal, repo.Platform)
	assert.Equal(t, "local", repo.Owner)
	assert.Equal(t, filepath.Base(tmpDir), repo.Name)
	assert.Equal(t, tmpDir, repo.Path)
	assert.Equal(t, tmpDir, repo.PathWithNamespace)
	assert.Equal(t, "file://"+tmpDir, repo.WebURL)
	assert.Contains(t, repo.Description, "Local folder:")
}

func TestClient_GetRepositoryTree(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	client, err := NewClient(tmpDir)
	require.NoError(t, err)

	ctx := context.Background()
	tree, err := client.GetRepositoryTree(ctx, "test", "main")
	require.NoError(t, err)

	// Check that we have the expected files
	expectedFiles := []string{"main.go", "config.json", "README.md", "subdir", "subdir/test.go", "binary.bin"}
	assert.GreaterOrEqual(t, len(tree), len(expectedFiles))

	// Create a map for easier lookup
	treeMap := make(map[string]models.RepositoryTree)
	for _, item := range tree {
		treeMap[item.Path] = item
	}

	// Check specific files
	mainGo, exists := treeMap["main.go"]
	assert.True(t, exists)
	assert.Equal(t, "main.go", mainGo.Name)
	assert.Equal(t, "blob", mainGo.Type)

	subdir, exists := treeMap["subdir"]
	assert.True(t, exists)
	assert.Equal(t, "subdir", subdir.Name)
	assert.Equal(t, "tree", subdir.Type)

	subdirTest, exists := treeMap["subdir/test.go"]
	assert.True(t, exists)
	assert.Equal(t, "test.go", subdirTest.Name)
	assert.Equal(t, "blob", subdirTest.Type)
}

func TestClient_GetFileContent(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	client, err := NewClient(tmpDir)
	require.NoError(t, err)

	ctx := context.Background()

	tests := []struct {
		name        string
		filePath    string
		expectError bool
		contains    string
	}{
		{
			name:        "valid Go file",
			filePath:    "main.go",
			expectError: false,
			contains:    "func main()",
		},
		{
			name:        "valid JSON file",
			filePath:    "config.json",
			expectError: false,
			contains:    "\"name\": \"test\"",
		},
		{
			name:        "file in subdirectory",
			filePath:    "subdir/test.go",
			expectError: false,
			contains:    "package subdir",
		},
		{
			name:        "non-existent file",
			filePath:    "nonexistent.go",
			expectError: true,
		},
		{
			name:        "binary file",
			filePath:    "binary.bin",
			expectError: true,
		},
		{
			name:        "directory instead of file",
			filePath:    "subdir",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := client.GetFileContent(ctx, "test", tt.filePath, "main")
			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, content)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, content, tt.contains)
			}
		})
	}
}

func TestClient_GetFileInfo(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	client, err := NewClient(tmpDir)
	require.NoError(t, err)

	ctx := context.Background()

	tests := []struct {
		name        string
		filePath    string
		expectError bool
		isDir       bool
		isBinary    bool
		hasContent  bool
	}{
		{
			name:        "valid Go file",
			filePath:    "main.go",
			expectError: false,
			isDir:       false,
			isBinary:    false,
			hasContent:  true,
		},
		{
			name:        "directory",
			filePath:    "subdir",
			expectError: false,
			isDir:       true,
			isBinary:    false,
			hasContent:  false,
		},
		{
			name:        "binary file",
			filePath:    "binary.bin",
			expectError: false,
			isDir:       false,
			isBinary:    true,
			hasContent:  false,
		},
		{
			name:        "non-existent file",
			filePath:    "nonexistent.go",
			expectError: true, // GetFileInfo returns error in FileInfo.Error, not as return value
			isDir:       false,
			isBinary:    false,
			hasContent:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileInfo, err := client.GetFileInfo(ctx, "test", tt.filePath, "main")
			require.NoError(t, err) // GetFileInfo doesn't return errors directly
			require.NotNil(t, fileInfo)

			assert.Equal(t, tt.filePath, fileInfo.Path)
			assert.Equal(t, tt.isDir, fileInfo.IsDir)
			assert.Equal(t, tt.isBinary, fileInfo.IsBinary)

			if tt.expectError {
				assert.NotNil(t, fileInfo.Error)
			} else {
				assert.Nil(t, fileInfo.Error)
			}

			if tt.hasContent {
				assert.NotEmpty(t, fileInfo.Content)
				assert.True(t, fileInfo.IsText)
			}

			if tt.isDir {
				assert.Empty(t, fileInfo.Content)
			}
		})
	}
}

func TestClient_GetMultipleFiles(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	client, err := NewClient(tmpDir)
	require.NoError(t, err)

	ctx := context.Background()
	filePaths := []string{"main.go", "config.json", "README.md", "subdir/test.go", "binary.bin", "nonexistent.go"}

	results, err := client.GetMultipleFiles(ctx, "test", filePaths, "main", 3)
	require.NoError(t, err)
	require.Equal(t, len(filePaths), len(results))

	// Check results
	for i, result := range results {
		expectedPath := filePaths[i]
		assert.Equal(t, expectedPath, result.Path)

		switch expectedPath {
		case "main.go":
			assert.Nil(t, result.Error)
			assert.Contains(t, result.Content, "func main()")
			assert.False(t, result.IsBinary)
		case "config.json":
			assert.Nil(t, result.Error)
			assert.Contains(t, result.Content, "\"name\": \"test\"")
			assert.False(t, result.IsBinary)
		case "README.md":
			assert.Nil(t, result.Error)
			assert.Contains(t, result.Content, "# Test Project")
			assert.False(t, result.IsBinary)
		case "subdir/test.go":
			assert.Nil(t, result.Error)
			assert.Contains(t, result.Content, "package subdir")
			assert.False(t, result.IsBinary)
		case "binary.bin":
			assert.Nil(t, result.Error)
			assert.True(t, result.IsBinary)
			assert.Empty(t, result.Content)
		case "nonexistent.go":
			assert.NotNil(t, result.Error)
			assert.Empty(t, result.Content)
		}
	}
}

func TestClient_GetMultipleFiles_Concurrency(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	client, err := NewClient(tmpDir)
	require.NoError(t, err)

	ctx := context.Background()

	// Create more files to test concurrency
	moreFiles := make([]string, 20)
	for i := 0; i < 20; i++ {
		filename := filepath.Join(tmpDir, fmt.Sprintf("test%d.go", i))
		content := fmt.Sprintf("package main\n\n// File %d\nfunc test%d() {}", i, i)
		require.NoError(t, os.WriteFile(filename, []byte(content), 0644))
		moreFiles[i] = fmt.Sprintf("test%d.go", i)
	}

	// Test with different concurrency levels
	concurrencyLevels := []int{1, 3, 5, 10}

	for _, maxConcurrency := range concurrencyLevels {
		t.Run(fmt.Sprintf("concurrency_%d", maxConcurrency), func(t *testing.T) {
			results, err := client.GetMultipleFiles(ctx, "test", moreFiles, "main", maxConcurrency)
			require.NoError(t, err)
			require.Equal(t, len(moreFiles), len(results))

			// Verify all files were processed
			for i, result := range results {
				expectedPath := moreFiles[i]
				assert.Equal(t, expectedPath, result.Path)
				assert.Nil(t, result.Error)
				assert.Contains(t, result.Content, fmt.Sprintf("// File %d", i))
				assert.False(t, result.IsBinary)
			}
		})
	}
}

func TestClient_TestConnection(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	client, err := NewClient(tmpDir)
	require.NoError(t, err)

	ctx := context.Background()
	err = client.TestConnection(ctx)
	assert.NoError(t, err)

	// Test with invalid directory
	invalidClient := &Client{basePath: "/non/existent/path"}
	err = invalidClient.TestConnection(ctx)
	assert.Error(t, err)
}

func TestClient_GetBasePath(t *testing.T) {
	tmpDir := setupTestDir(t)
	defer os.RemoveAll(tmpDir)

	client, err := NewClient(tmpDir)
	require.NoError(t, err)

	assert.Equal(t, tmpDir, client.GetBasePath())
}
