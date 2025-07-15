package adapters

import (
	"os"
	"path/filepath"
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

func TestCreateLocalProvider(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "sherpa-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.go")
	require.NoError(t, os.WriteFile(testFile, []byte("package main"), 0644))

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
			path:      testFile,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := CreateLocalProvider(tt.path)
			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, provider)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)
			}
		})
	}
}

func TestCreateProvider_Local(t *testing.T) {
	config := &models.Config{}

	// Test that local platform returns an error (should use CreateLocalProvider instead)
	_, err := CreateProvider(models.PlatformLocal, config, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "local platform requires special handling")
}

func TestIsLocalPath(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "sherpa-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "absolute path",
			input:    "/usr/local/bin",
			expected: true,
		},
		{
			name:     "relative path with dot",
			input:    "./src",
			expected: true,
		},
		{
			name:     "relative path with dot dot",
			input:    "../parent",
			expected: true,
		},
		{
			name:     "home directory",
			input:    "~/documents",
			expected: true,
		},
		{
			name:     "windows drive letter",
			input:    "C:\\Users\\test",
			expected: true,
		},
		{
			name:     "existing directory",
			input:    tmpDir,
			expected: true,
		},
		{
			name:     "github repo format",
			input:    "owner/repo",
			expected: false,
		},
		{
			name:     "github url",
			input:    "https://github.com/owner/repo",
			expected: false,
		},
		{
			name:     "gitlab url",
			input:    "https://gitlab.com/owner/repo",
			expected: false,
		},
		{
			name:     "simple name",
			input:    "myproject",
			expected: false,
		},
		{
			name:     "non-existent relative path",
			input:    "nonexistent",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isLocalPath(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseRepositoryURL_Local(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "sherpa-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a subdirectory
	subDir := filepath.Join(tmpDir, "subdir")
	require.NoError(t, os.Mkdir(subDir, 0755))

	tests := []struct {
		name      string
		input     string
		wantError bool
		expected  *models.RepositoryInfo
	}{
		{
			name:      "absolute path",
			input:     tmpDir,
			wantError: false,
			expected: &models.RepositoryInfo{
				Platform: models.PlatformLocal,
				Owner:    "local",
				Name:     filepath.Base(tmpDir),
				FullName: tmpDir,
				URL:      "file://" + tmpDir,
			},
		},
		{
			name:      "subdirectory",
			input:     subDir,
			wantError: false,
			expected: &models.RepositoryInfo{
				Platform: models.PlatformLocal,
				Owner:    "local",
				Name:     "subdir",
				FullName: subDir,
				URL:      "file://" + subDir,
			},
		},
		{
			name:      "non-existent directory",
			input:     "/non/existent/path",
			wantError: true,
		},
		{
			name:      "file instead of directory",
			input:     filepath.Join(tmpDir, "test.txt"),
			wantError: true,
		},
	}

	// Create a test file for the file test case
	testFile := filepath.Join(tmpDir, "test.txt")
	require.NoError(t, os.WriteFile(testFile, []byte("test content"), 0644))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseRepositoryURL(tt.input, "")
			if tt.wantError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expected.Platform, result.Platform)
				assert.Equal(t, tt.expected.Owner, result.Owner)
				assert.Equal(t, tt.expected.Name, result.Name)
				assert.Equal(t, tt.expected.FullName, result.FullName)
				assert.Equal(t, tt.expected.URL, result.URL)
			}
		})
	}
}

func TestParseRepositoryURL_LocalWithBranch(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "sherpa-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Test branch specification (should be ignored for local paths)
	input := tmpDir + "#main"
	result, err := ParseRepositoryURL(input, "")
	require.NoError(t, err)
	assert.Equal(t, models.PlatformLocal, result.Platform)
	assert.Equal(t, "main", result.Branch)
	assert.Equal(t, tmpDir, result.FullName) // Branch should be stripped from path
}
