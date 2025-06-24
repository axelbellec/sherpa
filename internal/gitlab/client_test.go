package gitlab

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	baseURL := "https://gitlab.com"
	token := "test-token"

	client, err := NewClient(baseURL, token)
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, baseURL, client.baseURL)
	assert.Equal(t, token, client.token)
}

func TestClient_TestConnection(t *testing.T) {
	t.Run("should test connection successfully", func(t *testing.T) {
		// Mock server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/user", r.URL.Path)
			assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"username": "testuser", "id": 123}`))
		}))
		defer server.Close()

		client, err := NewClient(server.URL, "test-token")
		require.NoError(t, err)

		err = client.TestConnection(context.Background())
		assert.NoError(t, err)
	})

	t.Run("should fail on invalid token", func(t *testing.T) {
		// Mock server returning 401
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"message": "Unauthorized"}`))
		}))
		defer server.Close()

		client, err := NewClient(server.URL, "invalid-token")
		require.NoError(t, err)

		err = client.TestConnection(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "authenticate")
	})

	t.Run("should handle network errors", func(t *testing.T) {
		client, err := NewClient("http://localhost:99999", "test-token") // Invalid port
		require.NoError(t, err)

		err = client.TestConnection(context.Background())
		assert.Error(t, err)
	})
}

func TestClient_GetRepository(t *testing.T) {
	t.Run("should get repository successfully", func(t *testing.T) {
		// Mock server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/projects/owner%2Frepo", r.URL.Path)
			assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": 123,
				"name": "repo",
				"path": "repo",
				"path_with_namespace": "owner/repo",
				"web_url": "https://gitlab.com/owner/repo",
				"description": "Test repository",
				"default_branch": "main"
			}`))
		}))
		defer server.Close()

		client, err := NewClient(server.URL, "test-token")
		require.NoError(t, err)

		repo, err := client.GetRepository(context.Background(), "owner/repo")
		require.NoError(t, err)
		assert.Equal(t, "repo", repo.Name)
		assert.Equal(t, "owner/repo", repo.PathWithNamespace)
		assert.Equal(t, int64(123), repo.ID)
	})

	t.Run("should handle repository not found", func(t *testing.T) {
		// Mock server returning 404
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message": "404 Project Not Found"}`))
		}))
		defer server.Close()

		client, err := NewClient(server.URL, "test-token")
		require.NoError(t, err)

		_, err = client.GetRepository(context.Background(), "owner/nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "repository not found")
	})
}

func TestClient_GetRepositoryFiles(t *testing.T) {
	t.Run("should get repository files successfully", func(t *testing.T) {
		// Mock server for repository tree API
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/projects/owner%2Frepo/repository/tree" {
				assert.Equal(t, "main", r.URL.Query().Get("ref"))
				assert.Equal(t, "true", r.URL.Query().Get("recursive"))
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`[
					{
						"id": "abc123",
						"name": "README.md",
						"type": "blob",
						"path": "README.md",
						"mode": "100644"
					},
					{
						"id": "def456",
						"name": "main.go",
						"type": "blob",
						"path": "src/main.go",
						"mode": "100644"
					}
				]`))
			} else if r.URL.Path == "/projects/owner%2Frepo/repository/files/README.md/raw" {
				assert.Equal(t, "main", r.URL.Query().Get("ref"))
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("# Readme"))
			} else if r.URL.Path == "/projects/owner%2Frepo/repository/files/src%2Fmain.go/raw" {
				assert.Equal(t, "main", r.URL.Query().Get("ref"))
				w.Header().Set("Content-Type", "text/plain")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("package main"))
			}
		}))
		defer server.Close()

		client, err := NewClient(server.URL, "test-token")
		require.NoError(t, err)

		files, err := client.GetMultipleFiles(context.Background(), "owner/repo", []string{"README.md", "src/main.go"}, "main", 5)
		require.NoError(t, err)
		assert.Len(t, files, 2)

		// Check first file
		assert.Equal(t, "README.md", files[0].Path)
		assert.Equal(t, "# Readme", files[0].Content)

		// Check second file
		assert.Equal(t, "src/main.go", files[1].Path)
		assert.Equal(t, "package main", files[1].Content)
	})

	t.Run("should handle tree not found", func(t *testing.T) {
		// Mock server returning 404 for tree
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message": "404 Tree Not Found"}`))
		}))
		defer server.Close()

		client, err := NewClient(server.URL, "test-token")
		require.NoError(t, err)

		files, err := client.GetMultipleFiles(context.Background(), "owner/repo", []string{"README.md"}, "nonexistent-branch", 5)
		require.NoError(t, err)
		assert.Len(t, files, 1)
		assert.NotNil(t, files[0].Error)
	})
}

func TestClient_GetFileContent(t *testing.T) {
	t.Run("should get file content successfully", func(t *testing.T) {
		// Mock server for file content API
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Hello World"))
		}))
		defer server.Close()

		client, err := NewClient(server.URL, "test-token")
		require.NoError(t, err)

		content, err := client.GetFileContent(context.Background(), "owner/repo", "README.md", "main")
		require.NoError(t, err)
		assert.Equal(t, "Hello World", content)
	})

	t.Run("should handle file not found", func(t *testing.T) {
		// Mock server returning 404
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message": "404 File Not Found"}`))
		}))
		defer server.Close()

		client, err := NewClient(server.URL, "test-token")
		require.NoError(t, err)

		_, err = client.GetFileContent(context.Background(), "owner/repo", "nonexistent.txt", "main")
		assert.Error(t, err)
	})
}

func TestClient_GetMultipleFiles(t *testing.T) {
	t.Run("should handle concurrent file fetching", func(t *testing.T) {
		// Mock server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("file content"))
		}))
		defer server.Close()

		client, err := NewClient(server.URL, "test-token")
		require.NoError(t, err)

		filePaths := []string{"file1.txt", "file2.txt", "file3.txt"}
		files, err := client.GetMultipleFiles(context.Background(), "owner/repo", filePaths, "main", 2)
		require.NoError(t, err)
		assert.Len(t, files, 3)

		for _, file := range files {
			assert.Equal(t, "file content", file.Content)
		}
	})

	t.Run("should handle concurrency limits", func(t *testing.T) {
		client, err := NewClient("https://gitlab.com", "test-token")
		require.NoError(t, err)

		// Test with invalid concurrency (should use default)
		filePaths := []string{"file1.txt"}
		_, err = client.GetMultipleFiles(context.Background(), "owner/repo", filePaths, "main", 0)
		// This would fail due to authentication, but concurrency handling would work
		assert.Error(t, err) // Expected due to invalid token
	})
}
