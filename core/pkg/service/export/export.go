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
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/telem"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot"
	"github.com/synnaxlabs/synnax/pkg/service/log"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/table"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/user"
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

// User is the exported representation of a user (no credentials).
type User struct {
	Key       uuid.UUID `json:"key"`
	Username  string    `json:"username"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
}

func newUser(u user.User) User {
	return User{
		Key:       u.Key,
		Username:  u.Username,
		FirstName: u.FirstName,
		LastName:  u.LastName,
	}
}

// Device is the exported representation of a hardware device.
type Device struct {
	Key        string                `json:"key"`
	Name       string                `json:"name"`
	Make       string                `json:"make"`
	Model      string                `json:"model"`
	Location   string                `json:"location"`
	Properties binary.MsgpackEncodedJSON `json:"properties"`
	Rack       rack.Key              `json:"rack"`
	Configured bool                  `json:"configured"`
}

func newDevice(d device.Device) Device {
	return Device{
		Key:        d.Key,
		Name:       d.Name,
		Make:       d.Make,
		Model:      d.Model,
		Location:   d.Location,
		Properties: d.Properties,
		Rack:       d.Rack,
		Configured: d.Configured,
	}
}

// Task is the exported representation of a task (driver or arc).
type Task struct {
	Key      task.Key                  `json:"key"`
	Name     string                    `json:"name"`
	Type     string                    `json:"type"`
	Config   binary.MsgpackEncodedJSON `json:"config"`
	Internal bool                      `json:"internal"`
}

func newTask(t task.Task) Task {
	return Task{
		Key:      t.Key,
		Name:     t.Name,
		Type:     t.Type,
		Config:   t.Config,
		Internal: t.Internal,
	}
}

// Range is the exported representation of a time range.
type Range struct {
	Key       uuid.UUID       `json:"key"`
	Name      string          `json:"name"`
	Color     color.Color     `json:"color"`
	TimeRange telem.TimeRange `json:"time_range"`
}

func newRange(r ranger.Range) Range {
	return Range{
		Key:       r.Key,
		Name:      r.Name,
		Color:     r.Color,
		TimeRange: r.TimeRange,
	}
}

// Channel is the exported representation of a channel (metadata only, no data).
type Channel struct {
	Key         distchannel.Key    `json:"key"`
	Name        string             `json:"name"`
	DataType    telem.DataType     `json:"data_type"`
	IsIndex     bool               `json:"is_index"`
	Index       distchannel.Key    `json:"index"`
	Leaseholder cluster.NodeKey    `json:"leaseholder"`
	Virtual     bool               `json:"virtual"`
	Internal    bool               `json:"internal"`
	Expression  string             `json:"expression"`
}

func newChannel(c distchannel.Channel) Channel {
	return Channel{
		Key:         c.Key(),
		Name:        c.Name,
		DataType:    c.DataType,
		IsIndex:     c.IsIndex,
		Index:       c.Index(),
		Leaseholder: c.Leaseholder,
		Virtual:     c.Virtual,
		Internal:    c.Internal,
		Expression:  c.Expression,
	}
}
