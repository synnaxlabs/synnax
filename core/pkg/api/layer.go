// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package api implements the client interfaces for interacting with the Synnax cluster.
// The package is transport agnostic, defining freighter-compatible interfaces (via the
// Transport struct) and service implementations (via the Layer struct) for all of its
// services.
package api

import (
	"go/types"

	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/alamos"
	fgorp "github.com/synnaxlabs/freighter/gorp"
	"github.com/synnaxlabs/freighter/recovery"
	"github.com/synnaxlabs/synnax/pkg/api/access"
	"github.com/synnaxlabs/synnax/pkg/api/arc"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/channel"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/api/connectivity"
	"github.com/synnaxlabs/synnax/pkg/api/device"
	"github.com/synnaxlabs/synnax/pkg/api/framer"
	"github.com/synnaxlabs/synnax/pkg/api/group"
	"github.com/synnaxlabs/synnax/pkg/api/imex"
	"github.com/synnaxlabs/synnax/pkg/api/label"
	"github.com/synnaxlabs/synnax/pkg/api/lineplot"
	"github.com/synnaxlabs/synnax/pkg/api/log"
	"github.com/synnaxlabs/synnax/pkg/api/ontology"
	"github.com/synnaxlabs/synnax/pkg/api/rack"
	"github.com/synnaxlabs/synnax/pkg/api/ranger"
	"github.com/synnaxlabs/synnax/pkg/api/ranger/alias"
	"github.com/synnaxlabs/synnax/pkg/api/ranger/kv"
	"github.com/synnaxlabs/synnax/pkg/api/schematic"
	"github.com/synnaxlabs/synnax/pkg/api/status"
	"github.com/synnaxlabs/synnax/pkg/api/table"
	"github.com/synnaxlabs/synnax/pkg/api/task"
	"github.com/synnaxlabs/synnax/pkg/api/user"
	"github.com/synnaxlabs/synnax/pkg/api/view"
	"github.com/synnaxlabs/synnax/pkg/api/workspace"
	xconfig "github.com/synnaxlabs/x/config"
)

// LayerConfig is the configuration for opening the API layer.
type LayerConfig = config.LayerConfig

var DefaultLayerConfig = config.DefaultLayerConfig

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
	RangeCreate   freighter.UnaryServer[ranger.CreateRequest, ranger.CreateResponse]
	RangeRetrieve freighter.UnaryServer[ranger.RetrieveRequest, ranger.RetrieveResponse]
	RangeDelete   freighter.UnaryServer[ranger.DeleteRequest, types.Nil]
	RangeRename   freighter.UnaryServer[ranger.RenameRequest, types.Nil]
	// KV
	KVGet    freighter.UnaryServer[kv.GetRequest, kv.GetResponse]
	KVSet    freighter.UnaryServer[kv.SetRequest, types.Nil]
	KVDelete freighter.UnaryServer[kv.DeleteRequest, types.Nil]
	// ALIAS
	AliasSet      freighter.UnaryServer[alias.SetRequest, types.Nil]
	AliasResolve  freighter.UnaryServer[alias.ResolveRequest, alias.ResolveResponse]
	AliasDelete   freighter.UnaryServer[alias.DeleteRequest, types.Nil]
	AliasList     freighter.UnaryServer[alias.ListRequest, alias.ListResponse]
	AliasRetrieve freighter.UnaryServer[alias.RetrieveRequest, alias.RetrieveResponse]
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
	SchematicDispatch freighter.UnaryServer[schematic.DispatchRequest, types.Nil]
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
	LogDispatch freighter.UnaryServer[log.DispatchRequest, types.Nil]
	// TABLE
	TableCreate   freighter.UnaryServer[table.CreateRequest, table.CreateResponse]
	TableRetrieve freighter.UnaryServer[table.RetrieveRequest, table.RetrieveResponse]
	TableDelete   freighter.UnaryServer[table.DeleteRequest, types.Nil]
	TableDispatch freighter.UnaryServer[table.DispatchRequest, types.Nil]
	// LINE PLOT
	LinePlotCreate   freighter.UnaryServer[lineplot.CreateRequest, lineplot.CreateResponse]
	LinePlotRetrieve freighter.UnaryServer[lineplot.RetrieveRequest, lineplot.RetrieveResponse]
	LinePlotDelete   freighter.UnaryServer[lineplot.DeleteRequest, types.Nil]
	LinePlotDispatch freighter.UnaryServer[lineplot.DispatchRequest, types.Nil]
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
	StatusSet            freighter.UnaryServer[status.SetRequest, status.SetResponse]
	StatusRetrieve       freighter.UnaryServer[status.RetrieveRequest, status.RetrieveResponse]
	StatusDelete         freighter.UnaryServer[status.DeleteRequest, types.Nil]
	StatusSetByKeyOrName freighter.UnaryServer[status.SetByKeyOrNameRequest, status.SetByKeyOrNameResponse]
	// ARC
	ArcCreate   freighter.UnaryServer[arc.CreateRequest, arc.CreateResponse]
	ArcDelete   freighter.UnaryServer[arc.DeleteRequest, types.Nil]
	ArcRetrieve freighter.UnaryServer[arc.RetrieveRequest, arc.RetrieveResponse]
	ArcLSP      freighter.StreamServer[arc.LSPMessage, arc.LSPMessage]
	// VIEW
	ViewCreate   freighter.UnaryServer[view.CreateRequest, view.CreateResponse]
	ViewRetrieve freighter.UnaryServer[view.RetrieveRequest, view.RetrieveResponse]
	ViewDelete   freighter.UnaryServer[view.DeleteRequest, types.Nil]
	// IMPORT/EXPORT
	ImExImport freighter.UnaryServer[imex.ImportRequest, imex.ImportResponse]
	ImExExport freighter.UnaryServer[imex.ExportRequest, imex.ExportResponse]
}

// Layer wraps all implemented API services into a single container. Protocol-specific Layer
// implementations should use this struct during instantiation.
type Layer struct {
	Workspace    *workspace.Service
	LinePlot     *lineplot.Service
	User         *user.Service
	Framer       *framer.Service
	Channel      *channel.Service
	Connectivity *connectivity.Service
	Ontology     *ontology.Service
	Range        *ranger.Service
	KV           *kv.Service
	Alias        *alias.Service
	Group        *group.Service
	Log          *log.Service
	Auth         *auth.Service
	Schematic    *schematic.Service
	View         *view.Service
	Table        *table.Service
	Label        *label.Service
	Rack         *rack.Service
	Task         *task.Service
	Device       *device.Service
	Access       *access.Service
	Arc          *arc.Service
	Status       *status.Service
	ImEx         *imex.Service
	config       config.LayerConfig
}

// BindTo binds the API layer to the provided Transport implementation.
func (l *Layer) BindTo(t Transport) {
	var (
		tk                 = auth.TokenMiddleware(l.config.Service.Token)
		instrumentation    = lo.Must(alamos.Middleware(alamos.Config{Instrumentation: l.config.Instrumentation}))
		rec                = recovery.Middleware(l.config.Instrumentation)
		insecureMiddleware = []freighter.Middleware{rec, instrumentation}
		secureMiddleware   = make(
			[]freighter.Middleware, len(insecureMiddleware), len(insecureMiddleware)+1,
		)
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
		t.RangeRename,

		// KV
		t.KVGet,
		t.KVSet,
		t.KVDelete,

		// ALIAS
		t.AliasSet,
		t.AliasResolve,
		t.AliasRetrieve,
		t.AliasList,
		t.AliasDelete,

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
		t.SchematicDispatch,
		t.SchematicCopy,

		// SCHEMATIC SYMBOL
		t.SchematicCreateSymbol,
		t.SchematicRetrieveSymbol,
		t.SchematicDeleteSymbol,
		t.SchematicRenameSymbol,
		t.SchematicRetrieveSymbolGroup,

		// LINE PLOT
		t.LinePlotCreate,
		t.LinePlotDispatch,
		t.LinePlotRetrieve,
		t.LinePlotDelete,

		// LOG
		t.LogCreate,
		t.LogRetrieve,
		t.LogDelete,
		t.LogDispatch,

		// TABLE
		t.TableCreate,
		t.TableRetrieve,
		t.TableDelete,
		t.TableDispatch,

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
		t.StatusSetByKeyOrName,

		// VIEW
		t.ViewCreate,
		t.ViewRetrieve,
		t.ViewDelete,

		// ARC
		t.ArcCreate,
		t.ArcDelete,
		t.ArcRetrieve,

		// IMPORT/EXPORT
		t.ImExImport,
		t.ImExExport,
	)

	db := l.config.Distribution.DB

	// AUTH
	t.AuthLogin.BindHandler(fgorp.CreateUnaryHandler(db, l.Auth.Login))
	t.AuthChangePassword.BindHandler(
		fgorp.CreateUnaryHandler(db, l.Auth.ChangePassword),
	)

	// USER
	t.UserRename.BindHandler(fgorp.CreateUnaryHandler(db, l.User.Rename))
	t.UserChangeUsername.BindHandler(
		fgorp.CreateUnaryHandler(db, l.User.ChangeUsername),
	)
	t.UserCreate.BindHandler(fgorp.CreateUnaryHandler(db, l.User.Create))
	t.UserDelete.BindHandler(fgorp.CreateUnaryHandler(db, l.User.Delete))
	t.UserRetrieve.BindHandler(fgorp.CreateUnaryHandler(db, l.User.Retrieve))

	// CHANNEL
	t.ChannelCreate.BindHandler(fgorp.CreateUnaryHandler(db, l.Channel.Create))
	t.ChannelRetrieve.BindHandler(fgorp.CreateUnaryHandler(db, l.Channel.Retrieve))
	t.ConnectivityCheck.BindHandler(l.Connectivity.Check)
	t.ChannelDelete.BindHandler(fgorp.CreateUnaryHandler(db, l.Channel.Delete))
	t.ChannelRename.BindHandler(fgorp.CreateUnaryHandler(db, l.Channel.Rename))
	t.ChannelRetrieveGroup.BindHandler(
		fgorp.CreateUnaryHandler(db, l.Channel.RetrieveGroup),
	)

	// FRAME
	t.FrameWriter.BindHandler(l.Framer.Write)
	t.FrameIterator.BindHandler(l.Framer.Iterate)
	t.FrameStreamer.BindHandler(l.Framer.Stream)
	t.FrameDelete.BindHandler(fgorp.CreateUnaryHandler(db, l.Framer.Delete))

	// ONTOLOGY
	t.OntologyRetrieve.BindHandler(fgorp.CreateUnaryHandler(db, l.Ontology.Retrieve))
	t.OntologyAddChildren.BindHandler(
		fgorp.CreateUnaryHandler(db, l.Ontology.AddChildren),
	)
	t.OntologyRemoveChildren.BindHandler(
		fgorp.CreateUnaryHandler(db, l.Ontology.RemoveChildren),
	)
	t.OntologyMoveChildren.BindHandler(
		fgorp.CreateUnaryHandler(db, l.Ontology.MoveChildren),
	)

	// GROUP
	t.GroupCreate.BindHandler(fgorp.CreateUnaryHandler(db, l.Group.Create))
	t.GroupDelete.BindHandler(fgorp.CreateUnaryHandler(db, l.Group.Delete))
	t.GroupRename.BindHandler(fgorp.CreateUnaryHandler(db, l.Group.Rename))

	// RANGE
	t.RangeRetrieve.BindHandler(fgorp.CreateUnaryHandler(db, l.Range.Retrieve))
	t.RangeCreate.BindHandler(fgorp.CreateUnaryHandler(db, l.Range.Create))
	t.RangeDelete.BindHandler(fgorp.CreateUnaryHandler(db, l.Range.Delete))
	t.RangeRename.BindHandler(fgorp.CreateUnaryHandler(db, l.Range.Rename))

	// KV
	t.KVGet.BindHandler(fgorp.CreateUnaryHandler(db, l.KV.Get))
	t.KVSet.BindHandler(fgorp.CreateUnaryHandler(db, l.KV.Set))
	t.KVDelete.BindHandler(fgorp.CreateUnaryHandler(db, l.KV.Delete))

	// ALIAS
	t.AliasSet.BindHandler(fgorp.CreateUnaryHandler(db, l.Alias.Set))
	t.AliasResolve.BindHandler(fgorp.CreateUnaryHandler(db, l.Alias.Resolve))
	t.AliasRetrieve.BindHandler(fgorp.CreateUnaryHandler(db, l.Alias.Retrieve))
	t.AliasList.BindHandler(fgorp.CreateUnaryHandler(db, l.Alias.List))
	t.AliasDelete.BindHandler(fgorp.CreateUnaryHandler(db, l.Alias.Delete))

	// WORKSPACE
	t.WorkspaceCreate.BindHandler(fgorp.CreateUnaryHandler(db, l.Workspace.Create))
	t.WorkspaceDelete.BindHandler(fgorp.CreateUnaryHandler(db, l.Workspace.Delete))
	t.WorkspaceRetrieve.BindHandler(fgorp.CreateUnaryHandler(db, l.Workspace.Retrieve))
	t.WorkspaceRename.BindHandler(fgorp.CreateUnaryHandler(db, l.Workspace.Rename))
	t.WorkspaceSetLayout.BindHandler(fgorp.CreateUnaryHandler(db, l.Workspace.SetLayout))

	// SCHEMATIC
	t.SchematicCreate.BindHandler(fgorp.CreateUnaryHandler(db, l.Schematic.Create))
	t.SchematicRetrieve.BindHandler(fgorp.CreateUnaryHandler(db, l.Schematic.Retrieve))
	t.SchematicDelete.BindHandler(fgorp.CreateUnaryHandler(db, l.Schematic.Delete))
	t.SchematicDispatch.BindHandler(fgorp.CreateUnaryHandler(db, l.Schematic.Dispatch))
	t.SchematicCopy.BindHandler(fgorp.CreateUnaryHandler(db, l.Schematic.Copy))

	// SCHEMATIC SYMBOL
	t.SchematicCreateSymbol.BindHandler(
		fgorp.CreateUnaryHandler(db, l.Schematic.CreateSymbol))
	t.SchematicRetrieveSymbol.BindHandler(
		fgorp.CreateUnaryHandler(db, l.Schematic.RetrieveSymbol),
	)
	t.SchematicDeleteSymbol.BindHandler(
		fgorp.CreateUnaryHandler(db, l.Schematic.DeleteSymbol),
	)
	t.SchematicRenameSymbol.BindHandler(
		fgorp.CreateUnaryHandler(db, l.Schematic.RenameSymbol),
	)
	t.SchematicRetrieveSymbolGroup.BindHandler(
		fgorp.CreateUnaryHandler(db, l.Schematic.RetrieveSymbolGroup),
	)

	// LINE PLOT
	t.LinePlotCreate.BindHandler(fgorp.CreateUnaryHandler(db, l.LinePlot.Create))
	t.LinePlotDispatch.BindHandler(fgorp.CreateUnaryHandler(db, l.LinePlot.Dispatch))
	t.LinePlotRetrieve.BindHandler(fgorp.CreateUnaryHandler(db, l.LinePlot.Retrieve))
	t.LinePlotDelete.BindHandler(fgorp.CreateUnaryHandler(db, l.LinePlot.Delete))

	// LOG
	t.LogCreate.BindHandler(fgorp.CreateUnaryHandler(db, l.Log.Create))
	t.LogRetrieve.BindHandler(fgorp.CreateUnaryHandler(db, l.Log.Retrieve))
	t.LogDelete.BindHandler(fgorp.CreateUnaryHandler(db, l.Log.Delete))
	t.LogDispatch.BindHandler(fgorp.CreateUnaryHandler(db, l.Log.Dispatch))

	// TABLE
	t.TableCreate.BindHandler(fgorp.CreateUnaryHandler(db, l.Table.Create))
	t.TableRetrieve.BindHandler(fgorp.CreateUnaryHandler(db, l.Table.Retrieve))
	t.TableDelete.BindHandler(fgorp.CreateUnaryHandler(db, l.Table.Delete))
	t.TableDispatch.BindHandler(fgorp.CreateUnaryHandler(db, l.Table.Dispatch))

	// LABEL
	t.LabelCreate.BindHandler(fgorp.CreateUnaryHandler(db, l.Label.Create))
	t.LabelRetrieve.BindHandler(fgorp.CreateUnaryHandler(db, l.Label.Retrieve))
	t.LabelDelete.BindHandler(fgorp.CreateUnaryHandler(db, l.Label.Delete))
	t.LabelAdd.BindHandler(fgorp.CreateUnaryHandler(db, l.Label.Add))
	t.LabelRemove.BindHandler(fgorp.CreateUnaryHandler(db, l.Label.Remove))

	// RACK
	t.RackCreate.BindHandler(fgorp.CreateUnaryHandler(db, l.Rack.Create))
	t.RackRetrieve.BindHandler(fgorp.CreateUnaryHandler(db, l.Rack.Retrieve))
	t.RackDelete.BindHandler(fgorp.CreateUnaryHandler(db, l.Rack.Delete))

	// TASK
	t.TaskCreate.BindHandler(fgorp.CreateUnaryHandler(db, l.Task.Create))
	t.TaskRetrieve.BindHandler(fgorp.CreateUnaryHandler(db, l.Task.Retrieve))
	t.TaskDelete.BindHandler(fgorp.CreateUnaryHandler(db, l.Task.Delete))
	t.TaskCopy.BindHandler(fgorp.CreateUnaryHandler(db, l.Task.Copy))

	// DEVICE
	t.DeviceCreate.BindHandler(fgorp.CreateUnaryHandler(db, l.Device.Create))
	t.DeviceRetrieve.BindHandler(fgorp.CreateUnaryHandler(db, l.Device.Retrieve))
	t.DeviceDelete.BindHandler(fgorp.CreateUnaryHandler(db, l.Device.Delete))

	// ACCESS
	t.AccessCreatePolicy.BindHandler(
		fgorp.CreateUnaryHandler(db, l.Access.CreatePolicy),
	)
	t.AccessDeletePolicy.BindHandler(
		fgorp.CreateUnaryHandler(db, l.Access.DeletePolicy),
	)
	t.AccessRetrievePolicy.BindHandler(
		fgorp.CreateUnaryHandler(db, l.Access.RetrievePolicy),
	)
	t.AccessCreateRole.BindHandler(fgorp.CreateUnaryHandler(db, l.Access.CreateRole))
	t.AccessDeleteRole.BindHandler(fgorp.CreateUnaryHandler(db, l.Access.DeleteRole))
	t.AccessRetrieveRole.BindHandler(
		fgorp.CreateUnaryHandler(db, l.Access.RetrieveRole),
	)
	t.AccessAssignRole.BindHandler(fgorp.CreateUnaryHandler(db, l.Access.AssignRole))
	t.AccessUnassignRole.BindHandler(
		fgorp.CreateUnaryHandler(db, l.Access.UnassignRole),
	)

	// STATUS
	t.StatusSet.BindHandler(fgorp.CreateUnaryHandler(db, l.Status.Set))
	t.StatusRetrieve.BindHandler(fgorp.CreateUnaryHandler(db, l.Status.Retrieve))
	t.StatusDelete.BindHandler(fgorp.CreateUnaryHandler(db, l.Status.Delete))
	t.StatusSetByKeyOrName.BindHandler(
		fgorp.CreateUnaryHandler(db, l.Status.SetByKeyOrName),
	)

	// VIEW
	t.ViewCreate.BindHandler(fgorp.CreateUnaryHandler(db, l.View.Create))
	t.ViewRetrieve.BindHandler(fgorp.CreateUnaryHandler(db, l.View.Retrieve))
	t.ViewDelete.BindHandler(fgorp.CreateUnaryHandler(db, l.View.Delete))

	// ARC
	t.ArcCreate.BindHandler(fgorp.CreateUnaryHandler(db, l.Arc.Create))
	t.ArcDelete.BindHandler(fgorp.CreateUnaryHandler(db, l.Arc.Delete))
	t.ArcRetrieve.BindHandler(fgorp.CreateUnaryHandler(db, l.Arc.Retrieve))
	t.ArcLSP.BindHandler(l.Arc.LSP)

	// IMPORT/EXPORT
	t.ImExImport.BindHandler(fgorp.CreateUnaryHandler(db, l.ImEx.Import))
	t.ImExExport.BindHandler(fgorp.CreateUnaryHandler(db, l.ImEx.Export))
}

// NewLayer instantiates the server API layer using the provided Configs. This should
// only be called once.
func NewLayer(cfgs ...LayerConfig) (*Layer, error) {
	cfg, err := xconfig.New(config.DefaultLayerConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	l := &Layer{config: cfg}
	if l.Auth, err = auth.NewService(cfg); err != nil {
		return nil, err
	}
	if l.User, err = user.NewService(cfg); err != nil {
		return nil, err
	}
	if l.Access, err = access.NewService(cfg); err != nil {
		return nil, err
	}
	if l.Framer, err = framer.NewService(cfg); err != nil {
		return nil, err
	}
	if l.Channel, err = channel.NewService(cfg); err != nil {
		return nil, err
	}
	if l.Connectivity, err = connectivity.NewService(cfg); err != nil {
		return nil, err
	}
	if l.Ontology, err = ontology.NewService(cfg); err != nil {
		return nil, err
	}
	if l.Range, err = ranger.NewService(cfg); err != nil {
		return nil, err
	}
	if l.KV, err = kv.NewService(cfg); err != nil {
		return nil, err
	}
	if l.Alias, err = alias.NewService(cfg); err != nil {
		return nil, err
	}
	if l.Group, err = group.NewService(cfg); err != nil {
		return nil, err
	}
	if l.Workspace, err = workspace.NewService(cfg); err != nil {
		return nil, err
	}
	if l.Schematic, err = schematic.NewService(cfg); err != nil {
		return nil, err
	}
	if l.LinePlot, err = lineplot.NewService(cfg); err != nil {
		return nil, err
	}
	if l.Label, err = label.NewService(cfg); err != nil {
		return nil, err
	}
	if l.Rack, err = rack.NewService(cfg); err != nil {
		return nil, err
	}
	if l.Task, err = task.NewService(cfg); err != nil {
		return nil, err
	}
	if l.Device, err = device.NewService(cfg); err != nil {
		return nil, err
	}
	if l.Log, err = log.NewService(cfg); err != nil {
		return nil, err
	}
	if l.Table, err = table.NewService(cfg); err != nil {
		return nil, err
	}
	if l.Status, err = status.NewService(cfg); err != nil {
		return nil, err
	}
	if l.Arc, err = arc.NewService(cfg); err != nil {
		return nil, err
	}
	if l.View, err = view.NewService(cfg); err != nil {
		return nil, err
	}
	if l.ImEx, err = imex.NewService(cfg); err != nil {
		return nil, err
	}
	return l, nil
}
