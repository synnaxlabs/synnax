// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package api implements the client interfaces for interacting with the Synnax cluster.
// The top level package is transport agnostic, and provides freighter
// compatible interfaces for all of its services. sub-packages in this directory wrap
// the core API services to provide transport-specific implementations.
package api

import (
	"go/types"

	"github.com/samber/lo"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/falamos"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/validate"
)

// Config is all required configuration parameters and services necessary to instantiate
// the API.
type Config struct {
	alamos.Instrumentation
	Service      *service.Layer
	Distribution *distribution.Layer
}

var (
	_             config.Config[Config] = Config{}
	DefaultConfig                       = Config{}
)

// Validate implements config.Config.
func (c Config) Validate() error {
	v := validate.New("api")
	validate.NotNil(v, "service", c.Service)
	validate.NotNil(v, "dist", c.Distribution)
	return v.Error()
}

// Override implements config.Config.
func (c Config) Override(other Config) Config {
	c.Instrumentation = override.Zero(c.Instrumentation, other.Instrumentation)
	c.Service = override.Nil(c.Service, other.Service)
	c.Distribution = override.Nil(c.Distribution, other.Distribution)
	return c
}

type Transport struct {
	// AUTH
	AuthLogin          freighter.UnaryServer[AuthLoginRequest, AuthLoginResponse]
	AuthChangePassword freighter.UnaryServer[AuthChangePasswordRequest, types.Nil]
	// USER
	UserRename         freighter.UnaryServer[UserRenameRequest, types.Nil]
	UserChangeUsername freighter.UnaryServer[UserChangeUsernameRequest, types.Nil]
	UserCreate         freighter.UnaryServer[UserCreateRequest, UserCreateResponse]
	UserDelete         freighter.UnaryServer[UserDeleteRequest, types.Nil]
	UserRetrieve       freighter.UnaryServer[UserRetrieveRequest, UserRetrieveResponse]
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
	RangeCreate        freighter.UnaryServer[RangeCreateRequest, RangeCreateResponse]
	RangeRetrieve      freighter.UnaryServer[RangeRetrieveRequest, RangeRetrieveResponse]
	RangeDelete        freighter.UnaryServer[RangeDeleteRequest, types.Nil]
	RangeKVGet         freighter.UnaryServer[RangeKVGetRequest, RangeKVGetResponse]
	RangeKVSet         freighter.UnaryServer[RangeKVSetRequest, types.Nil]
	RangeKVDelete      freighter.UnaryServer[RangeKVDeleteRequest, types.Nil]
	RangeAliasSet      freighter.UnaryServer[RangeAliasSetRequest, types.Nil]
	RangeAliasResolve  freighter.UnaryServer[RangeAliasResolveRequest, RangeAliasResolveResponse]
	RangeAliasList     freighter.UnaryServer[RangeAliasListRequest, RangeAliasListResponse]
	RangeAliasRetrieve freighter.UnaryServer[RangeAliasRetrieveRequest, RangeAliasRetrieveResponse]
	RangeRename        freighter.UnaryServer[RangeRenameRequest, types.Nil]
	RangeAliasDelete   freighter.UnaryServer[RangeAliasDeleteRequest, types.Nil]
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
	// SCHEMATIC
	SchematicCreate   freighter.UnaryServer[SchematicCreateRequest, SchematicCreateResponse]
	SchematicRetrieve freighter.UnaryServer[SchematicRetrieveRequest, SchematicRetrieveResponse]
	SchematicDelete   freighter.UnaryServer[SchematicDeleteRequest, types.Nil]
	SchematicRename   freighter.UnaryServer[SchematicRenameRequest, types.Nil]
	SchematicSetData  freighter.UnaryServer[SchematicSetDataRequest, types.Nil]
	SchematicCopy     freighter.UnaryServer[SchematicCopyRequest, SchematicCopyResponse]
	// SCHEMATIC SYMBOL
	SchematicSymbolCreate        freighter.UnaryServer[SymbolCreateRequest, SymbolCreateResponse]
	SchematicSymbolRetrieve      freighter.UnaryServer[SymbolRetrieveRequest, SymbolRetrieveResponse]
	SchematicSymbolDelete        freighter.UnaryServer[SymbolDeleteRequest, types.Nil]
	SchematicSymbolRename        freighter.UnaryServer[SymbolRenameRequest, types.Nil]
	SchematicSymbolRetrieveGroup freighter.UnaryServer[SymbolRetrieveGroupRequest, SymbolRetrieveGroupResponse]
	// LOG
	LogCreate   freighter.UnaryServer[LogCreateRequest, LogCreateResponse]
	LogRetrieve freighter.UnaryServer[LogRetrieveRequest, LogRetrieveResponse]
	LogDelete   freighter.UnaryServer[LogDeleteRequest, types.Nil]
	LogRename   freighter.UnaryServer[LogRenameRequest, types.Nil]
	LogSetData  freighter.UnaryServer[LogSetDataRequest, types.Nil]
	// TABLE
	TableCreate   freighter.UnaryServer[TableCreateRequest, TableCreateResponse]
	TableRetrieve freighter.UnaryServer[TableRetrieveRequest, TableRetrieveResponse]
	TableDelete   freighter.UnaryServer[TableDeleteRequest, types.Nil]
	TableRename   freighter.UnaryServer[TableRenameRequest, types.Nil]
	TableSetData  freighter.UnaryServer[TableSetDataRequest, types.Nil]
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
	LabelAdd      freighter.UnaryServer[LabelAddRequest, types.Nil]
	LabelRemove   freighter.UnaryServer[LabelRemoveRequest, types.Nil]
	// DEVICE
	HardwareCreateRack     freighter.UnaryServer[HardwareCreateRackRequest, HardwareCreateRackResponse]
	HardwareRetrieveRack   freighter.UnaryServer[HardwareRetrieveRackRequest, HardwareRetrieveRackResponse]
	HardwareDeleteRack     freighter.UnaryServer[HardwareDeleteRackRequest, types.Nil]
	HardwareCreateTask     freighter.UnaryServer[HardwareCreateTaskRequest, HardwareCreateTaskResponse]
	HardwareRetrieveTask   freighter.UnaryServer[HardwareRetrieveTaskRequest, HardwareRetrieveTaskResponse]
	HardwareCopyTask       freighter.UnaryServer[HardwareCopyTaskRequest, HardwareCopyTaskResponse]
	HardwareDeleteTask     freighter.UnaryServer[HardwareDeleteTaskRequest, types.Nil]
	HardwareCreateDevice   freighter.UnaryServer[HardwareCreateDeviceRequest, HardwareCreateDeviceResponse]
	HardwareRetrieveDevice freighter.UnaryServer[HardwareRetrieveDeviceRequest, HardwareRetrieveDeviceResponse]
	HardwareDeleteDevice   freighter.UnaryServer[HardwareDeleteDeviceRequest, types.Nil]
	// ACCESS
	AccessCreatePolicy   freighter.UnaryServer[AccessCreatePolicyRequest, AccessCreatePolicyResponse]
	AccessDeletePolicy   freighter.UnaryServer[AccessDeletePolicyRequest, types.Nil]
	AccessRetrievePolicy freighter.UnaryServer[AccessRetrievePolicyRequest, AccessRetrievePolicyResponse]
}

// Layer wraps all implemented API services into a single container. Protocol-specific Layer
// implementations should use this struct during instantiation.
type Layer struct {
	provider     Provider
	config       Config
	Auth         *AuthService
	User         *UserService
	Framer       *FrameService
	Channel      *ChannelService
	Connectivity *ConnectivityService
	Ontology     *OntologyService
	Range        *RangeService
	Workspace    *WorkspaceService
	Schematic    *SchematicService
	LinePlot     *LinePlotService
	Log          *LogService
	Table        *TableService
	Label        *LabelService
	Hardware     *HardwareService
	Access       *AccessService
}

// BindTo binds the API layer to the provided Transport implementation.
func (a *Layer) BindTo(t Transport) {
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
		t.AuthLogin,
		t.ConnectivityCheck,
	)

	freighter.UseOnAll(
		secureMiddleware,

		// AUTH
		t.AuthChangePassword,

		// USER
		t.UserRename,
		t.UserChangeUsername,
		t.UserCreate,
		t.UserDelete,
		t.UserRetrieve,

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
		t.RangeDelete,
		t.RangeKVGet,
		t.RangeKVSet,
		t.RangeKVDelete,
		t.RangeAliasSet,
		t.RangeAliasResolve,
		t.RangeAliasList,
		t.RangeAliasRetrieve,
		t.RangeRename,
		t.RangeAliasDelete,

		// WORKSPACE
		t.WorkspaceDelete,
		t.WorkspaceCreate,
		t.WorkspaceRetrieve,
		t.WorkspaceRename,
		t.WorkspaceSetLayout,

		// SCHEMATIC
		t.SchematicCreate,
		t.SchematicRetrieve,
		t.SchematicDelete,
		t.SchematicRename,
		t.SchematicSetData,
		t.SchematicCopy,

		// SCHEMATIC SYMBOL
		t.SchematicSymbolCreate,
		t.SchematicSymbolRetrieve,
		t.SchematicSymbolDelete,
		t.SchematicSymbolRename,
		t.SchematicSymbolRetrieveGroup,

		// LINE PLOT
		t.LinePlotCreate,
		t.LinePlotRename,
		t.LinePlotSetData,
		t.LinePlotRetrieve,
		t.LinePlotDelete,

		// LOG
		t.LogCreate,
		t.LogRetrieve,
		t.LogDelete,
		t.LogRename,
		t.LogSetData,

		// TABLE
		t.TableCreate,
		t.TableRetrieve,
		t.TableDelete,
		t.TableRename,
		t.TableSetData,

		// LABEL
		t.LabelCreate,
		t.LabelRetrieve,
		t.LabelDelete,
		t.LabelAdd,
		t.LabelRemove,

		// HARDWARE
		t.HardwareCreateRack,
		t.HardwareDeleteRack,
		t.HardwareRetrieveRack,
		t.HardwareDeleteTask,
		t.HardwareCreateTask,
		t.HardwareRetrieveTask,
		t.HardwareDeleteTask,
		t.HardwareCopyTask,
		t.HardwareCreateDevice,
		t.HardwareRetrieveDevice,
		t.HardwareDeleteDevice,

		// ACCESS
		t.AccessCreatePolicy,
		t.AccessDeletePolicy,
		t.AccessRetrievePolicy,
	)

	// AUTH
	t.AuthLogin.BindHandler(a.Auth.Login)
	t.AuthChangePassword.BindHandler(a.Auth.ChangePassword)

	// USER
	t.UserRename.BindHandler(a.User.Rename)
	t.UserChangeUsername.BindHandler(a.User.ChangeUsername)
	t.UserCreate.BindHandler(a.User.Create)
	t.UserDelete.BindHandler(a.User.Delete)
	t.UserRetrieve.BindHandler(a.User.Retrieve)

	// CHANNEL
	t.ChannelCreate.BindHandler(a.Channel.Create)
	t.ChannelRetrieve.BindHandler(a.Channel.Retrieve)
	t.ConnectivityCheck.BindHandler(a.Connectivity.Check)
	t.ChannelDelete.BindHandler(a.Channel.Delete)
	t.ChannelRename.BindHandler(a.Channel.Rename)
	t.ChannelRetrieveGroup.BindHandler(a.Channel.RetrieveGroup)

	// FRAME
	t.FrameWriter.BindHandler(a.Framer.Write)
	t.FrameIterator.BindHandler(a.Framer.Iterate)
	t.FrameStreamer.BindHandler(a.Framer.Stream)
	t.FrameDelete.BindHandler(a.Framer.FrameDelete)

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
	t.RangeAliasRetrieve.BindHandler(a.Range.AliasRetrieve)
	t.RangeAliasDelete.BindHandler(a.Range.AliasDelete)

	// WORKSPACE
	t.WorkspaceCreate.BindHandler(a.Workspace.Create)
	t.WorkspaceDelete.BindHandler(a.Workspace.Delete)
	t.WorkspaceRetrieve.BindHandler(a.Workspace.Retrieve)
	t.WorkspaceRename.BindHandler(a.Workspace.Rename)
	t.WorkspaceSetLayout.BindHandler(a.Workspace.SetLayout)

	// SCHEMATIC
	t.SchematicCreate.BindHandler(a.Schematic.Create)
	t.SchematicRetrieve.BindHandler(a.Schematic.Retrieve)
	t.SchematicDelete.BindHandler(a.Schematic.Delete)
	t.SchematicRename.BindHandler(a.Schematic.Rename)
	t.SchematicSetData.BindHandler(a.Schematic.SetData)
	t.SchematicCopy.BindHandler(a.Schematic.Copy)

	// SCHEMATIC SYMBOL
	t.SchematicSymbolCreate.BindHandler(a.Schematic.CreateSymbol)
	t.SchematicSymbolRetrieve.BindHandler(a.Schematic.RetrieveSymbol)
	t.SchematicSymbolDelete.BindHandler(a.Schematic.DeleteSymbol)
	t.SchematicSymbolRename.BindHandler(a.Schematic.RenameSymbol)
	t.SchematicSymbolRetrieveGroup.BindHandler(a.Schematic.RetrieveSymbolGroup)

	// LINE PLOT
	t.LinePlotCreate.BindHandler(a.LinePlot.Create)
	t.LinePlotRename.BindHandler(a.LinePlot.Rename)
	t.LinePlotSetData.BindHandler(a.LinePlot.SetData)
	t.LinePlotRetrieve.BindHandler(a.LinePlot.Retrieve)
	t.LinePlotDelete.BindHandler(a.LinePlot.Delete)

	// LOG
	t.LogCreate.BindHandler(a.Log.Create)
	t.LogRetrieve.BindHandler(a.Log.Retrieve)
	t.LogDelete.BindHandler(a.Log.Delete)
	t.LogRename.BindHandler(a.Log.Rename)
	t.LogSetData.BindHandler(a.Log.SetData)

	// TABLE
	t.TableCreate.BindHandler(a.Table.Create)
	t.TableRetrieve.BindHandler(a.Table.Retrieve)
	t.TableDelete.BindHandler(a.Table.Delete)
	t.TableRename.BindHandler(a.Table.Rename)
	t.TableSetData.BindHandler(a.Table.SetData)

	// LABEL
	t.LabelCreate.BindHandler(a.Label.Create)
	t.LabelRetrieve.BindHandler(a.Label.Retrieve)
	t.LabelDelete.BindHandler(a.Label.Delete)
	t.LabelAdd.BindHandler(a.Label.Add)
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
	t.HardwareCopyTask.BindHandler(a.Hardware.CopyTask)

	// ACCESS
	t.AccessCreatePolicy.BindHandler(a.Access.CreatePolicy)
	t.AccessDeletePolicy.BindHandler(a.Access.DeletePolicy)
	t.AccessRetrievePolicy.BindHandler(a.Access.RetrievePolicy)
}

// New instantiates the server API layer using the provided Config. This should only be called
// once.
func New(configs ...Config) (*Layer, error) {
	cfg, err := config.New(DefaultConfig, configs...)
	if err != nil {
		return nil, err
	}
	api := &Layer{config: cfg, provider: NewProvider(cfg)}
	api.Auth = NewAuthService(api.provider)
	api.User = NewUserService(api.provider)
	api.Access = NewAccessService(api.provider)
	api.Framer = NewFrameService(api.provider)
	api.Channel = NewChannelService(api.provider)
	api.Connectivity = NewConnectivityService(api.provider)
	api.Ontology = NewOntologyService(api.provider)
	api.Range = NewRangeService(api.provider)
	api.Workspace = NewWorkspaceService(api.provider)
	api.Schematic = NewSchematicService(api.provider)
	api.LinePlot = NewLinePlotService(api.provider)
	api.Label = NewLabelService(api.provider)
	api.Hardware = NewHardwareService(api.provider)
	api.Log = NewLogService(api.provider)
	api.Table = NewTableService(api.provider)
	return api, nil
}
