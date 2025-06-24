package pipeline

import (
	"errors"
	"testing"
	"time"

	"sherpa/pkg/models"

	"github.com/stretchr/testify/assert"
)

func TestNewStatsCalculator(t *testing.T) {
	calculator := NewStatsCalculator()
	assert.NotNil(t, calculator)
}

func TestStatsCalculator_GetProcessingStats(t *testing.T) {
	calculator := NewStatsCalculator()

	// Test with basic processing result
	result := &models.ProcessingResult{
		TotalFiles: 3,
		TotalSize:  1024,
		Duration:   time.Second * 5,
		Files: []models.FileInfo{
			{Path: "main.go", Size: 512, IsText: true},
			{Path: "image.png", Size: 256, IsText: false},
			{Path: "readme.txt", Size: 256, IsText: true},
		},
		Errors: []error{errors.New("error1"), errors.New("error2")},
	}

	stats := calculator.GetProcessingStats(result)

	assert.Equal(t, 3, stats["total_files"])
	assert.Equal(t, int64(1024), stats["total_size"])
	assert.Equal(t, "1.0 KB", stats["total_size_human"])
	assert.Equal(t, "5s", stats["processing_duration"])
	assert.Equal(t, 2, stats["errors_count"])
	assert.Equal(t, int64(341), stats["avg_file_size"]) // 1024/3
	assert.Equal(t, "341 B", stats["avg_file_size_human"])
	assert.Equal(t, 2, stats["text_files"])
	assert.Equal(t, 1, stats["binary_files"])
}

func TestStatsCalculator_GetProcessingStats_EmptyResult(t *testing.T) {
	calculator := NewStatsCalculator()

	result := &models.ProcessingResult{
		TotalFiles: 0,
		TotalSize:  0,
		Duration:   time.Second,
		Files:      []models.FileInfo{},
		Errors:     []error{},
	}

	stats := calculator.GetProcessingStats(result)

	assert.Equal(t, 0, stats["total_files"])
	assert.Equal(t, int64(0), stats["total_size"])
	assert.Equal(t, "0 B", stats["total_size_human"])
	assert.Equal(t, "1s", stats["processing_duration"])
	assert.Equal(t, 0, stats["errors_count"])
	assert.Equal(t, int64(0), stats["avg_file_size"])
	assert.Equal(t, 0, stats["text_files"])
	assert.Equal(t, 0, stats["binary_files"])
}

func TestStatsCalculator_GetProcessingStats_OnlyTextFiles(t *testing.T) {
	calculator := NewStatsCalculator()

	result := &models.ProcessingResult{
		TotalFiles: 2,
		TotalSize:  500,
		Duration:   time.Millisecond * 500,
		Files: []models.FileInfo{
			{Path: "main.go", Size: 300, IsText: true},
			{Path: "readme.md", Size: 200, IsText: true},
		},
		Errors: []error{},
	}

	stats := calculator.GetProcessingStats(result)

	assert.Equal(t, 2, stats["total_files"])
	assert.Equal(t, int64(500), stats["total_size"])
	assert.Equal(t, int64(250), stats["avg_file_size"]) // 500/2
	assert.Equal(t, 2, stats["text_files"])
	assert.Equal(t, 0, stats["binary_files"])
	assert.Equal(t, 0, stats["errors_count"])
}

func TestStatsCalculator_GetProcessingStats_WithErrors(t *testing.T) {
	calculator := NewStatsCalculator()

	result := &models.ProcessingResult{
		TotalFiles: 1,
		TotalSize:  100,
		Duration:   time.Millisecond * 100,
		Files: []models.FileInfo{
			{Path: "test.txt", Size: 100, IsText: true},
		},
		Errors: []error{errors.New("error1"), errors.New("error2"), errors.New("error3")},
	}

	stats := calculator.GetProcessingStats(result)

	assert.Equal(t, 1, stats["total_files"])
	assert.Equal(t, int64(100), stats["total_size"])
	assert.Equal(t, 3, stats["errors_count"])
	assert.Equal(t, 1, stats["text_files"])
	assert.Equal(t, 0, stats["binary_files"])
}
