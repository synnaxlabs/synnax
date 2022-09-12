// Package api implements the client interfaces for interacting with the delta cluster.
// The top level package is completely transport agnostic, and provides freighter compatible
// interfaces for all of its services. Sub-packages in this directory wrap the core API
// services to provide transport specific implementations.
package api

import (
	"github.com/synnaxlabs/synnax/pkg/access"
	"github.com/synnaxlabs/synnax/pkg/auth"
	"github.com/synnaxlabs/synnax/pkg/auth/token"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/segment"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/synnax/pkg/user"
	"go.uber.org/zap"
)

// Config is all required configuration parameters and services necessary to
// instantiate the API.
type Config struct {
	Logger        *zap.Logger
	Channel       *channel.Service
	Segment       *segment.Service
	Ontology      *ontology.Ontology
	Storage       *storage.Store
	User          *user.Service
	Token         *token.Service
	Authenticator auth.Authenticator
	Enforcer      access.Enforcer
}

// API wraps all implemented API services into a single container. Protocol-specific
// API implementations should use this struct during instantiation.
type API struct {
	Provider Provider
	Config   Config
	Auth     *AuthService
	Segment  *SegmentService
	Channel  *ChannelService
}

// New instantiates the delta API using the provided Config. This should probably
// only be called once.
func New(cfg Config) API {
	api := API{Config: cfg, Provider: NewProvider(cfg)}
	api.Auth = NewAuthService(api.Provider)
	api.Segment = NewSegmentService(api.Provider)
	api.Channel = NewChannelService(api.Provider)
	return api
}
