// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package legacy is the single entry point for migrating an opaque schematic data blob
// through the chain of historical wire formats up to the latest legacy snapshot,
// v5.Data. Each subpackage v0..v5 owns a frozen Data shape and a single Migrate
// function that lifts the previous version's Data into its own; this package owns the
// version dispatch and the forward chain, so callers never have to think about either.
package legacy

import (
	"slices"

	"github.com/synnaxlabs/synnax/pkg/service/imex"
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v4"
	v5 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/legacy/v5"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
)

// Data is the latest legacy snapshot; the migration chain terminates in it.
type Data = v5.Data

// MigrateData decodes the opaque schematic data blob, dispatches on its declared
// version, and walks the per-step Migrate functions forward to Data. A nil blob and a
// blob without a version field both fall through to v0 and walk the full chain. Orphan
// edges (empty source or target — persisted by ReactFlow after partial-drop
// interactions) are filtered after the chain runs since the condition is uniform across
// every legacy version.
func MigrateData(blob msgpack.EncodedJSON) (Data, error) {
	version, err := imex.PeekVersion(blob, "schematic data")
	if err != nil {
		return Data{}, err
	}
	d, err := dispatch(blob, version)
	if err != nil {
		return Data{}, err
	}
	d.Edges = slices.DeleteFunc(d.Edges, func(e v3.Edge) bool {
		return e.Source == "" || e.Target == ""
	})
	return d, nil
}

func dispatch(blob msgpack.EncodedJSON, version imex.Version) (Data, error) {
	switch version {
	case v5.Version:
		return imex.DecodeBlob[v5.Data](blob, "schematic data", version)
	case v4.Version:
		d, err := imex.DecodeBlob[v4.Data](blob, "schematic data", version)
		if err != nil {
			return Data{}, err
		}
		return v5.Migrate(d), nil
	case v3.Version:
		d, err := imex.DecodeBlob[v3.Data](blob, "schematic data", version)
		if err != nil {
			return Data{}, err
		}
		return v5.Migrate(v4.Migrate(d)), nil
	case v2.Version:
		d, err := imex.DecodeBlob[v2.Data](blob, "schematic data", version)
		if err != nil {
			return Data{}, err
		}
		return v5.Migrate(v4.Migrate(v3.Migrate(d))), nil
	case v1.Version:
		d, err := imex.DecodeBlob[v1.Data](blob, "schematic data", version)
		if err != nil {
			return Data{}, err
		}
		return v5.Migrate(v4.Migrate(v3.Migrate(v2.Migrate(d)))), nil
	case v0.Version:
		d, err := imex.DecodeBlob[v0.Data](blob, "schematic data", version)
		if err != nil {
			return Data{}, err
		}
		return v5.Migrate(v4.Migrate(v3.Migrate(v2.Migrate(v1.Migrate(d))))), nil
	default:
		return Data{}, errors.Newf("unknown schematic data version %d", version)
	}
}
