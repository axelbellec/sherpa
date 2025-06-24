package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	baseURL := "https://api.github.com"
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
			w.Write([]byte(`{"login": "testuser"}`))
		}))
		defer server.Close()

		client, err := NewClient(server.URL+"/", "test-token")
		require.NoError(t, err)

		err = client.TestConnection(context.Background())
		assert.NoError(t, err)
	})

	t.Run("should fail on invalid token", func(t *testing.T) {
		// Mock server returning 401
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"message": "Bad credentials"}`))
		}))
		defer server.Close()

		client, err := NewClient(server.URL, "invalid-token")
		require.NoError(t, err)

		err = client.TestConnection(context.Background())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "authenticate")
	})

	t.Run("should handle network errors", func(t *testing.T) {
		client, err := NewClient("http://localhost:99999/", "test-token") // Invalid port
		require.NoError(t, err)

		err = client.TestConnection(context.Background())
		assert.Error(t, err)
	})
}

func TestClient_GetRepository(t *testing.T) {
	t.Run("should get repository successfully", func(t *testing.T) {
		// Mock server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/repos/owner/repo", r.URL.Path)
			assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
			
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"name": "repo",
				"full_name": "owner/repo",
				"default_branch": "main",
				"private": false
			}`))
		}))
		defer server.Close()

		client, err := NewClient(server.URL+"/", "test-token")
		require.NoError(t, err)

		repo, err := client.GetRepository(context.Background(), "owner", "repo")
		require.NoError(t, err)
		assert.Equal(t, "repo", repo.Name)
		assert.Equal(t, "owner/repo", repo.PathWithNamespace)
	})

	t.Run("should handle repository not found", func(t *testing.T) {
		// Mock server returning 404
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message": "Not Found"}`))
		}))
		defer server.Close()

		client, err := NewClient(server.URL+"/", "test-token")
		require.NoError(t, err)

		_, err = client.GetRepository(context.Background(), "owner", "nonexistent")
		assert.Error(t, err)
	})
}

func TestClient_GetRepositoryFiles(t *testing.T) {
	t.Run("should get repository files successfully", func(t *testing.T) {
		// Mock server for tree API
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/repos/owner/repo/git/trees/main" {
				assert.Equal(t, "true", r.URL.Query().Get("recursive"))
				assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
				
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{
					"tree": [
						{
							"path": "README.md",
							"type": "blob",
							"sha": "abc123",
							"size": 100
						},
						{
							"path": "src/main.go",
							"type": "blob",
							"sha": "def456",
							"size": 200
						}
					]
				}`))
			} else if r.URL.Path == "/repos/owner/repo/git/blobs/abc123" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{
					"content": "IyBSZWFkbWU=",
					"encoding": "base64"
				}`))
			} else if r.URL.Path == "/repos/owner/repo/git/blobs/def456" {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{
					"content": "cGFja2FnZSBtYWlu",
					"encoding": "base64"
				}`))
			}
		}))
		defer server.Close()

		client, err := NewClient(server.URL+"/", "test-token")
		require.NoError(t, err)

		files, err := client.GetMultipleFiles(context.Background(), "owner", "repo", []string{"README.md", "src/main.go"}, "main", 5)
		require.NoError(t, err)
		assert.Len(t, files, 2)
		
		// Check first file
		assert.Equal(t, "README.md", files[0].Path)
		assert.Equal(t, "# Readme", files[0].Content)
		assert.Equal(t, int64(100), files[0].Size)
		
		// Check second file
		assert.Equal(t, "src/main.go", files[1].Path)
		assert.Equal(t, "package main", files[1].Content)
		assert.Equal(t, int64(200), files[1].Size)
	})

	t.Run("should handle tree not found", func(t *testing.T) {
		// Mock server returning 404 for tree
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message": "Not Found"}`))
		}))
		defer server.Close()

		client, err := NewClient(server.URL+"/", "test-token")
		require.NoError(t, err)

		files, err := client.GetMultipleFiles(context.Background(), "owner", "repo", []string{"README.md"}, "nonexistent-branch", 5)
		require.NoError(t, err)
		assert.Len(t, files, 1)
		assert.NotNil(t, files[0].Error)
	})
}

func TestClient_GetFileContent(t *testing.T) {
	t.Run("should get file content successfully", func(t *testing.T) {
		// Mock server for contents API
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/repos/owner/repo/contents/test-file.txt", r.URL.Path)
			assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
			
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"content": "SGVsbG8gV29ybGQ=",
				"encoding": "base64"
			}`))
		}))
		defer server.Close()

		client, err := NewClient(server.URL+"/", "test-token")
		require.NoError(t, err)

		content, err := client.GetFileContent(context.Background(), "owner", "repo", "test-file.txt", "main")
		require.NoError(t, err)
		assert.Equal(t, "Hello World", content)
	})

	t.Run("should handle file not found", func(t *testing.T) {
		// Mock server returning 404
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"message": "Not Found"}`))
		}))
		defer server.Close()

		client, err := NewClient(server.URL+"/", "test-token")
		require.NoError(t, err)

		_, err = client.GetFileContent(context.Background(), "owner", "repo", "nonexistent-file.txt", "main")
		assert.Error(t, err)
	})
}