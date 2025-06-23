package processor

import (
	"sherpa/pkg/models"
	"sherpa/pkg/utils"
)

// StatsCalculator handles processing statistics calculation
type StatsCalculator struct{}

// NewStatsCalculator creates a new stats calculator
func NewStatsCalculator() *StatsCalculator {
	return &StatsCalculator{}
}

// GetProcessingStats returns statistics about the processing
func (sc *StatsCalculator) GetProcessingStats(result *models.ProcessingResult) map[string]interface{} {
	stats := make(map[string]interface{})
	
	stats["total_files"] = result.TotalFiles
	stats["total_size"] = result.TotalSize
	stats["total_size_human"] = utils.FormatBytes(result.TotalSize)
	stats["processing_duration"] = result.Duration.String()
	stats["errors_count"] = len(result.Errors)
	stats["avg_file_size"] = int64(0)
	
	if result.TotalFiles > 0 {
		stats["avg_file_size"] = result.TotalSize / int64(result.TotalFiles)
		stats["avg_file_size_human"] = utils.FormatBytes(result.TotalSize / int64(result.TotalFiles))
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