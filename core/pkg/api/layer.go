// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package api implements the client interfaces for interacting with the Synnax cluster.
// The package is transport agnostic, defining freighter-compatible interfaces (via the
// Transport struct) and service implementations (via the Layer struct) for all of its
// services.
package api

import (
	"context"
	"slices"

	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/alamos"
	fgorp "github.com/synnaxlabs/freighter/gorp"
	"github.com/synnaxlabs/freighter/recovery"
	"github.com/synnaxlabs/synnax/pkg/api/access/policy"
	"github.com/synnaxlabs/synnax/pkg/api/access/role"
	"github.com/synnaxlabs/synnax/pkg/api/arc"
	arcdirectory "github.com/synnaxlabs/synnax/pkg/api/arc/directory"
	"github.com/synnaxlabs/synnax/pkg/api/auth"
	"github.com/synnaxlabs/synnax/pkg/api/batch"
	"github.com/synnaxlabs/synnax/pkg/api/bind"
	"github.com/synnaxlabs/synnax/pkg/api/channel"
	"github.com/synnaxlabs/synnax/pkg/api/config"
	"github.com/synnaxlabs/synnax/pkg/api/connectivity"
	"github.com/synnaxlabs/synnax/pkg/api/control"
	"github.com/synnaxlabs/synnax/pkg/api/device"
	"github.com/synnaxlabs/synnax/pkg/api/framer"
	"github.com/synnaxlabs/synnax/pkg/api/group"
	"github.com/synnaxlabs/synnax/pkg/api/imex"
	"github.com/synnaxlabs/synnax/pkg/api/label"
	"github.com/synnaxlabs/synnax/pkg/api/license"
	"github.com/synnaxlabs/synnax/pkg/api/lineplot"
	"github.com/synnaxlabs/synnax/pkg/api/log"
	"github.com/synnaxlabs/synnax/pkg/api/panel"
	"github.com/synnaxlabs/synnax/pkg/api/project"
	"github.com/synnaxlabs/synnax/pkg/api/rack"
	"github.com/synnaxlabs/synnax/pkg/api/ranger"
	"github.com/synnaxlabs/synnax/pkg/api/ranger/alias"
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
	xconfig "github.com/synnaxlabs/x/config"
)

// LayerConfig is the configuration for opening the API layer.
type LayerConfig = config.LayerConfig

var DefaultLayerConfig = config.DefaultLayerConfig

type Transport struct {
	// AUTH
	AuthLogin           freighter.UnaryServer[auth.LoginRequest, auth.LoginResponse]
	AuthChangePassword  freighter.UnaryServer[auth.ChangePasswordRequest, struct{}]
	AuthSessionCreate   freighter.UnaryServer[auth.SessionCreateRequest, auth.SessionCreateResponse]
	AuthSessionRenew    freighter.UnaryServer[struct{}, auth.SessionRenewResponse]
	AuthSessionRetrieve freighter.UnaryServer[auth.SessionRetrieveRequest, auth.SessionRetrieveResponse]
	AuthSessionDelete   freighter.UnaryServer[auth.SessionDeleteRequest, struct{}]
	AuthKeyCreate       freighter.UnaryServer[auth.KeyCreateRequest, auth.KeyCreateResponse]
	AuthKeyRetrieve     freighter.UnaryServer[auth.KeyRetrieveRequest, auth.KeyRetrieveResponse]
	AuthKeyDelete       freighter.UnaryServer[auth.KeyDeleteRequest, struct{}]
	// AuthPasswordSet sets the caller's password or another user's.
	AuthPasswordSet freighter.UnaryServer[auth.PasswordSetRequest, auth.PasswordSetResponse]
	// USER
	UserCreate   freighter.UnaryServer[user.CreateRequest, user.CreateResponse]
	UserDelete   freighter.UnaryServer[user.DeleteRequest, struct{}]
	UserRetrieve freighter.UnaryServer[user.RetrieveRequest, user.RetrieveResponse]
	// UserDispatch applies actions to a user.
	UserDispatch freighter.UnaryServer[user.DispatchRequest, struct{}]
	// CHANNEL
	ChannelCreate   freighter.UnaryServer[channel.CreateRequest, channel.CreateResponse]
	ChannelRetrieve freighter.UnaryServer[channel.AliasedRetrieveRequest, channel.RetrieveResponse]
	ChannelDelete   freighter.UnaryServer[channel.DeleteRequest, struct{}]
	// ChannelDispatch applies actions to a channel.
	ChannelDispatch freighter.UnaryServer[channel.DispatchRequest, channel.DispatchResponse]
	// CONNECTIVITY
	ConnectivityCheck freighter.UnaryServer[struct{}, connectivity.CheckResponse]
	// LICENSE
	LicenseRetrieve   freighter.UnaryServer[license.RetrieveRequest, license.RetrieveResponse]
	LicenseActivate   freighter.UnaryServer[license.ActivateRequest, license.ActivateResponse]
	LicenseDeactivate freighter.UnaryServer[license.DeactivateRequest, license.DeactivateResponse]
	// FRAME
	FrameWriter   freighter.StreamServer[framer.WriterRequest, framer.WriterResponse]
	FrameIterator freighter.StreamServer[framer.IteratorRequest, framer.IteratorResponse]
	FrameStreamer freighter.StreamServer[framer.StreamerRequest, framer.StreamerResponse]
	FrameDelete   freighter.UnaryServer[framer.DeleteRequest, struct{}]
	FrameRead     freighter.UnaryServer[framer.ReadRequest, framer.ReadResponse]
	// CONTROL
	ControlRetrieve freighter.UnaryServer[control.RetrieveRequest, control.RetrieveResponse]
	// RANGE
	RangeCreate   freighter.UnaryServer[ranger.CreateRequest, ranger.CreateResponse]
	RangeRetrieve freighter.UnaryServer[ranger.RetrieveRequest, ranger.RetrieveResponse]
	RangeDelete   freighter.UnaryServer[ranger.DeleteRequest, struct{}]
	// RangeDispatch applies path actions to a range.
	RangeDispatch freighter.UnaryServer[ranger.DispatchRequest, struct{}]
	// KV
	KVGet    freighter.UnaryServer[kv.GetRequest, kv.GetResponse]
	KVSet    freighter.UnaryServer[kv.SetRequest, struct{}]
	KVDelete freighter.UnaryServer[kv.DeleteRequest, struct{}]
	// KVDispatch applies path actions to a range key-value pair.
	KVDispatch freighter.UnaryServer[kv.DispatchRequest, struct{}]
	// ALIAS
	AliasSet      freighter.UnaryServer[alias.SetRequest, struct{}]
	AliasResolve  freighter.UnaryServer[alias.ResolveRequest, alias.ResolveResponse]
	AliasDelete   freighter.UnaryServer[alias.DeleteRequest, struct{}]
	AliasList     freighter.UnaryServer[alias.ListRequest, alias.ListResponse]
	AliasRetrieve freighter.UnaryServer[alias.RetrieveRequest, alias.RetrieveResponse]
	// AliasDispatch applies path actions to a range alias.
	AliasDispatch freighter.UnaryServer[alias.DispatchRequest, struct{}]
	// RESOURCE
	ResourceSearch   freighter.UnaryServer[apiresource.SearchRequest, apiresource.SearchResponse]
	ResourceRetrieve freighter.UnaryServer[apiresource.RetrieveRequest, apiresource.RetrieveResponse]
	// GROUP
	GroupCreate            freighter.UnaryServer[group.CreateRequest, group.CreateResponse]
	GroupDelete            freighter.UnaryServer[group.DeleteRequest, struct{}]
	GroupDispatch          freighter.UnaryServer[group.DispatchRequest, struct{}]
	GroupRetrieve          freighter.UnaryServer[group.RetrieveRequest, group.RetrieveResponse]
	GroupRetrieveAncestors freighter.UnaryServer[group.RetrieveAncestorsRequest, group.RetrieveAncestorsResponse]
	// PROJECT
	ProjectCreate   freighter.UnaryServer[project.CreateRequest, project.CreateResponse]
	ProjectRetrieve freighter.UnaryServer[project.RetrieveRequest, project.RetrieveResponse]
	ProjectDelete   freighter.UnaryServer[project.DeleteRequest, struct{}]
	ProjectDispatch freighter.UnaryServer[project.DispatchRequest, struct{}]
	ProjectExport   freighter.UnaryServer[project.ExportRequest, project.ExportResponse]
	ProjectImport   freighter.UnaryServer[project.ImportRequest, project.ImportResponse]
	// SCHEMATIC
	SchematicCreate   freighter.UnaryServer[schematic.CreateRequest, schematic.CreateResponse]
	SchematicRetrieve freighter.UnaryServer[schematic.RetrieveRequest, schematic.RetrieveResponse]
	SchematicDelete   freighter.UnaryServer[schematic.DeleteRequest, struct{}]
	SchematicDispatch freighter.UnaryServer[schematic.DispatchRequest, schematic.DispatchResponse]
	SchematicCopy     freighter.UnaryServer[schematic.CopyRequest, schematic.CopyResponse]
	// SCHEMATIC SYMBOL
	SchematicSymbolCreate      freighter.UnaryServer[symbol.CreateRequest, symbol.CreateResponse]
	SchematicSymbolRetrieve    freighter.UnaryServer[symbol.RetrieveRequest, symbol.RetrieveResponse]
	SchematicSymbolDelete      freighter.UnaryServer[symbol.DeleteRequest, struct{}]
	SchematicSymbolDispatch    freighter.UnaryServer[symbol.DispatchRequest, struct{}]
	SchematicSymbolExportGroup freighter.UnaryServer[symbol.ExportGroupRequest, symbol.ExportGroupResponse]
	SchematicSymbolImportGroup freighter.UnaryServer[symbol.ImportGroupRequest, symbol.ImportGroupResponse]
	SchematicSymbolDeleteGroup freighter.UnaryServer[symbol.DeleteGroupRequest, struct{}]
	// LOG
	LogCreate   freighter.UnaryServer[log.CreateRequest, log.CreateResponse]
	LogRetrieve freighter.UnaryServer[log.RetrieveRequest, log.RetrieveResponse]
	LogDelete   freighter.UnaryServer[log.DeleteRequest, struct{}]
	LogDispatch freighter.UnaryServer[log.DispatchRequest, log.DispatchResponse]
	// TABLE
	TableCreate   freighter.UnaryServer[table.CreateRequest, table.CreateResponse]
	TableRetrieve freighter.UnaryServer[table.RetrieveRequest, table.RetrieveResponse]
	TableDelete   freighter.UnaryServer[table.DeleteRequest, struct{}]
	TableDispatch freighter.UnaryServer[table.DispatchRequest, table.DispatchResponse]
	// LINE PLOT
	LinePlotCreate   freighter.UnaryServer[lineplot.CreateRequest, lineplot.CreateResponse]
	LinePlotRetrieve freighter.UnaryServer[lineplot.RetrieveRequest, lineplot.RetrieveResponse]
	LinePlotDelete   freighter.UnaryServer[lineplot.DeleteRequest, struct{}]
	LinePlotDispatch freighter.UnaryServer[lineplot.DispatchRequest, lineplot.DispatchResponse]
	// PANEL
	PanelCreate   freighter.UnaryServer[panel.CreateRequest, panel.CreateResponse]
	PanelRetrieve freighter.UnaryServer[panel.RetrieveRequest, panel.RetrieveResponse]
	PanelDelete   freighter.UnaryServer[panel.DeleteRequest, struct{}]
	PanelDispatch freighter.UnaryServer[panel.DispatchRequest, panel.DispatchResponse]
	// LABEL
	LabelCreate   freighter.UnaryServer[label.CreateRequest, label.CreateResponse]
	LabelRetrieve freighter.UnaryServer[label.RetrieveRequest, label.RetrieveResponse]
	LabelDelete   freighter.UnaryServer[label.DeleteRequest, struct{}]
	LabelAdd      freighter.UnaryServer[label.AddRequest, struct{}]
	LabelRemove   freighter.UnaryServer[label.RemoveRequest, struct{}]
	// LabelDispatch applies path actions to a label.
	LabelDispatch freighter.UnaryServer[label.DispatchRequest, struct{}]
	// RACK
	RackCreate   freighter.UnaryServer[rack.CreateRequest, rack.CreateResponse]
	RackRetrieve freighter.UnaryServer[rack.RetrieveRequest, rack.RetrieveResponse]
	RackDelete   freighter.UnaryServer[rack.DeleteRequest, struct{}]
	// RackDispatch applies actions to a rack.
	RackDispatch freighter.UnaryServer[rack.DispatchRequest, struct{}]
	// TASK
	TaskCreate   freighter.UnaryServer[task.CreateRequest, task.CreateResponse]
	TaskRetrieve freighter.UnaryServer[task.RetrieveRequest, task.RetrieveResponse]
	TaskDelete   freighter.UnaryServer[task.DeleteRequest, struct{}]
	TaskCopy     freighter.UnaryServer[task.CopyRequest, task.CopyResponse]
	// TaskDispatch applies path actions to a task.
	TaskDispatch freighter.UnaryServer[task.DispatchRequest, task.DispatchResponse]
	// DEVICE
	DeviceCreate   freighter.UnaryServer[device.CreateRequest, device.CreateResponse]
	DeviceRetrieve freighter.UnaryServer[device.RetrieveRequest, device.RetrieveResponse]
	DeviceDelete   freighter.UnaryServer[device.DeleteRequest, struct{}]
	// DeviceDispatch applies actions to a device.
	DeviceDispatch freighter.UnaryServer[device.DispatchRequest, struct{}]
	// ACCESS
	AccessCreatePolicy   freighter.UnaryServer[policy.CreateRequest, policy.CreateResponse]
	AccessDeletePolicy   freighter.UnaryServer[policy.DeleteRequest, struct{}]
	AccessRetrievePolicy freighter.UnaryServer[policy.RetrieveRequest, policy.RetrieveResponse]
	AccessCreateRole     freighter.UnaryServer[role.CreateRequest, role.CreateResponse]
	AccessDeleteRole     freighter.UnaryServer[role.DeleteRequest, struct{}]
	AccessRetrieveRole   freighter.UnaryServer[role.RetrieveRequest, role.RetrieveResponse]
	// AccessDispatchPolicy applies path actions to a policy.
	AccessDispatchPolicy freighter.UnaryServer[policy.DispatchRequest, struct{}]
	// AccessDispatchRole applies path actions to a role.
	AccessDispatchRole freighter.UnaryServer[role.DispatchRequest, struct{}]
	// STATUS
	StatusSet      freighter.UnaryServer[status.SetRequest, status.SetResponse]
	StatusRetrieve freighter.UnaryServer[status.RetrieveRequest, status.RetrieveResponse]
	StatusDelete   freighter.UnaryServer[status.DeleteRequest, struct{}]
	// StatusRename renames a status.
	StatusRename         freighter.UnaryServer[status.RenameRequest, struct{}]
	StatusSetByKeyOrName freighter.UnaryServer[status.SetByKeyOrNameRequest, status.SetByKeyOrNameResponse]
	// ARC
	ArcCreate            freighter.UnaryServer[arc.CreateRequest, arc.CreateResponse]
	ArcDelete            freighter.UnaryServer[arc.DeleteRequest, struct{}]
	ArcRetrieve          freighter.UnaryServer[arc.RetrieveRequest, arc.RetrieveResponse]
	ArcDispatch          freighter.UnaryServer[arc.DispatchRequest, struct{}]
	ArcUpdateTask        freighter.UnaryServer[arc.UpdateTaskRequest, arc.UpdateTaskResponse]
	ArcRetrieveDirectory freighter.UnaryServer[arcdirectory.RetrieveRequest, arcdirectory.RetrieveResponse]
	ArcCreateDirectory   freighter.UnaryServer[arc.CreateDirectoryRequest, arc.CreateDirectoryResponse]
	ArcDispatchDirectory freighter.UnaryServer[arc.DispatchDirectoryRequest, struct{}]
	ArcDeleteDirectory   freighter.UnaryServer[arc.DeleteDirectoryRequest, struct{}]
	ArcLSP               freighter.StreamServer[arc.LSPMessage, arc.LSPMessage]
	// VIEW
	ViewCreate   freighter.UnaryServer[view.CreateRequest, view.CreateResponse]
	ViewRetrieve freighter.UnaryServer[view.RetrieveRequest, view.RetrieveResponse]
	ViewDelete   freighter.UnaryServer[view.DeleteRequest, struct{}]
	// SECRET
	SecretSet      freighter.UnaryServer[secret.SetRequest, secret.SetResponse]
	SecretRetrieve freighter.UnaryServer[secret.RetrieveRequest, secret.RetrieveResponse]
	SecretDelete   freighter.UnaryServer[secret.DeleteRequest, struct{}]
	// SecretDispatch applies path actions to a secret.
	SecretDispatch    freighter.UnaryServer[secret.DispatchRequest, struct{}]
	SecretGrantCreate freighter.UnaryServer[secret.GrantCreateRequest, secret.GrantCreateResponse]
	SecretResolve     freighter.UnaryServer[secret.ResolveRequest, secret.ResolveResponse]
	// ViewDispatch applies path actions to a view.
	ViewDispatch freighter.UnaryServer[view.DispatchRequest, struct{}]
	// IMPORT/EXPORT
	ImExImport freighter.UnaryServer[imex.ImportRequest, imex.ImportResponse]
	ImExExport freighter.UnaryServer[imex.ExportRequest, imex.ExportResponse]
	// BATCH
	Batch freighter.UnaryServer[batch.ExecRequest, batch.ExecResponse]
}

// Layer wraps all implemented API services into a single container. Protocol-specific
// Layer implementations should use this struct during instantiation.
type Layer struct {
	Project      *project.Service
	LinePlot     *lineplot.Service
	User         *user.Service
	Framer       *framer.Service
	Channel      *channel.Service
	Control      *control.Service
	Connectivity *connectivity.Service
	License      *license.Service
	Resource     *apiresource.Service
	Range        *ranger.Service
	KV           *kv.Service
	Alias        *alias.Service
	Group        *group.Service
	Log          *log.Service
	Auth         *auth.Service
	Schematic    *schematic.Service
	Symbol       *symbol.Service
	View         *view.Service
	Secret       *secret.Service
	Table        *table.Service
	Panel        *panel.Service
	Label        *label.Service
	Rack         *rack.Service
	Task         *task.Service
	Device       *device.Service
	Policy       *policy.Service
	Role         *role.Service
	Arc          *arc.Service
	Status       *status.Service
	ImEx         *imex.Service
	Batch        *batch.Service
	binder       *bind.Service
	config       config.LayerConfig
}

// BindTo binds the API layer to the provided Transport implementation.
func (l *Layer) BindTo(t Transport) {
	var (
		tk              = auth.Middleware(l.config.Service.Auth)
		instrumentation = lo.Must(
			alamos.Middleware(alamos.Config{Instrumentation: l.config.Instrumentation}),
		)
		rec      = recovery.Middleware(l.config.Instrumentation)
		gate     = license.Middleware(l.config.Service.License)
		restrict = restrictMiddleware(
			l.config.Distribution.DB,
			l.config.Service.Resource,
			l.config.Service.RBAC,
		)
		insecureMiddleware = []freighter.Middleware{rec, instrumentation}
		secureMiddleware   = append(slices.Clone(insecureMiddleware), tk)
		gatedMiddleware    = append(slices.Clone(secureMiddleware), gate, restrict)
	)

	freighter.UseOnAll(
		insecureMiddleware,
		t.AuthLogin,
		t.AuthSessionCreate,
		t.ConnectivityCheck,
	)

	// Session upkeep skips the gate so a client can renew and log out of an unlicensed
	// Core.
	freighter.UseOnAll(
		secureMiddleware,
		t.AuthSessionRenew,
		t.AuthSessionRetrieve,
		t.AuthSessionDelete,
	)

	// The license endpoints skip the gate so an unlicensed Core can be activated.
	freighter.UseOnAll(
		secureMiddleware,
		t.LicenseRetrieve,
		t.LicenseActivate,
		t.LicenseDeactivate,
	)

	freighter.UseOnAll(
		gatedMiddleware,

		// AUTH
		t.AuthChangePassword,
		t.AuthKeyCreate,
		t.AuthKeyRetrieve,
		t.AuthKeyDelete,
		t.AuthPasswordSet,

		// USER
		t.UserCreate,
		t.UserDelete,
		t.UserRetrieve,
		t.UserDispatch,

		// CHANNEL
		t.ChannelCreate,
		t.ChannelRetrieve,
		t.ChannelDelete,
		t.ChannelDispatch,

		// FRAME
		t.FrameWriter,
		t.FrameIterator,
		t.FrameStreamer,
		t.FrameDelete,
		t.FrameRead,

		// CONTROL
		t.ControlRetrieve,

		// RESOURCE
		t.ResourceSearch,
		t.ResourceRetrieve,

		// GROUP
		t.GroupCreate,
		t.GroupDelete,
		t.GroupDispatch,
		t.GroupRetrieve,
		t.GroupRetrieveAncestors,

		// RANGE
		t.RangeCreate,
		t.RangeRetrieve,
		t.RangeDelete,
		t.RangeDispatch,

		// KV
		t.KVGet,
		t.KVSet,
		t.KVDelete,
		t.KVDispatch,

		// ALIAS
		t.AliasSet,
		t.AliasResolve,
		t.AliasRetrieve,
		t.AliasList,
		t.AliasDelete,
		t.AliasDispatch,

		// PROJECT
		t.ProjectDelete,
		t.ProjectCreate,
		t.ProjectRetrieve,
		t.ProjectDispatch,
		t.ProjectExport,
		t.ProjectImport,

		// SCHEMATIC
		t.SchematicCreate,
		t.SchematicRetrieve,
		t.SchematicDelete,
		t.SchematicDispatch,
		t.SchematicCopy,

		// SCHEMATIC SYMBOL
		t.SchematicSymbolCreate,
		t.SchematicSymbolRetrieve,
		t.SchematicSymbolDelete,
		t.SchematicSymbolDispatch,
		t.SchematicSymbolExportGroup,
		t.SchematicSymbolImportGroup,
		t.SchematicSymbolDeleteGroup,

		// LINE PLOT
		t.LinePlotCreate,
		t.LinePlotDispatch,
		t.LinePlotRetrieve,
		t.LinePlotDelete,

		// PANEL
		t.PanelCreate,
		t.PanelRetrieve,
		t.PanelDelete,
		t.PanelDispatch,

		// LOG
		t.LogCreate,
		t.LogRetrieve,
		t.LogDelete,
		t.LogDispatch,

		// TABLE
		t.TableCreate,
		t.TableRetrieve,
		t.TableDelete,
		t.TableDispatch,

		// LABEL
		t.LabelCreate,
		t.LabelRetrieve,
		t.LabelDelete,
		t.LabelAdd,
		t.LabelRemove,
		t.LabelDispatch,

		// RACK
		t.RackCreate,
		t.RackRetrieve,
		t.RackDelete,
		t.RackDispatch,

		// TASK
		t.TaskCreate,
		t.TaskRetrieve,
		t.TaskDelete,
		t.TaskCopy,
		t.TaskDispatch,

		// DEVICE
		t.DeviceCreate,
		t.DeviceRetrieve,
		t.DeviceDelete,
		t.DeviceDispatch,

		// ACCESS
		t.AccessCreatePolicy,
		t.AccessDeletePolicy,
		t.AccessRetrievePolicy,
		t.AccessCreateRole,
		t.AccessDeleteRole,
		t.AccessRetrieveRole,
		t.AccessDispatchPolicy,
		t.AccessDispatchRole,

		// STATUS
		t.StatusSet,
		t.StatusRetrieve,
		t.StatusDelete,
		t.StatusRename,
		t.StatusSetByKeyOrName,

		// VIEW
		t.ViewCreate,
		t.ViewRetrieve,
		t.ViewDelete,
		t.ViewDispatch,

		// SECRET
		t.SecretSet,
		t.SecretRetrieve,
		t.SecretDelete,
		t.SecretDispatch,
		t.SecretGrantCreate,
		t.SecretResolve,

		// ARC
		t.ArcCreate,
		t.ArcDelete,
		t.ArcRetrieve,
		t.ArcDispatch,
		t.ArcUpdateTask,
		t.ArcRetrieveDirectory,
		t.ArcCreateDirectory,
		t.ArcDispatchDirectory,
		t.ArcDeleteDirectory,
		t.ArcLSP,

		// IMPORT/EXPORT
		t.ImExImport,
		t.ImExExport,

		// BATCH
		t.Batch,
	)

	db := l.config.Distribution.DB

	// AUTH
	t.AuthLogin.BindHandler(l.Auth.Login)
	t.AuthChangePassword.BindHandler(bind.WritePrepared(
		l.binder,
		l.Auth.PrepareChangePassword,
		l.Auth.ChangePassword,
	))
	t.AuthSessionCreate.BindHandler(l.Auth.CreateSession)
	t.AuthSessionRenew.BindHandler(l.Auth.RenewSession)
	t.AuthSessionRetrieve.BindHandler(bind.Read(l.binder, func(
		ctx context.Context,
		snap bind.Snapshot,
		req auth.SessionRetrieveRequest,
	) (auth.SessionRetrieveResponse, error) {
		return l.Auth.RetrieveSessions(ctx, snap, req)
	}))
	t.AuthSessionDelete.BindHandler(bind.Write(l.binder, l.Auth.DeleteSessions))
	t.AuthKeyCreate.BindHandler(bind.Write(l.binder, l.Auth.CreateKey))
	t.AuthKeyRetrieve.BindHandler(bind.Read(l.binder, func(
		ctx context.Context,
		snap bind.Snapshot,
		req auth.KeyRetrieveRequest,
	) (auth.KeyRetrieveResponse, error) {
		return l.Auth.RetrieveKeys(ctx, snap, req)
	}))
	t.AuthKeyDelete.BindHandler(bind.Write(l.binder, l.Auth.DeleteKeys))
	t.AuthPasswordSet.BindHandler(bind.WritePrepared(
		l.binder,
		l.Auth.PrepareSetPassword,
		l.Auth.SetPassword,
	))

	// LICENSE
	t.LicenseRetrieve.BindHandler(bind.Read(l.binder, l.License.Retrieve))
	t.LicenseActivate.BindHandler(l.License.Activate)
	t.LicenseDeactivate.BindHandler(l.License.Deactivate)

	// USER
	t.UserCreate.BindHandler(
		bind.WritePrepared(l.binder, l.User.PrepareCreate, l.User.Create),
	)
	t.UserDelete.BindHandler(bind.Write(l.binder, l.User.Delete))
	t.UserRetrieve.BindHandler(bind.Read(l.binder, l.User.Retrieve))
	t.UserDispatch.BindHandler(bind.Write(l.binder, l.User.Dispatch))

	// CHANNEL
	t.ChannelCreate.BindHandler(bind.Write(l.binder, l.Channel.Create))
	t.ChannelRetrieve.BindHandler(bind.Read(l.binder, l.Channel.Retrieve))
	t.ConnectivityCheck.BindHandler(l.Connectivity.Check)
	t.ChannelDelete.BindHandler(bind.Write(l.binder, l.Channel.Delete))
	t.ChannelDispatch.BindHandler(bind.Write(l.binder, l.Channel.Dispatch))

	// FRAME
	t.FrameWriter.BindHandler(l.Framer.Write)
	t.FrameIterator.BindHandler(l.Framer.Iterate)
	t.FrameStreamer.BindHandler(l.Framer.Stream)
	// Frame delete changes Cesium data, not Gorp records, so it checks its own access.
	t.FrameDelete.BindHandler(fgorp.CreateWriteUnaryHandler(db, l.Framer.Delete))
	t.FrameRead.BindHandler(l.Framer.Read)

	// CONTROL
	t.ControlRetrieve.BindHandler(bind.Read(l.binder, l.Control.Retrieve))

	// RESOURCE
	t.ResourceSearch.BindHandler(bind.Read(l.binder, l.Resource.Search))
	t.ResourceRetrieve.BindHandler(bind.Read(l.binder, l.Resource.Retrieve))

	// GROUP
	t.GroupCreate.BindHandler(bind.Write(l.binder, l.Group.Create))
	t.GroupDelete.BindHandler(bind.Write(l.binder, l.Group.Delete))
	t.GroupDispatch.BindHandler(bind.Write(l.binder, l.Group.Dispatch))
	t.GroupRetrieve.BindHandler(bind.Read(l.binder, l.Group.Retrieve))
	t.GroupRetrieveAncestors.BindHandler(bind.Read(l.binder, l.Group.RetrieveAncestors))

	// BATCH
	t.Batch.BindHandler(bind.Write(l.binder, l.Batch.Exec))

	// RANGE
	t.RangeRetrieve.BindHandler(bind.Read(l.binder, l.Range.Retrieve))
	t.RangeCreate.BindHandler(bind.Write(l.binder, l.Range.Create))
	t.RangeDelete.BindHandler(bind.Write(l.binder, l.Range.Delete))
	t.RangeDispatch.BindHandler(bind.Write(l.binder, l.Range.Dispatch))

	// KV
	t.KVGet.BindHandler(bind.Read(l.binder, l.KV.Get))
	t.KVSet.BindHandler(bind.Write(l.binder, l.KV.Set))
	t.KVDelete.BindHandler(bind.Write(l.binder, l.KV.Delete))
	t.KVDispatch.BindHandler(bind.Write(l.binder, l.KV.Dispatch))

	// ALIAS
	t.AliasSet.BindHandler(bind.Write(l.binder, l.Alias.Set))
	t.AliasResolve.BindHandler(bind.Read(l.binder, l.Alias.Resolve))
	t.AliasRetrieve.BindHandler(bind.Read(l.binder, l.Alias.Retrieve))
	t.AliasList.BindHandler(bind.Read(l.binder, l.Alias.List))
	t.AliasDelete.BindHandler(bind.Write(l.binder, l.Alias.Delete))
	t.AliasDispatch.BindHandler(bind.Write(l.binder, l.Alias.Dispatch))

	// PROJECT
	t.ProjectCreate.BindHandler(bind.Write(l.binder, l.Project.Create))
	t.ProjectDelete.BindHandler(bind.Write(l.binder, l.Project.Delete))
	t.ProjectRetrieve.BindHandler(bind.Read(l.binder, l.Project.Retrieve))
	t.ProjectDispatch.BindHandler(bind.Write(l.binder, l.Project.Dispatch))
	t.ProjectExport.BindHandler(bind.Read(l.binder, l.Project.Export))
	t.ProjectImport.BindHandler(
		bind.WritePrepared(l.binder, l.Project.PrepareImport, l.Project.Import),
	)

	// SCHEMATIC
	t.SchematicCreate.BindHandler(bind.Write(l.binder, l.Schematic.Create))
	t.SchematicRetrieve.BindHandler(bind.Read(l.binder, l.Schematic.Retrieve))
	t.SchematicDelete.BindHandler(bind.Write(l.binder, l.Schematic.Delete))
	t.SchematicDispatch.BindHandler(bind.Write(l.binder, l.Schematic.Dispatch))
	t.SchematicCopy.BindHandler(bind.Write(l.binder, l.Schematic.Copy))

	// SCHEMATIC SYMBOL
	t.SchematicSymbolCreate.BindHandler(bind.Write(l.binder, l.Symbol.Create))
	t.SchematicSymbolRetrieve.BindHandler(bind.Read(l.binder, l.Symbol.Retrieve))
	t.SchematicSymbolDelete.BindHandler(bind.Write(l.binder, l.Symbol.Delete))
	t.SchematicSymbolDispatch.BindHandler(bind.Write(l.binder, l.Symbol.Dispatch))
	t.SchematicSymbolExportGroup.BindHandler(
		bind.Read(l.binder, l.Symbol.ExportGroup),
	)
	t.SchematicSymbolImportGroup.BindHandler(
		bind.Write(l.binder, l.Symbol.ImportGroup),
	)
	t.SchematicSymbolDeleteGroup.BindHandler(
		bind.Write(l.binder, l.Symbol.DeleteGroup),
	)

	// LINE PLOT
	t.LinePlotCreate.BindHandler(bind.Write(l.binder, l.LinePlot.Create))
	t.LinePlotDispatch.BindHandler(bind.Write(l.binder, l.LinePlot.Dispatch))
	t.LinePlotRetrieve.BindHandler(bind.Read(l.binder, l.LinePlot.Retrieve))
	t.LinePlotDelete.BindHandler(bind.Write(l.binder, l.LinePlot.Delete))

	// PANEL
	t.PanelCreate.BindHandler(bind.Write(l.binder, l.Panel.Create))
	t.PanelRetrieve.BindHandler(bind.Read(l.binder, l.Panel.Retrieve))
	t.PanelDelete.BindHandler(bind.Write(l.binder, l.Panel.Delete))
	t.PanelDispatch.BindHandler(bind.Write(l.binder, l.Panel.Dispatch))

	// LOG
	t.LogCreate.BindHandler(bind.Write(l.binder, l.Log.Create))
	t.LogRetrieve.BindHandler(bind.Read(l.binder, l.Log.Retrieve))
	t.LogDelete.BindHandler(bind.Write(l.binder, l.Log.Delete))
	t.LogDispatch.BindHandler(bind.Write(l.binder, l.Log.Dispatch))

	// TABLE
	t.TableCreate.BindHandler(bind.Write(l.binder, l.Table.Create))
	t.TableRetrieve.BindHandler(bind.Read(l.binder, l.Table.Retrieve))
	t.TableDelete.BindHandler(bind.Write(l.binder, l.Table.Delete))
	t.TableDispatch.BindHandler(bind.Write(l.binder, l.Table.Dispatch))

	// LABEL
	t.LabelCreate.BindHandler(bind.Write(l.binder, l.Label.Create))
	t.LabelRetrieve.BindHandler(bind.Read(l.binder, l.Label.Retrieve))
	t.LabelDelete.BindHandler(bind.Write(l.binder, l.Label.Delete))
	t.LabelAdd.BindHandler(bind.Write(l.binder, l.Label.Add))
	t.LabelRemove.BindHandler(bind.Write(l.binder, l.Label.Remove))
	t.LabelDispatch.BindHandler(bind.Write(l.binder, l.Label.Dispatch))

	// RACK
	t.RackCreate.BindHandler(bind.Write(l.binder, l.Rack.Create))
	t.RackRetrieve.BindHandler(bind.Read(l.binder, l.Rack.Retrieve))
	t.RackDelete.BindHandler(bind.Write(l.binder, l.Rack.Delete))
	t.RackDispatch.BindHandler(bind.Write(l.binder, l.Rack.Dispatch))

	// TASK
	t.TaskCreate.BindHandler(bind.Write(l.binder, l.Task.Create))
	t.TaskRetrieve.BindHandler(bind.Read(l.binder, l.Task.Retrieve))
	t.TaskDelete.BindHandler(
		bind.WritePrepared(l.binder, l.Task.PrepareDelete, l.Task.Delete),
	)
	t.TaskCopy.BindHandler(bind.Write(l.binder, l.Task.Copy))
	t.TaskDispatch.BindHandler(bind.Write(l.binder, l.Task.Dispatch))

	// DEVICE
	t.DeviceCreate.BindHandler(bind.Write(l.binder, l.Device.Create))
	t.DeviceRetrieve.BindHandler(bind.Read(l.binder, l.Device.Retrieve))
	t.DeviceDelete.BindHandler(bind.Write(l.binder, l.Device.Delete))
	t.DeviceDispatch.BindHandler(bind.Write(l.binder, l.Device.Dispatch))

	// ACCESS
	t.AccessCreatePolicy.BindHandler(bind.Write(l.binder, l.Policy.Create))
	t.AccessDeletePolicy.BindHandler(bind.Write(l.binder, l.Policy.Delete))
	t.AccessRetrievePolicy.BindHandler(bind.Read(l.binder, l.Policy.Retrieve))
	t.AccessCreateRole.BindHandler(bind.Write(l.binder, l.Role.Create))
	t.AccessDeleteRole.BindHandler(bind.Write(l.binder, l.Role.Delete))
	t.AccessRetrieveRole.BindHandler(bind.Read(l.binder, l.Role.Retrieve))
	t.AccessDispatchPolicy.BindHandler(bind.Write(l.binder, l.Policy.Dispatch))
	t.AccessDispatchRole.BindHandler(bind.Write(l.binder, l.Role.Dispatch))

	// STATUS
	t.StatusSet.BindHandler(bind.Write(l.binder, l.Status.Set))
	t.StatusRetrieve.BindHandler(bind.Read(l.binder, l.Status.Retrieve))
	t.StatusDelete.BindHandler(bind.Write(l.binder, l.Status.Delete))
	t.StatusRename.BindHandler(bind.Write(l.binder, l.Status.Rename))
	t.StatusSetByKeyOrName.BindHandler(bind.Write(l.binder, l.Status.SetByKeyOrName))

	// VIEW
	t.ViewCreate.BindHandler(bind.Write(l.binder, l.View.Create))
	t.ViewRetrieve.BindHandler(bind.Read(l.binder, l.View.Retrieve))
	t.ViewDelete.BindHandler(bind.Write(l.binder, l.View.Delete))
	t.ViewDispatch.BindHandler(bind.Write(l.binder, l.View.Dispatch))

	// SECRET
	t.SecretSet.BindHandler(bind.Write(l.binder, l.Secret.Set))
	t.SecretRetrieve.BindHandler(bind.Read(l.binder, l.Secret.Retrieve))
	t.SecretDelete.BindHandler(bind.Write(l.binder, l.Secret.Delete))
	t.SecretDispatch.BindHandler(bind.Write(l.binder, l.Secret.Dispatch))
	t.SecretGrantCreate.BindHandler(
		fgorp.CreateWriteUnaryHandler(db, l.Secret.CreateGrant),
	)
	t.SecretResolve.BindHandler(l.Secret.Resolve)

	// ARC
	t.ArcCreate.BindHandler(bind.Write(l.binder, l.Arc.Create))
	t.ArcDelete.BindHandler(
		bind.WritePrepared(l.binder, l.Arc.PrepareDelete, l.Arc.Delete),
	)
	t.ArcRetrieve.BindHandler(l.Arc.Compiled(bind.Read(l.binder, l.Arc.Retrieve)))
	t.ArcDispatch.BindHandler(bind.Write(l.binder, l.Arc.Dispatch))
	t.ArcUpdateTask.BindHandler(
		bind.WritePrepared(l.binder, l.Arc.PrepareUpdateTask, l.Arc.UpdateTask),
	)
	t.ArcRetrieveDirectory.BindHandler(bind.Read(l.binder, l.Arc.RetrieveDirectories))
	t.ArcCreateDirectory.BindHandler(bind.Write(l.binder, l.Arc.CreateDirectory))
	t.ArcDispatchDirectory.BindHandler(bind.Write(l.binder, l.Arc.DispatchDirectory))
	t.ArcDeleteDirectory.BindHandler(bind.Write(l.binder, l.Arc.DeleteDirectory))
	t.ArcLSP.BindHandler(l.Arc.LSP)

	// IMPORT/EXPORT
	t.ImExImport.BindHandler(bind.Write(l.binder, l.ImEx.Import))
	t.ImExExport.BindHandler(bind.Read(l.binder, l.ImEx.Export))
}

// NewLayer instantiates the server API layer using the provided Configs. This should
// only be called once.
func NewLayer(cfgs ...LayerConfig) (*Layer, error) {
	cfg, err := xconfig.New(config.DefaultLayerConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	l := &Layer{config: cfg}
	if l.binder, err = bind.NewService(bind.ServiceConfig{
		DB:   cfg.Distribution.DB,
		RBAC: cfg.Service.RBAC,
		Held: auth.HeldTables(),
	}); err != nil {
		return nil, err
	}
	if l.Auth, err = auth.NewService(auth.ServiceConfig{
		DB:       cfg.Distribution.DB,
		Auth:     cfg.Service.Auth,
		Password: cfg.Service.Password,
		APIKey:   cfg.Service.APIKey,
		User:     cfg.Service.User,
		RBAC:     cfg.Service.RBAC,
		Identity: cfg.Distribution.Identity,
	}); err != nil {
		return nil, err
	}
	if l.User, err = user.NewService(user.ServiceConfig{
		DB:       cfg.Distribution.DB,
		RBAC:     cfg.Service.RBAC,
		User:     cfg.Service.User,
		Password: cfg.Service.Password,
	}); err != nil {
		return nil, err
	}
	if l.Policy, err = policy.NewService(policy.ServiceConfig{
		RBAC: cfg.Service.RBAC,
	}); err != nil {
		return nil, err
	}
	if l.Role, err = role.NewService(role.ServiceConfig{
		RBAC: cfg.Service.RBAC,
	}); err != nil {
		return nil, err
	}
	if l.Framer, err = framer.NewService(framer.ServiceConfig{
		Instrumentation: cfg.Instrumentation,
		DB:              cfg.Distribution.DB,
		Framer:          cfg.Service.Framer,
		Channel:         cfg.Service.Channel,
		RBAC:            cfg.Service.RBAC,
		Records:         cfg.Service.Signals.Records(),
	}); err != nil {
		return nil, err
	}
	if l.Channel, err = channel.NewService(channel.ServiceConfig{
		DB:      cfg.Distribution.DB,
		RBAC:    cfg.Service.RBAC,
		Channel: cfg.Service.Channel,
		Status:  cfg.Service.Status,
		Alias:   cfg.Service.Alias,
	}); err != nil {
		return nil, err
	}
	if l.Control, err = control.NewService(control.ServiceConfig{
		Control: cfg.Service.Control,
	}); err != nil {
		return nil, err
	}
	if l.Connectivity, err = connectivity.NewService(connectivity.ServiceConfig{
		Identity: cfg.Distribution.Identity,
		License:  cfg.Service.License,
	}); err != nil {
		return nil, err
	}
	if l.License, err = license.NewService(license.ServiceConfig{
		DB:      cfg.Distribution.DB,
		RBAC:    cfg.Service.RBAC,
		License: cfg.Service.License,
	}); err != nil {
		return nil, err
	}
	if l.Resource, err = apiresource.NewService(apiresource.ServiceConfig{
		Resource: cfg.Service.Resource,
	}); err != nil {
		return nil, err
	}
	if l.Range, err = ranger.NewService(ranger.ServiceConfig{
		Ranger: cfg.Service.Ranger,
	}); err != nil {
		return nil, err
	}
	if l.KV, err = kv.NewService(kv.ServiceConfig{
		KV: cfg.Service.KV,
	}); err != nil {
		return nil, err
	}
	if l.Alias, err = alias.NewService(alias.ServiceConfig{
		Alias: cfg.Service.Alias,
	}); err != nil {
		return nil, err
	}
	if l.Group, err = group.NewService(group.ServiceConfig{
		Group: cfg.Service.Group,
	}); err != nil {
		return nil, err
	}
	if l.Project, err = project.NewService(project.ServiceConfig{
		Project: cfg.Service.Project,
		Bundle:  cfg.Service.Bundle,
	}); err != nil {
		return nil, err
	}
	if l.Schematic, err = schematic.NewService(schematic.ServiceConfig{
		Schematic: cfg.Service.Schematic,
	}); err != nil {
		return nil, err
	}
	if l.Symbol, err = symbol.NewService(symbol.ServiceConfig{
		Symbol: cfg.Service.Schematic.Symbol,
	}); err != nil {
		return nil, err
	}
	if l.LinePlot, err = lineplot.NewService(lineplot.ServiceConfig{
		LinePlot: cfg.Service.LinePlot,
	}); err != nil {
		return nil, err
	}
	if l.Panel, err = panel.NewService(panel.ServiceConfig{
		Panel: cfg.Service.Panel,
	}); err != nil {
		return nil, err
	}
	if l.Label, err = label.NewService(label.ServiceConfig{
		Label:  cfg.Service.Label,
		Ranger: cfg.Service.Ranger,
		Status: cfg.Service.Status,
	}); err != nil {
		return nil, err
	}
	if l.Rack, err = rack.NewService(rack.ServiceConfig{
		Rack:   cfg.Service.Rack,
		Task:   cfg.Service.Task,
		Status: cfg.Service.Status,
		RBAC:   cfg.Service.RBAC,
	}); err != nil {
		return nil, err
	}
	if l.Task, err = task.NewService(task.ServiceConfig{
		DB:      cfg.Distribution.DB,
		Task:    cfg.Service.Task,
		Status:  cfg.Service.Status,
		RBAC:    cfg.Service.RBAC,
		Stopper: cfg.Service.TaskStopper,
	}); err != nil {
		return nil, err
	}
	if l.Device, err = device.NewService(device.ServiceConfig{
		Device: cfg.Service.Device,
		Status: cfg.Service.Status,
		RBAC:   cfg.Service.RBAC,
	}); err != nil {
		return nil, err
	}
	if l.Log, err = log.NewService(log.ServiceConfig{
		Log: cfg.Service.Log,
	}); err != nil {
		return nil, err
	}
	if l.Table, err = table.NewService(table.ServiceConfig{
		Table: cfg.Service.Table,
	}); err != nil {
		return nil, err
	}
	if l.Status, err = status.NewService(status.ServiceConfig{
		Status: cfg.Service.Status,
	}); err != nil {
		return nil, err
	}
	if l.Arc, err = arc.NewService(arc.ServiceConfig{
		DB:              cfg.Distribution.DB,
		Instrumentation: cfg.Instrumentation,
		Arc:             cfg.Service.Arc,
		Status:          cfg.Service.Status,
		RBAC:            cfg.Service.RBAC,
		DispatchChannel: bind.Write(l.binder, l.Channel.Dispatch),
		Stopper:         cfg.Service.TaskStopper,
	}); err != nil {
		return nil, err
	}
	if l.View, err = view.NewService(view.ServiceConfig{
		View: cfg.Service.View,
	}); err != nil {
		return nil, err
	}
	if l.Secret, err = secret.NewService(secret.ServiceConfig{
		Secret: cfg.Service.Secret,
		RBAC:   cfg.Service.RBAC,
	}); err != nil {
		return nil, err
	}
	if l.ImEx, err = imex.NewService(imex.ServiceConfig{
		ImEx: cfg.Service.ImEx,
	}); err != nil {
		return nil, err
	}
	l.Batch = batch.NewService(l.batchRoutes())
	return l, nil
}
