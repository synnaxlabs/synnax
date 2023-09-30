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
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter/falamos"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/group"
	"github.com/synnaxlabs/synnax/pkg/ranger"
	"github.com/synnaxlabs/synnax/pkg/workspace"
	"github.com/synnaxlabs/synnax/pkg/workspace/lineplot"
	"github.com/synnaxlabs/synnax/pkg/workspace/pid"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
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
	Ranger        *ranger.Service
	Framer        *framer.Service
	Ontology      *ontology.Ontology
	Group         *group.Service
	Storage       *storage.Storage
	User          *user.Service
	Workspace     *workspace.Service
	PID           *pid.Service
	LinePlot      *lineplot.Service
	Token         *token.Service
	Authenticator auth.Authenticator
	Enforcer      access.Enforcer
	Cluster       dcore.Cluster
	Insecure      *bool
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("api")
	validate.NotNil(v, "channel", c.Channel)
	validate.NotNil(v, "ranger", c.Ranger)
	validate.NotNil(v, "framer", c.Framer)
	validate.NotNil(v, "ontology", c.Ontology)
	validate.NotNil(v, "storage", c.Storage)
	validate.NotNil(v, "user", c.User)
	validate.NotNil(v, "workspace", c.Workspace)
	validate.NotNil(v, "token", c.Token)
	validate.NotNil(v, "authenticator", c.Authenticator)
	validate.NotNil(v, "enforcer", c.Enforcer)
	validate.NotNil(v, "cluster", c.Cluster)
	validate.NotNil(v, "group", c.Group)
	validate.NotNil(v, "pid", c.PID)
	validate.NotNil(v, "insecure", c.Insecure)
	return v.Error()
}

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Channel = override.Nil(c.Channel, other.Channel)
	c.Ranger = override.Nil(c.Ranger, other.Ranger)
	c.Framer = override.Nil(c.Framer, other.Framer)
	c.Ontology = override.Nil(c.Ontology, other.Ontology)
	c.Storage = override.Nil(c.Storage, other.Storage)
	c.User = override.Nil(c.User, other.User)
	c.Workspace = override.Nil(c.Workspace, other.Workspace)
	c.Token = override.Nil(c.Token, other.Token)
	c.Authenticator = override.Nil(c.Authenticator, other.Authenticator)
	c.Enforcer = override.Nil(c.Enforcer, other.Enforcer)
	c.Cluster = override.Nil(c.Cluster, other.Cluster)
	c.Insecure = override.Nil(c.Insecure, other.Insecure)
	c.Group = override.Nil(c.Group, other.Group)
	c.Insecure = override.Nil(c.Insecure, other.Insecure)
	c.PID = override.Nil(c.PID, other.PID)
	c.LinePlot = override.Nil(c.LinePlot, other.LinePlot)
	return c
}

type Transport struct {
	AuthLogin              freighter.UnaryServer[auth.InsecureCredentials, TokenResponse]
	AuthChangeUsername     freighter.UnaryServer[ChangeUsernameRequest, types.Nil]
	AuthChangePassword     freighter.UnaryServer[ChangePasswordRequest, types.Nil]
	AuthRegistration       freighter.UnaryServer[RegistrationRequest, TokenResponse]
	ChannelCreate          freighter.UnaryServer[ChannelCreateRequest, ChannelCreateResponse]
	ChannelRetrieve        freighter.UnaryServer[ChannelRetrieveRequest, ChannelRetrieveResponse]
	ConnectivityCheck      freighter.UnaryServer[types.Nil, ConnectivityCheckResponse]
	FrameWriter            freighter.StreamServer[FrameWriterRequest, FrameWriterResponse]
	FrameIterator          freighter.StreamServer[FrameIteratorRequest, FrameIteratorResponse]
	FrameStreamer          freighter.StreamServer[FrameStreamerRequest, FrameStreamerResponse]
	RangeCreate            freighter.UnaryServer[RangeCreateRequest, RangeCreateResponse]
	RangeRetrieve          freighter.UnaryServer[RangeRetrieveRequest, RangeRetrieveResponse]
	OntologyRetrieve       freighter.UnaryServer[OntologyRetrieveRequest, OntologyRetrieveResponse]
	OntologyGroupCreate    freighter.UnaryServer[OntologyCreateGroupRequest, OntologyCreateGroupResponse]
	OntologyGroupDelete    freighter.UnaryServer[OntologyDeleteGroupRequest, types.Nil]
	OntologyGroupRename    freighter.UnaryServer[OntologyRenameGroupRequest, types.Nil]
	OntologyAddChildren    freighter.UnaryServer[OntologyAddChildrenRequest, types.Nil]
	OntologyRemoveChildren freighter.UnaryServer[OntologyRemoveChildrenRequest, types.Nil]
	OntologyMoveChildren   freighter.UnaryServer[OntologyMoveChildrenRequest, types.Nil]
	WorkspaceCreate        freighter.UnaryServer[WorkspaceCreateRequest, WorkspaceCreateResponse]
	WorkspaceRetrieve      freighter.UnaryServer[WorkspaceRetrieveRequest, WorkspaceRetrieveResponse]
	WorkspaceDelete        freighter.UnaryServer[WorkspaceDeleteRequest, types.Nil]
	WorkspaceRename        freighter.UnaryServer[WorkspaceRenameRequest, types.Nil]
	WorkspaceSetLayout     freighter.UnaryServer[WorkspaceSetLayoutRequest, types.Nil]
	PIDCreate              freighter.UnaryServer[PIDCreateRequest, PIDCreateResponse]
	PIDRetrieve            freighter.UnaryServer[PIDRetrieveRequest, PIDRetrieveResponse]
	PIDDelete              freighter.UnaryServer[PIDDeleteRequest, types.Nil]
	PIDRename              freighter.UnaryServer[PIDRenameRequest, types.Nil]
	PIDSetData             freighter.UnaryServer[PIDSetDataRequest, types.Nil]
	LinePlotCreate         freighter.UnaryServer[LinePlotCreateRequest, LinePlotCreateResponse]
	LinePlotRename         freighter.UnaryServer[LinePlotRenameRequest, types.Nil]
	LinePlotSetData        freighter.UnaryServer[LinePlotSetDataRequest, types.Nil]
	LinePlotRetrieve       freighter.UnaryServer[LinePlotRetrieveRequest, LinePlotRetrieveResponse]
	LinePlotDelete         freighter.UnaryServer[LinePlotDeleteRequest, types.Nil]
}

// API wraps all implemented API services into a single container. Protocol-specific
// API implementations should use this struct during instantiation.
type API struct {
	provider     Provider
	config       Config
	Auth         *AuthService
	Telem        *FrameService
	Channel      *ChannelService
	Connectivity *ConnectivityService
	Ontology     *OntologyService
	Range        *RangeService
	Workspace    *WorkspaceService
	PID          *PIDService
	LinePlot     *LinePlotService
}

// BindTo binds the API to the provided Transport implementation.
func (a *API) BindTo(t Transport) {
	var (
		err                = errors.Middleware()
		tk                 = tokenMiddleware(a.provider.auth.token)
		instrumentation    = lo.Must(falamos.Middleware(falamos.Config{Instrumentation: a.config.Instrumentation}))
		insecureMiddleware = []freighter.Middleware{
			instrumentation,
			err,
		}
		secureMiddleware = make([]freighter.Middleware, len(insecureMiddleware))
	)
	copy(secureMiddleware, insecureMiddleware)
	//if !*a.config.Insecure {
	secureMiddleware = append(secureMiddleware, tk)
	//}

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
		t.FrameIterator,
		t.ConnectivityCheck,
		t.OntologyRetrieve,
		t.FrameStreamer,
		t.RangeCreate,
		t.RangeRetrieve,
		t.OntologyGroupCreate,
		t.OntologyGroupDelete,
		t.OntologyAddChildren,
		t.OntologyRemoveChildren,
		t.OntologyMoveChildren,
		t.OntologyGroupRename,
		t.WorkspaceDelete,
		t.WorkspaceCreate,
		t.WorkspaceRetrieve,
		t.WorkspaceRename,
		t.WorkspaceSetLayout,
		t.PIDCreate,
		t.PIDRetrieve,
		t.PIDDelete,
		t.PIDRename,
		t.PIDSetData,
		t.LinePlotCreate,
		t.LinePlotRename,
		t.LinePlotSetData,
		t.LinePlotRetrieve,
		t.LinePlotDelete,
	)

	t.AuthLogin.BindHandler(typedUnaryWrapper(a.Auth.Login))
	t.AuthChangeUsername.BindHandler(noResponseWrapper(a.Auth.ChangeUsername))
	t.AuthChangePassword.BindHandler(noResponseWrapper(a.Auth.ChangePassword))
	t.AuthRegistration.BindHandler(typedUnaryWrapper(a.Auth.Register))
	t.ChannelCreate.BindHandler(typedUnaryWrapper(a.Channel.Create))
	t.ChannelRetrieve.BindHandler(typedUnaryWrapper(a.Channel.Retrieve))
	t.ConnectivityCheck.BindHandler(typedUnaryWrapper(a.Connectivity.Check))
	t.FrameWriter.BindHandler(typedStreamWrapper(a.Telem.Write))
	t.FrameIterator.BindHandler(typedStreamWrapper(a.Telem.Iterate))
	t.OntologyRetrieve.BindHandler(typedUnaryWrapper(a.Ontology.Retrieve))
	t.FrameStreamer.BindHandler(typedStreamWrapper(a.Telem.Stream))
	t.RangeRetrieve.BindHandler(typedUnaryWrapper(a.Range.Retrieve))
	t.RangeCreate.BindHandler(typedUnaryWrapper(a.Range.Create))
	t.OntologyGroupCreate.BindHandler(typedUnaryWrapper(a.Ontology.CreateGroup))
	t.OntologyGroupDelete.BindHandler(typedUnaryWrapper(a.Ontology.DeleteGroup))
	t.OntologyGroupRename.BindHandler(typedUnaryWrapper(a.Ontology.RenameGroup))
	t.OntologyAddChildren.BindHandler(typedUnaryWrapper(a.Ontology.AddChildren))
	t.OntologyRemoveChildren.BindHandler(typedUnaryWrapper(a.Ontology.RemoveChildren))
	t.OntologyMoveChildren.BindHandler(typedUnaryWrapper(a.Ontology.MoveChildren))
	t.WorkspaceCreate.BindHandler(typedUnaryWrapper(a.Workspace.Create))
	t.WorkspaceDelete.BindHandler(typedUnaryWrapper(a.Workspace.Delete))
	t.WorkspaceRetrieve.BindHandler(typedUnaryWrapper(a.Workspace.Retrieve))
	t.WorkspaceDelete.BindHandler(typedUnaryWrapper(a.Workspace.Delete))
	t.WorkspaceCreate.BindHandler(typedUnaryWrapper(a.Workspace.Create))
	t.WorkspaceRename.BindHandler(typedUnaryWrapper(a.Workspace.Rename))
	t.WorkspaceSetLayout.BindHandler(typedUnaryWrapper(a.Workspace.SetLayout))
	t.PIDCreate.BindHandler(typedUnaryWrapper(a.PID.Create))
	t.PIDRetrieve.BindHandler(typedUnaryWrapper(a.PID.Retrieve))
	t.PIDDelete.BindHandler(typedUnaryWrapper(a.PID.Delete))
	t.PIDRename.BindHandler(typedUnaryWrapper(a.PID.Rename))
	t.PIDSetData.BindHandler(typedUnaryWrapper(a.PID.SetData))
	t.LinePlotCreate.BindHandler(typedUnaryWrapper(a.LinePlot.Create))
	t.LinePlotRename.BindHandler(typedUnaryWrapper(a.LinePlot.Rename))
	t.LinePlotSetData.BindHandler(typedUnaryWrapper(a.LinePlot.SetData))
	t.LinePlotRetrieve.BindHandler(typedUnaryWrapper(a.LinePlot.Retrieve))
	t.LinePlotDelete.BindHandler(typedUnaryWrapper(a.LinePlot.Delete))
}

// New instantiates the delta API using the provided Config. This should probably
// only be called once.
func New(configs ...Config) (API, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return API{}, err
	}
	api := API{config: cfg, provider: NewProvider(cfg)}
	api.Auth = NewAuthServer(api.provider)
	api.Telem = NewFrameService(api.provider)
	api.Channel = NewChannelService(api.provider)
	api.Connectivity = NewConnectivityService(api.provider)
	api.Ontology = NewOntologyService(api.provider)
	api.Range = NewRangeService(api.provider)
	api.Workspace = NewWorkspaceService(api.provider)
	api.PID = NewPIDService(api.provider)
	api.LinePlot = NewLinePlotService(api.provider)
	return api, nil
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
