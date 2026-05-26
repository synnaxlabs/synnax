// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package legacy is the single entry point for migrating an opaque log data blob through
// the chain of historical wire formats up to the latest legacy snapshot, v1.Data. Each
// subpackage v0..v1 owns a frozen Data shape and (for v1) a Migrate function that lifts
// the previous version's Data into its own; this package owns the version-string
// dispatch and the forward chain, so callers never have to think about either.
package legacy

import (
	v0 "github.com/synnaxlabs/synnax/pkg/service/log/migrations/legacy/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/log/migrations/legacy/v1"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
)

// MigrateData decodes the opaque log data blob, dispatches on its declared version, and
// walks the per-step Migrate functions forward to v1.Data. A nil blob and a blob without
// a version field both fall through to v0 and walk the full chain. v1 is parsed via
// v1.ParseLenient (which scrubs invalid color hex pre-unmarshal); no closed-set
// validation is performed, so any enum value outside the closed set flows through as a
// typed string for the latest-Log lift to substitute.
func MigrateData(blob msgpack.EncodedJSON) (v1.Data, error) {
	var peek struct {
		Version string `json:"version"`
	}
	if blob != nil {
		if err := blob.Unmarshal(&peek); err != nil {
			return v1.Data{}, errors.Wrap(err, "peek log data version")
		}
	}
	switch peek.Version {
	case v1.Version:
		return v1.ParseLenient(map[string]any(blob))
	case v0.Version, "":
		d, err := v0.Parse(map[string]any(blob))
		if err != nil {
			return v1.Data{}, err
		}
		return v1.Migrate(d), nil
	default:
		return v1.Data{}, errors.Newf("unknown log data version %q", peek.Version)
	}
}
