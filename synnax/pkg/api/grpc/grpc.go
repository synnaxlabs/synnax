// Copyright 2023 Synnax Labs, Inc.
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
)

func New() (a api.Transport, transports []fgrpc.BindableTransport) {
	transports = make([]fgrpc.BindableTransport, 0, 20)
	transports = append(transports, newChannel(&a)...)
	transports = append(transports, newFramer(&a))
	transports = append(transports, newConnectivity(&a))
	transports = append(transports, newAuth(&a))
	transports = append(transports, newRanger(&a))
	transports = append(transports, newHardware(&a))

	// AUTH
	a.AuthChangePassword = fnoop.UnaryServer[api.AuthChangePasswordRequest, types.Nil]{}
	a.AuthChangeUsername = fnoop.UnaryServer[api.AuthChangeUsernameRequest, types.Nil]{}

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

	// Schematic
	a.SchematicCreate = fnoop.UnaryServer[api.SchematicCreateRequest, api.SchematicCreateResponse]{}
	a.SchematicDelete = fnoop.UnaryServer[api.SchematicDeleteRequest, types.Nil]{}
	a.SchematicRetrieve = fnoop.UnaryServer[api.SchematicRetrieveRequest, api.SchematicRetrieveResponse]{}
	a.SchematicRename = fnoop.UnaryServer[api.SchematicRenameRequest, types.Nil]{}
	a.SchematicSetData = fnoop.UnaryServer[api.SchematicSetDataRequest, types.Nil]{}
	a.SchematicCopy = fnoop.UnaryServer[api.SchematicCopyRequest, api.SchematicCopyResponse]{}

	// LINE PLOT
	a.LinePlotCreate = fnoop.UnaryServer[api.LinePlotCreateRequest, api.LinePlotCreateResponse]{}
	a.LinePlotRetrieve = fnoop.UnaryServer[api.LinePlotRetrieveRequest, api.LinePlotRetrieveResponse]{}
	a.LinePlotDelete = fnoop.UnaryServer[api.LinePlotDeleteRequest, types.Nil]{}
	a.LinePlotRename = fnoop.UnaryServer[api.LinePlotRenameRequest, types.Nil]{}
	a.LinePlotSetData = fnoop.UnaryServer[api.LinePlotSetDataRequest, types.Nil]{}

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

	return a, transports
}
