// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package http

import (
	"go/types"

	"github.com/synnaxlabs/freighter/fhttp"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/access"
	"github.com/synnaxlabs/synnax/pkg/api/arc"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/channel"
	"github.com/synnaxlabs/synnax/pkg/api/connectivity"
	"github.com/synnaxlabs/synnax/pkg/api/device"
	"github.com/synnaxlabs/synnax/pkg/api/framer"
	"github.com/synnaxlabs/synnax/pkg/api/group"
	framer2 "github.com/synnaxlabs/synnax/pkg/api/http/framer"
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
)

func New(router *fhttp.Router, cfg api.Config) api.Transport {
	framerResolver := fhttp.WithCodecResolver(framer2.NewCodecResolver(cfg.Distribution.Channel))
	return api.Transport{
		// AUTH
		AuthLogin:          fhttp.UnaryServer[auth.LoginRequest, auth.LoginResponse](router, "/api/v1/auth/login"),
		AuthChangePassword: fhttp.UnaryServer[auth.ChangePasswordRequest, types.Nil](router, "/api/v1/auth/change-password"),

		// USER
		UserRename:         fhttp.UnaryServer[user.RenameRequest, types.Nil](router, "/api/v1/user/rename"),
		UserChangeUsername: fhttp.UnaryServer[user.ChangeUsernameRequest, types.Nil](router, "/api/v1/user/change-username"),
		UserCreate:         fhttp.UnaryServer[user.CreateRequest, user.CreateResponse](router, "/api/v1/user/create"),
		UserDelete:         fhttp.UnaryServer[user.DeleteRequest, types.Nil](router, "/api/v1/user/delete"),
		UserRetrieve:       fhttp.UnaryServer[user.RetrieveRequest, user.RetrieveResponse](router, "/api/v1/user/retrieve"),

		// CHANNEL
		ChannelCreate:        fhttp.UnaryServer[channel.CreateRequest, channel.CreateResponse](router, "/api/v1/channel/create"),
		ChannelRetrieve:      fhttp.UnaryServer[channel.RetrieveRequest, channel.RetrieveResponse](router, "/api/v1/channel/retrieve"),
		ChannelDelete:        fhttp.UnaryServer[channel.DeleteRequest, types.Nil](router, "/api/v1/channel/delete"),
		ChannelRename:        fhttp.UnaryServer[channel.RenameRequest, types.Nil](router, "/api/v1/channel/rename"),
		ChannelRetrieveGroup: fhttp.UnaryServer[channel.RetrieveGroupRequest, channel.RetrieveGroupResponse](router, "/api/v1/channel/retrieve-group"),

		// CONNECTIVITY
		ConnectivityCheck: fhttp.UnaryServer[types.Nil, connectivity.CheckResponse](router, "/api/v1/connectivity/check"),

		// FRAME
		FrameWriter:   fhttp.StreamServer[framer.WriterRequest, framer.WriterResponse](router, "/api/v1/frame/write", framerResolver),
		FrameIterator: fhttp.StreamServer[framer.IteratorRequest, framer.IteratorResponse](router, "/api/v1/frame/iterate"),
		FrameStreamer: fhttp.StreamServer[framer.StreamerRequest, framer.StreamerResponse](router, "/api/v1/frame/stream", framerResolver),
		FrameDelete:   fhttp.UnaryServer[framer.DeleteRequest, types.Nil](router, "/api/v1/frame/delete"),

		// ONTOLOGY
		OntologyRetrieve:       fhttp.UnaryServer[ontology.RetrieveRequest, ontology.RetrieveResponse](router, "/api/v1/ontology/retrieve"),
		OntologyAddChildren:    fhttp.UnaryServer[ontology.AddChildrenRequest, types.Nil](router, "/api/v1/ontology/add-children"),
		OntologyRemoveChildren: fhttp.UnaryServer[ontology.RemoveChildrenRequest, types.Nil](router, "/api/v1/ontology/remove-children"),
		OntologyMoveChildren:   fhttp.UnaryServer[ontology.MoveChildrenRequest, types.Nil](router, "/api/v1/ontology/move-children"),

		// GROUP
		GroupCreate: fhttp.UnaryServer[group.CreateRequest, group.CreateResponse](router, "/api/v1/ontology/create-group"),
		GroupDelete: fhttp.UnaryServer[group.DeleteRequest, types.Nil](router, "/api/v1/ontology/delete-group"),
		GroupRename: fhttp.UnaryServer[group.RenameRequest, types.Nil](router, "/api/v1/ontology/rename-group"),

		// RANGE
		RangeRetrieve: fhttp.UnaryServer[ranger.RetrieveRequest, ranger.RetrieveResponse](router, "/api/v1/range/retrieve"),
		RangeCreate:   fhttp.UnaryServer[ranger.CreateRequest, ranger.CreateResponse](router, "/api/v1/range/create"),
		RangeDelete:   fhttp.UnaryServer[ranger.DeleteRequest, types.Nil](router, "/api/v1/range/delete"),
		RangeRename:   fhttp.UnaryServer[ranger.RenameRequest, types.Nil](router, "/api/v1/range/rename"),
		KVGet:         fhttp.UnaryServer[kv.GetRequest, kv.GetResponse](router, "/api/v1/range/kv/get"),
		KVSet:         fhttp.UnaryServer[kv.SetRequest, types.Nil](router, "/api/v1/range/kv/set"),
		KVDelete:      fhttp.UnaryServer[kv.DeleteRequest, types.Nil](router, "/api/v1/range/kv/delete"),
		AliasSet:      fhttp.UnaryServer[alias.SetRequest, types.Nil](router, "/api/v1/range/alias/set"),
		AliasResolve:  fhttp.UnaryServer[alias.ResolveRequest, alias.ResolveResponse](router, "/api/v1/range/alias/resolve"),
		AliasRetrieve: fhttp.UnaryServer[alias.RetrieveRequest, alias.RetrieveResponse](router, "/api/v1/range/alias/retrieve"),
		AliasList:     fhttp.UnaryServer[alias.ListRequest, alias.ListResponse](router, "/api/v1/range/alias/list"),
		AliasDelete:   fhttp.UnaryServer[alias.DeleteRequest, types.Nil](router, "/api/v1/range/alias/delete"),

		// WORKSPACE
		WorkspaceCreate:    fhttp.UnaryServer[workspace.CreateRequest, workspace.CreateResponse](router, "/api/v1/workspace/create"),
		WorkspaceRetrieve:  fhttp.UnaryServer[workspace.RetrieveRequest, workspace.RetrieveResponse](router, "/api/v1/workspace/retrieve"),
		WorkspaceDelete:    fhttp.UnaryServer[workspace.DeleteRequest, types.Nil](router, "/api/v1/workspace/delete"),
		WorkspaceRename:    fhttp.UnaryServer[workspace.RenameRequest, types.Nil](router, "/api/v1/workspace/rename"),
		WorkspaceSetLayout: fhttp.UnaryServer[workspace.SetLayoutRequest, types.Nil](router, "/api/v1/workspace/set-layout"),

		// SCHEMATIC
		SchematicCreate:   fhttp.UnaryServer[schematic.CreateRequest, schematic.CreateResponse](router, "/api/v1/workspace/schematic/create"),
		SchematicRetrieve: fhttp.UnaryServer[schematic.RetrieveRequest, schematic.RetrieveResponse](router, "/api/v1/workspace/schematic/retrieve"),
		SchematicDelete:   fhttp.UnaryServer[schematic.DeleteRequest, types.Nil](router, "/api/v1/workspace/schematic/delete"),
		SchematicRename:   fhttp.UnaryServer[schematic.RenameRequest, types.Nil](router, "/api/v1/workspace/schematic/rename"),
		SchematicSetData:  fhttp.UnaryServer[schematic.SetDataRequest, types.Nil](router, "/api/v1/workspace/schematic/set-data"),
		SchematicCopy:     fhttp.UnaryServer[schematic.CopyRequest, schematic.CopyResponse](router, "/api/v1/workspace/schematic/copy"),

		// SCHEMATIC SYMBOL
		SchematicCreateSymbol:        fhttp.UnaryServer[schematic.CreateSymbolRequest, schematic.CreateSymbolResponse](router, "/api/v1/workspace/schematic/symbol/create"),
		SchematicRetrieveSymbol:      fhttp.UnaryServer[schematic.RetrieveSymbolRequest, schematic.RetrieveSymbolResponse](router, "/api/v1/workspace/schematic/symbol/retrieve"),
		SchematicDeleteSymbol:        fhttp.UnaryServer[schematic.DeleteSymbolRequest, types.Nil](router, "/api/v1/workspace/schematic/symbol/delete"),
		SchematicRenameSymbol:        fhttp.UnaryServer[schematic.RenameSymbolRequest, types.Nil](router, "/api/v1/workspace/schematic/symbol/rename"),
		SchematicRetrieveSymbolGroup: fhttp.UnaryServer[schematic.RetrieveSymbolGroupRequest, schematic.RetrieveSymbolGroupResponse](router, "/api/v1/workspace/schematic/symbol/retrieve_group"),

		// LINE PLOT
		LinePlotCreate:   fhttp.UnaryServer[lineplot.CreateRequest, lineplot.CreateResponse](router, "/api/v1/workspace/lineplot/create"),
		LinePlotRetrieve: fhttp.UnaryServer[lineplot.RetrieveRequest, lineplot.RetrieveResponse](router, "/api/v1/workspace/lineplot/retrieve"),
		LinePlotDelete:   fhttp.UnaryServer[lineplot.DeleteRequest, types.Nil](router, "/api/v1/workspace/lineplot/delete"),
		LinePlotRename:   fhttp.UnaryServer[lineplot.RenameRequest, types.Nil](router, "/api/v1/workspace/lineplot/rename"),
		LinePlotSetData:  fhttp.UnaryServer[lineplot.SetDataRequest, types.Nil](router, "/api/v1/workspace/lineplot/set-data"),

		// LOG
		LogCreate:   fhttp.UnaryServer[log.CreateRequest, log.CreateResponse](router, "/api/v1/workspace/log/create"),
		LogRetrieve: fhttp.UnaryServer[log.RetrieveRequest, log.RetrieveResponse](router, "/api/v1/workspace/log/retrieve"),
		LogDelete:   fhttp.UnaryServer[log.DeleteRequest, types.Nil](router, "/api/v1/workspace/log/delete"),
		LogRename:   fhttp.UnaryServer[log.RenameRequest, types.Nil](router, "/api/v1/workspace/log/rename"),
		LogSetData:  fhttp.UnaryServer[log.SetDataRequest, types.Nil](router, "/api/v1/workspace/log/set-data"),

		// TABLE
		TableCreate:   fhttp.UnaryServer[table.CreateRequest, table.CreateResponse](router, "/api/v1/workspace/table/create"),
		TableRetrieve: fhttp.UnaryServer[table.RetrieveRequest, table.RetrieveResponse](router, "/api/v1/workspace/table/retrieve"),
		TableDelete:   fhttp.UnaryServer[table.DeleteRequest, types.Nil](router, "/api/v1/workspace/table/delete"),
		TableRename:   fhttp.UnaryServer[table.RenameRequest, types.Nil](router, "/api/v1/workspace/table/rename"),
		TableSetData:  fhttp.UnaryServer[table.SetDataRequest, types.Nil](router, "/api/v1/workspace/table/set-data"),

		// LABEL
		LabelCreate:   fhttp.UnaryServer[label.CreateRequest, label.CreateResponse](router, "/api/v1/label/create"),
		LabelRetrieve: fhttp.UnaryServer[label.RetrieveRequest, label.RetrieveResponse](router, "/api/v1/label/retrieve"),
		LabelDelete:   fhttp.UnaryServer[label.DeleteRequest, types.Nil](router, "/api/v1/label/delete"),
		LabelAdd:      fhttp.UnaryServer[label.AddRequest, types.Nil](router, "/api/v1/label/set"),
		LabelRemove:   fhttp.UnaryServer[label.RemoveRequest, types.Nil](router, "/api/v1/label/remove"),

		// RACK
		RackCreate:   fhttp.UnaryServer[rack.CreateRequest, rack.CreateResponse](router, "/api/v1/rack/create"),
		RackRetrieve: fhttp.UnaryServer[rack.RetrieveRequest, rack.RetrieveResponse](router, "/api/v1/rack/retrieve"),
		RackDelete:   fhttp.UnaryServer[rack.DeleteRequest, types.Nil](router, "/api/v1/rack/delete"),

		// TASK
		TaskCreate:   fhttp.UnaryServer[task.CreateRequest, task.CreateResponse](router, "/api/v1/task/create"),
		TaskRetrieve: fhttp.UnaryServer[task.RetrieveRequest, task.RetrieveResponse](router, "/api/v1/task/retrieve"),
		TaskDelete:   fhttp.UnaryServer[task.DeleteRequest, types.Nil](router, "/api/v1/task/delete"),
		TaskCopy:     fhttp.UnaryServer[task.CopyRequest, task.CopyResponse](router, "/api/v1/task/copy"),

		// DEVICE
		DeviceCreate:   fhttp.UnaryServer[device.CreateRequest, device.CreateResponse](router, "/api/v1/device/create"),
		DeviceRetrieve: fhttp.UnaryServer[device.RetrieveRequest, device.RetrieveResponse](router, "/api/v1/device/retrieve"),
		DeviceDelete:   fhttp.UnaryServer[device.DeleteRequest, types.Nil](router, "/api/v1/device/delete"),

		// ACCESS
		AccessCreatePolicy:   fhttp.UnaryServer[access.CreatePolicyRequest, access.CreatePolicyResponse](router, "/api/v1/access/policy/create"),
		AccessDeletePolicy:   fhttp.UnaryServer[access.DeletePolicyRequest, types.Nil](router, "/api/v1/access/policy/delete"),
		AccessRetrievePolicy: fhttp.UnaryServer[access.RetrievePolicyRequest, access.RetrievePolicyResponse](router, "/api/v1/access/policy/retrieve"),
		AccessCreateRole:     fhttp.UnaryServer[access.CreateRoleRequest, access.CreateRoleResponse](router, "/api/v1/access/role/create"),
		AccessDeleteRole:     fhttp.UnaryServer[access.DeleteRoleRequest, types.Nil](router, "/api/v1/access/role/delete"),
		AccessRetrieveRole:   fhttp.UnaryServer[access.RetrieveRoleRequest, access.RetrieveRoleResponse](router, "/api/v1/access/role/retrieve"),
		AccessAssignRole:     fhttp.UnaryServer[access.AssignRoleRequest, types.Nil](router, "/api/v1/access/role/assign"),
		AccessUnassignRole:   fhttp.UnaryServer[access.UnassignRoleRequest, types.Nil](router, "/api/v1/access/role/unassign"),

		// ARC
		ArcCreate:   fhttp.UnaryServer[arc.CreateRequest, arc.CreateResponse](router, "/api/v1/arc/create"),
		ArcDelete:   fhttp.UnaryServer[arc.DeleteRequest, types.Nil](router, "/api/v1/arc/delete"),
		ArcRetrieve: fhttp.UnaryServer[arc.RetrieveRequest, arc.RetrieveResponse](router, "/api/v1/arc/retrieve"),
		ArcLSP:      fhttp.StreamServer[arc.LSPMessage, arc.LSPMessage](router, "/api/v1/arc/lsp"),

		// STATUS
		StatusSet:      fhttp.UnaryServer[status.SetRequest, status.SetResponse](router, "/api/v1/status/set"),
		StatusRetrieve: fhttp.UnaryServer[status.RetrieveRequest, status.RetrieveResponse](router, "/api/v1/status/retrieve"),
		StatusDelete:   fhttp.UnaryServer[status.DeleteRequest, types.Nil](router, "/api/v1/status/delete"),

		// VIEW
		ViewCreate:   fhttp.UnaryServer[view.CreateRequest, view.CreateResponse](router, "/api/v1/view/create"),
		ViewRetrieve: fhttp.UnaryServer[view.RetrieveRequest, view.RetrieveResponse](router, "/api/v1/view/retrieve"),
		ViewDelete:   fhttp.UnaryServer[view.DeleteRequest, types.Nil](router, "/api/v1/view/delete"),
	}
}
