package pipeline

import (
	"context"
	"testing"
	"time"

	"sherpa/pkg/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockProvider implements the Provider interface for testing
type MockProvider struct {
	mock.Mock
}

func (m *MockProvider) GetRepository(ctx context.Context, repoPath string) (*models.Repository, error) {
	args := m.Called(ctx, repoPath)
	return args.Get(0).(*models.Repository), args.Error(1)
}

func (m *MockProvider) GetRepositoryTree(ctx context.Context, repoPath, branch string) ([]models.RepositoryTree, error) {
	args := m.Called(ctx, repoPath, branch)
	return args.Get(0).([]models.RepositoryTree), args.Error(1)
}

func (m *MockProvider) GetFileContent(ctx context.Context, repoPath, filePath, branch string) (string, error) {
	args := m.Called(ctx, repoPath, filePath, branch)
	return args.String(0), args.Error(1)
}

func (m *MockProvider) GetFileInfo(ctx context.Context, repoPath, filePath, branch string) (*models.FileInfo, error) {
	args := m.Called(ctx, repoPath, filePath, branch)
	return args.Get(0).(*models.FileInfo), args.Error(1)
}

func (m *MockProvider) GetMultipleFiles(ctx context.Context, repoPath string, filePaths []string, branch string, maxConcurrency int, config *models.ProcessingConfig) ([]models.FileInfo, error) {
	args := m.Called(ctx, repoPath, filePaths, branch, maxConcurrency, config)
	return args.Get(0).([]models.FileInfo), args.Error(1)
}

func (m *MockProvider) TestConnection(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func TestNewRepoProcessor(t *testing.T) {
	mockProvider := &MockProvider{}
	config := models.ProcessingConfig{
		MaxConcurrency: 10,
	}

	processor := NewRepoProcessor(mockProvider, config)
	assert.NotNil(t, processor)
	assert.Equal(t, mockProvider, processor.provider)
	assert.Equal(t, config, processor.config)
}

func TestRepoProcessor_ProcessRepository(t *testing.T) {
	t.Run("should process repository successfully", func(t *testing.T) {
		mockProvider := &MockProvider{}
		config := models.ProcessingConfig{
			Ignore:         []string{"*.log"},
			MaxConcurrency: 5,
		}

		processor := NewRepoProcessor(mockProvider, config)

		// Mock repository
		repo := &models.Repository{
			ID:                123,
			Name:              "test-repo",
			PathWithNamespace: "owner/test-repo",
			Platform:          models.PlatformGitHub,
		}

		// Mock tree
		tree := []models.RepositoryTree{
			{
				ID:   "abc123",
				Name: "README.md",
				Path: "README.md",
				Type: "blob",
			},
			{
				ID:   "def456",
				Name: "main.go",
				Path: "src/main.go",
				Type: "blob",
			},
			{
				ID:   "ghi789",
				Name: "app.log",
				Path: "logs/app.log",
				Type: "blob",
			},
		}

		// Mock files (app.log should be filtered out)
		files := []models.FileInfo{
			{
				Path:    "README.md",
				Name:    "README.md",
				Content: "# Test Repository",
				Size:    16,
				IsText:  true,
			},
			{
				Path:    "src/main.go",
				Name:    "main.go",
				Content: "package main",
				Size:    12,
				IsText:  true,
			},
		}

		mockProvider.On("GetRepository", mock.Anything, "owner/test-repo").Return(repo, nil)
		mockProvider.On("GetRepositoryTree", mock.Anything, "owner/test-repo", "main").Return(tree, nil)
		mockProvider.On("GetMultipleFiles", mock.Anything, "owner/test-repo", []string{"README.md", "src/main.go"}, "main", 5, mock.Anything).Return(files, nil)

		result, err := processor.ProcessRepository(context.Background(), "owner/test-repo", "main")
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, *repo, result.Repository)
		assert.Len(t, result.Files, 2) // app.log should be filtered out
		assert.Equal(t, 2, result.TotalFiles)
		assert.Equal(t, int64(28), result.TotalSize)
		assert.Greater(t, result.Duration, time.Duration(0))

		mockProvider.AssertExpectations(t)
	})

	t.Run("should handle repository not found", func(t *testing.T) {
		mockProvider := &MockProvider{}
		config := models.ProcessingConfig{}
		processor := NewRepoProcessor(mockProvider, config)

		mockProvider.On("GetRepository", mock.Anything, "owner/nonexistent").Return((*models.Repository)(nil), assert.AnError)

		_, err := processor.ProcessRepository(context.Background(), "owner/nonexistent", "main")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get repository")

		mockProvider.AssertExpectations(t)
	})

	t.Run("should handle tree fetch error", func(t *testing.T) {
		mockProvider := &MockProvider{}
		config := models.ProcessingConfig{}
		processor := NewRepoProcessor(mockProvider, config)

		repo := &models.Repository{
			Name:              "test-repo",
			PathWithNamespace: "owner/test-repo",
		}

		mockProvider.On("GetRepository", mock.Anything, "owner/test-repo").Return(repo, nil)
		mockProvider.On("GetRepositoryTree", mock.Anything, "owner/test-repo", "main").Return([]models.RepositoryTree(nil), assert.AnError)

		_, err := processor.ProcessRepository(context.Background(), "owner/test-repo", "main")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get repository tree")

		mockProvider.AssertExpectations(t)
	})

	t.Run("should handle file fetch errors gracefully", func(t *testing.T) {
		mockProvider := &MockProvider{}
		config := models.ProcessingConfig{
			MaxConcurrency: 2,
		}
		processor := NewRepoProcessor(mockProvider, config)

		repo := &models.Repository{
			Name:              "test-repo",
			PathWithNamespace: "owner/test-repo",
		}

		tree := []models.RepositoryTree{
			{
				ID:   "abc123",
				Name: "README.md",
				Path: "README.md",
				Type: "blob",
			},
		}

		// Mock files with error
		files := []models.FileInfo{
			{
				Path:  "README.md",
				Name:  "README.md",
				Error: assert.AnError,
			},
		}

		mockProvider.On("GetRepository", mock.Anything, "owner/test-repo").Return(repo, nil)
		mockProvider.On("GetRepositoryTree", mock.Anything, "owner/test-repo", "main").Return(tree, nil)
		mockProvider.On("GetMultipleFiles", mock.Anything, "owner/test-repo", []string{"README.md"}, "main", 2, mock.Anything).Return(files, nil)

		result, err := processor.ProcessRepository(context.Background(), "owner/test-repo", "main")
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Errors, 1)       // Should capture file error
		assert.Equal(t, 0, result.TotalFiles) // File with error not counted
		assert.Equal(t, int64(0), result.TotalSize)

		mockProvider.AssertExpectations(t)
	})
}
