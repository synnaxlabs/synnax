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

func New(router *fhttp.Router, codecResolver httputil.CodecResolver) api.Transport {
	return api.Transport{
		// AUTH
		AuthLogin:          fhttp.UnaryServer[api.AuthLoginRequest, api.AuthLoginResponse](router, "/api/v1/auth/login"),
		AuthChangePassword: fhttp.UnaryServer[api.AuthChangePasswordRequest, types.Nil](router, "/api/v1/auth/change-password"),

		// USER
		UserRename:         fhttp.UnaryServer[api.UserRenameRequest, types.Nil](router, "/api/v1/user/rename"),
		UserChangeUsername: fhttp.UnaryServer[api.UserChangeUsernameRequest, types.Nil](router, "/api/v1/user/change-username"),
		UserCreate:         fhttp.UnaryServer[api.UserCreateRequest, api.UserCreateResponse](router, "/api/v1/user/create"),
		UserDelete:         fhttp.UnaryServer[api.UserDeleteRequest, types.Nil](router, "/api/v1/user/delete"),
		UserRetrieve:       fhttp.UnaryServer[api.UserRetrieveRequest, api.UserRetrieveResponse](router, "/api/v1/user/retrieve"),

		// CHANNEL
		ChannelCreate:        fhttp.UnaryServer[api.ChannelCreateRequest, api.ChannelCreateResponse](router, "/api/v1/channel/create"),
		ChannelRetrieve:      fhttp.UnaryServer[api.ChannelRetrieveRequest, api.ChannelRetrieveResponse](router, "/api/v1/channel/retrieve"),
		ChannelDelete:        fhttp.UnaryServer[api.ChannelDeleteRequest, types.Nil](router, "/api/v1/channel/delete"),
		ChannelRename:        fhttp.UnaryServer[api.ChannelRenameRequest, types.Nil](router, "/api/v1/channel/rename"),
		ChannelRetrieveGroup: fhttp.UnaryServer[api.ChannelRetrieveGroupRequest, api.ChannelRetrieveGroupResponse](router, "/api/v1/channel/retrieve-group"),

		// CONNECTIVITY
		ConnectivityCheck: fhttp.UnaryServer[types.Nil, api.ConnectivityCheckResponse](router, "/api/v1/connectivity/check"),

		// FRAME
		FrameWriter:   fhttp.StreamServer[api.FrameWriterRequest, api.FrameWriterResponse](router, "/api/v1/frame/write", fhttp.WithCodecResolver(codecResolver)),
		FrameIterator: fhttp.StreamServer[api.FrameIteratorRequest, api.FrameIteratorResponse](router, "/api/v1/frame/iterate"),
		FrameStreamer: fhttp.StreamServer[api.FrameStreamerRequest, api.FrameStreamerResponse](router, "/api/v1/frame/stream", fhttp.WithCodecResolver(codecResolver)),
		FrameDelete:   fhttp.UnaryServer[api.FrameDeleteRequest, types.Nil](router, "/api/v1/frame/delete"),

		// ONTOLOGY
		OntologyRetrieve:       fhttp.UnaryServer[api.OntologyRetrieveRequest, api.OntologyRetrieveResponse](router, "/api/v1/ontology/retrieve"),
		OntologyAddChildren:    fhttp.UnaryServer[api.OntologyAddChildrenRequest, types.Nil](router, "/api/v1/ontology/add-children"),
		OntologyRemoveChildren: fhttp.UnaryServer[api.OntologyRemoveChildrenRequest, types.Nil](router, "/api/v1/ontology/remove-children"),
		OntologyMoveChildren:   fhttp.UnaryServer[api.OntologyMoveChildrenRequest, types.Nil](router, "/api/v1/ontology/move-children"),

		// GROUP
		GroupCreate: fhttp.UnaryServer[api.GroupCreateRequest, api.GroupCreateResponse](router, "/api/v1/ontology/create-group"),
		GroupDelete: fhttp.UnaryServer[api.GroupDeleteRequest, types.Nil](router, "/api/v1/ontology/delete-group"),
		GroupRename: fhttp.UnaryServer[api.GroupRenameRequest, types.Nil](router, "/api/v1/ontology/rename-group"),

		// RANGE
		RangeRetrieve:      fhttp.UnaryServer[api.RangeRetrieveRequest, api.RangeRetrieveResponse](router, "/api/v1/range/retrieve"),
		RangeCreate:        fhttp.UnaryServer[api.RangeCreateRequest, api.RangeCreateResponse](router, "/api/v1/range/create"),
		RangeDelete:        fhttp.UnaryServer[api.RangeDeleteRequest, types.Nil](router, "/api/v1/range/delete"),
		RangeRename:        fhttp.UnaryServer[api.RangeRenameRequest, types.Nil](router, "/api/v1/range/rename"),
		RangeKVGet:         fhttp.UnaryServer[api.RangeKVGetRequest, api.RangeKVGetResponse](router, "/api/v1/range/kv/get"),
		RangeKVSet:         fhttp.UnaryServer[api.RangeKVSetRequest, types.Nil](router, "/api/v1/range/kv/set"),
		RangeKVDelete:      fhttp.UnaryServer[api.RangeKVDeleteRequest, types.Nil](router, "/api/v1/range/kv/delete"),
		RangeAliasSet:      fhttp.UnaryServer[api.RangeAliasSetRequest, types.Nil](router, "/api/v1/range/alias/set"),
		RangeAliasResolve:  fhttp.UnaryServer[api.RangeAliasResolveRequest, api.RangeAliasResolveResponse](router, "/api/v1/range/alias/resolve"),
		RangeAliasRetrieve: fhttp.UnaryServer[api.RangeAliasRetrieveRequest, api.RangeAliasRetrieveResponse](router, "/api/v1/range/alias/retrieve"),
		RangeAliasList:     fhttp.UnaryServer[api.RangeAliasListRequest, api.RangeAliasListResponse](router, "/api/v1/range/alias/list"),
		RangeAliasDelete:   fhttp.UnaryServer[api.RangeAliasDeleteRequest, types.Nil](router, "/api/v1/range/alias/delete"),

		// WORKSPACE
		WorkspaceCreate:    fhttp.UnaryServer[api.WorkspaceCreateRequest, api.WorkspaceCreateResponse](router, "/api/v1/workspace/create"),
		WorkspaceRetrieve:  fhttp.UnaryServer[api.WorkspaceRetrieveRequest, api.WorkspaceRetrieveResponse](router, "/api/v1/workspace/retrieve"),
		WorkspaceDelete:    fhttp.UnaryServer[api.WorkspaceDeleteRequest, types.Nil](router, "/api/v1/workspace/delete"),
		WorkspaceRename:    fhttp.UnaryServer[api.WorkspaceRenameRequest, types.Nil](router, "/api/v1/workspace/rename"),
		WorkspaceSetLayout: fhttp.UnaryServer[api.WorkspaceSetLayoutRequest, types.Nil](router, "/api/v1/workspace/set-layout"),

		// SCHEMATIC
		SchematicCreate:   fhttp.UnaryServer[api.SchematicCreateRequest, api.SchematicCreateResponse](router, "/api/v1/workspace/schematic/create"),
		SchematicRetrieve: fhttp.UnaryServer[api.SchematicRetrieveRequest, api.SchematicRetrieveResponse](router, "/api/v1/workspace/schematic/retrieve"),
		SchematicDelete:   fhttp.UnaryServer[api.SchematicDeleteRequest, types.Nil](router, "/api/v1/workspace/schematic/delete"),
		SchematicRename:   fhttp.UnaryServer[api.SchematicRenameRequest, types.Nil](router, "/api/v1/workspace/schematic/rename"),
		SchematicSetData:  fhttp.UnaryServer[api.SchematicSetDataRequest, types.Nil](router, "/api/v1/workspace/schematic/set-data"),
		SchematicCopy:     fhttp.UnaryServer[api.SchematicCopyRequest, api.SchematicCopyResponse](router, "/api/v1/workspace/schematic/copy"),

		// SCHEMATIC SYMBOL
		SchematicSymbolCreate:        fhttp.UnaryServer[api.SymbolCreateRequest, api.SymbolCreateResponse](router, "/api/v1/workspace/schematic/symbol/create"),
		SchematicSymbolRetrieve:      fhttp.UnaryServer[api.SymbolRetrieveRequest, api.SymbolRetrieveResponse](router, "/api/v1/workspace/schematic/symbol/retrieve"),
		SchematicSymbolDelete:        fhttp.UnaryServer[api.SymbolDeleteRequest, types.Nil](router, "/api/v1/workspace/schematic/symbol/delete"),
		SchematicSymbolRename:        fhttp.UnaryServer[api.SymbolRenameRequest, types.Nil](router, "/api/v1/workspace/schematic/symbol/rename"),
		SchematicSymbolRetrieveGroup: fhttp.UnaryServer[api.SymbolRetrieveGroupRequest, api.SymbolRetrieveGroupResponse](router, "/api/v1/workspace/schematic/symbol/retrieve_group"),

		// LINE PLOT
		LinePlotCreate:   fhttp.UnaryServer[api.LinePlotCreateRequest, api.LinePlotCreateResponse](router, "/api/v1/workspace/lineplot/create"),
		LinePlotRetrieve: fhttp.UnaryServer[api.LinePlotRetrieveRequest, api.LinePlotRetrieveResponse](router, "/api/v1/workspace/lineplot/retrieve"),
		LinePlotDelete:   fhttp.UnaryServer[api.LinePlotDeleteRequest, types.Nil](router, "/api/v1/workspace/lineplot/delete"),
		LinePlotRename:   fhttp.UnaryServer[api.LinePlotRenameRequest, types.Nil](router, "/api/v1/workspace/lineplot/rename"),
		LinePlotSetData:  fhttp.UnaryServer[api.LinePlotSetDataRequest, types.Nil](router, "/api/v1/workspace/lineplot/set-data"),

		// LOG
		LogCreate:   fhttp.UnaryServer[api.LogCreateRequest, api.LogCreateResponse](router, "/api/v1/workspace/log/create"),
		LogRetrieve: fhttp.UnaryServer[api.LogRetrieveRequest, api.LogRetrieveResponse](router, "/api/v1/workspace/log/retrieve"),
		LogDelete:   fhttp.UnaryServer[api.LogDeleteRequest, types.Nil](router, "/api/v1/workspace/log/delete"),
		LogRename:   fhttp.UnaryServer[api.LogRenameRequest, types.Nil](router, "/api/v1/workspace/log/rename"),
		LogSetData:  fhttp.UnaryServer[api.LogSetDataRequest, types.Nil](router, "/api/v1/workspace/log/set-data"),

		// TABLE
		TableCreate:   fhttp.UnaryServer[api.TableCreateRequest, api.TableCreateResponse](router, "/api/v1/workspace/table/create"),
		TableRetrieve: fhttp.UnaryServer[api.TableRetrieveRequest, api.TableRetrieveResponse](router, "/api/v1/workspace/table/retrieve"),
		TableDelete:   fhttp.UnaryServer[api.TableDeleteRequest, types.Nil](router, "/api/v1/workspace/table/delete"),
		TableRename:   fhttp.UnaryServer[api.TableRenameRequest, types.Nil](router, "/api/v1/workspace/table/rename"),
		TableSetData:  fhttp.UnaryServer[api.TableSetDataRequest, types.Nil](router, "/api/v1/workspace/table/set-data"),

		// LABEL
		LabelCreate:   fhttp.UnaryServer[api.LabelCreateRequest, api.LabelCreateResponse](router, "/api/v1/label/create"),
		LabelRetrieve: fhttp.UnaryServer[api.LabelRetrieveRequest, api.LabelRetrieveResponse](router, "/api/v1/label/retrieve"),
		LabelDelete:   fhttp.UnaryServer[api.LabelDeleteRequest, types.Nil](router, "/api/v1/label/delete"),
		LabelAdd:      fhttp.UnaryServer[api.LabelAddRequest, types.Nil](router, "/api/v1/label/set"),
		LabelRemove:   fhttp.UnaryServer[api.LabelRemoveRequest, types.Nil](router, "/api/v1/label/remove"),

		// RACK
		RackCreate:   fhttp.UnaryServer[api.RackCreateRequest, api.RackCreateResponse](router, "/api/v1/rack/create"),
		RackRetrieve: fhttp.UnaryServer[api.RackRetrieveRequest, api.RackRetrieveResponse](router, "/api/v1/rack/retrieve"),
		RackDelete:   fhttp.UnaryServer[api.RackDeleteRequest, types.Nil](router, "/api/v1/rack/delete"),

		// TASK
		TaskCreate:   fhttp.UnaryServer[api.TaskCreateRequest, api.TaskCreateResponse](router, "/api/v1/task/create"),
		TaskRetrieve: fhttp.UnaryServer[api.TaskRetrieveRequest, api.TaskRetrieveResponse](router, "/api/v1/task/retrieve"),
		TaskDelete:   fhttp.UnaryServer[api.TaskDeleteRequest, types.Nil](router, "/api/v1/task/delete"),
		TaskCopy:     fhttp.UnaryServer[api.TaskCopyRequest, api.TaskCopyResponse](router, "/api/v1/task/copy"),

		// DEVICE
		DeviceCreate:   fhttp.UnaryServer[api.DeviceCreateRequest, api.DeviceCreateResponse](router, "/api/v1/device/create"),
		DeviceRetrieve: fhttp.UnaryServer[api.DeviceRetrieveRequest, api.DeviceRetrieveResponse](router, "/api/v1/device/retrieve"),
		DeviceDelete:   fhttp.UnaryServer[api.DeviceDeleteRequest, types.Nil](router, "/api/v1/device/delete"),

		// ACCESS
		AccessCreatePolicy:   fhttp.UnaryServer[api.AccessCreatePolicyRequest, api.AccessCreatePolicyResponse](router, "/api/v1/access/policy/create"),
		AccessDeletePolicy:   fhttp.UnaryServer[api.AccessDeletePolicyRequest, types.Nil](router, "/api/v1/access/policy/delete"),
		AccessRetrievePolicy: fhttp.UnaryServer[api.AccessRetrievePolicyRequest, api.AccessRetrievePolicyResponse](router, "/api/v1/access/policy/retrieve"),

		// ARC
		ArcCreate:   fhttp.UnaryServer[api.ArcCreateRequest, api.ArcCreateResponse](router, "/api/v1/arc/create"),
		ArcDelete:   fhttp.UnaryServer[api.ArcDeleteRequest, types.Nil](router, "/api/v1/arc/delete"),
		ArcRetrieve: fhttp.UnaryServer[api.ArcRetrieveRequest, api.ArcRetrieveResponse](router, "/api/v1/arc/retrieve"),
		ArcLSP:      fhttp.StreamServer[api.ArcLSPMessage, api.ArcLSPMessage](router, "/api/v1/arc/lsp"),

		// STATUS
		StatusSet:      fhttp.UnaryServer[api.StatusSetRequest, api.StatusSetResponse](router, "/api/v1/status/set"),
		StatusRetrieve: fhttp.UnaryServer[api.StatusRetrieveRequest, api.StatusRetrieveResponse](router, "/api/v1/status/retrieve"),
		StatusDelete:   fhttp.UnaryServer[api.StatusDeleteRequest, types.Nil](router, "/api/v1/status/delete"),
	}
}
