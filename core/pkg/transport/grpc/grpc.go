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
	"github.com/synnaxlabs/synnax/pkg/api/schematic/symbol"
	"github.com/synnaxlabs/synnax/pkg/api/table"
	"github.com/synnaxlabs/synnax/pkg/api/user"
	"github.com/synnaxlabs/synnax/pkg/transport/grpc/arc"
	"github.com/synnaxlabs/synnax/pkg/transport/grpc/auth"
	"github.com/synnaxlabs/synnax/pkg/transport/grpc/channel"
	"github.com/synnaxlabs/synnax/pkg/transport/grpc/connectivity"
	"github.com/synnaxlabs/synnax/pkg/transport/grpc/control"
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

// Bind constructs the gRPC transport for every API service, binds the API layer's
// handlers and middleware to it, and returns the bindable transports for registration
// with the server's gRPC branch. The framer codec resolves channel data types through
// the API layer's channel service.
func Bind(layer *api.Layer) []grpc.BindableTransport {
	var t api.Transport
	transports := grpc.CompoundBindableTransport{
		channel.New(&t),
		framer.New(&t, layer.Channel),
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
		control.New(&t),
	}

	// AUTH
	t.AuthChangePassword = noop.UnaryServer[apiauth.ChangePasswordRequest, struct{}]{}

	// CHANNEL
	t.ChannelRename = noop.UnaryServer[apichannel.RenameRequest, struct{}]{}
	t.ChannelRetrieveGroup = noop.UnaryServer[apichannel.RetrieveGroupRequest, apichannel.RetrieveGroupResponse]{}

	// USER
	t.UserRename = noop.UnaryServer[user.RenameRequest, struct{}]{}
	t.UserChangeUsername = noop.UnaryServer[user.ChangeUsernameRequest, struct{}]{}
	t.UserCreate = noop.UnaryServer[user.CreateRequest, user.CreateResponse]{}
	t.UserDelete = noop.UnaryServer[user.DeleteRequest, struct{}]{}
	t.UserRetrieve = noop.UnaryServer[user.RetrieveRequest, user.RetrieveResponse]{}

	// ONTOLOGY
	t.OntologyRetrieve = noop.UnaryServer[ontology.RetrieveRequest, ontology.RetrieveResponse]{}
	t.OntologyAddChildren = noop.UnaryServer[ontology.AddChildrenRequest, struct{}]{}
	t.OntologyRemoveChildren = noop.UnaryServer[ontology.RemoveChildrenRequest, struct{}]{}
	t.OntologyMoveChildren = noop.UnaryServer[ontology.MoveChildrenRequest, struct{}]{}

	// GROUP
	t.GroupCreate = noop.UnaryServer[group.CreateRequest, group.CreateResponse]{}
	t.GroupDelete = noop.UnaryServer[group.DeleteRequest, struct{}]{}
	t.GroupRename = noop.UnaryServer[group.RenameRequest, struct{}]{}
	t.GroupRetrieve = noop.UnaryServer[group.RetrieveRequest, group.RetrieveResponse]{}

	// PROJECT
	t.ProjectCreate = noop.UnaryServer[project.CreateRequest, project.CreateResponse]{}
	t.ProjectRetrieve = noop.UnaryServer[project.RetrieveRequest, project.RetrieveResponse]{}
	t.ProjectDelete = noop.UnaryServer[project.DeleteRequest, struct{}]{}
	t.ProjectRename = noop.UnaryServer[project.RenameRequest, struct{}]{}
	t.ProjectSetLayout = noop.UnaryServer[project.SetLayoutRequest, struct{}]{}
	t.ProjectExport = noop.UnaryServer[project.ExportRequest, project.ExportResponse]{}
	t.ProjectImport = noop.UnaryServer[project.ImportRequest, project.ImportResponse]{}

	// SCHEMATIC
	t.SchematicCreate = noop.UnaryServer[schematic.CreateRequest, schematic.CreateResponse]{}
	t.SchematicDelete = noop.UnaryServer[schematic.DeleteRequest, struct{}]{}
	t.SchematicRetrieve = noop.UnaryServer[schematic.RetrieveRequest, schematic.RetrieveResponse]{}
	t.SchematicDispatch = noop.UnaryServer[schematic.DispatchRequest, struct{}]{}
	t.SchematicCopy = noop.UnaryServer[schematic.CopyRequest, schematic.CopyResponse]{}

	// SCHEMATIC SYMBOL
	t.SchematicSymbolCreate = noop.UnaryServer[symbol.CreateRequest, symbol.CreateResponse]{}
	t.SchematicSymbolRetrieve = noop.UnaryServer[symbol.RetrieveRequest, symbol.RetrieveResponse]{}
	t.SchematicSymbolDelete = noop.UnaryServer[symbol.DeleteRequest, struct{}]{}
	t.SchematicSymbolRename = noop.UnaryServer[symbol.RenameRequest, struct{}]{}
	t.SchematicSymbolRetrieveGroup = noop.UnaryServer[symbol.RetrieveGroupRequest, symbol.RetrieveGroupResponse]{}
	t.SchematicSymbolExportGroup = noop.UnaryServer[symbol.ExportGroupRequest, symbol.ExportGroupResponse]{}
	t.SchematicSymbolImportGroup = noop.UnaryServer[symbol.ImportGroupRequest, symbol.ImportGroupResponse]{}
	t.SchematicSymbolDeleteGroup = noop.UnaryServer[symbol.DeleteGroupRequest, struct{}]{}

	// LINE PLOT
	t.LinePlotCreate = noop.UnaryServer[lineplot.CreateRequest, lineplot.CreateResponse]{}
	t.LinePlotRetrieve = noop.UnaryServer[lineplot.RetrieveRequest, lineplot.RetrieveResponse]{}
	t.LinePlotDelete = noop.UnaryServer[lineplot.DeleteRequest, struct{}]{}
	t.LinePlotDispatch = noop.UnaryServer[lineplot.DispatchRequest, struct{}]{}

	// PANEL
	t.PanelCreate = noop.UnaryServer[panel.CreateRequest, panel.CreateResponse]{}
	t.PanelRetrieve = noop.UnaryServer[panel.RetrieveRequest, panel.RetrieveResponse]{}
	t.PanelDelete = noop.UnaryServer[panel.DeleteRequest, struct{}]{}
	t.PanelDispatch = noop.UnaryServer[panel.DispatchRequest, struct{}]{}

	// LOG
	t.LogCreate = noop.UnaryServer[log.CreateRequest, log.CreateResponse]{}
	t.LogRetrieve = noop.UnaryServer[log.RetrieveRequest, log.RetrieveResponse]{}
	t.LogDelete = noop.UnaryServer[log.DeleteRequest, struct{}]{}
	t.LogDispatch = noop.UnaryServer[log.DispatchRequest, struct{}]{}

	// TABLE
	t.TableCreate = noop.UnaryServer[table.CreateRequest, table.CreateResponse]{}
	t.TableRetrieve = noop.UnaryServer[table.RetrieveRequest, table.RetrieveResponse]{}
	t.TableDelete = noop.UnaryServer[table.DeleteRequest, struct{}]{}
	t.TableDispatch = noop.UnaryServer[table.DispatchRequest, struct{}]{}

	// LABEL
	t.LabelCreate = noop.UnaryServer[label.CreateRequest, label.CreateResponse]{}
	t.LabelRetrieve = noop.UnaryServer[label.RetrieveRequest, label.RetrieveResponse]{}
	t.LabelDelete = noop.UnaryServer[label.DeleteRequest, struct{}]{}
	t.LabelAdd = noop.UnaryServer[label.AddRequest, struct{}]{}
	t.LabelRemove = noop.UnaryServer[label.RemoveRequest, struct{}]{}

	// ACCESS
	t.AccessCreatePolicy = noop.UnaryServer[access.CreatePolicyRequest, access.CreatePolicyResponse]{}
	t.AccessDeletePolicy = noop.UnaryServer[access.DeletePolicyRequest, struct{}]{}
	t.AccessRetrievePolicy = noop.UnaryServer[access.RetrievePolicyRequest, access.RetrievePolicyResponse]{}
	t.AccessCreateRole = noop.UnaryServer[access.CreateRoleRequest, access.CreateRoleResponse]{}
	t.AccessDeleteRole = noop.UnaryServer[access.DeleteRoleRequest, struct{}]{}
	t.AccessRetrieveRole = noop.UnaryServer[access.RetrieveRoleRequest, access.RetrieveRoleResponse]{}
	t.AccessAssignRole = noop.UnaryServer[access.AssignRoleRequest, struct{}]{}
	t.AccessUnassignRole = noop.UnaryServer[access.UnassignRoleRequest, struct{}]{}

	// IMPORT/EXPORT
	t.ImExImport = noop.UnaryServer[imex.ImportRequest, imex.ImportResponse]{}
	t.ImExExport = noop.UnaryServer[imex.ExportRequest, imex.ExportResponse]{}

	// ARC
	t.ArcDispatch = noop.UnaryServer[apiarc.DispatchRequest, struct{}]{}
	t.ArcSetRack = noop.UnaryServer[apiarc.SetRackRequest, apiarc.SetRackResponse]{}
	t.ArcLSP = noop.StreamServer[apiarc.LSPMessage, apiarc.LSPMessage]{}

	layer.BindTo(t)
	return transports
}
