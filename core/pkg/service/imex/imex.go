// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package imex

import (
	"context"
	"encoding/json"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
)

// Envelope is the standard format for importing and exporting a single resource.
// Type is the most specific type identifier (e.g., "log", "modbus_read"). Version
// is the data schema version (may be empty for old exports). Data contains the full
// serialized resource including key, name, and all fields.
type Envelope struct {
	Version string         `json:"version" msgpack:"version"`
	Type    string         `json:"type" msgpack:"type"`
	Data    map[string]any `json:"data" msgpack:"data"`
}

// ParseEnvelope converts raw JSON bytes into an Envelope. It detects the format
// automatically: if the JSON contains a nested "data" object, it's the new envelope
// format. Otherwise it's the old console flat format, and the entire JSON becomes
// the Data field.
func ParseEnvelope(raw []byte) (Envelope, error) {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return Envelope{}, err
	}
	return ParseEnvelopeFromMap(m), nil
}

// ParseEnvelopeFromMap converts a raw map into an Envelope, detecting the format.
func ParseEnvelopeFromMap(m map[string]any) Envelope {
	if data, ok := m["data"].(map[string]any); ok {
		env := Envelope{Data: data}
		if v, ok := m["version"].(string); ok {
			env.Version = v
		}
		if t, ok := m["type"].(string); ok {
			env.Type = t
		}
		return env
	}
	env := Envelope{Data: m}
	if v, ok := m["version"].(string); ok {
		env.Version = v
	}
	if t, ok := m["type"].(string); ok {
		env.Type = t
	}
	return env
}

// Importer can import a resource from an Envelope.
type Importer interface {
	Import(ctx context.Context, tx gorp.Tx, parent ontology.ID, env Envelope) error
}

// Exporter can export a resource to an Envelope.
type Exporter interface {
	Export(ctx context.Context, tx gorp.Tx, key string) (Envelope, error)
}

// ImporterExporter combines both interfaces.
type ImporterExporter interface {
	Importer
	Exporter
}
