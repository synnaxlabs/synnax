// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2

import (
	"context"
	"encoding/json"

	v1 "github.com/synnaxlabs/synnax/pkg/service/table/versions/v1"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/gorp"
)

// MigrateTable transforms the previous Table snapshot (v1) into the v2 strongly-typed
// Table. v1 stored cell props as opaque records written verbatim by the Console
// (camelCase field keys, never validated); each cell is normalized to the snake_case
// wire form and decoded into the typed cell config union. Cells that fail to decode
// fall back to an empty text cell rather than being dropped, so rows never reference
// missing entries.
func MigrateTable(ctx context.Context, old v1.Table) (Table, error) {
	out, err := autoMigrateTable(ctx, old)
	if err != nil {
		return Table{}, err
	}
	out.Cells = make(map[string]CellConfig, len(old.Cells))
	for k, c := range old.Cells {
		out.Cells[k] = migrateCell(c)
	}
	return out, nil
}

// migrateCell converts a v1 stored cell (variant string plus camelCase props written
// verbatim by the Console) into the typed cell config. Entries that conform to no known
// variant degrade to an empty text cell.
func migrateCell(c v1.Cell) CellConfig {
	fields := normalizeConfigKeys(c.Props)
	if fields == nil {
		fields = msgpack.EncodedJSON{}
	}
	fields["variant"] = c.Variant
	extractLegacyArgs(fields)
	cfg, err := decodeCellConfig(fields)
	if err != nil {
		cfg = CellConfig{Variant: TextCellConfig{}}
	}
	cfg.ApplyDefaults()
	return cfg
}

// decodeCellConfig validates an opaque cell config payload against the cell config
// union, returning an error when the payload conforms to no known variant.
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

// Migration lifts stored tables from the opaque v1 cell props to the typed v2 cell
// config union.
var Migration = gorp.NewEntryMigration("v2_typed_cell_configs", MigrateTable)
