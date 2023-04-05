package falamos

import (
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
	"go.opentelemetry.io/otel/propagation"
	"strings"
)

// Config is the configuration for New.
type Config struct {
	alamos.Instrumentation
	// EnableTracing sets whether the middleware starts traces. Defaults to true.
	// [OPTIONAL]
	EnableTracing *bool
	// EnablePropagation sets whether the middleware propagates any traces attached
	// to the request context. Defaults to true.
	// [OPTIONAL]
	EnablePropagation *bool
	// EnableLogging sets whether the middleware logs the trace. Defaults to true.
	// [OPTIONAL]
	EnableLogging *bool
	// Level is the level of the trace. Defaults to alamos.Info.
	// [OPTIONAL]
	Level alamos.Level
}

// Validate implements config.Config
func (c Config) Validate() error {
	v := validate.New("falamos.Config")
	validate.NotNil(v, "Instrumentation", c.Instrumentation)
	return nil
}

// Override implements config.Config
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Nil(c.Instrumentation, other.Instrumentation)
	return c
}

var _ config.Config[Config] = Config{}

// DefaultTracingMiddlewareConfig is the default configuration for the tracing middleware.
var DefaultTracingMiddlewareConfig = Config{
	EnableTracing:     config.True(),
	EnablePropagation: config.True(),
}

// New adds traces to incoming and outgoing requests and ensures that they
// are propagated across the network.
func New(configs ...Config) (freighter.Middleware, error) {
	cfg, err := config.New(DefaultTracingMiddlewareConfig, configs...)
	if err != nil {
		return nil, err
	}
	return freighter.MiddlewareFunc(func(
		ctx freighter.Context,
		next freighter.Next,
	) (freighter.Context, error) {
		var (
			span     alamos.Span
			carrier_ = carrier{Context: ctx}
		)

		if *cfg.EnablePropagation && ctx.Location == freighter.ServerSide {
			ctx.Context = alamos.Depropagate(ctx, carrier_)
		}

		if *cfg.EnableTracing {
			ctx.Context, span = cfg.T.Trace(ctx.Context, ctx.Target.String(), cfg.Level)
		}

		if *cfg.EnableTracing && ctx.Location == freighter.ClientSide {
			alamos.Propagate(ctx, carrier_)
		}

		oCtx, err := next(ctx)

		if *cfg.EnableTracing {
			_ = span.EndWith(err)
		}

		return oCtx, err
	}), nil
}

type carrier struct {
	freighter.Context
}

var _ propagation.TextMapCarrier = carrier{}

const keyPrefix = "alamos"

func keyF(k string) string {
	return keyPrefix + "-" + k
}

func (c carrier) Get(key string) string {
	v, ok := c.Context.Get(keyF(key))
	if !ok {
		return ""
	}
	vStr, ok := v.(string)
	if !ok {
		return ""
	}
	return vStr
}

func (c carrier) Set(key, value string) {
	c.Context.Params.Set(keyF(key), value)
}

func (c carrier) Keys() []string {
	keys := make([]string, 0, len(c.Context.Params))
	for k, _ := range c.Context.Params {
		if strings.HasPrefix(k, keyPrefix) {
			keys = append(keys, strings.TrimPrefix(k, keyPrefix+"-"))
		}
	}
	return keys
}
