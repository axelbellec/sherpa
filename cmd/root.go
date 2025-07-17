package cmd

import (
	"context"
	"fmt"
	"strings"

	"sherpa/internal/adapters"
	"sherpa/internal/config"
	"sherpa/internal/orchestration"
	"sherpa/pkg/logger"
	"sherpa/pkg/models"

	"github.com/spf13/cobra"
)

var (
	// Version information
	Version = "0.0.1"

	// CLI flags
	token               string
	baseURL             string
	outputDir           string
	ignoreFlag          string
	includeOnly         string
	configFile          string
	verbose             bool
	quiet               bool
	defaultPlatform     string
	maxReposConcurrency int
	maxFilesConcurrency int
	maxMemoryPerFile    int64
	maxTotalMemory      int64
	maxFiles            int
	dryRun              bool
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:     "sherpa [repository...]",
	Short:   "Git Repository to LLMs Context Generator",
	Version: Version,
	Long: `Sherpa is a lightweight CLI tool that processes repositories from
GitLab, GitHub, and local folders, generating comprehensive llms-full.txt files for LLM context.

It helps developers quickly create LLM-readable context from internal
codebases for debugging and cross-project analysis.

Platform Detection:
  Sherpa automatically detects the platform based on the repository URL or path:
  - GitHub: https://github.com/owner/repo or owner/repo
  - GitLab: https://gitlab.com/owner/repo or bare repo names (default)
  - Local: /path/to/folder, ./relative/path, or ~/home/path

Branch Targeting:
  Specify a target branch using URL fragment syntax (#branch):
  - https://gitlab.com/owner/repo#develop
  - https://github.com/owner/repo#feature-branch
  - owner/repo#main
  
  If no branch is specified, the repository's default branch is used.
  Note: Branch targeting is not applicable to local folders.

Examples:
  # GitHub repositories
  sherpa https://github.com/owner/repo --token $GITHUB_TOKEN
  sherpa owner/repo --token $GITHUB_TOKEN

  # GitLab repositories  
  sherpa https://gitlab.com/owner/repo --token $GITLAB_TOKEN
  sherpa platform-api --token $GITLAB_TOKEN

  # Local folders
  sherpa /path/to/my/project
  sherpa ./src/backend
  sherpa ~/my-projects/frontend

  # Branch targeting
  sherpa owner/repo#feature-branch --token $GITHUB_TOKEN
  sherpa https://github.com/user/repo1#main https://gitlab.com/group/repo2#develop

  # Use default platform for owner/repo format
  sherpa owner/repo --default-platform github
  sherpa owner/repo --default-platform gitlab

  # Mixed platforms with environment tokens
  sherpa owner/repo platform-api ./local-project

  # Use configuration file
  sherpa platform-api --config .sherpa.yml

  # Specify output directory
  sherpa platform-api --token $GITLAB_TOKEN --output ./contexts

  # Use ignore patterns
  sherpa platform-api --token $GITLAB_TOKEN --ignore "*.test.go,vendor/,*.log"
  
  # Preview operations with dry run
  sherpa owner/repo --dry-run --token $GITHUB_TOKEN
  sherpa repo1 repo2 repo3 ./local-folder --dry-run --token $GITHUB_TOKEN`,
	Args: cobra.MinimumNArgs(1),
	RunE: runFetch,
}

func init() {
	// Flags for root command
	RootCmd.Flags().StringVarP(&token, "token", "t", "", "Personal access token for Git platform (required)")
	RootCmd.Flags().StringVar(&baseURL, "base-url", "", "Custom base URL for self-hosted instances")
	RootCmd.Flags().StringVarP(&outputDir, "output", "o", "./sherpa-output", "Output directory")
	RootCmd.Flags().StringVar(&ignoreFlag, "ignore", "", "Comma-separated ignore patterns")
	RootCmd.Flags().StringVar(&includeOnly, "include-only", "", "Include only matching patterns")
	RootCmd.Flags().StringVarP(&configFile, "config", "c", "", "Configuration file path")
	RootCmd.Flags().StringVar(&defaultPlatform, "default-platform", "", "Default platform for owner/repo format (github or gitlab)")
	RootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	RootCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Suppress progress output")
	RootCmd.Flags().IntVarP(&maxReposConcurrency, "max-repos-concurrency", "m", 5, "Maximum number of repositories to process concurrently")
	RootCmd.Flags().IntVar(&maxFilesConcurrency, "max-files-concurrency", 20, "Maximum number of files to process concurrently per repository")
	RootCmd.Flags().Int64Var(&maxMemoryPerFile, "max-memory-per-file", 50*1024*1024, "Maximum memory per file in bytes (default: 50MB)")
	RootCmd.Flags().Int64Var(&maxTotalMemory, "max-total-memory", 2*1024*1024*1024, "Maximum total memory in bytes (default: 2GB)")
	RootCmd.Flags().IntVar(&maxFiles, "max-files", 1000, "Maximum number of files to process")
	RootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview operations without making API calls or creating files")
}

// runFetch executes the fetch command
func runFetch(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// Configure logging based on flags
	if quiet {
		logger.SetQuiet()
	} else if verbose {
		logger.SetVerbose()
	}

	logger.Logger.Info("Starting sherpa operation")

	// Create CLI options from flags
	cliOptions := &models.CLIOptions{
		Token:               token,
		BaseURL:             baseURL,
		Output:              outputDir,
		Ignore:              ignoreFlag,
		IncludeOnly:         includeOnly,
		ConfigFile:          configFile,
		DefaultPlatform:     defaultPlatform,
		MaxReposConcurrency: maxReposConcurrency,
		MaxFilesConcurrency: maxFilesConcurrency,
		MaxMemoryPerFile:    maxMemoryPerFile,
		MaxTotalMemory:      maxTotalMemory,
		MaxFiles:            maxFiles,
		Verbose:             verbose,
		Quiet:               quiet,
		DryRun:              dryRun,
	}

	// Load and configure
	configLoader := config.NewLoader()
	config, err := configLoader.LoadConfig(configFile)
	if err != nil {
		logger.Logger.WithError(err).Error("Failed to load configuration")
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Override config with command line flags
	if err := configLoader.OverrideWithFlags(config, cliOptions); err != nil {
		logger.Logger.WithError(err).Error("Failed to process configuration")
		return fmt.Errorf("failed to process configuration: %w", err)
	}

	// Validate configuration
	if err := configLoader.ValidateConfig(config); err != nil {
		logger.Logger.WithError(err).Error("Configuration validation failed")
		return fmt.Errorf("configuration validation failed: %w", err)
	}

	// Parse and group repositories by platform
	reposByPlatform, err := parseRepositories(args, cliOptions.DefaultPlatform)
	if err != nil {
		logger.Logger.WithError(err).Error("Failed to parse repositories")
		return fmt.Errorf("failed to parse repositories: %w", err)
	}

	logger.Logger.Debug("Configuration loaded and repositories parsed successfully")

	// Create orchestrator and process repositories
	orchestrator := orchestration.NewOrchestrator(config, cliOptions)
	return orchestrator.ProcessRepositories(ctx, reposByPlatform)
}

// parseRepositories parses repository arguments and groups them by platform
func parseRepositories(args []string, defaultPlatformFlag string) (map[models.Platform][]*models.RepositoryInfo, error) {
	reposByPlatform := make(map[models.Platform][]*models.RepositoryInfo)

	// Parse the default platform from the flag
	var defaultPlatformEnum models.Platform
	switch strings.ToLower(defaultPlatformFlag) {
	case "github":
		defaultPlatformEnum = models.PlatformGitHub
	case "gitlab":
		defaultPlatformEnum = models.PlatformGitLab
	case "":
		// No default platform specified, use existing logic
		defaultPlatformEnum = ""
	default:
		return nil, fmt.Errorf("invalid default platform '%s'. Valid options: github, gitlab", defaultPlatformFlag)
	}

	for _, arg := range args {
		repoInfo, err := adapters.ParseRepositoryURL(arg, defaultPlatformEnum)
		if err != nil {
			return nil, fmt.Errorf("failed to parse repository '%s': %w", arg, err)
		}

		reposByPlatform[repoInfo.Platform] = append(reposByPlatform[repoInfo.Platform], repoInfo)
	}

	return reposByPlatform, nil
}
