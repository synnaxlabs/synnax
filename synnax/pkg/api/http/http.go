// Copyright 2023 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/freighter/fhttp"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/auth"
)

func New(router *fhttp.Router) (a api.Transport) {
	a.AuthLogin = fhttp.UnaryServer[auth.InsecureCredentials, api.TokenResponse](router, "/api/v1/auth/login")
	a.AuthRegistration = fhttp.UnaryServer[api.RegistrationRequest, api.TokenResponse](router, "/api/v1/auth/register")
	a.AuthChangePassword = fhttp.UnaryServer[api.ChangePasswordRequest, types.Nil](router, "/api/v1/auth/protected/change-password")
	a.AuthChangeUsername = fhttp.UnaryServer[api.ChangeUsernameRequest, types.Nil](router, "/api/v1/auth/protected/change-username")
	a.ChannelCreate = fhttp.UnaryServer[api.ChannelCreateRequest, api.ChannelCreateResponse](router, "/api/v1/channel/create")
	a.ChannelRetrieve = fhttp.UnaryServer[api.ChannelRetrieveRequest, api.ChannelRetrieveResponse](router, "/api/v1/channel/retrieve")
	a.ConnectivityCheck = fhttp.UnaryServer[types.Nil, api.ConnectivityCheckResponse](router, "/api/v1/connectivity/check")
	a.FrameWriter = fhttp.StreamServer[api.FrameWriterRequest, api.FrameWriterResponse](router, "/api/v1/frame/write")
	a.FrameIterator = fhttp.StreamServer[api.FrameIteratorRequest, api.FrameIteratorResponse](router, "/api/v1/frame/iterate")
	a.FrameStreamer = fhttp.StreamServer[api.FrameStreamerRequest, api.FrameStreamerResponse](router, "/api/v1/frame/stream")
	a.OntologyRetrieve = fhttp.UnaryServer[api.OntologyRetrieveRequest, api.OntologyRetrieveResponse](router, "/api/v1/ontology/retrieve")
	a.OntologyGroupCreate = fhttp.UnaryServer[api.OntologyCreateGroupRequest, api.OntologyCreateGroupResponse](router, "/api/v1/ontology/create-group")
	a.OntologyGroupDelete = fhttp.UnaryServer[api.OntologyDeleteGroupRequest, types.Nil](router, "/api/v1/ontology/delete-group")
	a.OntologyGroupRename = fhttp.UnaryServer[api.OntologyRenameGroupRequest, types.Nil](router, "/api/v1/ontology/rename-group")
	a.OntologyAddChildren = fhttp.UnaryServer[api.OntologyAddChildrenRequest, types.Nil](router, "/api/v1/ontology/add-children")
	a.OntologyRemoveChildren = fhttp.UnaryServer[api.OntologyRemoveChildrenRequest, types.Nil](router, "/api/v1/ontology/remove-children")
	a.OntologyMoveChildren = fhttp.UnaryServer[api.OntologyMoveChildrenRequest, types.Nil](router, "/api/v1/ontology/move-children")
	a.RangeRetrieve = fhttp.UnaryServer[api.RangeRetrieveRequest, api.RangeRetrieveResponse](router, "/api/v1/range/retrieve")
	a.RangeCreate = fhttp.UnaryServer[api.RangeCreateRequest, api.RangeCreateResponse](router, "/api/v1/range/create")
	a.WorkspaceCreate = fhttp.UnaryServer[api.WorkspaceCreateRequest, api.WorkspaceCreateResponse](router, "/api/v1/workspace/create")
	a.WorkspaceRetrieve = fhttp.UnaryServer[api.WorkspaceRetrieveRequest, api.WorkspaceRetrieveResponse](router, "/api/v1/workspace/retrieve")
	a.WorkspaceDelete = fhttp.UnaryServer[api.WorkspaceDeleteRequest, types.Nil](router, "/api/v1/workspace/delete")
	a.WorkspacePIDCreate = fhttp.UnaryServer[api.WorkspacePIDCreateRequest, api.WorkspacePIDCreateResponse](router, "/api/v1/workspace/pid/create")
	a.WorkspacePIDRetrieve = fhttp.UnaryServer[api.WorkspacePIDRetrieveRequest, api.WorkspacePIDRetrieveResponse](router, "/api/v1/workspace/pid/retrieve")
	a.WorkspacePIDDelete = fhttp.UnaryServer[api.WorkspacePIDDeleteRequest, types.Nil](router, "/api/v1/workspace/pid/delete")
	return a
}
