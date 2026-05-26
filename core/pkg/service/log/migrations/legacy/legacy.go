// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package legacy is the single entry point for migrating an opaque log data payload
// through the chain of historical wire formats up to the latest legacy snapshot,
// v1.Data. Each subpackage v0..v1 owns a frozen Data shape and (for v1) a Migrate
// function that lifts the previous version's Data into its own; this package owns the
// version dispatch and the forward chain, so callers never have to think about either.
package legacy

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	v0 "github.com/synnaxlabs/synnax/pkg/service/log/migrations/legacy/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/log/migrations/legacy/v1"
)

// MigrateData parses a stored log data payload at its declared version and walks the
// chain of typed lifts forward to the latest legacy snapshot, v1.Data. v1 is parsed via
// v1.ParseLenient (which scrubs invalid color hex pre-unmarshal); no closed-set
// validation is performed, so any enum value outside the closed set flows through as a
// typed string for the latest-Log lift to substitute. Lower versions fall through the
// v0→v1 chain because their fields are too narrow to be invalid in the same way. A
// version greater than v1.Version is rejected as imex.ErrUnsupportedVersion.
func MigrateData(version imex.Version, data map[string]any) (v1.Data, error) {
	if version > v1.Version {
		return v1.Data{}, imex.NewErrUnsupportedVersion(
			string(ontology.ResourceTypeLog), version, v1.Version,
		)
	}
	switch version {
	case v1.Version:
		return v1.ParseLenient(data)
	default:
		d, err := v0.Parse(data)
		if err != nil {
			return v1.Data{}, err
		}
		return v1.Migrate(d), nil
	}
}
