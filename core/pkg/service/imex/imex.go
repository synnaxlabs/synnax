// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package imex defines the Envelope, the portable format for one importable or
// exportable resource. The wire shape is flat at the top level:
//
//	{"version":1,"type":"log","name":"...","channels":[...]}
//
// Version, Type, and Name are promoted to typed fields for routing, access control, and
// file naming; the rest of the body is opaque, decoded by Decode and built by Encode.
// Services register with the Service registry and are routed by Type.
package imex

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"strconv"
	"strings"

	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	"github.com/synnaxlabs/x/encoding"
	xjson "github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/validate"
)

// Version is the per-schema integer version stamped on every envelope. It decodes from
// the canonical numeric form and from the legacy "N.0.0" semver strings older Console
// exports wrote.
type Version uint64

// UnmarshalJSON decodes a Version from the numeric JSON form or a legacy "N.0.0" semver
// string.
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

// NewErrUnsupportedVersion reports that given exceeds the highest version this Core
// supports for typ. The error is path-scoped to the "version" field.
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

// Envelope is the portable format for one importable or exportable resource. The public
// fields hold the wire headers; the body is private. Import handlers discriminate with
// Body and BodyNamed and call Decode once; exporters call Encode.
type Envelope struct {
	// Version is the per-schema integer version stamped on every envelope.
	Version Version
	// Type is the routing key for the registered Importer or Exporter. A service may
	// register under fine-grained strings ("http_read") while Importer.Type reports the
	// coarser ontology type ("task"). Empty on legacy Console state files, which
	// Service.Import routes through Match.
	Type string
	// Name is the resource's human-readable name. Encode requires it on export. On
	// import
	// it may be empty until Service.Import applies the opts.FileName fallback.
	Name string

	codec encoding.Codec
	raw   []byte
	body  map[string]any
}

// MarshalJSON emits the body built by Encode. It returns an error when the envelope has
// no body, so a service that returns an empty Envelope from Export fails loudly instead
// of sending null.
func (e Envelope) MarshalJSON() ([]byte, error) {
	if e.body == nil {
		return nil, errors.New(
			"envelope has no body; build one with Encode before marshaling",
		)
	}
	return json.Marshal(e.body)
}

// UnmarshalJSON reads a flat JSON object, promoting the headers and retaining the bytes
// for a later Decode. Numbers decode in UseNumber mode so the Version keeps full int64
// precision.
func (e *Envelope) UnmarshalJSON(b []byte) error {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return err
	}
	return e.unmarshal(m, b, xjson.Codec)
}

// unmarshal promotes the {version, type, name} headers onto the receiver and stashes
// raw and codec for a later Decode. Both `type` and `name` are optional: legacy Console
// state files carry neither.
func (e *Envelope) unmarshal(m map[string]any, raw []byte, codec encoding.Codec) error {
	if m == nil {
		return errors.Wrap(validate.ErrValidation, "envelope must be a JSON object")
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
			return newFieldError("type", "type must be a string, got %T", v)
		}
		e.Type = s
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
	e.body = m
	return nil
}

// Body returns the envelope body as the flat map parsed from the wire (or built by
// Encode). It is nil on hand-constructed envelopes. Callers must not mutate it.
func (e Envelope) Body() map[string]any { return e.body }

// BodyNamed reports whether the body carries a top-level `name` field. Every typed
// export does; legacy Console-state files never do. Distinct from Envelope.Name, which
// the registry may have filled from the file name.
func (e Envelope) BodyNamed() bool {
	_, ok := e.body["name"]
	return ok
}

// Decode materializes the envelope body as T using the codec bound when the envelope
// was unmarshaled. T may name only the fields it needs. Envelopes built by Encode have
// no codec bound and must be marshaled and unmarshaled before Decode.
//
// Decode is a free function because Go has no generic methods.
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

// Encode reduces data to a flat map and stamps it onto env as the body, merging in the
// Version, Type, and Name headers. data wins for `type` and `name`; both must end up
// non-empty. A top-level `key` is always dropped — importers mint a fresh one. On any
// error env is left untouched.
//
// data may be a map[string]any, which is merged flat; any other value must be a struct.
func Encode[T any](env *Envelope, data T) error {
	// The map branch exists only for the task exporter, whose config is an opaque
	// object it merges flat rather than a Go struct. Once task configs are strongly
	// typed, this assertion (and its test) can go: every caller passes a struct.
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

// legacyToNumeric converts a legacy "N.0.0" semver string to the integer version N.
// Minor and patch must both be zero — older Console exports only ever stamped the
// major.
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

// structToMap reduces a struct to a flat map keyed by each field's json tag name.
// Untagged fields and `json:"-"` are skipped; embedded untagged structs are flattened,
// shallower fields winning, with same-depth collisions resolved last-wins rather than
// dropped as encoding/json would.
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
// desired parent resource.
type ImportOptions struct {
	// FileName is the name of the file the envelope was read from. When the envelope
	// body carries no `name` field, the file name — with any trailing extension
	// stripped — becomes the envelope's name. A `name` in the body always wins. The
	// fallback is applied by the registry before the envelope reaches an Importer.
	FileName string
	// Parent is the ontology resource to create the imported resource under — a
	// project for workspace items, a group for symbols. Required: Service.Import
	// rejects a zero Parent. The registry passes it through untouched; each Importer
	// decides how the parent applies to its resource type.
	Parent ontology.ID
}

// Importer materializes a resource from an Envelope and persists it. The envelope's
// Type is informational only, since the registry has already routed to this handler.
type Importer interface {
	// Import validates and persists the given envelope on tx, returning the ontology.ID
	// of the newly-created resource. The envelope's Name is fully resolved (non-empty)
	// by the time Import is called — the registry has already applied the file-name
	// fallback — so importers should treat it as the resource's name rather than
	// re-deriving one from the body. The importer owns all ontology writes for the
	// resource, including attaching it under opts.Parent when one is given.
	Import(context.Context, gorp.Tx, Envelope, ImportOptions) (ontology.ID, error)
	// Match reports whether body, the envelope body decoded as a flat map, is this
	// importer's resource. It routes envelopes carrying no `type` header: legacy
	// Console state files never carried one, so Service.ResolveType offers the body to
	// every registered importer's Match. Markers tested by Match are frozen — they
	// describe historical file shapes — and must be mutually exclusive across
	// importers. Importers with no typeless legacy formats return false.
	Match(body map[string]any) bool
	// Type returns the broader ontology resource type the importer creates. For
	// services with asymmetric registration (e.g. a task service registered under
	// "http_read" and "opc_scan") this is the coarser ontology type ("task"); it is the
	// resource type used for access control and ontology accounting.
	Type() ontology.ResourceType
}

// Exporter serializes a stored resource as an Envelope, stamping its own per-schema
// version on the returned value.
type Exporter interface {
	// Export retrieves the resource identified by id and serializes it, stamping the
	// exporter's per-schema Version.
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
