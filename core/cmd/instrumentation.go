// Copyright 2026 Synnax Labs, Inc.
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
	"log"
	"os"
	"time"

	"github.com/samber/lo"
	"github.com/spf13/viper"
	"github.com/synnaxlabs/alamos"
	"github.com/uptrace/uptrace-go/uptrace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

func configureInstrumentation() alamos.Instrumentation {
	logger, err := configureLogger()
	if err != nil {
		log.Fatal(err)
	}
	return alamos.New("sy", alamos.WithLogger(logger))
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
	var (
		verbose              = viper.GetBool(verboseFlag)
		debug                = viper.GetBool(debugFlag)
		opts                 = []zap.Option{zap.AddStacktrace(zap.ErrorLevel), zap.AddCaller()}
		consoleEncoderConfig = zap.NewProductionEncoderConfig()
		fileEncoderConfig    = zap.NewProductionEncoderConfig()
		consoleOutput        = zapcore.Lock(os.Stdout)
		fileOutput           = zapcore.AddSync(&lumberjack.Logger{
			Filename:   viper.GetString(logFilePathFlag),
			MaxSize:    viper.GetInt(logFileMaxSizeFlag),
			MaxBackups: viper.GetInt(logFileMaxBackupsFlag),
			MaxAge:     viper.GetInt(logFileMaxAgeFlag),
			Compress:   viper.GetBool(logFileCompressFlag),
		})
		level = lo.Ternary(verbose, zap.DebugLevel, zap.InfoLevel)
	)

	if debug {
		opts = append(opts, zap.Development())
	}

	consoleEncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	consoleEncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout("01-02 15:04:05.000")
	consoleEncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder

	fileEncoder := zapcore.NewJSONEncoder(fileEncoderConfig)
	consoleEncoder := zapcore.NewConsoleEncoder(consoleEncoderConfig)

	core := zapcore.NewTee(
		alamos.CustomZapCore(zapcore.NewCore(consoleEncoder, consoleOutput, level)),
		alamos.CustomZapCore(zapcore.NewCore(fileEncoder, fileOutput, level)),
	)
	if logger, err = alamos.NewLogger(alamos.LoggerConfig{
		ZapLogger: zap.New(core, opts...),
	}); err != nil {
		return
	}
	zap.ReplaceGlobals(logger.Zap())
	return
}
