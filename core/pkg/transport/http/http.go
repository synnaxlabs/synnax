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
	"github.com/synnaxlabs/synnax/pkg/api/status"
	"github.com/synnaxlabs/synnax/pkg/api/table"
	"github.com/synnaxlabs/synnax/pkg/api/task"
	"github.com/synnaxlabs/synnax/pkg/api/user"
	"github.com/synnaxlabs/synnax/pkg/api/view"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/transport/http/framer"
	"github.com/synnaxlabs/x/encoding/json"
)

// Bind registers an HTTP endpoint for every API service onto router and binds the API
// layer's handlers and middleware to them. ch resolves channel keys for the frame
// codec.
func Bind(layer *api.Layer, router *http.Router, ch *distchannel.Service) {
	framerServerOption := framer.WithCodec(ch)
	layer.BindTo(api.Transport{
		// AUTH
		AuthLogin:          http.NewUnaryServer[auth.LoginRequest, auth.LoginResponse](router, "/api/v1/auth/login"),
		AuthChangePassword: http.NewUnaryServer[auth.ChangePasswordRequest, types.Nil](router, "/api/v1/auth/change-password"),

		// USER
		UserRename:         http.NewUnaryServer[user.RenameRequest, types.Nil](router, "/api/v1/user/rename"),
		UserChangeUsername: http.NewUnaryServer[user.ChangeUsernameRequest, types.Nil](router, "/api/v1/user/change-username"),
		UserCreate:         http.NewUnaryServer[user.CreateRequest, user.CreateResponse](router, "/api/v1/user/create"),
		UserDelete:         http.NewUnaryServer[user.DeleteRequest, types.Nil](router, "/api/v1/user/delete"),
		UserRetrieve:       http.NewUnaryServer[user.RetrieveRequest, user.RetrieveResponse](router, "/api/v1/user/retrieve"),

		// CHANNEL
		ChannelCreate:        http.NewUnaryServer[channel.CreateRequest, channel.CreateResponse](router, "/api/v1/channel/create"),
		ChannelRetrieve:      http.NewUnaryServer[channel.RetrieveRequest, channel.RetrieveResponse](router, "/api/v1/channel/retrieve"),
		ChannelDelete:        http.NewUnaryServer[channel.DeleteRequest, types.Nil](router, "/api/v1/channel/delete"),
		ChannelRename:        http.NewUnaryServer[channel.RenameRequest, types.Nil](router, "/api/v1/channel/rename"),
		ChannelRetrieveGroup: http.NewUnaryServer[channel.RetrieveGroupRequest, channel.RetrieveGroupResponse](router, "/api/v1/channel/retrieve-group"),

		// CONNECTIVITY
		ConnectivityCheck: http.NewUnaryServer[types.Nil, connectivity.CheckResponse](router, "/api/v1/connectivity/check"),

		// FRAME
		FrameWriter:   http.NewStreamServer[framer.WriterRequest, framer.WriterResponse](router, "/api/v1/frame/write", framerServerOption),
		FrameIterator: http.NewStreamServer[framer.IteratorRequest, framer.IteratorResponse](router, "/api/v1/frame/iterate", framerServerOption),
		FrameStreamer: http.NewStreamServer[framer.StreamerRequest, framer.StreamerResponse](router, "/api/v1/frame/stream", framerServerOption),
		FrameDelete:   http.NewUnaryServer[framer.DeleteRequest, types.Nil](router, "/api/v1/frame/delete"),

		// ONTOLOGY
		OntologyRetrieve:       http.NewUnaryServer[ontology.RetrieveRequest, ontology.RetrieveResponse](router, "/api/v1/ontology/retrieve"),
		OntologyAddChildren:    http.NewUnaryServer[ontology.AddChildrenRequest, types.Nil](router, "/api/v1/ontology/add-children"),
		OntologyRemoveChildren: http.NewUnaryServer[ontology.RemoveChildrenRequest, types.Nil](router, "/api/v1/ontology/remove-children"),
		OntologyMoveChildren:   http.NewUnaryServer[ontology.MoveChildrenRequest, types.Nil](router, "/api/v1/ontology/move-children"),

		// GROUP
		GroupCreate: http.NewUnaryServer[group.CreateRequest, group.CreateResponse](router, "/api/v1/ontology/create-group"),
		GroupDelete: http.NewUnaryServer[group.DeleteRequest, types.Nil](router, "/api/v1/ontology/delete-group"),
		GroupRename: http.NewUnaryServer[group.RenameRequest, types.Nil](router, "/api/v1/ontology/rename-group"),

		// RANGE
		RangeRetrieve: http.NewUnaryServer[ranger.RetrieveRequest, ranger.RetrieveResponse](router, "/api/v1/range/retrieve"),
		RangeCreate:   http.NewUnaryServer[ranger.CreateRequest, ranger.CreateResponse](router, "/api/v1/range/create"),
		RangeDelete:   http.NewUnaryServer[ranger.DeleteRequest, types.Nil](router, "/api/v1/range/delete"),
		RangeRename:   http.NewUnaryServer[ranger.RenameRequest, types.Nil](router, "/api/v1/range/rename"),

		// KV
		KVGet:    http.NewUnaryServer[kv.GetRequest, kv.GetResponse](router, "/api/v1/range/kv/get"),
		KVSet:    http.NewUnaryServer[kv.SetRequest, types.Nil](router, "/api/v1/range/kv/set"),
		KVDelete: http.NewUnaryServer[kv.DeleteRequest, types.Nil](router, "/api/v1/range/kv/delete"),

		// ALIAS
		AliasSet:      http.NewUnaryServer[alias.SetRequest, types.Nil](router, "/api/v1/range/alias/set"),
		AliasResolve:  http.NewUnaryServer[alias.ResolveRequest, alias.ResolveResponse](router, "/api/v1/range/alias/resolve"),
		AliasRetrieve: http.NewUnaryServer[alias.RetrieveRequest, alias.RetrieveResponse](router, "/api/v1/range/alias/retrieve"),
		AliasList:     http.NewUnaryServer[alias.ListRequest, alias.ListResponse](router, "/api/v1/range/alias/list"),
		AliasDelete:   http.NewUnaryServer[alias.DeleteRequest, types.Nil](router, "/api/v1/range/alias/delete"),

		// PROJECT
		ProjectCreate:    http.NewUnaryServer[project.CreateRequest, project.CreateResponse](router, "/api/v1/project/create"),
		ProjectRetrieve:  http.NewUnaryServer[project.RetrieveRequest, project.RetrieveResponse](router, "/api/v1/project/retrieve"),
		ProjectDelete:    http.NewUnaryServer[project.DeleteRequest, types.Nil](router, "/api/v1/project/delete"),
		ProjectRename:    http.NewUnaryServer[project.RenameRequest, types.Nil](router, "/api/v1/project/rename"),
		ProjectSetLayout: http.NewUnaryServer[project.SetLayoutRequest, types.Nil](router, "/api/v1/project/set-layout"),

		// SCHEMATIC
		SchematicCreate:   http.NewUnaryServer[schematic.CreateRequest, schematic.CreateResponse](router, "/api/v1/schematic/create"),
		SchematicRetrieve: http.NewUnaryServer[schematic.RetrieveRequest, schematic.RetrieveResponse](router, "/api/v1/schematic/retrieve"),
		SchematicDelete:   http.NewUnaryServer[schematic.DeleteRequest, types.Nil](router, "/api/v1/schematic/delete"),
		SchematicDispatch: http.NewUnaryServer[schematic.DispatchRequest, types.Nil](router, "/api/v1/schematic/dispatch"),
		SchematicCopy:     http.NewUnaryServer[schematic.CopyRequest, schematic.CopyResponse](router, "/api/v1/schematic/copy"),

		// SCHEMATIC SYMBOL
		SchematicCreateSymbol:        http.NewUnaryServer[schematic.CreateSymbolRequest, schematic.CreateSymbolResponse](router, "/api/v1/schematic/symbol/create"),
		SchematicRetrieveSymbol:      http.NewUnaryServer[schematic.RetrieveSymbolRequest, schematic.RetrieveSymbolResponse](router, "/api/v1/schematic/symbol/retrieve"),
		SchematicDeleteSymbol:        http.NewUnaryServer[schematic.DeleteSymbolRequest, types.Nil](router, "/api/v1/schematic/symbol/delete"),
		SchematicRenameSymbol:        http.NewUnaryServer[schematic.RenameSymbolRequest, types.Nil](router, "/api/v1/schematic/symbol/rename"),
		SchematicRetrieveSymbolGroup: http.NewUnaryServer[schematic.RetrieveSymbolGroupRequest, schematic.RetrieveSymbolGroupResponse](router, "/api/v1/schematic/symbol/retrieve-group"),

		// LINE PLOT
		LinePlotCreate:   http.NewUnaryServer[lineplot.CreateRequest, lineplot.CreateResponse](router, "/api/v1/lineplot/create"),
		LinePlotRetrieve: http.NewUnaryServer[lineplot.RetrieveRequest, lineplot.RetrieveResponse](router, "/api/v1/lineplot/retrieve"),
		LinePlotDelete:   http.NewUnaryServer[lineplot.DeleteRequest, types.Nil](router, "/api/v1/lineplot/delete"),
		LinePlotDispatch: http.NewUnaryServer[lineplot.DispatchRequest, types.Nil](router, "/api/v1/lineplot/dispatch"),

		// PANEL
		PanelCreate:   http.NewUnaryServer[panel.CreateRequest, panel.CreateResponse](router, "/api/v1/panel/create"),
		PanelRetrieve: http.NewUnaryServer[panel.RetrieveRequest, panel.RetrieveResponse](router, "/api/v1/panel/retrieve"),
		PanelDelete:   http.NewUnaryServer[panel.DeleteRequest, types.Nil](router, "/api/v1/panel/delete"),
		PanelDispatch: http.NewUnaryServer[panel.DispatchRequest, types.Nil](router, "/api/v1/panel/dispatch"),

		// LOG
		LogCreate:   http.NewUnaryServer[log.CreateRequest, log.CreateResponse](router, "/api/v1/log/create"),
		LogRetrieve: http.NewUnaryServer[log.RetrieveRequest, log.RetrieveResponse](router, "/api/v1/log/retrieve"),
		LogDelete:   http.NewUnaryServer[log.DeleteRequest, types.Nil](router, "/api/v1/log/delete"),
		LogDispatch: http.NewUnaryServer[log.DispatchRequest, types.Nil](router, "/api/v1/log/dispatch"),

		// TABLE
		TableCreate:   http.NewUnaryServer[table.CreateRequest, table.CreateResponse](router, "/api/v1/table/create"),
		TableRetrieve: http.NewUnaryServer[table.RetrieveRequest, table.RetrieveResponse](router, "/api/v1/table/retrieve"),
		TableDelete:   http.NewUnaryServer[table.DeleteRequest, types.Nil](router, "/api/v1/table/delete"),
		TableDispatch: http.NewUnaryServer[table.DispatchRequest, types.Nil](router, "/api/v1/table/dispatch"),

		// LABEL
		LabelCreate:   http.NewUnaryServer[label.CreateRequest, label.CreateResponse](router, "/api/v1/label/create"),
		LabelRetrieve: http.NewUnaryServer[label.RetrieveRequest, label.RetrieveResponse](router, "/api/v1/label/retrieve"),
		LabelDelete:   http.NewUnaryServer[label.DeleteRequest, types.Nil](router, "/api/v1/label/delete"),
		LabelAdd:      http.NewUnaryServer[label.AddRequest, types.Nil](router, "/api/v1/label/set"),
		LabelRemove:   http.NewUnaryServer[label.RemoveRequest, types.Nil](router, "/api/v1/label/remove"),

		// RACK
		RackCreate:   http.NewUnaryServer[rack.CreateRequest, rack.CreateResponse](router, "/api/v1/rack/create"),
		RackRetrieve: http.NewUnaryServer[rack.RetrieveRequest, rack.RetrieveResponse](router, "/api/v1/rack/retrieve"),
		RackDelete:   http.NewUnaryServer[rack.DeleteRequest, types.Nil](router, "/api/v1/rack/delete"),

		// TASK
		TaskCreate:   http.NewUnaryServer[task.CreateRequest, task.CreateResponse](router, "/api/v1/task/create"),
		TaskRetrieve: http.NewUnaryServer[task.RetrieveRequest, task.RetrieveResponse](router, "/api/v1/task/retrieve"),
		TaskDelete:   http.NewUnaryServer[task.DeleteRequest, types.Nil](router, "/api/v1/task/delete"),
		TaskCopy:     http.NewUnaryServer[task.CopyRequest, task.CopyResponse](router, "/api/v1/task/copy"),

		// DEVICE
		DeviceCreate:   http.NewUnaryServer[device.CreateRequest, device.CreateResponse](router, "/api/v1/device/create"),
		DeviceRetrieve: http.NewUnaryServer[device.RetrieveRequest, device.RetrieveResponse](router, "/api/v1/device/retrieve"),
		DeviceDelete:   http.NewUnaryServer[device.DeleteRequest, types.Nil](router, "/api/v1/device/delete"),

		// ACCESS
		AccessCreatePolicy:   http.NewUnaryServer[access.CreatePolicyRequest, access.CreatePolicyResponse](router, "/api/v1/access/policy/create"),
		AccessDeletePolicy:   http.NewUnaryServer[access.DeletePolicyRequest, types.Nil](router, "/api/v1/access/policy/delete"),
		AccessRetrievePolicy: http.NewUnaryServer[access.RetrievePolicyRequest, access.RetrievePolicyResponse](router, "/api/v1/access/policy/retrieve"),
		AccessCreateRole:     http.NewUnaryServer[access.CreateRoleRequest, access.CreateRoleResponse](router, "/api/v1/access/role/create"),
		AccessDeleteRole:     http.NewUnaryServer[access.DeleteRoleRequest, types.Nil](router, "/api/v1/access/role/delete"),
		AccessRetrieveRole:   http.NewUnaryServer[access.RetrieveRoleRequest, access.RetrieveRoleResponse](router, "/api/v1/access/role/retrieve"),
		AccessAssignRole:     http.NewUnaryServer[access.AssignRoleRequest, types.Nil](router, "/api/v1/access/role/assign"),
		AccessUnassignRole:   http.NewUnaryServer[access.UnassignRoleRequest, types.Nil](router, "/api/v1/access/role/unassign"),

		// ARC
		ArcCreate:   http.NewUnaryServer[arc.CreateRequest, arc.CreateResponse](router, "/api/v1/arc/create"),
		ArcDelete:   http.NewUnaryServer[arc.DeleteRequest, types.Nil](router, "/api/v1/arc/delete"),
		ArcRetrieve: http.NewUnaryServer[arc.RetrieveRequest, arc.RetrieveResponse](router, "/api/v1/arc/retrieve"),
		ArcLSP:      http.NewStreamServer[arc.LSPMessage, arc.LSPMessage](router, "/api/v1/arc/lsp"),

		// STATUS
		StatusSet:            http.NewUnaryServer[status.SetRequest, status.SetResponse](router, "/api/v1/status/set"),
		StatusRetrieve:       http.NewUnaryServer[status.RetrieveRequest, status.RetrieveResponse](router, "/api/v1/status/retrieve"),
		StatusDelete:         http.NewUnaryServer[status.DeleteRequest, types.Nil](router, "/api/v1/status/delete"),
		StatusSetByKeyOrName: http.NewUnaryServer[status.SetByKeyOrNameRequest, status.SetByKeyOrNameResponse](router, "/api/v1/status/set-by-key-or-name"),

		// VIEW
		ViewCreate:   http.NewUnaryServer[view.CreateRequest, view.CreateResponse](router, "/api/v1/view/create"),
		ViewRetrieve: http.NewUnaryServer[view.RetrieveRequest, view.RetrieveResponse](router, "/api/v1/view/retrieve"),
		ViewDelete:   http.NewUnaryServer[view.DeleteRequest, types.Nil](router, "/api/v1/view/delete"),

		// IMPORT/EXPORT
		ImExImport: http.NewUnaryServer[imex.ImportRequest, imex.ImportResponse](router, "/api/v1/imex/import", http.WithRequestDecoders(json.Codec)),
		ImExExport: http.NewUnaryServer[imex.ExportRequest, imex.ExportResponse](router, "/api/v1/imex/export", http.WithResponseEncoders(json.Codec)),
	})
}
