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
	"github.com/synnaxlabs/freighter/http"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/access/policy"
	"github.com/synnaxlabs/synnax/pkg/api/access/role"
	"github.com/synnaxlabs/synnax/pkg/api/arc"
	arcdirectory "github.com/synnaxlabs/synnax/pkg/api/arc/directory"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/batch"
	"github.com/synnaxlabs/synnax/pkg/api/certificate"
	"github.com/synnaxlabs/synnax/pkg/api/channel"
	"github.com/synnaxlabs/synnax/pkg/api/connectivity"
	"github.com/synnaxlabs/synnax/pkg/api/control"
	"github.com/synnaxlabs/synnax/pkg/api/device"
	"github.com/synnaxlabs/synnax/pkg/api/group"
	"github.com/synnaxlabs/synnax/pkg/api/imex"
	"github.com/synnaxlabs/synnax/pkg/api/label"
	labjackdevice "github.com/synnaxlabs/synnax/pkg/api/labjack/device"
	"github.com/synnaxlabs/synnax/pkg/api/license"
	"github.com/synnaxlabs/synnax/pkg/api/lineplot"
	"github.com/synnaxlabs/synnax/pkg/api/log"
	nichassis "github.com/synnaxlabs/synnax/pkg/api/ni/chassis"
	nidevice "github.com/synnaxlabs/synnax/pkg/api/ni/device"
	"github.com/synnaxlabs/synnax/pkg/api/panel"
	"github.com/synnaxlabs/synnax/pkg/api/project"
	"github.com/synnaxlabs/synnax/pkg/api/rack"
	"github.com/synnaxlabs/synnax/pkg/api/ranger"
	"github.com/synnaxlabs/synnax/pkg/api/ranger/kv"
	apiresource "github.com/synnaxlabs/synnax/pkg/api/resource"
	"github.com/synnaxlabs/synnax/pkg/api/schematic"
	"github.com/synnaxlabs/synnax/pkg/api/schematic/symbol"
	"github.com/synnaxlabs/synnax/pkg/api/secret"
	"github.com/synnaxlabs/synnax/pkg/api/status"
	"github.com/synnaxlabs/synnax/pkg/api/table"
	"github.com/synnaxlabs/synnax/pkg/api/task"
	"github.com/synnaxlabs/synnax/pkg/api/user"
	"github.com/synnaxlabs/synnax/pkg/api/view"
	"github.com/synnaxlabs/synnax/pkg/transport/http/framer"
	"github.com/synnaxlabs/x/encoding/zip"
)

// Bind registers an HTTP endpoint for every API service onto router and binds the API
// layer's handlers and middleware to them. The frame codec resolves channel keys
// through the API layer's channel service. Frame reads open their iterators under the
// router's context, so a streamed body can outlive its request.
func Bind(layer *api.Layer, router *http.Router) {
	ctx := router.Context()
	framerServerOption := framer.WithCodec(layer.Channel)
	csvEncoder := framer.NewCSVEncoder(ctx)
	layer.BindTo(api.Transport{
		// AUTH
		AuthLogin: unary[auth.LoginRequest, auth.LoginResponse](
			router,
			"/api/v1/auth/login",
		),
		AuthChangePassword: unary[auth.ChangePasswordRequest, struct{}](
			router,
			"/api/v1/auth/change-password",
		),
		AuthSessionCreate: unary[auth.SessionCreateRequest, auth.SessionCreateResponse](
			router,
			"/api/v1/auth/session/create",
		),
		AuthSessionRenew: unary[struct{}, auth.SessionRenewResponse](
			router,
			"/api/v1/auth/session/renew",
		),
		AuthSessionRetrieve: unary[auth.SessionRetrieveRequest, auth.SessionRetrieveResponse](
			router,
			"/api/v1/auth/session/retrieve",
		),
		AuthSessionDelete: unary[auth.SessionDeleteRequest, struct{}](
			router,
			"/api/v1/auth/session/delete",
		),
		AuthKeyCreate: unary[auth.KeyCreateRequest, auth.KeyCreateResponse](
			router,
			"/api/v1/auth/key/create",
		),
		AuthKeyRetrieve: unary[auth.KeyRetrieveRequest, auth.KeyRetrieveResponse](
			router,
			"/api/v1/auth/key/retrieve",
		),
		AuthKeyDelete: unary[auth.KeyDeleteRequest, struct{}](
			router,
			"/api/v1/auth/key/delete",
		),
		AuthPasswordSet: unary[auth.PasswordSetRequest, auth.PasswordSetResponse](
			router,
			"/api/v1/auth/password/set",
		),

		// USER
		UserCreate: unary[user.CreateRequest, user.CreateResponse](
			router,
			"/api/v1/user/create",
		),
		UserDelete: unary[user.DeleteRequest, struct{}](
			router,
			"/api/v1/user/delete",
		),
		UserRetrieve: unary[user.RetrieveRequest, user.RetrieveResponse](
			router,
			"/api/v1/user/retrieve",
		),
		UserDispatch: unary[user.DispatchRequest, struct{}](
			router,
			"/api/v1"+api.PathUserDispatch,
		),

		// CHANNEL
		ChannelCreate: unary[channel.CreateRequest, channel.CreateResponse](
			router,
			"/api/v1/channel/create",
		),
		ChannelRetrieve: unary[channel.RetrieveRequest, channel.RetrieveResponse](
			router,
			"/api/v1/channel/retrieve",
		),
		ChannelDelete: unary[channel.DeleteRequest, struct{}](
			router,
			"/api/v1/channel/delete",
		),
		ChannelDispatch: unary[channel.DispatchRequest, channel.DispatchResponse](
			router,
			"/api/v1"+api.PathChannelDispatch,
		),

		// CONNECTIVITY
		ConnectivityCheck: unary[struct{}, connectivity.CheckResponse](
			router,
			"/api/v1/connectivity/check",
		),

		// LICENSE
		LicenseRetrieve: unary[license.RetrieveRequest, license.RetrieveResponse](
			router,
			"/api/v1/license/retrieve",
		),
		LicenseActivate: unary[license.ActivateRequest, license.ActivateResponse](
			router,
			"/api/v1/license/activate",
		),
		LicenseDeactivate: unary[license.DeactivateRequest, license.DeactivateResponse](
			router,
			"/api/v1/license/deactivate",
		),

		// FRAME
		FrameWriter: stream[framer.WriterRequest, framer.WriterResponse](
			router,
			"/api/v1/frame/write",
			framerServerOption,
		),
		FrameIterator: stream[framer.IteratorRequest, framer.IteratorResponse](
			router,
			"/api/v1/frame/iterate",
			framerServerOption,
		),
		FrameStreamer: stream[framer.StreamerRequest, framer.StreamerResponse](
			router,
			"/api/v1/frame/stream",
			framerServerOption,
		),
		FrameDelete: unary[framer.DeleteRequest, struct{}](
			router,
			"/api/v1/frame/delete",
		),
		FrameRead: unary[framer.ReadRequest, framer.ReadResponse](
			router,
			"/api/v1/frame/read",
			http.WithResponseEncoders(
				framer.NewFrameEncoder(ctx),
				csvEncoder,
				framer.NewTDMSEncoder(ctx),
			),
			// CSV is decimal text and compresses by 70% or more. Frame and TDMS
			// bodies are binary samples: gzip shrinks them by under 10% and costs more
			// time than the smaller body saves on all but a slow link.
			http.WithStreamingResponse(csvEncoder),
		),

		// CONTROL
		ControlRetrieve: unary[control.RetrieveRequest, control.RetrieveResponse](
			router,
			"/api/v1/control/retrieve",
		),

		// RESOURCE
		ResourceSearch: unary[apiresource.SearchRequest, apiresource.SearchResponse](
			router,
			"/api/v1/resource/search",
		),
		ResourceRetrieve: unary[apiresource.RetrieveRequest, apiresource.RetrieveResponse](
			router,
			"/api/v1/resource/retrieve",
		),

		// GROUP
		GroupCreate: unary[group.CreateRequest, group.CreateResponse](
			router,
			"/api/v1"+api.PathGroupCreate,
		),
		GroupDelete: unary[group.DeleteRequest, struct{}](
			router,
			"/api/v1"+api.PathGroupDelete,
		),
		GroupDispatch: unary[group.DispatchRequest, struct{}](
			router,
			"/api/v1"+api.PathGroupDispatch,
		),
		GroupRetrieve: unary[group.RetrieveRequest, group.RetrieveResponse](
			router,
			"/api/v1/group/retrieve",
		),
		GroupRetrieveAncestors: unary[
			group.RetrieveAncestorsRequest,
			group.RetrieveAncestorsResponse,
		](router, "/api/v1/group/retrieve-ancestors"),
		Batch: unary[batch.ExecRequest, batch.ExecResponse](
			router,
			"/api/v1/batch",
		),

		// RANGE
		RangeRetrieve: unary[ranger.RetrieveRequest, ranger.RetrieveResponse](
			router,
			"/api/v1/range/retrieve",
		),
		RangeCreate: unary[ranger.CreateRequest, ranger.CreateResponse](
			router,
			"/api/v1/range/create",
		),
		RangeDelete: unary[ranger.DeleteRequest, struct{}](
			router,
			"/api/v1/range/delete",
		),
		RangeDispatch: unary[ranger.DispatchRequest, struct{}](
			router,
			"/api/v1"+api.PathRangeDispatch,
		),

		// KV
		KVGet:    unary[kv.GetRequest, kv.GetResponse](router, "/api/v1/range/kv/get"),
		KVSet:    unary[kv.SetRequest, struct{}](router, "/api/v1/range/kv/set"),
		KVDelete: unary[kv.DeleteRequest, struct{}](router, "/api/v1/range/kv/delete"),
		KVDispatch: unary[kv.DispatchRequest, struct{}](
			router,
			"/api/v1"+api.PathRangeKVDispatch,
		),

		// PROJECT
		ProjectCreate: unary[project.CreateRequest, project.CreateResponse](
			router,
			"/api/v1/project/create",
		),
		ProjectRetrieve: unary[project.RetrieveRequest, project.RetrieveResponse](
			router,
			"/api/v1/project/retrieve",
		),
		ProjectDelete: unary[project.DeleteRequest, struct{}](
			router,
			"/api/v1/project/delete",
		),
		ProjectDispatch: unary[project.DispatchRequest, struct{}](
			router,
			"/api/v1"+api.PathProjectDispatch,
		),
		ProjectExport: unary[project.ExportRequest, project.ExportResponse](
			router,
			"/api/v1/project/export",
			http.WithResponseEncoders(zip.Codec),
		),
		ProjectImport: unary[project.ImportRequest, project.ImportResponse](
			router,
			"/api/v1/project/import",
			http.WithRequestDecoders(zip.Codec),
		),

		// SCHEMATIC
		SchematicCreate: unary[schematic.CreateRequest, schematic.CreateResponse](
			router,
			"/api/v1/schematic/create",
		),
		SchematicRetrieve: unary[schematic.RetrieveRequest, schematic.RetrieveResponse](
			router,
			"/api/v1/schematic/retrieve",
		),
		SchematicDelete: unary[schematic.DeleteRequest, struct{}](
			router,
			"/api/v1/schematic/delete",
		),
		SchematicDispatch: unary[schematic.DispatchRequest, schematic.DispatchResponse](
			router,
			"/api/v1"+api.PathSchematicDispatch,
		),
		SchematicCopy: unary[schematic.CopyRequest, schematic.CopyResponse](
			router,
			"/api/v1/schematic/copy",
		),

		// SCHEMATIC SYMBOL
		SchematicSymbolCreate: unary[symbol.CreateRequest, symbol.CreateResponse](
			router,
			"/api/v1/schematic/symbol/create",
		),
		SchematicSymbolRetrieve: unary[symbol.RetrieveRequest, symbol.RetrieveResponse](
			router,
			"/api/v1/schematic/symbol/retrieve",
		),
		SchematicSymbolDelete: unary[symbol.DeleteRequest, struct{}](
			router,
			"/api/v1/schematic/symbol/delete",
		),
		SchematicSymbolDispatch: unary[symbol.DispatchRequest, struct{}](
			router,
			"/api/v1"+api.PathSchematicSymbolDispatch,
		),
		SchematicSymbolExportGroup: unary[symbol.ExportGroupRequest, symbol.ExportGroupResponse](
			router,
			"/api/v1/schematic/symbol/group/export",
			http.WithResponseEncoders(zip.Codec),
		),
		SchematicSymbolImportGroup: unary[symbol.ImportGroupRequest, symbol.ImportGroupResponse](
			router,
			"/api/v1/schematic/symbol/group/import",
			http.WithRequestDecoders(zip.Codec),
		),
		SchematicSymbolDeleteGroup: unary[symbol.DeleteGroupRequest, struct{}](
			router,
			"/api/v1/schematic/symbol/group/delete",
		),

		// LINE PLOT
		LinePlotCreate: unary[lineplot.CreateRequest, lineplot.CreateResponse](
			router,
			"/api/v1/lineplot/create",
		),
		LinePlotRetrieve: unary[lineplot.RetrieveRequest, lineplot.RetrieveResponse](
			router,
			"/api/v1/lineplot/retrieve",
		),
		LinePlotDelete: unary[lineplot.DeleteRequest, struct{}](
			router,
			"/api/v1/lineplot/delete",
		),
		LinePlotDispatch: unary[lineplot.DispatchRequest, lineplot.DispatchResponse](
			router,
			"/api/v1"+api.PathLinePlotDispatch,
		),

		// PANEL
		PanelCreate: unary[panel.CreateRequest, panel.CreateResponse](
			router,
			"/api/v1/panel/create",
		),
		PanelRetrieve: unary[panel.RetrieveRequest, panel.RetrieveResponse](
			router,
			"/api/v1/panel/retrieve",
		),
		PanelDelete: unary[panel.DeleteRequest, struct{}](
			router,
			"/api/v1/panel/delete",
		),
		PanelDispatch: unary[panel.DispatchRequest, panel.DispatchResponse](
			router,
			"/api/v1"+api.PathPanelDispatch,
		),

		// LOG
		LogCreate: unary[log.CreateRequest, log.CreateResponse](
			router,
			"/api/v1/log/create",
		),
		LogRetrieve: unary[log.RetrieveRequest, log.RetrieveResponse](
			router,
			"/api/v1/log/retrieve",
		),
		LogDelete: unary[log.DeleteRequest, struct{}](router, "/api/v1/log/delete"),
		LogDispatch: unary[log.DispatchRequest, log.DispatchResponse](
			router,
			"/api/v1"+api.PathLogDispatch,
		),

		// TABLE
		TableCreate: unary[table.CreateRequest, table.CreateResponse](
			router,
			"/api/v1/table/create",
		),
		TableRetrieve: unary[table.RetrieveRequest, table.RetrieveResponse](
			router,
			"/api/v1/table/retrieve",
		),
		TableDelete: unary[table.DeleteRequest, struct{}](
			router,
			"/api/v1/table/delete",
		),
		TableDispatch: unary[table.DispatchRequest, table.DispatchResponse](
			router,
			"/api/v1"+api.PathTableDispatch,
		),

		// LABEL
		LabelCreate: unary[label.CreateRequest, label.CreateResponse](
			router,
			"/api/v1/label/create",
		),
		LabelRetrieve: unary[label.RetrieveRequest, label.RetrieveResponse](
			router,
			"/api/v1/label/retrieve",
		),
		LabelDelete: unary[label.DeleteRequest, struct{}](
			router,
			"/api/v1/label/delete",
		),
		LabelAdd: unary[label.AddRequest, struct{}](router, "/api/v1"+api.PathLabelSet),
		LabelRemove: unary[label.RemoveRequest, struct{}](
			router,
			"/api/v1"+api.PathLabelRemove,
		),
		LabelDispatch: unary[label.DispatchRequest, struct{}](
			router,
			"/api/v1"+api.PathLabelDispatch,
		),

		// RACK
		RackCreate: unary[rack.CreateRequest, rack.CreateResponse](
			router,
			"/api/v1/rack/create",
		),
		RackRetrieve: unary[rack.RetrieveRequest, rack.RetrieveResponse](
			router,
			"/api/v1/rack/retrieve",
		),
		RackDelete: unary[rack.DeleteRequest, struct{}](
			router,
			"/api/v1/rack/delete",
		),
		RackDispatch: unary[rack.DispatchRequest, struct{}](
			router,
			"/api/v1"+api.PathRackDispatch,
		),

		// TASK
		TaskCreate: unary[task.CreateRequest, task.CreateResponse](
			router,
			"/api/v1/task/create",
		),
		TaskRetrieve: unary[task.RetrieveRequest, task.RetrieveResponse](
			router,
			"/api/v1/task/retrieve",
		),
		TaskDelete: unary[task.DeleteRequest, struct{}](
			router,
			"/api/v1/task/delete",
		),
		TaskCopy: unary[task.CopyRequest, task.CopyResponse](
			router,
			"/api/v1/task/copy",
		),
		TaskDispatch: unary[task.DispatchRequest, task.DispatchResponse](
			router,
			"/api/v1"+api.PathTaskDispatch,
		),

		// DEVICE
		DeviceCreate: unary[device.CreateRequest, device.CreateResponse](
			router,
			"/api/v1/device/create",
		),
		DeviceRetrieve: unary[device.RetrieveRequest, device.RetrieveResponse](
			router,
			"/api/v1/device/retrieve",
		),
		DeviceDelete: unary[device.DeleteRequest, struct{}](
			router,
			"/api/v1/device/delete",
		),
		DeviceDispatch: unary[device.DispatchRequest, struct{}](
			router,
			"/api/v1"+api.PathDeviceDispatch,
		),

		// NI DEVICE
		NIDeviceCreate: unary[nidevice.CreateRequest, nidevice.CreateResponse](
			router,
			"/api/v1/ni/device/create",
		),
		NIDeviceRetrieve: unary[nidevice.RetrieveRequest, nidevice.RetrieveResponse](
			router,
			"/api/v1/ni/device/retrieve",
		),
		NIDeviceUpdate: unary[nidevice.UpdateRequest, struct{}](
			router,
			"/api/v1/ni/device/update",
		),
		NIDeviceDelete: unary[nidevice.DeleteRequest, struct{}](
			router,
			"/api/v1/ni/device/delete",
		),

		// NI CHASSIS
		NIChassisCreate: unary[nichassis.CreateRequest, nichassis.CreateResponse](
			router,
			"/api/v1/ni/chassis/create",
		),
		NIChassisRetrieve: unary[nichassis.RetrieveRequest, nichassis.RetrieveResponse](
			router,
			"/api/v1/ni/chassis/retrieve",
		),
		NIChassisUpdate: unary[nichassis.UpdateRequest, struct{}](
			router,
			"/api/v1/ni/chassis/update",
		),
		NIChassisDelete: unary[nichassis.DeleteRequest, struct{}](
			router,
			"/api/v1/ni/chassis/delete",
		),

		// LABJACK DEVICE
		LabJackDeviceCreate: unary[labjackdevice.CreateRequest, labjackdevice.CreateResponse](
			router,
			"/api/v1/labjack/device/create",
		),
		LabJackDeviceRetrieve: unary[labjackdevice.RetrieveRequest, labjackdevice.RetrieveResponse](
			router,
			"/api/v1/labjack/device/retrieve",
		),
		LabJackDeviceUpdate: unary[labjackdevice.UpdateRequest, struct{}](
			router,
			"/api/v1/labjack/device/update",
		),
		LabJackDeviceDelete: unary[labjackdevice.DeleteRequest, struct{}](
			router,
			"/api/v1/labjack/device/delete",
		),

		// ACCESS
		AccessCreatePolicy: unary[policy.CreateRequest, policy.CreateResponse](
			router,
			"/api/v1/access/policy/create",
		),
		AccessDeletePolicy: unary[policy.DeleteRequest, struct{}](
			router,
			"/api/v1/access/policy/delete",
		),
		AccessRetrievePolicy: unary[policy.RetrieveRequest, policy.RetrieveResponse](
			router,
			"/api/v1/access/policy/retrieve",
		),
		AccessCreateRole: unary[role.CreateRequest, role.CreateResponse](
			router,
			"/api/v1/access/role/create",
		),
		AccessDeleteRole: unary[role.DeleteRequest, struct{}](
			router,
			"/api/v1/access/role/delete",
		),
		AccessRetrieveRole: unary[role.RetrieveRequest, role.RetrieveResponse](
			router,
			"/api/v1/access/role/retrieve",
		),
		AccessDispatchPolicy: unary[policy.DispatchRequest, struct{}](
			router,
			"/api/v1"+api.PathPolicyDispatch,
		),
		AccessDispatchRole: unary[role.DispatchRequest, struct{}](
			router,
			"/api/v1"+api.PathRoleDispatch,
		),

		// ARC
		ArcCreate: unary[arc.CreateRequest, arc.CreateResponse](
			router,
			"/api/v1/arc/create",
		),
		ArcDelete: unary[arc.DeleteRequest, struct{}](router, "/api/v1/arc/delete"),
		ArcRetrieve: unary[arc.RetrieveRequest, arc.RetrieveResponse](
			router,
			"/api/v1/arc/retrieve",
		),
		ArcDispatch: unary[arc.DispatchRequest, struct{}](
			router,
			"/api/v1"+api.PathArcDispatch,
		),
		ArcUpdateTask: unary[arc.UpdateTaskRequest, arc.UpdateTaskResponse](
			router,
			"/api/v1/arc/update-task",
		),
		ArcRetrieveDirectory: unary[
			arcdirectory.RetrieveRequest,
			arcdirectory.RetrieveResponse,
		](router, "/api/v1/arc/directory/retrieve"),
		ArcCreateDirectory: unary[arc.CreateDirectoryRequest, arc.CreateDirectoryResponse](
			router,
			"/api/v1/arc/directory/create",
		),
		ArcDispatchDirectory: unary[arc.DispatchDirectoryRequest, struct{}](
			router,
			"/api/v1"+api.PathArcDirectoryDispatch,
		),
		ArcDeleteDirectory: unary[arc.DeleteDirectoryRequest, struct{}](
			router,
			"/api/v1/arc/directory/delete",
		),
		ArcLSP: stream[arc.LSPMessage, arc.LSPMessage](
			router,
			"/api/v1/arc/lsp",
		),

		// STATUS
		StatusSet: unary[status.SetRequest, status.SetResponse](
			router,
			"/api/v1"+api.PathStatusSet,
		),
		StatusRetrieve: unary[status.RetrieveRequest, status.RetrieveResponse](
			router,
			"/api/v1/status/retrieve",
		),
		StatusDelete: unary[status.DeleteRequest, struct{}](
			router,
			"/api/v1/status/delete",
		),
		StatusSetByKeyOrName: unary[status.SetByKeyOrNameRequest, status.SetByKeyOrNameResponse](
			router,
			"/api/v1/status/set-by-key-or-name",
		),
		StatusRename: unary[status.RenameRequest, struct{}](
			router,
			"/api/v1/status/rename",
		),

		// VIEW
		ViewCreate: unary[view.CreateRequest, view.CreateResponse](
			router,
			"/api/v1/view/create",
		),
		ViewRetrieve: unary[view.RetrieveRequest, view.RetrieveResponse](
			router,
			"/api/v1/view/retrieve",
		),
		ViewDelete: unary[view.DeleteRequest, struct{}](
			router,
			"/api/v1/view/delete",
		),
		ViewDispatch: unary[view.DispatchRequest, struct{}](
			router,
			"/api/v1"+api.PathViewDispatch,
		),

		// SECRET
		SecretSet: unary[secret.SetRequest, secret.SetResponse](
			router,
			"/api/v1/secret/set",
		),
		SecretRetrieve: unary[secret.RetrieveRequest, secret.RetrieveResponse](
			router,
			"/api/v1/secret/retrieve",
		),
		SecretDelete: unary[secret.DeleteRequest, struct{}](
			router,
			"/api/v1/secret/delete",
		),
		SecretDispatch: unary[secret.DispatchRequest, struct{}](
			router,
			"/api/v1"+api.PathSecretDispatch,
		),
		SecretGrantCreate: unary[secret.GrantCreateRequest, secret.GrantCreateResponse](
			router,
			"/api/v1/secret/grant/create",
		),
		SecretResolve: unary[secret.ResolveRequest, secret.ResolveResponse](
			router,
			"/api/v1/secret/resolve",
		),

		// CERTIFICATE
		CertificateSet: unary[certificate.SetRequest, certificate.SetResponse](
			router,
			"/api/v1/certificate/set",
		),
		CertificateRetrieve: unary[certificate.RetrieveRequest, certificate.RetrieveResponse](
			router,
			"/api/v1/certificate/retrieve",
		),
		CertificateDelete: unary[certificate.DeleteRequest, struct{}](
			router,
			"/api/v1/certificate/delete",
		),
		CertificateDispatch: unary[certificate.DispatchRequest, struct{}](
			router,
			"/api/v1"+api.PathCertificateDispatch,
		),

		// IMPORT/EXPORT
		ImExImport: unary[imex.ImportRequest, imex.ImportResponse](
			router,
			"/api/v1/imex/import",
			http.WithRequestDecoders(imex.ImportCodec),
		),
		ImExExport: unary[imex.ExportRequest, imex.ExportResponse](
			router,
			"/api/v1/imex/export",
			http.WithResponseEncoders(imex.JSONCodec),
		),
	})
}
