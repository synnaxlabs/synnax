// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package legacy is the single entry point for migrating an opaque line plot data blob
// through the chain of historical wire formats up to the latest legacy snapshot,
// v4.Data. Each subpackage v0..v4 owns a frozen Data shape and a single Migrate
// function that lifts the previous version's Data into its own; this package owns the
// version dispatch and the forward chain, so callers never have to think about either.
package legacy

import (
	"github.com/synnaxlabs/synnax/pkg/service/imex"
	v0 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/legacy/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/legacy/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/legacy/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/legacy/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/legacy/v4"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
)

// Data is the latest legacy snapshot; the migration chain terminates in it.
type Data = v4.Data

// MigrateData decodes the opaque line plot data blob, dispatches on its declared
// version, and walks the per-step Migrate functions forward to Data. A nil blob and a
// blob without a version field both fall through to v0 and walk the full chain.
func MigrateData(blob msgpack.EncodedJSON) (Data, error) {
	version, err := imex.PeekVersion(blob, "line plot data")
	if err != nil {
		return Data{}, err
	}
	return dispatch(blob, version)
}

func dispatch(blob msgpack.EncodedJSON, version imex.Version) (Data, error) {
	switch version {
	case v4.Version:
		return imex.DecodeBlob[v4.Data](blob, "line plot data", version)
	case v3.Version:
		d, err := imex.DecodeBlob[v3.Data](blob, "line plot data", version)
		if err != nil {
			return Data{}, err
		}
		return v4.Migrate(d), nil
	case v2.Version:
		d, err := imex.DecodeBlob[v2.Data](blob, "line plot data", version)
		if err != nil {
			return Data{}, err
		}
		return v4.Migrate(v3.Migrate(d)), nil
	case v1.Version:
		d, err := imex.DecodeBlob[v1.Data](blob, "line plot data", version)
		if err != nil {
			return Data{}, err
		}
		return v4.Migrate(v3.Migrate(v2.Migrate(d))), nil
	case v0.Version:
		d, err := imex.DecodeBlob[v0.Data](blob, "line plot data", version)
		if err != nil {
			return Data{}, err
		}
		return v4.Migrate(v3.Migrate(v2.Migrate(v1.Migrate(d)))), nil
	default:
		return Data{}, errors.Newf("unknown line plot data version %d", version)
	}
}
