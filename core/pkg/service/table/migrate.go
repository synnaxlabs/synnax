// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package table

import (
	"context"
	"encoding/json"

	v56 "github.com/synnaxlabs/synnax/pkg/service/table/migrations/v56"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
)

// MigrateTable transforms the previous table snapshot (v56) into the current
// Table. v56 stored cell props as opaque records written verbatim by the
// Console (camelCase field keys, never validated); each cell is normalized to
// the snake_case wire form and decoded into the typed cell config union.
// Cells that fail to decode fall back to an empty text cell rather than being
// dropped, so rows never reference missing entries.
func MigrateTable(ctx context.Context, old v56.Table) (Table, error) {
	out, err := AutoMigrateTable(ctx, old)
	if err != nil {
		return Table{}, err
	}
	out.Cells = make(map[string]CellConfig, len(old.Cells))
	for k, c := range old.Cells {
		out.Cells[k] = migrateCellEntry(c)
	}
	return out, nil
}

// migrateCellEntry converts a v56 stored cell (variant string plus camelCase
// props written verbatim by the Console) into the typed cell config. Entries
// that conform to no known variant degrade to an empty text cell.
func migrateCellEntry(c v56.Cell) CellConfig {
	fields := normalizeConfigKeys(c.Props)
	if fields == nil {
		fields = msgpack.EncodedJSON{}
	}
	fields["variant"] = c.Variant
	extractLegacyArgs(fields)
	cfg, err := decodeCellConfig(fields)
	if err != nil {
		return CellConfig{Variant: CellConfigText{}}
	}
	return cfg
}

// decodeCellConfig validates an opaque cell config payload against the cell
// config union, returning an error when the payload conforms to no known
// variant.
func decodeCellConfig(raw msgpack.EncodedJSON) (CellConfig, error) {
	b, err := json.Marshal(raw)
	if err != nil {
		return CellConfig{}, err
	}
	var cfg CellConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return CellConfig{}, errors.Wrap(err, "invalid cell config")
	}
	return cfg, nil
}
