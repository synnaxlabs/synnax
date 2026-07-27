// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1

import (
	"context"

	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v0"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
)

// Migration lifts stored symbols out of the untyped v0 shape into the v1 typed shape.
var Migration = gorp.NewEntryMigration("v1_typed_symbol", migrateSymbol)

// migrateSymbol lifts a symbol persisted in the untyped v0 shape into the v1 Symbol,
// decoding the specification map into a typed Spec and stamping the schema version. The
// v0 map keys already match Spec's msgpack tags, so the transform is a re-encode; it
// never reshapes stored fields.
func migrateSymbol(ctx context.Context, old v0.Symbol) (Symbol, error) {
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
