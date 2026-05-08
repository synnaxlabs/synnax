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
	"maps"
	"strconv"
	"strings"

	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
)

// Version is the per-schema integer version stamped on every envelope. It accepts both
// numeric values (the canonical form) and semver strings (the legacy form emitted by
// older Console exports) on the wire — see UnmarshalJSON.
type Version uint64

// UnmarshalJSON accepts either a JSON number or a semver-style string ("1.0.0"). The
// semver form is converted by taking the major component; minor and patch are discarded.
func (v *Version) UnmarshalJSON(b []byte) error {
	var n uint64
	if err := json.Unmarshal(b, &n); err == nil {
		*v = Version(n)
		return nil
	}
	var s string
	if err := json.Unmarshal(b, &s); err == nil {
		major, err := legacyToNumeric(s)
		if err != nil {
			return err
		}
		*v = Version(major)
		return nil
	}
	return errors.Newf("version must be a number or semver string, got %s", string(b))
}

// Envelope is the portable format for a single importable/exportable resource. All
// fields are flat at the JSON level. The wire format looks like:
//
//	{"version":1,"type":"log","name":"...","channels":[...]}
//
// Version, Type, and Name are promoted to typed fields for convenient access (routing,
// file naming). Data holds the schema-specific payload as a generic map; the promoted
// fields are stripped from Data on unmarshal and re-merged on marshal. The resource key
// is not part of the envelope: import always assigns a fresh key, and export takes the
// key as a separate argument.
type Envelope struct {
	Version Version
	Type    string
	Name    string
	Data    map[string]any
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
// remaining keys into Data. The version field is dispatched to Version.UnmarshalJSON,
// which accepts both numeric values and legacy semver strings.
func (e *Envelope) UnmarshalJSON(b []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	if v, ok := raw["version"]; ok {
		if err := json.Unmarshal(v, &e.Version); err != nil {
			return err
		}
		delete(raw, "version")
	}
	if v, ok := raw["type"]; ok {
		if err := json.Unmarshal(v, &e.Type); err != nil {
			return err
		}
		delete(raw, "type")
	}
	if v, ok := raw["name"]; ok {
		if err := json.Unmarshal(v, &e.Name); err != nil {
			return err
		}
		delete(raw, "name")
	}
	if len(raw) == 0 {
		return nil
	}
	e.Data = make(map[string]any, len(raw))
	for k, v := range raw {
		var d any
		if err := json.Unmarshal(v, &d); err != nil {
			return err
		}
		e.Data[k] = d
	}
	return nil
}

// legacyToNumeric converts a semver string like "1.0.0" to a numeric version by taking
// the major component. Per-schema versioning starts at 0 and increments by 1, so the
// minor and patch components carry no meaning at the import boundary; they are accepted
// for backward compatibility with older payloads but discarded.
func legacyToNumeric(s string) (uint64, error) {
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return 0, errors.Newf("invalid semver %q: expected major.minor.patch", s)
	}
	major, err := strconv.ParseUint(parts[0], 10, 64)
	if err != nil {
		return 0, errors.Wrapf(err, "invalid semver major %q", parts[0])
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
	Import(context.Context, gorp.Tx, Envelope) (string, error)
	Type() ontology.ResourceType
}

// Exporter can export a resource to an Envelope. The exporter is responsible for
// stamping its own per-schema Version on the returned envelope. Type returns the
// ontology resource type this exporter handles.
type Exporter interface {
	Export(context.Context, string) (Envelope, error)
	Type() ontology.ResourceType
}

// ImportExporter is a service that implements both the Importer and Exporter interfaces
// and can be registered with RegisterImportExporter.
type ImportExporter interface {
	Importer
	Exporter
}
