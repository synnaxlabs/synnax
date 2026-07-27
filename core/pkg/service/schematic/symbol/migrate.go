// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package symbol

import (
	"context"

	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/migrations/v0"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/migrate"
)

// migrationV0TypedSymbol keys the migration that lifts stored symbols out of
// the untyped v0 shape into the current strongly-typed Symbol.
const migrationV0TypedSymbol = "v0_typed_symbol"

// migrations is the ordered set of migrations run when the symbol table opens.
var migrations = []migrate.Migration{
	gorp.NewEntryMigration[Key, Key, v0.Symbol, Symbol](
		migrationV0TypedSymbol,
		MigrateSymbol,
	),
}

// MigrateSymbol lifts a symbol persisted in the untyped v0 shape into the
// current Symbol, decoding the specification map into a typed Spec and stamping
// the current schema version. The v0 map keys already match Spec's msgpack
// tags, so the transform is a re-encode; it never reshapes stored fields.
func MigrateSymbol(ctx context.Context, old v0.Symbol) (Symbol, error) {
	out := Symbol{Key: old.Key, Name: old.Name}
	if len(old.Data) > 0 {
		b, err := msgpack.Codec.Encode(ctx, old.Data)
		if err != nil {
			return Symbol{}, errors.Wrap(err, "encode v0 symbol data")
		}
		if err = msgpack.Codec.Decode(ctx, b, &out.Data); err != nil {
			return Symbol{}, errors.Wrap(err, "decode v0 symbol data")
		}
	}
	out.ApplyDefaults()
	return out, nil
}
