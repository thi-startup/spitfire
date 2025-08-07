package log

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

// Logger wraps logrus with spitfire-specific configuration
type Logger struct {
	*logrus.Logger
	userOutput io.Writer // For clean user-facing messages
}

// LogLevel represents the logging level
type LogLevel string

const (
	TraceLevel LogLevel = "trace"
	DebugLevel LogLevel = "debug"
	InfoLevel  LogLevel = "info"
	WarnLevel  LogLevel = "warn"
	ErrorLevel LogLevel = "error"
)

// Config holds logging configuration
type Config struct {
	Level   LogLevel
	Verbose bool
	Debug   bool
	Quiet   bool
	LogFile string
	Format  string // "text" or "json"
}

var (
	// DefaultLogger is the global logger instance
	DefaultLogger *Logger
)

// NewLogger creates a new logger with the given configuration
func NewLogger(config Config) *Logger {
	logger := logrus.New()

	// Set output (logs go to stderr by default, user output to stdout)
	userOutput := os.Stdout
	logOutput := os.Stderr

	// Handle log file output
	if config.LogFile != "" {
		file, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err == nil {
			logOutput = file
		}
		// If file creation fails, fall back to stderr
	}

	logger.SetOutput(logOutput)

	// Set log level
	level := parseLogLevel(config)
	logger.SetLevel(level)

	// Set formatter
	if config.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		// Custom text formatter for better CLI experience
		logger.SetFormatter(&logrus.TextFormatter{
			DisableTimestamp: true,
			DisableColors:    config.LogFile != "", // Disable colors when logging to file
			PadLevelText:     true,
		})
	}

	return &Logger{
		Logger:     logger,
		userOutput: userOutput,
	}
}

// parseLogLevel converts config to logrus level
func parseLogLevel(config Config) logrus.Level {
	// Handle explicit level first
	switch config.Level {
	case TraceLevel:
		return logrus.TraceLevel
	case DebugLevel:
		return logrus.DebugLevel
	case InfoLevel:
		return logrus.InfoLevel
	case WarnLevel:
		return logrus.WarnLevel
	case ErrorLevel:
		return logrus.ErrorLevel
	}

	// Handle flags
	if config.Debug {
		return logrus.DebugLevel
	}
	if config.Verbose {
		return logrus.InfoLevel
	}
	if config.Quiet {
		return logrus.ErrorLevel
	}

	// Default to warn level (show warnings and errors)
	return logrus.WarnLevel
}

// ParseLogLevel parses a string log level
func ParseLogLevel(level string) LogLevel {
	switch strings.ToLower(level) {
	case "trace":
		return TraceLevel
	case "debug":
		return DebugLevel
	case "info":
		return InfoLevel
	case "warn", "warning":
		return WarnLevel
	case "error":
		return ErrorLevel
	default:
		return InfoLevel
	}
}

// User outputs clean messages to stdout for user consumption
func (l *Logger) User(format string, args ...interface{}) {
	fmt.Fprintf(l.userOutput, format, args...)
}

// UserLn outputs clean messages to stdout with newline
func (l *Logger) UserLn(format string, args ...interface{}) {
	fmt.Fprintf(l.userOutput, format+"\n", args...)
}

// Success outputs success messages in green (if terminal supports it)
func (l *Logger) Success(format string, args ...interface{}) {
	// Use green color for success messages
	fmt.Fprintf(l.userOutput, "✓ "+format+"\n", args...)
}

// Progress outputs progress messages for user operations
func (l *Logger) Progress(format string, args ...interface{}) {
	if !l.IsLevelEnabled(logrus.WarnLevel) { // If not quiet
		fmt.Fprintf(l.userOutput, format+"...\n", args...)
	}
}

// WithField creates a new logger with a field
func (l *Logger) WithField(key string, value interface{}) *logrus.Entry {
	return l.Logger.WithField(key, value)
}

// WithFields creates a new logger with multiple fields
func (l *Logger) WithFields(fields logrus.Fields) *logrus.Entry {
	return l.Logger.WithFields(fields)
}

// InitializeGlobalLogger sets up the global logger
func InitializeGlobalLogger(config Config) {
	DefaultLogger = NewLogger(config)
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() *Logger {
	if DefaultLogger == nil {
		// Initialize with default config if not set
		InitializeGlobalLogger(Config{
			Level: InfoLevel,
		})
	}
	return DefaultLogger
}

// Convenience functions for global logger
func Debug(args ...interface{}) {
	GetGlobalLogger().Debug(args...)
}

func Debugf(format string, args ...interface{}) {
	GetGlobalLogger().Debugf(format, args...)
}

func Info(args ...interface{}) {
	GetGlobalLogger().Info(args...)
}

func Infof(format string, args ...interface{}) {
	GetGlobalLogger().Infof(format, args...)
}

func Warn(args ...interface{}) {
	GetGlobalLogger().Warn(args...)
}

func Warnf(format string, args ...interface{}) {
	GetGlobalLogger().Warnf(format, args...)
}

func Error(args ...interface{}) {
	GetGlobalLogger().Error(args...)
}

func Errorf(format string, args ...interface{}) {
	GetGlobalLogger().Errorf(format, args...)
}

func WithField(key string, value interface{}) *logrus.Entry {
	return GetGlobalLogger().WithField(key, value)
}

func WithFields(fields logrus.Fields) *logrus.Entry {
	return GetGlobalLogger().WithFields(fields)
}

// User convenience functions
func User(format string, args ...interface{}) {
	GetGlobalLogger().User(format, args...)
}

func UserLn(format string, args ...interface{}) {
	GetGlobalLogger().UserLn(format, args...)
}

func Success(format string, args ...interface{}) {
	GetGlobalLogger().Success(format, args...)
}

func Progress(format string, args ...interface{}) {
	GetGlobalLogger().Progress(format, args...)
}
