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
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/freighter/fnoop"
	"github.com/synnaxlabs/synnax/pkg/api"
	"go/types"
)

func New() (a api.Transport, transports []fgrpc.BindableTransport) {
	transports = make([]fgrpc.BindableTransport, 0, 20)
	transports = append(transports, newChannel(&a)...)
	transports = append(transports, newFramer(&a))
	transports = append(transports, newConnectivity(&a))
	transports = append(transports, newAuth(&a))

	// AUTH
	a.AuthChangeUsername = fnoop.UnaryServer[api.ChangeUsernameRequest, types.Nil]{}
	a.AuthChangePassword = fnoop.UnaryServer[api.ChangePasswordRequest, types.Nil]{}
	a.AuthRegistration = fnoop.UnaryServer[api.RegistrationRequest, api.TokenResponse]{}

	// RANGE
	a.RangeCreate = fnoop.UnaryServer[api.RangeCreateRequest, api.RangeCreateResponse]{}
	a.RangeRetrieve = fnoop.UnaryServer[api.RangeRetrieveRequest, api.RangeRetrieveResponse]{}
	a.RangeKVGet = fnoop.UnaryServer[api.RangeKVGetRequest, api.RangeKVGetResponse]{}
	a.RangeKVSet = fnoop.UnaryServer[api.RangeKVSetRequest, types.Nil]{}
	a.RangeKVDelete = fnoop.UnaryServer[api.RangeKVDeleteRequest, types.Nil]{}
	a.RangeAliasSet = fnoop.UnaryServer[api.RangeAliasSetRequest, types.Nil]{}
	a.RangeAliasResolve = fnoop.UnaryServer[api.RangeAliasResolveRequest, api.RangeAliasResolveResponse]{}

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

	// PID
	a.PIDCreate = fnoop.UnaryServer[api.PIDCreateRequest, api.PIDCreateResponse]{}
	a.PIDDelete = fnoop.UnaryServer[api.PIDDeleteRequest, types.Nil]{}
	a.PIDRetrieve = fnoop.UnaryServer[api.PIDRetrieveRequest, api.PIDRetrieveResponse]{}
	a.PIDRename = fnoop.UnaryServer[api.PIDRenameRequest, types.Nil]{}
	a.PIDSetData = fnoop.UnaryServer[api.PIDSetDataRequest, types.Nil]{}

	// LINE PLOT
	a.LinePlotCreate = fnoop.UnaryServer[api.LinePlotCreateRequest, api.LinePlotCreateResponse]{}
	a.LinePlotRetrieve = fnoop.UnaryServer[api.LinePlotRetrieveRequest, api.LinePlotRetrieveResponse]{}
	a.LinePlotDelete = fnoop.UnaryServer[api.LinePlotDeleteRequest, types.Nil]{}
	a.LinePlotRename = fnoop.UnaryServer[api.LinePlotRenameRequest, types.Nil]{}
	a.LinePlotSetData = fnoop.UnaryServer[api.LinePlotSetDataRequest, types.Nil]{}

	return a, transports
}
