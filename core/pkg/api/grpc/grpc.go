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
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
)

func New(channelSvc *channel.Service) (api.Transport, []grpc.BindableTransport) {
	var a api.Transport
	transports := grpc.CompoundBindableTransport{
		newChannel(&a),
		newFramer(&a, channelSvc),
		newConnectivity(&a),
		newAuth(&a),
		newRanger(&a),
		newRack(&a),
		newTask(&a),
		newDevice(&a),
		newStatus(&a),
		newArc(&a),
	}

	// AUTH
	a.AuthChangePassword = noop.UnaryServer[api.AuthChangePasswordRequest, types.Nil]{}

	// CHANNEL
	a.ChannelRename = noop.UnaryServer[api.ChannelRenameRequest, types.Nil]{}
	a.ChannelRetrieveGroup = noop.UnaryServer[api.ChannelRetrieveGroupRequest, api.ChannelRetrieveGroupResponse]{}

	// USER
	a.UserRename = noop.UnaryServer[api.UserRenameRequest, types.Nil]{}
	a.UserChangeUsername = noop.UnaryServer[api.UserChangeUsernameRequest, types.Nil]{}
	a.UserCreate = noop.UnaryServer[api.UserCreateRequest, api.UserCreateResponse]{}
	a.UserDelete = noop.UnaryServer[api.UserDeleteRequest, types.Nil]{}
	a.UserRetrieve = noop.UnaryServer[api.UserRetrieveRequest, api.UserRetrieveResponse]{}

	// RANGE
	a.RangeRename = noop.UnaryServer[api.RangeRenameRequest, types.Nil]{}
	a.RangeAliasRetrieve = noop.UnaryServer[api.RangeAliasRetrieveRequest, api.RangeAliasRetrieveResponse]{}

	// ONTOLOGY
	a.OntologyRetrieve = noop.UnaryServer[api.OntologyRetrieveRequest, api.OntologyRetrieveResponse]{}
	a.OntologyAddChildren = noop.UnaryServer[api.OntologyAddChildrenRequest, types.Nil]{}
	a.OntologyRemoveChildren = noop.UnaryServer[api.OntologyRemoveChildrenRequest, types.Nil]{}
	a.OntologyMoveChildren = noop.UnaryServer[api.OntologyMoveChildrenRequest, types.Nil]{}

	// GROUP
	a.GroupCreate = noop.UnaryServer[api.GroupCreateRequest, api.GroupCreateResponse]{}
	a.GroupDelete = noop.UnaryServer[api.GroupDeleteRequest, types.Nil]{}
	a.GroupRename = noop.UnaryServer[api.GroupRenameRequest, types.Nil]{}

	// WORKSPACE
	a.WorkspaceCreate = noop.UnaryServer[api.WorkspaceCreateRequest, api.WorkspaceCreateResponse]{}
	a.WorkspaceRetrieve = noop.UnaryServer[api.WorkspaceRetrieveRequest, api.WorkspaceRetrieveResponse]{}
	a.WorkspaceDelete = noop.UnaryServer[api.WorkspaceDeleteRequest, types.Nil]{}
	a.WorkspaceRename = noop.UnaryServer[api.WorkspaceRenameRequest, types.Nil]{}
	a.WorkspaceSetLayout = noop.UnaryServer[api.WorkspaceSetLayoutRequest, types.Nil]{}

	// SCHEMATIC
	a.SchematicCreate = noop.UnaryServer[api.SchematicCreateRequest, api.SchematicCreateResponse]{}
	a.SchematicDelete = noop.UnaryServer[api.SchematicDeleteRequest, types.Nil]{}
	a.SchematicRetrieve = noop.UnaryServer[api.SchematicRetrieveRequest, api.SchematicRetrieveResponse]{}
	a.SchematicRename = noop.UnaryServer[api.SchematicRenameRequest, types.Nil]{}
	a.SchematicSetData = noop.UnaryServer[api.SchematicSetDataRequest, types.Nil]{}
	a.SchematicCopy = noop.UnaryServer[api.SchematicCopyRequest, api.SchematicCopyResponse]{}

	// SCHEMATIC SYMBOL
	a.SchematicCreateSymbol = noop.UnaryServer[api.SchematicCreateSymbolRequest, api.SchematicCreateSymbolResponse]{}
	a.SchematicRetrieveSymbol = noop.UnaryServer[api.SchematicRetrieveSymbolRequest, api.SchematicRetrieveSymbolResponse]{}
	a.SchematicDeleteSymbol = noop.UnaryServer[api.SchematicDeleteSymbolRequest, types.Nil]{}
	a.SchematicRenameSymbol = noop.UnaryServer[api.SchematicRenameSymbolRequest, types.Nil]{}
	a.SchematicRetrieveSymbolGroup = noop.UnaryServer[api.SchematicRetrieveSymbolGroupRequest, api.SchematicRetrieveSymbolGroupResponse]{}

	// LINE PLOT
	a.LinePlotCreate = noop.UnaryServer[api.LinePlotCreateRequest, api.LinePlotCreateResponse]{}
	a.LinePlotRetrieve = noop.UnaryServer[api.LinePlotRetrieveRequest, api.LinePlotRetrieveResponse]{}
	a.LinePlotDelete = noop.UnaryServer[api.LinePlotDeleteRequest, types.Nil]{}
	a.LinePlotRename = noop.UnaryServer[api.LinePlotRenameRequest, types.Nil]{}
	a.LinePlotSetData = noop.UnaryServer[api.LinePlotSetDataRequest, types.Nil]{}

	// LOG
	a.LogCreate = noop.UnaryServer[api.LogCreateRequest, api.LogCreateResponse]{}
	a.LogRetrieve = noop.UnaryServer[api.LogRetrieveRequest, api.LogRetrieveResponse]{}
	a.LogDelete = noop.UnaryServer[api.LogDeleteRequest, types.Nil]{}
	a.LogRename = noop.UnaryServer[api.LogRenameRequest, types.Nil]{}
	a.LogSetData = noop.UnaryServer[api.LogSetDataRequest, types.Nil]{}

	// TABLE
	a.TableCreate = noop.UnaryServer[api.TableCreateRequest, api.TableCreateResponse]{}
	a.TableRetrieve = noop.UnaryServer[api.TableRetrieveRequest, api.TableRetrieveResponse]{}
	a.TableDelete = noop.UnaryServer[api.TableDeleteRequest, types.Nil]{}
	a.TableRename = noop.UnaryServer[api.TableRenameRequest, types.Nil]{}
	a.TableSetData = noop.UnaryServer[api.TableSetDataRequest, types.Nil]{}

	// LABEL
	a.LabelCreate = noop.UnaryServer[api.LabelCreateRequest, api.LabelCreateResponse]{}
	a.LabelRetrieve = noop.UnaryServer[api.LabelRetrieveRequest, api.LabelRetrieveResponse]{}
	a.LabelDelete = noop.UnaryServer[api.LabelDeleteRequest, types.Nil]{}
	a.LabelAdd = noop.UnaryServer[api.LabelAddRequest, types.Nil]{}
	a.LabelRemove = noop.UnaryServer[api.LabelRemoveRequest, types.Nil]{}

	// ACCESS
	a.AccessCreatePolicy = noop.UnaryServer[api.AccessCreatePolicyRequest, api.AccessCreatePolicyResponse]{}
	a.AccessDeletePolicy = noop.UnaryServer[api.AccessDeletePolicyRequest, types.Nil]{}
	a.AccessRetrievePolicy = noop.UnaryServer[api.AccessRetrievePolicyRequest, api.AccessRetrievePolicyResponse]{}
	a.AccessCreateRole = noop.UnaryServer[api.AccessCreateRoleRequest, api.AccessCreateRoleResponse]{}
	a.AccessDeleteRole = noop.UnaryServer[api.AccessDeleteRoleRequest, types.Nil]{}
	a.AccessRetrieveRole = noop.UnaryServer[api.AccessRetrieveRoleRequest, api.AccessRetrieveRoleResponse]{}
	a.AccessAssignRole = noop.UnaryServer[api.AccessAssignRoleRequest, types.Nil]{}
	a.AccessUnassignRole = noop.UnaryServer[api.AccessUnassignRoleRequest, types.Nil]{}

	// ARC LSP (streaming, not implemented via gRPC yet)
	a.ArcLSP = noop.StreamServer[api.ArcLSPMessage, api.ArcLSPMessage]{}

	// VIEW
	a.ViewCreate = noop.UnaryServer[api.ViewCreateRequest, api.ViewCreateResponse]{}
	a.ViewRetrieve = noop.UnaryServer[api.ViewRetrieveRequest, api.ViewRetrieveResponse]{}
	a.ViewDelete = noop.UnaryServer[api.ViewDeleteRequest, types.Nil]{}

	return a, transports
}
