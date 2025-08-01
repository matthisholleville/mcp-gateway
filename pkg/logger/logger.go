// Package logger provides a logger for the MCP Gateway.
//
//nolint:revive // need to match the interface
package logger

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger is an interface that provides logging methods.
type Logger interface {
	// These are ops that call directly to the actual zap implementation
	Debug(string, ...zap.Field)
	Info(string, ...zap.Field)
	Warn(string, ...zap.Field)
	Error(string, ...zap.Field)
	Panic(string, ...zap.Field)
	Fatal(string, ...zap.Field)
	With(...zap.Field) Logger
	Printf(string, ...interface{})
	Verbose() bool

	// These are the equivalent logger function but with context provided
	DebugWithContext(context.Context, string, ...zap.Field)
	InfoWithContext(context.Context, string, ...zap.Field)
	WarnWithContext(context.Context, string, ...zap.Field)
	ErrorWithContext(context.Context, string, ...zap.Field)
	PanicWithContext(context.Context, string, ...zap.Field)
	FatalWithContext(context.Context, string, ...zap.Field)
}

// NewNoopLogger provides a noop logger.
func NewNoopLogger() *ZapLogger {
	return &ZapLogger{
		zap.NewNop(),
	}
}

// ZapLogger is an implementation of Logger that uses the uber/zap logger underneath.
// It provides additional methods such as ones that logs based on context.
type ZapLogger struct {
	*zap.Logger
}

var _ Logger = (*ZapLogger)(nil)

// With creates a child logger and adds structured context to it. Fields added
// to the child don't affect the parent, and vice versa. Any fields that
// require evaluation (such as Objects) are evaluated upon invocation of With.
func (l *ZapLogger) With(fields ...zap.Field) Logger {
	return &ZapLogger{l.Logger.With(fields...)}
}

//nolint:revive // need to match the interface
func (l *ZapLogger) Debug(msg string, fields ...zap.Field) {
	l.Logger.Debug(msg, fields...)
}

//nolint:revive // need to match the interface
func (l *ZapLogger) Info(msg string, fields ...zap.Field) {
	l.Logger.Info(msg, fields...)
}

//nolint:revive //need to match the interface
func (l *ZapLogger) Warn(msg string, fields ...zap.Field) {
	l.Logger.Warn(msg, fields...)
}

func (l *ZapLogger) Error(msg string, fields ...zap.Field) { //nolint:revive // need to match the interface
	l.Logger.Error(msg, fields...)
}

//nolint:revive // need to match the interface
func (l *ZapLogger) Panic(msg string, fields ...zap.Field) {
	l.Logger.Panic(msg, fields...)
}

//nolint:revive // need to match the interface
func (l *ZapLogger) Fatal(msg string, fields ...zap.Field) {
	l.Logger.Fatal(msg, fields...)
}

//nolint:revive // need to match the interface
func (l *ZapLogger) Printf(format string, v ...interface{}) {
	l.Logger.Info(fmt.Sprintf(format, v...))
}

//nolint:revive // need to match the interface
func (l *ZapLogger) Verbose() bool {
	return true
}

//nolint:revive // need to match the interface
func (l *ZapLogger) DebugWithContext(ctx context.Context, msg string, fields ...zap.Field) {
	l.Logger.Debug(msg, fields...)
}

//nolint:revive // need to match the interface
func (l *ZapLogger) InfoWithContext(ctx context.Context, msg string, fields ...zap.Field) {
	l.Logger.Info(msg, fields...)
}

//nolint:revive // need to match the interface
func (l *ZapLogger) WarnWithContext(ctx context.Context, msg string, fields ...zap.Field) {
	l.Logger.Warn(msg, fields...)
}

//nolint:revive // need to match the interface
func (l *ZapLogger) ErrorWithContext(ctx context.Context, msg string, fields ...zap.Field) {
	l.Logger.Error(msg, fields...)
}

//nolint:revive // need to match the interface
func (l *ZapLogger) PanicWithContext(ctx context.Context, msg string, fields ...zap.Field) {
	l.Logger.Panic(msg, fields...)
}

//nolint:revive // need to match the interface
func (l *ZapLogger) FatalWithContext(ctx context.Context, msg string, fields ...zap.Field) {
	l.Logger.Fatal(msg, fields...)
}

// OptionsLogger implements options for logger.
type OptionsLogger struct {
	format          string
	level           string
	timestampFormat string
	outputPaths     []string
}

// OptionLogger is a function that sets an option for the logger.
type OptionLogger func(ol *OptionsLogger)

// WithFormat sets the log format for the logger.
func WithFormat(format string) OptionLogger {
	return func(ol *OptionsLogger) {
		ol.format = format
	}
}

// WithLevel sets the log level for the logger.
func WithLevel(level string) OptionLogger {
	return func(ol *OptionsLogger) {
		ol.level = level
	}
}

// WithTimestampFormat sets the timestamp format for the logger.
func WithTimestampFormat(timestampFormat string) OptionLogger {
	return func(ol *OptionsLogger) {
		ol.timestampFormat = timestampFormat
	}
}

// WithOutputPaths sets a list of URLs or file paths to write logging output to.
//
// URLs with the "file" scheme must use absolute paths on the local filesystem.
// No user, password, port, fragments, or query parameters are allowed, and the
// hostname must be empty or "localhost".
//
// Since it's common to write logs to the local filesystem, URLs without a scheme
// (e.g., "/var/log/foo.log") are treated as local file paths. Without a scheme,
// the special paths "stdout" and "stderr" are interpreted as os.Stdout and os.Stderr.
// When specified without a scheme, relative file paths also work.
//
// Defaults to "stdout".
func WithOutputPaths(paths ...string) OptionLogger {
	return func(ol *OptionsLogger) {
		ol.outputPaths = paths
	}
}

// NewLogger creates a new logger with the given options.
func NewLogger(options ...OptionLogger) (*ZapLogger, error) {
	logOptions := &OptionsLogger{
		level:           "info",
		format:          "text",
		timestampFormat: "ISO8601",
		outputPaths:     []string{"stdout"},
	}

	for _, opt := range options {
		opt(logOptions)
	}

	if logOptions.level == "none" {
		return NewNoopLogger(), nil
	}

	level, err := zap.ParseAtomicLevel(logOptions.level)
	if err != nil {
		return nil, fmt.Errorf("unknown log level: %s, error: %w", logOptions.level, err)
	}

	cfg := zap.NewProductionConfig()
	cfg.Level = level
	cfg.OutputPaths = logOptions.outputPaths
	cfg.EncoderConfig.TimeKey = "timestamp"
	cfg.EncoderConfig.CallerKey = "" // remove the "caller" field
	cfg.DisableStacktrace = true

	if logOptions.format == "text" {
		cfg.Encoding = "console"
		cfg.DisableCaller = true
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else { // Json
		cfg.EncoderConfig.EncodeTime = zapcore.EpochTimeEncoder // default in json for backward compatibility
		if logOptions.timestampFormat == "ISO8601" {
			cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		}
	}

	log, err := cfg.Build()
	if err != nil {
		return nil, err
	}

	return &ZapLogger{log}, nil
}

// MustNewLogger creates a new logger with the given format, level, and timestamp format.
// It panics if the logger creation fails.
func MustNewLogger(logFormat, logLevel, logTimestampFormat string) *ZapLogger {
	logger, err := NewLogger(
		WithFormat(logFormat),
		WithLevel(logLevel),
		WithTimestampFormat(logTimestampFormat))
	if err != nil {
		panic(err)
	}

	return logger
}
