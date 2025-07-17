package orchestration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"sherpa/internal/adapters"
	"sherpa/internal/generators"
	"sherpa/internal/pipeline"
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
	llmsGenerator := generators.NewGenerator(true)

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

			// Get token for this platform (skip for local platform)
			var platformToken string
			var err error
			if platform != models.PlatformLocal {
				platformToken, err = GetTokenForPlatform(platform, o.config, o.cliOptions.Token)
				if err != nil {
					logger.Logger.WithError(err).WithField("platform", platform).Error("Failed to get token for platform")

					platformMu.Lock()
					fmt.Fprintf(os.Stderr, "Failed to get token for platform %s: %v\n", platform, err)
					platformMu.Unlock()
					return
				}
			}

			// Create provider for this platform
			var provider adapters.Provider
			if platform == models.PlatformLocal {
				// For local platform, use the folder path from the first repository
				if len(repoInfos) > 0 {
					provider, err = adapters.CreateLocalProvider(repoInfos[0].FullName)
					if err != nil {
						logger.Logger.WithError(err).WithField("platform", platform).Error("Failed to create local provider")

						platformMu.Lock()
						fmt.Fprintf(os.Stderr, "Failed to create local provider for platform %s: %v\n", platform, err)
						platformMu.Unlock()
						return
					}
				} else {
					logger.Logger.WithField("platform", platform).Error("No repositories provided for local platform")
					platformMu.Lock()
					fmt.Fprintf(os.Stderr, "No repositories provided for local platform\n")
					platformMu.Unlock()
					return
				}
			} else {
				provider, err = adapters.CreateProvider(platform, o.config, platformToken)
				if err != nil {
					logger.Logger.WithError(err).WithField("platform", platform).Error("Failed to create provider")

					platformMu.Lock()
					fmt.Fprintf(os.Stderr, "Failed to create provider for platform %s: %v\n", platform, err)
					platformMu.Unlock()
					return
				}
			}

			// Test connection (skip in dry run mode)
			if !o.cliOptions.DryRun {
				logger.Logger.WithField("platform", platform).Info("Testing connection...")
				if err := provider.TestConnection(ctx); err != nil {
					logger.Logger.WithError(err).WithField("platform", platform).Error("Connection test failed")

					platformMu.Lock()
					fmt.Fprintf(os.Stderr, "Connection test failed for platform %s: %v\n", platform, err)
					platformMu.Unlock()
					return
				}
				logger.Logger.WithField("platform", platform).Info("Connection successful")
			} else {
				logger.Logger.WithField("platform", platform).Info("[DRY RUN] Skipping connection test")
			}

			// Create processor for this platform
			logger.Logger.Debug("Creating repository processor")
			repoProcessor := pipeline.NewRepoProcessor(provider, o.config.Processing)

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
	repoProcessor *pipeline.RepoProcessor,
	llmsGenerator *generators.Generator,
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
	repoProcessor *pipeline.RepoProcessor,
	llmsGenerator *generators.Generator,
	platformMu *sync.Mutex,
) {
	repoPath := repoInfo.FullName
	logger.Logger.WithFields(map[string]interface{}{
		"repository": repoPath,
		"platform":   platform,
		"branch":     repoInfo.Branch,
		"dry_run":    o.cliOptions.DryRun,
	}).Info("Processing repository")

	// Handle dry run mode
	if o.cliOptions.DryRun {
		o.processDryRun(ctx, repoInfo, platform, repoProcessor, platformMu)
		return
	}

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

	// Generate and write llms-full.txt
	logger.Logger.WithField("repository", repoPath).Debug("Generating llms-full.txt")
	llmsFullText := llmsGenerator.GenerateLLMsFullText(llmsOutput)
	llmsFullPath := filepath.Join(repoOutputDir, "llms-full.txt")
	if err := WriteFile(llmsFullPath, llmsFullText); err != nil {
		logger.Logger.WithError(err).WithField("file", llmsFullPath).Error("Failed to write llms-full.txt")

		platformMu.Lock()
		fmt.Fprintf(os.Stderr, "Failed to write llms-full.txt for %s: %v\n", repoPath, err)
		platformMu.Unlock()
		return
	}
	logger.Logger.WithField("file", llmsFullPath).Debug("Successfully wrote llms-full.txt")

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

// processDryRun handles dry run mode for a repository
func (o *Orchestrator) processDryRun(
	ctx context.Context,
	repoInfo *models.RepositoryInfo,
	platform models.Platform,
	repoProcessor *pipeline.RepoProcessor,
	platformMu *sync.Mutex,
) {
	_ = ctx           // unused in dry run mode
	_ = repoProcessor // unused in dry run mode
	repoPath := repoInfo.FullName
	logger.Logger.WithFields(map[string]interface{}{
		"repository": repoPath,
		"platform":   platform,
		"branch":     repoInfo.Branch,
	}).Info("[DRY RUN] Processing repository")

	// Simulate repository processing with mock data
	mockResult := o.simulateRepositoryProcessing(repoInfo, platform)

	// Calculate output directory
	repoOutputDir := filepath.Join(o.config.Output.Directory, utils.SanitizeRepoName(repoPath))
	if o.config.Output.OrganizeByDate {
		dateDir := time.Now().Format("2006-01-02")
		repoOutputDir = filepath.Join(o.config.Output.Directory, dateDir, utils.SanitizeRepoName(repoPath))
	}

	// Display dry run results
	if !o.cliOptions.Quiet {
		platformMu.Lock()
		fmt.Printf("[DRY RUN] Would process %s (%s)\n", repoPath, platform)
		fmt.Printf("  Branch: %s\n", repoInfo.Branch)
		fmt.Printf("  Estimated files: %d\n", mockResult.EstimatedFiles)
		fmt.Printf("  Estimated size: %s\n", mockResult.EstimatedSize)
		fmt.Printf("  Would create output: %s\n", repoOutputDir)
		fmt.Printf("  File that would be created:\n")
		fmt.Printf("    - %s/llms-full.txt\n", repoOutputDir)
		fmt.Println()
		platformMu.Unlock()
	}

	logger.Logger.WithFields(map[string]interface{}{
		"repository":      repoPath,
		"platform":        platform,
		"estimated_files": mockResult.EstimatedFiles,
		"estimated_size":  mockResult.EstimatedSize,
		"output_dir":      repoOutputDir,
	}).Info("[DRY RUN] Repository processing simulation completed")
}

// DryRunResult contains simulated processing results
type DryRunResult struct {
	EstimatedFiles int
	EstimatedSize  string
}

// simulateRepositoryProcessing simulates repository processing and returns mock results
func (o *Orchestrator) simulateRepositoryProcessing(repoInfo *models.RepositoryInfo, platform models.Platform) *DryRunResult {
	_ = repoInfo // unused for now, could be used for better estimates
	_ = platform // unused for now, could be used for platform-specific estimates
	// These are mock estimates - in a real implementation, you might want to
	// make lightweight API calls to get basic repo info without fetching all files
	estimatedFiles := 50     // Mock estimate
	estimatedSize := "2.5MB" // Mock estimate

	// You could potentially make a single API call to get repository metadata
	// without fetching the full tree or file contents to provide better estimates

	return &DryRunResult{
		EstimatedFiles: estimatedFiles,
		EstimatedSize:  estimatedSize,
	}
}

// GetTokenForPlatform gets the appropriate token for a platform
func GetTokenForPlatform(platform models.Platform, config *models.Config, cliToken string) (string, error) {
	// If a token was provided via CLI flag, use it for all platforms
	if cliToken != "" {
		return cliToken, nil
	}

	// Get platform-specific token from environment based on the detected platform
	switch platform {
	case models.PlatformGitLab:
		if envToken := os.Getenv(config.GitLab.TokenEnv); envToken != "" {
			return envToken, nil
		}
		return "", fmt.Errorf("GitLab token not found. Set %s environment variable or use --token flag", config.GitLab.TokenEnv)
	case models.PlatformGitHub:
		if envToken := os.Getenv(config.GitHub.TokenEnv); envToken != "" {
			return envToken, nil
		}
		return "", fmt.Errorf("GitHub token not found. Set %s environment variable or use --token flag", config.GitHub.TokenEnv)
	default:
		return "", fmt.Errorf("unsupported platform: %s", platform)
	}
}

// WriteFile writes content to a file
func WriteFile(path, content string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString(content)
	return err
}
