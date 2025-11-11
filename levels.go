package go11y

import (
	"log/slog"
	"strings"
)

const (
	LevelDevelop = slog.Level(-8) // Custom level for development only logging, will be disabled in staging and production
	LevelDebug   = slog.Level(-4) // LevelDebug represents debug-level logging
	LevelInfo    = slog.Level(0)  // LevelInfo represents informational-level logging
	LevelNotice  = slog.Level(2)  // LevelNotice represents notice-level logging
	LevelWarning = slog.Level(4)  // LevelWarning represents warning-level logging
	LevelError   = slog.Level(8)  // LevelError represents error-level logging
	LevelFatal   = slog.Level(12) // LevelFatal represents fatal-level logging
)

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
