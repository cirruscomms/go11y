package go11y

import (
	"context"
	"fmt"
	"os"
	"slices"
)

// Develop logs a development-only message and adds an event to the span if available.
func (o *Observer) Develop(msg string, ephemeralArgs ...any) {
	logged := o.log(context.Background(), 3, LevelDevelop, msg, ephemeralArgs...)
	if logged && o.span != nil {
		attrs := argsToAttributes(append(o.stableArgs, ephemeralArgs)...)
		o.span.SetAttributes(attrs...)
		o.span.AddEvent(msg)
	}
}

// Debug logs a debug message and adds an event to the span if available.
func (o *Observer) Debug(msg string, ephemeralArgs ...any) {
	logged := o.log(context.Background(), 3, LevelDebug, msg, ephemeralArgs...)
	if logged && o.span != nil {
		attrs := argsToAttributes(append(o.stableArgs, ephemeralArgs)...)
		o.span.SetAttributes(attrs...)
		o.span.AddEvent(msg)
	}
}

// Info logs an informational message and adds an event to the span if available.
func (o *Observer) Info(msg string, ephemeralArgs ...any) {
	logged := o.log(context.Background(), 3, LevelInfo, msg, ephemeralArgs...)
	if logged && o.span != nil {
		attrs := argsToAttributes(append(o.stableArgs, ephemeralArgs)...)
		o.span.SetAttributes(attrs...)
		o.span.AddEvent(msg)
	}
}

// Notice logs a notice message and adds an event to the span if available.
func (o *Observer) Notice(msg string, ephemeralArgs ...any) {
	logged := o.log(context.Background(), 3, LevelNotice, msg, ephemeralArgs...)
	if logged && o.span != nil {
		attrs := argsToAttributes(append(o.stableArgs, ephemeralArgs)...)
		o.span.SetAttributes(attrs...)
		o.span.AddEvent(msg)
	}
}

// Warning logs a warning message and adds an event to the span if available.
func (o *Observer) Warning(msg string, ephemeralArgs ...any) {
	logged := o.log(context.Background(), 3, LevelWarning, msg, ephemeralArgs...)
	if logged && o.span != nil {
		attrs := argsToAttributes(append(o.stableArgs, ephemeralArgs)...)
		o.span.SetAttributes(attrs...)
		o.span.AddEvent(msg)
	}
}

// Warn a backward compatibility alias for Warning.
func (o *Observer) Warn(msg string, ephemeralArgs ...any) {
	logged := o.log(context.Background(), 3, LevelWarning, msg, ephemeralArgs...)
	if logged && o.span != nil {
		attrs := argsToAttributes(append(o.stableArgs, ephemeralArgs)...)
		o.span.SetAttributes(attrs...)
		o.span.AddEvent(msg)
	}
}

// Error logs an error message, records the error in the span if available, and sets the severity.
func (o *Observer) Error(msg string, err error, severity string, ephemeralArgs ...any) {
	logged := o.error(context.Background(), 3, LevelFatal, msg, append(ephemeralArgs, "error", err.Error(), "severity", severity)...)
	if logged && o.span != nil {
		attrs := argsToAttributes(append(o.stableArgs, ephemeralArgs)...)
		o.span.SetAttributes(attrs...)
		o.span.RecordError(err)
	}
}

// Fatal logs a fatal error message, records the error in the span if available, and sets the severity to highest.
func (o *Observer) Fatal(msg string, err error, ephemeralArgs ...any) {
	logged := o.error(context.Background(), 3, LevelFatal, msg, append(ephemeralArgs, "error", err.Error(), "severity", SeverityHighest)...)
	if logged && o.span != nil {
		attrs := argsToAttributes(append(o.stableArgs, ephemeralArgs)...)
		o.span.SetAttributes(attrs...)
		o.span.RecordError(err)
	}

	os.Exit(1)
}

// Fatal is intended to be called before the observer has been configured.
// It will log the fatal error to stderr in the JSON format used by go11y and exit the application.
func Fatal(msg string, err error, exitCode int, ephemeralArgs ...any) {
	cfg := &Configuration{
		logLevel:    LevelFatal,
		otelURL:     "",
		strLevel:    "fatal",
		databaseURL: "",
		serviceName: "",
		trimModules: []string{},
		trimPaths:   []string{},
	}
	ctx := context.Background()
	_, o, _ := Initialise(ctx, cfg, nil, os.Stderr)
	ephemeralArgs = append(ephemeralArgs, "error", err.Error(), "severity", SeverityHighest)
	o.error(ctx, o.skipCallers, LevelFatal, msg, ephemeralArgs...)

	if exitCode < 1 {
		exitCode = 1
	}

	os.Exit(exitCode)
}

// Error is intended to be called before the observer has been configured.
// It will log the error to stderr in the JSON format used by go11y.
func Error(msg string, err error, severity string, ephemeralArgs ...any) {
	cfg := &Configuration{
		logLevel:    LevelError,
		otelURL:     "",
		strLevel:    "error",
		databaseURL: "",
		serviceName: "",
		trimModules: []string{},
		trimPaths:   []string{},
	}

	ctx := context.Background()
	_, o, _ := Initialise(ctx, cfg, nil, os.Stderr)
	ephemeralArgs = append(ephemeralArgs, "error", err.Error(), "severity", severity)
	o.error(ctx, o.skipCallers, LevelFatal, msg, ephemeralArgs...)
}

// DeduplicateArgs removes duplicate keys from a list of key-value pairs.
func DeduplicateArgs(args []any) (deduped []any) {
	keys := []string{}
	uniq := []any{}

	for i := 0; i < len(args); i += 2 {
		if len(args) >= i+2 {
			key := fmt.Sprintf("%v", args[i])
			if slices.Contains(keys, key) {
				continue
			}

			keys = append(keys, key)
			uniq = append(uniq, args[i], args[i+1])
		}
	}

	return uniq
}
