// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package transport binds the transport-agnostic API layer
// (github.com/synnaxlabs/synnax/pkg/api) to Synnax's supported wire protocols. The
// protocol-specific implementations live in the http and grpc sub-packages; this
// package composes them into a single Layer that mirrors the storage, distribution,
// service, and api layers.
package transport

import (
	"github.com/synnaxlabs/alamos"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	fhttp "github.com/synnaxlabs/freighter/http"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/transport/grpc"
	"github.com/synnaxlabs/synnax/pkg/transport/http"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// LayerConfig is all required configuration parameters and services necessary to bind
// the API layer to its supported transport protocols.
type LayerConfig struct {
	alamos.Instrumentation
	// API is the API layer whose service handlers are bound to each protocol.
	API *api.Layer
	// Router is the HTTP router onto which API HTTP endpoints are registered. It is
	// shared with other HTTP branches (e.g. the embedded console), so the transport
	// layer binds onto it rather than owning it.
	Router *fhttp.Router
	// Channel resolves channel keys to data types for the frame codec used by both the
	// HTTP and gRPC framer endpoints.
	Channel *channel.Service
}

var _ config.Config[LayerConfig] = LayerConfig{}

// Validate implements config.Config.
func (c LayerConfig) Validate() error {
	v := validate.New("transport")
	validate.NotNil(v, "api", c.API)
	validate.NotNil(v, "router", c.Router)
	validate.NotNil(v, "channel", c.Channel)
	return v.Error()
}

// Override implements config.Config.
func (c LayerConfig) Override(other LayerConfig) LayerConfig {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.API = override.Nil(c.API, other.API)
	c.Router = override.Nil(c.Router, other.Router)
	c.Channel = override.Nil(c.Channel, other.Channel)
	return c
}

// Layer binds the API layer to Synnax's supported transport protocols. HTTP endpoints
// are registered directly onto the configured router; gRPC services are exposed via
// GRPC for registration with the server's gRPC branch.
type Layer struct {
	// GRPC holds the bindable gRPC transports that must be registered with the server's
	// gRPC branch.
	GRPC []fgrpc.BindableTransport
}

// NewLayer binds the configured API layer to the HTTP router and gRPC transports,
// returning a Layer that exposes the gRPC transports for server registration.
func NewLayer(cfgs ...LayerConfig) (Layer, error) {
	cfg, err := config.New(LayerConfig{}, cfgs...)
	if err != nil {
		return Layer{}, err
	}
	http.Bind(cfg.API, cfg.Router, cfg.Channel)
	return Layer{GRPC: grpc.Bind(cfg.API, cfg.Channel)}, nil
}
