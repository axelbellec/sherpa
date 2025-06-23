package utils

import (
	"path/filepath"
	"strings"
)

// PatternMatcher handles file pattern matching for ignore and include patterns
type PatternMatcher struct {
	ignorePatterns  []string
	includePatterns []string
}

// NewPatternMatcher creates a new pattern matcher
func NewPatternMatcher(ignorePatterns, includePatterns []string) *PatternMatcher {
	return &PatternMatcher{
		ignorePatterns:  ignorePatterns,
		includePatterns: includePatterns,
	}
}

// ShouldIgnore checks if a file should be ignored based on ignore patterns
func (pm *PatternMatcher) ShouldIgnore(filePath string) bool {
	for _, pattern := range pm.ignorePatterns {
		if pm.matchesPattern(filePath, pattern) {
			return true
		}
	}
	return false
}

// ShouldInclude checks if a file should be included based on include patterns
// Returns true if no include patterns are specified or if the file matches any include pattern
func (pm *PatternMatcher) ShouldInclude(filePath string) bool {
	if len(pm.includePatterns) == 0 {
		return true
	}
	
	for _, pattern := range pm.includePatterns {
		if pm.matchesPattern(filePath, pattern) {
			return true
		}
	}
	return false
}

// matchesPattern checks if a file path matches a pattern
func (pm *PatternMatcher) matchesPattern(filePath, pattern string) bool {
	// Handle glob patterns
	if matched, err := filepath.Match(pattern, filepath.Base(filePath)); err == nil && matched {
		return true
	}
	
	// Handle full path patterns
	if matched, err := filepath.Match(pattern, filePath); err == nil && matched {
		return true
	}
	
	// Handle directory patterns (ending with /)
	if strings.HasSuffix(pattern, "/") {
		dirPattern := strings.TrimSuffix(pattern, "/")
		if strings.Contains(filePath, dirPattern+"/") || strings.HasPrefix(filePath, dirPattern+"/") {
			return true
		}
	}
	
	// Handle substring matching for simple patterns
	if strings.Contains(filePath, pattern) {
		return true
	}
	
	return false
}

// ParsePatterns parses comma-separated pattern strings into slices
func ParsePatterns(patternStr string) []string {
	if patternStr == "" {
		return nil
	}
	
	patterns := strings.Split(patternStr, ",")
	var result []string
	
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern != "" {
			result = append(result, pattern)
		}
	}
	
	return result
} 