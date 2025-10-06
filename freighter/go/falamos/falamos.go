// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package falamos

import (
	"context"
	"strings"

	"go.uber.org/zap"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// Config is the configuration for the instrumentation Middleware.
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
	// Level is the level of the trace. Defaults to alamos.Prod.
	// [OPTIONAL]
	Level alamos.Environment
}

// Validate implements config.Config
func (cfg Config) Validate() error {
	v := validate.New("falamos.Properties")
	validate.NotNil(v, "Instrumentation", cfg.Instrumentation)
	return v.Error()
}

// Override implements config.Config
func (cfg Config) Override(other Config) Config {
	cfg.Instrumentation = override.Zero(cfg.Instrumentation, other.Instrumentation)
	cfg.EnablePropagation = override.Nil(cfg.EnableLogging, other.EnableLogging)
	cfg.EnableLogging = override.Nil(cfg.EnablePropagation, other.EnablePropagation)
	cfg.EnableTracing = override.Nil(cfg.EnableTracing, other.EnableTracing)
	return cfg
}

var _ config.Config[Config] = Config{}

// Default is the default configuration for the tracing middleware.
var Default = Config{
	Level:             alamos.Prod,
	EnableTracing:     config.True(),
	EnablePropagation: config.True(),
	EnableLogging:     config.True(),
}

// Middleware adds traces and logs to incoming and outgoing requests and ensures
// that they are propagated across the network.
func Middleware(configs ...Config) (freighter.Middleware, error) {
	cfg, err := config.New(Default, configs...)
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

		if *cfg.EnablePropagation && ctx.Role == freighter.Server {
			ctx.Context = cfg.T.Depropagate(ctx, carrier_)
		}

		if *cfg.EnableTracing {
			ctx.Context, span = cfg.T.Trace(ctx.Context, ctx.Target.String(), cfg.Level)
		}

		if *cfg.EnablePropagation && ctx.Role == freighter.Client {
			cfg.T.Propagate(ctx, carrier_)
		}

		oCtx, err := next(ctx)

		if *cfg.EnableLogging {
			log(ctx, err, cfg)
		}

		if *cfg.EnableTracing {
			_ = span.EndWith(err)
		}

		return oCtx, err
	}), nil
}

type carrier struct{ freighter.Context }

var _ alamos.TraceCarrier = carrier{}

const keyPrefix = "alamos"

func keyF(k string) string { return keyPrefix + "-" + k }

// Get implements TextMapCarrier.
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

// Set implements TextMapCarrier.
func (c carrier) Set(key, value string) { c.Params.Set(keyF(key), value) }

// Keys implements TextMapCarrier.
func (c carrier) Keys() []string {
	keys := make([]string, 0, len(c.Params))
	for k := range c.Params {
		if strings.HasPrefix(k, keyPrefix) {
			keys = append(keys, strings.TrimPrefix(k, keyPrefix+"-"))
		}
	}
	return keys
}

func log(ctx freighter.Context, err error, cfg Config) {
	args := []zap.Field{
		zap.String("protocol", ctx.Protocol),
		zap.Stringer("variant", ctx.Variant),
		zap.Stringer("role", ctx.Role),
	}
	// context.Canceled is returned when the client abruptly closes the connection,
	// which happens when performing tasks like reloading web pages. As such,
	// we don't consider it anomalous and don't log it.
	if errors.Skip(err, context.Canceled) != nil {
		cfg.L.Warn(ctx.Target.String(), append(args, zap.String("error", err.Error()))...)
	} else {
		cfg.L.Debug(ctx.Target.String(), args...)
	}
}
