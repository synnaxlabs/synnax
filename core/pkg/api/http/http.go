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

	fhttp "github.com/synnaxlabs/freighter/http"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/access"
	"github.com/synnaxlabs/synnax/pkg/api/arc"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/channel"
	"github.com/synnaxlabs/synnax/pkg/api/connectivity"
	"github.com/synnaxlabs/synnax/pkg/api/device"
	"github.com/synnaxlabs/synnax/pkg/api/framer"
	"github.com/synnaxlabs/synnax/pkg/api/group"
	httpframer "github.com/synnaxlabs/synnax/pkg/api/http/framer"
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
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
)

func NewTransport(router *fhttp.Router, ch *distchannel.Service) api.Transport {
	framerCodec := httpframer.WithCodec(ch)
	return api.Transport{
		// AUTH
		AuthLogin:          fhttp.NewUnaryServer[auth.LoginRequest, auth.LoginResponse](router, "/api/v1/auth/login"),
		AuthChangePassword: fhttp.NewUnaryServer[auth.ChangePasswordRequest, types.Nil](router, "/api/v1/auth/change-password"),

		// USER
		UserRename:         fhttp.NewUnaryServer[user.RenameRequest, types.Nil](router, "/api/v1/user/rename"),
		UserChangeUsername: fhttp.NewUnaryServer[user.ChangeUsernameRequest, types.Nil](router, "/api/v1/user/change-username"),
		UserCreate:         fhttp.NewUnaryServer[user.CreateRequest, user.CreateResponse](router, "/api/v1/user/create"),
		UserDelete:         fhttp.NewUnaryServer[user.DeleteRequest, types.Nil](router, "/api/v1/user/delete"),
		UserRetrieve:       fhttp.NewUnaryServer[user.RetrieveRequest, user.RetrieveResponse](router, "/api/v1/user/retrieve"),

		// CHANNEL
		ChannelCreate:        fhttp.NewUnaryServer[channel.CreateRequest, channel.CreateResponse](router, "/api/v1/channel/create"),
		ChannelRetrieve:      fhttp.NewUnaryServer[channel.RetrieveRequest, channel.RetrieveResponse](router, "/api/v1/channel/retrieve"),
		ChannelDelete:        fhttp.NewUnaryServer[channel.DeleteRequest, types.Nil](router, "/api/v1/channel/delete"),
		ChannelRename:        fhttp.NewUnaryServer[channel.RenameRequest, types.Nil](router, "/api/v1/channel/rename"),
		ChannelRetrieveGroup: fhttp.NewUnaryServer[channel.RetrieveGroupRequest, channel.RetrieveGroupResponse](router, "/api/v1/channel/retrieve-group"),

		// CONNECTIVITY
		ConnectivityCheck: fhttp.NewUnaryServer[types.Nil, connectivity.CheckResponse](router, "/api/v1/connectivity/check"),

		// FRAME
		FrameWriter:   fhttp.NewStreamServer[framer.WriterRequest, framer.WriterResponse](router, "/api/v1/frame/write", framerCodec),
		FrameIterator: fhttp.NewStreamServer[framer.IteratorRequest, framer.IteratorResponse](router, "/api/v1/frame/iterate"),
		FrameStreamer: fhttp.NewStreamServer[framer.StreamerRequest, framer.StreamerResponse](router, "/api/v1/frame/stream", framerCodec),
		FrameDelete:   fhttp.NewUnaryServer[framer.DeleteRequest, types.Nil](router, "/api/v1/frame/delete"),

		// ONTOLOGY
		OntologyRetrieve:       fhttp.NewUnaryServer[ontology.RetrieveRequest, ontology.RetrieveResponse](router, "/api/v1/ontology/retrieve"),
		OntologyAddChildren:    fhttp.NewUnaryServer[ontology.AddChildrenRequest, types.Nil](router, "/api/v1/ontology/add-children"),
		OntologyRemoveChildren: fhttp.NewUnaryServer[ontology.RemoveChildrenRequest, types.Nil](router, "/api/v1/ontology/remove-children"),
		OntologyMoveChildren:   fhttp.NewUnaryServer[ontology.MoveChildrenRequest, types.Nil](router, "/api/v1/ontology/move-children"),

		// GROUP
		GroupCreate: fhttp.NewUnaryServer[group.CreateRequest, group.CreateResponse](router, "/api/v1/ontology/create-group"),
		GroupDelete: fhttp.NewUnaryServer[group.DeleteRequest, types.Nil](router, "/api/v1/ontology/delete-group"),
		GroupRename: fhttp.NewUnaryServer[group.RenameRequest, types.Nil](router, "/api/v1/ontology/rename-group"),

		// RANGE
		RangeRetrieve: fhttp.NewUnaryServer[ranger.RetrieveRequest, ranger.RetrieveResponse](router, "/api/v1/range/retrieve"),
		RangeCreate:   fhttp.NewUnaryServer[ranger.CreateRequest, ranger.CreateResponse](router, "/api/v1/range/create"),
		RangeDelete:   fhttp.NewUnaryServer[ranger.DeleteRequest, types.Nil](router, "/api/v1/range/delete"),
		RangeRename:   fhttp.NewUnaryServer[ranger.RenameRequest, types.Nil](router, "/api/v1/range/rename"),

		// KV
		KVGet:    fhttp.NewUnaryServer[kv.GetRequest, kv.GetResponse](router, "/api/v1/range/kv/get"),
		KVSet:    fhttp.NewUnaryServer[kv.SetRequest, types.Nil](router, "/api/v1/range/kv/set"),
		KVDelete: fhttp.NewUnaryServer[kv.DeleteRequest, types.Nil](router, "/api/v1/range/kv/delete"),

		// ALIAS
		AliasSet:      fhttp.NewUnaryServer[alias.SetRequest, types.Nil](router, "/api/v1/range/alias/set"),
		AliasResolve:  fhttp.NewUnaryServer[alias.ResolveRequest, alias.ResolveResponse](router, "/api/v1/range/alias/resolve"),
		AliasRetrieve: fhttp.NewUnaryServer[alias.RetrieveRequest, alias.RetrieveResponse](router, "/api/v1/range/alias/retrieve"),
		AliasList:     fhttp.NewUnaryServer[alias.ListRequest, alias.ListResponse](router, "/api/v1/range/alias/list"),
		AliasDelete:   fhttp.NewUnaryServer[alias.DeleteRequest, types.Nil](router, "/api/v1/range/alias/delete"),

		// WORKSPACE
		WorkspaceCreate:    fhttp.NewUnaryServer[workspace.CreateRequest, workspace.CreateResponse](router, "/api/v1/workspace/create"),
		WorkspaceRetrieve:  fhttp.NewUnaryServer[workspace.RetrieveRequest, workspace.RetrieveResponse](router, "/api/v1/workspace/retrieve"),
		WorkspaceDelete:    fhttp.NewUnaryServer[workspace.DeleteRequest, types.Nil](router, "/api/v1/workspace/delete"),
		WorkspaceRename:    fhttp.NewUnaryServer[workspace.RenameRequest, types.Nil](router, "/api/v1/workspace/rename"),
		WorkspaceSetLayout: fhttp.NewUnaryServer[workspace.SetLayoutRequest, types.Nil](router, "/api/v1/workspace/set-layout"),

		// SCHEMATIC
		SchematicCreate:   fhttp.NewUnaryServer[schematic.CreateRequest, schematic.CreateResponse](router, "/api/v1/schematic/create"),
		SchematicRetrieve: fhttp.NewUnaryServer[schematic.RetrieveRequest, schematic.RetrieveResponse](router, "/api/v1/schematic/retrieve"),
		SchematicDelete:   fhttp.NewUnaryServer[schematic.DeleteRequest, types.Nil](router, "/api/v1/schematic/delete"),
		SchematicRename:   fhttp.NewUnaryServer[schematic.RenameRequest, types.Nil](router, "/api/v1/schematic/rename"),
		SchematicSetData:  fhttp.NewUnaryServer[schematic.SetDataRequest, types.Nil](router, "/api/v1/schematic/set-data"),
		SchematicCopy:     fhttp.NewUnaryServer[schematic.CopyRequest, schematic.CopyResponse](router, "/api/v1/schematic/copy"),

		// SCHEMATIC SYMBOL
		SchematicCreateSymbol:        fhttp.NewUnaryServer[schematic.CreateSymbolRequest, schematic.CreateSymbolResponse](router, "/api/v1/schematic/symbol/create"),
		SchematicRetrieveSymbol:      fhttp.NewUnaryServer[schematic.RetrieveSymbolRequest, schematic.RetrieveSymbolResponse](router, "/api/v1/schematic/symbol/retrieve"),
		SchematicDeleteSymbol:        fhttp.NewUnaryServer[schematic.DeleteSymbolRequest, types.Nil](router, "/api/v1/schematic/symbol/delete"),
		SchematicRenameSymbol:        fhttp.NewUnaryServer[schematic.RenameSymbolRequest, types.Nil](router, "/api/v1/schematic/symbol/rename"),
		SchematicRetrieveSymbolGroup: fhttp.NewUnaryServer[schematic.RetrieveSymbolGroupRequest, schematic.RetrieveSymbolGroupResponse](router, "/api/v1/schematic/symbol/retrieve-group"),

		// LINE PLOT
		LinePlotCreate:   fhttp.NewUnaryServer[lineplot.CreateRequest, lineplot.CreateResponse](router, "/api/v1/lineplot/create"),
		LinePlotRetrieve: fhttp.NewUnaryServer[lineplot.RetrieveRequest, lineplot.RetrieveResponse](router, "/api/v1/lineplot/retrieve"),
		LinePlotDelete:   fhttp.NewUnaryServer[lineplot.DeleteRequest, types.Nil](router, "/api/v1/lineplot/delete"),
		LinePlotRename:   fhttp.NewUnaryServer[lineplot.RenameRequest, types.Nil](router, "/api/v1/lineplot/rename"),
		LinePlotSetData:  fhttp.NewUnaryServer[lineplot.SetDataRequest, types.Nil](router, "/api/v1/lineplot/set-data"),

		// LOG
		LogCreate:   fhttp.NewUnaryServer[log.CreateRequest, log.CreateResponse](router, "/api/v1/log/create"),
		LogRetrieve: fhttp.NewUnaryServer[log.RetrieveRequest, log.RetrieveResponse](router, "/api/v1/log/retrieve"),
		LogDelete:   fhttp.NewUnaryServer[log.DeleteRequest, types.Nil](router, "/api/v1/log/delete"),
		LogRename:   fhttp.NewUnaryServer[log.RenameRequest, types.Nil](router, "/api/v1/log/rename"),
		LogSetData:  fhttp.NewUnaryServer[log.SetDataRequest, types.Nil](router, "/api/v1/log/set-data"),

		// TABLE
		TableCreate:   fhttp.NewUnaryServer[table.CreateRequest, table.CreateResponse](router, "/api/v1/table/create"),
		TableRetrieve: fhttp.NewUnaryServer[table.RetrieveRequest, table.RetrieveResponse](router, "/api/v1/table/retrieve"),
		TableDelete:   fhttp.NewUnaryServer[table.DeleteRequest, types.Nil](router, "/api/v1/table/delete"),
		TableRename:   fhttp.NewUnaryServer[table.RenameRequest, types.Nil](router, "/api/v1/table/rename"),
		TableSetData:  fhttp.NewUnaryServer[table.SetDataRequest, types.Nil](router, "/api/v1/table/set-data"),

		// LABEL
		LabelCreate:   fhttp.NewUnaryServer[label.CreateRequest, label.CreateResponse](router, "/api/v1/label/create"),
		LabelRetrieve: fhttp.NewUnaryServer[label.RetrieveRequest, label.RetrieveResponse](router, "/api/v1/label/retrieve"),
		LabelDelete:   fhttp.NewUnaryServer[label.DeleteRequest, types.Nil](router, "/api/v1/label/delete"),
		LabelAdd:      fhttp.NewUnaryServer[label.AddRequest, types.Nil](router, "/api/v1/label/set"),
		LabelRemove:   fhttp.NewUnaryServer[label.RemoveRequest, types.Nil](router, "/api/v1/label/remove"),

		// RACK
		RackCreate:   fhttp.NewUnaryServer[rack.CreateRequest, rack.CreateResponse](router, "/api/v1/rack/create"),
		RackRetrieve: fhttp.NewUnaryServer[rack.RetrieveRequest, rack.RetrieveResponse](router, "/api/v1/rack/retrieve"),
		RackDelete:   fhttp.NewUnaryServer[rack.DeleteRequest, types.Nil](router, "/api/v1/rack/delete"),

		// TASK
		TaskCreate:   fhttp.NewUnaryServer[task.CreateRequest, task.CreateResponse](router, "/api/v1/task/create"),
		TaskRetrieve: fhttp.NewUnaryServer[task.RetrieveRequest, task.RetrieveResponse](router, "/api/v1/task/retrieve"),
		TaskDelete:   fhttp.NewUnaryServer[task.DeleteRequest, types.Nil](router, "/api/v1/task/delete"),
		TaskCopy:     fhttp.NewUnaryServer[task.CopyRequest, task.CopyResponse](router, "/api/v1/task/copy"),

		// DEVICE
		DeviceCreate:   fhttp.NewUnaryServer[device.CreateRequest, device.CreateResponse](router, "/api/v1/device/create"),
		DeviceRetrieve: fhttp.NewUnaryServer[device.RetrieveRequest, device.RetrieveResponse](router, "/api/v1/device/retrieve"),
		DeviceDelete:   fhttp.NewUnaryServer[device.DeleteRequest, types.Nil](router, "/api/v1/device/delete"),

		// ACCESS
		AccessCreatePolicy:   fhttp.NewUnaryServer[access.CreatePolicyRequest, access.CreatePolicyResponse](router, "/api/v1/access/policy/create"),
		AccessDeletePolicy:   fhttp.NewUnaryServer[access.DeletePolicyRequest, types.Nil](router, "/api/v1/access/policy/delete"),
		AccessRetrievePolicy: fhttp.NewUnaryServer[access.RetrievePolicyRequest, access.RetrievePolicyResponse](router, "/api/v1/access/policy/retrieve"),
		AccessCreateRole:     fhttp.NewUnaryServer[access.CreateRoleRequest, access.CreateRoleResponse](router, "/api/v1/access/role/create"),
		AccessDeleteRole:     fhttp.NewUnaryServer[access.DeleteRoleRequest, types.Nil](router, "/api/v1/access/role/delete"),
		AccessRetrieveRole:   fhttp.NewUnaryServer[access.RetrieveRoleRequest, access.RetrieveRoleResponse](router, "/api/v1/access/role/retrieve"),
		AccessAssignRole:     fhttp.NewUnaryServer[access.AssignRoleRequest, types.Nil](router, "/api/v1/access/role/assign"),
		AccessUnassignRole:   fhttp.NewUnaryServer[access.UnassignRoleRequest, types.Nil](router, "/api/v1/access/role/unassign"),

		// ARC
		ArcCreate:   fhttp.NewUnaryServer[arc.CreateRequest, arc.CreateResponse](router, "/api/v1/arc/create"),
		ArcDelete:   fhttp.NewUnaryServer[arc.DeleteRequest, types.Nil](router, "/api/v1/arc/delete"),
		ArcRetrieve: fhttp.NewUnaryServer[arc.RetrieveRequest, arc.RetrieveResponse](router, "/api/v1/arc/retrieve"),
		ArcLSP:      fhttp.NewStreamServer[arc.LSPMessage, arc.LSPMessage](router, "/api/v1/arc/lsp"),

		// STATUS
		StatusSet:      fhttp.NewUnaryServer[status.SetRequest, status.SetResponse](router, "/api/v1/status/set"),
		StatusRetrieve: fhttp.NewUnaryServer[status.RetrieveRequest, status.RetrieveResponse](router, "/api/v1/status/retrieve"),
		StatusDelete:   fhttp.NewUnaryServer[status.DeleteRequest, types.Nil](router, "/api/v1/status/delete"),

		// VIEW
		ViewCreate:   fhttp.NewUnaryServer[view.CreateRequest, view.CreateResponse](router, "/api/v1/view/create"),
		ViewRetrieve: fhttp.NewUnaryServer[view.RetrieveRequest, view.RetrieveResponse](router, "/api/v1/view/retrieve"),
		ViewDelete:   fhttp.NewUnaryServer[view.DeleteRequest, types.Nil](router, "/api/v1/view/delete"),
	}
}
