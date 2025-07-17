package models

import (
	"time"
)

// Config represents the complete configuration for Sherpa
type Config struct {
	GitLab     GitLabConfig     `yaml:"gitlab"`
	GitHub     GitHubConfig     `yaml:"github"`
	Processing ProcessingConfig `yaml:"processing"`
	Output     OutputConfig     `yaml:"output"`
	Cache      CacheConfig      `yaml:"cache"`
}

// GitLabConfig contains GitLab connection settings
type GitLabConfig struct {
	BaseURL  string `yaml:"base_url"`
	TokenEnv string `yaml:"token_env"`
}

// GitHubConfig contains GitHub connection settings
type GitHubConfig struct {
	BaseURL  string `yaml:"base_url"`
	TokenEnv string `yaml:"token_env"`
}

// ProcessingConfig contains file processing settings
type ProcessingConfig struct {
	Ignore           []string `yaml:"ignore"`
	IncludeOnly      []string `yaml:"include_only"`
	MaxFileSize      string   `yaml:"max_file_size"`
	SkipBinary       bool     `yaml:"skip_binary"`
	MaxConcurrency   int      `yaml:"max_concurrency"`
	MaxMemoryPerFile int64    `yaml:"max_memory_per_file"` // Maximum memory per file in bytes
	MaxTotalMemory   int64    `yaml:"max_total_memory"`    // Maximum total memory in bytes
	MaxFiles         int      `yaml:"max_files"`           // Maximum number of files to process
}

// OutputConfig contains output generation settings
type OutputConfig struct {
	Directory      string `yaml:"directory"`
	OrganizeByDate bool   `yaml:"organize_by_date"`
}

// CacheConfig contains caching settings
type CacheConfig struct {
	Enabled   bool          `yaml:"enabled"`
	Directory string        `yaml:"directory"`
	TTL       time.Duration `yaml:"ttl"`
}

// Platform represents the VCS platform type
type Platform string

const (
	PlatformGitLab Platform = "gitlab"
	PlatformGitHub Platform = "github"
	PlatformLocal  Platform = "local"
)

// Repository represents a VCS repository
type Repository struct {
	ID                interface{} `json:"id"` // int for GitLab, int64 for GitHub
	Name              string      `json:"name"`
	Path              string      `json:"path"`
	PathWithNamespace string      `json:"path_with_namespace"`
	WebURL            string      `json:"web_url"`
	Description       string      `json:"description"`
	Platform          Platform    `json:"platform"`
	Owner             string      `json:"owner"`
}

// RepositoryTree represents the tree structure of a repository
type RepositoryTree struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
	Path string `json:"path"`
	Mode string `json:"mode"`
}

// FileInfo contains information about a file in the repository
type FileInfo struct {
	Path     string
	Name     string
	Size     int64
	Content  string
	IsText   bool
	IsBinary bool
	IsDir    bool
	Error    error
}

// ProcessingResult contains the result of processing a repository
type ProcessingResult struct {
	Repository  Repository
	Files       []FileInfo
	TotalFiles  int
	TotalSize   int64
	ProcessedAt time.Time
	Duration    time.Duration
	Errors      []error
}

// LLMsOutput represents the structure for generating llms.txt files
type LLMsOutput struct {
	Repository    Repository
	GeneratedAt   time.Time
	TotalFiles    int
	TotalSize     int64
	ProjectTree   []TreeNode
	ConfigFiles   []FileInfo
	Documentation []FileInfo
	FileContents  []FileInfo
}

// TreeNode represents a node in the project tree structure
type TreeNode struct {
	Name     string
	Path     string
	Size     int64
	IsDir    bool
	Children []TreeNode
}

// RepositoryInfo contains parsed repository information
type RepositoryInfo struct {
	Platform Platform
	Owner    string
	Name     string
	FullName string // owner/repo format
	URL      string // original URL if provided
	Branch   string // target branch, empty means default branch
}

// CLIOptions contains command-line options
type CLIOptions struct {
	Token               string
	BaseURL             string
	Output              string
	Ignore              string
	IncludeOnly         string
	ConfigFile          string
	DefaultPlatform     string
	MaxReposConcurrency int
	MaxFilesConcurrency int
	MaxMemoryPerFile    int64
	MaxTotalMemory      int64
	MaxFiles            int
	Verbose             bool
	Quiet               bool
	DryRun              bool
}
