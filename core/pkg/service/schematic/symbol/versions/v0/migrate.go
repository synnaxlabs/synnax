// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import (
	"context"

	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/legacy"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/migrate"
)

// Migrations is the ordered migration chain that lifts stored symbols into the v0
// typed shape.
var Migrations = []migrate.Migration{
	gorp.NewEntryMigration[Key, Key, legacy.Symbol, Symbol](
		"v0_typed_symbol",
		MigrateSymbol,
	),
}

// MigrateSymbol lifts a symbol persisted in the untyped legacy shape into the v0
// Symbol, decoding the specification map into a typed Spec and stamping the schema
// version. The legacy map keys already match Spec's msgpack tags, so the transform is
// a re-encode; it never reshapes stored fields.
func MigrateSymbol(ctx context.Context, old legacy.Symbol) (Symbol, error) {
	out := Symbol{Key: old.Key, Name: old.Name}
	if len(old.Data) > 0 {
		b, err := msgpack.Codec.Encode(ctx, old.Data)
		if err != nil {
			return Symbol{}, errors.Wrap(err, "encode legacy symbol data")
		}
		if err = msgpack.Codec.Decode(ctx, b, &out.Data); err != nil {
			return Symbol{}, errors.Wrap(err, "decode legacy symbol data")
		}
	}
	out.ApplyDefaults()
	return out, nil
}
