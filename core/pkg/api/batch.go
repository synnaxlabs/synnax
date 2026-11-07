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
	PathGroupRename             = "/group/rename"
	PathGroupDelete             = "/group/delete"
	PathGroupSetParent          = "/group/set-parent"
	PathChannelSetGroup         = "/channel/set-group"
	PathDeviceSetGroup          = "/device/set-group"
	PathRackSetGroup            = "/rack/set-group"
	PathProjectSetGroup         = "/project/set-group"
	PathSchematicSetGroup       = "/schematic/set-group"
	PathSchematicSymbolSetGroup = "/schematic/symbol/set-group"
	PathLinePlotSetGroup        = "/lineplot/set-group"
	PathLogSetGroup             = "/log/set-group"
	PathTableSetGroup           = "/table/set-group"
	PathPanelSetGroup           = "/panel/set-group"
	PathStatusSet               = "/status/set"
	PathLabelSet                = "/label/set"
	PathLabelRemove             = "/label/remove"
)

// batchRoutes returns the handler of each endpoint a batch accepts. Every handler does
// all of its work in the transaction it receives. Channel create and delete stay out:
// they write Cesium, outside the transaction.
func (l *Layer) batchRoutes() batch.Routes {
	return batch.Routes{
		PathGroupCreate:             batch.Route(l.Group.Create),
		PathGroupRename:             batch.Route(l.Group.Rename),
		PathGroupDelete:             batch.Route(l.Group.Delete),
		PathGroupSetParent:          batch.Route(l.Group.SetParent),
		PathChannelSetGroup:         batch.Route(l.Channel.SetGroup),
		PathDeviceSetGroup:          batch.Route(l.Device.SetGroup),
		PathRackSetGroup:            batch.Route(l.Rack.SetGroup),
		PathProjectSetGroup:         batch.Route(l.Project.SetGroup),
		PathSchematicSetGroup:       batch.Route(l.Schematic.SetGroup),
		PathSchematicSymbolSetGroup: batch.Route(l.Symbol.SetGroup),
		PathLinePlotSetGroup:        batch.Route(l.LinePlot.SetGroup),
		PathLogSetGroup:             batch.Route(l.Log.SetGroup),
		PathTableSetGroup:           batch.Route(l.Table.SetGroup),
		PathPanelSetGroup:           batch.Route(l.Panel.SetGroup),
		PathStatusSet:               batch.Route(l.Status.Set),
		PathLabelSet:                batch.Route(l.Label.Add),
		PathLabelRemove:             batch.Route(l.Label.Remove),
	}
}
