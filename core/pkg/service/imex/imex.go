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
	"strconv"
	"strings"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
)

// Envelope is the portable format for a single importable/exportable resource.
// All fields are flat at the JSON level. The wire format looks like:
//
//	{"version":1,"type":"log","key":"...","name":"...","channels":[...]}
//
// Version, Type, Key, and Name are promoted to typed fields for convenient
// access (routing, identity, file naming). Data holds the raw JSON bytes of
// the full top-level object, including the promoted fields. On import the Key
// is ignored; each handler always assigns a fresh key.
type Envelope struct {
	Version int
	Type    string
	Key     string
	Name    string
	Data    json.RawMessage
}

// MarshalJSON emits the flat wire format. Promoted fields (version, type,
// optional key and name) are spliced on top of Data's raw bytes so that the
// service-stamped export version wins over any version embedded in Data.
func (e Envelope) MarshalJSON() ([]byte, error) {
	fields := make(map[string]json.RawMessage)
	if len(e.Data) > 0 {
		if err := json.Unmarshal(e.Data, &fields); err != nil {
			return nil, errors.Wrap(err, "envelope data must be a JSON object")
		}
	}
	version, err := json.Marshal(e.Version)
	if err != nil {
		return nil, err
	}
	fields["version"] = version
	if e.Type != "" {
		typ, err := json.Marshal(e.Type)
		if err != nil {
			return nil, err
		}
		fields["type"] = typ
	}
	if e.Key != "" {
		key, err := json.Marshal(e.Key)
		if err != nil {
			return nil, err
		}
		fields["key"] = key
	}
	if e.Name != "" {
		name, err := json.Marshal(e.Name)
		if err != nil {
			return nil, err
		}
		fields["name"] = name
	}
	return json.Marshal(fields)
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
// standard json struct tags. Data receives a copy of the raw bytes with all
// fields intact so handlers can decode into their own typed struct. The
// version field accepts both numeric values (new format) and semver strings
// (old Console format), converting the latter via legacyToNumeric.
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
	// Copy the input so later reuse of the caller's buffer by encoding/json
	// does not corrupt Data.
	e.Data = append(json.RawMessage(nil), b...)
	return nil
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
	return 0, errors.Newf("version must be a number or semver string, got %s", string(raw))
}

// legacyToNumeric converts a semver string like "1.0.0" to a numeric version
// by taking the major component. Per-schema versioning starts at 0 and
// increments by 1, so the minor and patch components carry no meaning at the
// import boundary; they are accepted for backward compatibility with older
// payloads but discarded.
func legacyToNumeric(s string) (int, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return 0, errors.Newf("invalid semver %q: expected major.minor.patch", s)
	}
	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, errors.Wrapf(err, "invalid semver major %q", parts[0])
	}
	return major, nil
}

// ImportPayload is the validated, decoded input handed to an Importer. Type is
// not included: the registry has already routed to the right handler. Key is
// not included: the envelope's key is ignored and each handler assigns a fresh
// key, returned from Import.
type ImportPayload struct {
	Version int
	Name    string
	Data    map[string]any
}

// Importer can import a resource from an ImportPayload. It returns the new key
// assigned to the imported resource.
type Importer interface {
	Import(ctx context.Context, tx gorp.Tx, payload ImportPayload) (string, error)
}

// Exporter can export a resource to an Envelope. The exporter is responsible
// for stamping its own per-schema Version on the returned envelope.
type Exporter interface {
	Export(ctx context.Context, key string) (Envelope, error)
}

// ImporterExporter combines both interfaces.
type ImporterExporter interface {
	Importer
	Exporter
}
