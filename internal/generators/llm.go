package generators

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"sherpa/pkg/models"
)

// Generator handles the generation of llms-full.txt files
type Generator struct {
	includeFullContent bool
}

// NewGenerator creates a new LLMs generator
func NewGenerator(includeFullContent bool) *Generator {
	return &Generator{
		includeFullContent: includeFullContent,
	}
}

// GenerateOutput generates the LLMs output from processing results
func (g *Generator) GenerateOutput(result *models.ProcessingResult) (*models.LLMsOutput, error) {
	// Build project tree
	projectTree := g.buildProjectTree(result.Files)

	// Prepare output structure
	output := &models.LLMsOutput{
		Repository:    result.Repository,
		GeneratedAt:   time.Now(),
		TotalFiles:    result.TotalFiles,
		TotalSize:     result.TotalSize,
		ProjectTree:   projectTree,
		ConfigFiles:   []models.FileInfo{},
		Documentation: []models.FileInfo{},
		FileContents:  result.Files,
	}

	return output, nil
}

// GenerateLLMsText generates the basic llms.txt content
func (g *Generator) GenerateLLMsText(output *models.LLMsOutput) string {
	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf("# Repository: %s\n", output.Repository.Name))
	sb.WriteString(fmt.Sprintf("# Generated: %s\n", output.GeneratedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("# Total Files: %d\n", output.TotalFiles))
	sb.WriteString(fmt.Sprintf("# Total Size: %s\n", formatBytes(output.TotalSize)))
	sb.WriteString("\n")

	// Repository information
	sb.WriteString("## Repository Information\n\n")
	sb.WriteString(fmt.Sprintf("**Name:** %s\n", output.Repository.Name))
	sb.WriteString(fmt.Sprintf("**Path:** %s\n", output.Repository.PathWithNamespace))
	sb.WriteString(fmt.Sprintf("**URL:** %s\n", output.Repository.WebURL))
	if output.Repository.Description != "" {
		sb.WriteString(fmt.Sprintf("**Description:** %s\n", output.Repository.Description))
	}
	sb.WriteString("\n")

	// Project Structure
	sb.WriteString("## Project Structure\n\n")
	g.writeProjectTreeUnix(&sb, output.ProjectTree)
	sb.WriteString("\n")

	return sb.String()
}

// GenerateLLMsTextWithoutUnixTree generates the basic llms.txt content with regular tree format
func (g *Generator) GenerateLLMsTextWithoutUnixTree(output *models.LLMsOutput) string {
	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf("# Repository: %s\n", output.Repository.Name))
	sb.WriteString(fmt.Sprintf("# Generated: %s\n", output.GeneratedAt.Format(time.RFC3339)))
	sb.WriteString(fmt.Sprintf("# Total Files: %d\n", output.TotalFiles))
	sb.WriteString(fmt.Sprintf("# Total Size: %s\n", formatBytes(output.TotalSize)))
	sb.WriteString("\n")

	// Repository information
	sb.WriteString("## Repository Information\n\n")
	sb.WriteString(fmt.Sprintf("**Name:** %s\n", output.Repository.Name))
	sb.WriteString(fmt.Sprintf("**Path:** %s\n", output.Repository.PathWithNamespace))
	sb.WriteString(fmt.Sprintf("**URL:** %s\n", output.Repository.WebURL))
	if output.Repository.Description != "" {
		sb.WriteString(fmt.Sprintf("**Description:** %s\n", output.Repository.Description))
	}
	sb.WriteString("\n")

	// Project Structure (regular format)
	sb.WriteString("## Project Structure\n\n")
	g.writeProjectTree(&sb, output.ProjectTree, "")
	sb.WriteString("\n")

	return sb.String()
}

// File size constants for security
const (
	MaxFileSize     = 5 * 1024 * 1024   // 5MB per file (increased from 1MB)
	MaxTotalSize    = 100 * 1024 * 1024  // 100MB total
	WarningFileSize = 1024 * 1024        // 1MB warning threshold
)

// GenerateLLMsFullText generates the complete llms-full.txt content with file contents
func (g *Generator) GenerateLLMsFullText(output *models.LLMsOutput) string {
	var sb strings.Builder

	// Validate total file size before processing
	if err := g.validateFileSize(output.FileContents); err != nil {
		sb.WriteString(fmt.Sprintf("## Error: %s\n\n", err.Error()))
		return sb.String()
	}

	// Include basic structure but with regular tree format (not Unix tree)
	sb.WriteString(g.GenerateLLMsTextWithoutUnixTree(output))

	// Add file contents section
	sb.WriteString("## File Contents\n\n")

	// Sort files by category and name
	sortedFiles := g.sortFilesByImportance(output.FileContents)

	for _, file := range sortedFiles {
		// Skip directories in the file contents section
		if file.IsDir {
			continue
		}

		// Skip binary files
		if file.IsBinary {
			continue
		}

		// Skip files with errors
		if file.Error != nil {
			continue
		}

		// Skip very large files (>5MB)
		if file.Size > MaxFileSize {
			sb.WriteString(fmt.Sprintf("### %s\n", file.Path))
			sb.WriteString(fmt.Sprintf("```\n[File too large to include - %s (max: %s)]\n```\n\n", formatBytes(file.Size), formatBytes(MaxFileSize)))
			continue
		}

		// Add header with warning for large files
		if file.Size > WarningFileSize {
			sb.WriteString(fmt.Sprintf("### %s (Large file: %s)\n", file.Path, formatBytes(file.Size)))
		} else {
			sb.WriteString(fmt.Sprintf("### %s\n", file.Path))
		}

		// Determine file extension for syntax highlighting
		ext := strings.ToLower(filepath.Ext(file.Path))
		lang := g.getLanguageFromExtension(ext)

		sb.WriteString(fmt.Sprintf("```%s\n", lang))
		sb.WriteString(file.Content)
		if !strings.HasSuffix(file.Content, "\n") {
			sb.WriteString("\n")
		}
		sb.WriteString("```\n\n")
	}

	return sb.String()
}

// validateFileSize validates that files don't exceed size limits
func (g *Generator) validateFileSize(files []models.FileInfo) error {
	var totalSize int64
	
	for _, file := range files {
		// Skip directories
		if file.IsDir {
			continue
		}
		
		// Check individual file size
		if file.Size > MaxFileSize {
			return fmt.Errorf("file %s exceeds maximum size (%s > %s)", file.Path, formatBytes(file.Size), formatBytes(MaxFileSize))
		}
		
		totalSize += file.Size
		
		// Check total size
		if totalSize > MaxTotalSize {
			return fmt.Errorf("total file size exceeds limit (%s > %s)", formatBytes(totalSize), formatBytes(MaxTotalSize))
		}
	}
	
	return nil
}

// buildProjectTree creates a hierarchical tree structure
func (g *Generator) buildProjectTree(files []models.FileInfo) []models.TreeNode {
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
	g.sortTreeNodesRecursive(root)

	return root.Children
}

// sortTreeNodesRecursive recursively sorts tree nodes to match tree command output
func (g *Generator) sortTreeNodesRecursive(node *models.TreeNode) {
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
		g.sortTreeNodesRecursive(&node.Children[i])
	}
}

// writeProjectTree recursively writes the project tree structure
func (g *Generator) writeProjectTree(sb *strings.Builder, nodes []models.TreeNode, indent string) {
	for _, node := range nodes {
		if node.IsDir {
			sb.WriteString(fmt.Sprintf("%s%s/\n", indent, node.Name))
			g.writeProjectTree(sb, node.Children, indent+"  ")
		} else {
			sb.WriteString(fmt.Sprintf("%s%s (%s)\n", indent, node.Name, formatBytes(node.Size)))
		}
	}
}

// writeProjectTreeUnix writes the project tree in Unix tree format
func (g *Generator) writeProjectTreeUnix(sb *strings.Builder, nodes []models.TreeNode) {
	sb.WriteString(".\n")
	g.writeProjectTreeUnixRecursive(sb, nodes, "", true)

	// Count directories and files
	dirCount, fileCount := g.countDirectoriesAndFiles(nodes)
	sb.WriteString(fmt.Sprintf("\n%d directories, %d files\n", dirCount, fileCount))
}

// writeProjectTreeUnixRecursive recursively writes the Unix-style tree structure
func (g *Generator) writeProjectTreeUnixRecursive(sb *strings.Builder, nodes []models.TreeNode, prefix string, isLast bool) {
	for i, node := range nodes {
		isLastChild := i == len(nodes)-1

		// Choose the appropriate prefix
		var currentPrefix, nextPrefix string
		if isLastChild {
			currentPrefix = prefix + "└── "
			nextPrefix = prefix + "    "
		} else {
			currentPrefix = prefix + "├── "
			nextPrefix = prefix + "│   "
		}

		// Write the current node
		if node.IsDir {
			sb.WriteString(fmt.Sprintf("%s%s\n", currentPrefix, node.Name))
			// Recursively write children
			if len(node.Children) > 0 {
				g.writeProjectTreeUnixRecursive(sb, node.Children, nextPrefix, false)
			}
		} else {
			sb.WriteString(fmt.Sprintf("%s%s\n", currentPrefix, node.Name))
		}
	}
}

// countDirectoriesAndFiles recursively counts directories and files in the tree
func (g *Generator) countDirectoriesAndFiles(nodes []models.TreeNode) (dirCount, fileCount int) {
	for _, node := range nodes {
		if node.IsDir {
			dirCount++
			childDirs, childFiles := g.countDirectoriesAndFiles(node.Children)
			dirCount += childDirs
			fileCount += childFiles
		} else {
			fileCount++
		}
	}
	return dirCount, fileCount
}

// sortFilesByImportance sorts files by importance for inclusion in full text
func (g *Generator) sortFilesByImportance(files []models.FileInfo) []models.FileInfo {
	// Create a copy to avoid modifying the original
	sorted := make([]models.FileInfo, len(files))
	copy(sorted, files)

	// Sort by category priority and then by name
	sort.Slice(sorted, func(i, j int) bool {
		iPriority := g.getFilePriority(sorted[i])
		jPriority := g.getFilePriority(sorted[j])

		if iPriority != jPriority {
			return iPriority < jPriority
		}

		return sorted[i].Path < sorted[j].Path
	})

	return sorted
}

// getFilePriority returns priority order for file inclusion (lower = higher priority)
func (g *Generator) getFilePriority(file models.FileInfo) int {
	fileName := strings.ToLower(filepath.Base(file.Path))
	filePath := strings.ToLower(file.Path)

	// Highest priority: main files and entry points
	if strings.Contains(fileName, "main") || strings.Contains(fileName, "index") {
		return 1
	}

	// High priority: configuration files
	configExts := []string{".json", ".yaml", ".yml", ".toml", ".env"}
	for _, ext := range configExts {
		if strings.HasSuffix(fileName, ext) {
			return 2
		}
	}

	// Medium-high priority: documentation
	if strings.HasSuffix(fileName, ".md") || strings.HasPrefix(fileName, "readme") {
		return 3
	}

	// Medium priority: source code files
	codeExts := []string{".go", ".py", ".js", ".ts", ".java", ".c", ".cpp", ".rs", ".rb"}
	for _, ext := range codeExts {
		if strings.HasSuffix(fileName, ext) {
			return 4
		}
	}

	// Lower priority: test files
	if strings.Contains(filePath, "test") || strings.Contains(fileName, "spec") {
		return 6
	}

	// Lowest priority: everything else
	return 5
}

// getLanguageFromExtension returns the language identifier for syntax highlighting
func (g *Generator) getLanguageFromExtension(ext string) string {
	languageMap := map[string]string{
		".go":         "go",
		".py":         "python",
		".js":         "javascript",
		".ts":         "typescript",
		".jsx":        "jsx",
		".tsx":        "tsx",
		".java":       "java",
		".c":          "c",
		".cpp":        "cpp",
		".cxx":        "cpp",
		".cc":         "cpp",
		".h":          "c",
		".hpp":        "cpp",
		".cs":         "csharp",
		".php":        "php",
		".rb":         "ruby",
		".rs":         "rust",
		".swift":      "swift",
		".kt":         "kotlin",
		".scala":      "scala",
		".sh":         "bash",
		".bash":       "bash",
		".zsh":        "zsh",
		".fish":       "fish",
		".ps1":        "powershell",
		".sql":        "sql",
		".html":       "html",
		".htm":        "html",
		".xml":        "xml",
		".css":        "css",
		".scss":       "scss",
		".sass":       "sass",
		".less":       "less",
		".json":       "json",
		".yaml":       "yaml",
		".yml":        "yaml",
		".toml":       "toml",
		".ini":        "ini",
		".cfg":        "ini",
		".conf":       "conf",
		".properties": "properties",
		".dockerfile": "dockerfile",
		".makefile":   "makefile",
		".mk":         "makefile",
		".cmake":      "cmake",
		".md":         "markdown",
		".rst":        "rst",
		".adoc":       "asciidoc",
		".tex":        "latex",
		".r":          "r",
		".m":          "matlab",
		".pl":         "perl",
		".lua":        "lua",
		".vim":        "vim",
		".el":         "elisp",
		".clj":        "clojure",
		".hs":         "haskell",
		".ml":         "ocaml",
		".fs":         "fsharp",
		".ex":         "elixir",
		".exs":        "elixir",
		".erl":        "erlang",
		".dart":       "dart",
	}

	if lang, exists := languageMap[ext]; exists {
		return lang
	}

	// Default to no language specification
	return ""
}

// Helper function to format bytes
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
