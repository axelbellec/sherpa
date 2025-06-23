package vcs

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"sherpa/pkg/models"
)

// URLParser handles parsing repository URLs and paths
type URLParser struct{}

// NewURLParser creates a new URL parser
func NewURLParser() *URLParser {
	return &URLParser{}
}

// ParseRepositoryURL parses a repository URL or path and returns repository information
func (p *URLParser) ParseRepositoryURL(input string, defaultPlatform models.Platform) (*models.RepositoryInfo, error) {
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

	// Handle URLs
	if strings.HasPrefix(input, "http://") || strings.HasPrefix(input, "https://") {
		repoInfo, err := p.parseURL(input)
		if err != nil {
			return nil, err
		}
		repoInfo.Branch = branch
		return repoInfo, nil
	}

	// Handle SSH URLs
	if strings.HasPrefix(input, "git@") {
		repoInfo, err := p.parseSSHURL(input)
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

// parseURL parses HTTP/HTTPS URLs
func (p *URLParser) parseURL(input string) (*models.RepositoryInfo, error) {
	u, err := url.Parse(input)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	switch u.Hostname() {
	case "github.com", "www.github.com":
		return p.parseGitHubURL(u, input)
	case "gitlab.com", "www.gitlab.com":
		return p.parseGitLabURL(u, input)
	default:
		// For self-hosted instances, try to determine by URL structure
		if strings.Contains(u.Path, "/tree/") || strings.Contains(u.Path, "/blob/") {
			// GitHub-style URL structure
			return p.parseGitHubURL(u, input)
		} else {
			// Default to GitLab for self-hosted
			return p.parseGitLabURL(u, input)
		}
	}
}

// parseGitHubURL parses GitHub URLs
func (p *URLParser) parseGitHubURL(u *url.URL, original string) (*models.RepositoryInfo, error) {
	// GitHub URL format: https://github.com/owner/repo
	pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(pathParts) < 2 {
		return nil, fmt.Errorf("invalid GitHub URL format")
	}

	owner := pathParts[0]
	repo := pathParts[1]

	// Remove .git suffix if present
	if strings.HasSuffix(repo, ".git") {
		repo = strings.TrimSuffix(repo, ".git")
	}

	return &models.RepositoryInfo{
		Platform: models.PlatformGitHub,
		Owner:    owner,
		Name:     repo,
		FullName: fmt.Sprintf("%s/%s", owner, repo),
		URL:      original,
	}, nil
}

// parseGitLabURL parses GitLab URLs
func (p *URLParser) parseGitLabURL(u *url.URL, original string) (*models.RepositoryInfo, error) {
	// GitLab URL format: https://gitlab.com/owner/repo or https://gitlab.com/group/subgroup/repo
	pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(pathParts) < 2 {
		return nil, fmt.Errorf("invalid GitLab URL format")
	}

	// For GitLab, the full path is the "owner" for API purposes
	fullPath := strings.Join(pathParts, "/")

	// Remove .git suffix if present
	if strings.HasSuffix(fullPath, ".git") {
		fullPath = strings.TrimSuffix(fullPath, ".git")
	}

	return &models.RepositoryInfo{
		Platform: models.PlatformGitLab,
		Owner:    pathParts[0],
		Name:     pathParts[len(pathParts)-1],
		FullName: fullPath,
		URL:      original,
	}, nil
}

// parseSSHURL parses SSH URLs
func (p *URLParser) parseSSHURL(input string) (*models.RepositoryInfo, error) {
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
