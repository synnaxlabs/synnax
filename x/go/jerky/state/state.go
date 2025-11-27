// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package state provides version tracking and state management for jerky.
package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const (
	// SchemaVersion is the current state file schema version.
	SchemaVersion = "jerky-state-v1"
	// StateFileName is the name of the state file.
	StateFileName = "jerky.state.json"
)

// FieldInfo represents information about a struct field at a specific version.
type FieldInfo struct {
	Type string            `json:"type"`
	Tags map[string]string `json:"tags,omitempty"`
}

// VersionDiff represents changes between two versions.
type VersionDiff struct {
	Added   []string `json:"added,omitempty"`
	Removed []string `json:"removed,omitempty"`
	Changed []string `json:"changed,omitempty"`
}

// VersionHistory represents the history of a single version.
type VersionHistory struct {
	Version           int                 `json:"version"`
	CreatedAt         time.Time           `json:"created_at"`
	StructHash        string              `json:"struct_hash"`
	DependencyHashes  map[string]string   `json:"dependency_hashes,omitempty"`
	CompositeHash     string              `json:"composite_hash"`
	MigrationType     string              `json:"migration_type"`
	Fields            map[string]FieldInfo `json:"fields"`
	Diff              *VersionDiff        `json:"diff,omitempty"`
}

// TypeState represents the state of a single jerky-managed type.
type TypeState struct {
	Package        string           `json:"package"`
	CurrentVersion int              `json:"current_version"`
	FieldOrder     []string         `json:"field_order"`
	History        []VersionHistory `json:"history"`
}

// StateMetadata contains metadata about the state file.
type StateMetadata struct {
	JerkyVersion  string    `json:"jerky_version"`
	LastGenerated time.Time `json:"last_generated"`
}

// File represents the complete state file structure.
type File struct {
	Schema   string               `json:"$schema"`
	Types    map[string]TypeState `json:"types"`
	Metadata StateMetadata        `json:"metadata"`
}

// NewFile creates a new empty state file.
func NewFile() *File {
	return &File{
		Schema: SchemaVersion,
		Types:  make(map[string]TypeState),
		Metadata: StateMetadata{
			JerkyVersion:  "1.0.0",
			LastGenerated: time.Now(),
		},
	}
}

// GetTypeState returns the state for a given type name.
func (f *File) GetTypeState(typeName string) (TypeState, bool) {
	ts, ok := f.Types[typeName]
	return ts, ok
}

// SetTypeState sets the state for a given type name.
func (f *File) SetTypeState(typeName string, ts TypeState) {
	f.Types[typeName] = ts
	f.Metadata.LastGenerated = time.Now()
}

// Load reads a state file from disk.
func Load(dir string) (*File, error) {
	path := filepath.Join(dir, StateFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewFile(), nil
		}
		return nil, err
	}

	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, err
	}

	return &f, nil
}

// Save writes the state file to disk.
func (f *File) Save(dir string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	path := filepath.Join(dir, StateFileName)
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// LatestVersion returns the latest version history entry for a type.
func (ts *TypeState) LatestVersion() *VersionHistory {
	if len(ts.History) == 0 {
		return nil
	}
	return &ts.History[len(ts.History)-1]
}

// AddVersion adds a new version to the type's history.
func (ts *TypeState) AddVersion(vh VersionHistory) {
	ts.History = append(ts.History, vh)
	ts.CurrentVersion = vh.Version
}
