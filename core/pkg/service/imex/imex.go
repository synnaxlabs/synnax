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
	"math"
	"strconv"
	"strings"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
	"gopkg.in/yaml.v3"
)

// Version is the per-schema integer version stamped on every envelope. On the wire it
// is decoded by the envelope's format-specific unmarshalers (which accept numeric values
// and legacy "N.0.0" semver strings via versionFromAny); standalone JSON unmarshal of a
// Version only accepts the numeric form.
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

// flatten returns the flat wire representation of the envelope: a copy of Data with the
// promoted fields merged on top. Promoted fields always win over any same-named entry
// already present in Data, so the handler-stamped export version wins over a stale value
// the schema may have carried. It is the shared core every format-specific marshaler
// renders.
func (e Envelope) flatten() map[string]any {
	fields := make(map[string]any, len(e.Data)+3)
	maps.Copy(fields, e.Data)
	fields["version"] = e.Version
	fields["type"] = e.Type
	fields["name"] = e.Name
	return fields
}

// unflatten populates the envelope from a decoded flat map: it plucks the promoted
// version, type, and name fields into their typed fields and leaves the remaining keys
// as Data. The Go types of the map values depend on the codec that produced it
// (json.Number under JSON's UseNumber decode, native int/int64 under YAML and TOML), so
// versionFromAny accepts every numeric form. It is the shared core every format-specific
// unmarshaler delegates to.
func (e *Envelope) unflatten(m map[string]any) error {
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

// MarshalJSON emits the flat wire format described on flatten.
func (e Envelope) MarshalJSON() ([]byte, error) { return json.Marshal(e.flatten()) }

// UnmarshalJSON reads a flat JSON object into the envelope. The decoder runs in UseNumber
// mode so JSON numbers come through as json.Number, preserving full int64 precision
// regardless of magnitude.
func (e *Envelope) UnmarshalJSON(b []byte) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return err
	}
	return e.unflatten(m)
}

// MarshalYAML emits the flat wire format as YAML by handing the flattened map to the
// yaml encoder; see flatten.
func (e Envelope) MarshalYAML() (any, error) { return e.flatten(), nil }

// UnmarshalYAML reads a flat YAML mapping into the envelope. yaml.v3 decodes string-keyed
// mappings into map[string]any and numbers into native int/float, which unflatten and
// versionFromAny consume directly.
func (e *Envelope) UnmarshalYAML(node *yaml.Node) error {
	var m map[string]any
	if err := node.Decode(&m); err != nil {
		return err
	}
	return e.unflatten(m)
}

// TOMLValue returns the flat wire representation for the TOML codec to encode. go-toml/v2
// exposes no marshal-side interface of its own, so the TOML codec discovers this method
// through its own Marshaler interface; see flatten.
func (e Envelope) TOMLValue() (any, error) { return e.flatten(), nil }

// FromTOMLValue populates the envelope from the TOML codec's decoded table. go-toml/v2's
// native unmarshal hook surfaces only raw bytes, so the TOML codec decodes into a map and
// hands it here through its own Unmarshaler interface; see unflatten.
func (e *Envelope) FromTOMLValue(m map[string]any) error { return e.unflatten(m) }

// versionFromAny converts a generic Go value (as produced by decoding into
// map[string]any) to a Version. The concrete numeric type varies by codec: JSON's
// UseNumber decode yields json.Number (preserving full integer precision), while YAML and
// TOML yield native int/int64. It also accepts uint64, an integral float64, and legacy
// "N.0.0" semver strings.
func versionFromAny(v any) (Version, error) {
	switch x := v.(type) {
	case json.Number:
		n, err := strconv.ParseUint(x.String(), 10, 64)
		if err != nil {
			return 0, errors.Wrapf(err, "invalid version number %q", x.String())
		}
		return Version(n), nil
	case int:
		return versionFromInt64(int64(x))
	case int64:
		return versionFromInt64(x)
	case uint64:
		return Version(x), nil
	case float64:
		if x < 0 || x != math.Trunc(x) {
			return 0, errors.Newf("version must be a non-negative integer, got %v", x)
		}
		return Version(x), nil
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

// versionFromInt64 converts a signed integer version to a Version, rejecting negatives.
func versionFromInt64(n int64) (Version, error) {
	if n < 0 {
		return 0, errors.Newf("version must be non-negative, got %d", n)
	}
	return Version(n), nil
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
