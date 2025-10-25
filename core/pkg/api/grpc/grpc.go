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

func New(channels channel.Readable) (a api.Transport, transports []fgrpc.BindableTransport) {
	transports = make([]fgrpc.BindableTransport, 0, 20)
	transports = append(transports, newChannel(&a)...)
	transports = append(transports, newFramer(&a, channels))
	transports = append(transports, newConnectivity(&a))
	transports = append(transports, newAuth(&a))
	transports = append(transports, newRanger(&a))
	transports = append(transports, newHardware(&a))

	// AUTH
	a.AuthChangePassword = fnoop.UnaryServer[api.AuthChangePasswordRequest, types.Nil]{}

	// HARDWARE
	a.HardwareCopyTask = fnoop.UnaryServer[api.HardwareCopyTaskRequest, api.HardwareCopyTaskResponse]{}

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
	a.OntologyGroupCreate = fnoop.UnaryServer[api.OntologyCreateGroupRequest, api.OntologyCreateGroupResponse]{}
	a.OntologyGroupDelete = fnoop.UnaryServer[api.OntologyDeleteGroupRequest, types.Nil]{}
	a.OntologyGroupRename = fnoop.UnaryServer[api.OntologyRenameGroupRequest, types.Nil]{}

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
	a.SchematicSymbolCreate = fnoop.UnaryServer[api.SymbolCreateRequest, api.SymbolCreateResponse]{}
	a.SchematicSymbolRetrieve = fnoop.UnaryServer[api.SymbolRetrieveRequest, api.SymbolRetrieveResponse]{}
	a.SchematicSymbolDelete = fnoop.UnaryServer[api.SymbolDeleteRequest, types.Nil]{}
	a.SchematicSymbolRename = fnoop.UnaryServer[api.SymbolRenameRequest, types.Nil]{}
	a.SchematicSymbolRetrieveGroup = fnoop.UnaryServer[api.SymbolRetrieveGroupRequest, api.SymbolRetrieveGroupResponse]{}

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

	// STATUS
	a.StatusSet = fnoop.UnaryServer[api.StatusSetRequest, api.StatusSetResponse]{}
	a.StatusRetrieve = fnoop.UnaryServer[api.StatusRetrieveRequest, api.StatusRetrieveResponse]{}
	a.StatusDelete = fnoop.UnaryServer[api.StatusDeleteRequest, types.Nil]{}

	// arc
	a.ArcCreate = fnoop.UnaryServer[api.ArcCreateRequest, api.ArcCreateResponse]{}
	a.ArcDelete = fnoop.UnaryServer[api.ArcDeleteRequest, types.Nil]{}
	a.ArcRetrieve = fnoop.UnaryServer[api.ArcRetrieveRequest, api.ArcRetrieveResponse]{}
	a.ArcLSP = fnoop.StreamServer[api.ArcLSPMessage, api.ArcLSPMessage]{}

	return a, transports
}
