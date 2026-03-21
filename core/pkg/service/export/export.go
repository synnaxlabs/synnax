// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package export

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	"github.com/synnaxlabs/synnax/pkg/service/workspace"
	"github.com/synnaxlabs/x/telem"
)

const Version = "1.0.0"

// Manifest describes the contents and metadata of a .syc export archive.
type Manifest struct {
	Version   string    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	Sections  []string  `json:"sections"`
}

// Workspace wraps workspace.Workspace with json.RawMessage for Layout
// to avoid double-encoding the JSON string stored in the DB.
type Workspace struct {
	Name   string          `json:"name"`
	Key    uuid.UUID       `json:"key"`
	Author uuid.UUID       `json:"author"`
	Layout json.RawMessage `json:"layout"`
}

func newWorkspace(ws workspace.Workspace) Workspace {
	return Workspace{
		Name:   ws.Name,
		Key:    ws.Key,
		Author: ws.Author,
		Layout: rawJSON(ws.Layout),
	}
}

// DataVisualization is the exported representation of a visualization that stores
// its configuration as a JSON string (line plots, tables, logs).
type DataVisualization struct {
	Name string          `json:"name"`
	Key  uuid.UUID       `json:"key"`
	Data json.RawMessage `json:"data"`
}

func newDataVisualizationFromLinePlot(lp lineplot.LinePlot) DataVisualization {
	return DataVisualization{Name: lp.Name, Key: lp.Key, Data: rawJSON(lp.Data)}
}

func newDataVisualizationFromTable(t table.Table) DataVisualization {
	return DataVisualization{Name: t.Name, Key: t.Key, Data: rawJSON(t.Data)}
}

func newDataVisualizationFromLog(l log.Log) DataVisualization {
	return DataVisualization{Name: l.Name, Key: l.Key, Data: rawJSON(l.Data)}
}

// Schematic wraps schematic.Schematic with json.RawMessage for Data.
// Separate from DataVisualization because it has the Snapshot field.
type Schematic struct {
	Name     string          `json:"name"`
	Key      uuid.UUID       `json:"key"`
	Data     json.RawMessage `json:"data"`
	Snapshot bool            `json:"snapshot"`
}

func newSchematic(s schematic.Schematic) Schematic {
	return Schematic{Name: s.Name, Key: s.Key, Data: rawJSON(s.Data), Snapshot: s.Snapshot}
}

// Channel projects distchannel.Channel's computed fields into a flat struct.
type Channel struct {
	Key         distchannel.Key `json:"key"`
	Name        string          `json:"name"`
	DataType    telem.DataType  `json:"data_type"`
	IsIndex     bool            `json:"is_index"`
	Index       distchannel.Key `json:"index"`
	Leaseholder cluster.NodeKey `json:"leaseholder"`
	Virtual     bool            `json:"virtual"`
	Internal    bool            `json:"internal"`
	Expression  string          `json:"expression"`
}

func newChannel(c distchannel.Channel) Channel {
	return Channel{
		Key: c.Key(), Name: c.Name, DataType: c.DataType, IsIndex: c.IsIndex,
		Index: c.Index(), Leaseholder: c.Leaseholder, Virtual: c.Virtual,
		Internal: c.Internal, Expression: c.Expression,
	}
}

// rawJSON converts a string to json.RawMessage, defaulting to "{}" if empty.
func rawJSON(s string) json.RawMessage {
	if len(s) == 0 {
		return json.RawMessage("{}")
	}
	return json.RawMessage(s)
}
