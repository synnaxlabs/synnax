// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TableCells, Theming } from "@synnaxlabs/pluto";
import { id, type record, xy } from "@synnaxlabs/x";
import { z } from "zod";

const VERSION = "0.0.0";

const cellLayoutZ = z.object({ key: z.string() });

export const BASE_COL_SIZE = 6 * 12;
export const BASE_ROW_SIZE = 6 * 6;

export type CellLayout = z.infer<typeof cellLayoutZ>;

const rowLayoutZ = z.object({ size: z.number(), cells: z.array(cellLayoutZ) });

export type RowLayout = z.infer<typeof rowLayoutZ>;

const colLayoutZ = z.object({ size: z.number() });

const cellStateZ = z.object({
  key: z.string(),
  variant: z.string(),
  selected: z.boolean(),
  props: z.unknown(),
});

export interface CellState<
  V extends TableCells.Variant = TableCells.Variant,
  P extends object = record.Unknown,
> extends z.infer<typeof cellStateZ> {
  variant: V;
  props: P;
}

export const ZERO_TEXT_CELL_PROPS = TableCells.CELLS.text.defaultProps(
  Theming.themeZ.parse(Theming.SYNNAX_THEMES.synnaxDark),
);

export const ZERO_CELL_STATE: CellState = {
  key: "",
  variant: "text",
  selected: false,
  props: ZERO_TEXT_CELL_PROPS,
};

export const stateZ = z.object({
  key: z.string(),
  version: z.literal(VERSION),
  lastSelected: z.string().nullable(),
  editable: z.boolean(),
  layout: z.object({ rows: z.array(rowLayoutZ), columns: colLayoutZ.array() }),
  cells: z.record(z.string(), cellStateZ),
  remoteCreated: z.boolean(),
});

export type State = z.infer<typeof stateZ>;

const cellOneKey = id.create();
const cellTwoKey = id.create();
const cellThreeKey = id.create();
const cellFourKey = id.create();

export const ZERO_STATE: State = {
  key: "",
  version: VERSION,
  lastSelected: null,
  editable: true,
  remoteCreated: false,
  layout: {
    rows: [
      { size: BASE_ROW_SIZE, cells: [{ key: cellOneKey }, { key: cellTwoKey }] },
      { size: BASE_ROW_SIZE, cells: [{ key: cellThreeKey }, { key: cellFourKey }] },
    ],
    columns: [{ size: BASE_COL_SIZE }, { size: BASE_COL_SIZE }],
  },
  cells: {
    [cellOneKey]: { ...ZERO_CELL_STATE, key: cellOneKey },
    [cellTwoKey]: { ...ZERO_CELL_STATE, key: cellTwoKey },
    [cellThreeKey]: { ...ZERO_CELL_STATE, key: cellThreeKey },
    [cellFourKey]: { ...ZERO_CELL_STATE, key: cellFourKey },
  },
};

export const sliceStateZ = z.object({
  version: z.literal(VERSION),
  tables: z.record(z.string(), stateZ),
  copyBuffer: z.object({
    epicenter: z.string(),
    cells: z.record(z.string(), cellStateZ),
    positions: z.record(z.string(), xy.xy),
  }),
});

export type SliceState = z.infer<typeof sliceStateZ>;

export const ZERO_SLICE_STATE: SliceState = {
  version: VERSION,
  tables: {},
  copyBuffer: { epicenter: "", cells: {}, positions: {} },
};
