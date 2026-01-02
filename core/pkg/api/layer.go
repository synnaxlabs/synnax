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
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/falamos"
	"github.com/synnaxlabs/synnax/pkg/api/access"
	"github.com/synnaxlabs/synnax/pkg/api/arc"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/channel"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/api/connectivity"
	"github.com/synnaxlabs/synnax/pkg/api/device"
	"github.com/synnaxlabs/synnax/pkg/api/framer"
	"github.com/synnaxlabs/synnax/pkg/api/group"
	"github.com/synnaxlabs/synnax/pkg/api/label"
	"github.com/synnaxlabs/synnax/pkg/api/lineplot"
	"github.com/synnaxlabs/synnax/pkg/api/log"
	"github.com/synnaxlabs/synnax/pkg/api/ontology"
	"github.com/synnaxlabs/synnax/pkg/api/rack"
	"github.com/synnaxlabs/synnax/pkg/api/ranger"
	"github.com/synnaxlabs/synnax/pkg/api/schematic"
	"github.com/synnaxlabs/synnax/pkg/api/status"
	"github.com/synnaxlabs/synnax/pkg/api/table"
	"github.com/synnaxlabs/synnax/pkg/api/task"
	"github.com/synnaxlabs/synnax/pkg/api/user"
	"github.com/synnaxlabs/synnax/pkg/api/view"
	"github.com/synnaxlabs/synnax/pkg/api/workspace"
	xconfig "github.com/synnaxlabs/x/config"
)

type Config = config.Config

var DefaultConfig = config.Default

type Transport struct {
	// AUTH
	AuthLogin          freighter.UnaryServer[auth.LoginRequest, auth.LoginResponse]
	AuthChangePassword freighter.UnaryServer[auth.ChangePasswordRequest, types.Nil]
	// USER
	UserRename         freighter.UnaryServer[user.RenameRequest, types.Nil]
	UserChangeUsername freighter.UnaryServer[user.ChangeUsernameRequest, types.Nil]
	UserCreate         freighter.UnaryServer[user.CreateRequest, user.CreateResponse]
	UserDelete         freighter.UnaryServer[user.DeleteRequest, types.Nil]
	UserRetrieve       freighter.UnaryServer[user.RetrieveRequest, user.RetrieveResponse]
	// CHANNEL
	ChannelCreate        freighter.UnaryServer[channel.CreateRequest, channel.CreateResponse]
	ChannelRetrieve      freighter.UnaryServer[channel.RetrieveRequest, channel.RetrieveResponse]
	ChannelDelete        freighter.UnaryServer[channel.DeleteRequest, types.Nil]
	ChannelRename        freighter.UnaryServer[channel.RenameRequest, types.Nil]
	ChannelRetrieveGroup freighter.UnaryServer[channel.RetrieveGroupRequest, channel.RetrieveGroupResponse]
	// CONNECTIVITY
	ConnectivityCheck freighter.UnaryServer[types.Nil, connectivity.CheckResponse]
	// FRAME
	FrameWriter   freighter.StreamServer[framer.WriterRequest, framer.WriterResponse]
	FrameIterator freighter.StreamServer[framer.IteratorRequest, framer.IteratorResponse]
	FrameStreamer freighter.StreamServer[framer.StreamerRequest, framer.StreamerResponse]
	FrameDelete   freighter.UnaryServer[framer.DeleteRequest, types.Nil]
	// RANGE
	RangeCreate        freighter.UnaryServer[ranger.CreateRequest, ranger.CreateResponse]
	RangeRetrieve      freighter.UnaryServer[ranger.RetrieveRequest, ranger.RetrieveResponse]
	RangeDelete        freighter.UnaryServer[ranger.DeleteRequest, types.Nil]
	RangeKVGet         freighter.UnaryServer[ranger.KVGetRequest, ranger.KVGetResponse]
	RangeKVSet         freighter.UnaryServer[ranger.KVSetRequest, types.Nil]
	RangeKVDelete      freighter.UnaryServer[ranger.KVDeleteRequest, types.Nil]
	RangeAliasSet      freighter.UnaryServer[ranger.AliasSetRequest, types.Nil]
	RangeAliasResolve  freighter.UnaryServer[ranger.AliasResolveRequest, ranger.AliasResolveResponse]
	RangeAliasList     freighter.UnaryServer[ranger.AliasListRequest, ranger.AliasListResponse]
	RangeAliasRetrieve freighter.UnaryServer[ranger.AliasRetrieveRequest, ranger.AliasRetrieveResponse]
	RangeRename        freighter.UnaryServer[ranger.RenameRequest, types.Nil]
	RangeAliasDelete   freighter.UnaryServer[ranger.AliasDeleteRequest, types.Nil]
	// ONTOLOGY
	OntologyRetrieve       freighter.UnaryServer[ontology.RetrieveRequest, ontology.RetrieveResponse]
	OntologyAddChildren    freighter.UnaryServer[ontology.AddChildrenRequest, types.Nil]
	OntologyRemoveChildren freighter.UnaryServer[ontology.RemoveChildrenRequest, types.Nil]
	OntologyMoveChildren   freighter.UnaryServer[ontology.MoveChildrenRequest, types.Nil]
	// GROUP
	GroupCreate freighter.UnaryServer[group.CreateRequest, group.CreateResponse]
	GroupDelete freighter.UnaryServer[group.DeleteRequest, types.Nil]
	GroupRename freighter.UnaryServer[group.RenameRequest, types.Nil]
	// WORKSPACE
	WorkspaceCreate    freighter.UnaryServer[workspace.CreateRequest, workspace.CreateResponse]
	WorkspaceRetrieve  freighter.UnaryServer[workspace.RetrieveRequest, workspace.RetrieveResponse]
	WorkspaceDelete    freighter.UnaryServer[workspace.DeleteRequest, types.Nil]
	WorkspaceRename    freighter.UnaryServer[workspace.RenameRequest, types.Nil]
	WorkspaceSetLayout freighter.UnaryServer[workspace.SetLayoutRequest, types.Nil]
	// SCHEMATIC
	SchematicCreate   freighter.UnaryServer[schematic.CreateRequest, schematic.CreateResponse]
	SchematicRetrieve freighter.UnaryServer[schematic.RetrieveRequest, schematic.RetrieveResponse]
	SchematicDelete   freighter.UnaryServer[schematic.DeleteRequest, types.Nil]
	SchematicRename   freighter.UnaryServer[schematic.RenameRequest, types.Nil]
	SchematicSetData  freighter.UnaryServer[schematic.SetDataRequest, types.Nil]
	SchematicCopy     freighter.UnaryServer[schematic.CopyRequest, schematic.CopyResponse]
	// SCHEMATIC SYMBOL
	SchematicCreateSymbol        freighter.UnaryServer[schematic.CreateSymbolRequest, schematic.CreateSymbolResponse]
	SchematicRetrieveSymbol      freighter.UnaryServer[schematic.RetrieveSymbolRequest, schematic.RetrieveSymbolResponse]
	SchematicDeleteSymbol        freighter.UnaryServer[schematic.DeleteSymbolRequest, types.Nil]
	SchematicRenameSymbol        freighter.UnaryServer[schematic.RenameSymbolRequest, types.Nil]
	SchematicRetrieveSymbolGroup freighter.UnaryServer[schematic.RetrieveSymbolGroupRequest, schematic.RetrieveSymbolGroupResponse]
	// LOG
	LogCreate   freighter.UnaryServer[log.CreateRequest, log.CreateResponse]
	LogRetrieve freighter.UnaryServer[log.RetrieveRequest, log.RetrieveResponse]
	LogDelete   freighter.UnaryServer[log.DeleteRequest, types.Nil]
	LogRename   freighter.UnaryServer[log.RenameRequest, types.Nil]
	LogSetData  freighter.UnaryServer[log.SetDataRequest, types.Nil]
	// TABLE
	TableCreate   freighter.UnaryServer[table.CreateRequest, table.CreateResponse]
	TableRetrieve freighter.UnaryServer[table.RetrieveRequest, table.RetrieveResponse]
	TableDelete   freighter.UnaryServer[table.DeleteRequest, types.Nil]
	TableRename   freighter.UnaryServer[table.RenameRequest, types.Nil]
	TableSetData  freighter.UnaryServer[table.SetDataRequest, types.Nil]
	// LINE PLOT
	LinePlotCreate   freighter.UnaryServer[lineplot.CreateRequest, lineplot.CreateResponse]
	LinePlotRetrieve freighter.UnaryServer[lineplot.RetrieveRequest, lineplot.RetrieveResponse]
	LinePlotDelete   freighter.UnaryServer[lineplot.DeleteRequest, types.Nil]
	LinePlotRename   freighter.UnaryServer[lineplot.RenameRequest, types.Nil]
	LinePlotSetData  freighter.UnaryServer[lineplot.SetDataRequest, types.Nil]
	// LABEL
	LabelCreate   freighter.UnaryServer[label.CreateRequest, label.CreateResponse]
	LabelRetrieve freighter.UnaryServer[label.RetrieveRequest, label.RetrieveResponse]
	LabelDelete   freighter.UnaryServer[label.DeleteRequest, types.Nil]
	LabelAdd      freighter.UnaryServer[label.AddRequest, types.Nil]
	LabelRemove   freighter.UnaryServer[label.RemoveRequest, types.Nil]
	// RACK
	RackCreate   freighter.UnaryServer[rack.CreateRequest, rack.CreateResponse]
	RackRetrieve freighter.UnaryServer[rack.RetrieveRequest, rack.RetrieveResponse]
	RackDelete   freighter.UnaryServer[rack.DeleteRequest, types.Nil]
	// TASK
	TaskCreate   freighter.UnaryServer[task.CreateRequest, task.CreateResponse]
	TaskRetrieve freighter.UnaryServer[task.RetrieveRequest, task.RetrieveResponse]
	TaskDelete   freighter.UnaryServer[task.DeleteRequest, types.Nil]
	TaskCopy     freighter.UnaryServer[task.CopyRequest, task.CopyResponse]
	// DEVICE
	DeviceCreate   freighter.UnaryServer[device.CreateRequest, device.CreateResponse]
	DeviceRetrieve freighter.UnaryServer[device.RetrieveRequest, device.RetrieveResponse]
	DeviceDelete   freighter.UnaryServer[device.DeleteRequest, types.Nil]
	// ACCESS
	AccessCreatePolicy   freighter.UnaryServer[access.CreatePolicyRequest, access.CreatePolicyResponse]
	AccessDeletePolicy   freighter.UnaryServer[access.DeletePolicyRequest, types.Nil]
	AccessRetrievePolicy freighter.UnaryServer[access.RetrievePolicyRequest, access.RetrievePolicyResponse]
	AccessCreateRole     freighter.UnaryServer[access.CreateRoleRequest, access.CreateRoleResponse]
	AccessDeleteRole     freighter.UnaryServer[access.DeleteRoleRequest, types.Nil]
	AccessRetrieveRole   freighter.UnaryServer[access.RetrieveRoleRequest, access.RetrieveRoleResponse]
	AccessAssignRole     freighter.UnaryServer[access.AssignRoleRequest, types.Nil]
	AccessUnassignRole   freighter.UnaryServer[access.UnassignRoleRequest, types.Nil]
	// STATUS
	StatusSet      freighter.UnaryServer[status.SetRequest, status.SetResponse]
	StatusRetrieve freighter.UnaryServer[status.RetrieveRequest, status.RetrieveResponse]
	StatusDelete   freighter.UnaryServer[status.DeleteRequest, types.Nil]
	// ARC
	ArcCreate   freighter.UnaryServer[arc.CreateRequest, arc.CreateResponse]
	ArcDelete   freighter.UnaryServer[arc.DeleteRequest, types.Nil]
	ArcRetrieve freighter.UnaryServer[arc.RetrieveRequest, arc.RetrieveResponse]
	ArcLSP      freighter.StreamServer[arc.LSPMessage, arc.LSPMessage]
	// VIEW
	ViewCreate   freighter.UnaryServer[view.CreateRequest, view.CreateResponse]
	ViewRetrieve freighter.UnaryServer[view.RetrieveRequest, view.RetrieveResponse]
	ViewDelete   freighter.UnaryServer[view.DeleteRequest, types.Nil]
}

// Layer wraps all implemented API services into a single container. Protocol-specific Layer
// implementations should use this struct during instantiation.
type Layer struct {
	config       config.Config
	Auth         *auth.Service
	User         *user.Service
	Framer       *framer.Service
	Channel      *channel.Service
	Connectivity *connectivity.Service
	Ontology     *ontology.Service
	Range        *ranger.Service
	Group        *group.Service
	Workspace    *workspace.Service
	Schematic    *schematic.Service
	LinePlot     *lineplot.Service
	Log          *log.Service
	Table        *table.Service
	Label        *label.Service
	Rack         *rack.Service
	Task         *task.Service
	Device       *device.Service
	Access       *access.Service
	Arc          *arc.Service
	Status       *status.Service
	View         *view.Service
}

// BindTo binds the API layer to the provided Transport implementation.
func (a *Layer) BindTo(t Transport) {
	var (
		tk                 = auth.TokenMiddleware(a.config.Service.Token)
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
		t.GroupCreate,
		t.GroupDelete,
		t.GroupRename,

		// RANGE
		t.RangeCreate,
		t.RangeRetrieve,
		t.RangeDelete,
		t.RangeKVGet,
		t.RangeKVSet,
		t.RangeKVDelete,
		t.RangeAliasSet,
		t.RangeAliasResolve,
		t.RangeAliasRetrieve,
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

		// SCHEMATIC SYMBOL
		t.SchematicCreateSymbol,
		t.SchematicRetrieveSymbol,
		t.SchematicDeleteSymbol,
		t.SchematicRenameSymbol,
		t.SchematicRetrieveSymbolGroup,

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

		// RACK
		t.RackCreate,
		t.RackRetrieve,
		t.RackDelete,

		// TASK
		t.TaskCreate,
		t.TaskRetrieve,
		t.TaskDelete,
		t.TaskCopy,

		// DEVICE
		t.DeviceCreate,
		t.DeviceRetrieve,
		t.DeviceDelete,

		// ACCESS
		t.AccessCreatePolicy,
		t.AccessDeletePolicy,
		t.AccessRetrievePolicy,
		t.AccessCreateRole,
		t.AccessDeleteRole,
		t.AccessRetrieveRole,
		t.AccessAssignRole,
		t.AccessUnassignRole,

		// STATUS
		t.StatusSet,
		t.StatusRetrieve,
		t.StatusDelete,

		// VIEW
		t.ViewCreate,
		t.ViewRetrieve,
		t.ViewDelete,

		// ARC
		t.ArcCreate,
		t.ArcDelete,
		t.ArcRetrieve,
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
	t.FrameDelete.BindHandler(a.Framer.Delete)

	// ONTOLOGY
	t.OntologyRetrieve.BindHandler(a.Ontology.Retrieve)
	t.OntologyAddChildren.BindHandler(a.Ontology.AddChildren)
	t.OntologyRemoveChildren.BindHandler(a.Ontology.RemoveChildren)
	t.OntologyMoveChildren.BindHandler(a.Ontology.MoveChildren)

	// GROUP
	t.GroupCreate.BindHandler(a.Group.Create)
	t.GroupDelete.BindHandler(a.Group.Delete)
	t.GroupRename.BindHandler(a.Group.Rename)

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
	t.RangeAliasRetrieve.BindHandler(a.Range.AliasRetrieve)
	t.RangeAliasList.BindHandler(a.Range.AliasList)
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
	t.SchematicCreateSymbol.BindHandler(a.Schematic.CreateSymbol)
	t.SchematicRetrieveSymbol.BindHandler(a.Schematic.RetrieveSymbol)
	t.SchematicDeleteSymbol.BindHandler(a.Schematic.DeleteSymbol)
	t.SchematicRenameSymbol.BindHandler(a.Schematic.RenameSymbol)
	t.SchematicRetrieveSymbolGroup.BindHandler(a.Schematic.RetrieveSymbolGroup)

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

	// RACK
	t.RackCreate.BindHandler(a.Rack.Create)
	t.RackRetrieve.BindHandler(a.Rack.Retrieve)
	t.RackDelete.BindHandler(a.Rack.Delete)

	// TASK
	t.TaskCreate.BindHandler(a.Task.Create)
	t.TaskRetrieve.BindHandler(a.Task.Retrieve)
	t.TaskDelete.BindHandler(a.Task.Delete)
	t.TaskCopy.BindHandler(a.Task.Copy)

	// DEVICE
	t.DeviceCreate.BindHandler(a.Device.Create)
	t.DeviceRetrieve.BindHandler(a.Device.Retrieve)
	t.DeviceDelete.BindHandler(a.Device.Delete)

	// ACCESS
	t.AccessCreatePolicy.BindHandler(a.Access.CreatePolicy)
	t.AccessDeletePolicy.BindHandler(a.Access.DeletePolicy)
	t.AccessRetrievePolicy.BindHandler(a.Access.RetrievePolicy)
	t.AccessCreateRole.BindHandler(a.Access.CreateRole)
	t.AccessDeleteRole.BindHandler(a.Access.DeleteRole)
	t.AccessRetrieveRole.BindHandler(a.Access.RetrieveRole)
	t.AccessAssignRole.BindHandler(a.Access.AssignRole)
	t.AccessUnassignRole.BindHandler(a.Access.UnassignRole)

	// STATUS
	t.StatusSet.BindHandler(a.Status.Set)
	t.StatusRetrieve.BindHandler(a.Status.Retrieve)
	t.StatusDelete.BindHandler(a.Status.Delete)

	// VIEW
	t.ViewCreate.BindHandler(a.View.Create)
	t.ViewRetrieve.BindHandler(a.View.Retrieve)
	t.ViewDelete.BindHandler(a.View.Delete)

	// ARC
	t.ArcCreate.BindHandler(a.Arc.Create)
	t.ArcDelete.BindHandler(a.Arc.Delete)
	t.ArcRetrieve.BindHandler(a.Arc.Retrieve)
	t.ArcLSP.BindHandler(a.Arc.LSP)
}

// New instantiates the server API layer using the provided Configs. This should only be
// called once.
func New(cfgs ...config.Config) (*Layer, error) {
	cfg, err := xconfig.New(config.Default, cfgs...)
	if err != nil {
		return nil, err
	}
	return &Layer{
		config:       cfg,
		Auth:         auth.NewService(cfg),
		User:         user.NewService(cfg),
		Access:       access.NewService(cfg),
		Framer:       framer.NewService(cfg),
		Channel:      channel.NewService(cfg),
		Connectivity: connectivity.NewService(cfg),
		Ontology:     ontology.NewService(cfg),
		Range:        ranger.NewService(cfg),
		Group:        group.NewService(cfg),
		Workspace:    workspace.NewService(cfg),
		Schematic:    schematic.NewService(cfg),
		LinePlot:     lineplot.NewService(cfg),
		Label:        label.NewService(cfg),
		Rack:         rack.NewService(cfg),
		Task:         task.NewService(cfg),
		Device:       device.NewService(cfg),
		Log:          log.NewService(cfg),
		Table:        table.NewService(cfg),
		Status:       status.NewService(cfg),
		Arc:          arc.NewService(cfg),
		View:         view.NewService(cfg),
	}, nil
}
