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

	"github.com/synnaxlabs/freighter"
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
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/encoding/msgpack"
)

// newUnaryServer wraps fhttp.NewUnaryServer with the JSON and MessagePack codecs that
// the Synnax HTTP API negotiates with its clients.
func newUnaryServer[RQ, RS freighter.Payload](
	router *fhttp.Router,
	path string,
) freighter.UnaryServer[RQ, RS] {
	return fhttp.NewUnaryServer[RQ, RS](
		router,
		path,
		fhttp.WithRequestDecoders(json.Codec, msgpack.Codec),
		fhttp.WithResponseEncoders(json.Codec, msgpack.Codec),
	)
}

// streamServer wraps fhttp.NewStreamServer with the JSON and MessagePack codecs that
// the Synnax HTTP API negotiates with its clients. Additional stream codec options
// (e.g. the framer codec) can be passed alongside to register stateful codecs.
func streamServer[RQ, RS freighter.Payload](
	router *fhttp.Router,
	path string,
	opts ...fhttp.StreamServerOption,
) freighter.StreamServer[RQ, RS] {
	opts = append(
		[]fhttp.StreamServerOption{
			fhttp.WithCodec(json.Codec),
			fhttp.WithCodec(msgpack.Codec),
		},
		opts...,
	)
	return fhttp.NewStreamServer[RQ, RS](router, path, opts...)
}

func NewTransport(router *fhttp.Router, ch *distchannel.Service) api.Transport {
	framerCodec := httpframer.WithCodec(ch)
	return api.Transport{
		// AUTH
		AuthLogin:          newUnaryServer[auth.LoginRequest, auth.LoginResponse](router, "/api/v1/auth/login"),
		AuthChangePassword: newUnaryServer[auth.ChangePasswordRequest, types.Nil](router, "/api/v1/auth/change-password"),

		// USER
		UserRename:         newUnaryServer[user.RenameRequest, types.Nil](router, "/api/v1/user/rename"),
		UserChangeUsername: newUnaryServer[user.ChangeUsernameRequest, types.Nil](router, "/api/v1/user/change-username"),
		UserCreate:         newUnaryServer[user.CreateRequest, user.CreateResponse](router, "/api/v1/user/create"),
		UserDelete:         newUnaryServer[user.DeleteRequest, types.Nil](router, "/api/v1/user/delete"),
		UserRetrieve:       newUnaryServer[user.RetrieveRequest, user.RetrieveResponse](router, "/api/v1/user/retrieve"),

		// CHANNEL
		ChannelCreate:        newUnaryServer[channel.CreateRequest, channel.CreateResponse](router, "/api/v1/channel/create"),
		ChannelRetrieve:      newUnaryServer[channel.RetrieveRequest, channel.RetrieveResponse](router, "/api/v1/channel/retrieve"),
		ChannelDelete:        newUnaryServer[channel.DeleteRequest, types.Nil](router, "/api/v1/channel/delete"),
		ChannelRename:        newUnaryServer[channel.RenameRequest, types.Nil](router, "/api/v1/channel/rename"),
		ChannelRetrieveGroup: newUnaryServer[channel.RetrieveGroupRequest, channel.RetrieveGroupResponse](router, "/api/v1/channel/retrieve-group"),

		// CONNECTIVITY
		ConnectivityCheck: newUnaryServer[types.Nil, connectivity.CheckResponse](router, "/api/v1/connectivity/check"),

		// FRAME
		FrameWriter:   streamServer[framer.WriterRequest, framer.WriterResponse](router, "/api/v1/frame/write", framerCodec),
		FrameIterator: streamServer[framer.IteratorRequest, framer.IteratorResponse](router, "/api/v1/frame/iterate"),
		FrameStreamer: streamServer[framer.StreamerRequest, framer.StreamerResponse](router, "/api/v1/frame/stream", framerCodec),
		FrameDelete:   newUnaryServer[framer.DeleteRequest, types.Nil](router, "/api/v1/frame/delete"),

		// ONTOLOGY
		OntologyRetrieve:       newUnaryServer[ontology.RetrieveRequest, ontology.RetrieveResponse](router, "/api/v1/ontology/retrieve"),
		OntologyAddChildren:    newUnaryServer[ontology.AddChildrenRequest, types.Nil](router, "/api/v1/ontology/add-children"),
		OntologyRemoveChildren: newUnaryServer[ontology.RemoveChildrenRequest, types.Nil](router, "/api/v1/ontology/remove-children"),
		OntologyMoveChildren:   newUnaryServer[ontology.MoveChildrenRequest, types.Nil](router, "/api/v1/ontology/move-children"),

		// GROUP
		GroupCreate: newUnaryServer[group.CreateRequest, group.CreateResponse](router, "/api/v1/ontology/create-group"),
		GroupDelete: newUnaryServer[group.DeleteRequest, types.Nil](router, "/api/v1/ontology/delete-group"),
		GroupRename: newUnaryServer[group.RenameRequest, types.Nil](router, "/api/v1/ontology/rename-group"),

		// RANGE
		RangeRetrieve: newUnaryServer[ranger.RetrieveRequest, ranger.RetrieveResponse](router, "/api/v1/range/retrieve"),
		RangeCreate:   newUnaryServer[ranger.CreateRequest, ranger.CreateResponse](router, "/api/v1/range/create"),
		RangeDelete:   newUnaryServer[ranger.DeleteRequest, types.Nil](router, "/api/v1/range/delete"),
		RangeRename:   newUnaryServer[ranger.RenameRequest, types.Nil](router, "/api/v1/range/rename"),

		// KV
		KVGet:    newUnaryServer[kv.GetRequest, kv.GetResponse](router, "/api/v1/range/kv/get"),
		KVSet:    newUnaryServer[kv.SetRequest, types.Nil](router, "/api/v1/range/kv/set"),
		KVDelete: newUnaryServer[kv.DeleteRequest, types.Nil](router, "/api/v1/range/kv/delete"),

		// ALIAS
		AliasSet:      newUnaryServer[alias.SetRequest, types.Nil](router, "/api/v1/range/alias/set"),
		AliasResolve:  newUnaryServer[alias.ResolveRequest, alias.ResolveResponse](router, "/api/v1/range/alias/resolve"),
		AliasRetrieve: newUnaryServer[alias.RetrieveRequest, alias.RetrieveResponse](router, "/api/v1/range/alias/retrieve"),
		AliasList:     newUnaryServer[alias.ListRequest, alias.ListResponse](router, "/api/v1/range/alias/list"),
		AliasDelete:   newUnaryServer[alias.DeleteRequest, types.Nil](router, "/api/v1/range/alias/delete"),

		// WORKSPACE
		WorkspaceCreate:    newUnaryServer[workspace.CreateRequest, workspace.CreateResponse](router, "/api/v1/workspace/create"),
		WorkspaceRetrieve:  newUnaryServer[workspace.RetrieveRequest, workspace.RetrieveResponse](router, "/api/v1/workspace/retrieve"),
		WorkspaceDelete:    newUnaryServer[workspace.DeleteRequest, types.Nil](router, "/api/v1/workspace/delete"),
		WorkspaceRename:    newUnaryServer[workspace.RenameRequest, types.Nil](router, "/api/v1/workspace/rename"),
		WorkspaceSetLayout: newUnaryServer[workspace.SetLayoutRequest, types.Nil](router, "/api/v1/workspace/set-layout"),

		// SCHEMATIC
		SchematicCreate:   newUnaryServer[schematic.CreateRequest, schematic.CreateResponse](router, "/api/v1/schematic/create"),
		SchematicRetrieve: newUnaryServer[schematic.RetrieveRequest, schematic.RetrieveResponse](router, "/api/v1/schematic/retrieve"),
		SchematicDelete:   newUnaryServer[schematic.DeleteRequest, types.Nil](router, "/api/v1/schematic/delete"),
		SchematicRename:   newUnaryServer[schematic.RenameRequest, types.Nil](router, "/api/v1/schematic/rename"),
		SchematicSetData:  newUnaryServer[schematic.SetDataRequest, types.Nil](router, "/api/v1/schematic/set-data"),
		SchematicCopy:     newUnaryServer[schematic.CopyRequest, schematic.CopyResponse](router, "/api/v1/schematic/copy"),

		// SCHEMATIC SYMBOL
		SchematicCreateSymbol:        newUnaryServer[schematic.CreateSymbolRequest, schematic.CreateSymbolResponse](router, "/api/v1/schematic/symbol/create"),
		SchematicRetrieveSymbol:      newUnaryServer[schematic.RetrieveSymbolRequest, schematic.RetrieveSymbolResponse](router, "/api/v1/schematic/symbol/retrieve"),
		SchematicDeleteSymbol:        newUnaryServer[schematic.DeleteSymbolRequest, types.Nil](router, "/api/v1/schematic/symbol/delete"),
		SchematicRenameSymbol:        newUnaryServer[schematic.RenameSymbolRequest, types.Nil](router, "/api/v1/schematic/symbol/rename"),
		SchematicRetrieveSymbolGroup: newUnaryServer[schematic.RetrieveSymbolGroupRequest, schematic.RetrieveSymbolGroupResponse](router, "/api/v1/schematic/symbol/retrieve-group"),

		// LINE PLOT
		LinePlotCreate:   newUnaryServer[lineplot.CreateRequest, lineplot.CreateResponse](router, "/api/v1/lineplot/create"),
		LinePlotRetrieve: newUnaryServer[lineplot.RetrieveRequest, lineplot.RetrieveResponse](router, "/api/v1/lineplot/retrieve"),
		LinePlotDelete:   newUnaryServer[lineplot.DeleteRequest, types.Nil](router, "/api/v1/lineplot/delete"),
		LinePlotRename:   newUnaryServer[lineplot.RenameRequest, types.Nil](router, "/api/v1/lineplot/rename"),
		LinePlotSetData:  newUnaryServer[lineplot.SetDataRequest, types.Nil](router, "/api/v1/lineplot/set-data"),

		// LOG
		LogCreate:   newUnaryServer[log.CreateRequest, log.CreateResponse](router, "/api/v1/log/create"),
		LogRetrieve: newUnaryServer[log.RetrieveRequest, log.RetrieveResponse](router, "/api/v1/log/retrieve"),
		LogDelete:   newUnaryServer[log.DeleteRequest, types.Nil](router, "/api/v1/log/delete"),
		LogRename:   newUnaryServer[log.RenameRequest, types.Nil](router, "/api/v1/log/rename"),
		LogSetData:  newUnaryServer[log.SetDataRequest, types.Nil](router, "/api/v1/log/set-data"),

		// TABLE
		TableCreate:   newUnaryServer[table.CreateRequest, table.CreateResponse](router, "/api/v1/table/create"),
		TableRetrieve: newUnaryServer[table.RetrieveRequest, table.RetrieveResponse](router, "/api/v1/table/retrieve"),
		TableDelete:   newUnaryServer[table.DeleteRequest, types.Nil](router, "/api/v1/table/delete"),
		TableRename:   newUnaryServer[table.RenameRequest, types.Nil](router, "/api/v1/table/rename"),
		TableSetData:  newUnaryServer[table.SetDataRequest, types.Nil](router, "/api/v1/table/set-data"),

		// LABEL
		LabelCreate:   newUnaryServer[label.CreateRequest, label.CreateResponse](router, "/api/v1/label/create"),
		LabelRetrieve: newUnaryServer[label.RetrieveRequest, label.RetrieveResponse](router, "/api/v1/label/retrieve"),
		LabelDelete:   newUnaryServer[label.DeleteRequest, types.Nil](router, "/api/v1/label/delete"),
		LabelAdd:      newUnaryServer[label.AddRequest, types.Nil](router, "/api/v1/label/set"),
		LabelRemove:   newUnaryServer[label.RemoveRequest, types.Nil](router, "/api/v1/label/remove"),

		// RACK
		RackCreate:   newUnaryServer[rack.CreateRequest, rack.CreateResponse](router, "/api/v1/rack/create"),
		RackRetrieve: newUnaryServer[rack.RetrieveRequest, rack.RetrieveResponse](router, "/api/v1/rack/retrieve"),
		RackDelete:   newUnaryServer[rack.DeleteRequest, types.Nil](router, "/api/v1/rack/delete"),

		// TASK
		TaskCreate:   newUnaryServer[task.CreateRequest, task.CreateResponse](router, "/api/v1/task/create"),
		TaskRetrieve: newUnaryServer[task.RetrieveRequest, task.RetrieveResponse](router, "/api/v1/task/retrieve"),
		TaskDelete:   newUnaryServer[task.DeleteRequest, types.Nil](router, "/api/v1/task/delete"),
		TaskCopy:     newUnaryServer[task.CopyRequest, task.CopyResponse](router, "/api/v1/task/copy"),

		// DEVICE
		DeviceCreate:   newUnaryServer[device.CreateRequest, device.CreateResponse](router, "/api/v1/device/create"),
		DeviceRetrieve: newUnaryServer[device.RetrieveRequest, device.RetrieveResponse](router, "/api/v1/device/retrieve"),
		DeviceDelete:   newUnaryServer[device.DeleteRequest, types.Nil](router, "/api/v1/device/delete"),

		// ACCESS
		AccessCreatePolicy:   newUnaryServer[access.CreatePolicyRequest, access.CreatePolicyResponse](router, "/api/v1/access/policy/create"),
		AccessDeletePolicy:   newUnaryServer[access.DeletePolicyRequest, types.Nil](router, "/api/v1/access/policy/delete"),
		AccessRetrievePolicy: newUnaryServer[access.RetrievePolicyRequest, access.RetrievePolicyResponse](router, "/api/v1/access/policy/retrieve"),
		AccessCreateRole:     newUnaryServer[access.CreateRoleRequest, access.CreateRoleResponse](router, "/api/v1/access/role/create"),
		AccessDeleteRole:     newUnaryServer[access.DeleteRoleRequest, types.Nil](router, "/api/v1/access/role/delete"),
		AccessRetrieveRole:   newUnaryServer[access.RetrieveRoleRequest, access.RetrieveRoleResponse](router, "/api/v1/access/role/retrieve"),
		AccessAssignRole:     newUnaryServer[access.AssignRoleRequest, types.Nil](router, "/api/v1/access/role/assign"),
		AccessUnassignRole:   newUnaryServer[access.UnassignRoleRequest, types.Nil](router, "/api/v1/access/role/unassign"),

		// ARC
		ArcCreate:   newUnaryServer[arc.CreateRequest, arc.CreateResponse](router, "/api/v1/arc/create"),
		ArcDelete:   newUnaryServer[arc.DeleteRequest, types.Nil](router, "/api/v1/arc/delete"),
		ArcRetrieve: newUnaryServer[arc.RetrieveRequest, arc.RetrieveResponse](router, "/api/v1/arc/retrieve"),
		ArcLSP:      streamServer[arc.LSPMessage, arc.LSPMessage](router, "/api/v1/arc/lsp"),

		// STATUS
		StatusSet:      newUnaryServer[status.SetRequest, status.SetResponse](router, "/api/v1/status/set"),
		StatusRetrieve: newUnaryServer[status.RetrieveRequest, status.RetrieveResponse](router, "/api/v1/status/retrieve"),
		StatusDelete:   newUnaryServer[status.DeleteRequest, types.Nil](router, "/api/v1/status/delete"),

		// VIEW
		ViewCreate:   newUnaryServer[view.CreateRequest, view.CreateResponse](router, "/api/v1/view/create"),
		ViewRetrieve: newUnaryServer[view.RetrieveRequest, view.RetrieveResponse](router, "/api/v1/view/retrieve"),
		ViewDelete:   newUnaryServer[view.DeleteRequest, types.Nil](router, "/api/v1/view/delete"),
	}
}
