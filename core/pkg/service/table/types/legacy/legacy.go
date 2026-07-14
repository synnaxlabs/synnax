// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package legacy is the single entry point for migrating an opaque table data
// blob through the chain of historical wire formats up to the latest legacy
// snapshot, v0.Data. Each subpackage owns a frozen Data shape and a single
// Migrate function that lifts the previous version's Data into its own; this
// package owns the version-string dispatch and the forward chain, so callers
// never have to think about either. Table currently has a single legacy
// snapshot, but the package shape matches line plot and schematic so adding a
// future v1 is mechanical.
package legacy

import (
	v0 "github.com/synnaxlabs/synnax/pkg/service/table/types/legacy/v0"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
)

// MigrateData decodes the opaque table data blob, dispatches on its declared
// version, and walks the per-step Migrate functions forward to v0.Data. A nil
// blob and a blob without a version field both fall through to v0.
func MigrateData(blob msgpack.EncodedJSON) (v0.Data, error) {
	var peek struct {
		Version string `json:"version"`
	}
	if blob != nil {
		if err := blob.Unmarshal(&peek); err != nil {
			return v0.Data{}, errors.Wrap(err, "peek table data version")
		}
	}
	return dispatch(blob, peek.Version)
}

func dispatch(blob msgpack.EncodedJSON, version string) (v0.Data, error) {
	switch version {
	case v0.Version, "":
		return decode[v0.Data](blob, version)
	default:
		return v0.Data{}, errors.Newf("unknown table data version %q", version)
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
		return d, errors.Wrapf(err, "decode v%s table data", version)
	}
	return d, nil
}
