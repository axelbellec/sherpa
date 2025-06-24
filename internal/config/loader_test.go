package config

import (
	"os"
	"testing"

	"sherpa/pkg/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLoader(t *testing.T) {
	loader := NewLoader()
	assert.NotNil(t, loader)
}

func TestLoader_LoadConfig(t *testing.T) {
	loader := NewLoader()

	t.Run("should load default config when no file specified", func(t *testing.T) {
		config, err := loader.LoadConfig("")
		require.NoError(t, err)
		assert.NotNil(t, config)

		// Verify default values
		assert.Equal(t, "./sherpa-output", config.Output.Directory)
		assert.Equal(t, "GITLAB_TOKEN", config.GitLab.TokenEnv)
		assert.Equal(t, "GITHUB_TOKEN", config.GitHub.TokenEnv)
	})

	t.Run("should use default config when file does not exist", func(t *testing.T) {
		config, err := loader.LoadConfig("nonexistent.yml")
		require.NoError(t, err)
		assert.NotNil(t, config)
		assert.Equal(t, "./sherpa-output", config.Output.Directory)
	})

	t.Run("should load config from valid file", func(t *testing.T) {
		// Create temporary config file
		tempFile, err := os.CreateTemp("", "test-config-*.yml")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name())

		configContent := `
output:
  directory: "./test-output"
  organize_by_date: true
gitlab:
  token_env: "CUSTOM_GITLAB_TOKEN"
github:
  token_env: "CUSTOM_GITHUB_TOKEN"
processing:
  ignore: ["*.log", "*.tmp"]
  include_only: ["*.go", "*.py"]
`
		_, err = tempFile.WriteString(configContent)
		require.NoError(t, err)
		tempFile.Close()

		config, err := loader.LoadConfig(tempFile.Name())
		require.NoError(t, err)

		assert.Equal(t, "./test-output", config.Output.Directory)
		assert.True(t, config.Output.OrganizeByDate)
		assert.Equal(t, "CUSTOM_GITLAB_TOKEN", config.GitLab.TokenEnv)
		assert.Equal(t, "CUSTOM_GITHUB_TOKEN", config.GitHub.TokenEnv)
		assert.Contains(t, config.Processing.Ignore, "*.log")
		assert.Contains(t, config.Processing.IncludeOnly, "*.go")
	})

	t.Run("should error on invalid YAML", func(t *testing.T) {
		// Create temporary config file with invalid YAML
		tempFile, err := os.CreateTemp("", "test-config-*.yml")
		require.NoError(t, err)
		defer os.Remove(tempFile.Name())

		_, err = tempFile.WriteString("invalid: yaml: content: [")
		require.NoError(t, err)
		tempFile.Close()

		_, err = loader.LoadConfig(tempFile.Name())
		assert.Error(t, err)
	})
}

func TestLoader_OverrideWithFlags(t *testing.T) {
	loader := NewLoader()

	t.Run("should override config with CLI options", func(t *testing.T) {
		config := &models.Config{
			Output: models.OutputConfig{
				Directory: "./default-output",
			},
			Processing: models.ProcessingConfig{
				Ignore:         []string{"*.log"},
				MaxConcurrency: 1,
			},
		}

		cliOptions := &models.CLIOptions{
			Output:      "./custom-output",
			Ignore:      "*.tmp,*.cache",
			IncludeOnly: "*.go,*.py",
			BaseURL:     "https://custom.gitlab.com",
		}

		err := loader.OverrideWithFlags(config, cliOptions)
		require.NoError(t, err)

		assert.Equal(t, "./custom-output", config.Output.Directory)
		assert.Contains(t, config.Processing.Ignore, "*.tmp")
		assert.Contains(t, config.Processing.Ignore, "*.cache")
		assert.Contains(t, config.Processing.IncludeOnly, "*.go")
		assert.Contains(t, config.Processing.IncludeOnly, "*.py")
		assert.Equal(t, "https://custom.gitlab.com", config.GitLab.BaseURL)
	})

	t.Run("should not override empty CLI options", func(t *testing.T) {
		config := &models.Config{
			Output: models.OutputConfig{
				Directory: "./default-output",
			},
		}

		cliOptions := &models.CLIOptions{
			Output: "", // Empty should not override
		}

		err := loader.OverrideWithFlags(config, cliOptions)
		require.NoError(t, err)

		assert.Equal(t, "./default-output", config.Output.Directory)
	})
}

func TestLoader_ValidateConfig(t *testing.T) {
	loader := NewLoader()

	t.Run("should validate valid config", func(t *testing.T) {
		config := &models.Config{
			Output: models.OutputConfig{
				Directory: "./valid-output",
			},
			GitLab: models.GitLabConfig{
				TokenEnv: "GITLAB_TOKEN",
			},
			GitHub: models.GitHubConfig{
				TokenEnv: "GITHUB_TOKEN",
			},
			Processing: models.ProcessingConfig{
				MaxConcurrency: 1,
			},
		}

		err := loader.ValidateConfig(config)
		assert.NoError(t, err)
	})

	t.Run("should error on invalid max_concurrency", func(t *testing.T) {
		config := &models.Config{
			Output: models.OutputConfig{
				Directory: "./valid-output",
			},
			Processing: models.ProcessingConfig{
				MaxConcurrency: 0, // Invalid concurrency
			},
		}

		err := loader.ValidateConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "max_concurrency")
	})

	t.Run("should error on invalid file size", func(t *testing.T) {
		config := &models.Config{
			Output: models.OutputConfig{
				Directory: "./valid-output",
			},
			Processing: models.ProcessingConfig{
				MaxConcurrency: 1,
				MaxFileSize:    "invalid-size", // Invalid file size
			},
		}

		err := loader.ValidateConfig(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid max_file_size")
	})
}
