// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/x/httputil"
)

func New(router *fhttp.Router, codecResolver httputil.CodecResolver) (t api.Transport) {
	// AUTH
	t.AuthLogin = fhttp.UnaryServer[api.AuthLoginRequest, api.AuthLoginResponse](router, "/api/v1/auth/login")
	t.AuthChangePassword = fhttp.UnaryServer[api.AuthChangePasswordRequest, types.Nil](router, "/api/v1/auth/change-password")

	// USER
	t.UserRename = fhttp.UnaryServer[api.UserRenameRequest, types.Nil](router, "/api/v1/user/rename")
	t.UserChangeUsername = fhttp.UnaryServer[api.UserChangeUsernameRequest, types.Nil](router, "/api/v1/user/change-username")
	t.UserCreate = fhttp.UnaryServer[api.UserCreateRequest, api.UserCreateResponse](router, "/api/v1/user/create")
	t.UserDelete = fhttp.UnaryServer[api.UserDeleteRequest, types.Nil](router, "/api/v1/user/delete")
	t.UserRetrieve = fhttp.UnaryServer[api.UserRetrieveRequest, api.UserRetrieveResponse](router, "/api/v1/user/retrieve")

	// CHANNEL
	t.ChannelCreate = fhttp.UnaryServer[api.ChannelCreateRequest, api.ChannelCreateResponse](router, "/api/v1/channel/create")
	t.ChannelRetrieve = fhttp.UnaryServer[api.ChannelRetrieveRequest, api.ChannelRetrieveResponse](router, "/api/v1/channel/retrieve")
	t.ChannelDelete = fhttp.UnaryServer[api.ChannelDeleteRequest, types.Nil](router, "/api/v1/channel/delete")
	t.ChannelRename = fhttp.UnaryServer[api.ChannelRenameRequest, types.Nil](router, "/api/v1/channel/rename")
	t.ChannelRetrieveGroup = fhttp.UnaryServer[api.ChannelRetrieveGroupRequest, api.ChannelRetrieveGroupResponse](router, "/api/v1/channel/retrieve-group")

	// CONNECTIVITY
	t.ConnectivityCheck = fhttp.UnaryServer[types.Nil, api.ConnectivityCheckResponse](router, "/api/v1/connectivity/check")

	// FRAME
	t.FrameWriter = fhttp.StreamServer[api.FrameWriterRequest, api.FrameWriterResponse](router, "/api/v1/frame/write", fhttp.WithCodecResolver(codecResolver))
	t.FrameIterator = fhttp.StreamServer[api.FrameIteratorRequest, api.FrameIteratorResponse](router, "/api/v1/frame/iterate")
	t.FrameStreamer = fhttp.StreamServer[api.FrameStreamerRequest, api.FrameStreamerResponse](router, "/api/v1/frame/stream", fhttp.WithCodecResolver(codecResolver))
	t.FrameDelete = fhttp.UnaryServer[api.FrameDeleteRequest, types.Nil](router, "/api/v1/frame/delete")

	// ONTOLOGY
	t.OntologyRetrieve = fhttp.UnaryServer[api.OntologyRetrieveRequest, api.OntologyRetrieveResponse](router, "/api/v1/ontology/retrieve")
	t.OntologyAddChildren = fhttp.UnaryServer[api.OntologyAddChildrenRequest, types.Nil](router, "/api/v1/ontology/add-children")
	t.OntologyRemoveChildren = fhttp.UnaryServer[api.OntologyRemoveChildrenRequest, types.Nil](router, "/api/v1/ontology/remove-children")
	t.OntologyMoveChildren = fhttp.UnaryServer[api.OntologyMoveChildrenRequest, types.Nil](router, "/api/v1/ontology/move-children")

	// GROUP
	t.OntologyGroupCreate = fhttp.UnaryServer[api.OntologyCreateGroupRequest, api.OntologyCreateGroupResponse](router, "/api/v1/ontology/create-group")
	t.OntologyGroupDelete = fhttp.UnaryServer[api.OntologyDeleteGroupRequest, types.Nil](router, "/api/v1/ontology/delete-group")
	t.OntologyGroupRename = fhttp.UnaryServer[api.OntologyRenameGroupRequest, types.Nil](router, "/api/v1/ontology/rename-group")

	// RANGE
	t.RangeRetrieve = fhttp.UnaryServer[api.RangeRetrieveRequest, api.RangeRetrieveResponse](router, "/api/v1/range/retrieve")
	t.RangeCreate = fhttp.UnaryServer[api.RangeCreateRequest, api.RangeCreateResponse](router, "/api/v1/range/create")
	t.RangeDelete = fhttp.UnaryServer[api.RangeDeleteRequest, types.Nil](router, "/api/v1/range/delete")
	t.RangeRename = fhttp.UnaryServer[api.RangeRenameRequest, types.Nil](router, "/api/v1/range/rename")
	t.RangeKVGet = fhttp.UnaryServer[api.RangeKVGetRequest, api.RangeKVGetResponse](router, "/api/v1/range/kv/get")
	t.RangeKVSet = fhttp.UnaryServer[api.RangeKVSetRequest, types.Nil](router, "/api/v1/range/kv/set")
	t.RangeKVDelete = fhttp.UnaryServer[api.RangeKVDeleteRequest, types.Nil](router, "/api/v1/range/kv/delete")
	t.RangeAliasSet = fhttp.UnaryServer[api.RangeAliasSetRequest, types.Nil](router, "/api/v1/range/alias/set")
	t.RangeAliasResolve = fhttp.UnaryServer[api.RangeAliasResolveRequest, api.RangeAliasResolveResponse](router, "/api/v1/range/alias/resolve")
	t.RangeAliasRetrieve = fhttp.UnaryServer[api.RangeAliasRetrieveRequest, api.RangeAliasRetrieveResponse](router, "/api/v1/range/alias/retrieve")
	t.RangeAliasList = fhttp.UnaryServer[api.RangeAliasListRequest, api.RangeAliasListResponse](router, "/api/v1/range/alias/list")
	t.RangeAliasDelete = fhttp.UnaryServer[api.RangeAliasDeleteRequest, types.Nil](router, "/api/v1/range/alias/delete")

	// WORKSPACE
	t.WorkspaceCreate = fhttp.UnaryServer[api.WorkspaceCreateRequest, api.WorkspaceCreateResponse](router, "/api/v1/workspace/create")
	t.WorkspaceRetrieve = fhttp.UnaryServer[api.WorkspaceRetrieveRequest, api.WorkspaceRetrieveResponse](router, "/api/v1/workspace/retrieve")
	t.WorkspaceDelete = fhttp.UnaryServer[api.WorkspaceDeleteRequest, types.Nil](router, "/api/v1/workspace/delete")
	t.WorkspaceRename = fhttp.UnaryServer[api.WorkspaceRenameRequest, types.Nil](router, "/api/v1/workspace/rename")
	t.WorkspaceSetLayout = fhttp.UnaryServer[api.WorkspaceSetLayoutRequest, types.Nil](router, "/api/v1/workspace/set-layout")

	// SCHEMATIC
	t.SchematicCreate = fhttp.UnaryServer[api.SchematicCreateRequest, api.SchematicCreateResponse](router, "/api/v1/workspace/schematic/create")
	t.SchematicRetrieve = fhttp.UnaryServer[api.SchematicRetrieveRequest, api.SchematicRetrieveResponse](router, "/api/v1/workspace/schematic/retrieve")
	t.SchematicDelete = fhttp.UnaryServer[api.SchematicDeleteRequest, types.Nil](router, "/api/v1/workspace/schematic/delete")
	t.SchematicRename = fhttp.UnaryServer[api.SchematicRenameRequest, types.Nil](router, "/api/v1/workspace/schematic/rename")
	t.SchematicSetData = fhttp.UnaryServer[api.SchematicSetDataRequest, types.Nil](router, "/api/v1/workspace/schematic/set-data")
	t.SchematicCopy = fhttp.UnaryServer[api.SchematicCopyRequest, api.SchematicCopyResponse](router, "/api/v1/workspace/schematic/copy")

	// SCHEMATIC SYMBOL
	t.SchematicSymbolCreate = fhttp.UnaryServer[api.SymbolCreateRequest, api.SymbolCreateResponse](router, "/api/v1/workspace/schematic/symbol/create")
	t.SchematicSymbolRetrieve = fhttp.UnaryServer[api.SymbolRetrieveRequest, api.SymbolRetrieveResponse](router, "/api/v1/workspace/schematic/symbol/retrieve")
	t.SchematicSymbolDelete = fhttp.UnaryServer[api.SymbolDeleteRequest, types.Nil](router, "/api/v1/workspace/schematic/symbol/delete")
	t.SchematicSymbolRename = fhttp.UnaryServer[api.SymbolRenameRequest, types.Nil](router, "/api/v1/workspace/schematic/symbol/rename")
	t.SchematicSymbolRetrieveGroup = fhttp.UnaryServer[api.SymbolRetrieveGroupRequest, api.SymbolRetrieveGroupResponse](router, "/api/v1/workspace/schematic/symbol/retrieve_group")

	// LINE PLOT
	t.LinePlotCreate = fhttp.UnaryServer[api.LinePlotCreateRequest, api.LinePlotCreateResponse](router, "/api/v1/workspace/lineplot/create")
	t.LinePlotRetrieve = fhttp.UnaryServer[api.LinePlotRetrieveRequest, api.LinePlotRetrieveResponse](router, "/api/v1/workspace/lineplot/retrieve")
	t.LinePlotDelete = fhttp.UnaryServer[api.LinePlotDeleteRequest, types.Nil](router, "/api/v1/workspace/lineplot/delete")
	t.LinePlotRename = fhttp.UnaryServer[api.LinePlotRenameRequest, types.Nil](router, "/api/v1/workspace/lineplot/rename")
	t.LinePlotSetData = fhttp.UnaryServer[api.LinePlotSetDataRequest, types.Nil](router, "/api/v1/workspace/lineplot/set-data")

	// LOG
	t.LogCreate = fhttp.UnaryServer[api.LogCreateRequest, api.LogCreateResponse](router, "/api/v1/workspace/log/create")
	t.LogRetrieve = fhttp.UnaryServer[api.LogRetrieveRequest, api.LogRetrieveResponse](router, "/api/v1/workspace/log/retrieve")
	t.LogDelete = fhttp.UnaryServer[api.LogDeleteRequest, types.Nil](router, "/api/v1/workspace/log/delete")
	t.LogRename = fhttp.UnaryServer[api.LogRenameRequest, types.Nil](router, "/api/v1/workspace/log/rename")
	t.LogSetData = fhttp.UnaryServer[api.LogSetDataRequest, types.Nil](router, "/api/v1/workspace/log/set-data")

	// TABLE
	t.TableCreate = fhttp.UnaryServer[api.TableCreateRequest, api.TableCreateResponse](router, "/api/v1/workspace/table/create")
	t.TableRetrieve = fhttp.UnaryServer[api.TableRetrieveRequest, api.TableRetrieveResponse](router, "/api/v1/workspace/table/retrieve")
	t.TableDelete = fhttp.UnaryServer[api.TableDeleteRequest, types.Nil](router, "/api/v1/workspace/table/delete")
	t.TableRename = fhttp.UnaryServer[api.TableRenameRequest, types.Nil](router, "/api/v1/workspace/table/rename")
	t.TableSetData = fhttp.UnaryServer[api.TableSetDataRequest, types.Nil](router, "/api/v1/workspace/table/set-data")

	// LABEL
	t.LabelCreate = fhttp.UnaryServer[api.LabelCreateRequest, api.LabelCreateResponse](router, "/api/v1/label/create")
	t.LabelRetrieve = fhttp.UnaryServer[api.LabelRetrieveRequest, api.LabelRetrieveResponse](router, "/api/v1/label/retrieve")
	t.LabelDelete = fhttp.UnaryServer[api.LabelDeleteRequest, types.Nil](router, "/api/v1/label/delete")
	t.LabelAdd = fhttp.UnaryServer[api.LabelAddRequest, types.Nil](router, "/api/v1/label/set")
	t.LabelRemove = fhttp.UnaryServer[api.LabelRemoveRequest, types.Nil](router, "/api/v1/label/remove")

	// HARDWARE
	t.HardwareCreateRack = fhttp.UnaryServer[api.HardwareCreateRackRequest, api.HardwareCreateRackResponse](router, "/api/v1/hardware/rack/create")
	t.HardwareRetrieveRack = fhttp.UnaryServer[api.HardwareRetrieveRackRequest, api.HardwareRetrieveRackResponse](router, "/api/v1/hardware/rack/retrieve")
	t.HardwareDeleteRack = fhttp.UnaryServer[api.HardwareDeleteRackRequest, types.Nil](router, "/api/v1/hardware/rack/delete")
	t.HardwareCreateTask = fhttp.UnaryServer[api.HardwareCreateTaskRequest, api.HardwareCreateTaskResponse](router, "/api/v1/hardware/task/create")
	t.HardwareRetrieveTask = fhttp.UnaryServer[api.HardwareRetrieveTaskRequest, api.HardwareRetrieveTaskResponse](router, "/api/v1/hardware/task/retrieve")
	t.HardwareDeleteTask = fhttp.UnaryServer[api.HardwareDeleteTaskRequest, types.Nil](router, "/api/v1/hardware/task/delete")
	t.HardwareCopyTask = fhttp.UnaryServer[api.HardwareCopyTaskRequest, api.HardwareCopyTaskResponse](router, "/api/v1/hardware/task/copy")
	t.HardwareCreateDevice = fhttp.UnaryServer[api.HardwareCreateDeviceRequest, api.HardwareCreateDeviceResponse](router, "/api/v1/hardware/device/create")
	t.HardwareRetrieveDevice = fhttp.UnaryServer[api.HardwareRetrieveDeviceRequest, api.HardwareRetrieveDeviceResponse](router, "/api/v1/hardware/device/retrieve")
	t.HardwareDeleteDevice = fhttp.UnaryServer[api.HardwareDeleteDeviceRequest, types.Nil](router, "/api/v1/hardware/device/delete")

	// ACCESS
	t.AccessCreatePolicy = fhttp.UnaryServer[api.AccessCreatePolicyRequest, api.AccessCreatePolicyResponse](router, "/api/v1/access/policy/create")
	t.AccessDeletePolicy = fhttp.UnaryServer[api.AccessDeletePolicyRequest, types.Nil](router, "/api/v1/access/policy/delete")
	t.AccessRetrievePolicy = fhttp.UnaryServer[api.AccessRetrievePolicyRequest, api.AccessRetrievePolicyResponse](router, "/api/v1/access/policy/retrieve")

	// ARC
	t.ArcCreate = fhttp.UnaryServer[api.ArcCreateRequest, api.ArcCreateResponse](router, "/api/v1/arc/create")
	t.ArcDelete = fhttp.UnaryServer[api.ArcDeleteRequest, types.Nil](router, "/api/v1/arc/delete")
	t.ArcRetrieve = fhttp.UnaryServer[api.ArcRetrieveRequest, api.ArcRetrieveResponse](router, "/api/v1/arc/retrieve")

	// STATUS
	t.StatusSet = fhttp.UnaryServer[api.StatusSetRequest, api.StatusSetResponse](router, "/api/v1/status/set")
	t.StatusRetrieve = fhttp.UnaryServer[api.StatusRetrieveRequest, api.StatusRetrieveResponse](router, "/api/v1/status/retrieve")
	t.StatusDelete = fhttp.UnaryServer[api.StatusDeleteRequest, types.Nil](router, "/api/v1/status/delete")

	// VIEW
	t.ViewCreate = fhttp.UnaryServer[api.ViewCreateRequest, api.ViewCreateResponse](
		router,
		"/api/v1/view/create",
	)
	t.ViewRetrieve = fhttp.UnaryServer[
		api.ViewRetrieveRequest,
		api.ViewRetrieveResponse,
	](router, "/api/v1/view/retrieve")
	t.ViewDelete = fhttp.UnaryServer[api.ViewDeleteRequest, types.Nil](
		router,
		"/api/v1/view/delete",
	)

	return t
}
