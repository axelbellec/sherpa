package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPlatform(t *testing.T) {
	t.Run("should have correct platform constants", func(t *testing.T) {
		assert.Equal(t, Platform("github"), PlatformGitHub)
		assert.Equal(t, Platform("gitlab"), PlatformGitLab)
	})
}

func TestRepositoryInfo(t *testing.T) {
	t.Run("should create repository info", func(t *testing.T) {
		repo := &RepositoryInfo{
			FullName: "owner/repo",
			Owner:    "owner",
			Name:     "repo",
			Platform: PlatformGitHub,
			Branch:   "main",
		}

		assert.Equal(t, "owner/repo", repo.FullName)
		assert.Equal(t, "owner", repo.Owner)
		assert.Equal(t, "repo", repo.Name)
		assert.Equal(t, PlatformGitHub, repo.Platform)
		assert.Equal(t, "main", repo.Branch)
	})
}

func TestRepository(t *testing.T) {
	t.Run("should create repository model", func(t *testing.T) {
		repo := &Repository{
			ID:                123,
			Name:              "test-repo",
			Path:              "test-repo",
			PathWithNamespace: "owner/test-repo",
			WebURL:            "https://github.com/owner/test-repo",
			Description:       "Test repository",
			Platform:          PlatformGitHub,
			Owner:             "owner",
		}

		assert.Equal(t, 123, repo.ID)
		assert.Equal(t, "test-repo", repo.Name)
		assert.Equal(t, "test-repo", repo.Path)
		assert.Equal(t, "owner/test-repo", repo.PathWithNamespace)
		assert.Equal(t, "https://github.com/owner/test-repo", repo.WebURL)
		assert.Equal(t, "Test repository", repo.Description)
		assert.Equal(t, PlatformGitHub, repo.Platform)
		assert.Equal(t, "owner", repo.Owner)
	})
}

func TestFileInfo(t *testing.T) {
	t.Run("should create file info", func(t *testing.T) {
		file := &FileInfo{
			Path:     "src/main.go",
			Name:     "main.go",
			Content:  "package main",
			Size:     12,
			IsText:   true,
			IsBinary: false,
			Error:    nil,
		}

		assert.Equal(t, "src/main.go", file.Path)
		assert.Equal(t, "main.go", file.Name)
		assert.Equal(t, "package main", file.Content)
		assert.Equal(t, int64(12), file.Size)
		assert.True(t, file.IsText)
		assert.False(t, file.IsBinary)
		assert.Nil(t, file.Error)
	})

	t.Run("should handle file with error", func(t *testing.T) {
		file := &FileInfo{
			Path:  "error.txt",
			Name:  "error.txt",
			Error: assert.AnError,
		}

		assert.Equal(t, "error.txt", file.Path)
		assert.Equal(t, "error.txt", file.Name)
		assert.NotNil(t, file.Error)
	})
}

func TestProcessingResult(t *testing.T) {
	t.Run("should create processing result", func(t *testing.T) {
		repo := &Repository{
			Name:     "test-repo",
			Platform: PlatformGitHub,
		}

		files := []FileInfo{
			{
				Path:    "README.md",
				Content: "# Test",
				Size:    6,
			},
			{
				Path:    "main.go",
				Content: "package main",
				Size:    12,
			},
		}

		duration := time.Second * 5

		result := &ProcessingResult{
			Repository:  *repo,
			Files:       files,
			TotalFiles:  len(files),
			TotalSize:   18,
			Duration:    duration,
			Errors:      []error{},
		}

		assert.Equal(t, *repo, result.Repository)
		assert.Len(t, result.Files, 2)
		assert.Equal(t, 2, result.TotalFiles)
		assert.Equal(t, int64(18), result.TotalSize)
		assert.Equal(t, duration, result.Duration)
		assert.Empty(t, result.Errors)
	})
}

func TestLLMsOutput(t *testing.T) {
	t.Run("should create LLMs output", func(t *testing.T) {
		repo := &Repository{
			Name:     "test-repo",
			Platform: PlatformGitLab,
		}

		files := []FileInfo{
			{
				Path:    "test.txt",
				Content: "test content",
				Size:    12,
			},
		}

		output := &LLMsOutput{
			Repository:    *repo,
			TotalFiles:    1,
			TotalSize:     12,
			FileContents:  files,
		}

		assert.Equal(t, *repo, output.Repository)
		assert.Len(t, output.FileContents, 1)
		assert.Equal(t, 1, output.TotalFiles)
		assert.Equal(t, int64(12), output.TotalSize)
	})
}

func TestConfig(t *testing.T) {
	t.Run("should create config with all sections", func(t *testing.T) {
		config := &Config{
			GitLab: GitLabConfig{
				BaseURL:  "https://gitlab.com",
				TokenEnv: "GITLAB_TOKEN",
			},
			GitHub: GitHubConfig{
				BaseURL:  "https://api.github.com",
				TokenEnv: "GITHUB_TOKEN",
			},
			Processing: ProcessingConfig{
				Ignore:         []string{"*.log"},
				IncludeOnly:    []string{"*.go"},
				MaxFileSize:    "1MB",
				SkipBinary:     true,
				MaxConcurrency: 10,
			},
			Output: OutputConfig{
				Directory:      "./output",
				OrganizeByDate: true,
			},
			Cache: CacheConfig{
				Enabled:   true,
				Directory: ".cache",
				TTL:       3600,
			},
		}

		assert.Equal(t, "https://gitlab.com", config.GitLab.BaseURL)
		assert.Equal(t, "GITLAB_TOKEN", config.GitLab.TokenEnv)
		assert.Equal(t, "https://api.github.com", config.GitHub.BaseURL)
		assert.Equal(t, "GITHUB_TOKEN", config.GitHub.TokenEnv)
		assert.Contains(t, config.Processing.Ignore, "*.log")
		assert.Contains(t, config.Processing.IncludeOnly, "*.go")
		assert.Equal(t, "1MB", config.Processing.MaxFileSize)
		assert.True(t, config.Processing.SkipBinary)
		assert.Equal(t, 10, config.Processing.MaxConcurrency)
		assert.Equal(t, "./output", config.Output.Directory)
		assert.True(t, config.Output.OrganizeByDate)
		assert.True(t, config.Cache.Enabled)
		assert.Equal(t, ".cache", config.Cache.Directory)
		assert.Equal(t, time.Duration(3600), config.Cache.TTL)
	})
}

func TestCLIOptions(t *testing.T) {
	t.Run("should create CLI options", func(t *testing.T) {
		options := &CLIOptions{
			Token:               "test-token",
			BaseURL:             "https://custom.gitlab.com",
			Output:              "./custom-output",
			Ignore:              "*.log,*.tmp",
			IncludeOnly:         "*.go,*.py",
			ConfigFile:          "config.yml",
			DefaultPlatform:     "github",
			MaxReposConcurrency: 3,
			MaxFilesConcurrency: 15,
			Verbose:             true,
			Quiet:               false,
		}

		assert.Equal(t, "test-token", options.Token)
		assert.Equal(t, "https://custom.gitlab.com", options.BaseURL)
		assert.Equal(t, "./custom-output", options.Output)
		assert.Equal(t, "*.log,*.tmp", options.Ignore)
		assert.Equal(t, "*.go,*.py", options.IncludeOnly)
		assert.Equal(t, "config.yml", options.ConfigFile)
		assert.Equal(t, "github", options.DefaultPlatform)
		assert.Equal(t, 3, options.MaxReposConcurrency)
		assert.Equal(t, 15, options.MaxFilesConcurrency)
		assert.True(t, options.Verbose)
		assert.False(t, options.Quiet)
	})
}