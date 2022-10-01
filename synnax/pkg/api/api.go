// Package api implements the client interfaces for interacting with the delta cluster.
// The top level package is completely transport agnostic, and provides freighter compatible
// interfaces for all of its services. Sub-packages in this directory wrap the core API
// services to provide transport specific implementations.
package api

import (
	"context"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/access"
	"github.com/synnaxlabs/synnax/pkg/api/errors"
	"github.com/synnaxlabs/synnax/pkg/auth"
	"github.com/synnaxlabs/synnax/pkg/auth/token"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/segment"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/synnax/pkg/user"
	"go.uber.org/zap"
	"go/types"
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

type Server struct {
	AuthLogin          freighter.UnaryServer[auth.InsecureCredentials, TokenResponse]
	AuthChangeUsername freighter.UnaryServer[ChangeUsernameRequest, types.Nil]
	AuthChangePassword freighter.UnaryServer[ChangePasswordRequest, types.Nil]
	AuthRegistration   freighter.UnaryServer[RegistrationRequest, TokenResponse]
	ChannelCreate      freighter.UnaryServer[ChannelCreateRequest, ChannelCreateResponse]
	ChannelRetrieve    freighter.UnaryServer[ChannelRetrieveRequest, ChannelRetrieveResponse]
	SegmentWriter      freighter.StreamServer[SegmentWriterRequest, SegmentWriterResponse]
	SegmentIterator    freighter.StreamServer[SegmentIteratorRequest, SegmentIteratorResponse]
}

type Client struct {
	AuthLogin          freighter.UnaryClient[auth.InsecureCredentials, TokenResponse]
	AuthChangeUsername freighter.UnaryClient[ChangeUsernameRequest, types.Nil]
	AuthChangePassword freighter.UnaryClient[ChangePasswordRequest, types.Nil]
	AuthRegistration   freighter.UnaryClient[RegistrationRequest, TokenResponse]
	ChannelCreate      freighter.UnaryClient[ChannelCreateRequest, ChannelCreateResponse]
	ChannelRetrieve    freighter.UnaryClient[ChannelRetrieveRequest, ChannelRetrieveResponse]
	SegmentWriter      freighter.StreamClient[SegmentWriterRequest, SegmentWriterResponse]
	SegmentIterator    freighter.StreamClient[SegmentIteratorRequest, SegmentIteratorResponse]
}

// API wraps all implemented API services into a single container. Protocol-specific
// API implementations should use this struct during instantiation.
type API struct {
	provider Provider
	config   Config
	Auth     *AuthService
	Segment  *SegmentService
	Channel  *ChannelService
}

func (a *API) BindTo(t Server) {
	logger := logMiddleware(a.provider.Logging.logger)
	tk := tokenMiddleware(a.provider.auth.token)
	t.AuthLogin.Use(logger, tk)
	t.AuthChangeUsername.Use(logger, tk)
	t.AuthChangePassword.Use(logger, tk)
	t.AuthRegistration.Use(logger, tk)
	t.ChannelCreate.Use(logger, tk)
	t.ChannelRetrieve.Use(logger, tk)
	t.SegmentWriter.Use(logger, tk)
	t.SegmentIterator.Use(logger, tk)

	t.AuthLogin.BindHandler(typedUnaryWrapper(a.Auth.Login))
	t.AuthChangeUsername.BindHandler(noResponseWrapper(a.Auth.ChangeUsername))
	t.AuthChangePassword.BindHandler(noResponseWrapper(a.Auth.ChangePassword))
	t.AuthRegistration.BindHandler(typedUnaryWrapper(a.Auth.Register))
	t.ChannelCreate.BindHandler(typedUnaryWrapper(a.Channel.Create))
	t.ChannelRetrieve.BindHandler(typedUnaryWrapper(a.Channel.Retrieve))
	t.SegmentWriter.BindHandler(typedStreamWrapper(a.Segment.Write))
	t.SegmentIterator.BindHandler(typedStreamWrapper(a.Segment.Iterate))
}

// New instantiates the delta API using the provided Config. This should probably
// only be called once.
func New(cfg Config) API {
	api := API{config: cfg, provider: NewProvider(cfg)}
	api.Auth = NewAuthServer(api.provider)
	api.Segment = NewSegmentService(api.provider)
	api.Channel = NewChannelService(api.provider)
	return api
}

func typedUnaryWrapper[RQ, RS freighter.Payload](
	handler func(context.Context, RQ) (RS, errors.Typed),
) func(context.Context, RQ) (RS, error) {
	return func(ctx context.Context, rq RQ) (RS, error) {
		return handler(ctx, rq)
	}
}

func typedStreamWrapper[RQ, RS freighter.Payload](
	handler func(context.Context, freighter.ServerStream[RQ, RS]) errors.Typed,
) func(context.Context, freighter.ServerStream[RQ, RS]) error {
	return func(ctx context.Context, stream freighter.ServerStream[RQ, RS]) error {
		return handler(ctx, stream)
	}
}

func noResponseWrapper[RQ freighter.Payload](
	handler func(ctx context.Context, rq RQ) (error errors.Typed),
) func(ctx context.Context, rq RQ) (rs types.Nil, error error) {
	return func(ctx context.Context, rq RQ) (types.Nil, error) {
		return types.Nil{}, handler(ctx, rq)
	}
}
