package processor

import (
	"testing"

	"sherpa/pkg/models"

	"github.com/stretchr/testify/assert"
)

func TestNewFileFilter(t *testing.T) {
	ignorePatterns := []string{"*.log", "*.tmp"}
	includePatterns := []string{"*.go", "*.py"}

	filter := NewFileFilter(ignorePatterns, includePatterns)
	assert.NotNil(t, filter)
	assert.NotNil(t, filter.patternMatcher)
}

func TestFileFilter_FilterFiles(t *testing.T) {
	tests := []struct {
		name            string
		ignorePatterns  []string
		includePatterns []string
		inputFiles      []models.RepositoryTree
		expectedFiles   []models.RepositoryTree
	}{
		{
			name:           "should filter out log files",
			ignorePatterns: []string{"*.log"},
			inputFiles: []models.RepositoryTree{
				{Path: "main.go", Type: "blob"},
				{Path: "app.log", Type: "blob"},
			},
			expectedFiles: []models.RepositoryTree{
				{Path: "main.go", Type: "blob"},
			},
		},
		{
			name:            "should include only specified patterns",
			includePatterns: []string{"*.go"},
			inputFiles: []models.RepositoryTree{
				{Path: "main.go", Type: "blob"},
				{Path: "readme.txt", Type: "blob"},
			},
			expectedFiles: []models.RepositoryTree{
				{Path: "main.go", Type: "blob"},
			},
		},
		{
			name: "should include directories",
			inputFiles: []models.RepositoryTree{
				{Path: "src", Type: "tree"},
				{Path: "main.go", Type: "blob"},
			},
			expectedFiles: []models.RepositoryTree{
				{Path: "src", Type: "tree"},
				{Path: "main.go", Type: "blob"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewFileFilter(tt.ignorePatterns, tt.includePatterns)
			result := filter.FilterFiles(tt.inputFiles)
			assert.Equal(t, tt.expectedFiles, result)
		})
	}
}

func TestFileFilter_SeparateFilesAndDirectories(t *testing.T) {
	filter := NewFileFilter(nil, nil)

	input := []models.RepositoryTree{
		{Path: "src", Type: "tree"},
		{Path: "main.go", Type: "blob"},
		{Path: "lib", Type: "tree"},
		{Path: "readme.txt", Type: "blob"},
	}

	files, directories := filter.SeparateFilesAndDirectories(input)

	expectedFiles := []models.RepositoryTree{
		{Path: "main.go", Type: "blob"},
		{Path: "readme.txt", Type: "blob"},
	}

	expectedDirectories := []models.RepositoryTree{
		{Path: "src", Type: "tree"},
		{Path: "lib", Type: "tree"},
	}

	assert.Equal(t, expectedFiles, files)
	assert.Equal(t, expectedDirectories, directories)
}

func TestFileFilter_PatternMatching(t *testing.T) {
	tests := []struct {
		name             string
		ignorePatterns   []string
		includePatterns  []string
		filePath         string
		fileType         string
		shouldBeFiltered bool
	}{
		{
			name:             "should filter out log files",
			ignorePatterns:   []string{"*.log"},
			filePath:         "app.log",
			fileType:         "blob",
			shouldBeFiltered: true,
		},
		{
			name:             "should not filter go files when not ignored",
			ignorePatterns:   []string{"*.log"},
			filePath:         "main.go",
			fileType:         "blob",
			shouldBeFiltered: false,
		},
		{
			name:             "should include only go files",
			includePatterns:  []string{"*.go"},
			filePath:         "main.go",
			fileType:         "blob",
			shouldBeFiltered: false,
		},
		{
			name:             "should filter out non-included files",
			includePatterns:  []string{"*.go"},
			filePath:         "readme.txt",
			fileType:         "blob",
			shouldBeFiltered: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := NewFileFilter(tt.ignorePatterns, tt.includePatterns)
			input := []models.RepositoryTree{{Path: tt.filePath, Type: tt.fileType}}
			result := filter.FilterFiles(input)

			if tt.shouldBeFiltered {
				assert.Empty(t, result, "Expected file to be filtered out")
			} else {
				assert.Len(t, result, 1, "Expected file to be included")
				assert.Equal(t, tt.filePath, result[0].Path)
			}
		})
	}
}
