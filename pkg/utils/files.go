package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

// FormatBytes formats byte counts into human-readable strings
func FormatBytes(bytes int64) string {
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

// ParseSize parses size strings like "1MB", "500KB" into bytes
func ParseSize(sizeStr string) (int64, error) {
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

// ExtractFileName extracts the filename from a file path
func ExtractFileName(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
}

// IsTextFile determines if content is text-based
func IsTextFile(content string) bool {
	if len(content) == 0 {
		return true
	}
	
	// Check for null bytes (binary indicator)
	if strings.Contains(content, "\x00") {
		return false
	}
	
	// Check for high ratio of non-printable characters
	nonPrintable := 0
	for _, r := range content {
		if !unicode.IsPrint(r) && !unicode.IsSpace(r) {
			nonPrintable++
		}
	}
	
	// If more than 20% non-printable, consider it binary
	if len(content) > 0 && float64(nonPrintable)/float64(len(content)) > 0.2 {
		return false
	}
	
	return true
}

// SanitizeRepoName sanitizes repository names for use in filenames
func SanitizeRepoName(repoPath string) string {
	// Replace problematic characters with underscores
	sanitized := strings.ReplaceAll(repoPath, "/", "_")
	sanitized = strings.ReplaceAll(sanitized, "\\", "_")
	sanitized = strings.ReplaceAll(sanitized, ":", "_")
	sanitized = strings.ReplaceAll(sanitized, "*", "_")
	sanitized = strings.ReplaceAll(sanitized, "?", "_")
	sanitized = strings.ReplaceAll(sanitized, "<", "_")
	sanitized = strings.ReplaceAll(sanitized, ">", "_")
	sanitized = strings.ReplaceAll(sanitized, "|", "_")
	sanitized = strings.ReplaceAll(sanitized, "\"", "_")
	return sanitized
} 