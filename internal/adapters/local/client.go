package local

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"sherpa/pkg/models"
	"sherpa/pkg/utils"
)

// Client handles local folder operations
type Client struct {
	basePath string
}

// NewClient creates a new local folder client
func NewClient(basePath string) (*Client, error) {
	// Validate that the path exists and is a directory
	info, err := os.Stat(basePath)
	if err != nil {
		return nil, fmt.Errorf("invalid path %s: %w", basePath, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path %s is not a directory", basePath)
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path: %w", err)
	}

	return &Client{
		basePath: absPath,
	}, nil
}

// GetRepository returns repository information for the local folder
func (c *Client) GetRepository(ctx context.Context, repoPath string) (*models.Repository, error) {
	// For local folders, we create a mock repository object
	folderName := filepath.Base(c.basePath)

	return &models.Repository{
		ID:                folderName,
		Name:              folderName,
		Path:              c.basePath,
		PathWithNamespace: c.basePath,
		WebURL:            fmt.Sprintf("file://%s", c.basePath),
		Description:       fmt.Sprintf("Local folder: %s", c.basePath),
		Platform:          models.PlatformLocal,
		Owner:             "local",
	}, nil
}

// GetRepositoryTree returns the tree structure of the local folder
func (c *Client) GetRepositoryTree(ctx context.Context, repoPath, branch string) ([]models.RepositoryTree, error) {
	var treeItems []models.RepositoryTree

	err := filepath.WalkDir(c.basePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // Continue walking even if we can't read a specific file
		}

		// Skip the root directory itself
		if path == c.basePath {
			return nil
		}

		// Check if this is a symlink and skip it for security
		if d.Type()&fs.ModeSymlink != 0 {
			return nil // Skip symlinks for security
		}

		// Get relative path from base
		relPath, err := filepath.Rel(c.basePath, path)
		if err != nil {
			return nil // Continue walking
		}

		// Convert to forward slashes for consistency
		relPath = filepath.ToSlash(relPath)

		// Determine type
		itemType := "blob"
		if d.IsDir() {
			itemType = "tree"
		}

		treeItems = append(treeItems, models.RepositoryTree{
			ID:   relPath,
			Name: d.Name(),
			Type: itemType,
			Path: relPath,
			Mode: "100644", // Default file mode
		})

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	return treeItems, nil
}

// sanitizePath validates and sanitizes file paths to prevent directory traversal attacks
func (c *Client) sanitizePath(filePath string) (string, error) {
	// Clean the path to resolve any . or .. elements
	cleanPath := filepath.Clean(filePath)

	// Check for absolute paths or parent directory traversal
	if filepath.IsAbs(cleanPath) || strings.HasPrefix(cleanPath, "..") {
		return "", fmt.Errorf("invalid file path: %s", filePath)
	}

	// Construct full path and ensure it's within base directory
	fullPath := filepath.Join(c.basePath, cleanPath)

	// Get absolute paths for comparison
	absBase, err := filepath.Abs(c.basePath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve base path: %w", err)
	}

	absPath, err := filepath.Abs(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve file path: %w", err)
	}

	// Ensure the resolved path is within the base directory
	if !strings.HasPrefix(absPath, absBase+string(filepath.Separator)) {
		return "", fmt.Errorf("path outside base directory: %s", filePath)
	}

	return fullPath, nil
}

// GetFileContent returns the content of a file
func (c *Client) GetFileContent(ctx context.Context, repoPath, filePath, branch string) (string, error) {
	fullPath, err := c.sanitizePath(filePath)
	if err != nil {
		return "", err
	}

	// Check if file exists and is readable
	info, err := os.Stat(fullPath)
	if err != nil {
		return "", fmt.Errorf("file not found: %s", filePath)
	}

	if info.IsDir() {
		return "", fmt.Errorf("path is a directory: %s", filePath)
	}

	// Check if file is binary
	if utils.IsBinaryFile(fullPath) {
		return "", fmt.Errorf("file is binary: %s", filePath)
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", filePath, err)
	}

	return string(content), nil
}

// GetFileInfo returns information about a file
func (c *Client) GetFileInfo(ctx context.Context, repoPath, filePath, branch string) (*models.FileInfo, error) {
	fullPath, err := c.sanitizePath(filePath)
	if err != nil {
		return &models.FileInfo{
			Path:  filePath,
			Name:  filepath.Base(filePath),
			Error: err,
		}, nil
	}

	info, err := os.Stat(fullPath)
	if err != nil {
		return &models.FileInfo{
			Path:  filePath,
			Name:  filepath.Base(filePath),
			Error: fmt.Errorf("file not found: %s", filePath),
		}, nil
	}

	fileInfo := &models.FileInfo{
		Path:     filePath,
		Name:     info.Name(),
		Size:     info.Size(),
		IsDir:    info.IsDir(),
		IsBinary: false,
		IsText:   true,
	}

	// Don't try to read content for directories
	if info.IsDir() {
		return fileInfo, nil
	}

	// Check if file is binary
	if utils.IsBinaryFile(fullPath) {
		fileInfo.IsBinary = true
		fileInfo.IsText = false
		return fileInfo, nil
	}

	// Read file content
	content, err := os.ReadFile(fullPath)
	if err != nil {
		fileInfo.Error = fmt.Errorf("failed to read file: %w", err)
		return fileInfo, nil
	}

	fileInfo.Content = string(content)
	return fileInfo, nil
}

// GetMultipleFiles returns information about multiple files
func (c *Client) GetMultipleFiles(ctx context.Context, repoPath string, filePaths []string, branch string, maxConcurrency int, config *models.ProcessingConfig) ([]models.FileInfo, error) {
	// Add resource limits for security
	maxMemoryPerFile := config.MaxMemoryPerFile
	maxTotalMemory := config.MaxTotalMemory
	maxFiles := config.MaxFiles

	if len(filePaths) > maxFiles {
		return nil, fmt.Errorf("too many files to process safely: %d (max: %d)", len(filePaths), maxFiles)
	}

	if int64(len(filePaths))*maxMemoryPerFile > maxTotalMemory {
		return nil, fmt.Errorf("estimated memory usage too high for %d files", len(filePaths))
	}

	// Use a semaphore to limit concurrency and WaitGroup to wait for completion
	semaphore := make(chan struct{}, maxConcurrency)
	results := make([]models.FileInfo, len(filePaths))

	// Use WaitGroup to properly wait for all goroutines
	var wg sync.WaitGroup

	// Process files concurrently
	for i, filePath := range filePaths {
		wg.Add(1)
		go func(index int, path string) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			fileInfo, err := c.GetFileInfo(ctx, repoPath, path, branch)
			if err != nil {
				results[index] = models.FileInfo{
					Path:  path,
					Name:  filepath.Base(path),
					Error: err,
				}
			} else {
				results[index] = *fileInfo
			}
		}(i, filePath)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	return results, nil
}

// TestConnection tests if the local folder is accessible
func (c *Client) TestConnection(ctx context.Context) error {
	// Test if we can read the directory
	_, err := os.ReadDir(c.basePath)
	if err != nil {
		return fmt.Errorf("cannot access local folder: %w", err)
	}
	return nil
}

// GetBasePath returns the base path of the local folder
func (c *Client) GetBasePath() string {
	return c.basePath
}
