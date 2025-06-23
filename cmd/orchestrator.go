package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"sherpa/internal/llms"
	"sherpa/internal/processor"
	"sherpa/internal/vcs"
	"sherpa/pkg/logger"
	"sherpa/pkg/models"
	"sherpa/pkg/utils"
)

// Orchestrator handles the coordination of repository processing across platforms
type Orchestrator struct {
	config     *models.Config
	cliOptions *models.CLIOptions
}

// NewOrchestrator creates a new orchestrator instance
func NewOrchestrator(config *models.Config, cliOptions *models.CLIOptions) *Orchestrator {
	return &Orchestrator{
		config:     config,
		cliOptions: cliOptions,
	}
}

// ProcessRepositories processes repositories grouped by platform
func (o *Orchestrator) ProcessRepositories(ctx context.Context, reposByPlatform map[models.Platform][]*models.RepositoryInfo) error {
	// Create LLMs generator
	logger.Logger.Debug("Creating LLMs generator")
	llmsGenerator := llms.NewGenerator(true)

	// Process repositories by platform
	totalRepos := 0
	for _, repos := range reposByPlatform {
		totalRepos += len(repos)
	}
	logger.Logger.WithField("total_repos", totalRepos).Info("Starting repository processing")

	// Process platforms concurrently
	var platformWg sync.WaitGroup
	var platformMu sync.Mutex // Protect stdout/stderr writes

	for platform, repoInfos := range reposByPlatform {
		platformWg.Add(1)

		go func(platform models.Platform, repoInfos []*models.RepositoryInfo) {
			defer platformWg.Done()

			logger.Logger.WithField("platform", platform).Info("Processing repositories for platform")

			// Get token for this platform
			platformToken, err := getTokenForPlatform(platform, o.config, o.cliOptions.Token)
			if err != nil {
				logger.Logger.WithError(err).WithField("platform", platform).Error("Failed to get token for platform")

				platformMu.Lock()
				fmt.Fprintf(os.Stderr, "Failed to get token for platform %s: %v\n", platform, err)
				platformMu.Unlock()
				return
			}

			// Create provider for this platform
			provider, err := vcs.CreateProvider(platform, o.config, platformToken)
			if err != nil {
				logger.Logger.WithError(err).WithField("platform", platform).Error("Failed to create provider")

				platformMu.Lock()
				fmt.Fprintf(os.Stderr, "Failed to create provider for platform %s: %v\n", platform, err)
				platformMu.Unlock()
				return
			}

			// Test connection
			logger.Logger.WithField("platform", platform).Info("Testing connection...")
			if err := provider.TestConnection(ctx); err != nil {
				logger.Logger.WithError(err).WithField("platform", platform).Error("Connection test failed")

				platformMu.Lock()
				fmt.Fprintf(os.Stderr, "Connection test failed for platform %s: %v\n", platform, err)
				platformMu.Unlock()
				return
			}
			logger.Logger.WithField("platform", platform).Info("Connection successful")

			// Create processor for this platform
			logger.Logger.Debug("Creating repository processor")
			repoProcessor := processor.NewRepoProcessor(provider, o.config.Processing)

			// Process repositories concurrently within this platform
			if err := o.processRepositoriesConcurrently(ctx, repoInfos, platform, repoProcessor, llmsGenerator, &platformMu); err != nil {
				logger.Logger.WithError(err).WithField("platform", platform).Error("Failed to process repositories concurrently")

				platformMu.Lock()
				fmt.Fprintf(os.Stderr, "Failed to process repositories for platform %s: %v\n", platform, err)
				platformMu.Unlock()
			}
		}(platform, repoInfos)
	}

	platformWg.Wait()

	logger.Logger.Info("Sherpa fetch operation completed successfully")
	return nil
}

// processRepositoriesConcurrently processes multiple repositories concurrently within a platform
func (o *Orchestrator) processRepositoriesConcurrently(
	ctx context.Context,
	repoInfos []*models.RepositoryInfo,
	platform models.Platform,
	repoProcessor *processor.RepoProcessor,
	llmsGenerator *llms.Generator,
	platformMu *sync.Mutex,
) error {
	maxConcurrency := o.cliOptions.MaxReposConcurrency
	if maxConcurrency <= 0 {
		maxConcurrency = 5 // Default concurrency
	}

	// Use a semaphore to limit concurrency
	semaphore := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	logger.Logger.WithFields(map[string]interface{}{
		"platform":         platform,
		"repository_count": len(repoInfos),
		"max_concurrency":  maxConcurrency,
	}).Info("Starting concurrent repository processing")

	for _, repoInfo := range repoInfos {
		wg.Add(1)

		go func(repoInfo *models.RepositoryInfo) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			o.processRepository(ctx, repoInfo, platform, repoProcessor, llmsGenerator, platformMu)
		}(repoInfo)
	}

	wg.Wait()
	return nil
}

// processRepository processes a single repository
func (o *Orchestrator) processRepository(
	ctx context.Context,
	repoInfo *models.RepositoryInfo,
	platform models.Platform,
	repoProcessor *processor.RepoProcessor,
	llmsGenerator *llms.Generator,
	platformMu *sync.Mutex,
) {
	repoPath := repoInfo.FullName
	logger.Logger.WithFields(map[string]interface{}{
		"repository": repoPath,
		"platform":   platform,
		"branch":     repoInfo.Branch,
	}).Info("Processing repository")

	// Process repository
	result, err := repoProcessor.ProcessRepository(ctx, repoPath, repoInfo.Branch)
	if err != nil {
		logger.Logger.WithError(err).WithFields(map[string]interface{}{
			"repository": repoPath,
			"platform":   platform,
		}).Error("Failed to process repository")

		platformMu.Lock()
		fmt.Fprintf(os.Stderr, "Failed to process repository %s: %v\n", repoPath, err)
		platformMu.Unlock()
		return
	}

	// Report any errors encountered during processing
	if len(result.Errors) > 0 {
		logger.Logger.WithField("error_count", len(result.Errors)).WithField("repository", repoPath).Warn("Encountered errors during processing")
		for _, e := range result.Errors {
			logger.Logger.WithError(e).Debug("Processing error")
		}
		if o.cliOptions.Verbose {
			platformMu.Lock()
			fmt.Printf("Encountered %d errors during processing:\n", len(result.Errors))
			for _, e := range result.Errors {
				fmt.Printf("  - %v\n", e)
			}
			platformMu.Unlock()
		}
	}

	// Generate LLMs output
	logger.Logger.WithField("repository", repoPath).Debug("Generating LLMs output")
	llmsOutput, err := llmsGenerator.GenerateOutput(result)
	if err != nil {
		logger.Logger.WithError(err).WithField("repository", repoPath).Error("Failed to generate LLMs output")

		platformMu.Lock()
		fmt.Fprintf(os.Stderr, "Failed to generate LLMs output for %s: %v\n", repoPath, err)
		platformMu.Unlock()
		return
	}

	// Create output directory
	repoOutputDir := filepath.Join(o.config.Output.Directory, utils.SanitizeRepoName(repoPath))
	if o.config.Output.OrganizeByDate {
		dateDir := time.Now().Format("2006-01-02")
		repoOutputDir = filepath.Join(o.config.Output.Directory, dateDir, utils.SanitizeRepoName(repoPath))
	}

	logger.Logger.WithField("output_dir", repoOutputDir).Debug("Creating output directory")
	if err := os.MkdirAll(repoOutputDir, 0755); err != nil {
		logger.Logger.WithError(err).WithField("output_dir", repoOutputDir).Error("Failed to create output directory")

		platformMu.Lock()
		fmt.Fprintf(os.Stderr, "Failed to create output directory %s: %v\n", repoOutputDir, err)
		platformMu.Unlock()
		return
	}

	// Generate and write files concurrently
	var fileWg sync.WaitGroup
	fileWg.Add(2)

	// Generate and write llms.txt
	go func() {
		defer fileWg.Done()

		logger.Logger.WithField("repository", repoPath).Debug("Generating llms.txt")
		llmsText := llmsGenerator.GenerateLLMsText(llmsOutput)
		llmsPath := filepath.Join(repoOutputDir, "llms.txt")
		if err := writeFile(llmsPath, llmsText); err != nil {
			logger.Logger.WithError(err).WithField("file", llmsPath).Error("Failed to write llms.txt")

			platformMu.Lock()
			fmt.Fprintf(os.Stderr, "Failed to write llms.txt for %s: %v\n", repoPath, err)
			platformMu.Unlock()
			return
		}
		logger.Logger.WithField("file", llmsPath).Debug("Successfully wrote llms.txt")
	}()

	// Generate and write llms-full.txt
	go func() {
		defer fileWg.Done()

		logger.Logger.WithField("repository", repoPath).Debug("Generating llms-full.txt")
		llmsFullText := llmsGenerator.GenerateLLMsFullText(llmsOutput)
		llmsFullPath := filepath.Join(repoOutputDir, "llms-full.txt")
		if err := writeFile(llmsFullPath, llmsFullText); err != nil {
			logger.Logger.WithError(err).WithField("file", llmsFullPath).Error("Failed to write llms-full.txt")

			platformMu.Lock()
			fmt.Fprintf(os.Stderr, "Failed to write llms-full.txt for %s: %v\n", repoPath, err)
			platformMu.Unlock()
			return
		}
		logger.Logger.WithField("file", llmsFullPath).Debug("Successfully wrote llms-full.txt")
	}()

	fileWg.Wait()

	// Success message
	logger.Logger.WithFields(map[string]interface{}{
		"repository":      repoPath,
		"platform":        platform,
		"files_processed": result.TotalFiles,
		"total_size":      utils.FormatBytes(result.TotalSize),
		"duration":        result.Duration.Round(time.Millisecond),
		"output_dir":      repoOutputDir,
	}).Info("Successfully processed repository")

	if !o.cliOptions.Quiet {
		platformMu.Lock()
		fmt.Printf("✓ Successfully processed %s (%s)\n", repoPath, platform)
		fmt.Printf("  Files processed: %d\n", result.TotalFiles)
		fmt.Printf("  Total size: %s\n", utils.FormatBytes(result.TotalSize))
		fmt.Printf("  Duration: %s\n", result.Duration.Round(time.Millisecond))
		fmt.Printf("  Output: %s\n", repoOutputDir)
		fmt.Println()
		platformMu.Unlock()
	}
}
