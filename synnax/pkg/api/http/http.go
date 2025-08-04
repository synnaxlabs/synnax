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
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/binary"
)

func New(router *fhttp.Router, channels channel.Readable) api.Transport {
	streamCodecMapping := map[string]func() binary.Codec{
		"application/sy-framer": func() binary.Codec {
			return api.NewWSFramerCodec(channels)
		},
	}
	so := fhttp.WithAdditionalCodecs(streamCodecMapping)
	return api.Transport{
		AuthLogin:              fhttp.NewUnaryServer[api.AuthLoginRequest, api.AuthLoginResponse](router, "/api/v1/auth/login"),
		AuthChangePassword:     fhttp.NewUnaryServer[api.AuthChangePasswordRequest, types.Nil](router, "/api/v1/auth/change-password"),
		UserRename:             fhttp.NewUnaryServer[api.UserRenameRequest, types.Nil](router, "/api/v1/user/rename"),
		UserChangeUsername:     fhttp.NewUnaryServer[api.UserChangeUsernameRequest, types.Nil](router, "/api/v1/user/change-username"),
		UserCreate:             fhttp.NewUnaryServer[api.UserCreateRequest, api.UserCreateResponse](router, "/api/v1/user/create"),
		UserDelete:             fhttp.NewUnaryServer[api.UserDeleteRequest, types.Nil](router, "/api/v1/user/delete"),
		UserRetrieve:           fhttp.NewUnaryServer[api.UserRetrieveRequest, api.UserRetrieveResponse](router, "/api/v1/user/retrieve"),
		ChannelCreate:          fhttp.NewUnaryServer[api.ChannelCreateRequest, api.ChannelCreateResponse](router, "/api/v1/channel/create"),
		ChannelRetrieve:        fhttp.NewUnaryServer[api.ChannelRetrieveRequest, api.ChannelRetrieveResponse](router, "/api/v1/channel/retrieve"),
		ChannelDelete:          fhttp.NewUnaryServer[api.ChannelDeleteRequest, types.Nil](router, "/api/v1/channel/delete"),
		ChannelRename:          fhttp.NewUnaryServer[api.ChannelRenameRequest, types.Nil](router, "/api/v1/channel/rename"),
		ChannelRetrieveGroup:   fhttp.NewUnaryServer[types.Nil, api.ChannelRetrieveGroupResponse](router, "/api/v1/channel/retrieve-group"),
		ConnectivityCheck:      fhttp.NewUnaryServer[types.Nil, api.ConnectivityCheckResponse](router, "/api/v1/connectivity/check"),
		ExportCSV:              fhttp.NewUnaryServer[api.ExportCSVRequest, api.ExportCSVResponse](router, "/api/v1/export/csv"),
		FrameRead:              fhttp.NewUnaryServer[api.FrameReadRequest, api.FrameReadResponse](router, "/api/v1/frame/read", so),
		FrameWriter:            fhttp.NewStreamServer[api.FrameWriterRequest, api.FrameWriterResponse](router, "/api/v1/frame/write", so),
		FrameIterator:          fhttp.NewStreamServer[api.FrameIteratorRequest, api.FrameIteratorResponse](router, "/api/v1/frame/iterate"),
		FrameStreamer:          fhttp.NewStreamServer[api.FrameStreamerRequest, api.FrameStreamerResponse](router, "/api/v1/frame/stream", so),
		FrameDelete:            fhttp.NewUnaryServer[api.FrameDeleteRequest, types.Nil](router, "/api/v1/frame/delete"),
		RangeCreate:            fhttp.NewUnaryServer[api.RangeCreateRequest, api.RangeCreateResponse](router, "/api/v1/range/create"),
		RangeRetrieve:          fhttp.NewUnaryServer[api.RangeRetrieveRequest, api.RangeRetrieveResponse](router, "/api/v1/range/retrieve"),
		RangeDelete:            fhttp.NewUnaryServer[api.RangeDeleteRequest, types.Nil](router, "/api/v1/range/delete"),
		RangeKVGet:             fhttp.NewUnaryServer[api.RangeKVGetRequest, api.RangeKVGetResponse](router, "/api/v1/range/kv/get"),
		RangeKVSet:             fhttp.NewUnaryServer[api.RangeKVSetRequest, types.Nil](router, "/api/v1/range/kv/set"),
		RangeKVDelete:          fhttp.NewUnaryServer[api.RangeKVDeleteRequest, types.Nil](router, "/api/v1/range/kv/delete"),
		RangeAliasSet:          fhttp.NewUnaryServer[api.RangeAliasSetRequest, types.Nil](router, "/api/v1/range/alias/set"),
		RangeAliasResolve:      fhttp.NewUnaryServer[api.RangeAliasResolveRequest, api.RangeAliasResolveResponse](router, "/api/v1/range/alias/resolve"),
		RangeAliasList:         fhttp.NewUnaryServer[api.RangeAliasListRequest, api.RangeAliasListResponse](router, "/api/v1/range/alias/list"),
		RangeRename:            fhttp.NewUnaryServer[api.RangeRenameRequest, types.Nil](router, "/api/v1/range/rename"),
		RangeAliasDelete:       fhttp.NewUnaryServer[api.RangeAliasDeleteRequest, types.Nil](router, "/api/v1/range/alias/delete"),
		OntologyRetrieve:       fhttp.NewUnaryServer[api.OntologyRetrieveRequest, api.OntologyRetrieveResponse](router, "/api/v1/ontology/retrieve"),
		OntologyAddChildren:    fhttp.NewUnaryServer[api.OntologyAddChildrenRequest, types.Nil](router, "/api/v1/ontology/add-children"),
		OntologyRemoveChildren: fhttp.NewUnaryServer[api.OntologyRemoveChildrenRequest, types.Nil](router, "/api/v1/ontology/remove-children"),
		OntologyMoveChildren:   fhttp.NewUnaryServer[api.OntologyMoveChildrenRequest, types.Nil](router, "/api/v1/ontology/move-children"),
		OntologyGroupCreate:    fhttp.NewUnaryServer[api.OntologyCreateGroupRequest, api.OntologyCreateGroupResponse](router, "/api/v1/ontology/create-group"),
		OntologyGroupDelete:    fhttp.NewUnaryServer[api.OntologyDeleteGroupRequest, types.Nil](router, "/api/v1/ontology/delete-group"),
		OntologyGroupRename:    fhttp.NewUnaryServer[api.OntologyRenameGroupRequest, types.Nil](router, "/api/v1/ontology/rename-group"),
		WorkspaceCreate:        fhttp.NewUnaryServer[api.WorkspaceCreateRequest, api.WorkspaceCreateResponse](router, "/api/v1/workspace/create"),
		WorkspaceRetrieve:      fhttp.NewUnaryServer[api.WorkspaceRetrieveRequest, api.WorkspaceRetrieveResponse](router, "/api/v1/workspace/retrieve"),
		WorkspaceDelete:        fhttp.NewUnaryServer[api.WorkspaceDeleteRequest, types.Nil](router, "/api/v1/workspace/delete"),
		WorkspaceRename:        fhttp.NewUnaryServer[api.WorkspaceRenameRequest, types.Nil](router, "/api/v1/workspace/rename"),
		WorkspaceSetLayout:     fhttp.NewUnaryServer[api.WorkspaceSetLayoutRequest, types.Nil](router, "/api/v1/workspace/set-layout"),
		SchematicCreate:        fhttp.NewUnaryServer[api.SchematicCreateRequest, api.SchematicCreateResponse](router, "/api/v1/workspace/schematic/create"),
		SchematicRetrieve:      fhttp.NewUnaryServer[api.SchematicRetrieveRequest, api.SchematicRetrieveResponse](router, "/api/v1/workspace/schematic/retrieve"),
		SchematicDelete:        fhttp.NewUnaryServer[api.SchematicDeleteRequest, types.Nil](router, "/api/v1/workspace/schematic/delete"),
		SchematicRename:        fhttp.NewUnaryServer[api.SchematicRenameRequest, types.Nil](router, "/api/v1/workspace/schematic/rename"),
		SchematicSetData:       fhttp.NewUnaryServer[api.SchematicSetDataRequest, types.Nil](router, "/api/v1/workspace/schematic/set-data"),
		SchematicCopy:          fhttp.NewUnaryServer[api.SchematicCopyRequest, api.SchematicCopyResponse](router, "/api/v1/workspace/schematic/copy"),
		LogCreate:              fhttp.NewUnaryServer[api.LogCreateRequest, api.LogCreateResponse](router, "/api/v1/workspace/log/create"),
		LogRetrieve:            fhttp.NewUnaryServer[api.LogRetrieveRequest, api.LogRetrieveResponse](router, "/api/v1/workspace/log/retrieve"),
		LogDelete:              fhttp.NewUnaryServer[api.LogDeleteRequest, types.Nil](router, "/api/v1/workspace/log/delete"),
		LogRename:              fhttp.NewUnaryServer[api.LogRenameRequest, types.Nil](router, "/api/v1/workspace/log/rename"),
		LogSetData:             fhttp.NewUnaryServer[api.LogSetDataRequest, types.Nil](router, "/api/v1/workspace/log/set-data"),
		TableCreate:            fhttp.NewUnaryServer[api.TableCreateRequest, api.TableCreateResponse](router, "/api/v1/workspace/table/create"),
		TableRetrieve:          fhttp.NewUnaryServer[api.TableRetrieveRequest, api.TableRetrieveResponse](router, "/api/v1/workspace/table/retrieve"),
		TableDelete:            fhttp.NewUnaryServer[api.TableDeleteRequest, types.Nil](router, "/api/v1/workspace/table/delete"),
		TableRename:            fhttp.NewUnaryServer[api.TableRenameRequest, types.Nil](router, "/api/v1/workspace/table/rename"),
		TableSetData:           fhttp.NewUnaryServer[api.TableSetDataRequest, types.Nil](router, "/api/v1/workspace/table/set-data"),
		LinePlotCreate:         fhttp.NewUnaryServer[api.LinePlotCreateRequest, api.LinePlotCreateResponse](router, "/api/v1/workspace/lineplot/create"),
		LinePlotRetrieve:       fhttp.NewUnaryServer[api.LinePlotRetrieveRequest, api.LinePlotRetrieveResponse](router, "/api/v1/workspace/lineplot/retrieve"),
		LinePlotDelete:         fhttp.NewUnaryServer[api.LinePlotDeleteRequest, types.Nil](router, "/api/v1/workspace/lineplot/delete"),
		LinePlotRename:         fhttp.NewUnaryServer[api.LinePlotRenameRequest, types.Nil](router, "/api/v1/workspace/lineplot/rename"),
		LinePlotSetData:        fhttp.NewUnaryServer[api.LinePlotSetDataRequest, types.Nil](router, "/api/v1/workspace/lineplot/set-data"),
		LabelCreate:            fhttp.NewUnaryServer[api.LabelCreateRequest, api.LabelCreateResponse](router, "/api/v1/label/create"),
		LabelRetrieve:          fhttp.NewUnaryServer[api.LabelRetrieveRequest, api.LabelRetrieveResponse](router, "/api/v1/label/retrieve"),
		LabelDelete:            fhttp.NewUnaryServer[api.LabelDeleteRequest, types.Nil](router, "/api/v1/label/delete"),
		LabelAdd:               fhttp.NewUnaryServer[api.LabelAddRequest, types.Nil](router, "/api/v1/label/set"),
		LabelRemove:            fhttp.NewUnaryServer[api.LabelRemoveRequest, types.Nil](router, "/api/v1/label/remove"),
		HardwareCreateRack:     fhttp.NewUnaryServer[api.HardwareCreateRackRequest, api.HardwareCreateRackResponse](router, "/api/v1/hardware/rack/create"),
		HardwareRetrieveRack:   fhttp.NewUnaryServer[api.HardwareRetrieveRackRequest, api.HardwareRetrieveRackResponse](router, "/api/v1/hardware/rack/retrieve"),
		HardwareDeleteRack:     fhttp.NewUnaryServer[api.HardwareDeleteRackRequest, types.Nil](router, "/api/v1/hardware/rack/delete"),
		HardwareCreateTask:     fhttp.NewUnaryServer[api.HardwareCreateTaskRequest, api.HardwareCreateTaskResponse](router, "/api/v1/hardware/task/create"),
		HardwareRetrieveTask:   fhttp.NewUnaryServer[api.HardwareRetrieveTaskRequest, api.HardwareRetrieveTaskResponse](router, "/api/v1/hardware/task/retrieve"),
		HardwareCopyTask:       fhttp.NewUnaryServer[api.HardwareCopyTaskRequest, api.HardwareCopyTaskResponse](router, "/api/v1/hardware/task/copy"),
		HardwareDeleteTask:     fhttp.NewUnaryServer[api.HardwareDeleteTaskRequest, types.Nil](router, "/api/v1/hardware/task/delete"),
		HardwareCreateDevice:   fhttp.NewUnaryServer[api.HardwareCreateDeviceRequest, api.HardwareCreateDeviceResponse](router, "/api/v1/hardware/device/create"),
		HardwareRetrieveDevice: fhttp.NewUnaryServer[api.HardwareRetrieveDeviceRequest, api.HardwareRetrieveDeviceResponse](router, "/api/v1/hardware/device/retrieve"),
		HardwareDeleteDevice:   fhttp.NewUnaryServer[api.HardwareDeleteDeviceRequest, types.Nil](router, "/api/v1/hardware/device/delete"),
		AccessCreatePolicy:     fhttp.NewUnaryServer[api.AccessCreatePolicyRequest, api.AccessCreatePolicyResponse](router, "/api/v1/access/policy/create"),
		AccessDeletePolicy:     fhttp.NewUnaryServer[api.AccessDeletePolicyRequest, types.Nil](router, "/api/v1/access/policy/delete"),
		AccessRetrievePolicy:   fhttp.NewUnaryServer[api.AccessRetrievePolicyRequest, api.AccessRetrievePolicyResponse](router, "/api/v1/access/policy/retrieve"),
	}
}
