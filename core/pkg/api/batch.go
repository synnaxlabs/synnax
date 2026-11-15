// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api

import "github.com/synnaxlabs/synnax/pkg/api/batch"

// Paths of the endpoints that a batch accepts. A transport serves each one under its
// API root, and a batch request names one as its endpoint.
const (
	PathGroupCreate             = "/group/create"
	PathGroupDelete             = "/group/delete"
	PathStatusSet               = "/status/set"
	PathLabelSet                = "/label/set"
	PathLabelRemove             = "/label/remove"
	PathArcDirectoryCreate      = "/arc/directory/create"
	PathUserDispatch            = "/user/dispatch"
	PathChannelDispatch         = "/channel/dispatch"
	PathGroupDispatch           = "/group/dispatch"
	PathRangeDispatch           = "/range/dispatch"
	PathRangeKVDispatch         = "/range/kv/dispatch"
	PathProjectDispatch         = "/project/dispatch"
	PathSchematicDispatch       = "/schematic/dispatch"
	PathSchematicSymbolDispatch = "/schematic/symbol/dispatch"
	PathLinePlotDispatch        = "/lineplot/dispatch"
	PathPanelDispatch           = "/panel/dispatch"
	PathLogDispatch             = "/log/dispatch"
	PathTableDispatch           = "/table/dispatch"
	PathLabelDispatch           = "/label/dispatch"
	PathRackDispatch            = "/rack/dispatch"
	PathTaskDispatch            = "/task/dispatch"
	PathDeviceDispatch          = "/device/dispatch"
	PathPolicyDispatch          = "/access/policy/dispatch"
	PathRoleDispatch            = "/access/role/dispatch"
	PathArcDispatch             = "/arc/dispatch"
	PathArcDirectoryDispatch    = "/arc/directory/dispatch"
	PathViewDispatch            = "/view/dispatch"
	PathSecretDispatch          = "/secret/dispatch"
	PathCertificateDispatch     = "/certificate/dispatch"
)

// batchRoutes returns the handler of each endpoint a batch accepts. Every handler does
// all of its work in the transaction it receives, and runs the same checks it runs on
// its own route. Channel create and delete stay out: they write Cesium, outside the
// transaction.
func (l *Layer) batchRoutes() batch.Routes {
	return batch.Routes{
		PathGroupCreate:             batch.Route(l.Group.Create),
		PathGroupDelete:             batch.Route(l.Group.Delete),
		PathStatusSet:               batch.Route(l.Status.Set),
		PathLabelSet:                batch.Route(l.Label.Add),
		PathLabelRemove:             batch.Route(l.Label.Remove),
		PathArcDirectoryCreate:      batch.Route(l.Arc.CreateDirectory),
		PathUserDispatch:            batch.Route(l.User.Dispatch),
		PathChannelDispatch:         batch.Route(l.Channel.Dispatch),
		PathGroupDispatch:           batch.Route(l.Group.Dispatch),
		PathRangeDispatch:           batch.Route(l.Range.Dispatch),
		PathRangeKVDispatch:         batch.Route(l.KV.Dispatch),
		PathProjectDispatch:         batch.Route(l.Project.Dispatch),
		PathSchematicDispatch:       batch.Route(l.Schematic.Dispatch),
		PathSchematicSymbolDispatch: batch.Route(l.Symbol.Dispatch),
		PathLinePlotDispatch:        batch.Route(l.LinePlot.Dispatch),
		PathPanelDispatch:           batch.Route(l.Panel.Dispatch),
		PathLogDispatch:             batch.Route(l.Log.Dispatch),
		PathTableDispatch:           batch.Route(l.Table.Dispatch),
		PathLabelDispatch:           batch.Route(l.Label.Dispatch),
		PathRackDispatch:            batch.Route(l.Rack.Dispatch),
		PathTaskDispatch:            batch.Route(l.Task.Dispatch),
		PathDeviceDispatch:          batch.Route(l.Device.Dispatch),
		PathPolicyDispatch:          batch.Route(l.Policy.Dispatch),
		PathRoleDispatch:            batch.Route(l.Role.Dispatch),
		PathArcDispatch:             batch.Route(l.Arc.Dispatch),
		PathArcDirectoryDispatch:    batch.Route(l.Arc.DispatchDirectory),
		PathViewDispatch:            batch.Route(l.View.Dispatch),
		PathSecretDispatch:          batch.Route(l.Secret.Dispatch),
		PathCertificateDispatch:     batch.Route(l.Certificate.Dispatch),
	}
}
