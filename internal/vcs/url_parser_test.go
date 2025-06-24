package vcs

import (
	"testing"

	"sherpa/pkg/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestURLParserParseRepositoryURL(t *testing.T) {
	tests := []struct {
		name            string
		url             string
		defaultPlatform models.Platform
		expectedRepo    *models.RepositoryInfo
		expectedError   bool
	}{
		{
			name: "should parse GitHub HTTPS URL",
			url:  "https://github.com/owner/repo",
			expectedRepo: &models.RepositoryInfo{
				FullName: "owner/repo",
				Owner:    "owner",
				Name:     "repo",
				Platform: models.PlatformGitHub,
				Branch:   "",
			},
			expectedError: false,
		},
		{
			name: "should parse GitHub HTTPS URL with branch",
			url:  "https://github.com/owner/repo#develop",
			expectedRepo: &models.RepositoryInfo{
				FullName: "owner/repo",
				Owner:    "owner",
				Name:     "repo",
				Platform: models.PlatformGitHub,
				Branch:   "develop",
			},
			expectedError: false,
		},
		{
			name: "should parse GitLab HTTPS URL",
			url:  "https://gitlab.com/group/project",
			expectedRepo: &models.RepositoryInfo{
				FullName: "group/project",
				Owner:    "group",
				Name:     "project",
				Platform: models.PlatformGitLab,
				Branch:   "",
			},
			expectedError: false,
		},
		{
			name: "should parse GitLab HTTPS URL with branch",
			url:  "https://gitlab.com/group/project#feature-branch",
			expectedRepo: &models.RepositoryInfo{
				FullName: "group/project",
				Owner:    "group",
				Name:     "project",
				Platform: models.PlatformGitLab,
				Branch:   "feature-branch",
			},
			expectedError: false,
		},
		{
			name:            "should parse owner/repo format with GitHub default",
			url:             "owner/repo",
			defaultPlatform: models.PlatformGitHub,
			expectedRepo: &models.RepositoryInfo{
				FullName: "owner/repo",
				Owner:    "owner",
				Name:     "repo",
				Platform: models.PlatformGitHub,
				Branch:   "",
			},
			expectedError: false,
		},
		{
			name:            "should parse owner/repo format with GitLab default",
			url:             "owner/repo",
			defaultPlatform: models.PlatformGitLab,
			expectedRepo: &models.RepositoryInfo{
				FullName: "owner/repo",
				Owner:    "owner",
				Name:     "repo",
				Platform: models.PlatformGitLab,
				Branch:   "",
			},
			expectedError: false,
		},
		{
			name:            "should parse owner/repo format with branch",
			url:             "owner/repo#main",
			defaultPlatform: models.PlatformGitHub,
			expectedRepo: &models.RepositoryInfo{
				FullName: "owner/repo",
				Owner:    "owner",
				Name:     "repo",
				Platform: models.PlatformGitHub,
				Branch:   "main",
			},
			expectedError: false,
		},
		{
			name: "should use GitHub as default when no platform specified",
			url:  "owner/repo",
			expectedRepo: &models.RepositoryInfo{
				FullName: "owner/repo",
				Owner:    "owner",
				Name:     "repo",
				Platform: models.PlatformGitHub,
				Branch:   "",
			},
			expectedError: false,
		},
		{
			name: "should handle self-hosted GitLab (detected by URL structure)",
			url:  "https://github.company.com/owner/repo",
			expectedRepo: &models.RepositoryInfo{
				FullName: "owner/repo",
				Owner:    "owner",
				Name:     "repo",
				Platform: models.PlatformGitLab,
				Branch:   "",
			},
			expectedError: false,
		},
		{
			name: "should handle self-hosted GitLab",
			url:  "https://gitlab.company.com/owner/repo",
			expectedRepo: &models.RepositoryInfo{
				FullName: "owner/repo",
				Owner:    "owner",
				Name:     "repo",
				Platform: models.PlatformGitLab,
				Branch:   "",
			},
			expectedError: false,
		},
		{
			name:          "should handle invalid URL as repo name",
			url:           "invalid-url",
			expectedRepo: &models.RepositoryInfo{
				FullName: "invalid-url",
				Owner:    "",
				Name:     "invalid-url",
				Platform: models.PlatformGitLab,
				Branch:   "",
			},
			expectedError: false,
		},
		{
			name:          "should error on missing repository path",
			url:           "https://github.com/",
			expectedError: true,
		},
		{
			name:          "should error on incomplete repository path",
			url:           "https://github.com/owner",
			expectedError: true,
		},
		{
			name:          "should handle empty URL as default repo",
			url:           "",
			expectedRepo: &models.RepositoryInfo{
				FullName: "",
				Owner:    "",
				Name:     "",
				Platform: models.PlatformGitLab,
				Branch:   "",
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewURLParser()
			result, err := parser.ParseRepositoryURL(tt.url, tt.defaultPlatform)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedRepo.FullName, result.FullName)
			assert.Equal(t, tt.expectedRepo.Owner, result.Owner)
			assert.Equal(t, tt.expectedRepo.Name, result.Name)
			assert.Equal(t, tt.expectedRepo.Platform, result.Platform)
			assert.Equal(t, tt.expectedRepo.Branch, result.Branch)
		})
	}
}

func TestDetectPlatformFromURL(t *testing.T) {
	tests := []struct {
		name             string
		url              string
		expectedPlatform models.Platform
		expectedError    bool
	}{
		{
			name:             "should detect GitHub",
			url:              "https://github.com/owner/repo",
			expectedPlatform: models.PlatformGitHub,
			expectedError:    false,
		},
		{
			name:             "should detect GitLab",
			url:              "https://gitlab.com/owner/repo",
			expectedPlatform: models.PlatformGitLab,
			expectedError:    false,
		},
		{
			name:             "should detect self-hosted GitLab (by default for unknown hosts)",
			url:              "https://github.enterprise.com/owner/repo",
			expectedPlatform: models.PlatformGitLab,
			expectedError:    false,
		},
		{
			name:             "should detect self-hosted GitLab",
			url:              "https://gitlab.enterprise.com/owner/repo",
			expectedPlatform: models.PlatformGitLab,
			expectedError:    false,
		},
		{
			name:             "should detect unknown platform as GitLab (default)",
			url:              "https://bitbucket.org/owner/repo",
			expectedPlatform: models.PlatformGitLab,
			expectedError:    false,
		},
		{
			name:             "should handle invalid URL with default platform (GitHub)",
			url:              "invalid-url",
			expectedPlatform: models.PlatformGitHub,
			expectedError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewURLParser()
			repoInfo, err := parser.ParseRepositoryURL(tt.url, models.PlatformGitHub)
			var platform models.Platform
			if err == nil && repoInfo != nil {
				platform = repoInfo.Platform
			}

			if tt.expectedError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedPlatform, platform)
		})
	}
}

func TestExtractRepositoryPath(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		expectedPath  string
		expectedError bool
	}{
		{
			name:          "should extract GitHub repository path",
			url:           "https://github.com/owner/repo",
			expectedPath:  "owner/repo",
			expectedError: false,
		},
		{
			name:          "should extract GitLab repository path",
			url:           "https://gitlab.com/group/project",
			expectedPath:  "group/project",
			expectedError: false,
		},
		{
			name:          "should handle trailing slash",
			url:           "https://github.com/owner/repo/",
			expectedPath:  "owner/repo",
			expectedError: false,
		},
		{
			name:          "should handle query parameters",
			url:           "https://github.com/owner/repo?tab=readme",
			expectedPath:  "owner/repo",
			expectedError: false,
		},
		{
			name:          "should handle fragment",
			url:           "https://github.com/owner/repo#readme",
			expectedPath:  "owner/repo",
			expectedError: false,
		},
		{
			name:          "should error on incomplete path",
			url:           "https://github.com/owner",
			expectedError: true,
		},
		{
			name:          "should error on empty path",
			url:           "https://github.com/",
			expectedError: true,
		},
		{
			name:          "should handle invalid URL as repo name",
			url:           "invalid-url",
			expectedPath:  "invalid-url",
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewURLParser()
			repoInfo, err := parser.ParseRepositoryURL(tt.url, models.PlatformGitHub)
			var path string
			if err == nil && repoInfo != nil {
				path = repoInfo.FullName
			}

			if tt.expectedError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedPath, path)
		})
	}
}

func TestParseOwnerRepo(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		expectedOwner string
		expectedRepo  string
		expectedError bool
	}{
		{
			name:          "should parse valid owner/repo",
			path:          "owner/repo",
			expectedOwner: "owner",
			expectedRepo:  "repo",
			expectedError: false,
		},
		{
			name:          "should parse with dashes",
			path:          "my-org/my-repo",
			expectedOwner: "my-org",
			expectedRepo:  "my-repo",
			expectedError: false,
		},
		{
			name:          "should parse with underscores",
			path:          "my_org/my_repo",
			expectedOwner: "my_org",
			expectedRepo:  "my_repo",
			expectedError: false,
		},
		{
			name:          "should handle invalid format as single repo name",
			path:          "invalid-format",
			expectedOwner: "",
			expectedRepo:  "invalid-format",
			expectedError: false,
		},
		{
			name:          "should handle empty path",
			path:          "",
			expectedOwner: "",
			expectedRepo:  "",
			expectedError: false,
		},
		{
			name:          "should handle too many parts as single repo name",
			path:          "owner/repo/extra",
			expectedOwner: "",
			expectedRepo:  "owner/repo/extra",
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewURLParser()
			repoInfo, err := parser.ParseRepositoryURL(tt.path, models.PlatformGitHub)
			var owner, repo string
			if err == nil && repoInfo != nil {
				owner = repoInfo.Owner
				repo = repoInfo.Name
			}

			if tt.expectedError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedOwner, owner)
			assert.Equal(t, tt.expectedRepo, repo)
		})
	}
}

func TestExtractBranch(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		expectedURL    string
		expectedBranch string
	}{
		{
			name:           "should extract branch from URL",
			url:            "https://github.com/owner/repo#develop",
			expectedURL:    "https://github.com/owner/repo",
			expectedBranch: "develop",
		},
		{
			name:           "should extract branch from short format",
			url:            "owner/repo#main",
			expectedURL:    "",
			expectedBranch: "main",
		},
		{
			name:           "should handle URL without branch",
			url:            "https://github.com/owner/repo",
			expectedURL:    "https://github.com/owner/repo",
			expectedBranch: "",
		},
		{
			name:           "should handle empty URL",
			url:            "",
			expectedURL:    "",
			expectedBranch: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewURLParser()
			repoInfo, err := parser.ParseRepositoryURL(tt.url, models.PlatformGitHub)
			var url, branch string
			if err == nil && repoInfo != nil {
				url = repoInfo.URL
				branch = repoInfo.Branch
			}
			assert.Equal(t, tt.expectedURL, url)
			assert.Equal(t, tt.expectedBranch, branch)
		})
	}
}

func TestParseRepositoryURL(t *testing.T) {
	tests := []struct {
		name            string
		url             string
		defaultPlatform models.Platform
		expectedRepo    *models.RepositoryInfo
		expectedError   bool
	}{
		{
			name: "should parse GitHub HTTPS URL",
			url:  "https://github.com/owner/repo",
			expectedRepo: &models.RepositoryInfo{
				FullName: "owner/repo",
				Owner:    "owner",
				Name:     "repo",
				Platform: models.PlatformGitHub,
				Branch:   "",
			},
			expectedError: false,
		},
		{
			name: "should parse owner/repo format",
			url:  "owner/repo",
			defaultPlatform: models.PlatformGitHub,
			expectedRepo: &models.RepositoryInfo{
				FullName: "owner/repo",
				Owner:    "owner",
				Name:     "repo",
				Platform: models.PlatformGitHub,
				Branch:   "",
			},
			expectedError: false,
		},
		{
			name:          "should handle empty URL by creating default repo info",
			url:           "",
			expectedRepo: &models.RepositoryInfo{
				FullName: "",
				Owner:    "",
				Name:     "",
				Platform: models.PlatformGitLab,
				Branch:   "",
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseRepositoryURL(tt.url, tt.defaultPlatform)

			if tt.expectedError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedRepo.FullName, result.FullName)
			assert.Equal(t, tt.expectedRepo.Owner, result.Owner)
			assert.Equal(t, tt.expectedRepo.Name, result.Name)
			assert.Equal(t, tt.expectedRepo.Platform, result.Platform)
			assert.Equal(t, tt.expectedRepo.Branch, result.Branch)
		})
	}
}
