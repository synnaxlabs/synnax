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
	"fmt"
	"maps"
	"strconv"
	"strings"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
)

// Envelope is the portable format for a single importable/exportable resource.
// All fields are flat at the JSON level. The wire format looks like:
//
//	{"version":54,"type":"log","key":"...","name":"...","channels":[...]}
//
// Version, Type, Key, and Name are promoted to typed fields for convenient
// access (routing, identity, file naming). Data is the complete flat map
// including key and name. The promoted fields are copies, not extractions.
// Handlers receive Data with all fields intact for zyn schema parsing.
type Envelope struct {
	Version int
	Type    string
	Key     string
	Name    string
	Data    map[string]any
}

// MarshalJSON flattens Data into the top-level JSON object and overwrites
// version and type with the promoted field values (since the export version
// is stamped by the service, not carried in Data).
func (e Envelope) MarshalJSON() ([]byte, error) {
	m := make(map[string]any, len(e.Data)+2)
	maps.Copy(m, e.Data)
	m["version"] = e.Version
	if e.Type != "" {
		m["type"] = e.Type
	}
	return json.Marshal(m)
}

// envelopeMeta holds the promoted fields for standard json unmarshaling.
// Version is raw because it can be either a number or a semver string.
type envelopeMeta struct {
	Version json.RawMessage `json:"version"`
	Type    string          `json:"type"`
	Key     string          `json:"key"`
	Name    string          `json:"name"`
}

// UnmarshalJSON reads a flat JSON object. Promoted fields are extracted via
// standard json struct tags. Data receives the complete map with all fields
// intact. The version field accepts both numeric values (new format) and
// semver strings (old Console format), converting the latter via
// legacyToNumeric.
func (e *Envelope) UnmarshalJSON(b []byte) error {
	var meta envelopeMeta
	if err := json.Unmarshal(b, &meta); err != nil {
		return err
	}
	e.Type = meta.Type
	e.Key = meta.Key
	e.Name = meta.Name
	if len(meta.Version) > 0 {
		v, err := parseVersionRaw(meta.Version)
		if err != nil {
			return err
		}
		e.Version = v
	}
	return json.Unmarshal(b, &e.Data)
}

// parseVersionRaw parses a JSON-encoded version value that can be either a
// number or a semver string.
func parseVersionRaw(raw json.RawMessage) (int, error) {
	var n float64
	if err := json.Unmarshal(raw, &n); err == nil {
		return int(n), nil
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return legacyToNumeric(s)
	}
	return 0, fmt.Errorf("version must be a number or semver string, got %s", string(raw))
}

// legacyToNumeric converts a semver string like "1.0.0" to a numeric version
// using the formula major*5 + minor*2 + patch.
func legacyToNumeric(s string) (int, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid semver %q: expected major.minor.patch", s)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, errors.Wrapf(err, "invalid semver major %q", parts[0])
	}
	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, errors.Wrapf(err, "invalid semver minor %q", parts[1])
	}
	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return 0, errors.Wrapf(err, "invalid semver patch %q", parts[2])
	}
	return major*5 + minor*2 + patch, nil
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
