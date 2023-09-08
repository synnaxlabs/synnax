// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/git"
	"github.com/uptrace/uptrace-go/uptrace"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
	"time"
)

func configureInstrumentation() alamos.Instrumentation {
	logger, err := configureLogger()
	if err != nil {
		log.Fatal(err)
	}
	tracer, err := configureTracer(logger)
	if err != nil {
		log.Fatal(err)
	}
	return alamos.New(
		"sy",
		alamos.WithLogger(logger),
		alamos.WithTracer(tracer),
	)
}

func cleanupInstrumentation(ctx context.Context, i alamos.Instrumentation) {
	ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	// Force flush to uptrace, so we can trace the shutdown life cycle
	if err := uptrace.ForceFlush(ctx); err != nil {
		i.L.Info("failed to flush instrumentation", zap.Error(err))
	}
}

func configureLogger() (*alamos.Logger, error) {
	verbose := viper.GetBool("verbose")
	var cfg zap.Config
	if verbose {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		cfg.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05.000")
		cfg.Encoding = "console"
	} else {
		cfg = zap.NewProductionConfig()
		cfg.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	}
	z, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	return alamos.NewLogger(alamos.LoggerConfig{Zap: z})
}

func configureTracer(logger *alamos.Logger) (*alamos.Tracer, error) {
	commit, err := git.CurrentCommit()
	if err != nil {
		commit = "unknown"
	}
	uptrace.ConfigureOpentelemetry(
		uptrace.WithDSN("http://synnax_dev@localhost:14317/2"),
		uptrace.WithServiceName("synnax"),
		uptrace.WithServiceVersion(commit),
	)
	otel.SetErrorHandler(otel.ErrorHandlerFunc(func(err error) {
		logger.Info("opentelemetry", alamos.DebugError(err))
	}))
	return alamos.NewTracer(alamos.TracingConfig{
		OtelProvider:   otel.GetTracerProvider(),
		OtelPropagator: otel.GetTextMapPropagator(),
	})
}
