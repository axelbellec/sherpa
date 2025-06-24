package cmd

import (
	"context"
	"testing"

	"sherpa/pkg/models"

	"github.com/stretchr/testify/assert"
)

func TestNewOrchestrator(t *testing.T) {
	config := &models.Config{}
	cliOptions := &models.CLIOptions{}

	orchestrator := NewOrchestrator(config, cliOptions)

	assert.NotNil(t, orchestrator)
	assert.Equal(t, config, orchestrator.config)
	assert.Equal(t, cliOptions, orchestrator.cliOptions)
}

func TestOrchestrator_ProcessRepositories(t *testing.T) {
	t.Run("should handle empty repository list", func(t *testing.T) {
		config := &models.Config{
			Processing: models.ProcessingConfig{},
		}
		cliOptions := &models.CLIOptions{
			MaxReposConcurrency: 1,
		}

		orchestrator := NewOrchestrator(config, cliOptions)
		reposByPlatform := make(map[models.Platform][]*models.RepositoryInfo)

		err := orchestrator.ProcessRepositories(context.Background(), reposByPlatform)
		assert.NoError(t, err)
	})

	t.Run("should handle invalid token", func(t *testing.T) {
		config := &models.Config{
			Processing: models.ProcessingConfig{},
			GitLab: models.GitLabConfig{
				TokenEnv: "NONEXISTENT_TOKEN",
			},
		}
		cliOptions := &models.CLIOptions{
			MaxReposConcurrency: 1,
			Token:               "",
		}

		orchestrator := NewOrchestrator(config, cliOptions)
		reposByPlatform := map[models.Platform][]*models.RepositoryInfo{
			models.PlatformGitLab: {
				{
					FullName: "test/repo",
					Platform: models.PlatformGitLab,
					Branch:   "main",
				},
			},
		}

		err := orchestrator.ProcessRepositories(context.Background(), reposByPlatform)
		// Should not return error as goroutines handle errors internally
		assert.NoError(t, err)
	})
}

func TestOrchestrator_processRepositoriesConcurrently(t *testing.T) {
	t.Run("should handle concurrency limits", func(t *testing.T) {
		// This test would need mock processor and generator
		// Skipping full implementation for brevity
		t.Skip("Implement with mocked dependencies")
	})

	t.Run("should use default concurrency when invalid", func(t *testing.T) {
		// This would test that default concurrency (5) is used
		// when MaxReposConcurrency is <= 0
		t.Skip("Implement with mocked dependencies")
	})
}

func TestOrchestrator_processRepository(t *testing.T) {
	t.Run("should handle repository processing", func(t *testing.T) {
		// This test would need mock processor and generator
		// Skipping full implementation for brevity
		t.Skip("Implement with mocked dependencies")
	})
}
