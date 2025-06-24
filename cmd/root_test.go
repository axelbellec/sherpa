package cmd

import (
	"testing"

	"sherpa/internal/orchestration"
	"sherpa/pkg/models"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootCmd(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{
			name:     "should have correct use",
			expected: "sherpa",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, RootCmd.Use)
		})
	}
}

func TestFetchCmd(t *testing.T) {
	t.Run("should have correct use", func(t *testing.T) {
		assert.Equal(t, "fetch [repository...]", fetchCmd.Use)
	})

	t.Run("should require minimum args", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.SetArgs([]string{})

		err := fetchCmd.Args(cmd, []string{})
		assert.Error(t, err)
	})

	t.Run("should accept valid args", func(t *testing.T) {
		cmd := &cobra.Command{}
		cmd.SetArgs([]string{"owner/repo"})

		err := fetchCmd.Args(cmd, []string{"owner/repo"})
		assert.NoError(t, err)
	})
}

func TestParseRepositories(t *testing.T) {
	tests := []struct {
		name            string
		args            []string
		defaultPlatform string
		expectedCount   int
		expectedError   bool
	}{
		{
			name:            "should parse github repository",
			args:            []string{"https://github.com/owner/repo"},
			defaultPlatform: "",
			expectedCount:   1,
			expectedError:   false,
		},
		{
			name:            "should parse gitlab repository",
			args:            []string{"https://gitlab.com/owner/repo"},
			defaultPlatform: "",
			expectedCount:   1,
			expectedError:   false,
		},
		{
			name:            "should parse multiple repositories",
			args:            []string{"https://github.com/owner/repo1", "https://gitlab.com/owner/repo2"},
			defaultPlatform: "",
			expectedCount:   2,
			expectedError:   false,
		},
		{
			name:            "should use default platform",
			args:            []string{"owner/repo"},
			defaultPlatform: "github",
			expectedCount:   1,
			expectedError:   false,
		},
		{
			name:            "should error on invalid default platform",
			args:            []string{"owner/repo"},
			defaultPlatform: "invalid",
			expectedCount:   0,
			expectedError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseRepositories(tt.args, tt.defaultPlatform)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			totalCount := 0
			for _, repos := range result {
				totalCount += len(repos)
			}
			assert.Equal(t, tt.expectedCount, totalCount)
		})
	}
}

func TestGetTokenForPlatform(t *testing.T) {
	config := &models.Config{
		GitLab: models.GitLabConfig{
			TokenEnv: "NONEXISTENT_TOKEN",
		},
		GitHub: models.GitHubConfig{
			TokenEnv: "NONEXISTENT_TOKEN",
		},
	}

	tests := []struct {
		name          string
		platform      models.Platform
		cliToken      string
		expectedError bool
	}{
		{
			name:          "should use cli token for gitlab",
			platform:      models.PlatformGitLab,
			cliToken:      "test-token",
			expectedError: false,
		},
		{
			name:          "should use cli token for github",
			platform:      models.PlatformGitHub,
			cliToken:      "test-token",
			expectedError: false,
		},
		{
			name:          "should error without token for gitlab",
			platform:      models.PlatformGitLab,
			cliToken:      "",
			expectedError: true,
		},
		{
			name:          "should error without token for github",
			platform:      models.PlatformGitHub,
			cliToken:      "",
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := orchestration.GetTokenForPlatform(tt.platform, config, tt.cliToken)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.cliToken, token)
			}
		})
	}
}

func TestWriteFile(t *testing.T) {
	t.Run("should write file successfully", func(t *testing.T) {
		// This test would need temporary file handling
		// Skipping implementation for brevity
		t.Skip("Implement with temporary file handling")
	})
}
