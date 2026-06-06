// Copyright 2026 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/freighter/noop"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/access"
	apiarc "github.com/synnaxlabs/synnax/pkg/api/arc"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	"github.com/synnaxlabs/synnax/pkg/api/group"
	"github.com/synnaxlabs/synnax/pkg/api/imex"
	"github.com/synnaxlabs/synnax/pkg/api/label"
	"github.com/synnaxlabs/synnax/pkg/api/lineplot"
	"github.com/synnaxlabs/synnax/pkg/api/log"
	"github.com/synnaxlabs/synnax/pkg/api/ontology"
	"github.com/synnaxlabs/synnax/pkg/api/panel"
	"github.com/synnaxlabs/synnax/pkg/api/project"
	"github.com/synnaxlabs/synnax/pkg/api/schematic"
	"github.com/synnaxlabs/synnax/pkg/api/table"
	"github.com/synnaxlabs/synnax/pkg/api/user"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/transport/grpc/arc"
	"github.com/synnaxlabs/synnax/pkg/transport/grpc/auth"
	"github.com/synnaxlabs/synnax/pkg/transport/grpc/channel"
	"github.com/synnaxlabs/synnax/pkg/transport/grpc/connectivity"
	"github.com/synnaxlabs/synnax/pkg/transport/grpc/device"
	"github.com/synnaxlabs/synnax/pkg/transport/grpc/framer"
	"github.com/synnaxlabs/synnax/pkg/transport/grpc/rack"
	"github.com/synnaxlabs/synnax/pkg/transport/grpc/ranger"
	"github.com/synnaxlabs/synnax/pkg/transport/grpc/ranger/alias"
	"github.com/synnaxlabs/synnax/pkg/transport/grpc/ranger/kv"
	"github.com/synnaxlabs/synnax/pkg/transport/grpc/status"
	"github.com/synnaxlabs/synnax/pkg/transport/grpc/task"
	"github.com/synnaxlabs/synnax/pkg/transport/grpc/view"
)

// Bind constructs the gRPC transport for every API service, binds the API
// layer's handlers and middleware to it, and returns the bindable transports
// for registration with the server's gRPC branch. channelSvc resolves channel
// keys for the frame codec.
func Bind(layer *api.Layer, channelSvc *distchannel.Service) []grpc.BindableTransport {
	var t api.Transport
	transports := grpc.CompoundBindableTransport{
		channel.New(&t),
		framer.New(&t, channelSvc),
		connectivity.New(&t),
		auth.New(&t),
		ranger.New(&t),
		kv.New(&t),
		alias.New(&t),
		rack.New(&t),
		task.New(&t),
		device.New(&t),
		status.New(&t),
		arc.New(&t),
		view.New(&t),
	}

	// AUTH
	t.AuthChangePassword = noop.UnaryServer[apiauth.ChangePasswordRequest, types.Nil]{}

	// CHANNEL
	t.ChannelRename = noop.UnaryServer[apichannel.RenameRequest, types.Nil]{}
	t.ChannelRetrieveGroup = noop.UnaryServer[apichannel.RetrieveGroupRequest, apichannel.RetrieveGroupResponse]{}

	// USER
	t.UserRename = noop.UnaryServer[user.RenameRequest, types.Nil]{}
	t.UserChangeUsername = noop.UnaryServer[user.ChangeUsernameRequest, types.Nil]{}
	t.UserCreate = noop.UnaryServer[user.CreateRequest, user.CreateResponse]{}
	t.UserDelete = noop.UnaryServer[user.DeleteRequest, types.Nil]{}
	t.UserRetrieve = noop.UnaryServer[user.RetrieveRequest, user.RetrieveResponse]{}

	// ONTOLOGY
	t.OntologyRetrieve = noop.UnaryServer[ontology.RetrieveRequest, ontology.RetrieveResponse]{}
	t.OntologyAddChildren = noop.UnaryServer[ontology.AddChildrenRequest, types.Nil]{}
	t.OntologyRemoveChildren = noop.UnaryServer[ontology.RemoveChildrenRequest, types.Nil]{}
	t.OntologyMoveChildren = noop.UnaryServer[ontology.MoveChildrenRequest, types.Nil]{}

	// GROUP
	t.GroupCreate = noop.UnaryServer[group.CreateRequest, group.CreateResponse]{}
	t.GroupDelete = noop.UnaryServer[group.DeleteRequest, types.Nil]{}
	t.GroupRename = noop.UnaryServer[group.RenameRequest, types.Nil]{}

	// PROJECT
	t.ProjectCreate = noop.UnaryServer[project.CreateRequest, project.CreateResponse]{}
	t.ProjectRetrieve = noop.UnaryServer[project.RetrieveRequest, project.RetrieveResponse]{}
	t.ProjectDelete = noop.UnaryServer[project.DeleteRequest, types.Nil]{}
	t.ProjectRename = noop.UnaryServer[project.RenameRequest, types.Nil]{}

	// PANEL
	t.PanelCreate = noop.UnaryServer[panel.CreateRequest, panel.CreateResponse]{}
	t.PanelRetrieve = noop.UnaryServer[panel.RetrieveRequest, panel.RetrieveResponse]{}
	t.PanelDelete = noop.UnaryServer[panel.DeleteRequest, types.Nil]{}
	t.PanelRename = noop.UnaryServer[panel.RenameRequest, types.Nil]{}
	t.PanelDispatch = noop.UnaryServer[panel.DispatchRequest, types.Nil]{}

	// SCHEMATIC
	t.SchematicCreate = noop.UnaryServer[schematic.CreateRequest, schematic.CreateResponse]{}
	t.SchematicDelete = noop.UnaryServer[schematic.DeleteRequest, types.Nil]{}
	t.SchematicRetrieve = noop.UnaryServer[schematic.RetrieveRequest, schematic.RetrieveResponse]{}
	t.SchematicDispatch = noop.UnaryServer[schematic.DispatchRequest, types.Nil]{}
	t.SchematicCopy = noop.UnaryServer[schematic.CopyRequest, schematic.CopyResponse]{}

	// SCHEMATIC SYMBOL
	t.SchematicCreateSymbol = noop.UnaryServer[schematic.CreateSymbolRequest, schematic.CreateSymbolResponse]{}
	t.SchematicRetrieveSymbol = noop.UnaryServer[schematic.RetrieveSymbolRequest, schematic.RetrieveSymbolResponse]{}
	t.SchematicDeleteSymbol = noop.UnaryServer[schematic.DeleteSymbolRequest, types.Nil]{}
	t.SchematicRenameSymbol = noop.UnaryServer[schematic.RenameSymbolRequest, types.Nil]{}
	t.SchematicRetrieveSymbolGroup = noop.UnaryServer[schematic.RetrieveSymbolGroupRequest, schematic.RetrieveSymbolGroupResponse]{}

	// LINE PLOT
	t.LinePlotCreate = noop.UnaryServer[lineplot.CreateRequest, lineplot.CreateResponse]{}
	t.LinePlotRetrieve = noop.UnaryServer[lineplot.RetrieveRequest, lineplot.RetrieveResponse]{}
	t.LinePlotDelete = noop.UnaryServer[lineplot.DeleteRequest, types.Nil]{}
	t.LinePlotRename = noop.UnaryServer[lineplot.RenameRequest, types.Nil]{}
	t.LinePlotSetData = noop.UnaryServer[lineplot.SetDataRequest, types.Nil]{}
	t.LinePlotDispatch = noop.UnaryServer[lineplot.DispatchRequest, types.Nil]{}

	// LOG
	t.LogCreate = noop.UnaryServer[log.CreateRequest, log.CreateResponse]{}
	t.LogRetrieve = noop.UnaryServer[log.RetrieveRequest, log.RetrieveResponse]{}
	t.LogDelete = noop.UnaryServer[log.DeleteRequest, types.Nil]{}
	t.LogRename = noop.UnaryServer[log.RenameRequest, types.Nil]{}
	t.LogSetData = noop.UnaryServer[log.SetDataRequest, types.Nil]{}

	// TABLE
	t.TableCreate = noop.UnaryServer[table.CreateRequest, table.CreateResponse]{}
	t.TableRetrieve = noop.UnaryServer[table.RetrieveRequest, table.RetrieveResponse]{}
	t.TableDelete = noop.UnaryServer[table.DeleteRequest, types.Nil]{}
	t.TableDispatch = noop.UnaryServer[table.DispatchRequest, types.Nil]{}

	// LABEL
	t.LabelCreate = noop.UnaryServer[label.CreateRequest, label.CreateResponse]{}
	t.LabelRetrieve = noop.UnaryServer[label.RetrieveRequest, label.RetrieveResponse]{}
	t.LabelDelete = noop.UnaryServer[label.DeleteRequest, types.Nil]{}
	t.LabelAdd = noop.UnaryServer[label.AddRequest, types.Nil]{}
	t.LabelRemove = noop.UnaryServer[label.RemoveRequest, types.Nil]{}

	// ACCESS
	t.AccessCreatePolicy = noop.UnaryServer[access.CreatePolicyRequest, access.CreatePolicyResponse]{}
	t.AccessDeletePolicy = noop.UnaryServer[access.DeletePolicyRequest, types.Nil]{}
	t.AccessRetrievePolicy = noop.UnaryServer[access.RetrievePolicyRequest, access.RetrievePolicyResponse]{}
	t.AccessCreateRole = noop.UnaryServer[access.CreateRoleRequest, access.CreateRoleResponse]{}
	t.AccessDeleteRole = noop.UnaryServer[access.DeleteRoleRequest, types.Nil]{}
	t.AccessRetrieveRole = noop.UnaryServer[access.RetrieveRoleRequest, access.RetrieveRoleResponse]{}
	t.AccessAssignRole = noop.UnaryServer[access.AssignRoleRequest, types.Nil]{}
	t.AccessUnassignRole = noop.UnaryServer[access.UnassignRoleRequest, types.Nil]{}

	// IMPORT/EXPORT
	t.ImExImport = noop.UnaryServer[imex.ImportRequest, imex.ImportResponse]{}
	t.ImExExport = noop.UnaryServer[imex.ExportRequest, imex.ExportResponse]{}

	// ARC LSP
	t.ArcLSP = noop.StreamServer[apiarc.LSPMessage, apiarc.LSPMessage]{}

	layer.BindTo(t)
	return transports
}
