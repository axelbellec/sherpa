package tree

import (
	"testing"

	"sherpa/pkg/models"

	"github.com/stretchr/testify/assert"
)

func TestNewBuilder(t *testing.T) {
	builder := NewBuilder()
	assert.NotNil(t, builder)
}

func TestBuilder_BuildProjectTree(t *testing.T) {
	builder := NewBuilder()

	t.Run("should build tree from file list", func(t *testing.T) {
		files := []models.FileInfo{
			{Path: "README.md", Size: 100},
			{Path: "src/main.go", Size: 200},
			{Path: "src/utils/helper.go", Size: 150},
			{Path: "docs/api.md", Size: 300},
			{Path: "docs/guide.md", Size: 250},
		}

		nodes := builder.BuildProjectTree(files)
		assert.NotEmpty(t, nodes)
		assert.Len(t, nodes, 3) // README.md, src/, docs/
	})

	t.Run("should handle empty file list", func(t *testing.T) {
		files := []models.FileInfo{}
		nodes := builder.BuildProjectTree(files)
		assert.Empty(t, nodes)
	})

	t.Run("should handle single file", func(t *testing.T) {
		files := []models.FileInfo{
			{Path: "single.txt", Size: 50},
		}
		
		nodes := builder.BuildProjectTree(files)
		assert.Len(t, nodes, 1)
		assert.Equal(t, "single.txt", nodes[0].Name)
		assert.False(t, nodes[0].IsDir)
		assert.Equal(t, int64(50), nodes[0].Size)
	})

	t.Run("should handle nested directories", func(t *testing.T) {
		files := []models.FileInfo{
			{Path: "a/b/c/deep.txt", Size: 100},
			{Path: "a/b/shallow.txt", Size: 200},
			{Path: "a/file.txt", Size: 150},
		}
		
		nodes := builder.BuildProjectTree(files)
		assert.Len(t, nodes, 1) // Only 'a' directory at root
		
		aNode := nodes[0]
		assert.Equal(t, "a", aNode.Name)
		assert.True(t, aNode.IsDir)
		assert.Len(t, aNode.Children, 2) // b/ and file.txt
	})

	t.Run("should handle files with same prefix", func(t *testing.T) {
		files := []models.FileInfo{
			{Path: "test.go", Size: 100},
			{Path: "test_helper.go", Size: 150},
			{Path: "test/unit.go", Size: 200},
		}
		
		nodes := builder.BuildProjectTree(files)
		assert.Len(t, nodes, 3) // test.go, test_helper.go, test/
	})
}

func TestBuilder_WriteProjectTreeUnix(t *testing.T) {
	builder := NewBuilder()

	t.Run("should build structured tree representation", func(t *testing.T) {
		files := []models.FileInfo{
			{Path: "package.json", Size: 100},
			{Path: "src/index.js", Size: 200},
			{Path: "src/components/Header.js", Size: 150},
			{Path: "src/components/Footer.js", Size: 120},
			{Path: "tests/unit.test.js", Size: 300},
		}

		nodes := builder.BuildProjectTree(files)
		tree := builder.WriteProjectTreeUnix(nodes)
		
		// Should start with root
		assert.Contains(t, tree, ".")
		
		// Should contain all files and directories
		assert.Contains(t, tree, "package.json")
		assert.Contains(t, tree, "src")
		assert.Contains(t, tree, "index.js")
		assert.Contains(t, tree, "components")
		assert.Contains(t, tree, "Header.js")
		assert.Contains(t, tree, "Footer.js")
		assert.Contains(t, tree, "tests")
		assert.Contains(t, tree, "unit.test.js")
		assert.Contains(t, tree, "directories")
		assert.Contains(t, tree, "files")
	})

	t.Run("should handle root level files only", func(t *testing.T) {
		files := []models.FileInfo{
			{Path: "README.md", Size: 100},
			{Path: "LICENSE", Size: 50},
			{Path: ".gitignore", Size: 25},
		}

		nodes := builder.BuildProjectTree(files)
		tree := builder.WriteProjectTreeUnix(nodes)
		assert.Contains(t, tree, "README.md")
		assert.Contains(t, tree, "LICENSE")
		assert.Contains(t, tree, ".gitignore")
	})
}

func TestBuilder_WriteProjectTree(t *testing.T) {
	builder := NewBuilder()

	t.Run("should write simple project tree", func(t *testing.T) {
		files := []models.FileInfo{
			{Path: "README.md", Size: 100},
			{Path: "src/main.go", Size: 200},
		}

		nodes := builder.BuildProjectTree(files)
		tree := builder.WriteProjectTree(nodes, "")
		
		assert.Contains(t, tree, "README.md")
		assert.Contains(t, tree, "src/")
		assert.Contains(t, tree, "main.go")
	})
}

// Helper function to split tree output into lines
func splitLines(tree string) []string {
	if tree == "" {
		return []string{}
	}
	
	lines := []string{}
	current := ""
	for _, char := range tree {
		if char == '\n' {
			lines = append(lines, current)
			current = ""
		} else {
			current += string(char)
		}
	}
	if current != "" {
		lines = append(lines, current)
	}
	return lines
}