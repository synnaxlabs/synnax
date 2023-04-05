// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package api implements the client interfaces for interacting with the delta cluster.
// The top level package is completely transport agnostic, and provides freighter compatible
// interfaces for all of its services. sub-packages in this directory wrap the core API
// services to provide transport specific implementations.
package api

import (
	"context"
	"github.com/synnaxlabs/alamos"
	"go/types"

	dcore "github.com/synnaxlabs/synnax/pkg/distribution/core"

	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/synnax/pkg/access"
	"github.com/synnaxlabs/synnax/pkg/api/errors"
	"github.com/synnaxlabs/synnax/pkg/auth"
	"github.com/synnaxlabs/synnax/pkg/auth/token"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/synnax/pkg/user"
)

// Config is all required configuration parameters and services necessary to
// instantiate the API.
type Config struct {
	alamos.Instrumentation
	Channel       channel.Service
	Framer        *framer.Service
	Ontology      *ontology.Ontology
	Storage       *storage.Store
	User          *user.Service
	Token         *token.Service
	Authenticator auth.Authenticator
	Enforcer      access.Enforcer
	Cluster       dcore.Cluster
	Insecure      bool
}

type Transport struct {
	AuthLogin          freighter.UnaryServer[auth.InsecureCredentials, TokenResponse]
	AuthChangeUsername freighter.UnaryServer[ChangeUsernameRequest, types.Nil]
	AuthChangePassword freighter.UnaryServer[ChangePasswordRequest, types.Nil]
	AuthRegistration   freighter.UnaryServer[RegistrationRequest, TokenResponse]
	ChannelCreate      freighter.UnaryServer[ChannelCreateRequest, ChannelCreateResponse]
	ChannelRetrieve    freighter.UnaryServer[ChannelRetrieveRequest, ChannelRetrieveResponse]
	ConnectivityCheck  freighter.UnaryServer[types.Nil, ConnectivityCheckResponse]
	FrameWriter        freighter.StreamServer[FrameWriterRequest, FrameWriterResponse]
	FrameReader        freighter.StreamServer[FrameIteratorRequest, FrameIteratorResponse]
	OntologyRetrieve   freighter.UnaryServer[OntologyRetrieveRequest, OntologyRetrieveResponse]
}

// API wraps all implemented API services into a single container. Protocol-specific
// API implementations should use this struct during instantiation.
type API struct {
	provider     Provider
	config       Config
	Auth         *AuthService
	Segment      *FrameService
	Channel      *ChannelService
	Connectivity *ConnectivityService
	Ontology     *OntologyService
}

// BindTo binds the API to the provided Transport implementation.
func (a *API) BindTo(t Transport) {
	var (
		err                = errors.Middleware()
		tk                 = tokenMiddleware(a.provider.auth.token)
		insecureMiddleware = []freighter.Middleware{err}
		secureMiddleware   = make([]freighter.Middleware, len(insecureMiddleware))
	)
	copy(secureMiddleware, insecureMiddleware)
	if !a.config.Insecure {
		secureMiddleware = append(secureMiddleware, tk)
	}

	freighter.UseOnAll(
		insecureMiddleware,
		t.AuthRegistration,
		t.AuthLogin,
	)

	freighter.UseOnAll(
		secureMiddleware,
		t.AuthChangeUsername,
		t.AuthChangePassword,
		t.ChannelCreate,
		t.ChannelRetrieve,
		t.ConnectivityCheck,
		t.FrameWriter,
		t.FrameReader,
		t.ConnectivityCheck,
		t.OntologyRetrieve,
	)

	t.AuthLogin.BindHandler(typedUnaryWrapper(a.Auth.Login))
	t.AuthChangeUsername.BindHandler(noResponseWrapper(a.Auth.ChangeUsername))
	t.AuthChangePassword.BindHandler(noResponseWrapper(a.Auth.ChangePassword))
	t.AuthRegistration.BindHandler(typedUnaryWrapper(a.Auth.Register))
	t.ChannelCreate.BindHandler(typedUnaryWrapper(a.Channel.Create))
	t.ChannelRetrieve.BindHandler(typedUnaryWrapper(a.Channel.Retrieve))
	t.ConnectivityCheck.BindHandler(typedUnaryWrapper(a.Connectivity.Check))
	t.FrameWriter.BindHandler(typedStreamWrapper(a.Segment.Write))
	t.FrameReader.BindHandler(typedStreamWrapper(a.Segment.Iterate))
	t.OntologyRetrieve.BindHandler(typedUnaryWrapper(a.Ontology.Retrieve))
}

// New instantiates the delta API using the provided Config. This should probably
// only be called once.
func New(cfg Config) API {
	api := API{config: cfg, provider: NewProvider(cfg)}
	api.Auth = NewAuthServer(api.provider)
	api.Segment = NewSegmentService(api.provider)
	api.Channel = NewChannelService(api.provider)
	api.Connectivity = NewConnectivityService(api.provider)
	api.Ontology = NewOntologyService(api.provider)
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
