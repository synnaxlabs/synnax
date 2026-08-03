// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package legacy is the single entry point for migrating an opaque log data blob
// through the chain of historical wire formats up to the latest legacy snapshot,
// v1.Data. Each subpackage v0..v1 owns a frozen Data shape and (for v1) a Migrate
// function that lifts the previous version's Data into its own; this package owns the
// version-string dispatch and the forward chain, so callers never have to think about
// either.
package legacy

import (
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	v0 "github.com/synnaxlabs/synnax/pkg/service/log/versions/legacy/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/log/versions/legacy/v1"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
)

// MigrateData decodes the opaque log data blob, dispatches on its declared version, and
// walks the per-step Migrate functions forward to v1.Data. A nil blob and a blob
// without a version field both fall through to v0 and walk the full chain. Enum strings
// outside their closed sets flow through untouched for the latest-Log lift to default.
func MigrateData(blob msgpack.EncodedJSON) (v1.Data, error) {
	var peek struct {
		Version imex.Version `json:"version"`
	}
	if blob != nil {
		if err := blob.Unmarshal(&peek); err != nil {
			return v1.Data{}, errors.Wrap(err, "peek log data version")
		}
	}
	return dispatch(blob, peek.Version)
}

func dispatch(blob msgpack.EncodedJSON, version imex.Version) (v1.Data, error) {
	switch version {
	case v1.Version:
		return decode[v1.Data](blob, version)
	case v0.Version:
		d, err := decode[v0.Data](blob, version)
		if err != nil {
			return v1.Data{}, err
		}
		return v1.Migrate(d), nil
	default:
		return v1.Data{}, errors.Newf("unknown log data version %d", version)
	}
}

// decode unmarshals blob as T, treating a nil blob as a zero T so empty entries
// round-trip without erroring.
func decode[T any](blob msgpack.EncodedJSON, version imex.Version) (T, error) {
	var d T
	if blob == nil {
		return d, nil
	}
	if err := blob.Unmarshal(&d); err != nil {
		return d, errors.Wrapf(err, "decode v%d log data", version)
	}
	return d, nil
}
