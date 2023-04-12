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
	"github.com/spf13/viper"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/errutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"log"
)

func configureInstrumentation() alamos.Instrumentation {
	c := errutil.NewCatch()
	logger := errutil.Exec1(c, configureLogger)
	tracer := errutil.Exec1(c, configureTracer)
	if c.Error() != nil {
		log.Fatal(c.Error())
	}
	return alamos.New(
		"synnax",
		alamos.WithLogger(logger),
		alamos.WithTracer(tracer),
	)
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

func configureTracer() (tracer *alamos.Tracer, err error) {
	return tracer, err
}
