// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
)

func New(channelSvc *channel.Service) (api.Transport, []fgrpc.BindableTransport) {
	var a api.Transport
	transports := fgrpc.CompoundBindableTransport{
		newChannel(&a),
		newFramer(&a, channelSvc),
		newConnectivity(&a),
		newAuth(&a),
		newRanger(&a),
		newRack(&a),
		newTask(&a),
		newDevice(&a),
		newStatus(&a),
	}

	// AUTH
	a.AuthChangePassword = fnoop.UnaryServer[api.AuthChangePasswordRequest, types.Nil]{}

	// CHANNEL
	a.ChannelRename = fnoop.UnaryServer[api.ChannelRenameRequest, types.Nil]{}
	a.ChannelRetrieveGroup = fnoop.UnaryServer[api.ChannelRetrieveGroupRequest, api.ChannelRetrieveGroupResponse]{}

	// USER
	a.UserRename = fnoop.UnaryServer[api.UserRenameRequest, types.Nil]{}
	a.UserChangeUsername = fnoop.UnaryServer[api.UserChangeUsernameRequest, types.Nil]{}
	a.UserCreate = fnoop.UnaryServer[api.UserCreateRequest, api.UserCreateResponse]{}
	a.UserDelete = fnoop.UnaryServer[api.UserDeleteRequest, types.Nil]{}
	a.UserRetrieve = fnoop.UnaryServer[api.UserRetrieveRequest, api.UserRetrieveResponse]{}

	// RANGE
	a.RangeRename = fnoop.UnaryServer[api.RangeRenameRequest, types.Nil]{}
	a.RangeAliasRetrieve = fnoop.UnaryServer[api.RangeAliasRetrieveRequest, api.RangeAliasRetrieveResponse]{}

	// ONTOLOGY
	a.OntologyRetrieve = fnoop.UnaryServer[api.OntologyRetrieveRequest, api.OntologyRetrieveResponse]{}
	a.OntologyAddChildren = fnoop.UnaryServer[api.OntologyAddChildrenRequest, types.Nil]{}
	a.OntologyRemoveChildren = fnoop.UnaryServer[api.OntologyRemoveChildrenRequest, types.Nil]{}
	a.OntologyMoveChildren = fnoop.UnaryServer[api.OntologyMoveChildrenRequest, types.Nil]{}

	// GROUP
	a.GroupCreate = fnoop.UnaryServer[api.GroupCreateRequest, api.GroupCreateResponse]{}
	a.GroupDelete = fnoop.UnaryServer[api.GroupDeleteRequest, types.Nil]{}
	a.GroupRename = fnoop.UnaryServer[api.GroupRenameRequest, types.Nil]{}

	// WORKSPACE
	a.WorkspaceCreate = fnoop.UnaryServer[api.WorkspaceCreateRequest, api.WorkspaceCreateResponse]{}
	a.WorkspaceRetrieve = fnoop.UnaryServer[api.WorkspaceRetrieveRequest, api.WorkspaceRetrieveResponse]{}
	a.WorkspaceDelete = fnoop.UnaryServer[api.WorkspaceDeleteRequest, types.Nil]{}
	a.WorkspaceRename = fnoop.UnaryServer[api.WorkspaceRenameRequest, types.Nil]{}
	a.WorkspaceSetLayout = fnoop.UnaryServer[api.WorkspaceSetLayoutRequest, types.Nil]{}

	// SCHEMATIC
	a.SchematicCreate = fnoop.UnaryServer[api.SchematicCreateRequest, api.SchematicCreateResponse]{}
	a.SchematicDelete = fnoop.UnaryServer[api.SchematicDeleteRequest, types.Nil]{}
	a.SchematicRetrieve = fnoop.UnaryServer[api.SchematicRetrieveRequest, api.SchematicRetrieveResponse]{}
	a.SchematicRename = fnoop.UnaryServer[api.SchematicRenameRequest, types.Nil]{}
	a.SchematicSetData = fnoop.UnaryServer[api.SchematicSetDataRequest, types.Nil]{}
	a.SchematicCopy = fnoop.UnaryServer[api.SchematicCopyRequest, api.SchematicCopyResponse]{}

	// SCHEMATIC SYMBOL
	a.SchematicCreateSymbol = fnoop.UnaryServer[api.SchematicCreateSymbolRequest, api.SchematicCreateSymbolResponse]{}
	a.SchematicRetrieveSymbol = fnoop.UnaryServer[api.SchematicRetrieveSymbolRequest, api.SchematicRetrieveSymbolResponse]{}
	a.SchematicDeleteSymbol = fnoop.UnaryServer[api.SchematicDeleteSymbolRequest, types.Nil]{}
	a.SchematicRenameSymbol = fnoop.UnaryServer[api.SchematicRenameSymbolRequest, types.Nil]{}
	a.SchematicRetrieveSymbolGroup = fnoop.UnaryServer[api.SchematicRetrieveSymbolGroupRequest, api.SchematicRetrieveSymbolGroupResponse]{}

	// LINE PLOT
	a.LinePlotCreate = fnoop.UnaryServer[api.LinePlotCreateRequest, api.LinePlotCreateResponse]{}
	a.LinePlotRetrieve = fnoop.UnaryServer[api.LinePlotRetrieveRequest, api.LinePlotRetrieveResponse]{}
	a.LinePlotDelete = fnoop.UnaryServer[api.LinePlotDeleteRequest, types.Nil]{}
	a.LinePlotRename = fnoop.UnaryServer[api.LinePlotRenameRequest, types.Nil]{}
	a.LinePlotSetData = fnoop.UnaryServer[api.LinePlotSetDataRequest, types.Nil]{}

	// LOG
	a.LogCreate = fnoop.UnaryServer[api.LogCreateRequest, api.LogCreateResponse]{}
	a.LogRetrieve = fnoop.UnaryServer[api.LogRetrieveRequest, api.LogRetrieveResponse]{}
	a.LogDelete = fnoop.UnaryServer[api.LogDeleteRequest, types.Nil]{}
	a.LogRename = fnoop.UnaryServer[api.LogRenameRequest, types.Nil]{}
	a.LogSetData = fnoop.UnaryServer[api.LogSetDataRequest, types.Nil]{}

	// TABLE
	a.TableCreate = fnoop.UnaryServer[api.TableCreateRequest, api.TableCreateResponse]{}
	a.TableRetrieve = fnoop.UnaryServer[api.TableRetrieveRequest, api.TableRetrieveResponse]{}
	a.TableDelete = fnoop.UnaryServer[api.TableDeleteRequest, types.Nil]{}
	a.TableRename = fnoop.UnaryServer[api.TableRenameRequest, types.Nil]{}
	a.TableSetData = fnoop.UnaryServer[api.TableSetDataRequest, types.Nil]{}

	// LABEL
	a.LabelCreate = fnoop.UnaryServer[api.LabelCreateRequest, api.LabelCreateResponse]{}
	a.LabelRetrieve = fnoop.UnaryServer[api.LabelRetrieveRequest, api.LabelRetrieveResponse]{}
	a.LabelDelete = fnoop.UnaryServer[api.LabelDeleteRequest, types.Nil]{}
	a.LabelAdd = fnoop.UnaryServer[api.LabelAddRequest, types.Nil]{}
	a.LabelRemove = fnoop.UnaryServer[api.LabelRemoveRequest, types.Nil]{}

	// ACCESS
	a.AccessCreatePolicy = fnoop.UnaryServer[api.AccessCreatePolicyRequest, api.AccessCreatePolicyResponse]{}
	a.AccessDeletePolicy = fnoop.UnaryServer[api.AccessDeletePolicyRequest, types.Nil]{}
	a.AccessRetrievePolicy = fnoop.UnaryServer[api.AccessRetrievePolicyRequest, api.AccessRetrievePolicyResponse]{}
	a.AccessCreateRole = fnoop.UnaryServer[api.AccessCreateRoleRequest, api.AccessCreateRoleResponse]{}
	a.AccessDeleteRole = fnoop.UnaryServer[api.AccessDeleteRoleRequest, types.Nil]{}
	a.AccessRetrieveRole = fnoop.UnaryServer[api.AccessRetrieveRoleRequest, api.AccessRetrieveRoleResponse]{}
	a.AccessAssignRole = fnoop.UnaryServer[api.AccessAssignRoleRequest, types.Nil]{}
	a.AccessUnassignRole = fnoop.UnaryServer[api.AccessUnassignRoleRequest, types.Nil]{}

	// arc LSP (streaming, not implemented via gRPC yet)
	a.ArcLSP = fnoop.StreamServer[api.ArcLSPMessage, api.ArcLSPMessage]{}

	return a, transports
}
