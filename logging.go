package go11y

import (
	"context"
	"os"
)

// Develop logs a development-only message and adds an event to the span if available.
func (o *Observer) Develop(msg string, ephemeralArgs ...any) {
	if o.span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		o.span.SetAttributes(attrs...)
		o.span.AddEvent(msg)
	}
	o.log(context.Background(), 3, LevelDevelop, msg, ephemeralArgs...)
}

// Debug logs a debug message and adds an event to the span if available.
func (o *Observer) Debug(msg string, ephemeralArgs ...any) {
	if o.span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		o.span.SetAttributes(attrs...)
		o.span.AddEvent(msg)
	}
	o.log(context.Background(), 3, LevelDebug, msg, ephemeralArgs...)
}

// Info logs an informational message and adds an event to the span if available.
func (o *Observer) Info(msg string, ephemeralArgs ...any) {
	if o.span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		o.span.SetAttributes(attrs...)
		o.span.AddEvent(msg)
	}
	o.log(context.Background(), 3, LevelInfo, msg, ephemeralArgs...)
}

// Notice logs a notice message and adds an event to the span if available.
func (o *Observer) Notice(msg string, ephemeralArgs ...any) {
	if o.span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		o.span.SetAttributes(attrs...)
		o.span.AddEvent(msg)
	}
	o.log(context.Background(), 3, LevelNotice, msg, ephemeralArgs...)
}

// Warn logs a warning message and adds an event to the span if available.
func (o *Observer) Warning(msg string, ephemeralArgs ...any) {
	if o.span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		o.span.SetAttributes(attrs...)
		o.span.AddEvent(msg)
	}
	o.log(context.Background(), 3, LevelWarning, msg, ephemeralArgs...)
}

// Warn is an alias for Warning to maintain backward compatibility.
func (o *Observer) Warn(msg string, ephemeralArgs ...any) {
	o.Warning(msg, ephemeralArgs...)
}

// Error logs an error message, records the error in the span if available, and sets the severity.
func (o *Observer) Error(msg string, err error, severity string, ephemeralArgs ...any) {
	if o.span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		o.span.SetAttributes(attrs...)
		o.span.RecordError(err)
	}
	ephemeralArgs = append(ephemeralArgs, "error", err.Error(), "severity", severity)
	o.log(context.Background(), 3, LevelError, msg, ephemeralArgs...)
}

// Fatal logs a fatal error message, records the error in the span if available, and sets the severity to highest.
func (o *Observer) Fatal(msg string, err error, ephemeralArgs ...any) {
	if o.span != nil {
		ephemeralArgs = append(o.stableArgs, ephemeralArgs...)
		attrs := argsToAttributes(ephemeralArgs...)
		o.span.SetAttributes(attrs...)
		o.span.RecordError(err)
	}
	ephemeralArgs = append(ephemeralArgs, "error", err.Error(), "severity", SeverityHighest)
	o.log(context.Background(), 3, LevelFatal, msg, ephemeralArgs...)
	os.Exit(1)
}

// Fatal is intended to be called before the observer has been configured.
// It will log the fatal error to stderr in the JSON format used by go11y and exit the application.
func Fatal(msg string, err error, ephemeralArgs ...any) {
	cfg := &Configuration{
		logLevel:    LevelInfo,
		otelURL:     "",
		strLevel:    "info",
		dbConStr:    "",
		serviceName: "",
		trimModules: []string{},
		trimPaths:   []string{},
	}
	ctx := context.Background()
	_, o, _ := Initialise(ctx, cfg, os.Stderr)
	o.log(ctx, 3, LevelFatal, msg, ephemeralArgs...)
	os.Exit(1)
}
