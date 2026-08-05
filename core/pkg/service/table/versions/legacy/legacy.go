// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package legacy is the single entry point for migrating an opaque table data blob
// through the chain of historical wire formats up to the latest legacy snapshot,
// v0.Data. Each subpackage owns a frozen Data shape and a single Migrate function that
// lifts the previous version's Data into its own; this package owns the version
// dispatch and the forward chain, so callers never have to think about either. Table
// currently has a single legacy snapshot, but the package shape matches line plot and
// schematic so adding a future v1 is mechanical.
package legacy

import (
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	v0 "github.com/synnaxlabs/synnax/pkg/service/table/versions/legacy/v0"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
)

// Data is the latest legacy snapshot; the migration chain terminates in it.
type Data = v0.Data

// MigrateData decodes the opaque table data blob, dispatches on its declared version,
// and walks the per-step Migrate functions forward to Data. A nil blob and a blob
// without a version field both fall through to v0.
func MigrateData(blob msgpack.EncodedJSON) (Data, error) {
	version, err := imex.PeekVersion(blob, "table data")
	if err != nil {
		return Data{}, err
	}
	switch version {
	case v0.Version:
		return imex.DecodeBlob[v0.Data](blob, "table data", version)
	default:
		return Data{}, errors.Newf("unknown table data version %d", version)
	}
}
