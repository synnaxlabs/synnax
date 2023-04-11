package testutil

import (
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/git"
	"github.com/synnaxlabs/x/override"
	"github.com/uptrace/uptrace-go/uptrace"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"os"
)

type InstrumentationConfig struct {
	// Trace enables tracing for this instrumentation.
	Trace *bool
	// Log enables logging for this instrumentation.
	Log *bool
	// Report enables reports for this instrumentation.
	Report *bool
}

var (
	_                            config.Config[InstrumentationConfig] = InstrumentationConfig{}
	DefaultInstrumentationConfig                                      = InstrumentationConfig{
		Trace:  config.Bool(false),
		Log:    config.Bool(false),
		Report: config.Bool(false),
	}
)

func (c InstrumentationConfig) Validate() error { return nil }

func (c InstrumentationConfig) Override(other InstrumentationConfig) InstrumentationConfig {
	c.Report = override.Nil(c.Report, other.Report)
	c.Log = override.Nil(c.Log, other.Log)
	c.Trace = override.Nil(c.Trace, other.Trace)
	return c
}

func serviceName() string {
	host, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	return host
}

const devDSN = "http://project2_secret_token@localhost:14317/2"

func newTracer(serviceName string) *alamos.Tracer {
	uptrace.ConfigureOpentelemetry(
		uptrace.WithDSN(devDSN),
		uptrace.WithServiceName(serviceName),
		uptrace.WithServiceVersion(lo.Must(git.CurrentCommit())),
	)
	return MustSucceed(alamos.NewTracer(alamos.TracingConfig{
		Provider:   otel.GetTracerProvider(),
		Propagator: otel.GetTextMapPropagator(),
	}))
}

func newLogger() *alamos.Logger {
	return MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
		Zap: MustSucceed(zap.NewDevelopment()),
	}))
}

func newReports() *alamos.Reporter {
	return MustSucceed(alamos.NewReporter())
}

func Instrumentation(key string, configs ...InstrumentationConfig) alamos.Instrumentation {
	cfg := MustSucceed(config.New(DefaultInstrumentationConfig, configs...))
	var options []alamos.Option
	if *cfg.Trace {
		options = append(options, alamos.WithTracer(newTracer(serviceName())))
	}
	if *cfg.Log {
		options = append(options, alamos.WithLogger(newLogger()))
	}
	if *cfg.Report {
		options = append(options, alamos.WithReports(newReports()))
	}

	return alamos.New(key, options...)
}
