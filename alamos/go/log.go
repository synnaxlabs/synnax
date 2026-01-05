// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alamos

import (
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/exp/slices"
)

// LoggerConfig is the config for a Logger.
type LoggerConfig struct {
	// ZapConfig sets the underlying zap.Logger. If nil, a no-op logger is used.
	ZapConfig zap.Config
	// ZapLogger provides a custom zap logger to override the default logger defined
	// in ZapConfig.
	ZapLogger *zap.Logger
}

var (
	_ config.Config[LoggerConfig] = LoggerConfig{}
	// DefaultLoggerConfig is the default config for a Logger.
	DefaultLoggerConfig = LoggerConfig{}
)

// Validate implements config.Config.
func (c LoggerConfig) Validate() error { return nil }

// Override implements config.Config.
func (c LoggerConfig) Override(other LoggerConfig) LoggerConfig {
	c.ZapConfig = other.ZapConfig
	c.ZapLogger = override.Nil(c.ZapLogger, other.ZapLogger)
	return c
}

// Logger provides logging functionality. It's an enhanced wrapper around a zap.Logger
// that provides no-lop logging when nil.
type Logger struct {
	config LoggerConfig
	zap    *zap.Logger
}

// NewLogger creates a new Logger with the given configuration.
func NewLogger(configs ...LoggerConfig) (*Logger, error) {
	cfg, err := config.New(DefaultLoggerConfig, configs...)
	if err != nil {
		return nil, err
	}
	l := &Logger{config: cfg}
	if cfg.ZapLogger != nil {
		l.zap = cfg.ZapLogger
	} else {
		z, err := cfg.ZapConfig.Build()
		if err != nil {
			return nil, err
		}
		l.zap = z
	}
	l.zap = l.zap.WithOptions(zap.AddCallerSkip(1))
	return l, nil
}

// Zap returns the underlying zap Logger
func (l *Logger) Zap() *zap.Logger {
	return l.zap
}

func (l *Logger) child(meta InstrumentationMeta) (nl *Logger) {
	if l != nil {
		nl = &Logger{zap: l.zap.Named(meta.Key), config: l.config}
	}
	return
}

// Debug logs a message at the Debug level with the given fields.
func (l *Logger) Debug(msg string, fields ...zap.Field) {
	if l != nil {
		l.zap.Debug(msg, fields...)
	}
}

func (l *Logger) Named(name string) *Logger {
	if l != nil {
		return &Logger{zap: l.zap.Named(name), config: l.config}
	}
	return nil
}

// Debugf logs a message at the Debug level using the given format. This is a slower
// method that should not be used in hot paths.
func (l *Logger) Debugf(format string, args ...any) {
	if l != nil {
		l.zap.Sugar().Debugf(format, args...)
	}
}

// Info logs a message at the Info level with the given fields.
func (l *Logger) Info(msg string, fields ...zap.Field) {
	if l != nil {
		l.zap.Info(msg, fields...)
	}
}

// Infof logs a message at the Info level using the given format. This is a slower
// method that should not be used in hot paths.
func (l *Logger) Infof(format string, args ...any) {
	if l != nil {
		l.zap.Sugar().Infof(format, args...)
	}
}

// Warn logs a message at the Warn level with the given fields.
func (l *Logger) Warn(msg string, fields ...zap.Field) {
	if l != nil {
		l.zap.Warn(msg, fields...)
	}
}

// Error logs a message at the Error level with the given fields.
func (l *Logger) Error(msg string, fields ...zap.Field) {
	if l != nil {
		l.zap.Error(msg, fields...)
	}
}

// Fatal logs a message at the Fatal level with the given fields and then exits the
// process with status code 1.
func (l *Logger) Fatal(msg string, fields ...zap.Field) {
	if l != nil {
		l.zap.Fatal(msg, fields...)
	}
}

// DPanic logs a message with the given fields that  panics in development mode and logs
// to the Error level in production mode.
func (l *Logger) DPanic(msg string, fields ...zap.Field) {
	if l != nil {
		l.zap.DPanic(msg, fields...)
	}
}

func (l *Logger) WithOptions(opts ...zap.Option) *Logger {
	if l != nil {
		return &Logger{zap: l.zap.WithOptions(opts...), config: l.config}
	}
	return nil
}

func (l *Logger) WithConfig(configs ...LoggerConfig) (*Logger, error) {
	if l == nil {
		return nil, nil
	}
	l2, err := NewLogger(configs...)
	if err != nil {
		return nil, err
	}
	return &Logger{zap: l2.zap.Named(l.zap.Name()), config: l2.config}, nil
}

func CustomZapCore(core zapcore.Core) zapcore.Core {
	return customCore{c: core}
}

type customCore struct{ c zapcore.Core }

var _ zapcore.Core = (*customCore)(nil)

// Enabled implements zapcore.Core.
func (c customCore) Enabled(level zapcore.Level) bool { return c.c.Enabled(level) }

// With implements zapcore.Core.
func (c customCore) With(fields []zapcore.Field) zapcore.Core { return &customCore{c.c.With(fields)} }

// Check implements zapcore.Core.
func (c customCore) Check(entry zapcore.Entry, entry2 *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	// Make sure that zap uses our custom core.
	if c.Enabled(entry.Level) {
		return entry2.AddCore(entry, c)
	}
	return c.c.Check(entry, entry2)
}

// Sync implements zapcore.Core.
func (c customCore) Sync() error { return c.c.Sync() }

// Write implements zapcore.Core.
func (c customCore) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	toRemove := -1
	for i, field := range fields {
		// If there is an error in the log, and we can get its stack trace, use that
		// instead of the built-in stack trace.
		if field.Type == zapcore.ErrorType {
			if err, ok := field.Interface.(error); ok {
				entry.Stack = errors.GetStackTrace(err).String()
			}
		} else if field.Key == "caller" && field.Type == zapcore.StringType && len(field.String) > 0 {
			// This means that we should specify a custom caller.
			entry.Caller = zapcore.EntryCaller{Defined: true, File: field.String}
			toRemove = i
		}
	}
	if toRemove >= 0 {
		// Clone the slice first to avoid accidentally modifying it if/when zap
		// uses it to write to an alternate core.
		fields = slices.Delete(slices.Clone(fields), toRemove, toRemove+1)
	}
	return c.c.Write(entry, fields)
}
