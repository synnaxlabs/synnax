// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package api implements the client interfaces for interacting with the Synnax cluster.
// The top level package is transport agnostic, and provides freighter-compatible
// interfaces for all of its services. Packages in this directory wrap the core API
// services to provide transport-specific implementations.
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

var _ config.Config[Config] = Config{}

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
	AuthLogin          freighter.UnaryServer[AuthLoginRequest, AuthLoginResponse]
	AuthChangePassword freighter.UnaryServer[AuthChangePasswordRequest, types.Nil]

	UserRename         freighter.UnaryServer[UserRenameRequest, types.Nil]
	UserChangeUsername freighter.UnaryServer[UserChangeUsernameRequest, types.Nil]
	UserCreate         freighter.UnaryServer[UserCreateRequest, UserCreateResponse]
	UserDelete         freighter.UnaryServer[UserDeleteRequest, types.Nil]
	UserRetrieve       freighter.UnaryServer[UserRetrieveRequest, UserRetrieveResponse]

	ChannelCreate        freighter.UnaryServer[ChannelCreateRequest, ChannelCreateResponse]
	ChannelRetrieve      freighter.UnaryServer[ChannelRetrieveRequest, ChannelRetrieveResponse]
	ChannelDelete        freighter.UnaryServer[ChannelDeleteRequest, types.Nil]
	ChannelRename        freighter.UnaryServer[ChannelRenameRequest, types.Nil]
	ChannelRetrieveGroup freighter.UnaryServer[types.Nil, ChannelRetrieveGroupResponse]

	ConnectivityCheck freighter.UnaryServer[types.Nil, ConnectivityCheckResponse]

	ExportCSV freighter.UnaryServer[ExportCSVRequest, ExportCSVResponse]

	FrameWriter   freighter.StreamServer[FrameWriterRequest, FrameWriterResponse]
	FrameIterator freighter.StreamServer[FrameIteratorRequest, FrameIteratorResponse]
	FrameStreamer freighter.StreamServer[FrameStreamerRequest, FrameStreamerResponse]
	FrameDelete   freighter.UnaryServer[FrameDeleteRequest, types.Nil]

	RangeCreate       freighter.UnaryServer[RangeCreateRequest, RangeCreateResponse]
	RangeRetrieve     freighter.UnaryServer[RangeRetrieveRequest, RangeRetrieveResponse]
	RangeDelete       freighter.UnaryServer[RangeDeleteRequest, types.Nil]
	RangeKVGet        freighter.UnaryServer[RangeKVGetRequest, RangeKVGetResponse]
	RangeKVSet        freighter.UnaryServer[RangeKVSetRequest, types.Nil]
	RangeKVDelete     freighter.UnaryServer[RangeKVDeleteRequest, types.Nil]
	RangeAliasSet     freighter.UnaryServer[RangeAliasSetRequest, types.Nil]
	RangeAliasResolve freighter.UnaryServer[RangeAliasResolveRequest, RangeAliasResolveResponse]
	RangeAliasList    freighter.UnaryServer[RangeAliasListRequest, RangeAliasListResponse]
	RangeRename       freighter.UnaryServer[RangeRenameRequest, types.Nil]
	RangeAliasDelete  freighter.UnaryServer[RangeAliasDeleteRequest, types.Nil]

	OntologyRetrieve       freighter.UnaryServer[OntologyRetrieveRequest, OntologyRetrieveResponse]
	OntologyAddChildren    freighter.UnaryServer[OntologyAddChildrenRequest, types.Nil]
	OntologyRemoveChildren freighter.UnaryServer[OntologyRemoveChildrenRequest, types.Nil]
	OntologyMoveChildren   freighter.UnaryServer[OntologyMoveChildrenRequest, types.Nil]
	OntologyGroupCreate    freighter.UnaryServer[OntologyCreateGroupRequest, OntologyCreateGroupResponse]
	OntologyGroupDelete    freighter.UnaryServer[OntologyDeleteGroupRequest, types.Nil]
	OntologyGroupRename    freighter.UnaryServer[OntologyRenameGroupRequest, types.Nil]

	WorkspaceCreate    freighter.UnaryServer[WorkspaceCreateRequest, WorkspaceCreateResponse]
	WorkspaceRetrieve  freighter.UnaryServer[WorkspaceRetrieveRequest, WorkspaceRetrieveResponse]
	WorkspaceDelete    freighter.UnaryServer[WorkspaceDeleteRequest, types.Nil]
	WorkspaceRename    freighter.UnaryServer[WorkspaceRenameRequest, types.Nil]
	WorkspaceSetLayout freighter.UnaryServer[WorkspaceSetLayoutRequest, types.Nil]

	SchematicCreate   freighter.UnaryServer[SchematicCreateRequest, SchematicCreateResponse]
	SchematicRetrieve freighter.UnaryServer[SchematicRetrieveRequest, SchematicRetrieveResponse]
	SchematicDelete   freighter.UnaryServer[SchematicDeleteRequest, types.Nil]
	SchematicRename   freighter.UnaryServer[SchematicRenameRequest, types.Nil]
	SchematicSetData  freighter.UnaryServer[SchematicSetDataRequest, types.Nil]
	SchematicCopy     freighter.UnaryServer[SchematicCopyRequest, SchematicCopyResponse]

	LogCreate   freighter.UnaryServer[LogCreateRequest, LogCreateResponse]
	LogRetrieve freighter.UnaryServer[LogRetrieveRequest, LogRetrieveResponse]
	LogDelete   freighter.UnaryServer[LogDeleteRequest, types.Nil]
	LogRename   freighter.UnaryServer[LogRenameRequest, types.Nil]
	LogSetData  freighter.UnaryServer[LogSetDataRequest, types.Nil]

	TableCreate   freighter.UnaryServer[TableCreateRequest, TableCreateResponse]
	TableRetrieve freighter.UnaryServer[TableRetrieveRequest, TableRetrieveResponse]
	TableDelete   freighter.UnaryServer[TableDeleteRequest, types.Nil]
	TableRename   freighter.UnaryServer[TableRenameRequest, types.Nil]
	TableSetData  freighter.UnaryServer[TableSetDataRequest, types.Nil]

	LinePlotCreate   freighter.UnaryServer[LinePlotCreateRequest, LinePlotCreateResponse]
	LinePlotRetrieve freighter.UnaryServer[LinePlotRetrieveRequest, LinePlotRetrieveResponse]
	LinePlotDelete   freighter.UnaryServer[LinePlotDeleteRequest, types.Nil]
	LinePlotRename   freighter.UnaryServer[LinePlotRenameRequest, types.Nil]
	LinePlotSetData  freighter.UnaryServer[LinePlotSetDataRequest, types.Nil]

	LabelCreate   freighter.UnaryServer[LabelCreateRequest, LabelCreateResponse]
	LabelRetrieve freighter.UnaryServer[LabelRetrieveRequest, LabelRetrieveResponse]
	LabelDelete   freighter.UnaryServer[LabelDeleteRequest, types.Nil]
	LabelAdd      freighter.UnaryServer[LabelAddRequest, types.Nil]
	LabelRemove   freighter.UnaryServer[LabelRemoveRequest, types.Nil]

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

	AccessCreatePolicy   freighter.UnaryServer[AccessCreatePolicyRequest, AccessCreatePolicyResponse]
	AccessDeletePolicy   freighter.UnaryServer[AccessDeletePolicyRequest, types.Nil]
	AccessRetrievePolicy freighter.UnaryServer[AccessRetrievePolicyRequest, AccessRetrievePolicyResponse]
}

// Layer wraps all implemented API services into a single container. Protocol-specific
// Layer implementations should use this struct during instantiation.
type Layer struct {
	provider     Provider
	config       Config
	Auth         *AuthService
	User         *UserService
	Export       *ExportService
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
func (l *Layer) BindTo(t Transport) {
	var (
		tk                 = tokenMiddleware(l.provider.auth.token)
		instrumentation    = lo.Must(falamos.Middleware(falamos.Config{Instrumentation: l.config.Instrumentation}))
		insecureMiddleware = []freighter.Middleware{instrumentation}
		secureMiddleware   = []freighter.Middleware{instrumentation, tk}
	)

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

		// EXPORT
		t.ExportCSV,

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
	t.AuthLogin.BindHandler(l.Auth.Login)
	t.AuthChangePassword.BindHandler(l.Auth.ChangePassword)

	// USER
	t.UserRename.BindHandler(l.User.Rename)
	t.UserChangeUsername.BindHandler(l.User.ChangeUsername)
	t.UserCreate.BindHandler(l.User.Create)
	t.UserDelete.BindHandler(l.User.Delete)
	t.UserRetrieve.BindHandler(l.User.Retrieve)

	// CHANNEL
	t.ChannelCreate.BindHandler(l.Channel.Create)
	t.ChannelRetrieve.BindHandler(l.Channel.Retrieve)
	t.ConnectivityCheck.BindHandler(l.Connectivity.Check)
	t.ChannelDelete.BindHandler(l.Channel.Delete)
	t.ChannelRename.BindHandler(l.Channel.Rename)
	t.ChannelRetrieveGroup.BindHandler(l.Channel.RetrieveGroup)

	// EXPORT
	t.ExportCSV.BindHandler(l.Export.CSV)

	// FRAME
	t.FrameWriter.BindHandler(l.Framer.Write)
	t.FrameIterator.BindHandler(l.Framer.Iterate)
	t.FrameStreamer.BindHandler(l.Framer.Stream)
	t.FrameDelete.BindHandler(l.Framer.FrameDelete)

	// ONTOLOGY
	t.OntologyRetrieve.BindHandler(l.Ontology.Retrieve)
	t.OntologyAddChildren.BindHandler(l.Ontology.AddChildren)
	t.OntologyRemoveChildren.BindHandler(l.Ontology.RemoveChildren)
	t.OntologyMoveChildren.BindHandler(l.Ontology.MoveChildren)

	// GROUP
	t.OntologyGroupCreate.BindHandler(l.Ontology.CreateGroup)
	t.OntologyGroupDelete.BindHandler(l.Ontology.DeleteGroup)
	t.OntologyGroupRename.BindHandler(l.Ontology.RenameGroup)

	// RANGE
	t.RangeRetrieve.BindHandler(l.Range.Retrieve)
	t.RangeCreate.BindHandler(l.Range.Create)
	t.RangeDelete.BindHandler(l.Range.Delete)
	t.RangeRename.BindHandler(l.Range.Rename)
	t.RangeKVGet.BindHandler(l.Range.KVGet)
	t.RangeKVSet.BindHandler(l.Range.KVSet)
	t.RangeKVDelete.BindHandler(l.Range.KVDelete)
	t.RangeAliasSet.BindHandler(l.Range.AliasSet)
	t.RangeAliasResolve.BindHandler(l.Range.AliasResolve)
	t.RangeAliasList.BindHandler(l.Range.AliasList)
	t.RangeAliasDelete.BindHandler(l.Range.AliasDelete)

	// WORKSPACE
	t.WorkspaceCreate.BindHandler(l.Workspace.Create)
	t.WorkspaceDelete.BindHandler(l.Workspace.Delete)
	t.WorkspaceRetrieve.BindHandler(l.Workspace.Retrieve)
	t.WorkspaceRename.BindHandler(l.Workspace.Rename)
	t.WorkspaceSetLayout.BindHandler(l.Workspace.SetLayout)

	// SCHEMATIC
	t.SchematicCreate.BindHandler(l.Schematic.Create)
	t.SchematicRetrieve.BindHandler(l.Schematic.Retrieve)
	t.SchematicDelete.BindHandler(l.Schematic.Delete)
	t.SchematicRename.BindHandler(l.Schematic.Rename)
	t.SchematicSetData.BindHandler(l.Schematic.SetData)
	t.SchematicCopy.BindHandler(l.Schematic.Copy)

	// LINE PLOT
	t.LinePlotCreate.BindHandler(l.LinePlot.Create)
	t.LinePlotRename.BindHandler(l.LinePlot.Rename)
	t.LinePlotSetData.BindHandler(l.LinePlot.SetData)
	t.LinePlotRetrieve.BindHandler(l.LinePlot.Retrieve)
	t.LinePlotDelete.BindHandler(l.LinePlot.Delete)

	// LOG
	t.LogCreate.BindHandler(l.Log.Create)
	t.LogRetrieve.BindHandler(l.Log.Retrieve)
	t.LogDelete.BindHandler(l.Log.Delete)
	t.LogRename.BindHandler(l.Log.Rename)
	t.LogSetData.BindHandler(l.Log.SetData)

	// TABLE
	t.TableCreate.BindHandler(l.Table.Create)
	t.TableRetrieve.BindHandler(l.Table.Retrieve)
	t.TableDelete.BindHandler(l.Table.Delete)
	t.TableRename.BindHandler(l.Table.Rename)
	t.TableSetData.BindHandler(l.Table.SetData)

	// LABEL
	t.LabelCreate.BindHandler(l.Label.Create)
	t.LabelRetrieve.BindHandler(l.Label.Retrieve)
	t.LabelDelete.BindHandler(l.Label.Delete)
	t.LabelAdd.BindHandler(l.Label.Add)
	t.LabelRemove.BindHandler(l.Label.Remove)

	// HARDWARE
	t.HardwareCreateRack.BindHandler(l.Hardware.CreateRack)
	t.HardwareRetrieveRack.BindHandler(l.Hardware.RetrieveRack)
	t.HardwareDeleteRack.BindHandler(l.Hardware.DeleteRack)
	t.HardwareCreateTask.BindHandler(l.Hardware.CreateTask)
	t.HardwareRetrieveTask.BindHandler(l.Hardware.RetrieveTask)
	t.HardwareDeleteTask.BindHandler(l.Hardware.DeleteTask)
	t.HardwareCreateDevice.BindHandler(l.Hardware.CreateDevice)
	t.HardwareRetrieveDevice.BindHandler(l.Hardware.RetrieveDevice)
	t.HardwareDeleteDevice.BindHandler(l.Hardware.DeleteDevice)
	t.HardwareCopyTask.BindHandler(l.Hardware.CopyTask)

	// ACCESS
	t.AccessCreatePolicy.BindHandler(l.Access.CreatePolicy)
	t.AccessDeletePolicy.BindHandler(l.Access.DeletePolicy)
	t.AccessRetrievePolicy.BindHandler(l.Access.RetrievePolicy)
}

// New instantiates the server API layer using the provided Config. This should only be
// called once.
func New(cfgs ...Config) (*Layer, error) {
	cfg, err := config.New(Config{}, cfgs...)
	if err != nil {
		return nil, err
	}
	layer := &Layer{
		config:   cfg,
		provider: NewProvider(cfg),
	}
	layer.Auth = NewAuthService(layer.provider)
	layer.User = NewUserService(layer.provider)
	layer.Access = NewAccessService(layer.provider)
	layer.Export = NewExportService(layer.provider)
	layer.Framer = NewFrameService(layer.provider)
	layer.Channel = NewChannelService(layer.provider)
	layer.Connectivity = NewConnectivityService(layer.provider)
	layer.Ontology = NewOntologyService(layer.provider)
	layer.Range = NewRangeService(layer.provider)
	layer.Workspace = NewWorkspaceService(layer.provider)
	layer.Schematic = NewSchematicService(layer.provider)
	layer.LinePlot = NewLinePlotService(layer.provider)
	layer.Label = NewLabelService(layer.provider)
	layer.Hardware = NewHardwareService(layer.provider)
	layer.Log = NewLogService(layer.provider)
	layer.Table = NewTableService(layer.provider)
	return layer, nil
}
