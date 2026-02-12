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

	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/freighter/fnoop"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/access"
	apiarc "github.com/synnaxlabs/synnax/pkg/api/arc"
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
	"github.com/synnaxlabs/synnax/pkg/api/label"
	"github.com/synnaxlabs/synnax/pkg/api/lineplot"
	"github.com/synnaxlabs/synnax/pkg/api/log"
	"github.com/synnaxlabs/synnax/pkg/api/ontology"
	"github.com/synnaxlabs/synnax/pkg/api/ranger"
	"github.com/synnaxlabs/synnax/pkg/api/ranger/alias"
	"github.com/synnaxlabs/synnax/pkg/api/schematic"
	"github.com/synnaxlabs/synnax/pkg/api/table"
	"github.com/synnaxlabs/synnax/pkg/api/user"
	"github.com/synnaxlabs/synnax/pkg/api/view"
	"github.com/synnaxlabs/synnax/pkg/api/workspace"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
)

func NewTransport(channelSvc *distchannel.Service) (api.Transport, []fgrpc.BindableTransport) {
	var a api.Transport
	transports := fgrpc.CompoundBindableTransport{
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
	}

	// AUTH
	a.AuthChangePassword = fnoop.UnaryServer[auth.ChangePasswordRequest, types.Nil]{}

	// CHANNEL
	a.ChannelRename = fnoop.UnaryServer[channel.RenameRequest, types.Nil]{}
	a.ChannelRetrieveGroup = fnoop.UnaryServer[channel.RetrieveGroupRequest, channel.RetrieveGroupResponse]{}

	// USER
	a.UserRename = fnoop.UnaryServer[user.RenameRequest, types.Nil]{}
	a.UserChangeUsername = fnoop.UnaryServer[user.ChangeUsernameRequest, types.Nil]{}
	a.UserCreate = fnoop.UnaryServer[user.CreateRequest, user.CreateResponse]{}
	a.UserDelete = fnoop.UnaryServer[user.DeleteRequest, types.Nil]{}
	a.UserRetrieve = fnoop.UnaryServer[user.RetrieveRequest, user.RetrieveResponse]{}

	// RANGE
	a.RangeRename = fnoop.UnaryServer[ranger.RenameRequest, types.Nil]{}
	a.AliasRetrieve = fnoop.UnaryServer[alias.RetrieveRequest, alias.RetrieveResponse]{}

	// ONTOLOGY
	a.OntologyRetrieve = fnoop.UnaryServer[ontology.RetrieveRequest, ontology.RetrieveResponse]{}
	a.OntologyAddChildren = fnoop.UnaryServer[ontology.AddChildrenRequest, types.Nil]{}
	a.OntologyRemoveChildren = fnoop.UnaryServer[ontology.RemoveChildrenRequest, types.Nil]{}
	a.OntologyMoveChildren = fnoop.UnaryServer[ontology.MoveChildrenRequest, types.Nil]{}

	// GROUP
	a.GroupCreate = fnoop.UnaryServer[group.CreateRequest, group.CreateResponse]{}
	a.GroupDelete = fnoop.UnaryServer[group.DeleteRequest, types.Nil]{}
	a.GroupRename = fnoop.UnaryServer[group.RenameRequest, types.Nil]{}

	// WORKSPACE
	a.WorkspaceCreate = fnoop.UnaryServer[workspace.CreateRequest, workspace.CreateResponse]{}
	a.WorkspaceRetrieve = fnoop.UnaryServer[workspace.RetrieveRequest, workspace.RetrieveResponse]{}
	a.WorkspaceDelete = fnoop.UnaryServer[workspace.DeleteRequest, types.Nil]{}
	a.WorkspaceRename = fnoop.UnaryServer[workspace.RenameRequest, types.Nil]{}
	a.WorkspaceSetLayout = fnoop.UnaryServer[workspace.SetLayoutRequest, types.Nil]{}

	// SCHEMATIC
	a.SchematicCreate = fnoop.UnaryServer[schematic.CreateRequest, schematic.CreateResponse]{}
	a.SchematicDelete = fnoop.UnaryServer[schematic.DeleteRequest, types.Nil]{}
	a.SchematicRetrieve = fnoop.UnaryServer[schematic.RetrieveRequest, schematic.RetrieveResponse]{}
	a.SchematicRename = fnoop.UnaryServer[schematic.RenameRequest, types.Nil]{}
	a.SchematicSetData = fnoop.UnaryServer[schematic.SetDataRequest, types.Nil]{}
	a.SchematicCopy = fnoop.UnaryServer[schematic.CopyRequest, schematic.CopyResponse]{}

	// SCHEMATIC SYMBOL
	a.SchematicCreateSymbol = fnoop.UnaryServer[schematic.CreateSymbolRequest, schematic.CreateSymbolResponse]{}
	a.SchematicRetrieveSymbol = fnoop.UnaryServer[schematic.RetrieveSymbolRequest, schematic.RetrieveSymbolResponse]{}
	a.SchematicDeleteSymbol = fnoop.UnaryServer[schematic.DeleteSymbolRequest, types.Nil]{}
	a.SchematicRenameSymbol = fnoop.UnaryServer[schematic.RenameSymbolRequest, types.Nil]{}
	a.SchematicRetrieveSymbolGroup = fnoop.UnaryServer[schematic.RetrieveSymbolGroupRequest, schematic.RetrieveSymbolGroupResponse]{}

	// LINE PLOT
	a.LinePlotCreate = fnoop.UnaryServer[lineplot.CreateRequest, lineplot.CreateResponse]{}
	a.LinePlotRetrieve = fnoop.UnaryServer[lineplot.RetrieveRequest, lineplot.RetrieveResponse]{}
	a.LinePlotDelete = fnoop.UnaryServer[lineplot.DeleteRequest, types.Nil]{}
	a.LinePlotRename = fnoop.UnaryServer[lineplot.RenameRequest, types.Nil]{}
	a.LinePlotSetData = fnoop.UnaryServer[lineplot.SetDataRequest, types.Nil]{}

	// LOG
	a.LogCreate = fnoop.UnaryServer[log.CreateRequest, log.CreateResponse]{}
	a.LogRetrieve = fnoop.UnaryServer[log.RetrieveRequest, log.RetrieveResponse]{}
	a.LogDelete = fnoop.UnaryServer[log.DeleteRequest, types.Nil]{}
	a.LogRename = fnoop.UnaryServer[log.RenameRequest, types.Nil]{}
	a.LogSetData = fnoop.UnaryServer[log.SetDataRequest, types.Nil]{}

	// TABLE
	a.TableCreate = fnoop.UnaryServer[table.CreateRequest, table.CreateResponse]{}
	a.TableRetrieve = fnoop.UnaryServer[table.RetrieveRequest, table.RetrieveResponse]{}
	a.TableDelete = fnoop.UnaryServer[table.DeleteRequest, types.Nil]{}
	a.TableRename = fnoop.UnaryServer[table.RenameRequest, types.Nil]{}
	a.TableSetData = fnoop.UnaryServer[table.SetDataRequest, types.Nil]{}

	// LABEL
	a.LabelCreate = fnoop.UnaryServer[label.CreateRequest, label.CreateResponse]{}
	a.LabelRetrieve = fnoop.UnaryServer[label.RetrieveRequest, label.RetrieveResponse]{}
	a.LabelDelete = fnoop.UnaryServer[label.DeleteRequest, types.Nil]{}
	a.LabelAdd = fnoop.UnaryServer[label.AddRequest, types.Nil]{}
	a.LabelRemove = fnoop.UnaryServer[label.RemoveRequest, types.Nil]{}

	// ACCESS
	a.AccessCreatePolicy = fnoop.UnaryServer[access.CreatePolicyRequest, access.CreatePolicyResponse]{}
	a.AccessDeletePolicy = fnoop.UnaryServer[access.DeletePolicyRequest, types.Nil]{}
	a.AccessRetrievePolicy = fnoop.UnaryServer[access.RetrievePolicyRequest, access.RetrievePolicyResponse]{}
	a.AccessCreateRole = fnoop.UnaryServer[access.CreateRoleRequest, access.CreateRoleResponse]{}
	a.AccessDeleteRole = fnoop.UnaryServer[access.DeleteRoleRequest, types.Nil]{}
	a.AccessRetrieveRole = fnoop.UnaryServer[access.RetrieveRoleRequest, access.RetrieveRoleResponse]{}
	a.AccessAssignRole = fnoop.UnaryServer[access.AssignRoleRequest, types.Nil]{}
	a.AccessUnassignRole = fnoop.UnaryServer[access.UnassignRoleRequest, types.Nil]{}

	// ARC LSP (streaming, not implemented via gRPC yet)
	a.ArcLSP = fnoop.StreamServer[apiarc.LSPMessage, apiarc.LSPMessage]{}

	// VIEW
	a.ViewCreate = fnoop.UnaryServer[view.CreateRequest, view.CreateResponse]{}
	a.ViewRetrieve = fnoop.UnaryServer[view.RetrieveRequest, view.RetrieveResponse]{}
	a.ViewDelete = fnoop.UnaryServer[view.DeleteRequest, types.Nil]{}

	return a, transports
}
