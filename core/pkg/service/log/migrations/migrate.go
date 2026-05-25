// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package migrations is the single entry point for migrating an opaque log data payload
// through the chain of historical wire formats up to the latest version. Each
// subpackage v0..vN owns a frozen Data shape and (for vN > v0) a typed Migrate function
// that lifts the previous version's Data into its own. This package owns the version-
// to-parser dispatch and the forward lift chain, so the per-version subpackages contain
// no version-dispatch logic.
package migrations

import (
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	v0 "github.com/synnaxlabs/synnax/pkg/service/log/migrations/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/log/migrations/v1"
)

// LatestData is the typed Data shape at the latest supported version.
type LatestData = v1.Data

// LatestVersion is the highest log data version this Core understands.
const LatestVersion = v1.Version

// Migrate parses a wire-format log payload at its declared version and walks the chain
// of typed lifts forward to Latest. v1 is parsed via json.Unmarshal then strictly
// validated; a malformed color hex or an enum value outside the closed set surfaces as
// an error. A version greater than LatestVersion is rejected as
// imex.ErrUnsupportedVersion; an unspecified or pre-v0 payload is parsed as v0 and
// migrated forward. Used by the imex import path.
func Migrate(version imex.Version, data map[string]any) (LatestData, error) {
	if version > LatestVersion {
		return LatestData{}, imex.NewErrUnsupportedVersion(
			string(ontology.ResourceTypeLog), version, LatestVersion,
		)
	}
	switch version {
	case v1.Version:
		d, err := v1.Parse(data)
		if err != nil {
			return LatestData{}, err
		}
		if err := d.Validate(); err != nil {
			return LatestData{}, err
		}
		return d, nil
	default:
		d, err := v0.Parse(data)
		if err != nil {
			return LatestData{}, err
		}
		if err := d.Validate(); err != nil {
			return LatestData{}, err
		}
		return v1.Migrate(d), nil
	}
}

// MigrateLenient is the gorp-boot variant of Migrate. v1 is parsed via v1.ParseLenient
// (which scrubs invalid color hex pre-unmarshal) and Validate is skipped entirely, so
// any enum value outside the closed set flows through as a typed string for the
// latest-Log lift to substitute. Lower versions fall through the same v0→v1 chain
// because their fields are too narrow to be invalid in the same way.
func MigrateLenient(version imex.Version, data map[string]any) (LatestData, error) {
	if version > LatestVersion {
		return LatestData{}, imex.NewErrUnsupportedVersion(
			string(ontology.ResourceTypeLog), version, LatestVersion,
		)
	}
	switch version {
	case v1.Version:
		return v1.ParseLenient(data)
	default:
		d, err := v0.Parse(data)
		if err != nil {
			return LatestData{}, err
		}
		return v1.Migrate(d), nil
	}
}
