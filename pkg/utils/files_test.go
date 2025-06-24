package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeRepoName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "should sanitize owner/repo format",
			input:    "owner/repo",
			expected: "owner_repo",
		},
		{
			name:     "should sanitize complex path",
			input:    "group/subgroup/project",
			expected: "group_subgroup_project",
		},
		{
			name:     "should handle special characters",
			input:    "owner/repo-name.with.dots",
			expected: "owner_repo-name.with.dots",
		},
		{
			name:     "should handle multiple slashes",
			input:    "group///subgroup//project",
			expected: "group___subgroup__project",
		},
		{
			name:     "should handle empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "should handle no slashes",
			input:    "simple-name",
			expected: "simple-name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeRepoName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "should format bytes",
			bytes:    500,
			expected: "500 B",
		},
		{
			name:     "should format kilobytes",
			bytes:    1536, // 1.5 KB
			expected: "1.5 KB",
		},
		{
			name:     "should format megabytes",
			bytes:    1572864, // 1.5 MB
			expected: "1.5 MB",
		},
		{
			name:     "should format gigabytes",
			bytes:    1610612736, // 1.5 GB
			expected: "1.5 GB",
		},
		{
			name:     "should handle zero bytes",
			bytes:    0,
			expected: "0 B",
		},
		{
			name:     "should handle exact kilobyte",
			bytes:    1024,
			expected: "1.0 KB",
		},
		{
			name:     "should handle exact megabyte",
			bytes:    1048576,
			expected: "1.0 MB",
		},
		{
			name:     "should handle exact gigabyte",
			bytes:    1073741824,
			expected: "1.0 GB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatBytes(tt.bytes)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    int64
		expectError bool
	}{
		{
			name:     "should parse bytes",
			input:    "500B",
			expected: 500,
		},
		{
			name:     "should parse bytes with space",
			input:    "500 B",
			expected: 500,
		},
		{
			name:     "should parse kilobytes",
			input:    "1KB",
			expected: 1024,
		},
		{
			name:     "should parse kilobytes with space",
			input:    "1 KB",
			expected: 1024,
		},
		{
			name:     "should parse megabytes",
			input:    "1MB",
			expected: 1048576,
		},
		{
			name:     "should parse megabytes with space",
			input:    "1 MB",
			expected: 1048576,
		},
		{
			name:     "should parse gigabytes",
			input:    "1GB",
			expected: 1073741824,
		},
		{
			name:     "should parse decimal values",
			input:    "1.5MB",
			expected: 1572864, // 1.5 * 1024 * 1024
		},
		{
			name:     "should handle lowercase units",
			input:    "1kb",
			expected: 1024,
		},
		{
			name:     "should handle mixed case units",
			input:    "1Mb",
			expected: 1048576,
		},
		{
			name:        "should error on invalid format",
			input:       "invalid",
			expectError: true,
		},
		{
			name:        "should error on unsupported unit",
			input:       "1TB",
			expectError: true,
		},
		{
			name:        "should error on empty string",
			input:       "",
			expectError: true,
		},
		{
			name:        "should error on negative values",
			input:       "-1MB",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseSize(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsTextFile(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{
			name:     "should detect text content",
			content:  "package main\n\nfunc main() {}",
			expected: true,
		},
		{
			name:     "should detect empty content as text",
			content:  "",
			expected: true,
		},
		{
			name:     "should detect content with newlines as text",
			content:  "line1\nline2\nline3",
			expected: true,
		},
		{
			name:     "should detect binary content with null bytes",
			content:  "binary\x00content",
			expected: false,
		},
		{
			name:     "should detect content with high non-printable ratio as binary",
			content:  "\x01\x02\x03\x04\x05text",
			expected: false,
		},
		{
			name:     "should handle content with some non-printable characters",
			content:  "text with \t tabs and \n newlines",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsTextFile(tt.content)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractFileName(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "should extract filename from simple path",
			path:     "file.txt",
			expected: "file.txt",
		},
		{
			name:     "should extract filename from nested path",
			path:     "path/to/file.go",
			expected: "file.go",
		},
		{
			name:     "should handle deep paths",
			path:     "very/deep/nested/path/file.js",
			expected: "file.js",
		},
		{
			name:     "should handle empty path",
			path:     "",
			expected: "",
		},
		{
			name:     "should handle root file",
			path:     "README",
			expected: "README",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractFileName(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}