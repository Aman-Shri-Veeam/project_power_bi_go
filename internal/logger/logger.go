package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

var Log *logrus.Logger

// InitLogger initializes the logger
func InitLogger(debug bool) {
	Log = logrus.New()
	Log.SetOutput(os.Stdout)
	Log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		ForceColors:   true,
	})

	if debug {
		Log.SetLevel(logrus.DebugLevel)
	} else {
		Log.SetLevel(logrus.InfoLevel)
	}
}

// LogInfo logs an info message
func LogInfo(message string) {
	Log.Info(message)
}

// LogError logs an error message
func LogError(message string, err error) {
	if err != nil {
		Log.WithError(err).Error(message)
	} else {
		Log.Error(message)
	}
}

// LogDebug logs a debug message
func LogDebug(message string) {
	Log.Debug(message)
}

// LogWarn logs a warning message
func LogWarn(message string) {
	Log.Warn(message)
}
