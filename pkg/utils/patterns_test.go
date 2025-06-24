package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParsePatterns(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "should parse comma-separated patterns",
			input:    "*.log,*.tmp,*.cache",
			expected: []string{"*.log", "*.tmp", "*.cache"},
		},
		{
			name:     "should handle spaces around commas",
			input:    "*.log, *.tmp , *.cache",
			expected: []string{"*.log", "*.tmp", "*.cache"},
		},
		{
			name:     "should handle single pattern",
			input:    "*.log",
			expected: []string{"*.log"},
		},
		{
			name:     "should handle empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "should handle patterns with paths",
			input:    "node_modules/,vendor/,*.log",
			expected: []string{"node_modules/", "vendor/", "*.log"},
		},
		{
			name:     "should filter out empty patterns",
			input:    "*.log,,*.tmp",
			expected: []string{"*.log", "*.tmp"},
		},
		{
			name:     "should handle trailing comma",
			input:    "*.log,*.tmp,",
			expected: []string{"*.log", "*.tmp"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParsePatterns(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPatternMatcher(t *testing.T) {
	t.Run("should handle ignore patterns", func(t *testing.T) {
		pm := NewPatternMatcher([]string{"*.log", "node_modules/"}, []string{})

		assert.True(t, pm.ShouldIgnore("app.log"))
		assert.True(t, pm.ShouldIgnore("node_modules/package/index.js"))
		assert.False(t, pm.ShouldIgnore("src/main.go"))
	})

	t.Run("should handle include patterns", func(t *testing.T) {
		pm := NewPatternMatcher([]string{}, []string{"*.go", "*.js"})

		assert.True(t, pm.ShouldInclude("main.go"))
		assert.True(t, pm.ShouldInclude("app.js"))
		assert.False(t, pm.ShouldInclude("README.md"))
	})

	t.Run("should include all when no patterns specified", func(t *testing.T) {
		pm := NewPatternMatcher([]string{}, []string{})

		assert.True(t, pm.ShouldInclude("any.file"))
		assert.False(t, pm.ShouldIgnore("any.file"))
	})
}
