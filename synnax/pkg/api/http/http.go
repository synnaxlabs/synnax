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

func New(router *fhttp.Router) (t api.Transport) {
	// AUTH
	t.AuthLogin = fhttp.UnaryServer[auth.InsecureCredentials, api.TokenResponse](router, false, "/api/v1/auth/login")
	t.AuthRegistration = fhttp.UnaryServer[api.RegistrationRequest, api.TokenResponse](router, false, "/api/v1/auth/register")
	t.AuthChangePassword = fhttp.UnaryServer[api.ChangePasswordRequest, types.Nil](router, false, "/api/v1/auth/protected/change-password")
	t.AuthChangeUsername = fhttp.UnaryServer[api.ChangeUsernameRequest, types.Nil](router, false, "/api/v1/auth/protected/change-username")

	// CHANNEL
	t.ChannelCreate = fhttp.UnaryServer[api.ChannelCreateRequest, api.ChannelCreateResponse](router, false, "/api/v1/channel/create")
	t.ChannelRetrieve = fhttp.UnaryServer[api.ChannelRetrieveRequest, api.ChannelRetrieveResponse](router, false, "/api/v1/channel/retrieve")
	t.ChannelDelete = fhttp.UnaryServer[api.ChannelDeleteRequest, types.Nil](router, false, "/api/v1/channel/delete")
	t.ChannelRename = fhttp.UnaryServer[api.ChannelRenameRequest, types.Nil](router, false, "/api/v1/channel/rename")

	// CONNECTIVITY
	t.ConnectivityCheck = fhttp.UnaryServer[types.Nil, api.ConnectivityCheckResponse](router, false, "/api/v1/connectivity/check")

	// FRAME
	t.FrameWriter = fhttp.StreamServer[api.FrameWriterRequest, api.FrameWriterResponse](router, false, "/api/v1/frame/write")
	t.FrameIterator = fhttp.StreamServer[api.FrameIteratorRequest, api.FrameIteratorResponse](router, false, "/api/v1/frame/iterate")
	t.FrameStreamer = fhttp.StreamServer[api.FrameStreamerRequest, api.FrameStreamerResponse](router, false, "/api/v1/frame/stream")
	t.FrameDelete = fhttp.UnaryServer[api.FrameDeleteRequest, types.Nil](router, false, "/api/v1/frame/delete")

	// ONTOLOGY
	t.OntologyRetrieve = fhttp.UnaryServer[api.OntologyRetrieveRequest, api.OntologyRetrieveResponse](router, false, "/api/v1/ontology/retrieve")
	t.OntologyAddChildren = fhttp.UnaryServer[api.OntologyAddChildrenRequest, types.Nil](router, false, "/api/v1/ontology/add-children")
	t.OntologyRemoveChildren = fhttp.UnaryServer[api.OntologyRemoveChildrenRequest, types.Nil](router, false, "/api/v1/ontology/remove-children")
	t.OntologyMoveChildren = fhttp.UnaryServer[api.OntologyMoveChildrenRequest, types.Nil](router, false, "/api/v1/ontology/move-children")

	// GROUP
	t.OntologyGroupCreate = fhttp.UnaryServer[api.OntologyCreateGroupRequest, api.OntologyCreateGroupResponse](router, false, "/api/v1/ontology/create-group")
	t.OntologyGroupDelete = fhttp.UnaryServer[api.OntologyDeleteGroupRequest, types.Nil](router, false, "/api/v1/ontology/delete-group")
	t.OntologyGroupRename = fhttp.UnaryServer[api.OntologyRenameGroupRequest, types.Nil](router, false, "/api/v1/ontology/rename-group")

	// RANGE
	t.RangeRetrieve = fhttp.UnaryServer[api.RangeRetrieveRequest, api.RangeRetrieveResponse](router, false, "/api/v1/range/retrieve")
	t.RangeCreate = fhttp.UnaryServer[api.RangeCreateRequest, api.RangeCreateResponse](router, false, "/api/v1/range/create")
	t.RangeDelete = fhttp.UnaryServer[api.RangeDeleteRequest, types.Nil](router, false, "/api/v1/range/delete")
	t.RangeRename = fhttp.UnaryServer[api.RangeRenameRequest, types.Nil](router, false, "/api/v1/range/rename")
	t.RangeKVGet = fhttp.UnaryServer[api.RangeKVGetRequest, api.RangeKVGetResponse](router, false, "/api/v1/range/kv/get")
	t.RangeKVSet = fhttp.UnaryServer[api.RangeKVSetRequest, types.Nil](router, false, "/api/v1/range/kv/set")
	t.RangeKVDelete = fhttp.UnaryServer[api.RangeKVDeleteRequest, types.Nil](router, false, "/api/v1/range/kv/delete")
	t.RangeAliasSet = fhttp.UnaryServer[api.RangeAliasSetRequest, types.Nil](router, false, "/api/v1/range/alias/set")
	t.RangeAliasResolve = fhttp.UnaryServer[api.RangeAliasResolveRequest, api.RangeAliasResolveResponse](router, false, "/api/v1/range/alias/resolve")
	t.RangeAliasList = fhttp.UnaryServer[api.RangeAliasListRequest, api.RangeAliasListResponse](router, false, "/api/v1/range/alias/list")
	t.RangeAliasDelete = fhttp.UnaryServer[api.RangeAliasDeleteRequest, types.Nil](router, false, "/api/v1/range/alias/delete")
	t.RangeSetActive = fhttp.UnaryServer[api.RangeSetActiveRequest, types.Nil](router, false, "/api/v1/range/set-active")
	t.RangeRetrieveActive = fhttp.UnaryServer[types.Nil, api.RangeRetrieveActiveResponse](router, false, "/api/v1/range/retrieve-active")
	t.RangeClearActive = fhttp.UnaryServer[types.Nil, types.Nil](router, false, "/api/v1/range/clear-active")

	// WORKSPACE
	t.WorkspaceCreate = fhttp.UnaryServer[api.WorkspaceCreateRequest, api.WorkspaceCreateResponse](router, false, "/api/v1/workspace/create")
	t.WorkspaceRetrieve = fhttp.UnaryServer[api.WorkspaceRetrieveRequest, api.WorkspaceRetrieveResponse](router, false, "/api/v1/workspace/retrieve")
	t.WorkspaceDelete = fhttp.UnaryServer[api.WorkspaceDeleteRequest, types.Nil](router, false, "/api/v1/workspace/delete")
	t.WorkspaceRename = fhttp.UnaryServer[api.WorkspaceRenameRequest, types.Nil](router, false, "/api/v1/workspace/rename")
	t.WorkspaceSetLayout = fhttp.UnaryServer[api.WorkspaceSetLayoutRequest, types.Nil](router, false, "/api/v1/workspace/set-layout")

	// Schematic
	t.SchematicCreate = fhttp.UnaryServer[api.SchematicCreateRequest, api.SchematicCreateResponse](router, false, "/api/v1/workspace/schematic/create")
	t.SchematicRetrieve = fhttp.UnaryServer[api.SchematicRetrieveRequest, api.SchematicRetrieveResponse](router, false, "/api/v1/workspace/schematic/retrieve")
	t.SchematicDelete = fhttp.UnaryServer[api.SchematicDeleteRequest, types.Nil](router, false, "/api/v1/workspace/schematic/delete")
	t.SchematicRename = fhttp.UnaryServer[api.SchematicRenameRequest, types.Nil](router, false, "/api/v1/workspace/schematic/rename")
	t.SchematicSetData = fhttp.UnaryServer[api.SchematicSetDataRequest, types.Nil](router, false, "/api/v1/workspace/schematic/set-data")
	t.SchematicCopy = fhttp.UnaryServer[api.SchematicCopyRequest, api.SchematicCopyResponse](router, false, "/api/v1/workspace/schematic/copy")

	// LINE PLOT
	t.LinePlotCreate = fhttp.UnaryServer[api.LinePlotCreateRequest, api.LinePlotCreateResponse](router, false, "/api/v1/workspace/lineplot/create")
	t.LinePlotRetrieve = fhttp.UnaryServer[api.LinePlotRetrieveRequest, api.LinePlotRetrieveResponse](router, false, "/api/v1/workspace/lineplot/retrieve")
	t.LinePlotDelete = fhttp.UnaryServer[api.LinePlotDeleteRequest, types.Nil](router, false, "/api/v1/workspace/lineplot/delete")
	t.LinePlotRename = fhttp.UnaryServer[api.LinePlotRenameRequest, types.Nil](router, false, "/api/v1/workspace/lineplot/rename")
	t.LinePlotSetData = fhttp.UnaryServer[api.LinePlotSetDataRequest, types.Nil](router, false, "/api/v1/workspace/lineplot/set-data")

	// LABEL
	t.LabelCreate = fhttp.UnaryServer[api.LabelCreateRequest, api.LabelCreateResponse](router, false, "/api/v1/label/create")
	t.LabelRetrieve = fhttp.UnaryServer[api.LabelRetrieveRequest, api.LabelRetrieveResponse](router, false, "/api/v1/label/retrieve")
	t.LabelDelete = fhttp.UnaryServer[api.LabelDeleteRequest, types.Nil](router, false, "/api/v1/label/delete")
	t.LabelSet = fhttp.UnaryServer[api.LabelSetRequest, types.Nil](router, false, "/api/v1/label/set")
	t.LabelRemove = fhttp.UnaryServer[api.LabelRemoveRequest, types.Nil](router, false, "/api/v1/label/remove")

	// DEVICE
	t.HardwareCreateRack = fhttp.UnaryServer[api.HardwareCreateRackRequest, api.HardwareCreateRackResponse](router, false, "/api/v1/hardware/rack/create")
	t.HardwareRetrieveRack = fhttp.UnaryServer[api.HardwareRetrieveRackRequest, api.HardwareRetrieveRackResponse](router, false, "/api/v1/hardware/rack/retrieve")
	t.HardwareDeleteRack = fhttp.UnaryServer[api.HardwareDeleteRackRequest, types.Nil](router, false, "/api/v1/hardware/rack/delete")
	t.HardwareCreateTask = fhttp.UnaryServer[api.HardwareCreateTaskRequest, api.HardwareCreateTaskResponse](router, false, "/api/v1/hardware/task/create")
	t.HardwareRetrieveTask = fhttp.UnaryServer[api.HardwareRetrieveTaskRequest, api.HardwareRetrieveTaskResponse](router, false, "/api/v1/hardware/task/retrieve")
	t.HardwareDeleteTask = fhttp.UnaryServer[api.HardwareDeleteTaskRequest, types.Nil](router, false, "/api/v1/hardware/task/delete")
	t.HardwareCreateDevice = fhttp.UnaryServer[api.HardwareCreateDeviceRequest, api.HardwareCreateDeviceResponse](router, false, "/api/v1/hardware/device/create")
	t.HardwareRetrieveDevice = fhttp.UnaryServer[api.HardwareRetrieveDeviceRequest, api.HardwareRetrieveDeviceResponse](router, false, "/api/v1/hardware/device/retrieve")
	t.HardwareDeleteDevice = fhttp.UnaryServer[api.HardwareDeleteDeviceRequest, types.Nil](router, false, "/api/v1/hardware/device/delete")

	return t
}
