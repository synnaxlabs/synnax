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

// InstrumentationMiddlewareConfig is the configuration for InstrumentationMiddleware.
type InstrumentationMiddlewareConfig struct {
	alamos.Instrumentation
	// EnableTracing sets whether the middleware starts traces. Defaults to true.
	// [OPTIONAL]
	EnableTracing *bool
	// EnablePropagation sets whether the middleware propagates any traces attached
	// to the request context. Defaults to true.
	// [OPTIONAL]
	EnablePropagation *bool
}

// Validate implements config.Config
func (c InstrumentationMiddlewareConfig) Validate() error {
	v := validate.New("falamos.InstrumentationMiddlewareConfig")
	validate.NotNil(v, "Instrumentation", c.Instrumentation)
	return nil
}

// Override implements config.Config
func (c InstrumentationMiddlewareConfig) Override(other InstrumentationMiddlewareConfig) InstrumentationMiddlewareConfig {
	c.Instrumentation = override.Nil(c.Instrumentation, other.Instrumentation)
	return c
}

var _ config.Config[InstrumentationMiddlewareConfig] = InstrumentationMiddlewareConfig{}

// DefaultTracingMiddlewareConfig is the default configuration for the tracing middleware.
var DefaultTracingMiddlewareConfig = InstrumentationMiddlewareConfig{
	EnableTracing:     config.BoolPointer(true),
	EnablePropagation: config.BoolPointer(true),
}

// InstrumentationMiddleware adds traces to incoming and outgoing requests and ensures that they
// are propagated across the network.
func InstrumentationMiddleware(configs ...InstrumentationMiddlewareConfig) (freighter.Middleware, error) {
	cfg, err := config.OverrideAndValidate(DefaultTracingMiddlewareConfig, configs...)
	if err != nil {
		return nil, err
	}
	prop := alamos.ExtractTracer(cfg).Propagator
	return freighter.MiddlewareFunc(func(
		ctx freighter.Context,
		next freighter.Next,
	) (freighter.Context, error) {
		var (
			span     alamos.Span
			carrier_ = carrier{Context: ctx}
		)

		if *cfg.EnablePropagation && ctx.Location == freighter.ServerSide {
			ctx.Context = prop.Extract(ctx.Context, carrier_)
		}

		if *cfg.EnableTracing {
			ctx.Context, span = alamos.TraceI(ctx.Context, cfg, ctx.Target.String())
		}

		if *cfg.EnableTracing && ctx.Location == freighter.ClientSide {
			prop.Inject(ctx, carrier_)
		}

		oCtx, err := next(ctx)

		if *cfg.EnableTracing {
			span.EndWith(err)
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
