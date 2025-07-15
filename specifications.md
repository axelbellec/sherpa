# Sherpa - Multi-Platform Git Repository to LLMs.txt Generator Specification

| Code In, Context Out: Git-to-LLM Simplified

## Overview

**Sherpa** is a lightweight Go CLI tool designed to fetch private repositories from GitLab and GitHub, generating `llms.txt` and `llms-full.txt` files. It helps developers quickly create LLM-readable context from internal codebases for debugging and cross-project analysis.

## Purpose

- Generate LLM context from private repositories (GitLab and GitHub)
- Facilitate debugging across multiple internal projects
- Create standardized repository documentation for AI assistance
- Enable quick context sharing between team members
- Support both public and private repositories across platforms

## Core Features

### 1. Multi-Platform Integration

**GitLab Support:**

- Authenticate using GitLab personal access tokens
- Fetch repository contents via GitLab API (not git clone)
- Support for self-hosted GitLab instances
- Handle pagination for large repositories
- Respect rate limiting

**GitHub Support:**

- Authenticate using GitHub personal access tokens
- Fetch repository contents via GitHub API
- Support for GitHub Enterprise instances
- Handle GitHub API rate limiting
- Support for private repositories

**Automatic Platform Detection:**

- URL-based detection (https://github.com/owner/repo, https://gitlab.com/owner/repo)
- Format-based detection (owner/repo assumes GitHub, bare names assume GitLab)
- Support for SSH URLs (git@github.com:owner/repo.git)

### 2. High-Performance Concurrent Processing

**Multi-Level Concurrency:**

- **Platform-Level Concurrency**: Process multiple platforms (GitHub/GitLab) simultaneously
- **Repository-Level Concurrency**: Process multiple repositories within each platform concurrently
- **File-Level Concurrency**: Fetch multiple files per repository concurrently
- **Output Generation Concurrency**: Generate llms.txt and llms-full.txt files in parallel

**Configurable Concurrency Limits:**

- `--max-repos-concurrency` / `-m`: Control concurrent repository processing (default: 5)
- `--max-files-concurrency`: Control concurrent file fetching per repository (default: 20)
- Semaphore-based concurrency control to prevent resource exhaustion
- Intelligent worker pools for optimal performance

**Performance Optimizations:**

- Concurrent file fetching with worker pool
- Streaming for large files
- Intelligent caching of frequently accessed repos
- Batch API requests where possible
- Thread-safe output generation
- Mutex-protected console output for clean logging

### 3. File Processing

- Walk entire repository tree structure
- Identify files
- Apply ignore patterns (.gitignore + custom)
- Calculate file sizes and metadata
- Handle binary file detection
- Concurrent file content retrieval

### 4. LLMs.txt Generation

- **llms.txt**: Basic metadata and file listing
- **llms-full.txt**: Complete file contents included
- Follow official llms.txt specification
- Proper section headers and formatting
- Parallel generation of both output formats

### 5. CLI Interface

- Intuitive command structure
- Progress indicators for large repos
- Configurable output options
- Support for configuration files
- Advanced concurrency controls

## Technical Architecture

### Project Structure

```
sherpa/
├── cmd/
│   └── root.go          # Main CLI entry point with concurrent processing
├── internal/
│   ├── gitlab/
│   │   └── client.go    # GitLab API interactions with concurrency
│   ├── github/
│   │   └── client.go    # GitHub API interactions with concurrency
│   ├── vcs/
│   │   └── provider.go  # VCS provider abstraction & URL parsing
│   ├── processor/
│   │   └── repo.go      # Repository processing logic with concurrent file fetching
│   └── llms/
│       └── generator.go # LLMs.txt generation with parallel output
├── pkg/
│   ├── models/
│   │   └── types.go     # Shared data structures including concurrency config
│   └── logger/
│       └── logger.go    # Thread-safe logging utilities
├── go.mod
├── go.sum
└── main.go
```

### Dependencies

- `github.com/spf13/cobra` - CLI framework
- `gitlab.com/gitlab-org/api/client-go` - GitLab API client
- `github.com/google/go-github/v60` - GitHub API client
- `golang.org/x/oauth2` - OAuth2 authentication for GitHub
- `github.com/schollz/progressbar/v3` - Progress indication
- `gopkg.in/yaml.v3` - Configuration file parsing

## Usage Examples

### Basic Usage

```bash
# GitHub repositories (auto-detected)
sherpa fetch https://github.com/owner/repo --token $GITHUB_TOKEN
sherpa fetch owner/repo --token $GITHUB_TOKEN

# GitLab repositories (auto-detected)
sherpa fetch https://gitlab.com/owner/repo --token $GITLAB_TOKEN
sherpa fetch platform-api --token $GITLAB_TOKEN

# Self-hosted instances
sherpa fetch backend/auth-service --token $GITLAB_TOKEN --base-url https://gitlab.company.com
```

### Multi-Platform Usage

```bash
# Mixed platforms in single command (processed concurrently)
sherpa fetch owner/github-repo platform-api --token $GITHUB_TOKEN

# Platform-specific tokens via environment
export GITHUB_TOKEN=ghp_xxxx
export GITLAB_TOKEN=glpat_xxxx
sherpa fetch owner/repo gitlab-project

# SSH URLs supported
sherpa fetch git@github.com:owner/repo.git --token $GITHUB_TOKEN
```

### High-Performance Processing

```bash
# Process multiple repositories with custom concurrency
sherpa fetch repo1 repo2 repo3 repo4 repo5 \
  --max-repos-concurrency 10 \
  --max-files-concurrency 50 \
  --token $GITHUB_TOKEN

# Optimize for large repositories
sherpa fetch large-monorepo \
  --max-files-concurrency 100 \
  --token $GITHUB_TOKEN

# Process many small repositories quickly
sherpa fetch micro1 micro2 micro3 micro4 micro5 micro6 \
  --max-repos-concurrency 15 \
  --token $GITHUB_TOKEN
```

### Advanced Usage

```bash
# With ignore patterns
sherpa fetch services/payment --token $GITLAB_TOKEN \
  --ignore "*.test.go,vendor/,*.log"

# Include only specific files
sherpa fetch owner/frontend --token $GITHUB_TOKEN \
  --include-only "*.ts,*.tsx,*.md"

# Using configuration file
sherpa fetch api-gateway --token $GITLAB_TOKEN --config .sherpa.yml

# Fetch multiple repositories from different platforms with high concurrency
sherpa fetch owner/github-repo services/gitlab-auth services/gitlab-payment \
  --output ./debug-context \
  --max-repos-concurrency 8 \
  --max-files-concurrency 30
```

## Configuration

### CLI Flags

- `--token, -t` - Personal access token for Git platform (required for single-token usage)
- `--output, -o` - Output directory (default: `./sherpa-output`)
- `--base-url` - Custom base URL for self-hosted instances
- `--ignore` - Comma-separated ignore patterns
- `--include-only` - Include only matching patterns
- `--config, -c` - Configuration file path
- `--verbose, -v` - Verbose output
- `--quiet, -q` - Suppress progress output
- `--max-repos-concurrency, -m` - Maximum concurrent repositories (default: 5)
- `--max-files-concurrency` - Maximum concurrent file fetches per repo (default: 20)

### Token Management

**Single Token Mode:**
Use `--token` flag for all repositories (works if you have access to all platforms)

**Multi-Token Mode:**
Set platform-specific environment variables:

- `GITLAB_TOKEN` for GitLab repositories
- `GITHUB_TOKEN` for GitHub repositories

### Configuration File (.sherpa.yml)

```yaml
# GitLab settings
gitlab:
  base_url: https://gitlab.company.com
  token_env: GITLAB_TOKEN # Environment variable name

# GitHub settings
github:
  base_url: https://api.github.com
  token_env: GITHUB_TOKEN # Environment variable name

# File processing
processing:
  ignore:
    - "*.log"
    - "node_modules/"
    - "vendor/"
    - ".git/"
    - "coverage/"
  include_only:
    - "*.go"
    - "*.py"
    - "*.js"
    - "*.ts"
    - "*.md"
    - "*.yaml"
    - "*.yml"
    - "*.json"
  max_file_size: 10MB
  skip_binary: true
  max_concurrency: 20 # File fetching concurrency per repository

# Output settings
output:
  directory: "./sherpa-output"
  organize_by_date: true # Creates date-based subdirectories

# Cache settings
cache:
  enabled: true
  directory: "~/.sherpa/cache"
  ttl: 24h
```

## Output Format

### llms.txt Structure

```
# Repository: platform-api
# Generated: 2025-06-22T10:30:00Z
# Total Files: 156
# Total Size: 2.3MB

## Project Structure

src/
  main.go (1.2KB)
  config/
    config.go (890B)
    config_test.go (1.5KB)
  handlers/
    auth.go (3.2KB)
    users.go (2.8KB)
...

## Configuration Files

.gitlab-ci.yml
Dockerfile
go.mod
go.sum

## Documentation

README.md
docs/API.md
docs/SETUP.md
```

### llms-full.txt Structure

````
# Repository: platform-api
# Generated: 2025-06-22T10:30:00Z
# Full content dump with file contents

[... same header as llms.txt ...]

## File Contents

### src/main.go
```go
package main

import (
    "log"
    "net/http"
)

func main() {
    // ... full file content ...
}
````

[... continues with all text file contents ...]

```

## Performance Characteristics

### Concurrency Model

**Three-Tier Concurrency Architecture:**

1. **Platform Level**: Multiple Git platforms processed simultaneously
   - GitHub and GitLab repositories processed in parallel
   - Independent goroutines per platform
   - Platform-specific error isolation

2. **Repository Level**: Multiple repositories per platform processed concurrently
   - Configurable via `--max-repos-concurrency` flag
   - Semaphore-based limiting to prevent resource exhaustion
   - Default: 5 concurrent repositories per platform

3. **File Level**: Multiple files per repository fetched concurrently
   - Configurable via `--max-files-concurrency` flag
   - Enhanced from default 10 to 20 concurrent file fetches
   - Semaphore-controlled worker pools

### Performance Benchmarks

**Typical Performance Improvements:**
- **Multi-repository processing**: 3-5x faster with concurrent repository processing
- **File fetching**: 2-3x faster with increased file concurrency (10→20 default)
- **Platform processing**: 2x faster when processing both GitHub and GitLab repos
- **Output generation**: 1.5-2x faster with parallel llms.txt and llms-full.txt generation

**Scalability:**
- Supports processing 50+ repositories simultaneously
- Handles repositories with 1000+ files efficiently
- Memory usage scales linearly with concurrency settings
- CPU utilization optimized for I/O-bound operations

## Implementation Features

### Performance Optimizations
- **Multi-level concurrent processing** with configurable limits
- **Thread-safe output generation** with mutex-protected console writes
- **Parallel file generation** (llms.txt and llms-full.txt created simultaneously)
- **Semaphore-based resource management** to prevent system overload
- **Worker pool patterns** for efficient resource utilization
- Streaming for large files
- Intelligent caching of frequently accessed repos
- Batch API requests where possible

### Error Handling
- Graceful handling of API rate limits
- Clear error messages for authentication failures
- Network retry logic with exponential backoff
- Partial output on interruption
- **Concurrent error isolation** - failures in one repository don't affect others
- **Thread-safe error reporting** with synchronized console output

### Security Considerations
- Token stored in environment variables
- No tokens in configuration files
- Secure token handling in memory
- Optional token encryption at rest
- **Thread-safe token management** across concurrent operations

## Future Enhancements

### Phase 2 Features
- Integration with CI/CD pipelines
- Webhook support for automatic updates
- Team sharing capabilities
- Repository diff generation
- **Dynamic concurrency adjustment** based on system resources
- **Adaptive rate limiting** based on API response times

### Phase 3 Features
- Web UI for browsing generated contexts
- IDE plugins for quick context generation
- Custom LLM prompt templates
- Multi-repository relationship mapping
- **Distributed processing** across multiple machines
- **Smart caching** with dependency tracking

## Development Timeline

**Week 1: Core Implementation** ✅
- GitLab API client
- Basic repository fetching
- Simple llms.txt generation

**Week 2: Enhanced Features** ✅
- Configuration file support
- Advanced filtering options
- Progress indicators

**Week 3: Optimization** ✅
- **Multi-level concurrency implementation**
- **Performance improvements with configurable limits**
- **Thread-safe processing architecture**
- Error handling refinement

**Week 4: Polish & Release**
- Documentation updates
- Testing suite for concurrent operations
- Docker image
- Internal deployment guide

## Success Metrics

- **Time to generate context < 15 seconds for average repo** (improved from 30s with concurrency)
- **Support for repositories up to 5GB** (improved from 1GB with better memory management)
- **Process 10+ repositories simultaneously** without system overload
- Zero token exposure in logs or outputs
- **95% reduction in time** to set up LLM debugging context for multiple repositories
- **Thread-safe operation** with no race conditions or data corruption
- **Graceful resource management** under high concurrency loads
```
