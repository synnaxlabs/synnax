// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cmd

import (
	"context"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/synnax/cmd/internal/invariants"
	"github.com/uptrace/uptrace-go/uptrace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"time"
)

func configureInstrumentation(version string) (alamos.Instrumentation, *zap.Logger) {
	logger, err := configureLogger()
	if err != nil {
		log.Fatal(err)
	}
	tracer, err := configureTracer(version, logger)
	if err != nil {
		log.Fatal(err)
	}
	return alamos.New(
		"sy",
		alamos.WithLogger(logger),
		alamos.WithTracer(tracer),
	), newPrettyLogger()
}

func cleanupInstrumentation(ctx context.Context, i alamos.Instrumentation) {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	// Force flush to uptrace, so we can trace the shutdown life cycle
	if err := uptrace.ForceFlush(ctx); err != nil {
		i.L.Info("failed to flush instrumentation", zap.Error(err))
	}
}

func configureLogger() (logger *alamos.Logger, err error) {
	verbose := viper.GetBool(verboseFlag)
	debug := viper.GetBool(debugFlag)
	var cfg zap.Config
	if debug {
		cfg = zap.NewDevelopmentConfig()
	} else {
		cfg = zap.NewProductionConfig()
	}
	cfg.Development = invariants.IsDevelopment

	if verbose || debug {
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		cfg.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05.000")
		cfg.Encoding = "console"
		cfg.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	}
	if !debug {
		cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
		cfg.DisableStacktrace = true
		cfg.DisableCaller = true
	}
	logger, err = alamos.NewLogger(alamos.LoggerConfig{ZapConfig: cfg})
	if err != nil {
		return
	}

	zap.ReplaceGlobals(logger.Zap())
	return
}

func newPrettyLogger() *zap.Logger {
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	cfg.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05.000")
	cfg.Encoding = "console"
	cfg.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	cfg.DisableStacktrace = true
	logger, _ := cfg.Build()
	return logger
}

func configureTracer(version string, logger *alamos.Logger) (*alamos.Tracer, error) {
	return nil, nil
	//uptrace.ConfigureOpentelemetry(
	//	uptrace.WithDSN("http://synnax_dev@localhost:14317/2"),
	//	uptrace.WithServiceName("synnax"),
	//	uptrace.WithServiceVersion(version),
	//)
	//otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
	//	logger.Info("opentelemetry", alamos.DebugError(err))
	//}))
	//return alamos.NewTracer(alamos.TracingConfig{
	//	OtelProvider:   otel.GetTracerProvider(),
	//	OtelPropagator: otel.GetTextMapPropagator(),
	//})
}
