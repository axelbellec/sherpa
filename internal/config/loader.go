package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
	"sherpa/pkg/models"
	"sherpa/pkg/utils"
)

// Loader handles configuration loading and validation
type Loader struct{}

// NewLoader creates a new configuration loader
func NewLoader() *Loader {
	return &Loader{}
}

// LoadConfig loads configuration from file or returns default config
func (l *Loader) LoadConfig(configFile string) (*models.Config, error) {
	config := l.getDefaultConfig()
	
	if configFile != "" {
		if _, err := os.Stat(configFile); err == nil {
			data, err := os.ReadFile(configFile)
			if err != nil {
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}
			
			if err := yaml.Unmarshal(data, config); err != nil {
				return nil, fmt.Errorf("failed to parse config file: %w", err)
			}
		}
	}
	
	return config, nil
}

// getDefaultConfig returns the default configuration
func (l *Loader) getDefaultConfig() *models.Config {
	return &models.Config{
		GitLab: models.GitLabConfig{
			BaseURL:  "https://gitlab.com",
			TokenEnv: "GITLAB_TOKEN",
		},
		GitHub: models.GitHubConfig{
			BaseURL:  "https://api.github.com",
			TokenEnv: "GITHUB_TOKEN",
		},
		Processing: models.ProcessingConfig{
			Ignore: []string{
				".git/",
				"node_modules/",
				"vendor/",
				"*.log",
				"*.tmp",
				".DS_Store",
			},
			IncludeOnly:    []string{},
			MaxFileSize:    "1MB",
			SkipBinary:     true,
			MaxConcurrency: 20,
		},
		Output: models.OutputConfig{
			Directory:      "./sherpa-output",
			OrganizeByDate: false,
		},
		Cache: models.CacheConfig{
			Enabled:   false,
			Directory: "./.sherpa-cache",
			TTL:       0,
		},
	}
}

// OverrideWithFlags overrides config values with command line flags
func (l *Loader) OverrideWithFlags(config *models.Config, flags *models.CLIOptions) error {
	if flags.BaseURL != "" {
		// Determine which platform to update based on the base URL
		if flags.BaseURL == "https://api.github.com" || flags.BaseURL == "https://github.com" {
			config.GitHub.BaseURL = flags.BaseURL
		} else {
			config.GitLab.BaseURL = flags.BaseURL
		}
	}
	
	if flags.Output != "" {
		config.Output.Directory = flags.Output
	}
	
	if flags.Ignore != "" {
		config.Processing.Ignore = utils.ParsePatterns(flags.Ignore)
	}
	
	if flags.IncludeOnly != "" {
		config.Processing.IncludeOnly = utils.ParsePatterns(flags.IncludeOnly)
	}
	
	return nil
}

// ValidateConfig validates the configuration
func (l *Loader) ValidateConfig(config *models.Config) error {
	if config.Processing.MaxConcurrency <= 0 {
		return fmt.Errorf("max_concurrency must be greater than 0")
	}
	
	if config.Processing.MaxFileSize != "" {
		if _, err := utils.ParseSize(config.Processing.MaxFileSize); err != nil {
			return fmt.Errorf("invalid max_file_size: %w", err)
		}
	}
	
	return nil
} 