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
	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/falamos"
	"github.com/synnaxlabs/synnax/pkg/access"
	"github.com/synnaxlabs/synnax/pkg/auth"
	"github.com/synnaxlabs/synnax/pkg/auth/token"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	dcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology/group"
	"github.com/synnaxlabs/synnax/pkg/hardware"
	"github.com/synnaxlabs/synnax/pkg/label"
	"github.com/synnaxlabs/synnax/pkg/ranger"
	"github.com/synnaxlabs/synnax/pkg/storage"
	"github.com/synnaxlabs/synnax/pkg/user"
	"github.com/synnaxlabs/synnax/pkg/workspace"
	"github.com/synnaxlabs/synnax/pkg/workspace/lineplot"
	"github.com/synnaxlabs/synnax/pkg/workspace/schematic"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
	"go/types"
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
	Schematic     *schematic.Service
	LinePlot      *lineplot.Service
	Token         *token.Service
	Label         *label.Service
	Hardware      *hardware.Service
	Authenticator auth.Authenticator
	Enforcer      access.Enforcer
	Cluster       dcore.Cluster
	Insecure      *bool
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Validate implements config.Properties.
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
	validate.NotNil(v, "schematic", c.Schematic)
	validate.NotNil(v, "lineplot", c.LinePlot)
	validate.NotNil(v, "hardware", c.Hardware)
	validate.NotNil(v, "insecure", c.Insecure)
	validate.NotNil(v, "label", c.Label)
	return v.Error()
}

// Override implements config.Properties.
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
	c.Schematic = override.Nil(c.Schematic, other.Schematic)
	c.LinePlot = override.Nil(c.LinePlot, other.LinePlot)
	c.Label = override.Nil(c.Label, other.Label)
	c.Hardware = override.Nil(c.Hardware, other.Hardware)
	return c
}

type Transport struct {
	// AUTH
	AuthLogin          freighter.UnaryServer[auth.InsecureCredentials, TokenResponse]
	AuthChangeUsername freighter.UnaryServer[ChangeUsernameRequest, types.Nil]
	AuthChangePassword freighter.UnaryServer[ChangePasswordRequest, types.Nil]
	AuthRegistration   freighter.UnaryServer[RegistrationRequest, TokenResponse]
	// CHANNEL
	ChannelCreate        freighter.UnaryServer[ChannelCreateRequest, ChannelCreateResponse]
	ChannelRetrieve      freighter.UnaryServer[ChannelRetrieveRequest, ChannelRetrieveResponse]
	ChannelDelete        freighter.UnaryServer[ChannelDeleteRequest, types.Nil]
	ChannelRename        freighter.UnaryServer[ChannelRenameRequest, types.Nil]
	ChannelRetrieveGroup freighter.UnaryServer[ChannelRetrieveGroupRequest, ChannelRetrieveGroupResponse]
	// CONNECTIVITY
	ConnectivityCheck freighter.UnaryServer[types.Nil, ConnectivityCheckResponse]
	// FRAME
	FrameWriter   freighter.StreamServer[FrameWriterRequest, FrameWriterResponse]
	FrameIterator freighter.StreamServer[FrameIteratorRequest, FrameIteratorResponse]
	FrameStreamer freighter.StreamServer[FrameStreamerRequest, FrameStreamerResponse]
	FrameDelete   freighter.UnaryServer[FrameDeleteRequest, types.Nil]
	// RANGE
	RangeCreate         freighter.UnaryServer[RangeCreateRequest, RangeCreateResponse]
	RangeRetrieve       freighter.UnaryServer[RangeRetrieveRequest, RangeRetrieveResponse]
	RangeDelete         freighter.UnaryServer[RangeDeleteRequest, types.Nil]
	RangeKVGet          freighter.UnaryServer[RangeKVGetRequest, RangeKVGetResponse]
	RangeKVSet          freighter.UnaryServer[RangeKVSetRequest, types.Nil]
	RangeKVDelete       freighter.UnaryServer[RangeKVDeleteRequest, types.Nil]
	RangeAliasSet       freighter.UnaryServer[RangeAliasSetRequest, types.Nil]
	RangeAliasResolve   freighter.UnaryServer[RangeAliasResolveRequest, RangeAliasResolveResponse]
	RangeAliasList      freighter.UnaryServer[RangeAliasListRequest, RangeAliasListResponse]
	RangeRename         freighter.UnaryServer[RangeRenameRequest, types.Nil]
	RangeAliasDelete    freighter.UnaryServer[RangeAliasDeleteRequest, types.Nil]
	RangeSetActive      freighter.UnaryServer[RangeSetActiveRequest, types.Nil]
	RangeRetrieveActive freighter.UnaryServer[types.Nil, RangeRetrieveActiveResponse]
	RangeClearActive    freighter.UnaryServer[types.Nil, types.Nil]
	// ONTOLOGY
	OntologyRetrieve       freighter.UnaryServer[OntologyRetrieveRequest, OntologyRetrieveResponse]
	OntologyAddChildren    freighter.UnaryServer[OntologyAddChildrenRequest, types.Nil]
	OntologyRemoveChildren freighter.UnaryServer[OntologyRemoveChildrenRequest, types.Nil]
	OntologyMoveChildren   freighter.UnaryServer[OntologyMoveChildrenRequest, types.Nil]
	// GROUP
	OntologyGroupCreate freighter.UnaryServer[OntologyCreateGroupRequest, OntologyCreateGroupResponse]
	OntologyGroupDelete freighter.UnaryServer[OntologyDeleteGroupRequest, types.Nil]
	OntologyGroupRename freighter.UnaryServer[OntologyRenameGroupRequest, types.Nil]
	// WORKSPACE
	WorkspaceCreate    freighter.UnaryServer[WorkspaceCreateRequest, WorkspaceCreateResponse]
	WorkspaceRetrieve  freighter.UnaryServer[WorkspaceRetrieveRequest, WorkspaceRetrieveResponse]
	WorkspaceDelete    freighter.UnaryServer[WorkspaceDeleteRequest, types.Nil]
	WorkspaceRename    freighter.UnaryServer[WorkspaceRenameRequest, types.Nil]
	WorkspaceSetLayout freighter.UnaryServer[WorkspaceSetLayoutRequest, types.Nil]
	// Schematic
	SchematicCreate   freighter.UnaryServer[SchematicCreateRequest, SchematicCreateResponse]
	SchematicRetrieve freighter.UnaryServer[SchematicRetrieveRequest, SchematicRetrieveResponse]
	SchematicDelete   freighter.UnaryServer[SchematicDeleteRequest, types.Nil]
	SchematicRename   freighter.UnaryServer[SchematicRenameRequest, types.Nil]
	SchematicSetData  freighter.UnaryServer[SchematicSetDataRequest, types.Nil]
	SchematicCopy     freighter.UnaryServer[SchematicCopyRequest, SchematicCopyResponse]
	// LINE PLOT
	LinePlotCreate   freighter.UnaryServer[LinePlotCreateRequest, LinePlotCreateResponse]
	LinePlotRetrieve freighter.UnaryServer[LinePlotRetrieveRequest, LinePlotRetrieveResponse]
	LinePlotDelete   freighter.UnaryServer[LinePlotDeleteRequest, types.Nil]
	LinePlotRename   freighter.UnaryServer[LinePlotRenameRequest, types.Nil]
	LinePlotSetData  freighter.UnaryServer[LinePlotSetDataRequest, types.Nil]
	// LABEL
	LabelCreate   freighter.UnaryServer[LabelCreateRequest, LabelCreateResponse]
	LabelRetrieve freighter.UnaryServer[LabelRetrieveRequest, LabelRetrieveResponse]
	LabelDelete   freighter.UnaryServer[LabelDeleteRequest, types.Nil]
	LabelSet      freighter.UnaryServer[LabelSetRequest, types.Nil]
	LabelRemove   freighter.UnaryServer[LabelRemoveRequest, types.Nil]
	// DEVICE
	HardwareCreateRack     freighter.UnaryServer[HardwareCreateRackRequest, HardwareCreateRackResponse]
	HardwareRetrieveRack   freighter.UnaryServer[HardwareRetrieveRackRequest, HardwareRetrieveRackResponse]
	HardwareDeleteRack     freighter.UnaryServer[HardwareDeleteRackRequest, types.Nil]
	HardwareCreateTask     freighter.UnaryServer[HardwareCreateTaskRequest, HardwareCreateTaskResponse]
	HardwareRetrieveTask   freighter.UnaryServer[HardwareRetrieveTaskRequest, HardwareRetrieveTaskResponse]
	HardwareDeleteTask     freighter.UnaryServer[HardwareDeleteTaskRequest, types.Nil]
	HardwareCreateDevice   freighter.UnaryServer[HardwareCreateDeviceRequest, HardwareCreateDeviceResponse]
	HardwareRetrieveDevice freighter.UnaryServer[HardwareRetrieveDeviceRequest, HardwareRetrieveDeviceResponse]
	HardwareDeleteDevice   freighter.UnaryServer[HardwareDeleteDeviceRequest, types.Nil]
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
	Schematic    *SchematicService
	LinePlot     *LinePlotService
	Label        *LabelService
	Hardware     *HardwareService
}

// BindTo binds the API to the provided Transport implementation.
func (a *API) BindTo(t Transport) {
	var (
		tk                 = tokenMiddleware(a.provider.auth.token)
		instrumentation    = lo.Must(falamos.Middleware(falamos.Config{Instrumentation: a.config.Instrumentation}))
		insecureMiddleware = []freighter.Middleware{instrumentation}
		secureMiddleware   = make([]freighter.Middleware, len(insecureMiddleware))
	)
	copy(secureMiddleware, insecureMiddleware)
	secureMiddleware = append(secureMiddleware, tk)

	freighter.UseOnAll(
		insecureMiddleware,
		t.AuthRegistration,
		t.AuthLogin,
		t.ConnectivityCheck,
	)

	freighter.UseOnAll(
		secureMiddleware,

		// AUTH
		t.AuthChangeUsername,
		t.AuthChangePassword,

		// CHANNEL
		t.ChannelCreate,
		t.ChannelRetrieve,
		t.ChannelDelete,
		t.ChannelRename,
		t.ChannelRetrieveGroup,

		// FRAME
		t.FrameWriter,
		t.FrameIterator,
		t.FrameStreamer,
		t.FrameDelete,

		// ONTOLOGY
		t.OntologyRetrieve,
		t.OntologyAddChildren,
		t.OntologyRemoveChildren,
		t.OntologyMoveChildren,

		// GROUP
		t.OntologyGroupCreate,
		t.OntologyGroupDelete,
		t.OntologyGroupRename,

		// RANGE
		t.RangeCreate,
		t.RangeRetrieve,
		t.RangeKVGet,
		t.RangeKVSet,
		t.RangeKVDelete,
		t.RangeAliasSet,
		t.RangeAliasResolve,
		t.RangeAliasList,
		t.RangeRename,
		t.RangeAliasDelete,
		t.RangeSetActive,
		t.RangeRetrieveActive,
		t.RangeClearActive,

		// WORKSPACE
		t.WorkspaceDelete,
		t.WorkspaceCreate,
		t.WorkspaceRetrieve,
		t.WorkspaceRename,
		t.WorkspaceSetLayout,

		// Schematic
		t.SchematicCreate,
		t.SchematicRetrieve,
		t.SchematicDelete,
		t.SchematicRename,
		t.SchematicSetData,
		t.SchematicCopy,

		// LINE PLOT
		t.LinePlotCreate,
		t.LinePlotRename,
		t.LinePlotSetData,
		t.LinePlotRetrieve,
		t.LinePlotDelete,

		// LABEL
		t.LabelCreate,
		t.LabelRetrieve,
		t.LabelDelete,
		t.LabelSet,
		t.LabelRemove,

		// HARDWARE
		t.HardwareCreateRack,
		t.HardwareRetrieveRack,
		t.HardwareDeleteTask,
		t.HardwareCreateTask,
		t.HardwareRetrieveTask,
		t.HardwareDeleteTask,
		t.HardwareCreateDevice,
		t.HardwareRetrieveDevice,
		t.HardwareDeleteDevice,
	)

	// AUTH
	t.AuthLogin.BindHandler(a.Auth.Login)
	t.AuthChangeUsername.BindHandler(a.Auth.ChangeUsername)
	t.AuthChangePassword.BindHandler(a.Auth.ChangePassword)
	t.AuthRegistration.BindHandler(a.Auth.Register)

	// CHANNEL
	t.ChannelCreate.BindHandler(a.Channel.Create)
	t.ChannelRetrieve.BindHandler(a.Channel.Retrieve)
	t.ConnectivityCheck.BindHandler(a.Connectivity.Check)
	t.ChannelDelete.BindHandler(a.Channel.Delete)
	t.ChannelRename.BindHandler(a.Channel.Rename)
	t.ChannelRetrieveGroup.BindHandler(a.Channel.RetrieveGroup)

	// FRAME
	t.FrameWriter.BindHandler(a.Telem.Write)
	t.FrameIterator.BindHandler(a.Telem.Iterate)
	t.FrameStreamer.BindHandler(a.Telem.Stream)
	t.FrameDelete.BindHandler(a.Telem.FrameDelete)

	// ONTOLOGY
	t.OntologyRetrieve.BindHandler(a.Ontology.Retrieve)
	t.OntologyAddChildren.BindHandler(a.Ontology.AddChildren)
	t.OntologyRemoveChildren.BindHandler(a.Ontology.RemoveChildren)
	t.OntologyMoveChildren.BindHandler(a.Ontology.MoveChildren)

	// GROUP
	t.OntologyGroupCreate.BindHandler(a.Ontology.CreateGroup)
	t.OntologyGroupDelete.BindHandler(a.Ontology.DeleteGroup)
	t.OntologyGroupRename.BindHandler(a.Ontology.RenameGroup)

	// RANGE
	t.RangeRetrieve.BindHandler(a.Range.Retrieve)
	t.RangeCreate.BindHandler(a.Range.Create)
	t.RangeDelete.BindHandler(a.Range.Delete)
	t.RangeRename.BindHandler(a.Range.Rename)
	t.RangeKVGet.BindHandler(a.Range.KVGet)
	t.RangeKVSet.BindHandler(a.Range.KVSet)
	t.RangeKVDelete.BindHandler(a.Range.KVDelete)
	t.RangeAliasSet.BindHandler(a.Range.AliasSet)
	t.RangeAliasResolve.BindHandler(a.Range.AliasResolve)
	t.RangeAliasList.BindHandler(a.Range.AliasList)
	t.RangeAliasDelete.BindHandler(a.Range.AliasDelete)
	t.RangeSetActive.BindHandler(a.Range.SetActive)
	t.RangeRetrieveActive.BindHandler(a.Range.RetrieveActive)
	t.RangeClearActive.BindHandler(a.Range.ClearActive)

	// WORKSPACE
	t.WorkspaceCreate.BindHandler(a.Workspace.Create)
	t.WorkspaceDelete.BindHandler(a.Workspace.Delete)
	t.WorkspaceRetrieve.BindHandler(a.Workspace.Retrieve)
	t.WorkspaceRename.BindHandler(a.Workspace.Rename)
	t.WorkspaceSetLayout.BindHandler(a.Workspace.SetLayout)

	// Schematic
	t.SchematicCreate.BindHandler(a.Schematic.Create)
	t.SchematicRetrieve.BindHandler(a.Schematic.Retrieve)
	t.SchematicDelete.BindHandler(a.Schematic.Delete)
	t.SchematicRename.BindHandler(a.Schematic.Rename)
	t.SchematicSetData.BindHandler(a.Schematic.SetData)
	t.SchematicCopy.BindHandler(a.Schematic.Copy)

	// LINE PLOT
	t.LinePlotCreate.BindHandler(a.LinePlot.Create)
	t.LinePlotRename.BindHandler(a.LinePlot.Rename)
	t.LinePlotSetData.BindHandler(a.LinePlot.SetData)
	t.LinePlotRetrieve.BindHandler(a.LinePlot.Retrieve)
	t.LinePlotDelete.BindHandler(a.LinePlot.Delete)

	// LABEL
	t.LabelCreate.BindHandler(a.Label.Create)
	t.LabelRetrieve.BindHandler(a.Label.Retrieve)
	t.LabelDelete.BindHandler(a.Label.Delete)
	t.LabelSet.BindHandler(a.Label.Set)
	t.LabelRemove.BindHandler(a.Label.Remove)

	// HARDWARE
	t.HardwareCreateRack.BindHandler(a.Hardware.CreateRack)
	t.HardwareRetrieveRack.BindHandler(a.Hardware.RetrieveRack)
	t.HardwareDeleteRack.BindHandler(a.Hardware.DeleteRack)
	t.HardwareCreateTask.BindHandler(a.Hardware.CreateTask)
	t.HardwareRetrieveTask.BindHandler(a.Hardware.RetrieveTask)
	t.HardwareDeleteTask.BindHandler(a.Hardware.DeleteTask)
	t.HardwareCreateDevice.BindHandler(a.Hardware.CreateDevice)
	t.HardwareRetrieveDevice.BindHandler(a.Hardware.RetrieveDevice)
	t.HardwareDeleteDevice.BindHandler(a.Hardware.DeleteDevice)
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
	api.Schematic = NewSchematicService(api.provider)
	api.LinePlot = NewLinePlotService(api.provider)
	api.Label = NewLabelService(api.provider)
	api.Hardware = NewHardwareService(api.provider)
	return api, nil
}
