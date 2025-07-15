package adapters

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"sherpa/internal/adapters/github"
	"sherpa/internal/adapters/gitlab"
	"sherpa/internal/adapters/local"
	"sherpa/pkg/models"
)

// Provider defines the interface for VCS providers (GitLab, GitHub, etc.)
type Provider interface {
	GetRepository(ctx context.Context, repoPath string) (*models.Repository, error)
	GetRepositoryTree(ctx context.Context, repoPath, branch string) ([]models.RepositoryTree, error)
	GetFileContent(ctx context.Context, repoPath, filePath, branch string) (string, error)
	GetFileInfo(ctx context.Context, repoPath, filePath, branch string) (*models.FileInfo, error)
	GetMultipleFiles(ctx context.Context, repoPath string, filePaths []string, branch string, maxConcurrency int) ([]models.FileInfo, error)
	TestConnection(ctx context.Context) error
}

// GitLabProvider wraps the GitLab client to implement the Provider interface
type GitLabProvider struct {
	client *gitlab.Client
}

// NewGitLabProvider creates a new GitLab provider
func NewGitLabProvider(baseURL, token string) (*GitLabProvider, error) {
	client, err := gitlab.NewClient(baseURL, token)
	if err != nil {
		return nil, err
	}
	return &GitLabProvider{client: client}, nil
}

func (p *GitLabProvider) GetRepository(ctx context.Context, repoPath string) (*models.Repository, error) {
	return p.client.GetRepository(ctx, repoPath)
}

func (p *GitLabProvider) GetRepositoryTree(ctx context.Context, repoPath, branch string) ([]models.RepositoryTree, error) {
	return p.client.GetRepositoryTree(ctx, repoPath, branch)
}

func (p *GitLabProvider) GetFileContent(ctx context.Context, repoPath, filePath, branch string) (string, error) {
	return p.client.GetFileContent(ctx, repoPath, filePath, branch)
}

func (p *GitLabProvider) GetFileInfo(ctx context.Context, repoPath, filePath, branch string) (*models.FileInfo, error) {
	return p.client.GetFileInfo(ctx, repoPath, filePath, branch)
}

func (p *GitLabProvider) GetMultipleFiles(ctx context.Context, repoPath string, filePaths []string, branch string, maxConcurrency int) ([]models.FileInfo, error) {
	return p.client.GetMultipleFiles(ctx, repoPath, filePaths, branch, maxConcurrency)
}

func (p *GitLabProvider) TestConnection(ctx context.Context) error {
	return p.client.TestConnection(ctx)
}

// GitHubProvider wraps the GitHub client to implement the Provider interface
type GitHubProvider struct {
	client *github.Client
}

// NewGitHubProvider creates a new GitHub provider
func NewGitHubProvider(baseURL, token string) (*GitHubProvider, error) {
	client, err := github.NewClient(baseURL, token)
	if err != nil {
		return nil, err
	}
	return &GitHubProvider{client: client}, nil
}

func (p *GitHubProvider) GetRepository(ctx context.Context, repoPath string) (*models.Repository, error) {
	owner, repo, err := parseGitHubRepoPath(repoPath)
	if err != nil {
		return nil, err
	}
	return p.client.GetRepository(ctx, owner, repo)
}

func (p *GitHubProvider) GetRepositoryTree(ctx context.Context, repoPath, branch string) ([]models.RepositoryTree, error) {
	owner, repo, err := parseGitHubRepoPath(repoPath)
	if err != nil {
		return nil, err
	}
	return p.client.GetRepositoryTree(ctx, owner, repo, branch)
}

func (p *GitHubProvider) GetFileContent(ctx context.Context, repoPath, filePath, branch string) (string, error) {
	owner, repo, err := parseGitHubRepoPath(repoPath)
	if err != nil {
		return "", err
	}
	return p.client.GetFileContent(ctx, owner, repo, filePath, branch)
}

func (p *GitHubProvider) GetFileInfo(ctx context.Context, repoPath, filePath, branch string) (*models.FileInfo, error) {
	owner, repo, err := parseGitHubRepoPath(repoPath)
	if err != nil {
		return nil, err
	}
	return p.client.GetFileInfo(ctx, owner, repo, filePath, branch)
}

func (p *GitHubProvider) GetMultipleFiles(ctx context.Context, repoPath string, filePaths []string, branch string, maxConcurrency int) ([]models.FileInfo, error) {
	owner, repo, err := parseGitHubRepoPath(repoPath)
	if err != nil {
		return nil, err
	}
	return p.client.GetMultipleFiles(ctx, owner, repo, filePaths, branch, maxConcurrency)
}

func (p *GitHubProvider) TestConnection(ctx context.Context) error {
	return p.client.TestConnection(ctx)
}

// LocalProvider wraps the local client to implement the Provider interface
type LocalProvider struct {
	client *local.Client
}

// NewLocalProvider creates a new local provider
func NewLocalProvider(folderPath string) (*LocalProvider, error) {
	client, err := local.NewClient(folderPath)
	if err != nil {
		return nil, err
	}
	return &LocalProvider{client: client}, nil
}

func (p *LocalProvider) GetRepository(ctx context.Context, repoPath string) (*models.Repository, error) {
	return p.client.GetRepository(ctx, repoPath)
}

func (p *LocalProvider) GetRepositoryTree(ctx context.Context, repoPath, branch string) ([]models.RepositoryTree, error) {
	return p.client.GetRepositoryTree(ctx, repoPath, branch)
}

func (p *LocalProvider) GetFileContent(ctx context.Context, repoPath, filePath, branch string) (string, error) {
	return p.client.GetFileContent(ctx, repoPath, filePath, branch)
}

func (p *LocalProvider) GetFileInfo(ctx context.Context, repoPath, filePath, branch string) (*models.FileInfo, error) {
	return p.client.GetFileInfo(ctx, repoPath, filePath, branch)
}

func (p *LocalProvider) GetMultipleFiles(ctx context.Context, repoPath string, filePaths []string, branch string, maxConcurrency int) ([]models.FileInfo, error) {
	return p.client.GetMultipleFiles(ctx, repoPath, filePaths, branch, maxConcurrency)
}

func (p *LocalProvider) TestConnection(ctx context.Context) error {
	return p.client.TestConnection(ctx)
}

// ParseRepositoryURL parses a repository URL or path and returns repository information
func ParseRepositoryURL(input string, defaultPlatform models.Platform) (*models.RepositoryInfo, error) {
	input = strings.TrimSpace(input)

	// Extract branch from fragment (e.g., #develop)
	var branch string
	if strings.Contains(input, "#") {
		parts := strings.Split(input, "#")
		if len(parts) == 2 {
			input = parts[0]
			branch = parts[1]
		}
	}

	// Handle local paths (check if path exists on filesystem)
	if isLocalPath(input) {
		absPath, err := filepath.Abs(input)
		if err != nil {
			return nil, fmt.Errorf("invalid local path: %w", err)
		}
		
		// Validate that the path exists and is a directory
		if info, err := os.Stat(absPath); err != nil || !info.IsDir() {
			return nil, fmt.Errorf("local path does not exist or is not a directory: %s", input)
		}

		folderName := filepath.Base(absPath)
		return &models.RepositoryInfo{
			Platform: models.PlatformLocal,
			Owner:    "local",
			Name:     folderName,
			FullName: absPath,
			URL:      fmt.Sprintf("file://%s", absPath),
			Branch:   branch,
		}, nil
	}

	// Handle URLs
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		repoInfo, err := parseURL(input)
		if err != nil {
			return nil, err
		}
		repoInfo.Branch = branch
		return repoInfo, nil
	}

	// Handle SSH URLs
	if strings.HasPrefix(input, "git@") {
		repoInfo, err := parseSSHURL(input)
		if err != nil {
			return nil, err
		}
		repoInfo.Branch = branch
		return repoInfo, nil
	}

	// Handle owner/repo format (use specified default platform)
	if strings.Contains(input, "/") && !strings.Contains(input, " ") {
		parts := strings.Split(input, "/")
		if len(parts) == 2 {
			// Use the specified default platform, or fallback to GitHub if not specified
			platform := defaultPlatform
			if platform == "" {
				platform = models.PlatformGitHub
			}
			return &models.RepositoryInfo{
				Platform: platform,
				Owner:    parts[0],
				Name:     parts[1],
				FullName: input,
				Branch:   branch,
			}, nil
		}
	}

	// Default to specified platform for bare repository names, or GitLab for backward compatibility
	platform := defaultPlatform
	if platform == "" {
		platform = models.PlatformGitLab
	}
	return &models.RepositoryInfo{
		Platform: platform,
		Owner:    "",
		Name:     input,
		FullName: input,
		Branch:   branch,
	}, nil
}

// isLocalPath checks if the input appears to be a local filesystem path
func isLocalPath(input string) bool {
	// Check for common local path indicators
	if strings.HasPrefix(input, "/") || 
		strings.HasPrefix(input, "./") || 
		strings.HasPrefix(input, "../") || 
		strings.HasPrefix(input, "~") ||
		(len(input) > 2 && input[1] == ':' && (input[0] >= 'A' && input[0] <= 'Z' || input[0] >= 'a' && input[0] <= 'z')) { // Windows drive letters
		return true
	}
	
	// Check if it's a relative path that exists on the filesystem
	if info, err := os.Stat(input); err == nil && info.IsDir() {
		return true
	}
	
	return false
}

func parseURL(input string) (*models.RepositoryInfo, error) {
	u, err := url.Parse(input)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	switch u.Hostname() {
	case "github.com", "www.github.com":
		return parseGitHubURL(u, input)
	case "gitlab.com", "www.gitlab.com":
		return parseGitLabURL(u, input)
	default:
		// For self-hosted instances, try to determine by URL structure
		if strings.Contains(u.Path, "/tree/") || strings.Contains(u.Path, "/blob/") {
			// GitHub-style URL structure
			return parseGitHubURL(u, input)
		} else {
			// Default to GitLab for self-hosted
			return parseGitLabURL(u, input)
		}
	}
}

func parseGitHubURL(u *url.URL, original string) (*models.RepositoryInfo, error) {
	// GitHub URL format: https://github.com/owner/repo
	pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(pathParts) < 2 {
		return nil, fmt.Errorf("invalid GitHub URL format")
	}

	owner := pathParts[0]
	repo := pathParts[1]

	// Remove .git suffix if present
	repo = strings.TrimSuffix(repo, ".git")

	return &models.RepositoryInfo{
		Platform: models.PlatformGitHub,
		Owner:    owner,
		Name:     repo,
		FullName: fmt.Sprintf("%s/%s", owner, repo),
		URL:      original,
	}, nil
}

func parseGitLabURL(u *url.URL, original string) (*models.RepositoryInfo, error) {
	// GitLab URL format: https://gitlab.com/owner/repo or https://gitlab.com/group/subgroup/repo
	pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(pathParts) < 2 {
		return nil, fmt.Errorf("invalid GitLab URL format")
	}

	// For GitLab, the full path is the "owner" for API purposes
	fullPath := strings.Join(pathParts, "/")

	// Remove .git suffix if present
	fullPath = strings.TrimSuffix(fullPath, ".git")

	return &models.RepositoryInfo{
		Platform: models.PlatformGitLab,
		Owner:    pathParts[0],
		Name:     pathParts[len(pathParts)-1],
		FullName: fullPath,
		URL:      original,
	}, nil
}

func parseSSHURL(input string) (*models.RepositoryInfo, error) {
	// SSH URL formats:
	// git@github.com:owner/repo.git
	// git@gitlab.com:owner/repo.git

	re := regexp.MustCompile(`^git@([^:]+):(.+)\.git$`)
	matches := re.FindStringSubmatch(input)
	if len(matches) != 3 {
		return nil, fmt.Errorf("invalid SSH URL format")
	}

	hostname := matches[1]
	repoPath := matches[2]

	var platform models.Platform
	switch hostname {
	case "github.com":
		platform = models.PlatformGitHub
	case "gitlab.com":
		platform = models.PlatformGitLab
	default:
		// Default to GitLab for self-hosted
		platform = models.PlatformGitLab
	}

	pathParts := strings.Split(repoPath, "/")

	return &models.RepositoryInfo{
		Platform: platform,
		Owner:    pathParts[0],
		Name:     pathParts[len(pathParts)-1],
		FullName: repoPath,
		URL:      input,
	}, nil
}

// CreateProvider creates a VCS provider based on platform and configuration
func CreateProvider(platform models.Platform, config *models.Config, token string) (Provider, error) {
	switch platform {
	case models.PlatformGitLab:
		return NewGitLabProvider(config.GitLab.BaseURL, token)
	case models.PlatformGitHub:
		return NewGitHubProvider(config.GitHub.BaseURL, token)
	case models.PlatformLocal:
		// For local platform, token is not needed, but we need the folder path
		// This should be handled differently in the orchestration layer
		return nil, fmt.Errorf("local platform requires special handling in orchestration layer")
	default:
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}
}

// CreateLocalProvider creates a local provider for a specific folder path
func CreateLocalProvider(folderPath string) (Provider, error) {
	return NewLocalProvider(folderPath)
}

// Helper function for GitHub provider
func parseGitHubRepoPath(repoPath string) (owner, repo string, err error) {
	parts := strings.Split(repoPath, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid GitHub repository path format, expected 'owner/repo'")
	}
	return parts[0], parts[1], nil
}
