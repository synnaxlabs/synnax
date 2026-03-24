// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package backup

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/google/uuid"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/ranger"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/synnax/pkg/service/user"
	"github.com/synnaxlabs/synnax/pkg/service/workspace"
	"github.com/synnaxlabs/x/telem"
)

const Version = "1.1.0"

// Manifest describes the contents and metadata of a .sy backup archive.
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

// DataVisualization is the backup representation of a visualization that stores
// its configuration as a JSON string (line plots, tables, logs).
type DataVisualization struct {
	Name string          `json:"name"`
	Key  uuid.UUID       `json:"key"`
	Data json.RawMessage `json:"data"`
}

func newDataVisualization(name string, key uuid.UUID, data string) DataVisualization {
	return DataVisualization{Name: name, Key: key, Data: rawJSON(data)}
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
	// Group is the name of the ontology group this channel belongs to.
	// Empty if the channel is in the default "Channels" group.
	Group string `json:"group,omitempty"`
}

func newChannel(c distchannel.Channel) Channel {
	return Channel{
		Key: c.Key(), Name: c.Name, DataType: c.DataType, IsIndex: c.IsIndex,
		Index: c.Index(), Leaseholder: c.Leaseholder, Virtual: c.Virtual,
		Internal: c.Internal, Expression: c.Expression,
	}
}

// TelemetryManifest describes the telemetry data stored in the archive.
type TelemetryManifest struct {
	TimeRange telem.TimeRange            `json:"time_range"`
	Channels  []TelemetryChannelManifest `json:"channels"`
}

// TelemetryChannelManifest describes a single channel's telemetry data in the archive.
type TelemetryChannelManifest struct {
	Key        distchannel.Key `json:"key"`
	DataType   telem.DataType  `json:"data_type"`
	IsIndex    bool            `json:"is_index"`
	ChunkCount int             `json:"chunk_count"`
}

// TelemetryChunkMeta describes a single chunk of telemetry data for a channel.
type TelemetryChunkMeta struct {
	TimeRange   telem.TimeRange `json:"time_range"`
	Size        int64           `json:"size"`
	SampleCount int64           `json:"sample_count"`
}

// DataConflictPolicy determines what to do when imported telemetry data
// overlaps with existing data in the target time range.
type DataConflictPolicy string

const (
	DataPolicySkip      DataConflictPolicy = "skip"
	DataPolicyOverwrite DataConflictPolicy = "overwrite"
)

// ConflictStatus describes how an archive item relates to existing data.
type ConflictStatus string

const (
	StatusNew       ConflictStatus = "new"
	StatusConflict  ConflictStatus = "conflict"
	StatusIdentical ConflictStatus = "identical"
)

// ConflictPolicy determines what to do when an imported item conflicts with
// existing data.
type ConflictPolicy string

const (
	PolicySkip    ConflictPolicy = "skip"
	PolicyReplace ConflictPolicy = "replace"
)

// AnalysisItem describes a single entity found in an archive and its
// relationship to existing data.
type AnalysisItem struct {
	Type        string         `json:"type"`
	Name        string         `json:"name"`
	ArchiveKey  string         `json:"archive_key"`
	Status      ConflictStatus `json:"status"`
	ExistingKey string         `json:"existing_key,omitempty"`
	Details     string         `json:"details,omitempty"`
	ParentName  string         `json:"parent_name,omitempty"`
	// DataType is the channel data type (only populated for channel items).
	DataType telem.DataType `json:"data_type,omitempty"`
	// Disabled is true when the item cannot be imported (e.g., a non-virtual
	// data channel whose index channel is not in the archive). The UI should
	// grey out and prevent selection of disabled items.
	Disabled bool `json:"disabled,omitempty"`
}

// AnalyzeResponse is returned by the analyze endpoint with a session ID
// and the list of items found in the archive.
type AnalyzeResponse struct {
	SessionID string         `json:"session_id"`
	Items     []AnalysisItem `json:"items"`
}

// ImportRequest specifies how to handle conflicts during import.
type ImportRequest struct {
	SessionID          string                    `json:"session_id"`
	DefaultPolicy      ConflictPolicy            `json:"default_policy"`
	Overrides          map[string]ConflictPolicy `json:"overrides"`
	DataConflictPolicy DataConflictPolicy        `json:"data_conflict_policy"`
}

// PolicyFor returns the conflict policy for the given archive key, falling
// back to the default policy if no override is set.
func (r ImportRequest) PolicyFor(archiveKey string) ConflictPolicy {
	if p, ok := r.Overrides[archiveKey]; ok {
		return p
	}
	return r.DefaultPolicy
}

// ImportResponse summarizes the results of an import operation.
type ImportResponse struct {
	Imported  int      `json:"imported"`
	Skipped   int      `json:"skipped"`
	Replaced  int      `json:"replaced"`
	Identical int      `json:"identical"`
	Errors    []string `json:"errors"`
}

// OntologyIDsFromAnalysis builds ontology IDs from an analyze response so the
// API layer can enforce RBAC before executing an import.
func OntologyIDsFromAnalysis(resp AnalyzeResponse) []ontology.ID {
	ids := make([]ontology.ID, 0, len(resp.Items))
	for _, item := range resp.Items {
		switch item.Type {
		case "workspace":
			if k, err := uuid.Parse(item.ArchiveKey); err == nil {
				ids = append(ids, workspace.OntologyID(k))
			}
		case "user":
			if k, err := uuid.Parse(item.ArchiveKey); err == nil {
				ids = append(ids, user.OntologyID(k))
			}
		case "device":
			ids = append(ids, device.OntologyID(item.ArchiveKey))
		case "channel":
			if k, err := strconv.ParseUint(item.ArchiveKey, 10, 32); err == nil {
				ids = append(ids, distchannel.OntologyID(distchannel.Key(k)))
			}
		case "task":
			if k, err := strconv.ParseUint(item.ArchiveKey, 10, 64); err == nil {
				ids = append(ids, task.OntologyID(task.Key(k)))
			}
		case "range":
			if k, err := uuid.Parse(item.ArchiveKey); err == nil {
				ids = append(ids, ranger.OntologyID(k))
			}
		}
	}
	return ids
}

// rawJSON converts a string to json.RawMessage, defaulting to "{}" if empty.
func rawJSON(s string) json.RawMessage {
	if len(s) == 0 {
		return json.RawMessage("{}")
	}
	return json.RawMessage(s)
}
