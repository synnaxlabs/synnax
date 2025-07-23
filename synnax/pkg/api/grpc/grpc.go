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

func New(channels channel.Readable) (api.Transport, []fgrpc.BindableTransport) {
	a := api.Transport{
		AuthChangePassword:     fnoop.UnaryServer[api.AuthChangePasswordRequest, types.Nil]{},
		HardwareCopyTask:       fnoop.UnaryServer[api.HardwareCopyTaskRequest, api.HardwareCopyTaskResponse]{},
		ChannelRename:          fnoop.UnaryServer[api.ChannelRenameRequest, types.Nil]{},
		ChannelRetrieveGroup:   fnoop.UnaryServer[types.Nil, api.ChannelRetrieveGroupResponse]{},
		ExportCSV:              fnoop.UnaryServer[api.ExportCSVRequest, api.ExportCSVResponse]{},
		FrameRead:              fnoop.UnaryServer[api.FrameReadRequest, api.FrameReadResponse]{},
		UserRename:             fnoop.UnaryServer[api.UserRenameRequest, types.Nil]{},
		UserChangeUsername:     fnoop.UnaryServer[api.UserChangeUsernameRequest, types.Nil]{},
		UserCreate:             fnoop.UnaryServer[api.UserCreateRequest, api.UserCreateResponse]{},
		UserDelete:             fnoop.UnaryServer[api.UserDeleteRequest, types.Nil]{},
		UserRetrieve:           fnoop.UnaryServer[api.UserRetrieveRequest, api.UserRetrieveResponse]{},
		RangeRename:            fnoop.UnaryServer[api.RangeRenameRequest, types.Nil]{},
		OntologyRetrieve:       fnoop.UnaryServer[api.OntologyRetrieveRequest, api.OntologyRetrieveResponse]{},
		OntologyAddChildren:    fnoop.UnaryServer[api.OntologyAddChildrenRequest, types.Nil]{},
		OntologyRemoveChildren: fnoop.UnaryServer[api.OntologyRemoveChildrenRequest, types.Nil]{},
		OntologyMoveChildren:   fnoop.UnaryServer[api.OntologyMoveChildrenRequest, types.Nil]{},
		OntologyGroupCreate:    fnoop.UnaryServer[api.OntologyCreateGroupRequest, api.OntologyCreateGroupResponse]{},
		OntologyGroupDelete:    fnoop.UnaryServer[api.OntologyDeleteGroupRequest, types.Nil]{},
		OntologyGroupRename:    fnoop.UnaryServer[api.OntologyRenameGroupRequest, types.Nil]{},
		WorkspaceCreate:        fnoop.UnaryServer[api.WorkspaceCreateRequest, api.WorkspaceCreateResponse]{},
		WorkspaceRetrieve:      fnoop.UnaryServer[api.WorkspaceRetrieveRequest, api.WorkspaceRetrieveResponse]{},
		WorkspaceDelete:        fnoop.UnaryServer[api.WorkspaceDeleteRequest, types.Nil]{},
		WorkspaceRename:        fnoop.UnaryServer[api.WorkspaceRenameRequest, types.Nil]{},
		WorkspaceSetLayout:     fnoop.UnaryServer[api.WorkspaceSetLayoutRequest, types.Nil]{},
		SchematicCreate:        fnoop.UnaryServer[api.SchematicCreateRequest, api.SchematicCreateResponse]{},
		SchematicDelete:        fnoop.UnaryServer[api.SchematicDeleteRequest, types.Nil]{},
		SchematicRetrieve:      fnoop.UnaryServer[api.SchematicRetrieveRequest, api.SchematicRetrieveResponse]{},
		SchematicRename:        fnoop.UnaryServer[api.SchematicRenameRequest, types.Nil]{},
		SchematicSetData:       fnoop.UnaryServer[api.SchematicSetDataRequest, types.Nil]{},
		SchematicCopy:          fnoop.UnaryServer[api.SchematicCopyRequest, api.SchematicCopyResponse]{},
		LinePlotCreate:         fnoop.UnaryServer[api.LinePlotCreateRequest, api.LinePlotCreateResponse]{},
		LinePlotRetrieve:       fnoop.UnaryServer[api.LinePlotRetrieveRequest, api.LinePlotRetrieveResponse]{},
		LinePlotDelete:         fnoop.UnaryServer[api.LinePlotDeleteRequest, types.Nil]{},
		LinePlotRename:         fnoop.UnaryServer[api.LinePlotRenameRequest, types.Nil]{},
		LinePlotSetData:        fnoop.UnaryServer[api.LinePlotSetDataRequest, types.Nil]{},
		LogCreate:              fnoop.UnaryServer[api.LogCreateRequest, api.LogCreateResponse]{},
		LogRetrieve:            fnoop.UnaryServer[api.LogRetrieveRequest, api.LogRetrieveResponse]{},
		LogDelete:              fnoop.UnaryServer[api.LogDeleteRequest, types.Nil]{},
		LogRename:              fnoop.UnaryServer[api.LogRenameRequest, types.Nil]{},
		LogSetData:             fnoop.UnaryServer[api.LogSetDataRequest, types.Nil]{},
		TableCreate:            fnoop.UnaryServer[api.TableCreateRequest, api.TableCreateResponse]{},
		TableRetrieve:          fnoop.UnaryServer[api.TableRetrieveRequest, api.TableRetrieveResponse]{},
		TableDelete:            fnoop.UnaryServer[api.TableDeleteRequest, types.Nil]{},
		TableRename:            fnoop.UnaryServer[api.TableRenameRequest, types.Nil]{},
		TableSetData:           fnoop.UnaryServer[api.TableSetDataRequest, types.Nil]{},
		LabelCreate:            fnoop.UnaryServer[api.LabelCreateRequest, api.LabelCreateResponse]{},
		LabelRetrieve:          fnoop.UnaryServer[api.LabelRetrieveRequest, api.LabelRetrieveResponse]{},
		LabelDelete:            fnoop.UnaryServer[api.LabelDeleteRequest, types.Nil]{},
		LabelAdd:               fnoop.UnaryServer[api.LabelAddRequest, types.Nil]{},
		LabelRemove:            fnoop.UnaryServer[api.LabelRemoveRequest, types.Nil]{},
		AccessCreatePolicy:     fnoop.UnaryServer[api.AccessCreatePolicyRequest, api.AccessCreatePolicyResponse]{},
		AccessDeletePolicy:     fnoop.UnaryServer[api.AccessDeletePolicyRequest, types.Nil]{},
		AccessRetrievePolicy:   fnoop.UnaryServer[api.AccessRetrievePolicyRequest, api.AccessRetrievePolicyResponse]{},
	}
	transports := fgrpc.CompoundBindableTransport{
		newChannel(&a),
		newFramer(&a, channels),
		newConnectivity(&a),
		newAuth(&a),
		newRanger(&a),
		newHardware(&a),
	}
	return a, transports
}
