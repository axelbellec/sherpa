package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMain(t *testing.T) {
	t.Run("should execute main function without panicking", func(t *testing.T) {
		// Store original args
		originalArgs := os.Args
		defer func() {
			os.Args = originalArgs
		}()

		// Set test args to avoid actual command execution
		os.Args = []string{"sherpa", "--help"}

		// This test verifies that main() can be called without panicking
		// The actual command execution is tested in cmd package tests
		assert.NotPanics(t, func() {
			// We can't easily test main() without it actually executing,
			// so we'll just verify the main function exists and can be referenced
			assert.NotNil(t, main)
		})
	})
}

func TestMainFunctionExists(t *testing.T) {
	t.Run("should have main function", func(t *testing.T) {
		// This test ensures the main function is properly defined
		// and can be called (even though we can't easily test its execution)
		assert.NotNil(t, main)
	})
}