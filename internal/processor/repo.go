package processor

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"sherpa/internal/vcs"
	"sherpa/pkg/logger"
	"sherpa/pkg/models"
)

// RepoProcessor handles repository processing logic
type RepoProcessor struct {
	provider vcs.Provider
	config   models.ProcessingConfig
}

// NewRepoProcessor creates a new repository processor
func NewRepoProcessor(provider vcs.Provider, config models.ProcessingConfig) *RepoProcessor {
	return &RepoProcessor{
		provider: provider,
		config:   config,
	}
}

// ProcessRepository processes a complete repository
func (rp *RepoProcessor) ProcessRepository(ctx context.Context, repoPath string) (*models.ProcessingResult, error) {
	logger.Logger.WithField("repository", repoPath).Info("Starting repository processing")
	startTime := time.Now()
	
	// Get repository information
	logger.Logger.WithField("repository", repoPath).Debug("Fetching repository information")
	repo, err := rp.provider.GetRepository(ctx, repoPath)
	if err != nil {
		logger.Logger.WithError(err).WithField("repository", repoPath).Error("Failed to get repository info")
		return nil, fmt.Errorf("failed to get repository info: %w", err)
	}

	// Get repository tree
	logger.Logger.WithField("repository", repoPath).Debug("Fetching repository tree")
	tree, err := rp.provider.GetRepositoryTree(ctx, repoPath)

	if err != nil {
		logger.Logger.WithError(err).WithField("repository", repoPath).Error("Failed to get repository tree")
		return nil, fmt.Errorf("failed to get repository tree: %w", err)
	}

	// Filter files based on ignore and include patterns
	logger.Logger.WithFields(map[string]interface{}{
		"repository":   repoPath,
		"total_files":  len(tree),
	}).Debug("Filtering files based on ignore and include patterns")
	filteredFiles := rp.filterFiles(tree)
	logger.Logger.WithFields(map[string]interface{}{
		"repository":      repoPath,
		"filtered_files":  len(filteredFiles),
		"original_files":  len(tree),
	}).Debug("Files filtered successfully")

	var processedFiles []models.FileInfo
	var totalSize int64
	var errors []error

	// Separate files from directories
	var fileEntries []models.RepositoryTree
	var directoryEntries []models.RepositoryTree
	
	for _, entry := range filteredFiles {
		if entry.Type == "tree" {
			directoryEntries = append(directoryEntries, entry)
		} else {
			fileEntries = append(fileEntries, entry)
		}
	}

	// Process files with concurrency control
	maxConcurrency := rp.config.MaxConcurrency
	if maxConcurrency <= 0 {
		maxConcurrency = 20 // Default increased from 10 to 20 for better performance
	}
	logger.Logger.WithFields(map[string]interface{}{
		"repository":       repoPath,
		"file_count":       len(fileEntries),
		"directory_count":  len(directoryEntries),
		"max_concurrency": maxConcurrency,
	}).Debug("Processing files with concurrency control")

	filePaths := make([]string, len(fileEntries))
	for i, file := range fileEntries {
		filePaths[i] = file.Path
	}

	files, err := rp.provider.GetMultipleFiles(ctx, repoPath, filePaths, maxConcurrency)
	if err != nil {
		logger.Logger.WithError(err).WithField("repository", repoPath).Error("Failed to fetch files")
		return nil, fmt.Errorf("failed to fetch files: %w", err)
	}

	// Process each file
	for _, file := range files {
		// Apply file size limit
		if rp.config.MaxFileSize != "" {
			maxSize, err := parseSize(rp.config.MaxFileSize)
			if err == nil && file.Size > maxSize {
				logger.Logger.WithField("file", file.Path).Debug("Skipping file because it's too large")
				continue
			}
		}

		// Skip binary files if configured
		if rp.config.SkipBinary && file.IsBinary {
			logger.Logger.WithField("file", file.Path).Debug("Skipping binary file")
			continue
		}

		// Collect errors but continue processing
		if file.Error != nil {
			logger.Logger.WithField("file", file.Path).Debug("Skipping file because it has an error")
			errors = append(errors, file.Error)
			continue
		}

		processedFiles = append(processedFiles, file)
		totalSize += file.Size
	}

	// Add directories as empty FileInfo entries for tree building
	for _, dir := range directoryEntries {
		dirInfo := models.FileInfo{
			Path:   dir.Path,
			Name:   dir.Name,
			IsDir:  true,
			Size:   0,
			IsText: false,
		}
		processedFiles = append(processedFiles, dirInfo)
	}

	duration := time.Since(startTime)

	logger.Logger.WithFields(map[string]interface{}{
		"repository":       repoPath,
		"total_files":      len(processedFiles),
		"total_size":       formatBytes(totalSize),
		"duration":         duration.Round(time.Millisecond),
		"error_count":      len(errors),
	}).Info("Repository processing completed")

	return &models.ProcessingResult{
		Repository:  *repo,
		Files:       processedFiles,
		TotalFiles:  len(processedFiles),
		TotalSize:   totalSize,
		ProcessedAt: startTime,
		Duration:    duration,
		Errors:      errors,
	}, nil
}

// filterFiles applies ignore and include patterns to filter the file list
func (rp *RepoProcessor) filterFiles(tree []models.RepositoryTree) []models.RepositoryTree {
	var filtered []models.RepositoryTree

	for _, file := range tree {
		// Apply ignore patterns
		if rp.shouldIgnore(file.Path) {
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
		if len(rp.config.IncludeOnly) > 0 && !rp.shouldInclude(file.Path) {
			continue
		}

		filtered = append(filtered, file)
	}

	return filtered
}

// shouldIgnore checks if a file should be ignored based on ignore patterns
func (rp *RepoProcessor) shouldIgnore(filePath string) bool {
	if len(rp.config.Ignore) == 0 {
		return false
	}

	for _, pattern := range rp.config.Ignore {
		if matched, _ := filepath.Match(pattern, filepath.Base(filePath)); matched {
			return true
		}
		
		// Check if pattern matches the full path
		if matched, _ := filepath.Match(pattern, filePath); matched {
			return true
		}
		
		// Check if it's a directory pattern
		if strings.HasSuffix(pattern, "/") {
			dirPattern := strings.TrimSuffix(pattern, "/")
			if strings.Contains(filePath, dirPattern+"/") {
				return true
			}
		}
	}

	return false
}

// shouldInclude checks if a file should be included based on include-only patterns
func (rp *RepoProcessor) shouldInclude(filePath string) bool {
	if len(rp.config.IncludeOnly) == 0 {
		return true
	}

	for _, pattern := range rp.config.IncludeOnly {
		if matched, _ := filepath.Match(pattern, filepath.Base(filePath)); matched {
			return true
		}
		
		// Check if pattern matches the full path
		if matched, _ := filepath.Match(pattern, filePath); matched {
			return true
		}
	}

	return false
}

// BuildProjectTree builds a hierarchical tree structure from flat file list
func (rp *RepoProcessor) BuildProjectTree(files []models.FileInfo) []models.TreeNode {
	if len(files) == 0 {
		return []models.TreeNode{}
	}
	
	root := &models.TreeNode{
		Name:     "",
		Path:     "",
		IsDir:    true,
		Children: []models.TreeNode{},
	}
	
	// Build the tree structure
	for _, file := range files {
		if file.Path == "" {
			continue
		}
		
		parts := strings.Split(file.Path, "/")
		current := root
		
		// Navigate/create path to file
		for i, part := range parts {
			isLastPart := i == len(parts)-1
			
			// Find existing child or create new one
			var found *models.TreeNode
			for j := range current.Children {
				if current.Children[j].Name == part {
					found = &current.Children[j]
					break
				}
			}
			
			if found == nil {
				// Create new node
				newNode := models.TreeNode{
					Name:  part,
					Path:  strings.Join(parts[:i+1], "/"),
					IsDir: !isLastPart || file.IsDir,
					Size:  0,
				}
				
				if isLastPart && !file.IsDir {
					newNode.Size = file.Size
				}
				
				current.Children = append(current.Children, newNode)
				found = &current.Children[len(current.Children)-1]
			} else if isLastPart && !file.IsDir {
				// Update existing node with file info
				found.Size = file.Size
				found.IsDir = false
			}
			
			current = found
		}
	}
	
	// Sort children recursively (directories first, then alphabetically)
	rp.sortTreeNodes(root)
	
	return root.Children
}

// sortTreeNodes recursively sorts tree nodes to match tree command output
func (rp *RepoProcessor) sortTreeNodes(node *models.TreeNode) {
	if len(node.Children) == 0 {
		return
	}
	
	// Sort directories first, then files, both alphabetically
	sort.Slice(node.Children, func(i, j int) bool {
		a, b := &node.Children[i], &node.Children[j]
		
		// Directories come before files
		if a.IsDir != b.IsDir {
			return a.IsDir
		}
		
		// Within same type, sort alphabetically
		return a.Name < b.Name
	})
	
	// Recursively sort children
	for i := range node.Children {
		rp.sortTreeNodes(&node.Children[i])
	}
}


// GetProcessingStats returns statistics about the processing
func (rp *RepoProcessor) GetProcessingStats(result *models.ProcessingResult) map[string]interface{} {
	stats := make(map[string]interface{})
	
	stats["total_files"] = result.TotalFiles
	stats["total_size"] = result.TotalSize
	stats["total_size_human"] = formatBytes(result.TotalSize)
	stats["processing_duration"] = result.Duration.String()
	stats["errors_count"] = len(result.Errors)
	stats["avg_file_size"] = int64(0)
	
	if result.TotalFiles > 0 {
		stats["avg_file_size"] = result.TotalSize / int64(result.TotalFiles)
		stats["avg_file_size_human"] = formatBytes(result.TotalSize / int64(result.TotalFiles))
	}
	
	// File type statistics
	var textFiles, binaryFiles int
	for _, file := range result.Files {
		if file.IsText {
			textFiles++
		} else {
			binaryFiles++
		}
	}
	
	stats["text_files"] = textFiles
	stats["binary_files"] = binaryFiles
	
	return stats
}

// Helper functions

func parseSize(sizeStr string) (int64, error) {
	sizeStr = strings.TrimSpace(strings.ToUpper(sizeStr))
	
	// Define size multipliers
	multipliers := map[string]int64{
		"B":  1,
		"KB": 1024,
		"MB": 1024 * 1024,
		"GB": 1024 * 1024 * 1024,
	}
	
	// Extract number and unit
	re := regexp.MustCompile(`^(\d+(?:\.\d+)?)\s*([KMGT]?B)$`)
	matches := re.FindStringSubmatch(sizeStr)
	
	if len(matches) != 3 {
		return 0, fmt.Errorf("invalid size format: %s", sizeStr)
	}
	
	size, err := strconv.ParseFloat(matches[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size number: %s", matches[1])
	}
	
	unit := matches[2]
	multiplier, exists := multipliers[unit]
	if !exists {
		return 0, fmt.Errorf("unknown size unit: %s", unit)
	}
	
	return int64(size * float64(multiplier)), nil
}

func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	
	units := []string{"KB", "MB", "GB", "TB"}
	return fmt.Sprintf("%.1f %s", float64(bytes)/float64(div), units[exp])
}