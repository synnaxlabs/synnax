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

	"github.com/synnaxlabs/freighter/http"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/access"
	"github.com/synnaxlabs/synnax/pkg/api/arc"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/channel"
	"github.com/synnaxlabs/synnax/pkg/api/connectivity"
	"github.com/synnaxlabs/synnax/pkg/api/control"
	"github.com/synnaxlabs/synnax/pkg/api/device"
	"github.com/synnaxlabs/synnax/pkg/api/group"
	"github.com/synnaxlabs/synnax/pkg/api/imex"
	"github.com/synnaxlabs/synnax/pkg/api/label"
	"github.com/synnaxlabs/synnax/pkg/api/lineplot"
	"github.com/synnaxlabs/synnax/pkg/api/log"
	"github.com/synnaxlabs/synnax/pkg/api/ontology"
	"github.com/synnaxlabs/synnax/pkg/api/panel"
	"github.com/synnaxlabs/synnax/pkg/api/project"
	"github.com/synnaxlabs/synnax/pkg/api/rack"
	"github.com/synnaxlabs/synnax/pkg/api/ranger"
	"github.com/synnaxlabs/synnax/pkg/api/ranger/alias"
	"github.com/synnaxlabs/synnax/pkg/api/ranger/kv"
	"github.com/synnaxlabs/synnax/pkg/api/schematic"
	"github.com/synnaxlabs/synnax/pkg/api/schematic/symbol"
	"github.com/synnaxlabs/synnax/pkg/api/status"
	"github.com/synnaxlabs/synnax/pkg/api/table"
	"github.com/synnaxlabs/synnax/pkg/api/task"
	"github.com/synnaxlabs/synnax/pkg/api/user"
	"github.com/synnaxlabs/synnax/pkg/api/view"
	"github.com/synnaxlabs/synnax/pkg/transport/http/framer"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/encoding/zip"
)

// Bind registers an HTTP endpoint for every API service onto router and binds the API
// layer's handlers and middleware to them. The frame codec resolves channel keys
// through the API layer's channel service.
func Bind(layer *api.Layer, router *http.Router) {
	framerServerOption := framer.WithCodec(layer.Channel)
	layer.BindTo(api.Transport{
		// AUTH
		AuthLogin: router.NewUnaryServer[auth.LoginRequest, auth.LoginResponse](
			"/api/v1/auth/login",
		),
		AuthChangePassword: router.NewUnaryServer[auth.ChangePasswordRequest, types.Nil](
			"/api/v1/auth/change-password",
		),

		// USER
		UserRename: router.NewUnaryServer[user.RenameRequest, types.Nil](
			"/api/v1/user/rename",
		),
		UserChangeUsername: router.NewUnaryServer[user.ChangeUsernameRequest, types.Nil](
			"/api/v1/user/change-username",
		),
		UserCreate: router.NewUnaryServer[user.CreateRequest, user.CreateResponse](
			"/api/v1/user/create",
		),
		UserDelete: router.NewUnaryServer[user.DeleteRequest, types.Nil](
			"/api/v1/user/delete",
		),
		UserRetrieve: router.NewUnaryServer[user.RetrieveRequest, user.RetrieveResponse](
			"/api/v1/user/retrieve",
		),

		// CHANNEL
		ChannelCreate: router.NewUnaryServer[channel.CreateRequest, channel.CreateResponse](
			"/api/v1/channel/create",
		),
		ChannelRetrieve: router.NewUnaryServer[channel.RetrieveRequest, channel.RetrieveResponse](
			"/api/v1/channel/retrieve",
		),
		ChannelDelete: router.NewUnaryServer[channel.DeleteRequest, types.Nil](
			"/api/v1/channel/delete",
		),
		ChannelRename: router.NewUnaryServer[channel.RenameRequest, types.Nil](
			"/api/v1/channel/rename",
		),
		ChannelRetrieveGroup: router.NewUnaryServer[channel.RetrieveGroupRequest, channel.RetrieveGroupResponse](
			"/api/v1/channel/retrieve-group",
		),

		// CONNECTIVITY
		ConnectivityCheck: router.NewUnaryServer[types.Nil, connectivity.CheckResponse](
			"/api/v1/connectivity/check",
		),

		// FRAME
		FrameWriter: router.NewStreamServer[framer.WriterRequest, framer.WriterResponse](
			"/api/v1/frame/write",
			framerServerOption,
		),
		FrameIterator: router.NewStreamServer[framer.IteratorRequest, framer.IteratorResponse](
			"/api/v1/frame/iterate",
			framerServerOption,
		),
		FrameStreamer: router.NewStreamServer[framer.StreamerRequest, framer.StreamerResponse](
			"/api/v1/frame/stream",
			framerServerOption,
		),
		FrameDelete: router.NewUnaryServer[framer.DeleteRequest, types.Nil](
			"/api/v1/frame/delete",
		),

		// CONTROL
		ControlRetrieve: router.NewUnaryServer[control.RetrieveRequest, control.RetrieveResponse](
			"/api/v1/control/retrieve",
		),

		// ONTOLOGY
		OntologyRetrieve: router.NewUnaryServer[ontology.RetrieveRequest, ontology.RetrieveResponse](
			"/api/v1/ontology/retrieve",
		),
		OntologyAddChildren: router.NewUnaryServer[ontology.AddChildrenRequest, types.Nil](
			"/api/v1/ontology/add-children",
		),
		OntologyRemoveChildren: router.NewUnaryServer[ontology.RemoveChildrenRequest, types.Nil](
			"/api/v1/ontology/remove-children",
		),
		OntologyMoveChildren: router.NewUnaryServer[ontology.MoveChildrenRequest, types.Nil](
			"/api/v1/ontology/move-children",
		),

		// GROUP
		GroupCreate: router.NewUnaryServer[group.CreateRequest, group.CreateResponse](
			"/api/v1/ontology/create-group",
		),
		GroupDelete: router.NewUnaryServer[group.DeleteRequest, types.Nil](
			"/api/v1/ontology/delete-group",
		),
		GroupRename: router.NewUnaryServer[group.RenameRequest, types.Nil](
			"/api/v1/ontology/rename-group",
		),
		GroupRetrieve: router.NewUnaryServer[group.RetrieveRequest, group.RetrieveResponse](
			"/api/v1/ontology/retrieve-group",
		),

		// RANGE
		RangeRetrieve: router.NewUnaryServer[ranger.RetrieveRequest, ranger.RetrieveResponse](
			"/api/v1/range/retrieve",
		),
		RangeCreate: router.NewUnaryServer[ranger.CreateRequest, ranger.CreateResponse](
			"/api/v1/range/create",
		),
		RangeDelete: router.NewUnaryServer[ranger.DeleteRequest, types.Nil](
			"/api/v1/range/delete",
		),
		RangeRename: router.NewUnaryServer[ranger.RenameRequest, types.Nil](
			"/api/v1/range/rename",
		),
		RangeSetEnd: router.NewUnaryServer[ranger.SetEndRequest, types.Nil](
			"/api/v1/range/set-end",
		),

		// KV
		KVGet: router.NewUnaryServer[kv.GetRequest, kv.GetResponse](
			"/api/v1/range/kv/get",
		),
		KVSet: router.NewUnaryServer[kv.SetRequest, types.Nil](
			"/api/v1/range/kv/set",
		),
		KVDelete: router.NewUnaryServer[kv.DeleteRequest, types.Nil](
			"/api/v1/range/kv/delete",
		),

		// ALIAS
		AliasSet: router.NewUnaryServer[alias.SetRequest, types.Nil](
			"/api/v1/range/alias/set",
		),
		AliasResolve: router.NewUnaryServer[alias.ResolveRequest, alias.ResolveResponse](
			"/api/v1/range/alias/resolve",
		),
		AliasRetrieve: router.NewUnaryServer[alias.RetrieveRequest, alias.RetrieveResponse](
			"/api/v1/range/alias/retrieve",
		),
		AliasList: router.NewUnaryServer[alias.ListRequest, alias.ListResponse](
			"/api/v1/range/alias/list",
		),
		AliasDelete: router.NewUnaryServer[alias.DeleteRequest, types.Nil](
			"/api/v1/range/alias/delete",
		),

		// PROJECT
		ProjectCreate: router.NewUnaryServer[project.CreateRequest, project.CreateResponse](
			"/api/v1/project/create",
		),
		ProjectRetrieve: router.NewUnaryServer[project.RetrieveRequest, project.RetrieveResponse](
			"/api/v1/project/retrieve",
		),
		ProjectDelete: router.NewUnaryServer[project.DeleteRequest, types.Nil](
			"/api/v1/project/delete",
		),
		ProjectRename: router.NewUnaryServer[project.RenameRequest, types.Nil](
			"/api/v1/project/rename",
		),
		ProjectSetLayout: router.NewUnaryServer[project.SetLayoutRequest, types.Nil](
			"/api/v1/project/set-layout",
		),
		ProjectExport: router.NewUnaryServer[project.ExportRequest, project.ExportResponse](
			"/api/v1/project/export",
			http.WithResponseEncoders(zip.Codec),
		),
		ProjectImport: router.NewUnaryServer[project.ImportRequest, project.ImportResponse](
			"/api/v1/project/import",
			http.WithRequestDecoders(zip.Codec),
		),

		// SCHEMATIC
		SchematicCreate: router.NewUnaryServer[schematic.CreateRequest, schematic.CreateResponse](
			"/api/v1/schematic/create",
		),
		SchematicRetrieve: router.NewUnaryServer[schematic.RetrieveRequest, schematic.RetrieveResponse](
			"/api/v1/schematic/retrieve",
		),
		SchematicDelete: router.NewUnaryServer[schematic.DeleteRequest, types.Nil](
			"/api/v1/schematic/delete",
		),
		SchematicDispatch: router.NewUnaryServer[schematic.DispatchRequest, types.Nil](
			"/api/v1/schematic/dispatch",
		),
		SchematicCopy: router.NewUnaryServer[schematic.CopyRequest, schematic.CopyResponse](
			"/api/v1/schematic/copy",
		),

		// SCHEMATIC SYMBOL
		SchematicSymbolCreate: router.NewUnaryServer[symbol.CreateRequest, symbol.CreateResponse](
			"/api/v1/schematic/symbol/create",
		),
		SchematicSymbolRetrieve: router.NewUnaryServer[symbol.RetrieveRequest, symbol.RetrieveResponse](
			"/api/v1/schematic/symbol/retrieve",
		),
		SchematicSymbolDelete: router.NewUnaryServer[symbol.DeleteRequest, types.Nil](
			"/api/v1/schematic/symbol/delete",
		),
		SchematicSymbolRename: router.NewUnaryServer[symbol.RenameRequest, types.Nil](
			"/api/v1/schematic/symbol/rename",
		),
		SchematicSymbolRetrieveGroup: router.NewUnaryServer[symbol.RetrieveGroupRequest, symbol.RetrieveGroupResponse](
			"/api/v1/schematic/symbol/retrieve-group",
		),
		SchematicSymbolExportGroup: router.NewUnaryServer[symbol.ExportGroupRequest, symbol.ExportGroupResponse](
			"/api/v1/schematic/symbol/group/export",
			http.WithResponseEncoders(zip.Codec),
		),
		SchematicSymbolImportGroup: router.NewUnaryServer[symbol.ImportGroupRequest, symbol.ImportGroupResponse](
			"/api/v1/schematic/symbol/group/import",
			http.WithRequestDecoders(zip.Codec),
		),
		SchematicSymbolDeleteGroup: router.NewUnaryServer[symbol.DeleteGroupRequest, types.Nil](
			"/api/v1/schematic/symbol/group/delete",
		),

		// LINE PLOT
		LinePlotCreate: router.NewUnaryServer[lineplot.CreateRequest, lineplot.CreateResponse](
			"/api/v1/lineplot/create",
		),
		LinePlotRetrieve: router.NewUnaryServer[lineplot.RetrieveRequest, lineplot.RetrieveResponse](
			"/api/v1/lineplot/retrieve",
		),
		LinePlotDelete: router.NewUnaryServer[lineplot.DeleteRequest, types.Nil](
			"/api/v1/lineplot/delete",
		),
		LinePlotDispatch: router.NewUnaryServer[lineplot.DispatchRequest, types.Nil](
			"/api/v1/lineplot/dispatch",
		),

		// PANEL
		PanelCreate: router.NewUnaryServer[panel.CreateRequest, panel.CreateResponse](
			"/api/v1/panel/create",
		),
		PanelRetrieve: router.NewUnaryServer[panel.RetrieveRequest, panel.RetrieveResponse](
			"/api/v1/panel/retrieve",
		),
		PanelDelete: router.NewUnaryServer[panel.DeleteRequest, types.Nil](
			"/api/v1/panel/delete",
		),
		PanelDispatch: router.NewUnaryServer[panel.DispatchRequest, types.Nil](
			"/api/v1/panel/dispatch",
		),

		// LOG
		LogCreate: router.NewUnaryServer[log.CreateRequest, log.CreateResponse](
			"/api/v1/log/create",
		),
		LogRetrieve: router.NewUnaryServer[log.RetrieveRequest, log.RetrieveResponse](
			"/api/v1/log/retrieve",
		),
		LogDelete: router.NewUnaryServer[log.DeleteRequest, types.Nil](
			"/api/v1/log/delete",
		),
		LogDispatch: router.NewUnaryServer[log.DispatchRequest, types.Nil](
			"/api/v1/log/dispatch",
		),

		// TABLE
		TableCreate: router.NewUnaryServer[table.CreateRequest, table.CreateResponse](
			"/api/v1/table/create",
		),
		TableRetrieve: router.NewUnaryServer[table.RetrieveRequest, table.RetrieveResponse](
			"/api/v1/table/retrieve",
		),
		TableDelete: router.NewUnaryServer[table.DeleteRequest, types.Nil](
			"/api/v1/table/delete",
		),
		TableDispatch: router.NewUnaryServer[table.DispatchRequest, types.Nil](
			"/api/v1/table/dispatch",
		),

		// LABEL
		LabelCreate: router.NewUnaryServer[label.CreateRequest, label.CreateResponse](
			"/api/v1/label/create",
		),
		LabelRetrieve: router.NewUnaryServer[label.RetrieveRequest, label.RetrieveResponse](
			"/api/v1/label/retrieve",
		),
		LabelDelete: router.NewUnaryServer[label.DeleteRequest, types.Nil](
			"/api/v1/label/delete",
		),
		LabelAdd: router.NewUnaryServer[label.AddRequest, types.Nil](
			"/api/v1/label/set",
		),
		LabelRemove: router.NewUnaryServer[label.RemoveRequest, types.Nil](
			"/api/v1/label/remove",
		),

		// RACK
		RackCreate: router.NewUnaryServer[rack.CreateRequest, rack.CreateResponse](
			"/api/v1/rack/create",
		),
		RackRetrieve: router.NewUnaryServer[rack.RetrieveRequest, rack.RetrieveResponse](
			"/api/v1/rack/retrieve",
		),
		RackDelete: router.NewUnaryServer[rack.DeleteRequest, types.Nil](
			"/api/v1/rack/delete",
		),

		// TASK
		TaskCreate: router.NewUnaryServer[task.CreateRequest, task.CreateResponse](
			"/api/v1/task/create",
		),
		TaskRetrieve: router.NewUnaryServer[task.RetrieveRequest, task.RetrieveResponse](
			"/api/v1/task/retrieve",
		),
		TaskDelete: router.NewUnaryServer[task.DeleteRequest, types.Nil](
			"/api/v1/task/delete",
		),
		TaskCopy: router.NewUnaryServer[task.CopyRequest, task.CopyResponse](
			"/api/v1/task/copy",
		),

		// DEVICE
		DeviceCreate: router.NewUnaryServer[device.CreateRequest, device.CreateResponse](
			"/api/v1/device/create",
		),
		DeviceRetrieve: router.NewUnaryServer[device.RetrieveRequest, device.RetrieveResponse](
			"/api/v1/device/retrieve",
		),
		DeviceDelete: router.NewUnaryServer[device.DeleteRequest, types.Nil](
			"/api/v1/device/delete",
		),

		// ACCESS
		AccessCreatePolicy: router.NewUnaryServer[access.CreatePolicyRequest, access.CreatePolicyResponse](
			"/api/v1/access/policy/create",
		),
		AccessDeletePolicy: router.NewUnaryServer[access.DeletePolicyRequest, types.Nil](
			"/api/v1/access/policy/delete",
		),
		AccessRetrievePolicy: router.NewUnaryServer[access.RetrievePolicyRequest, access.RetrievePolicyResponse](
			"/api/v1/access/policy/retrieve",
		),
		AccessCreateRole: router.NewUnaryServer[access.CreateRoleRequest, access.CreateRoleResponse](
			"/api/v1/access/role/create",
		),
		AccessDeleteRole: router.NewUnaryServer[access.DeleteRoleRequest, types.Nil](
			"/api/v1/access/role/delete",
		),
		AccessRetrieveRole: router.NewUnaryServer[access.RetrieveRoleRequest, access.RetrieveRoleResponse](
			"/api/v1/access/role/retrieve",
		),
		AccessAssignRole: router.NewUnaryServer[access.AssignRoleRequest, types.Nil](
			"/api/v1/access/role/assign",
		),
		AccessUnassignRole: router.NewUnaryServer[access.UnassignRoleRequest, types.Nil](
			"/api/v1/access/role/unassign",
		),

		// ARC
		ArcCreate: router.NewUnaryServer[arc.CreateRequest, arc.CreateResponse](
			"/api/v1/arc/create",
		),
		ArcDelete: router.NewUnaryServer[arc.DeleteRequest, types.Nil](
			"/api/v1/arc/delete",
		),
		ArcRetrieve: router.NewUnaryServer[arc.RetrieveRequest, arc.RetrieveResponse](
			"/api/v1/arc/retrieve",
		),
		ArcDispatch: router.NewUnaryServer[arc.DispatchRequest, types.Nil](
			"/api/v1/arc/dispatch",
		),
		ArcSetRack: router.NewUnaryServer[arc.SetRackRequest, arc.SetRackResponse](
			"/api/v1/arc/set-rack",
		),
		ArcLSP: router.NewStreamServer[arc.LSPMessage, arc.LSPMessage](
			"/api/v1/arc/lsp",
		),

		// STATUS
		StatusSet: router.NewUnaryServer[status.SetRequest, status.SetResponse](
			"/api/v1/status/set",
		),
		StatusRetrieve: router.NewUnaryServer[status.RetrieveRequest, status.RetrieveResponse](
			"/api/v1/status/retrieve",
		),
		StatusDelete: router.NewUnaryServer[status.DeleteRequest, types.Nil](
			"/api/v1/status/delete",
		),
		StatusSetByKeyOrName: router.NewUnaryServer[status.SetByKeyOrNameRequest, status.SetByKeyOrNameResponse](
			"/api/v1/status/set-by-key-or-name",
		),

		// VIEW
		ViewCreate: router.NewUnaryServer[view.CreateRequest, view.CreateResponse](
			"/api/v1/view/create",
		),
		ViewRetrieve: router.NewUnaryServer[view.RetrieveRequest, view.RetrieveResponse](
			"/api/v1/view/retrieve",
		),
		ViewDelete: router.NewUnaryServer[view.DeleteRequest, types.Nil](
			"/api/v1/view/delete",
		),

		// IMPORT/EXPORT
		ImExImport: router.NewUnaryServer[imex.ImportRequest, imex.ImportResponse](
			"/api/v1/imex/import",
			http.WithRequestDecoders(json.Codec),
		),
		ImExExport: router.NewUnaryServer[imex.ExportRequest, imex.ExportResponse](
			"/api/v1/imex/export",
			http.WithResponseEncoders(imex.JSONCodec),
		),
	})
}
