// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package legacy is the single entry point for migrating an opaque schematic
// data blob through the chain of historical wire formats up to the latest
// legacy snapshot, v5.Data. Each subpackage v0..v5 owns a frozen Data shape
// and a single Migrate function that lifts the previous version's Data into
// its own; this package owns the version-string dispatch and the forward
// chain, so callers never have to think about either.
package legacy

import (
	"slices"

	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy/v4"
	v5 "github.com/synnaxlabs/synnax/pkg/service/schematic/migrations/legacy/v5"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
)

// MigrateData decodes the opaque schematic data blob, dispatches on its
// declared version, and walks the per-step Migrate functions forward to
// v5.Data. A nil blob and a blob without a version field both fall through
// to v0 and walk the full chain. Orphan edges (empty source or target —
// persisted by ReactFlow after partial-drop interactions) are filtered
// after the chain runs since the condition is uniform across every legacy
// version.
func MigrateData(blob msgpack.EncodedJSON) (v5.Data, error) {
	var peek struct {
		Version string `json:"version"`
	}
	if blob != nil {
		if err := blob.Unmarshal(&peek); err != nil {
			return v5.Data{}, errors.Wrap(err, "peek schematic data version")
		}
	}
	d, err := dispatch(blob, peek.Version)
	if err != nil {
		return v5.Data{}, err
	}
	d.Edges = slices.DeleteFunc(d.Edges, func(e v3.Edge) bool {
		return e.Source == "" || e.Target == ""
	})
	return d, nil
}

func dispatch(blob msgpack.EncodedJSON, version string) (v5.Data, error) {
	switch version {
	case v5.Version:
		return decode[v5.Data](blob, version)
	case v4.Version:
		d, err := decode[v4.Data](blob, version)
		if err != nil {
			return v5.Data{}, err
		}
		return v5.Migrate(d), nil
	case v3.Version:
		d, err := decode[v3.Data](blob, version)
		if err != nil {
			return v5.Data{}, err
		}
		return v5.Migrate(v4.Migrate(d)), nil
	case v2.Version:
		d, err := decode[v2.Data](blob, version)
		if err != nil {
			return v5.Data{}, err
		}
		return v5.Migrate(v4.Migrate(v3.Migrate(d))), nil
	case v1.Version:
		d, err := decode[v1.Data](blob, version)
		if err != nil {
			return v5.Data{}, err
		}
		return v5.Migrate(v4.Migrate(v3.Migrate(v2.Migrate(d)))), nil
	case v0.Version, "":
		d, err := decode[v0.Data](blob, version)
		if err != nil {
			return v5.Data{}, err
		}
		return v5.Migrate(v4.Migrate(v3.Migrate(v2.Migrate(v1.Migrate(d))))), nil
	default:
		return v5.Data{}, errors.Newf("unknown schematic data version %q", version)
	}
}

// decode unmarshals blob as T, treating a nil blob as a zero T so empty
// entries round-trip without erroring.
func decode[T any](blob msgpack.EncodedJSON, version string) (T, error) {
	var d T
	if blob == nil {
		return d, nil
	}
	if err := blob.Unmarshal(&d); err != nil {
		return d, errors.Wrapf(err, "decode v%s schematic data", version)
	}
	return d, nil
}
