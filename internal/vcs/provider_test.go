package vcs

import (
	"testing"

	"sherpa/pkg/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateProvider(t *testing.T) {
	t.Run("should create GitHub provider", func(t *testing.T) {
		config := &models.Config{
			GitHub: models.GitHubConfig{
				BaseURL:  "https://api.github.com",
				TokenEnv: "GITHUB_TOKEN",
			},
		}
		token := "github-token"

		provider, err := CreateProvider(models.PlatformGitHub, config, token)
		require.NoError(t, err)
		assert.NotNil(t, provider)
	})

	t.Run("should create GitLab provider", func(t *testing.T) {
		config := &models.Config{
			GitLab: models.GitLabConfig{
				BaseURL:  "https://gitlab.com",
				TokenEnv: "GITLAB_TOKEN",
			},
		}
		token := "gitlab-token"

		provider, err := CreateProvider(models.PlatformGitLab, config, token)
		require.NoError(t, err)
		assert.NotNil(t, provider)
	})

	t.Run("should error on unsupported platform", func(t *testing.T) {
		config := &models.Config{}
		token := "token"

		_, err := CreateProvider("unsupported", config, token)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported platform")
	})

	t.Run("should error on empty token for GitHub", func(t *testing.T) {
		config := &models.Config{
			GitHub: models.GitHubConfig{
				BaseURL: "https://api.github.com",
			},
		}

		_, err := CreateProvider(models.PlatformGitHub, config, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "token is required")
	})

	t.Run("should error on empty token for GitLab", func(t *testing.T) {
		config := &models.Config{
			GitLab: models.GitLabConfig{
				BaseURL: "https://gitlab.com",
			},
		}

		_, err := CreateProvider(models.PlatformGitLab, config, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "token is required")
	})

	t.Run("should handle custom base URLs", func(t *testing.T) {
		config := &models.Config{
			GitHub: models.GitHubConfig{
				BaseURL: "https://github.enterprise.com/api/v3",
			},
		}
		token := "enterprise-token"

		provider, err := CreateProvider(models.PlatformGitHub, config, token)
		require.NoError(t, err)
		assert.NotNil(t, provider)
	})
}

func TestGetProviderForPlatform(t *testing.T) {
	tests := []struct {
		name        string
		platform    models.Platform
		expectError bool
	}{
		{
			name:        "should support GitHub",
			platform:    models.PlatformGitHub,
			expectError: false,
		},
		{
			name:        "should support GitLab",
			platform:    models.PlatformGitLab,
			expectError: false,
		},
		{
			name:        "should error on unsupported platform",
			platform:    "bitbucket",
			expectError: true,
		},
		{
			name:        "should error on empty platform",
			platform:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &models.Config{
				GitHub: models.GitHubConfig{
					BaseURL: "https://api.github.com",
				},
				GitLab: models.GitLabConfig{
					BaseURL: "https://gitlab.com",
				},
			}
			token := "test-token"

			_, err := CreateProvider(tt.platform, config, token)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateProviderConfig(t *testing.T) {
	t.Run("should validate GitHub config", func(t *testing.T) {
		config := &models.Config{
			GitHub: models.GitHubConfig{
				BaseURL:  "https://api.github.com",
				TokenEnv: "GITHUB_TOKEN",
			},
		}

		_, err := CreateProvider(models.PlatformGitHub, config, "token")
		assert.NoError(t, err)
	})

	t.Run("should validate GitLab config", func(t *testing.T) {
		config := &models.Config{
			GitLab: models.GitLabConfig{
				BaseURL:  "https://gitlab.com",
				TokenEnv: "GITLAB_TOKEN",
			},
		}

		_, err := CreateProvider(models.PlatformGitLab, config, "token")
		assert.NoError(t, err)
	})

	t.Run("should handle custom GitHub base URL", func(t *testing.T) {
		config := &models.Config{
			GitHub: models.GitHubConfig{
				BaseURL: "https://github.enterprise.com/api/v3",
			},
		}

		provider, err := CreateProvider(models.PlatformGitHub, config, "test-token")
		assert.NoError(t, err)
		assert.NotNil(t, provider)
	})

	t.Run("should handle custom GitLab base URL", func(t *testing.T) {
		config := &models.Config{
			GitLab: models.GitLabConfig{
				BaseURL: "https://gitlab.enterprise.com",
			},
		}

		provider, err := CreateProvider(models.PlatformGitLab, config, "test-token")
		assert.NoError(t, err)
		assert.NotNil(t, provider)
	})
}

