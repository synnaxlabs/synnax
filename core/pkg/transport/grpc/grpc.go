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
	"github.com/synnaxlabs/synnax/pkg/api/access/policy"
	"github.com/synnaxlabs/synnax/pkg/api/access/role"
	apiarc "github.com/synnaxlabs/synnax/pkg/api/arc"
	apiarcdirectory "github.com/synnaxlabs/synnax/pkg/api/arc/directory"
	apiauth "github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/batch"
	apicertificate "github.com/synnaxlabs/synnax/pkg/api/certificate"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	apiframer "github.com/synnaxlabs/synnax/pkg/api/framer"
	"github.com/synnaxlabs/synnax/pkg/api/group"
	"github.com/synnaxlabs/synnax/pkg/api/imex"
	"github.com/synnaxlabs/synnax/pkg/api/label"
	"github.com/synnaxlabs/synnax/pkg/api/license"
	"github.com/synnaxlabs/synnax/pkg/api/lineplot"
	"github.com/synnaxlabs/synnax/pkg/api/log"
	apinidevice "github.com/synnaxlabs/synnax/pkg/api/ni/device"
	"github.com/synnaxlabs/synnax/pkg/api/panel"
	"github.com/synnaxlabs/synnax/pkg/api/project"
	apialias "github.com/synnaxlabs/synnax/pkg/api/ranger/alias"
	apikv "github.com/synnaxlabs/synnax/pkg/api/ranger/kv"
	apiresource "github.com/synnaxlabs/synnax/pkg/api/resource"
	"github.com/synnaxlabs/synnax/pkg/api/schematic"
	"github.com/synnaxlabs/synnax/pkg/api/schematic/symbol"
	apisecret "github.com/synnaxlabs/synnax/pkg/api/secret"
	apistatus "github.com/synnaxlabs/synnax/pkg/api/status"
	"github.com/synnaxlabs/synnax/pkg/api/table"
	"github.com/synnaxlabs/synnax/pkg/api/user"
	"github.com/synnaxlabs/synnax/pkg/api/view"
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
		control.New(&t),
	}

	// AUTH
	t.AuthChangePassword = noop.UnaryServer[apiauth.ChangePasswordRequest, struct{}]{}
	t.AuthPasswordSet = noop.UnaryServer[
		apiauth.PasswordSetRequest,
		apiauth.PasswordSetResponse,
	]{}

	// LICENSE
	t.LicenseRetrieve = noop.UnaryServer[license.RetrieveRequest, license.RetrieveResponse]{}
	t.LicenseActivate = noop.UnaryServer[license.ActivateRequest, license.ActivateResponse]{}
	t.LicenseDeactivate = noop.UnaryServer[license.DeactivateRequest, license.DeactivateResponse]{}

	// CHANNEL
	t.ChannelDispatch = noop.UnaryServer[
		apichannel.DispatchRequest,
		apichannel.DispatchResponse,
	]{}

	// FRAME
	t.FrameRead = noop.UnaryServer[apiframer.ReadRequest, apiframer.ReadResponse]{}

	// USER
	t.UserCreate = noop.UnaryServer[user.CreateRequest, user.CreateResponse]{}
	t.UserDelete = noop.UnaryServer[user.DeleteRequest, struct{}]{}
	t.UserRetrieve = noop.UnaryServer[user.RetrieveRequest, user.RetrieveResponse]{}
	t.UserDispatch = noop.UnaryServer[user.DispatchRequest, struct{}]{}

	// RESOURCE
	t.ResourceSearch = noop.UnaryServer[apiresource.SearchRequest, apiresource.SearchResponse]{}
	t.ResourceRetrieve = noop.UnaryServer[apiresource.RetrieveRequest, apiresource.RetrieveResponse]{}

	// GROUP
	t.GroupCreate = noop.UnaryServer[group.CreateRequest, group.CreateResponse]{}
	t.GroupDelete = noop.UnaryServer[group.DeleteRequest, struct{}]{}
	t.GroupDispatch = noop.UnaryServer[group.DispatchRequest, struct{}]{}
	t.GroupRetrieve = noop.UnaryServer[group.RetrieveRequest, group.RetrieveResponse]{}
	t.GroupRetrieveAncestors = noop.UnaryServer[
		group.RetrieveAncestorsRequest,
		group.RetrieveAncestorsResponse,
	]{}

	// BATCH
	t.Batch = noop.UnaryServer[batch.ExecRequest, batch.ExecResponse]{}

	// PROJECT
	t.ProjectCreate = noop.UnaryServer[project.CreateRequest, project.CreateResponse]{}
	t.ProjectRetrieve = noop.UnaryServer[project.RetrieveRequest, project.RetrieveResponse]{}
	t.ProjectDelete = noop.UnaryServer[project.DeleteRequest, struct{}]{}
	t.ProjectDispatch = noop.UnaryServer[project.DispatchRequest, struct{}]{}
	t.ProjectExport = noop.UnaryServer[project.ExportRequest, project.ExportResponse]{}
	t.ProjectImport = noop.UnaryServer[project.ImportRequest, project.ImportResponse]{}

	// NI DEVICE
	t.NIDeviceCreate = noop.UnaryServer[apinidevice.CreateRequest, apinidevice.CreateResponse]{}
	t.NIDeviceRetrieve = noop.UnaryServer[apinidevice.RetrieveRequest, apinidevice.RetrieveResponse]{}
	t.NIDeviceUpdate = noop.UnaryServer[apinidevice.UpdateRequest, struct{}]{}
	t.NIDeviceDelete = noop.UnaryServer[apinidevice.DeleteRequest, struct{}]{}

	// STATUS
	t.StatusRename = noop.UnaryServer[apistatus.RenameRequest, struct{}]{}

	// RANGE
	t.AliasDispatch = noop.UnaryServer[apialias.DispatchRequest, struct{}]{}
	t.KVDispatch = noop.UnaryServer[apikv.DispatchRequest, struct{}]{}

	// SCHEMATIC
	t.SchematicCreate = noop.UnaryServer[schematic.CreateRequest, schematic.CreateResponse]{}
	t.SchematicDelete = noop.UnaryServer[schematic.DeleteRequest, struct{}]{}
	t.SchematicRetrieve = noop.UnaryServer[schematic.RetrieveRequest, schematic.RetrieveResponse]{}
	t.SchematicDispatch = noop.UnaryServer[schematic.DispatchRequest, schematic.DispatchResponse]{}
	t.SchematicCopy = noop.UnaryServer[schematic.CopyRequest, schematic.CopyResponse]{}

	// SCHEMATIC SYMBOL
	t.SchematicSymbolCreate = noop.UnaryServer[symbol.CreateRequest, symbol.CreateResponse]{}
	t.SchematicSymbolRetrieve = noop.UnaryServer[symbol.RetrieveRequest, symbol.RetrieveResponse]{}
	t.SchematicSymbolDelete = noop.UnaryServer[symbol.DeleteRequest, struct{}]{}
	t.SchematicSymbolDispatch = noop.UnaryServer[symbol.DispatchRequest, struct{}]{}
	t.SchematicSymbolExportGroup = noop.UnaryServer[symbol.ExportGroupRequest, symbol.ExportGroupResponse]{}
	t.SchematicSymbolImportGroup = noop.UnaryServer[symbol.ImportGroupRequest, symbol.ImportGroupResponse]{}
	t.SchematicSymbolDeleteGroup = noop.UnaryServer[symbol.DeleteGroupRequest, struct{}]{}

	// LINE PLOT
	t.LinePlotCreate = noop.UnaryServer[lineplot.CreateRequest, lineplot.CreateResponse]{}
	t.LinePlotRetrieve = noop.UnaryServer[lineplot.RetrieveRequest, lineplot.RetrieveResponse]{}
	t.LinePlotDelete = noop.UnaryServer[lineplot.DeleteRequest, struct{}]{}
	t.LinePlotDispatch = noop.UnaryServer[lineplot.DispatchRequest, lineplot.DispatchResponse]{}

	// PANEL
	t.PanelCreate = noop.UnaryServer[panel.CreateRequest, panel.CreateResponse]{}
	t.PanelRetrieve = noop.UnaryServer[panel.RetrieveRequest, panel.RetrieveResponse]{}
	t.PanelDelete = noop.UnaryServer[panel.DeleteRequest, struct{}]{}
	t.PanelDispatch = noop.UnaryServer[panel.DispatchRequest, panel.DispatchResponse]{}

	// LOG
	t.LogCreate = noop.UnaryServer[log.CreateRequest, log.CreateResponse]{}
	t.LogRetrieve = noop.UnaryServer[log.RetrieveRequest, log.RetrieveResponse]{}
	t.LogDelete = noop.UnaryServer[log.DeleteRequest, struct{}]{}
	t.LogDispatch = noop.UnaryServer[log.DispatchRequest, log.DispatchResponse]{}

	// VIEW
	t.ViewCreate = noop.UnaryServer[view.CreateRequest, view.CreateResponse]{}
	t.ViewRetrieve = noop.UnaryServer[view.RetrieveRequest, view.RetrieveResponse]{}
	t.ViewDelete = noop.UnaryServer[view.DeleteRequest, struct{}]{}
	t.ViewDispatch = noop.UnaryServer[view.DispatchRequest, struct{}]{}

	// TABLE
	t.TableCreate = noop.UnaryServer[table.CreateRequest, table.CreateResponse]{}
	t.TableRetrieve = noop.UnaryServer[table.RetrieveRequest, table.RetrieveResponse]{}
	t.TableDelete = noop.UnaryServer[table.DeleteRequest, struct{}]{}
	t.TableDispatch = noop.UnaryServer[table.DispatchRequest, table.DispatchResponse]{}

	// LABEL
	t.LabelCreate = noop.UnaryServer[label.CreateRequest, label.CreateResponse]{}
	t.LabelRetrieve = noop.UnaryServer[label.RetrieveRequest, label.RetrieveResponse]{}
	t.LabelDelete = noop.UnaryServer[label.DeleteRequest, struct{}]{}
	t.LabelAdd = noop.UnaryServer[label.AddRequest, struct{}]{}
	t.LabelRemove = noop.UnaryServer[label.RemoveRequest, struct{}]{}
	t.LabelDispatch = noop.UnaryServer[label.DispatchRequest, struct{}]{}

	// ACCESS
	t.AccessCreatePolicy = noop.UnaryServer[policy.CreateRequest, policy.CreateResponse]{}
	t.AccessDeletePolicy = noop.UnaryServer[policy.DeleteRequest, struct{}]{}
	t.AccessRetrievePolicy = noop.UnaryServer[policy.RetrieveRequest, policy.RetrieveResponse]{}
	t.AccessCreateRole = noop.UnaryServer[role.CreateRequest, role.CreateResponse]{}
	t.AccessDeleteRole = noop.UnaryServer[role.DeleteRequest, struct{}]{}
	t.AccessRetrieveRole = noop.UnaryServer[role.RetrieveRequest, role.RetrieveResponse]{}
	t.AccessDispatchPolicy = noop.UnaryServer[policy.DispatchRequest, struct{}]{}
	t.AccessDispatchRole = noop.UnaryServer[role.DispatchRequest, struct{}]{}

	// SECRET
	t.SecretSet = noop.UnaryServer[apisecret.SetRequest, apisecret.SetResponse]{}
	t.SecretRetrieve = noop.UnaryServer[apisecret.RetrieveRequest, apisecret.RetrieveResponse]{}
	t.SecretDelete = noop.UnaryServer[apisecret.DeleteRequest, struct{}]{}
	t.SecretDispatch = noop.UnaryServer[apisecret.DispatchRequest, struct{}]{}
	t.SecretGrantCreate = noop.UnaryServer[apisecret.GrantCreateRequest, apisecret.GrantCreateResponse]{}
	t.SecretResolve = noop.UnaryServer[apisecret.ResolveRequest, apisecret.ResolveResponse]{}

	// CERTIFICATE
	t.CertificateSet = noop.UnaryServer[apicertificate.SetRequest, apicertificate.SetResponse]{}
	t.CertificateRetrieve = noop.UnaryServer[apicertificate.RetrieveRequest, apicertificate.RetrieveResponse]{}
	t.CertificateDelete = noop.UnaryServer[apicertificate.DeleteRequest, struct{}]{}
	t.CertificateDispatch = noop.UnaryServer[apicertificate.DispatchRequest, struct{}]{}

	// IMPORT/EXPORT
	t.ImExImport = noop.UnaryServer[imex.ImportRequest, imex.ImportResponse]{}
	t.ImExExport = noop.UnaryServer[imex.ExportRequest, imex.ExportResponse]{}

	// ARC
	t.ArcDispatch = noop.UnaryServer[apiarc.DispatchRequest, struct{}]{}
	t.ArcUpdateTask = noop.UnaryServer[apiarc.UpdateTaskRequest, apiarc.UpdateTaskResponse]{}
	t.ArcRetrieveDirectory = noop.UnaryServer[
		apiarcdirectory.RetrieveRequest,
		apiarcdirectory.RetrieveResponse,
	]{}
	t.ArcCreateDirectory = noop.UnaryServer[
		apiarc.CreateDirectoryRequest,
		apiarc.CreateDirectoryResponse,
	]{}
	t.ArcDispatchDirectory = noop.UnaryServer[apiarc.DispatchDirectoryRequest, struct{}]{}
	t.ArcDeleteDirectory = noop.UnaryServer[apiarc.DeleteDirectoryRequest, struct{}]{}
	t.ArcLSP = noop.StreamServer[apiarc.LSPMessage, apiarc.LSPMessage]{}

	layer.BindTo(t)
	return transports
}
