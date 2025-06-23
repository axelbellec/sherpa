package gitlab

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"sherpa/pkg/logger"
	"sherpa/pkg/models"

	"gitlab.com/gitlab-org/api/client-go"
)

// Client wraps the GitLab API client with additional functionality
type Client struct {
	client  *gitlab.Client
	baseURL string
	token   string
}

// NewClient creates a new GitLab client
func NewClient(baseURL, token string) (*Client, error) {
	if token == "" {
		return nil, fmt.Errorf("GitLab token is required")
	}

	if baseURL == "" {
		baseURL = "https://gitlab.com"
	}

	// Create GitLab client
	client, err := gitlab.NewClient(token, gitlab.WithBaseURL(baseURL))
	if err != nil {
		return nil, fmt.Errorf("failed to create GitLab client: %w", err)
	}

	return &Client{
		client:  client,
		baseURL: baseURL,
		token:   token,
	}, nil
}

// GetRepository fetches repository information by path
func (c *Client) GetRepository(ctx context.Context, repoPath string) (*models.Repository, error) {
	logger.Logger.WithField("repository", repoPath).Debug("Fetching repository information")
	project, _, err := c.client.Projects.GetProject(repoPath, &gitlab.GetProjectOptions{}, gitlab.WithContext(ctx))
	if err != nil {
		logger.Logger.WithError(err).WithField("repository", repoPath).Error("Failed to fetch repository")
		return nil, fmt.Errorf("failed to fetch repository %s: %w", repoPath, err)
	}

	return &models.Repository{
		ID:                project.ID,
		Name:              project.Name,
		Path:              project.Path,
		PathWithNamespace: project.PathWithNamespace,
		WebURL:            project.WebURL,
		Description:       project.Description,
	}, nil
}

// GetRepositoryTree fetches the complete repository tree structure
func (c *Client) GetRepositoryTree(ctx context.Context, repoPath string) ([]models.RepositoryTree, error) {
	logger.Logger.WithField("repository", repoPath).Debug("Fetching repository tree structure")
	var allFiles []models.RepositoryTree

	// Start with root directory
	files, err := c.getTreeRecursive(ctx, repoPath, "", &allFiles)
	if err != nil {
		logger.Logger.WithError(err).WithField("repository", repoPath).Error("Failed to fetch repository tree")
		return nil, fmt.Errorf("failed to fetch repository tree: %w", err)
	}

	logger.Logger.WithFields(map[string]interface{}{
		"repository": repoPath,
		"file_count": len(files),
	}).Debug("Successfully fetched repository tree")
	return files, nil
}

// getTreeRecursive recursively fetches tree structure
func (c *Client) getTreeRecursive(ctx context.Context, repoPath, path string, allFiles *[]models.RepositoryTree) ([]models.RepositoryTree, error) {
	opt := &gitlab.ListTreeOptions{
		Path:      &path,
		Recursive: &[]bool{true}[0],
		ListOptions: gitlab.ListOptions{
			PerPage: 100,
		},
	}

	var pageFiles []models.RepositoryTree

	for {
		treeNodes, resp, err := c.client.Repositories.ListTree(repoPath, opt, gitlab.WithContext(ctx))
		if err != nil {
			return nil, fmt.Errorf("failed to list tree for path %s: %w", path, err)
		}

		for _, node := range treeNodes {
			file := models.RepositoryTree{
				ID:   node.ID,
				Name: node.Name,
				Type: node.Type,
				Path: node.Path,
				Mode: node.Mode,
			}
			pageFiles = append(pageFiles, file)
			*allFiles = append(*allFiles, file)
		}

		if resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	return pageFiles, nil
}

// GetFileContent fetches the content of a specific file
func (c *Client) GetFileContent(ctx context.Context, repoPath, filePath string) (string, error) {
	logger.Logger.WithFields(map[string]interface{}{
		"repository": repoPath,
		"file":       filePath,
	}).Debug("Fetching file content")
	opt := &gitlab.GetFileOptions{
		Ref: &[]string{"main"}[0],
	}

	file, _, err := c.client.RepositoryFiles.GetFile(repoPath, filePath, opt, gitlab.WithContext(ctx))
	if err != nil {
		// Try with master branch if main doesn't exist
		logger.Logger.WithField("file", filePath).Debug("Trying master branch")
		opt.Ref = &[]string{"master"}[0]
		file, _, err = c.client.RepositoryFiles.GetFile(repoPath, filePath, opt, gitlab.WithContext(ctx))
		if err != nil {
			logger.Logger.WithError(err).WithFields(map[string]interface{}{
				"repository": repoPath,
				"file":       filePath,
			}).Error("Failed to fetch file from both main and master branches")
			return "", fmt.Errorf("failed to fetch file %s: %w", filePath, err)
		}
	}

	// Decode base64 content from GitLab API
	decoded, err := base64.StdEncoding.DecodeString(file.Content)
	if err != nil {
		return "", fmt.Errorf("failed to decode file content: %w", err)
	}
	
	return string(decoded), nil
}

// GetFileInfo fetches file information and content
func (c *Client) GetFileInfo(ctx context.Context, repoPath, filePath string) (*models.FileInfo, error) {
	fileInfo := &models.FileInfo{
		Path: filePath,
		Name: extractFileName(filePath),
	}

	// Get file content
	content, err := c.GetFileContent(ctx, repoPath, filePath)
	if err != nil {
		fileInfo.Error = err
		return fileInfo, nil
	}

	fileInfo.Content = content
	fileInfo.Size = int64(len(content))
	fileInfo.IsText = isTextFile(content)
	fileInfo.IsBinary = !fileInfo.IsText

	return fileInfo, nil
}

// GetMultipleFiles fetches multiple files concurrently with rate limiting
func (c *Client) GetMultipleFiles(ctx context.Context, repoPath string, filePaths []string, maxConcurrency int) ([]models.FileInfo, error) {
	logger.Logger.WithFields(map[string]interface{}{
		"repository":     repoPath,
		"file_count":     len(filePaths),
		"max_concurrency": maxConcurrency,
	}).Debug("Fetching multiple files concurrently")
	if maxConcurrency <= 0 {
		maxConcurrency = 5 // Default concurrency
	}

	semaphore := make(chan struct{}, maxConcurrency)
	results := make(chan models.FileInfo, len(filePaths))

	// Start workers
	for _, filePath := range filePaths {
		go func(path string) {
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			fileInfo, err := c.GetFileInfo(ctx, repoPath, path)
			if err != nil {
				fileInfo = &models.FileInfo{
					Path:  path,
					Name:  extractFileName(path),
					Error: err,
				}
			}
			results <- *fileInfo
		}(filePath)
	}

	// Collect results
	var files []models.FileInfo
	for i := 0; i < len(filePaths); i++ {
		files = append(files, <-results)
	}

	close(results)
	return files, nil
}

// TestConnection tests the GitLab connection and authentication
func (c *Client) TestConnection(ctx context.Context) error {
	logger.Logger.WithField("base_url", c.baseURL).Debug("Testing GitLab connection")
	user, _, err := c.client.Users.CurrentUser(gitlab.WithContext(ctx))
	if err != nil {
		logger.Logger.WithError(err).WithField("base_url", c.baseURL).Error("Failed to authenticate with GitLab")
		return fmt.Errorf("failed to authenticate with GitLab: %w", err)
	}

	if user == nil {
		logger.Logger.Error("Authentication failed: no user information returned")
		return fmt.Errorf("authentication failed: no user information returned")
	}

	logger.Logger.WithFields(map[string]interface{}{
		"user_id":   user.ID,
		"username":  user.Username,
		"base_url":  c.baseURL,
	}).Debug("GitLab connection test successful")
	return nil
}

// Helper functions

func extractFileName(path string) string {
	parts := strings.Split(path, "/")
	return parts[len(parts)-1]
}

func isTextFile(content string) bool {
	// Simple heuristic: if content contains null bytes, it's likely binary
	for _, b := range []byte(content) {
		if b == 0 {
			return false
		}
	}
	return true
}

// GetRateLimitInfo returns current rate limit information
func (c *Client) GetRateLimitInfo() *RateLimitInfo {
	// This is a placeholder for rate limit information
	// The GitLab client doesn't expose rate limit headers directly
	return &RateLimitInfo{
		Limit:     1000,
		Remaining: 1000,
		ResetTime: time.Now().Add(time.Hour),
	}
}

// RateLimitInfo contains rate limiting information
type RateLimitInfo struct {
	Limit     int
	Remaining int
	ResetTime time.Time
}

// WithRetry executes a function with exponential backoff retry logic
func (c *Client) WithRetry(ctx context.Context, maxRetries int, fn func() error) error {
	var lastErr error

	for i := 0; i <= maxRetries; i++ {
		if i > 0 {
			// Exponential backoff
			backoff := time.Duration(i*i) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		if err := fn(); err != nil {
			lastErr = err

			// Check if it's a rate limit error
			if isRateLimitError(err) {
				continue
			}

			// Check if it's a temporary network error
			if isTemporaryError(err) {
				continue
			}

			// For other errors, don't retry
			return err
		}

		return nil
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}

func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "rate limit") ||
		strings.Contains(err.Error(), "429")
}

func isTemporaryError(err error) bool {
	if err == nil {
		return false
	}

	// Check for common temporary HTTP errors
	if strings.Contains(err.Error(), "timeout") ||
		strings.Contains(err.Error(), "connection reset") ||
		strings.Contains(err.Error(), "temporary failure") {
		return true
	}

	return false
}
