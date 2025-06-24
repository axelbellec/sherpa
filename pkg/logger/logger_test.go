package logger

import (
	"bytes"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestSetVerbose(t *testing.T) {
	// Reset logger state
	Logger = logrus.New()
	
	SetVerbose()
	assert.Equal(t, logrus.DebugLevel, Logger.Level)
}

func TestSetQuiet(t *testing.T) {
	// Reset logger state
	Logger = logrus.New()
	
	SetQuiet()
	assert.Equal(t, logrus.ErrorLevel, Logger.Level)
}

func TestLoggerOutput(t *testing.T) {
	// Reset logger state
	Logger = logrus.New()
	
	// Capture output
	var buf bytes.Buffer
	Logger.SetOutput(&buf)
	
	t.Run("should log info messages by default", func(t *testing.T) {
		buf.Reset()
		Logger.Info("test info message")
		assert.Contains(t, buf.String(), "test info message")
	})
	
	t.Run("should log debug messages in verbose mode", func(t *testing.T) {
		SetVerbose()
		buf.Reset()
		Logger.Debug("test debug message")
		assert.Contains(t, buf.String(), "test debug message")
	})
	
	t.Run("should not log info messages in quiet mode", func(t *testing.T) {
		SetQuiet()
		buf.Reset()
		Logger.Info("test info message")
		assert.Empty(t, buf.String())
	})
	
	t.Run("should log error messages in quiet mode", func(t *testing.T) {
		SetQuiet()
		buf.Reset()
		Logger.Error("test error message")
		assert.Contains(t, buf.String(), "test error message")
	})
}

func TestLoggerFields(t *testing.T) {
	// Reset logger state
	Logger = logrus.New()
	
	// Capture output
	var buf bytes.Buffer
	Logger.SetOutput(&buf)
	Logger.SetFormatter(&logrus.JSONFormatter{})
	
	t.Run("should include structured fields", func(t *testing.T) {
		buf.Reset()
		Logger.WithFields(logrus.Fields{
			"key1": "value1",
			"key2": "value2",
		}).Info("test message with fields")
		
		output := buf.String()
		assert.Contains(t, output, "key1")
		assert.Contains(t, output, "value1")
		assert.Contains(t, output, "key2")
		assert.Contains(t, output, "value2")
		assert.Contains(t, output, "test message with fields")
	})
}

func TestLoggerError(t *testing.T) {
	// Reset logger state
	Logger = logrus.New()
	
	// Capture output
	var buf bytes.Buffer
	Logger.SetOutput(&buf)
	
	t.Run("should log errors with error field", func(t *testing.T) {
		buf.Reset()
		testErr := assert.AnError
		Logger.WithError(testErr).Error("test error logging")
		
		output := buf.String()
		assert.Contains(t, output, "test error logging")
		assert.Contains(t, output, testErr.Error())
	})
}