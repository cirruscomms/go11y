// Package go11y provides observability features including logging, tracing, and database logging of
// roundtrip requests to third-party APIs.
package go11y

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"time"

	otelSDKTrace "go.opentelemetry.io/otel/sdk/trace"
	otelTrace "go.opentelemetry.io/otel/trace"
)

// Fields represents a set of key-value pairs for logging.
type Fields map[string]any

// Observer is the main struct for observability, containing loggers, tracer providers, and database connections.
type Observer struct {
	cfg           Configurator
	output        io.Writer
	level         slog.Level
	outLogger     *slog.Logger
	errLogger     *slog.Logger
	traceProvider *otelSDKTrace.TracerProvider
	tracer        otelTrace.Tracer
	stableArgs    []any
	span          otelTrace.Span
	spans         []otelTrace.Span
	skipCallers   int
}

type go11yContextKey string

var obsKeyInstance go11yContextKey = "cirruscomms/go11y"

var ogx *Observer

// Initialise sets up the Observer with the provided configuration, log outputs, and initial arguments.
func Initialise(
	ctx context.Context,
	cfg Configurator,
	logOutput, errOutput io.Writer,
	initialArgs ...any,
) (
	ctxWithGo11y context.Context,
	observer *Observer,
	fault error,
) {
	if logOutput == nil {
		logOutput = os.Stdout
	}

	if errOutput == nil {
		errOutput = os.Stderr
	}

	var err error

	if cfg == nil {
		cfg, err = LoadConfig()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to load configuration: %w", err)
		}
	}

	tp, err := tracerProvider(ctx, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create tracer: %w", err)
	}

	opts := defaultOptions(cfg)

	o := &Observer{
		cfg:           cfg,
		output:        logOutput,
		outLogger:     slog.New(slog.NewJSONHandler(logOutput, opts)),
		errLogger:     slog.New(slog.NewJSONHandler(errOutput, opts)),
		traceProvider: tp,
		stableArgs:    initialArgs,
		skipCallers:   3, // default to 3 but allow it to be increased via o.IncreaseDistance()
	}

	ctx = context.WithValue(ctx, obsKeyInstance, o)
	if len(initialArgs) != 0 {
		ctx, o, _ = Extend(ctx, initialArgs...)
	}

	slog.SetDefault(o.outLogger)

	o.Debug("Initialised observer with context")

	return ctx, o, nil
}

// Reset resets the Observer in the context to its initial state.
func Reset(ctxWithGo11y context.Context) (ctxWithResetObservability context.Context) {
	ctxWithGo11y, o, err := Get(ctxWithGo11y)
	if err != nil {
		return ctxWithGo11y
	}

	o.outLogger = slog.New(slog.NewJSONHandler(o.output, defaultOptions(o.cfg)))
	o.errLogger = slog.New(slog.NewJSONHandler(o.output, defaultOptions(o.cfg)))
	o.Debug("Observer reset")
	o.stableArgs = []any{}

	return context.WithValue(ctxWithGo11y, obsKeyInstance, o)
}

// Get retrieves the Observer from the context. If none exists, it initializes a new one with default settings.
func Get(ctx context.Context) (ctxWithObserver context.Context, observer *Observer, fault error) {
	ob := ctx.Value(obsKeyInstance)
	if ob == nil {
		return ctx, nil, fmt.Errorf("go11y Observer not found in context - please initialise go11y first")
	}

	o := ob.(*Observer)

	return ctx, o, nil
}

// Extend retrieves the Observer from the context and adds new arguments to its logger.
// If no Observer exists in the context, it initializes a new one with default settings and adds the arguments.
func Extend(ctx context.Context, newArgs ...any) (ctxWithGo11y context.Context, observer *Observer, fault error) {
	ctx, o, err := Get(ctx)
	if err != nil {
		return ctx, nil, err
	}

	if len(newArgs) != 0 {
		o.outLogger = o.outLogger.With(newArgs...)
		o.errLogger = o.errLogger.With(newArgs...)
		o.stableArgs = o.AddArgs(newArgs...)
	}

	return context.WithValue(ctx, obsKeyInstance, o), o, nil
}

// Span gets the Observer from the context and starts a new tracing span with the given name.
// If no Observer exists in the context, it initializes a new one with default settings and starts the span.
// The tracing equivalent of Get()
func Span(
	ctx context.Context,
	tracer otelTrace.Tracer,
	spanName string,
	spanKind otelTrace.SpanKind,
) (
	ctxWithSpan context.Context,
	observer *Observer,
	fault error,
) {
	ctx, o, err := Get(ctx)
	if err != nil {
		return ctx, nil, err
	}

	ctx, span := tracer.Start(ctx, spanName, otelTrace.WithSpanKind(spanKind))

	o.span = span
	o.spans = append(o.spans, span)

	return context.WithValue(ctx, obsKeyInstance, o), o, nil
}

// Expand retrieves the Observer from the context, starts a new tracing span with the given name, and adds new arguments
// to its logger. If no Observer exists in the context, it initializes a new one with default settings and adds the
// arguments.
func Expand(
	ctx context.Context,
	tracer otelTrace.Tracer,
	spanName string,
	spanKind otelTrace.SpanKind,
	newArgs ...any,
) (
	ctxWithSpan context.Context,
	observer *Observer,
	fault error,
) {
	ctx, o, err := Span(ctx, tracer, spanName, spanKind)
	if err != nil {
		return ctx, nil, err
	}

	if len(newArgs) != 0 {
		o.outLogger = o.outLogger.With(newArgs...)
		o.errLogger = o.errLogger.With(newArgs...)
		o.stableArgs = o.AddArgs(newArgs...)
	}

	return context.WithValue(ctx, obsKeyInstance, o), o, nil
}

// Close ends all active spans and shuts down the trace provider to ensure all traces are flushed.
func (o *Observer) Close() {
	if o.span != nil {
		o.span.End()

		for _, s := range o.spans {
			s.End()
		}
	}

	if err := o.traceProvider.Shutdown(context.Background()); err != nil {
		o.Fatal("could not shut down tracer", err)
	}
}

// defaultReplacer creates a function to replace or modify log attributes
func defaultReplacer(trimModules, trimPaths []string) func(groups []string, a slog.Attr) slog.Attr {
	return func(groups []string, a slog.Attr) slog.Attr {
		if os.Getenv("ENV") == "test" && a.Key == slog.TimeKey {
			return slog.Attr{} // remove time key in test to make it easier to compare
		}

		switch a.Key {
		case slog.SourceKey:
			source, ok := a.Value.Any().(*slog.Source)
			if !ok {
				return a
			}

			for _, path := range trimPaths {
				if idx := strings.Index(source.File, path); idx != -1 {
					source.File = source.File[idx+len(path):]
				}
			}

			for _, module := range trimModules {
				if idx := strings.Index(source.Function, module); idx != -1 {
					source.Function = source.Function[idx+len(module):]
				}
			}

			return slog.Any(a.Key, source)
		case slog.LevelKey:
			var level slog.Level

			if lvl, ok := a.Value.Any().(slog.Level); ok {
				level = lvl
			} else {
				level = StringToLevel(fmt.Sprintf("%v", a.Value.Any()))
			}

			switch level {
			case LevelDebug:
				a.Value = slog.StringValue("DEBUG")
			case LevelInfo:
				a.Value = slog.StringValue("INFO")
			case LevelNotice:
				a.Value = slog.StringValue("NOTICE")
			case LevelWarning:
				a.Value = slog.StringValue("WARN")
			case LevelError:
				a.Value = slog.StringValue("ERR")
			case LevelFatal:
				a.Value = slog.StringValue("FATAL")
			default:
				a.Value = slog.StringValue("DEBUG")
			}
		}

		return a
	}
}

func (o *Observer) log(ctx context.Context, skipCallers int, level slog.Level, msg string, args ...any) (levelEnabled bool) {
	if o.outLogger == nil || !o.outLogger.Enabled(ctx, level) {
		return false
	}
	var pc uintptr
	var pcs [1]uintptr
	// skip [runtime.Callers, this function, this function's caller]
	runtime.Callers(skipCallers, pcs[:])
	pc = pcs[0]

	r := slog.NewRecord(time.Now(), level, msg, pc)

	if len(args) != 0 {
		r.Add(DeduplicateArgs(args)...)
	}

	if ctx == nil {
		ctx = context.Background()
	}
	_ = o.outLogger.Handler().Handle(ctx, r)

	return true
}

func (o *Observer) error(ctx context.Context, skipCallers int, level slog.Level, msg string, args ...any) (levelEnabled bool) {
	if o.errLogger == nil || !o.errLogger.Enabled(ctx, level) {
		return false
	}
	var pc uintptr
	var pcs [1]uintptr
	// skip [runtime.Callers, this function, this function's caller]
	runtime.Callers(skipCallers, pcs[:])
	pc = pcs[0]

	r := slog.NewRecord(time.Now(), level, msg, pc)

	if len(args) != 0 {
		r.Add(DeduplicateArgs(args)...)
	}

	if ctx == nil {
		ctx = context.Background()
	}
	_ = o.errLogger.Handler().Handle(ctx, r)

	return true
}

// AddArgs processes the provided arguments, ensuring that they are stable and formatted correctly.
func (o *Observer) AddArgs(args ...any) (filteredArgs []any) {
	args = append(o.stableArgs, args...)

	exArgs := map[any]any{}

	for len(args) > 0 {
		exArgs, args = processArgs(exArgs, args)
	}

	resArgs := make([]any, 0, len(exArgs)/2)
	for k, v := range exArgs {
		resArgs = append(resArgs, k, v)
	}

	return resArgs
}

func processArgs(exArgs map[any]any, args []any) (map[any]any, []any) {
	if len(args) < 2 {
		return exArgs, []any{}
	}

	exArgs[args[0]] = args[1]

	return exArgs, args[2:]
}

// End ends the current tracing span and reverts to the previous span in the stack.
func (o *Observer) End() {
	o.span.End()

	o.spans = o.spans[:len(o.spans)-1]
	if len(o.spans) > 0 {
		o.span = o.spans[len(o.spans)-1]
	} else {
		o.span = nil
	}
}

// InContext can be used to check if go11y has been added to a context before calling go11y.Get()
// This is useful for other packages imported by services that use go11y as well as other services that still use the
// go-logging package.
func InContext(ctx context.Context) (response bool) {
	return (ctx.Value(obsKeyInstance) != nil)
}

// IncreaseDistance increases the caller skip distance for logging purposes.
// This is useful when wrapping go11y (such as the go-common splitLog)
func (o *Observer) IncreaseDistance(distance int) {
	o.skipCallers += distance
}

// AddToContext adds the Observer to the provided context.
// This is useful for reducing boilerplate in handlers and middlewares.
func AddToContext(ctx context.Context, o *Observer) context.Context {
	return context.WithValue(ctx, obsKeyInstance, o)
}
