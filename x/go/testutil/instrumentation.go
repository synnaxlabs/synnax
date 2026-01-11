// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil

import (
	"fmt"
	"os"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/git"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/uuid"
	"github.com/uptrace/uptrace-go/uptrace"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
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
	Trace:  config.False(),
	Log:    config.False(),
	Report: config.False(),
}

func serviceName() string { return lo.Must(os.Hostname()) }

const devDSN = "http://synnax_dev@localhost:14317/2"

func newTracer(serviceName string) *alamos.Tracer {
	uptrace.ConfigureOpentelemetry(
		uptrace.WithDSN(devDSN),
		uptrace.WithServiceName(serviceName),
		uptrace.WithServiceVersion(lo.Must(git.CurrentCommit())),
	)
	return MustSucceed(alamos.NewTracer(alamos.TracingConfig{
		OtelProvider:   otel.GetTracerProvider(),
		OtelPropagator: otel.GetTextMapPropagator(),
	}))
}

func newLogger() *alamos.Logger {
	return MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
		ZapConfig: zap.NewDevelopmentConfig(),
	}))
}

func newReports() *alamos.Reporter { return MustSucceed(alamos.NewReporter()) }

func Instrumentation(key string, cfgs ...InstrumentationConfig) alamos.Instrumentation {
	cfg, err := config.New(DefaultInstrumentationConfig, cfgs...)
	if err != nil {
		zap.S().Fatal(err)
	}
	var options []alamos.Option
	if *cfg.Trace {
		options = append(options, alamos.WithTracer(newTracer(serviceName())))
	}
	if *cfg.Log {
		options = append(options, alamos.WithLogger(newLogger()))
	}
	if *cfg.Report {
		options = append(options, alamos.WithReporter(newReports()))
	}
	return alamos.New(key, options...)
}

// PanicLogger returns an Instrumentation instance that only contains a logger that only
// logs above PanicLevel and panics on DPanic.
func PanicLogger() alamos.Instrumentation {
	cfg := zap.NewDevelopmentConfig()
	cfg.Level.SetLevel(zap.PanicLevel)
	l := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{ZapConfig: cfg}))
	return alamos.New(
		fmt.Sprintf("synnax-testing-%s", uuid.New().String()),
		alamos.WithLogger(l),
	)
}
