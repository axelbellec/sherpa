package llms

import (
	"testing"
	"time"

	"sherpa/pkg/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGenerator(t *testing.T) {
	t.Run("should create generator with full content enabled", func(t *testing.T) {
		generator := NewGenerator(true)
		assert.NotNil(t, generator)
		assert.True(t, generator.includeFullContent)
	})

	t.Run("should create generator with full content disabled", func(t *testing.T) {
		generator := NewGenerator(false)
		assert.NotNil(t, generator)
		assert.False(t, generator.includeFullContent)
	})
}

func TestGenerator_GenerateOutput(t *testing.T) {
	generator := NewGenerator(true)

	t.Run("should generate output from processing result", func(t *testing.T) {
		result := &models.ProcessingResult{
			Repository: models.Repository{
				Name:              "test-repo",
				PathWithNamespace: "owner/test-repo",
				Description:       "Test repository",
				Platform:          models.PlatformGitHub,
			},
			Files: []models.FileInfo{
				{
					Path:     "README.md",
					Name:     "README.md",
					Content:  "# Test Repository",
					Size:     16,
					IsText:   true,
					IsBinary: false,
				},
				{
					Path:     "src/main.go",
					Name:     "main.go",
					Content:  "package main\n\nfunc main() {}",
					Size:     26,
					IsText:   true,
					IsBinary: false,
				},
			},
			TotalFiles: 2,
			TotalSize:  42,
			Duration:   time.Second * 5,
			Errors:     nil,
		}

		output, err := generator.GenerateOutput(result)
		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Equal(t, result.Repository, output.Repository)
		assert.Equal(t, result.Files, output.FileContents)
		assert.Equal(t, result.TotalFiles, output.TotalFiles)
		assert.Equal(t, result.TotalSize, output.TotalSize)
		assert.NotEmpty(t, output.ProjectTree)
	})

	t.Run("should handle empty result", func(t *testing.T) {
		result := &models.ProcessingResult{
			Repository: models.Repository{
				Name:     "empty-repo",
				Platform: models.PlatformGitLab,
			},
			Files:      []models.FileInfo{},
			TotalFiles: 0,
			TotalSize:  0,
			Duration:   time.Millisecond * 100,
		}

		output, err := generator.GenerateOutput(result)
		require.NoError(t, err)
		assert.NotNil(t, output)
		assert.Empty(t, output.FileContents)
		assert.Equal(t, 0, output.TotalFiles)
		assert.Equal(t, int64(0), output.TotalSize)
	})
}

func TestGenerator_GenerateLLMsText(t *testing.T) {
	generator := NewGenerator(true)

	t.Run("should generate llms.txt content", func(t *testing.T) {
		output := &models.LLMsOutput{
			Repository: models.Repository{
				Name:              "test-repo",
				PathWithNamespace: "owner/test-repo",
				Description:       "Test repository",
				Platform:          models.PlatformGitHub,
			},
			FileContents: []models.FileInfo{
				{
					Path:    "README.md",
					Name:    "README.md",
					Content: "# Test Repository\nThis is a test.",
					Size:    25,
					IsText:  true,
				},
				{
					Path:    "src/main.go",
					Name:    "main.go",
					Content: "package main\n\nfunc main() {\n\tfmt.Println(\"Hello\")\n}",
					Size:    48,
					IsText:  true,
				},
			},
			TotalFiles: 2,
			TotalSize:  73,
		}

		text := generator.GenerateLLMsText(output)
		assert.NotEmpty(t, text)
		
		// Check header
		assert.Contains(t, text, "test-repo")
		assert.Contains(t, text, "Test repository")
		
		// Check tree structure
		assert.Contains(t, text, "## Project Structure")
	})
}

func TestGenerator_GenerateLLMsFullText(t *testing.T) {
	generator := NewGenerator(true)

	t.Run("should generate llms-full.txt content", func(t *testing.T) {
		output := &models.LLMsOutput{
			Repository: models.Repository{
				Name:              "test-repo",
				PathWithNamespace: "owner/test-repo",
				Description:       "Test repository",
				Platform:          models.PlatformGitHub,
			},
			FileContents: []models.FileInfo{
				{
					Path:     "README.md",
					Name:     "README.md",
					Content:  "# Test Repository",
					Size:     16,
					IsText:   true,
					IsBinary: false,
				},
				{
					Path:     "binary.bin",
					Name:     "binary.bin",
					Content:  "",
					Size:     1024,
					IsText:   false,
					IsBinary: true,
				},
			},
			TotalFiles: 2,
			TotalSize:  1040,
		}

		text := generator.GenerateLLMsFullText(output)
		assert.NotEmpty(t, text)
		
		// Should include text files
		assert.Contains(t, text, "### README.md")
		assert.Contains(t, text, "# Test Repository")
	})
}

