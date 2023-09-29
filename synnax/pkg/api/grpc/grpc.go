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

	a.AuthChangeUsername = fnoop.UnaryServer[api.ChangeUsernameRequest, types.Nil]{}
	a.AuthChangePassword = fnoop.UnaryServer[api.ChangePasswordRequest, types.Nil]{}
	a.AuthRegistration = fnoop.UnaryServer[api.RegistrationRequest, api.TokenResponse]{}
	a.RangeCreate = fnoop.UnaryServer[api.RangeCreateRequest, api.RangeCreateResponse]{}
	a.RangeRetrieve = fnoop.UnaryServer[api.RangeRetrieveRequest, api.RangeRetrieveResponse]{}
	a.OntologyRetrieve = fnoop.UnaryServer[api.OntologyRetrieveRequest, api.OntologyRetrieveResponse]{}
	a.OntologyGroupCreate = fnoop.UnaryServer[api.OntologyCreateGroupRequest, api.OntologyCreateGroupResponse]{}
	a.OntologyGroupDelete = fnoop.UnaryServer[api.OntologyDeleteGroupRequest, types.Nil]{}
	a.OntologyGroupRename = fnoop.UnaryServer[api.OntologyRenameGroupRequest, types.Nil]{}
	a.OntologyAddChildren = fnoop.UnaryServer[api.OntologyAddChildrenRequest, types.Nil]{}
	a.OntologyRemoveChildren = fnoop.UnaryServer[api.OntologyRemoveChildrenRequest, types.Nil]{}
	a.OntologyMoveChildren = fnoop.UnaryServer[api.OntologyMoveChildrenRequest, types.Nil]{}
	a.WorkspaceCreate = fnoop.UnaryServer[api.WorkspaceCreateRequest, api.WorkspaceCreateResponse]{}
	a.WorkspaceRetrieve = fnoop.UnaryServer[api.WorkspaceRetrieveRequest, api.WorkspaceRetrieveResponse]{}
	a.WorkspaceDelete = fnoop.UnaryServer[api.WorkspaceDeleteRequest, types.Nil]{}
	a.WorkspacePIDCreate = fnoop.UnaryServer[api.WorkspacePIDCreateRequest, api.WorkspacePIDCreateResponse]{}
	a.WorkspacePIDCreate = fnoop.UnaryServer[api.WorkspacePIDCreateRequest, api.WorkspacePIDCreateResponse]{}
	a.WorkspacePIDRetrieve = fnoop.UnaryServer[api.WorkspacePIDRetrieveRequest, api.WorkspacePIDRetrieveResponse]{}
	a.WorkspacePIDDelete = fnoop.UnaryServer[api.WorkspacePIDDeleteRequest, types.Nil]{}
	return a, transports
}
