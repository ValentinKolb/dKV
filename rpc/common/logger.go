// Package util provides logging utilities for the application
package common

import (
	"fmt"
	"github.com/lni/dragonboat/v4/logger"
	"log"
	"os"
	"strings"
)

// --------------------------------------------------------------------------
// Custom Logger (implements dragenboats logger.ILogger)
// --------------------------------------------------------------------------

// dKVLogger implements the ILogger interface with custom formatting
type dKVLogger struct {
	name   string
	level  logger.LogLevel
	logger *log.Logger
}

func (l *dKVLogger) SetLevel(level logger.LogLevel) {
	l.level = level
}

func (l *dKVLogger) Debugf(format string, args ...interface{}) {
	if l.level >= logger.DEBUG {
		l.log("DEBUG", format, args...)
	}
}

func (l *dKVLogger) Infof(format string, args ...interface{}) {
	if l.level >= logger.INFO {
		l.log("INFO", format, args...)
	}
}

func (l *dKVLogger) Warningf(format string, args ...interface{}) {
	if l.level >= logger.WARNING {
		l.log("WARN", format, args...)
	}
}

func (l *dKVLogger) Errorf(format string, args ...interface{}) {
	if l.level >= logger.ERROR {
		l.log("ERROR", format, args...)
	}
}

func (l *dKVLogger) Panicf(format string, args ...interface{}) {
	if l.level >= logger.CRITICAL {
		panic(fmt.Sprintf(format, args...))
	}
}

// log formats and writes a log message. this internal helper is used by the public methods
func (l *dKVLogger) log(levelStr string, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	l.logger.Printf("%-5s | %-15s | %s", levelStr, l.name, message)
}

// --------------------------------------------------------------------------
// Logger Factory
// --------------------------------------------------------------------------

// CreateLogger implements the Factory interface - note the error return value
func CreateLogger(pkgName string) logger.ILogger {
	// Create standard logger with custom flags
	stdLogger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	return &dKVLogger{
		name:   pkgName,
		level:  logger.INFO,
		logger: stdLogger,
	}
}

// --------------------------------------------------------------------------
// Helper
// --------------------------------------------------------------------------

// parseLogLevel converts a string level to logger.LogLevel
func parseLogLevel(level string) logger.LogLevel {
	switch strings.ToLower(level) {
	case "debug":
		return logger.DEBUG
	case "info":
		return logger.INFO
	case "warning", "warn":
		return logger.WARNING
	case "error":
		return logger.ERROR
	default:
		panic(fmt.Sprintf("invalid log level: %s. must be one of debug, info, warn, error", level))
	}
}

// --------------------------------------------------------------------------
// Logger initialization
// --------------------------------------------------------------------------

// InitLoggers initializes all loggers with the custom format
func InitLoggers(config ServerConfig) {
	// Create custom logger factory

	// Set as the global logger factory for Dragonboat
	logger.SetLoggerFactory(CreateLogger)

	// Configure Dragonboat loggers
	logger.GetLogger("raft").SetLevel(parseLogLevel(config.LogLevel))
	logger.GetLogger("raftdb").SetLevel(parseLogLevel(config.LogLevel))
	logger.GetLogger("rsm").SetLevel(parseLogLevel(config.LogLevel))
	logger.GetLogger("transport").SetLevel(parseLogLevel(config.LogLevel))
	logger.GetLogger("dragonboat").SetLevel(parseLogLevel(config.LogLevel))
	logger.GetLogger("grpc").SetLevel(parseLogLevel(config.LogLevel))
	logger.GetLogger("util").SetLevel(parseLogLevel(config.LogLevel))
	logger.GetLogger("logdb").SetLevel(parseLogLevel(config.LogLevel))

	// configure custom loggers TODO
	logger.GetLogger("store").SetLevel(parseLogLevel(config.LogLevel))
	logger.GetLogger("transport/rpc").SetLevel(parseLogLevel(config.LogLevel))
	logger.GetLogger("rpc").SetLevel(parseLogLevel(config.LogLevel))
}
