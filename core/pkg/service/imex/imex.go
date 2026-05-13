// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package imex provides the core import/export types and interfaces for the Synnax
// Core. It defines the Envelope type, which is the portable format for a single
// importable/exportable resource to be used within Go code. Envelopes. The wire format
// will have fields flatten at the highest level like this:
//
//	{"version":1,"type":"log","name":"...","channels":[...]}
//
// Version, Type, and Name are promoted to typed fields for convenient access (routing,
// file naming, etc.). Individual services register themselves as Importers and
// Exporters for their own Type, and the Service routes to the correct handler based on
// the Type.
package imex

import (
	"bytes"
	"context"
	"encoding/json"
	"maps"
	"strconv"
	"strings"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
)

// Version is the per-schema integer version stamped on every envelope. On the wire it
// is decoded by Envelope.UnmarshalJSON (which accepts both numeric values and legacy
// "N.0.0" semver strings via versionFromAny); standalone JSON unmarshal of a Version
// only accepts the numeric form.
type Version uint64

// NewErrUnsupportedVersion constructs a validation error for the named resource type,
// indicating that the given version exceeds the highest version this Core supports. The
// returned error is path-scoped to the "version" field so API responses can present it
// as a structured field error.
func NewErrUnsupportedVersion(typ string, given, supported Version) error {
	return validate.PathedError(
		errors.Wrapf(
			validate.ErrValidation,
			"%s version %d is newer than this Core supports (latest: %d)",
			typ, given, supported,
		),
		"version",
	)
}

// Envelope is the portable format for a single importable/exportable resource. All
// fields are flat when transported over the wire. The wire format looks like:
//
//	{"version":1,"type":"log","name":"...","channels":[...]}
//
// Version, Type, and Name are promoted to typed fields for convenient access (routing,
// file naming). Data holds the schema-specific payload as a generic map; the promoted
// fields are stripped from Data on unmarshal and re-merged on marshal.
type Envelope struct {
	// Version is the per-schema integer version stamped on every envelope.
	Version Version
	// Type describes the resource type being imported/exported.
	Type string
	// Name is the human-readable name of the resource.
	Name string
	// Data holds the schema-specific payload as a generic map.
	Data map[string]any
}

// MarshalJSON emits the flat wire format by merging the promoted fields onto a copy of
// Data. Promoted fields always win over any same-named entry already present in Data,
// so the handler-stamped export version wins over a stale value the schema may have
// carried.
func (e Envelope) MarshalJSON() ([]byte, error) {
	fields := make(map[string]any, len(e.Data)+3)
	maps.Copy(fields, e.Data)
	fields["version"] = e.Version
	fields["type"] = e.Type
	fields["name"] = e.Name
	return json.Marshal(fields)
}

// UnmarshalJSON reads a flat JSON object, extracts the promoted fields, and puts the
// remaining keys into Data. It does this with a single pass: the decoder runs in
// UseNumber mode so JSON numbers come through as json.Number (preserving full int64
// precision regardless of magnitude), strings as string, bools as bool, and so on. The
// promoted fields are plucked out and type-asserted; the leftover map becomes Data
// directly — no per-key re-parse.
func (e *Envelope) UnmarshalJSON(b []byte) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return err
	}
	if v, ok := m["version"]; ok {
		ver, err := versionFromAny(v)
		if err != nil {
			return err
		}
		e.Version = ver
		delete(m, "version")
	}
	if v, ok := m["type"]; ok {
		s, ok := v.(string)
		if !ok {
			return errors.Newf("type must be a string, got %T", v)
		}
		e.Type = s
		delete(m, "type")
	}
	if v, ok := m["name"]; ok {
		s, ok := v.(string)
		if !ok {
			return errors.Newf("name must be a string, got %T", v)
		}
		e.Name = s
		delete(m, "name")
	}
	if len(m) > 0 {
		e.Data = m
	}
	return nil
}

// versionFromAny converts a generic Go value (as produced by a UseNumber-mode decode
// into map[string]any) to a Version. Accepts json.Number (the JSON number form,
// preserving full integer precision) and legacy "N.0.0" semver strings.
func versionFromAny(v any) (Version, error) {
	switch x := v.(type) {
	case json.Number:
		n, err := strconv.ParseUint(x.String(), 10, 64)
		if err != nil {
			return 0, errors.Wrapf(err, "invalid version number %q", x.String())
		}
		return Version(n), nil
	case string:
		n, err := legacyToNumeric(x)
		if err != nil {
			return 0, err
		}
		return Version(n), nil
	default:
		return 0, errors.Newf("version must be a number or semver string, got %T", v)
	}
}

// legacyToNumeric converts a legacy semver string of the form "N.0.0" into the integer
// schema version N. The minor and patch components must both be zero — older Console
// exports only ever stamped the major component, so any non-zero minor/patch indicates
// either a malformed payload or a wire format we don't recognize.
func legacyToNumeric(s string) (uint64, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return 0, errors.Newf("invalid version %q: expected N.0.0", s)
	}
	major, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, errors.Wrapf(err, "invalid version major %q", parts[0])
	}
	if parts[1] != "0" || parts[2] != "0" {
		return 0, errors.Newf("invalid version %q: only N.0.0 is supported", s)
	}
	return major, nil
}

// Importer can import a resource from an Envelope. It returns the new key assigned to
// the imported resource. The envelope's Type is informational only, since the registry
// has already routed to this handler. Type returns the broader ontology resource type
// the importer creates (e.g., an "http_read" task importer registered under "http_read"
// still returns "task" from Type). This is the resource type used for access control
// and ontology accounting.
type Importer interface {
	// Import validates and persists the given envelope within a single transaction,
	// returning the newly-assigned key for the imported resource.
	Import(context.Context, gorp.Tx, Envelope) (string, error)
	// Type returns the broader ontology resource type the importer creates.
	Type() ontology.ResourceType
}

// Exporter can export a resource to an Envelope. The exporter is responsible for
// stamping its own per-schema Version on the returned envelope. Type returns the
// ontology resource type this exporter handles.
type Exporter interface {
	// Export serializes the given resource as an envelope, stamping the exporter's own
	// per-schema Version on the returned envelope.
	Export(context.Context, string) (Envelope, error)
	// Type returns the ontology resource type this exporter handles.
	Type() ontology.ResourceType
}

// ImportExporter is a service that implements both the Importer and Exporter interfaces
// and can be registered with RegisterImportExporter.
type ImportExporter interface {
	// Importer allows the service to be registered as an Importer.
	Importer
	// Exporter allows the service to be registered as an Exporter.
	Exporter
}
