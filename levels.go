package go11y

import (
	"log/slog"
	"strings"
)

// Custom level for development only logging, will be disabled in staging and production
const LevelDevelop = slog.Level(-8)

// LevelDebug represents debug-level logging
const LevelDebug = slog.Level(-4)

// LevelInfo represents informational-level logging
const LevelInfo = slog.Level(0)

// LevelNotice represents notice-level logging
const LevelNotice = slog.Level(2)

// LevelWarning represents warning-level logging
const LevelWarning = slog.Level(4)

// LevelError represents error-level logging
const LevelError = slog.Level(8)

// LevelFatal represents fatal-level logging
const LevelFatal = slog.Level(12)

// StringToLevel maps a string representation of a log level to its corresponding slog.Level.
func StringToLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "develop":
		return LevelDevelop // Custom level for development, not used in production
	case "debug":
		return LevelDebug
	case "info":
		return LevelInfo
	case "notice":
		return LevelNotice
	case "warning", "warn":
		return LevelWarning
	case "error":
		return LevelError
	case "fatal":
		return LevelFatal
	default:
		return LevelDebug // default to debug if unknown level
	}
}
