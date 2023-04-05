package cmd

import (
	"github.com/spf13/viper"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/errutil"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func configureInstrumentation() (ins alamos.Instrumentation, err error) {
	c := errutil.NewCatch()
	logger := errutil.Exec1(c, configureLogger)
	tracer := errutil.Exec1(c, configureTracer)
	return alamos.New(
		"synnax",
		alamos.WithLogger(logger),
		alamos.WithTracer(tracer),
	), c.Error()
}

func configureLogger() (*alamos.Logger, error) {
	var cfg zap.Config
	verbose := viper.GetBool("verbose")
	if verbose {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
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
