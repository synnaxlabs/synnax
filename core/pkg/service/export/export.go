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
	"github.com/synnaxlabs/arc/graph"
	"github.com/synnaxlabs/arc/program"
	"github.com/synnaxlabs/arc/text"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	"github.com/synnaxlabs/synnax/pkg/service/workspace"
)

const Version = "1.0.0"

// Manifest describes the contents and metadata of a .syc export archive.
type Manifest struct {
	Version   string    `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	Sections  []string  `json:"sections"`
}

// Workspace is the exported representation of a workspace, using json.RawMessage
// for the Layout field to avoid double-encoding the JSON string stored in the DB.
type Workspace struct {
	Name   string          `json:"name"`
	Key    uuid.UUID       `json:"key"`
	Author uuid.UUID       `json:"author"`
	Layout json.RawMessage `json:"layout"`
}

func newWorkspace(ws workspace.Workspace) Workspace {
	layout := json.RawMessage(ws.Layout)
	if len(ws.Layout) == 0 {
		layout = json.RawMessage("{}")
	}
	return Workspace{
		Name:   ws.Name,
		Key:    ws.Key,
		Author: ws.Author,
		Layout: layout,
	}
}

// LinePlot is the exported representation of a line plot visualization.
type LinePlot struct {
	Name string          `json:"name"`
	Key  uuid.UUID       `json:"key"`
	Data json.RawMessage `json:"data"`
}

func newLinePlot(lp lineplot.LinePlot) LinePlot {
	data := json.RawMessage(lp.Data)
	if len(lp.Data) == 0 {
		data = json.RawMessage("{}")
	}
	return LinePlot{Name: lp.Name, Key: lp.Key, Data: data}
}

// Schematic is the exported representation of a schematic visualization.
type Schematic struct {
	Name     string          `json:"name"`
	Key      uuid.UUID       `json:"key"`
	Data     json.RawMessage `json:"data"`
	Snapshot bool            `json:"snapshot"`
}

func newSchematic(s schematic.Schematic) Schematic {
	data := json.RawMessage(s.Data)
	if len(s.Data) == 0 {
		data = json.RawMessage("{}")
	}
	return Schematic{Name: s.Name, Key: s.Key, Data: data, Snapshot: s.Snapshot}
}

// Table is the exported representation of a table visualization.
type Table struct {
	Name string          `json:"name"`
	Key  uuid.UUID       `json:"key"`
	Data json.RawMessage `json:"data"`
}

func newTable(t table.Table) Table {
	data := json.RawMessage(t.Data)
	if len(t.Data) == 0 {
		data = json.RawMessage("{}")
	}
	return Table{Name: t.Name, Key: t.Key, Data: data}
}

// Arc is the exported representation of an arc automation.
type Arc struct {
	Name    string          `json:"name"`
	Key     uuid.UUID       `json:"key"`
	Text    text.Text       `json:"text"`
	Version string          `json:"version"`
	Mode    arc.Mode        `json:"mode"`
	Program program.Program `json:"program"`
	Graph   graph.Graph     `json:"graph"`
}

func newArc(a arc.Arc) Arc {
	return Arc{
		Name:    a.Name,
		Key:     a.Key,
		Text:    a.Text,
		Version: a.Version,
		Mode:    a.Mode,
		Program: a.Program,
		Graph:   a.Graph,
	}
}

// Log is the exported representation of a log visualization.
type Log struct {
	Name string          `json:"name"`
	Key  uuid.UUID       `json:"key"`
	Data json.RawMessage `json:"data"`
}

func newLog(l log.Log) Log {
	data := json.RawMessage(l.Data)
	if len(l.Data) == 0 {
		data = json.RawMessage("{}")
	}
	return Log{Name: l.Name, Key: l.Key, Data: data}
}
