package tree

import (
	"fmt"
	"sort"
	"strings"

	"sherpa/pkg/models"
	"sherpa/pkg/utils"
)

// Builder handles building project tree structures
type Builder struct{}

// NewBuilder creates a new tree builder
func NewBuilder() *Builder {
	return &Builder{}
}

// BuildProjectTree creates a hierarchical tree structure from files
func (b *Builder) BuildProjectTree(files []models.FileInfo) []models.TreeNode {
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
		
		b.addFileToTree(root, file)
	}
	
	// Sort children recursively
	b.sortTreeNodesRecursive(root)
	
	return root.Children
}

// addFileToTree adds a file to the tree structure
func (b *Builder) addFileToTree(root *models.TreeNode, file models.FileInfo) {
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

// sortTreeNodesRecursive recursively sorts tree nodes
func (b *Builder) sortTreeNodesRecursive(node *models.TreeNode) {
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
		b.sortTreeNodesRecursive(&node.Children[i])
	}
}

// WriteProjectTree writes the project tree in a simple format
func (b *Builder) WriteProjectTree(nodes []models.TreeNode, indent string) string {
	var sb strings.Builder
	b.writeProjectTreeRecursive(&sb, nodes, indent)
	return sb.String()
}

// writeProjectTreeRecursive recursively writes the project tree structure
func (b *Builder) writeProjectTreeRecursive(sb *strings.Builder, nodes []models.TreeNode, indent string) {
	for _, node := range nodes {
		if node.IsDir {
			sb.WriteString(fmt.Sprintf("%s%s/\n", indent, node.Name))
			b.writeProjectTreeRecursive(sb, node.Children, indent+"  ")
		} else {
			sb.WriteString(fmt.Sprintf("%s%s (%s)\n", indent, node.Name, utils.FormatBytes(node.Size)))
		}
	}
}

// WriteProjectTreeUnix writes the project tree in Unix tree format
func (b *Builder) WriteProjectTreeUnix(nodes []models.TreeNode) string {
	var sb strings.Builder
	sb.WriteString(".\n")
	b.writeProjectTreeUnixRecursive(&sb, nodes, "", true)
	
	// Count directories and files
	dirCount, fileCount := b.countDirectoriesAndFiles(nodes)
	sb.WriteString(fmt.Sprintf("\n%d directories, %d files\n", dirCount, fileCount))
	
	return sb.String()
}

// writeProjectTreeUnixRecursive recursively writes the Unix-style tree structure
func (b *Builder) writeProjectTreeUnixRecursive(sb *strings.Builder, nodes []models.TreeNode, prefix string, isLast bool) {
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
				b.writeProjectTreeUnixRecursive(sb, node.Children, nextPrefix, false)
			}
		} else {
			sb.WriteString(fmt.Sprintf("%s%s\n", currentPrefix, node.Name))
		}
	}
}

// countDirectoriesAndFiles recursively counts directories and files in the tree
func (b *Builder) countDirectoriesAndFiles(nodes []models.TreeNode) (dirCount, fileCount int) {
	for _, node := range nodes {
		if node.IsDir {
			dirCount++
			childDirs, childFiles := b.countDirectoriesAndFiles(node.Children)
			dirCount += childDirs
			fileCount += childFiles
		} else {
			fileCount++
		}
	}
	return dirCount, fileCount
} 