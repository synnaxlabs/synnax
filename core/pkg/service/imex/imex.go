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
// importable/exportable resource. The wire shape is flat at the top level:
//
//	{"version":1,"type":"log","name":"...","channels":[...]}
//
// Version, Type, and Name are promoted to typed fields for routing, access control, and
// file naming. The rest of the body is opaque to the envelope: on import it is retained
// as raw bytes and decoded straight into a typed payload via Decode[T]; on export it is
// built by Encode[T], which reduces a typed value to a codec-independent map and merges
// the headers in. Individual services register themselves as Importers and Exporters
// with the Service registry and are routed by Type.
package imex

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/encoding"
	xjson "github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
)

// Key is the string-form identifier of a resource within its ontology type — i.e. the
// Key field of an ontology.ID. Importers return one on persist, Exporters look one up
// on retrieve.
type Key = string

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

// Envelope is the portable format for a single importable/exportable resource. The
// public fields hold the wire headers; the body is private — raw bytes plus a bound
// codec on the import path, a codec-independent map on the export path. Services never
// touch the body directly: import handlers call Decode[T] to materialize the payload
// (using the codec the matching UnmarshalX method bound on the way in), and export
// handlers call Encode[T] to construct an envelope from a typed value.
type Envelope struct {
	// Version is the per-schema integer version stamped on every envelope.
	Version Version
	// Type is the routing key used to look up the registered Importer / Exporter. For
	// services with asymmetric registration (a single task service registering under
	// fine-grained type strings like "http_read" or "opc_scan"), Type is the
	// fine-grained string; the broader ontology resource type is recovered through the
	// Importer's Type() method.
	Type string
	// Name is the human-readable name of the resource. Required on export — Encode
	// enforces that the input value carries a top-level string `name` field.
	Name string

	// codec is the encoding.Codec bound by the UnmarshalX method that produced this
	// envelope, and is what Decode[T] uses to interpret raw.
	codec encoding.Codec
	// raw holds the original wire bytes of an import payload, captured by the matching
	// UnmarshalX method. Decode[T] feeds these straight into the requested type via
	// codec, so the body is parsed once, in the right shape, with no untyped
	// intermediate.
	raw []byte
	// body holds the export-side representation: the typed value reduced to a
	// codec-independent map by Encode, with the headers merged in. Marshalling emits it
	// directly; nil for import-side envelopes.
	body map[string]any
}

// MarshalJSON emits the body built by Encode. Hand-constructed envelopes (no Encode
// call, no wire round-trip) have a nil body and marshal as JSON null — callers should
// go through Encode to produce a wire-valid payload.
func (e Envelope) MarshalJSON() ([]byte, error) { return json.Marshal(e.body) }

// UnmarshalJSON reads a flat JSON object, extracts the promoted headers, retains the
// original bytes for a later typed decode through Decode[T], and binds xjson.Codec on
// the receiver as the codec Decode[T] should use. The peek runs in UseNumber mode so
// JSON numbers come through as json.Number, preserving full int64 precision for the
// Version header. The rest of the body is left untouched until Decode[T] is called, at
// which point the bound codec parses it directly into the target type.
func (e *Envelope) UnmarshalJSON(b []byte) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return err
	}
	return e.unmarshal(m, b, xjson.Codec)
}

// unmarshal is the codec-agnostic tail shared by every UnmarshalX method on Envelope.
// Given the body already decoded as a flat map, the original wire bytes, and the codec
// that produced them, it promotes the {version, type, name} headers onto the receiver
// and stashes raw + codec for the later typed decode via Decode[T]. A nil m (e.g. a
// JSON `null` payload) is treated as a zero envelope: no headers, no codec binding, so
// Decode[T] surfaces the "no body to decode" path rather than silently succeeding.
func (e *Envelope) unmarshal(m map[string]any, raw []byte, codec encoding.Codec) error {
	if m == nil {
		return nil
	}
	if v, ok := m["version"]; ok {
		ver, err := versionFromAny(v)
		if err != nil {
			return err
		}
		e.Version = ver
	}
	if v, ok := m["type"]; ok {
		s, ok := v.(string)
		if !ok {
			return errors.Newf("type must be a string, got %T", v)
		}
		e.Type = s
	}
	if v, ok := m["name"]; ok {
		s, ok := v.(string)
		if !ok {
			return errors.Newf("name must be a string, got %T", v)
		}
		e.Name = s
	}
	e.codec = codec
	e.raw = raw
	return nil
}

// Decode materializes the envelope body as T using the encoding.Codec bound by the
// UnmarshalX method that produced this envelope. For envelopes built by Encode (export
// path) or constructed by hand without a wire round-trip, the body map is re-serialized
// through xjson.Codec. The flat wire shape means T may simply name the fields it cares
// about — unknown headers (version, type, name) are ignored by the bound codec.
//
// ctx is forwarded to the codec; xjson.Codec ignores it, but other in-tree codecs
// (msgpack, future YAML/TOML) may use it for tracing or cancellation.
//
// Decode is a free function because Go does not support generic methods; it becomes
// (e Envelope) Decode[T](ctx) when the language does.
func Decode[T any](ctx context.Context, e Envelope) (T, error) {
	var t T
	codec := e.codec
	if codec == nil {
		codec = xjson.Codec
	}
	src := e.raw
	if src == nil {
		if e.body == nil {
			return t, errors.New("envelope has no body to decode")
		}
		b, err := json.Marshal(e.body)
		if err != nil {
			return t, errors.Wrap(err, "decode envelope body")
		}
		src = b
	}
	if err := codec.Decode(ctx, src, &t); err != nil {
		var zero T
		return zero, errors.Wrap(err, "decode envelope body")
	}
	return t, nil
}

// Encode is the symmetric inverse of Decode. The typed value is reduced to a
// codec-independent map[string]any keyed by JSON tag name (via structToMap — fields
// without a `json:"name"` tag are dropped from the wire body), the headers are merged
// in as flat top-level entries, and the resulting envelope is ready to be handed to
// any of the MarshalX methods.
//
// Invariant: every imex-registered resource carries a top-level string `name` field
// on the wire — i.e. a Go field tagged `json:"name"`. Encode enforces this so that a
// resource without a name surfaces as a programmer bug at exporter-test time rather
// than at runtime.
func Encode[T any](
	data T, version Version, typ ontology.ResourceType,
) (Envelope, error) {
	body := structToMap(data)
	name, ok := body["name"].(string)
	if !ok {
		return Envelope{}, errors.Newf(
			"encode envelope: data must carry a top-level string `name` field",
		)
	}
	body["version"] = version
	body["type"] = string(typ)
	return Envelope{
		Version: version,
		Type:    string(typ),
		Name:    name,
		body:    body,
	}, nil
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
func legacyToNumeric(s string) (Version, error) {
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
	return Version(major), nil
}

// Importer materializes a resource from an Envelope and persists it. The envelope's
// Type is informational only, since the registry has already routed to this handler.
type Importer interface {
	// Import validates and persists the given envelope on tx, returning the
	// newly-assigned key for the imported resource.
	Import(context.Context, gorp.Tx, Envelope) (Key, error)
	// Type returns the broader ontology resource type the importer creates. For
	// services with asymmetric registration (e.g. a task service registered under
	// "http_read" and "opc_scan") this is the coarser ontology type ("task"); it is
	// the resource type used for access control and ontology accounting.
	Type() ontology.ResourceType
}

// Exporter serializes a stored resource as an Envelope, stamping its own per-schema
// version on the returned value.
type Exporter interface {
	// Export retrieves the resource identified by key on tx and serializes it as an
	// envelope, stamping the exporter's per-schema Version on the result.
	Export(context.Context, gorp.Tx, Key) (Envelope, error)
	// Type returns the ontology resource type this exporter handles.
	Type() ontology.ResourceType
}

// ImportExporter is a service that implements both Importer and Exporter under the
// same ontology resource type and can be registered with Service.RegisterImportExporter.
type ImportExporter interface {
	Importer
	Exporter
}
