package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

var Logger *logrus.Logger

func init() {
	Logger = logrus.New()

	// Set output to stdout
	Logger.SetOutput(os.Stdout)

	// Set default log level to Info
	Logger.SetLevel(logrus.InfoLevel)

	// Use colored text formatter for better readability
	Logger.SetFormatter(&logrus.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})
}

// SetLevel sets the logging level
func SetLevel(level string) {
	switch level {
	case "debug":
		Logger.SetLevel(logrus.DebugLevel)
	case "info":
		Logger.SetLevel(logrus.InfoLevel)
	case "warn":
		Logger.SetLevel(logrus.WarnLevel)
	case "error":
		Logger.SetLevel(logrus.ErrorLevel)
	default:
		Logger.SetLevel(logrus.InfoLevel)
	}
}

// SetQuiet disables all logging except errors
func SetQuiet() {
	Logger.SetLevel(logrus.ErrorLevel)
}

// SetVerbose enables debug logging
func SetVerbose() {
	Logger.SetLevel(logrus.DebugLevel)
}
