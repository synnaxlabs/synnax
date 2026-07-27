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
	"reflect"
	"strconv"
	"strings"

	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/x/encoding"
	xjson "github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
)

// Version is the per-schema integer version stamped on every envelope. On the wire it
// is the canonical numeric form, but it also decodes from the legacy "N.0.0" semver
// strings older Console exports wrote — see UnmarshalJSON. This holds both for the
// envelope header and for a standalone Version decoded out of a versioned payload.
type Version uint64

// UnmarshalJSON decodes a Version from either the canonical numeric JSON form or a
// legacy "N.0.0" semver string written by older Console exports. Decoding the version
// directly into a Version field — rather than a string that a caller must then parse —
// is why the legacy migration packages can peek the stamped version straight into a
// Version.
func (v *Version) UnmarshalJSON(b []byte) error {
	var n uint64
	if err := json.Unmarshal(b, &n); err == nil {
		*v = Version(n)
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return errors.Newf("version must be a number or semver string, got %s", b)
	}
	parsed, err := legacyToNumeric(s)
	if err != nil {
		return err
	}
	*v = parsed
	return nil
}

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

// newFieldError constructs a validation error scoped to the named wire field so API
// responses can present it as a structured field error, mirroring the path-scoping done
// by NewErrUnsupportedVersion for the "version" field.
func newFieldError(field, format string, args ...any) error {
	return validate.PathedError(
		errors.Wrapf(validate.ErrValidation, format, args...),
		field,
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
	// enforces that the input value carries a top-level string `name` field. On import
	// it may be empty at decode time; Service.Import fills it from the caller-supplied
	// file name and rejects the envelope if it is still empty after that fallback.
	Name string

	codec encoding.Codec
	raw   []byte
	body  map[string]any
}

// MarshalJSON emits the body built by Encode. Hand-constructed envelopes (no Encode
// call, no wire round-trip) have a nil body and would otherwise marshal as JSON null,
// which is rarely what the caller intended; MarshalJSON instead returns an error so a
// service that accidentally returns an empty Envelope from Export surfaces the bug
// loudly rather than silently sending null over the wire.
func (e Envelope) MarshalJSON() ([]byte, error) {
	if e.body == nil {
		return nil, errors.New(
			"envelope has no body; build one with Encode before marshaling",
		)
	}
	return json.Marshal(e.body)
}

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
// and stashes raw + codec for the later typed decode via Decode[T]. `type` is a
// required header — an envelope that omits it, or that carries an empty string for it,
// is rejected so the failure surfaces at the transport boundary instead of routing to a
// no-op handler. `name` is optional at decode time: the import path may fall back to a
// caller-supplied file name, so Service.Import enforces the non-empty name after that
// fallback is applied.
func (e *Envelope) unmarshal(m map[string]any, raw []byte, codec encoding.Codec) error {
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
			return newFieldError("type", "type must be a string, got %T", v)
		}
		e.Type = s
	}
	if e.Type == "" {
		return newFieldError("type", "type must be a non-empty string")
	}
	if v, ok := m["name"]; ok {
		s, ok := v.(string)
		if !ok {
			return newFieldError("name", "name must be a string, got %T", v)
		}
		e.Name = s
	}
	e.codec = codec
	e.raw = raw
	return nil
}

// Decode materializes the envelope body as T using the encoding.Codec bound by the
// UnmarshalX method that produced this envelope. The flat wire shape means T may simply
// name the fields it cares about — unknown headers (version, type, name) are ignored by
// the bound codec. Envelopes built by Encode without a wire round-trip have no codec
// bound; callers that need to decode an Encode-side envelope must Marshal it and
// unmarshal back through one of the UnmarshalX methods first.
//
// ctx is forwarded to the codec; xjson.Codec ignores it, but other in-tree codecs
// (MessagePack, future YAML/TOML) may use it for tracing or cancellation.
//
// Decode is a free function because Go does not support generic methods; it becomes (e
// Envelope) Decode[T](ctx) when the language does.
func Decode[T any](ctx context.Context, e Envelope) (T, error) {
	var t T
	if e.codec == nil {
		return t, errors.New(
			"decode envelope body: envelope has no codec bound; " +
				"Encode-side envelopes must be marshaled and unmarshaled through " +
				"one of the UnmarshalX methods before Decode",
		)
	}
	if err := e.codec.Decode(ctx, e.raw, &t); err != nil {
		var zero T
		return zero, errors.Wrap(err, "decode envelope body")
	}
	return t, nil
}

// Encode is the symmetric inverse of Decode. The caller supplies an envelope carrying
// the desired Version, Type, and (optionally) Name headers; Encode reduces data to a
// codec-independent map[string]any and stamps the merged body onto the envelope. A
// struct is reduced field-by-field via structToMap; a value that is already a
// map[string]any is used directly, letting a caller whose portable body is an opaque
// object (rather than a Go struct) merge it flat into the envelope. For both Type and
// Name, Encode treats data as the source of truth: if the body map carries a `type` (or
// `name`) entry, it must be a string and it overwrites the corresponding header on the
// envelope; otherwise the envelope's existing value is kept. At the end, Type and Name
// must both be non-empty. On any error the envelope is left untouched.
//
// Invariant: every imex-registered resource carries a non-empty top-level string `name`
// field on the wire. Encode enforces this so that a resource without a name surfaces as
// a programmer bug at exporter-test time rather than at runtime.
//
// A top-level `key` field is always dropped from the body: envelopes do not carry
// resource-local identity today, and importers mint a fresh key on the way in.
func Encode[T any](env *Envelope, data T) error {
	body, ok := any(data).(map[string]any)
	if !ok {
		var err error
		if body, err = structToMap(data); err != nil {
			return errors.Wrap(err, "encode envelope")
		}
	}
	// Keys are resource-local identity, not part of the portable envelope: an imported
	// resource is minted a fresh key on the way in, so a stale key on the wire is at
	// best noise and at worst a collision hazard. Strip it here. This may change if
	// envelopes ever need to carry stable identity across clusters.
	delete(body, "key")
	typ := env.Type
	if v, ok := body["type"]; ok {
		s, ok := v.(string)
		if !ok {
			return newFieldError("type", "type must be a string, got %T", v)
		}
		typ = s
	}
	if typ == "" {
		return newFieldError("type", "type must be a non-empty string")
	}
	name := env.Name
	if v, ok := body["name"]; ok {
		s, ok := v.(string)
		if !ok {
			return newFieldError("name", "name must be a string, got %T", v)
		}
		name = s
	}
	if name == "" {
		return newFieldError("name", "name must be a non-empty string")
	}
	body["version"] = env.Version
	body["type"] = typ
	body["name"] = name
	env.Type = typ
	env.Name = name
	env.body = body
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

// structToMap reduces a struct value to a flat map[string]any keyed by the name in each
// field's `json:"..."` struct tag. A field is emitted iff it carries a json tag whose
// name component is non-empty and not "-". Fields without a tag, with `json:"-"`, or
// with an empty tag name (e.g. `json:",omitempty"`) are skipped. Tag options after the
// name are ignored. Embedded (anonymous, untagged) struct fields are promoted: their
// fields are flattened into the top-level map.
//
// Promotion follows encoding/json's depth rule — a shallower field overrides a
// same-named field promoted from a deeper embedded struct — but does not reproduce its
// full conflict resolution: two fields with the same json name promoted from the same
// depth are resolved last-wins here, whereas encoding/json drops both.
func structToMap(v any) (map[string]any, error) {
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Struct {
		return nil, errors.Newf("expected struct, got %s", rv.Kind())
	}
	m := make(map[string]any)
	flattenStruct(rv, m)
	return m, nil
}

func flattenStruct(rv reflect.Value, m map[string]any) {
	t := rv.Type()
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.Anonymous {
			continue
		}
		if _, ok := f.Tag.Lookup("json"); ok {
			continue
		}
		fv := rv.Field(i)
		if fv.Kind() == reflect.Pointer {
			if fv.IsNil() {
				continue
			}
			fv = fv.Elem()
		}
		if fv.Kind() == reflect.Struct {
			flattenStruct(fv, m)
		}
	}
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		tag, ok := f.Tag.Lookup("json")
		if !ok {
			continue
		}
		name, _, _ := strings.Cut(tag, ",")
		if name == "" || name == "-" {
			continue
		}
		m[name] = rv.Field(i).Interface()
	}
}

// ImportOptions carries the per-request settings for an import that arrive out-of-band
// from the envelope body — transport metadata like the source file's name and the
// desired project resource.
type ImportOptions struct {
	// FileName is the name of the file the envelope was read from. When the envelope
	// body carries no `name` field, the file name — with any trailing extension
	// stripped — becomes the envelope's name. A `name` in the body always wins. The
	// fallback is applied by the registry before the envelope reaches an Importer.
	FileName string
	// Project is the key of the project to create the imported resource under. The
	// registry passes it through untouched: each Importer decides how (and whether) a
	// project applies to its resource type. A zero Project means no project was
	// requested.
	Project project.Key
}

// Importer materializes a resource from an Envelope and persists it. The envelope's
// Type is informational only, since the registry has already routed to this handler.
type Importer interface {
	// Import validates and persists the given envelope on tx, returning the ontology.ID
	// of the newly-created resource. The envelope's Name is fully resolved (non-empty)
	// by the time Import is called — the registry has already applied the file-name
	// fallback — so importers should treat it as the resource's name rather than
	// re-deriving one from the body. The importer owns all ontology writes for the
	// resource, including attaching it under opts.Project when one is given.
	Import(context.Context, gorp.Tx, Envelope, ImportOptions) (ontology.ID, error)
	// Type returns the broader ontology resource type the importer creates. For
	// services with asymmetric registration (e.g. a task service registered under
	// "http_read" and "opc_scan") this is the coarser ontology type ("task"); it is the
	// resource type used for access control and ontology accounting.
	Type() ontology.ResourceType
}

// Exporter serializes a stored resource as an Envelope, stamping its own per-schema
// version on the returned value.
type Exporter interface {
	// Export retrieves the resource identified by id and serializes it as an envelope,
	// stamping the exporter's per-schema Version on the result. Exporters read directly
	// from their own storage handle; the transactional Export path will return in a
	// follow-up change once the API surface is settled.
	Export(context.Context, ontology.ID) (Envelope, error)
	// Type returns the ontology resource type this exporter handles.
	Type() ontology.ResourceType
}

// ImportExporter is a service that implements both Importer and Exporter under the same
// ontology resource type and can be registered with Service.RegisterImportExporter.
type ImportExporter interface {
	Importer
	Exporter
}
