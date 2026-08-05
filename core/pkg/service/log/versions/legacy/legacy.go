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
// version dispatch and the forward chain, so callers never have to think about either.
package legacy

import (
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	v0 "github.com/synnaxlabs/synnax/pkg/service/log/versions/legacy/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/log/versions/legacy/v1"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
)

// Data is the latest legacy snapshot; the migration chain terminates in it.
type Data = v1.Data

// MigrateData decodes the opaque log data blob, dispatches on its declared version, and
// walks the per-step Migrate functions forward to Data. A nil blob and a blob without a
// version field both fall through to v0 and walk the full chain. Enum strings outside
// their closed sets flow through untouched for the latest-Log lift to default.
func MigrateData(blob msgpack.EncodedJSON) (Data, error) {
	version, err := imex.PeekVersion(blob, "log data")
	if err != nil {
		return Data{}, err
	}
	switch version {
	case v1.Version:
		return imex.DecodeBlob[v1.Data](blob, "log data", version)
	case v0.Version:
		d, err := imex.DecodeBlob[v0.Data](blob, "log data", version)
		if err != nil {
			return Data{}, err
		}
		return v1.Migrate(d), nil
	default:
		return Data{}, errors.Newf("unknown log data version %d", version)
	}
}
