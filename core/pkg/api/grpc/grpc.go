// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package grpc

import (
	"go/types"

	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/freighter/noop"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/access"
	"github.com/synnaxlabs/synnax/pkg/api/arc"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/channel"
	"github.com/synnaxlabs/synnax/pkg/api/group"
	arcgrpc "github.com/synnaxlabs/synnax/pkg/api/grpc/arc"
	authgrpc "github.com/synnaxlabs/synnax/pkg/api/grpc/auth"
	channelgrpc "github.com/synnaxlabs/synnax/pkg/api/grpc/channel"
	connectivitygrpc "github.com/synnaxlabs/synnax/pkg/api/grpc/connectivity"
	devicegrpc "github.com/synnaxlabs/synnax/pkg/api/grpc/device"
	framergrpc "github.com/synnaxlabs/synnax/pkg/api/grpc/framer"
	rackgrpc "github.com/synnaxlabs/synnax/pkg/api/grpc/rack"
	rangergrpc "github.com/synnaxlabs/synnax/pkg/api/grpc/ranger"
	aliasgrpc "github.com/synnaxlabs/synnax/pkg/api/grpc/ranger/alias"
	kvgrpc "github.com/synnaxlabs/synnax/pkg/api/grpc/ranger/kv"
	statusgrpc "github.com/synnaxlabs/synnax/pkg/api/grpc/status"
	taskgrpc "github.com/synnaxlabs/synnax/pkg/api/grpc/task"
	viewgrpc "github.com/synnaxlabs/synnax/pkg/api/grpc/view"
	"github.com/synnaxlabs/synnax/pkg/api/imex"
	"github.com/synnaxlabs/synnax/pkg/api/label"
	"github.com/synnaxlabs/synnax/pkg/api/lineplot"
	"github.com/synnaxlabs/synnax/pkg/api/log"
	"github.com/synnaxlabs/synnax/pkg/api/ontology"
	"github.com/synnaxlabs/synnax/pkg/api/ranger"
	"github.com/synnaxlabs/synnax/pkg/api/ranger/alias"
	"github.com/synnaxlabs/synnax/pkg/api/schematic"
	"github.com/synnaxlabs/synnax/pkg/api/table"
	"github.com/synnaxlabs/synnax/pkg/api/user"
	"github.com/synnaxlabs/synnax/pkg/api/workspace"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
)

func NewTransport(channelSvc *distchannel.Service) (api.Transport, []grpc.BindableTransport) {
	var a api.Transport
	transports := grpc.CompoundBindableTransport{
		channelgrpc.New(&a),
		framergrpc.New(&a, channelSvc),
		connectivitygrpc.New(&a),
		authgrpc.New(&a),
		rangergrpc.New(&a),
		kvgrpc.New(&a),
		aliasgrpc.New(&a),
		rackgrpc.New(&a),
		taskgrpc.New(&a),
		devicegrpc.New(&a),
		statusgrpc.New(&a),
		arcgrpc.New(&a),
		viewgrpc.New(&a),
	}

	// AUTH
	a.AuthChangePassword = noop.UnaryServer[auth.ChangePasswordRequest, types.Nil]{}

	// CHANNEL
	a.ChannelRename = noop.UnaryServer[channel.RenameRequest, types.Nil]{}
	a.ChannelRetrieveGroup = noop.UnaryServer[channel.RetrieveGroupRequest, channel.RetrieveGroupResponse]{}

	// USER
	a.UserRename = noop.UnaryServer[user.RenameRequest, types.Nil]{}
	a.UserChangeUsername = noop.UnaryServer[user.ChangeUsernameRequest, types.Nil]{}
	a.UserCreate = noop.UnaryServer[user.CreateRequest, user.CreateResponse]{}
	a.UserDelete = noop.UnaryServer[user.DeleteRequest, types.Nil]{}
	a.UserRetrieve = noop.UnaryServer[user.RetrieveRequest, user.RetrieveResponse]{}

	// RANGE
	a.RangeRename = noop.UnaryServer[ranger.RenameRequest, types.Nil]{}
	a.AliasRetrieve = noop.UnaryServer[alias.RetrieveRequest, alias.RetrieveResponse]{}

	// ONTOLOGY
	a.OntologyRetrieve = noop.UnaryServer[ontology.RetrieveRequest, ontology.RetrieveResponse]{}
	a.OntologyAddChildren = noop.UnaryServer[ontology.AddChildrenRequest, types.Nil]{}
	a.OntologyRemoveChildren = noop.UnaryServer[ontology.RemoveChildrenRequest, types.Nil]{}
	a.OntologyMoveChildren = noop.UnaryServer[ontology.MoveChildrenRequest, types.Nil]{}

	// GROUP
	a.GroupCreate = noop.UnaryServer[group.CreateRequest, group.CreateResponse]{}
	a.GroupDelete = noop.UnaryServer[group.DeleteRequest, types.Nil]{}
	a.GroupRename = noop.UnaryServer[group.RenameRequest, types.Nil]{}

	// WORKSPACE
	a.WorkspaceCreate = noop.UnaryServer[workspace.CreateRequest, workspace.CreateResponse]{}
	a.WorkspaceRetrieve = noop.UnaryServer[workspace.RetrieveRequest, workspace.RetrieveResponse]{}
	a.WorkspaceDelete = noop.UnaryServer[workspace.DeleteRequest, types.Nil]{}
	a.WorkspaceRename = noop.UnaryServer[workspace.RenameRequest, types.Nil]{}
	a.WorkspaceSetLayout = noop.UnaryServer[workspace.SetLayoutRequest, types.Nil]{}

	// SCHEMATIC
	a.SchematicCreate = noop.UnaryServer[schematic.CreateRequest, schematic.CreateResponse]{}
	a.SchematicDelete = noop.UnaryServer[schematic.DeleteRequest, types.Nil]{}
	a.SchematicRetrieve = noop.UnaryServer[schematic.RetrieveRequest, schematic.RetrieveResponse]{}
	a.SchematicRename = noop.UnaryServer[schematic.RenameRequest, types.Nil]{}
	a.SchematicSetData = noop.UnaryServer[schematic.SetDataRequest, types.Nil]{}
	a.SchematicDispatch = noop.UnaryServer[schematic.DispatchRequest, types.Nil]{}
	a.SchematicCopy = noop.UnaryServer[schematic.CopyRequest, schematic.CopyResponse]{}

	// SCHEMATIC SYMBOL
	a.SchematicCreateSymbol = noop.UnaryServer[schematic.CreateSymbolRequest, schematic.CreateSymbolResponse]{}
	a.SchematicRetrieveSymbol = noop.UnaryServer[schematic.RetrieveSymbolRequest, schematic.RetrieveSymbolResponse]{}
	a.SchematicDeleteSymbol = noop.UnaryServer[schematic.DeleteSymbolRequest, types.Nil]{}
	a.SchematicRenameSymbol = noop.UnaryServer[schematic.RenameSymbolRequest, types.Nil]{}
	a.SchematicRetrieveSymbolGroup = noop.UnaryServer[schematic.RetrieveSymbolGroupRequest, schematic.RetrieveSymbolGroupResponse]{}

	// LINE PLOT
	a.LinePlotCreate = noop.UnaryServer[lineplot.CreateRequest, lineplot.CreateResponse]{}
	a.LinePlotRetrieve = noop.UnaryServer[lineplot.RetrieveRequest, lineplot.RetrieveResponse]{}
	a.LinePlotDelete = noop.UnaryServer[lineplot.DeleteRequest, types.Nil]{}
	a.LinePlotRename = noop.UnaryServer[lineplot.RenameRequest, types.Nil]{}
	a.LinePlotSetData = noop.UnaryServer[lineplot.SetDataRequest, types.Nil]{}

	// LOG
	a.LogCreate = noop.UnaryServer[log.CreateRequest, log.CreateResponse]{}
	a.LogRetrieve = noop.UnaryServer[log.RetrieveRequest, log.RetrieveResponse]{}
	a.LogDelete = noop.UnaryServer[log.DeleteRequest, types.Nil]{}
	a.LogRename = noop.UnaryServer[log.RenameRequest, types.Nil]{}
	a.LogSetData = noop.UnaryServer[log.SetDataRequest, types.Nil]{}

	// TABLE
	a.TableCreate = noop.UnaryServer[table.CreateRequest, table.CreateResponse]{}
	a.TableRetrieve = noop.UnaryServer[table.RetrieveRequest, table.RetrieveResponse]{}
	a.TableDelete = noop.UnaryServer[table.DeleteRequest, types.Nil]{}
	a.TableRename = noop.UnaryServer[table.RenameRequest, types.Nil]{}
	a.TableSetData = noop.UnaryServer[table.SetDataRequest, types.Nil]{}

	// LABEL
	a.LabelCreate = noop.UnaryServer[label.CreateRequest, label.CreateResponse]{}
	a.LabelRetrieve = noop.UnaryServer[label.RetrieveRequest, label.RetrieveResponse]{}
	a.LabelDelete = noop.UnaryServer[label.DeleteRequest, types.Nil]{}
	a.LabelAdd = noop.UnaryServer[label.AddRequest, types.Nil]{}
	a.LabelRemove = noop.UnaryServer[label.RemoveRequest, types.Nil]{}

	// ACCESS
	a.AccessCreatePolicy = noop.UnaryServer[access.CreatePolicyRequest, access.CreatePolicyResponse]{}
	a.AccessDeletePolicy = noop.UnaryServer[access.DeletePolicyRequest, types.Nil]{}
	a.AccessRetrievePolicy = noop.UnaryServer[access.RetrievePolicyRequest, access.RetrievePolicyResponse]{}
	a.AccessCreateRole = noop.UnaryServer[access.CreateRoleRequest, access.CreateRoleResponse]{}
	a.AccessDeleteRole = noop.UnaryServer[access.DeleteRoleRequest, types.Nil]{}
	a.AccessRetrieveRole = noop.UnaryServer[access.RetrieveRoleRequest, access.RetrieveRoleResponse]{}
	a.AccessAssignRole = noop.UnaryServer[access.AssignRoleRequest, types.Nil]{}
	a.AccessUnassignRole = noop.UnaryServer[access.UnassignRoleRequest, types.Nil]{}

	// IMPORT/EXPORT
	a.ImExImport = noop.UnaryServer[imex.ImportRequest, imex.ImportResponse]{}
	a.ImExExport = noop.UnaryServer[imex.ExportRequest, imex.ExportResponse]{}

	// ARC LSP
	a.ArcLSP = noop.StreamServer[arc.LSPMessage, arc.LSPMessage]{}

	return a, transports
}
