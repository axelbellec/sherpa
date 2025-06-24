package github

import (
	"context"
	"fmt"
	"strings"
	"time"

	"sherpa/pkg/logger"
	"sherpa/pkg/models"

	"github.com/google/go-github/v60/github"
	"golang.org/x/oauth2"
)

// Client wraps the GitHub API client with additional functionality
type Client struct {
	client  *github.Client
	baseURL string
	token   string
}

// NewClient creates a new GitHub client
func NewClient(baseURL, token string) (*Client, error) {
	if token == "" {
		return nil, fmt.Errorf("GitHub token is required")
	}

	if baseURL == "" {
		baseURL = "https://api.github.com"
	}

	// Create OAuth2 token source
	tokenSource := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	oauth2Client := oauth2.NewClient(context.Background(), tokenSource)

	// Create GitHub client
	client := github.NewClient(oauth2Client)

	// Debug: log the initial base URL
	logger.Logger.WithField("initial_base_url", client.BaseURL.String()).Debug("Initial GitHub client BaseURL")

	if baseURL != "https://api.github.com" {
		newURL, err := client.BaseURL.Parse(baseURL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse base URL: %w", err)
		}
		client.BaseURL = newURL
		logger.Logger.WithField("custom_base_url", client.BaseURL.String()).Debug("Set custom GitHub BaseURL")
	}

	// Debug: log the final base URL
	logger.Logger.WithField("final_base_url", client.BaseURL.String()).Debug("Final GitHub client BaseURL")

	return &Client{
		client:  client,
		baseURL: baseURL,
		token:   token,
	}, nil
}

// GetRepository fetches repository information by owner/repo
func (c *Client) GetRepository(ctx context.Context, owner, repo string) (*models.Repository, error) {
	logger.Logger.WithFields(map[string]interface{}{
		"owner":      owner,
		"repository": repo,
	}).Debug("Fetching GitHub repository information")

	repository, _, err := c.client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		logger.Logger.WithError(err).WithFields(map[string]interface{}{
			"owner":      owner,
			"repository": repo,
		}).Error("Failed to fetch GitHub repository")
		return nil, fmt.Errorf("failed to fetch repository %s/%s: %w", owner, repo, err)
	}

	return &models.Repository{
		ID:                repository.GetID(),
		Name:              repository.GetName(),
		Path:              repository.GetName(),
		PathWithNamespace: repository.GetFullName(),
		WebURL:            repository.GetHTMLURL(),
		Description:       repository.GetDescription(),
		Platform:          models.PlatformGitHub,
		Owner:             owner,
	}, nil
}

// GetRepositoryTree fetches the complete repository tree structure
func (c *Client) GetRepositoryTree(ctx context.Context, owner, repo, branch string) ([]models.RepositoryTree, error) {
	logger.Logger.WithFields(map[string]interface{}{
		"owner":      owner,
		"repository": repo,
		"branch":     branch,
	}).Debug("Fetching GitHub repository tree structure")

	// Use specified branch or get default branch
	targetBranch := branch
	if targetBranch == "" {
		// Get default branch first
		repository, _, err := c.client.Repositories.Get(ctx, owner, repo)
		if err != nil {
			return nil, fmt.Errorf("failed to get repository info: %w", err)
		}
		targetBranch = repository.GetDefaultBranch()
		if targetBranch == "" {
			targetBranch = "main"
		}
	}

	// Get tree recursively
	tree, _, err := c.client.Git.GetTree(ctx, owner, repo, targetBranch, true)
	if err != nil {
		// If specified branch fails, try default branches
		if branch != "" {
			logger.Logger.WithFields(map[string]interface{}{
				"owner":      owner,
				"repository": repo,
				"branch":     branch,
			}).Debug("Branch-specific tree fetch failed, trying default branches")

			// Try main branch
			tree, _, err = c.client.Git.GetTree(ctx, owner, repo, "main", true)
			if err != nil {
				// Try master branch
				tree, _, err = c.client.Git.GetTree(ctx, owner, repo, "master", true)
			}
		}

		if err != nil {
			logger.Logger.WithError(err).WithFields(map[string]interface{}{
				"owner":      owner,
				"repository": repo,
				"branch":     branch,
			}).Error("Failed to fetch GitHub repository tree")
			return nil, fmt.Errorf("failed to fetch repository tree: %w", err)
		}
	}

	var allFiles []models.RepositoryTree
	for _, entry := range tree.Entries {
		if entry.GetType() == "blob" { // Only include files, not directories
			file := models.RepositoryTree{
				ID:   entry.GetSHA(),
				Name: extractFileName(entry.GetPath()),
				Type: "blob",
				Path: entry.GetPath(),
				Mode: entry.GetMode(),
			}
			allFiles = append(allFiles, file)
		}
	}

	logger.Logger.WithFields(map[string]interface{}{
		"owner":      owner,
		"repository": repo,
		"branch":     targetBranch,
		"file_count": len(allFiles),
	}).Debug("Successfully fetched GitHub repository tree")
	return allFiles, nil
}

// GetFileContent fetches the content of a specific file
func (c *Client) GetFileContent(ctx context.Context, owner, repo, filePath, branch string) (string, error) {
	logger.Logger.WithFields(map[string]interface{}{
		"owner":      owner,
		"repository": repo,
		"file":       filePath,
		"branch":     branch,
	}).Debug("Fetching GitHub file content")

	// Prepare options with branch if specified
	opts := &github.RepositoryContentGetOptions{}
	if branch != "" {
		opts.Ref = branch
	}

	fileContent, _, _, err := c.client.Repositories.GetContents(ctx, owner, repo, filePath, opts)
	if err != nil {
		// If branch-specific call fails, try without branch specification (default branch)
		if branch != "" {
			logger.Logger.WithFields(map[string]interface{}{
				"owner":      owner,
				"repository": repo,
				"file":       filePath,
				"branch":     branch,
			}).Debug("Branch-specific file fetch failed, trying default branch")

			fileContent, _, _, err = c.client.Repositories.GetContents(ctx, owner, repo, filePath, nil)
		}

		if err != nil {
			logger.Logger.WithError(err).WithFields(map[string]interface{}{
				"owner":      owner,
				"repository": repo,
				"file":       filePath,
				"branch":     branch,
			}).Error("Failed to fetch file from GitHub")
			return "", fmt.Errorf("failed to fetch file %s: %w", filePath, err)
		}
	}

	if fileContent == nil {
		return "", fmt.Errorf("file content is nil")
	}

	content, err := fileContent.GetContent()
	if err != nil {
		return "", fmt.Errorf("failed to decode file content: %w", err)
	}

	return content, nil
}

// GetFileInfo fetches file information and content
func (c *Client) GetFileInfo(ctx context.Context, owner, repo, filePath, branch string) (*models.FileInfo, error) {
	fileInfo := &models.FileInfo{
		Path: filePath,
		Name: extractFileName(filePath),
	}

	// Get file content
	content, err := c.GetFileContent(ctx, owner, repo, filePath, branch)
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
func (c *Client) GetMultipleFiles(ctx context.Context, owner, repo string, filePaths []string, branch string, maxConcurrency int) ([]models.FileInfo, error) {
	logger.Logger.WithFields(map[string]interface{}{
		"owner":           owner,
		"repository":      repo,
		"file_count":      len(filePaths),
		"max_concurrency": maxConcurrency,
	}).Debug("Fetching multiple files concurrently from GitHub")

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

			fileInfo, err := c.GetFileInfo(ctx, owner, repo, path, branch)
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

// TestConnection tests the GitHub connection and authentication
func (c *Client) TestConnection(ctx context.Context) error {
	logger.Logger.WithFields(map[string]interface{}{
		"base_url":     c.baseURL,
		"token_prefix": c.token[:10] + "...",
	}).Debug("Testing GitHub connection")

	user, resp, err := c.client.Users.Get(ctx, "")
	if err != nil {
		logger.Logger.WithError(err).WithFields(map[string]interface{}{
			"base_url": c.baseURL,
			"status_code": func() int {
				if resp != nil {
					return resp.StatusCode
				}
				return 0
			}(),
			"response_headers": func() map[string][]string {
				if resp != nil {
					return resp.Header
				}
				return nil
			}(),
		}).Error("Failed to authenticate with GitHub")
		return fmt.Errorf("failed to authenticate with GitHub: %w", err)
	}

	if user == nil {
		logger.Logger.Error("Authentication failed: no user information returned")
		return fmt.Errorf("authentication failed: no user information returned")
	}

	logger.Logger.WithFields(map[string]interface{}{
		"user_id":  user.GetID(),
		"username": user.GetLogin(),
		"base_url": c.baseURL,
	}).Debug("GitHub connection test successful")
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
func (c *Client) GetRateLimitInfo(ctx context.Context) (*RateLimitInfo, error) {
	rateLimits, _, err := c.client.RateLimit.Get(ctx)
	if err != nil {
		return nil, err
	}

	return &RateLimitInfo{
		Limit:     rateLimits.Core.Limit,
		Remaining: rateLimits.Core.Remaining,
		ResetTime: rateLimits.Core.Reset.Time,
	}, nil
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
		strings.Contains(err.Error(), "403") ||
		strings.Contains(err.Error(), "API rate limit exceeded")
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
