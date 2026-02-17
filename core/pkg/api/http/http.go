// Copyright 2026 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/freighter/http"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/x/httputil"
)

func New(router *http.Router, codecResolver httputil.CodecResolver) api.Transport {
	return api.Transport{
		// AUTH
		AuthLogin:          http.UnaryServer[api.AuthLoginRequest, api.AuthLoginResponse](router, "/api/v1/auth/login"),
		AuthChangePassword: http.UnaryServer[api.AuthChangePasswordRequest, types.Nil](router, "/api/v1/auth/change-password"),

		// USER
		UserRename:         http.UnaryServer[api.UserRenameRequest, types.Nil](router, "/api/v1/user/rename"),
		UserChangeUsername: http.UnaryServer[api.UserChangeUsernameRequest, types.Nil](router, "/api/v1/user/change-username"),
		UserCreate:         http.UnaryServer[api.UserCreateRequest, api.UserCreateResponse](router, "/api/v1/user/create"),
		UserDelete:         http.UnaryServer[api.UserDeleteRequest, types.Nil](router, "/api/v1/user/delete"),
		UserRetrieve:       http.UnaryServer[api.UserRetrieveRequest, api.UserRetrieveResponse](router, "/api/v1/user/retrieve"),

		// CHANNEL
		ChannelCreate:        http.UnaryServer[api.ChannelCreateRequest, api.ChannelCreateResponse](router, "/api/v1/channel/create"),
		ChannelRetrieve:      http.UnaryServer[api.ChannelRetrieveRequest, api.ChannelRetrieveResponse](router, "/api/v1/channel/retrieve"),
		ChannelDelete:        http.UnaryServer[api.ChannelDeleteRequest, types.Nil](router, "/api/v1/channel/delete"),
		ChannelRename:        http.UnaryServer[api.ChannelRenameRequest, types.Nil](router, "/api/v1/channel/rename"),
		ChannelRetrieveGroup: http.UnaryServer[api.ChannelRetrieveGroupRequest, api.ChannelRetrieveGroupResponse](router, "/api/v1/channel/retrieve-group"),

		// CONNECTIVITY
		ConnectivityCheck: http.UnaryServer[types.Nil, api.ConnectivityCheckResponse](router, "/api/v1/connectivity/check"),

		// FRAME
		FrameWriter:   http.StreamServer[api.FrameWriterRequest, api.FrameWriterResponse](router, "/api/v1/frame/write", http.WithCodecResolver(codecResolver)),
		FrameIterator: http.StreamServer[api.FrameIteratorRequest, api.FrameIteratorResponse](router, "/api/v1/frame/iterate"),
		FrameStreamer: http.StreamServer[api.FrameStreamerRequest, api.FrameStreamerResponse](router, "/api/v1/frame/stream", http.WithCodecResolver(codecResolver)),
		FrameDelete:   http.UnaryServer[api.FrameDeleteRequest, types.Nil](router, "/api/v1/frame/delete"),

		// ONTOLOGY
		OntologyRetrieve:       http.UnaryServer[api.OntologyRetrieveRequest, api.OntologyRetrieveResponse](router, "/api/v1/ontology/retrieve"),
		OntologyAddChildren:    http.UnaryServer[api.OntologyAddChildrenRequest, types.Nil](router, "/api/v1/ontology/add-children"),
		OntologyRemoveChildren: http.UnaryServer[api.OntologyRemoveChildrenRequest, types.Nil](router, "/api/v1/ontology/remove-children"),
		OntologyMoveChildren:   http.UnaryServer[api.OntologyMoveChildrenRequest, types.Nil](router, "/api/v1/ontology/move-children"),

		// GROUP
		GroupCreate: http.UnaryServer[api.GroupCreateRequest, api.GroupCreateResponse](router, "/api/v1/ontology/create-group"),
		GroupDelete: http.UnaryServer[api.GroupDeleteRequest, types.Nil](router, "/api/v1/ontology/delete-group"),
		GroupRename: http.UnaryServer[api.GroupRenameRequest, types.Nil](router, "/api/v1/ontology/rename-group"),

		// RANGE
		RangeRetrieve:      http.UnaryServer[api.RangeRetrieveRequest, api.RangeRetrieveResponse](router, "/api/v1/range/retrieve"),
		RangeCreate:        http.UnaryServer[api.RangeCreateRequest, api.RangeCreateResponse](router, "/api/v1/range/create"),
		RangeDelete:        http.UnaryServer[api.RangeDeleteRequest, types.Nil](router, "/api/v1/range/delete"),
		RangeRename:        http.UnaryServer[api.RangeRenameRequest, types.Nil](router, "/api/v1/range/rename"),
		RangeKVGet:         http.UnaryServer[api.RangeKVGetRequest, api.RangeKVGetResponse](router, "/api/v1/range/kv/get"),
		RangeKVSet:         http.UnaryServer[api.RangeKVSetRequest, types.Nil](router, "/api/v1/range/kv/set"),
		RangeKVDelete:      http.UnaryServer[api.RangeKVDeleteRequest, types.Nil](router, "/api/v1/range/kv/delete"),
		RangeAliasSet:      http.UnaryServer[api.RangeAliasSetRequest, types.Nil](router, "/api/v1/range/alias/set"),
		RangeAliasResolve:  http.UnaryServer[api.RangeAliasResolveRequest, api.RangeAliasResolveResponse](router, "/api/v1/range/alias/resolve"),
		RangeAliasRetrieve: http.UnaryServer[api.RangeAliasRetrieveRequest, api.RangeAliasRetrieveResponse](router, "/api/v1/range/alias/retrieve"),
		RangeAliasList:     http.UnaryServer[api.RangeAliasListRequest, api.RangeAliasListResponse](router, "/api/v1/range/alias/list"),
		RangeAliasDelete:   http.UnaryServer[api.RangeAliasDeleteRequest, types.Nil](router, "/api/v1/range/alias/delete"),

		// WORKSPACE
		WorkspaceCreate:    http.UnaryServer[api.WorkspaceCreateRequest, api.WorkspaceCreateResponse](router, "/api/v1/workspace/create"),
		WorkspaceRetrieve:  http.UnaryServer[api.WorkspaceRetrieveRequest, api.WorkspaceRetrieveResponse](router, "/api/v1/workspace/retrieve"),
		WorkspaceDelete:    http.UnaryServer[api.WorkspaceDeleteRequest, types.Nil](router, "/api/v1/workspace/delete"),
		WorkspaceRename:    http.UnaryServer[api.WorkspaceRenameRequest, types.Nil](router, "/api/v1/workspace/rename"),
		WorkspaceSetLayout: http.UnaryServer[api.WorkspaceSetLayoutRequest, types.Nil](router, "/api/v1/workspace/set-layout"),

		// SCHEMATIC
		SchematicCreate:   http.UnaryServer[api.SchematicCreateRequest, api.SchematicCreateResponse](router, "/api/v1/workspace/schematic/create"),
		SchematicRetrieve: http.UnaryServer[api.SchematicRetrieveRequest, api.SchematicRetrieveResponse](router, "/api/v1/workspace/schematic/retrieve"),
		SchematicDelete:   http.UnaryServer[api.SchematicDeleteRequest, types.Nil](router, "/api/v1/workspace/schematic/delete"),
		SchematicRename:   http.UnaryServer[api.SchematicRenameRequest, types.Nil](router, "/api/v1/workspace/schematic/rename"),
		SchematicSetData:  http.UnaryServer[api.SchematicSetDataRequest, types.Nil](router, "/api/v1/workspace/schematic/set-data"),
		SchematicCopy:     http.UnaryServer[api.SchematicCopyRequest, api.SchematicCopyResponse](router, "/api/v1/workspace/schematic/copy"),

		// SCHEMATIC SYMBOL
		SchematicCreateSymbol:        http.UnaryServer[api.SchematicCreateSymbolRequest, api.SchematicCreateSymbolResponse](router, "/api/v1/workspace/schematic/symbol/create"),
		SchematicRetrieveSymbol:      http.UnaryServer[api.SchematicRetrieveSymbolRequest, api.SchematicRetrieveSymbolResponse](router, "/api/v1/workspace/schematic/symbol/retrieve"),
		SchematicDeleteSymbol:        http.UnaryServer[api.SchematicDeleteSymbolRequest, types.Nil](router, "/api/v1/workspace/schematic/symbol/delete"),
		SchematicRenameSymbol:        http.UnaryServer[api.SchematicRenameSymbolRequest, types.Nil](router, "/api/v1/workspace/schematic/symbol/rename"),
		SchematicRetrieveSymbolGroup: http.UnaryServer[api.SchematicRetrieveSymbolGroupRequest, api.SchematicRetrieveSymbolGroupResponse](router, "/api/v1/workspace/schematic/symbol/retrieve_group"),

		// LINE PLOT
		LinePlotCreate:   http.UnaryServer[api.LinePlotCreateRequest, api.LinePlotCreateResponse](router, "/api/v1/workspace/lineplot/create"),
		LinePlotRetrieve: http.UnaryServer[api.LinePlotRetrieveRequest, api.LinePlotRetrieveResponse](router, "/api/v1/workspace/lineplot/retrieve"),
		LinePlotDelete:   http.UnaryServer[api.LinePlotDeleteRequest, types.Nil](router, "/api/v1/workspace/lineplot/delete"),
		LinePlotRename:   http.UnaryServer[api.LinePlotRenameRequest, types.Nil](router, "/api/v1/workspace/lineplot/rename"),
		LinePlotSetData:  http.UnaryServer[api.LinePlotSetDataRequest, types.Nil](router, "/api/v1/workspace/lineplot/set-data"),

		// LOG
		LogCreate:   http.UnaryServer[api.LogCreateRequest, api.LogCreateResponse](router, "/api/v1/workspace/log/create"),
		LogRetrieve: http.UnaryServer[api.LogRetrieveRequest, api.LogRetrieveResponse](router, "/api/v1/workspace/log/retrieve"),
		LogDelete:   http.UnaryServer[api.LogDeleteRequest, types.Nil](router, "/api/v1/workspace/log/delete"),
		LogRename:   http.UnaryServer[api.LogRenameRequest, types.Nil](router, "/api/v1/workspace/log/rename"),
		LogSetData:  http.UnaryServer[api.LogSetDataRequest, types.Nil](router, "/api/v1/workspace/log/set-data"),

		// TABLE
		TableCreate:   http.UnaryServer[api.TableCreateRequest, api.TableCreateResponse](router, "/api/v1/workspace/table/create"),
		TableRetrieve: http.UnaryServer[api.TableRetrieveRequest, api.TableRetrieveResponse](router, "/api/v1/workspace/table/retrieve"),
		TableDelete:   http.UnaryServer[api.TableDeleteRequest, types.Nil](router, "/api/v1/workspace/table/delete"),
		TableRename:   http.UnaryServer[api.TableRenameRequest, types.Nil](router, "/api/v1/workspace/table/rename"),
		TableSetData:  http.UnaryServer[api.TableSetDataRequest, types.Nil](router, "/api/v1/workspace/table/set-data"),

		// LABEL
		LabelCreate:   http.UnaryServer[api.LabelCreateRequest, api.LabelCreateResponse](router, "/api/v1/label/create"),
		LabelRetrieve: http.UnaryServer[api.LabelRetrieveRequest, api.LabelRetrieveResponse](router, "/api/v1/label/retrieve"),
		LabelDelete:   http.UnaryServer[api.LabelDeleteRequest, types.Nil](router, "/api/v1/label/delete"),
		LabelAdd:      http.UnaryServer[api.LabelAddRequest, types.Nil](router, "/api/v1/label/set"),
		LabelRemove:   http.UnaryServer[api.LabelRemoveRequest, types.Nil](router, "/api/v1/label/remove"),

		// RACK
		RackCreate:   http.UnaryServer[api.RackCreateRequest, api.RackCreateResponse](router, "/api/v1/rack/create"),
		RackRetrieve: http.UnaryServer[api.RackRetrieveRequest, api.RackRetrieveResponse](router, "/api/v1/rack/retrieve"),
		RackDelete:   http.UnaryServer[api.RackDeleteRequest, types.Nil](router, "/api/v1/rack/delete"),

		// TASK
		TaskCreate:   http.UnaryServer[api.TaskCreateRequest, api.TaskCreateResponse](router, "/api/v1/task/create"),
		TaskRetrieve: http.UnaryServer[api.TaskRetrieveRequest, api.TaskRetrieveResponse](router, "/api/v1/task/retrieve"),
		TaskDelete:   http.UnaryServer[api.TaskDeleteRequest, types.Nil](router, "/api/v1/task/delete"),
		TaskCopy:     http.UnaryServer[api.TaskCopyRequest, api.TaskCopyResponse](router, "/api/v1/task/copy"),

		// DEVICE
		DeviceCreate:   http.UnaryServer[api.DeviceCreateRequest, api.DeviceCreateResponse](router, "/api/v1/device/create"),
		DeviceRetrieve: http.UnaryServer[api.DeviceRetrieveRequest, api.DeviceRetrieveResponse](router, "/api/v1/device/retrieve"),
		DeviceDelete:   http.UnaryServer[api.DeviceDeleteRequest, types.Nil](router, "/api/v1/device/delete"),

		// ACCESS
		AccessCreatePolicy:   http.UnaryServer[api.AccessCreatePolicyRequest, api.AccessCreatePolicyResponse](router, "/api/v1/access/policy/create"),
		AccessDeletePolicy:   http.UnaryServer[api.AccessDeletePolicyRequest, types.Nil](router, "/api/v1/access/policy/delete"),
		AccessRetrievePolicy: http.UnaryServer[api.AccessRetrievePolicyRequest, api.AccessRetrievePolicyResponse](router, "/api/v1/access/policy/retrieve"),
		AccessCreateRole:     http.UnaryServer[api.AccessCreateRoleRequest, api.AccessCreateRoleResponse](router, "/api/v1/access/role/create"),
		AccessDeleteRole:     http.UnaryServer[api.AccessDeleteRoleRequest, types.Nil](router, "/api/v1/access/role/delete"),
		AccessRetrieveRole:   http.UnaryServer[api.AccessRetrieveRoleRequest, api.AccessRetrieveRoleResponse](router, "/api/v1/access/role/retrieve"),
		AccessAssignRole:     http.UnaryServer[api.AccessAssignRoleRequest, types.Nil](router, "/api/v1/access/role/assign"),
		AccessUnassignRole:   http.UnaryServer[api.AccessUnassignRoleRequest, types.Nil](router, "/api/v1/access/role/unassign"),

		// ARC
		ArcCreate:   http.UnaryServer[api.ArcCreateRequest, api.ArcCreateResponse](router, "/api/v1/arc/create"),
		ArcDelete:   http.UnaryServer[api.ArcDeleteRequest, types.Nil](router, "/api/v1/arc/delete"),
		ArcRetrieve: http.UnaryServer[api.ArcRetrieveRequest, api.ArcRetrieveResponse](router, "/api/v1/arc/retrieve"),
		ArcLSP:      http.StreamServer[api.ArcLSPMessage, api.ArcLSPMessage](router, "/api/v1/arc/lsp"),

		// STATUS
		StatusSet:      http.UnaryServer[api.StatusSetRequest, api.StatusSetResponse](router, "/api/v1/status/set"),
		StatusRetrieve: http.UnaryServer[api.StatusRetrieveRequest, api.StatusRetrieveResponse](router, "/api/v1/status/retrieve"),
		StatusDelete:   http.UnaryServer[api.StatusDeleteRequest, types.Nil](router, "/api/v1/status/delete"),

		// VIEW
		ViewCreate:   http.UnaryServer[api.ViewCreateRequest, api.ViewCreateResponse](router, "/api/v1/view/create"),
		ViewRetrieve: http.UnaryServer[api.ViewRetrieveRequest, api.ViewRetrieveResponse](router, "/api/v1/view/retrieve"),
		ViewDelete:   http.UnaryServer[api.ViewDeleteRequest, types.Nil](router, "/api/v1/view/delete"),
	}
}
