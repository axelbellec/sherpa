package cmd

import (
	"context"
	"fmt"
	"strings"

	"sherpa/internal/config"
	"sherpa/internal/adapters"
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
	dryRun              bool
)

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:     "sherpa",
	Short:   "Git Repository to LLMs Context Generator",
	Version: Version,
	Long: `Sherpa is a lightweight CLI tool that fetches private repositories from
GitLab and GitHub, generating comprehensive llms-full.txt files for LLM context.

It helps developers quickly create LLM-readable context from internal
codebases for debugging and cross-project analysis.

Supported platforms:
  - GitLab (gitlab.com and self-hosted)
  - GitHub (github.com and GitHub Enterprise)`,
}

// fetchCmd represents the fetch command
var fetchCmd = &cobra.Command{
	Use:   "fetch [repository...]",
	Short: "Fetch repository and generate LLMs context files",
	Long: `Fetch one or more repositories from GitLab or GitHub and generate comprehensive llms-full.txt files.

Platform Detection:
  Sherpa automatically detects the platform based on the repository URL:
  - GitHub: https://github.com/owner/repo or owner/repo
  - GitLab: https://gitlab.com/owner/repo or bare repo names (default)

Branch Targeting:
  Specify a target branch using URL fragment syntax (#branch):
  - https://gitlab.com/owner/repo#develop
  - https://github.com/owner/repo#feature-branch
  - owner/repo#main
  
  If no branch is specified, the repository's default branch is used.

Examples:
  # GitHub repositories
  sherpa fetch https://github.com/owner/repo --token $GITHUB_TOKEN
  sherpa fetch owner/repo --token $GITHUB_TOKEN

  # GitLab repositories  
  sherpa fetch https://gitlab.com/owner/repo --token $GITLAB_TOKEN
  sherpa fetch platform-api --token $GITLAB_TOKEN

  # Branch targeting
  sherpa fetch owner/repo#feature-branch --token $GITHUB_TOKEN
  sherpa fetch https://github.com/user/repo1#main https://gitlab.com/group/repo2#develop

  # Use default platform for owner/repo format
  sherpa fetch owner/repo --default-platform github
  sherpa fetch owner/repo --default-platform gitlab

  # Mixed platforms with environment tokens
  sherpa fetch owner/repo platform-api

  # Use configuration file
  sherpa fetch platform-api --config .sherpa.yml

  # Specify output directory
  sherpa fetch platform-api --token $GITLAB_TOKEN --output ./contexts

  # Use ignore patterns
  sherpa fetch platform-api --token $GITLAB_TOKEN --ignore "*.test.go,vendor/,*.log"
  
  # Preview operations with dry run
  sherpa fetch owner/repo --dry-run --token $GITHUB_TOKEN
  sherpa fetch repo1 repo2 repo3 --dry-run --token $GITHUB_TOKEN`,
	Args: cobra.MinimumNArgs(1),
	RunE: runFetch,
}

func init() {
	RootCmd.AddCommand(fetchCmd)

	// Persistent flags
	fetchCmd.Flags().StringVarP(&token, "token", "t", "", "Personal access token for Git platform (required)")
	fetchCmd.Flags().StringVar(&baseURL, "base-url", "", "Custom base URL for self-hosted instances")
	fetchCmd.Flags().StringVarP(&outputDir, "output", "o", "./sherpa-output", "Output directory")
	fetchCmd.Flags().StringVar(&ignoreFlag, "ignore", "", "Comma-separated ignore patterns")
	fetchCmd.Flags().StringVar(&includeOnly, "include-only", "", "Include only matching patterns")
	fetchCmd.Flags().StringVarP(&configFile, "config", "c", "", "Configuration file path")
	fetchCmd.Flags().StringVar(&defaultPlatform, "default-platform", "", "Default platform for owner/repo format (github or gitlab)")
	fetchCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")
	fetchCmd.Flags().BoolVarP(&quiet, "quiet", "q", false, "Suppress progress output")
	fetchCmd.Flags().IntVarP(&maxReposConcurrency, "max-repos-concurrency", "m", 5, "Maximum number of repositories to process concurrently")
	fetchCmd.Flags().IntVar(&maxFilesConcurrency, "max-files-concurrency", 20, "Maximum number of files to fetch concurrently per repository")
	fetchCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview operations without making API calls or creating files")
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

	logger.Logger.Info("Starting sherpa fetch operation")

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

