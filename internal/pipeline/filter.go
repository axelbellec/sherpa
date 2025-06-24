package pipeline

import (
	"sherpa/pkg/models"
	"sherpa/pkg/utils"
)

// FileFilter handles filtering files based on ignore and include patterns
type FileFilter struct {
	patternMatcher *utils.PatternMatcher
}

// NewFileFilter creates a new file filter
func NewFileFilter(ignorePatterns, includePatterns []string) *FileFilter {
	return &FileFilter{
		patternMatcher: utils.NewPatternMatcher(ignorePatterns, includePatterns),
	}
}

// FilterFiles applies ignore and include patterns to filter the file list
func (ff *FileFilter) FilterFiles(tree []models.RepositoryTree) []models.RepositoryTree {
	var filtered []models.RepositoryTree

	for _, file := range tree {
		// Apply ignore patterns
		if ff.patternMatcher.ShouldIgnore(file.Path) {
			continue
		}

		// For directories, include them if they contain any non-ignored files
		if file.Type == "tree" {
			// Always include directories to maintain tree structure
			// They will be used for building the project tree but not processed for content
			filtered = append(filtered, file)
			continue
		}

		// Apply include-only patterns for files
		if !ff.patternMatcher.ShouldInclude(file.Path) {
			continue
		}

		filtered = append(filtered, file)
	}

	return filtered
}

// SeparateFilesAndDirectories separates files from directories
func (ff *FileFilter) SeparateFilesAndDirectories(entries []models.RepositoryTree) (files, directories []models.RepositoryTree) {
	for _, entry := range entries {
		if entry.Type == "tree" {
			directories = append(directories, entry)
		} else {
			files = append(files, entry)
		}
	}
	return files, directories
}
