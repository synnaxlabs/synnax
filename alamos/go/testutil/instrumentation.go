// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package testutil provides Alamos instrumentation helpers for tests.
package testutil

import (
	"context"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/git"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/testutil"
	"github.com/uptrace/uptrace-go/uptrace"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

type InstrumentationConfig struct {
	// Trace enables tracing for this instrumentation.
	Trace *bool
	// Log enables logging for this instrumentation.
	Log *bool
	// Report enables reports for this instrumentation.
	Report *bool
}

var _ config.Config[InstrumentationConfig] = InstrumentationConfig{}

func (c InstrumentationConfig) Validate() error { return nil }

func (c InstrumentationConfig) Override(
	other InstrumentationConfig,
) InstrumentationConfig {
	c.Report = override.Nil(c.Report, other.Report)
	c.Log = override.Nil(c.Log, other.Log)
	c.Trace = override.Nil(c.Trace, other.Trace)
	return c
}

var DefaultInstrumentationConfig = InstrumentationConfig{
	Trace:  new(false),
	Log:    new(false),
	Report: new(false),
}

func serviceName() string { return lo.Must(os.Hostname()) }

const devDSN = "http://synnax_dev@localhost:14317/2"

func newTracer(serviceName string) (*alamos.Tracer, error) {
	commit, err := git.CurrentCommit()
	if err != nil {
		return nil, err
	}
	uptrace.ConfigureOpentelemetry(
		uptrace.WithDSN(devDSN),
		uptrace.WithServiceName(serviceName),
		uptrace.WithServiceVersion(commit),
	)
	return alamos.NewTracer(alamos.TracingConfig{
		OtelProvider:   otel.GetTracerProvider(),
		OtelPropagator: otel.GetTextMapPropagator(),
	})
}

func newLogger() (*alamos.Logger, error) {
	return alamos.NewLogger(alamos.LoggerConfig{ZapConfig: zap.NewDevelopmentConfig()})
}

// shutdownUptrace stops the process-global OpenTelemetry SDK that newTracer configured.
// uptrace.Shutdown flushes buffered spans and metrics to the dev collector (devDSN) as
// part of shutting down; that collector is not running during tests, so the flush fails
// with a connection error. Stopping the SDK's background goroutines does not depend on
// the flush succeeding, so the upload error is expected and intentionally dropped here —
// surfacing it would fail every suite that enables tracing.
func shutdownUptrace(ctx context.Context) error {
	_ = uptrace.Shutdown(ctx)
	return nil
}

// OpenInstrumentation builds Instrumentation from the given config and returns it
// alongside no separate closer: the returned Instrumentation is itself an io.Closer.
// When tracing is enabled it configures the process-global OpenTelemetry SDK and
// registers uptrace.Shutdown as a shutdown function, so Closing the Instrumentation tears
// the SDK exporter back down. Pair it with MustOpen / DeferClose in tests.
func OpenInstrumentation(
	key string,
	cfgs ...InstrumentationConfig,
) (alamos.Instrumentation, error) {
	cfg, err := config.New(DefaultInstrumentationConfig, cfgs...)
	if err != nil {
		return alamos.Instrumentation{}, err
	}
	var options []alamos.Option
	if *cfg.Trace {
		tracer, err := newTracer(serviceName())
		if err != nil {
			return alamos.Instrumentation{}, err
		}
		options = append(
			options,
			alamos.WithTracer(tracer),
			alamos.WithShutdown(shutdownUptrace),
		)
	}
	if *cfg.Log {
		logger, err := newLogger()
		if err != nil {
			return alamos.Instrumentation{}, err
		}
		options = append(options, alamos.WithLogger(logger))
	}
	if *cfg.Report {
		reporter, err := alamos.NewReporter()
		if err != nil {
			return alamos.Instrumentation{}, err
		}
		options = append(options, alamos.WithReporter(reporter))
	}
	return alamos.New(key, options...), nil
}

// ObservedInstrumentation returns an Instrumentation backed by a zap observer that
// captures all log entries at or above the given level. Use the returned
// *observer.ObservedLogs to assert on log output in tests.
func ObservedInstrumentation(
	level zapcore.Level,
) (alamos.Instrumentation, *observer.ObservedLogs) {
	core, logs := observer.New(level)
	l := testutil.MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
		ZapLogger: zap.New(core),
	}))
	return alamos.New("test", alamos.WithLogger(l)), logs
}

// PanicLogger returns an Instrumentation instance that only contains a logger that only
// logs above PanicLevel and panics on DPanic.
func PanicLogger() alamos.Instrumentation {
	cfg := zap.NewDevelopmentConfig()
	cfg.Level.SetLevel(zap.PanicLevel)
	l := testutil.MustSucceed(alamos.NewLogger(alamos.LoggerConfig{ZapConfig: cfg}))
	return alamos.New(
		fmt.Sprintf("synnax-testing-%s", uuid.New().String()),
		alamos.WithLogger(l),
	)
}
