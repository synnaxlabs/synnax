import { TableCells, Theming } from "@synnaxlabs/pluto";
import { id, type UnknownRecord, xy } from "@synnaxlabs/x";
import { z } from "zod";

const cellLayout = z.object({
  key: z.string(),
});

export const BASE_COL_SIZE = 6 * 12;
export const BASE_ROW_SIZE = 6 * 6;

export type CellLayout = z.infer<typeof cellLayout>;

const rowLayout = z.object({
  size: z.number(),
  cells: z.array(cellLayout),
});

export type RowLayout = z.infer<typeof rowLayout>;

const colLayout = z.object({
  size: z.number(),
});

const cellState = z.object({
  key: z.string(),
  variant: z.string(),
  selected: z.boolean(),
  props: z.unknown(),
});

export interface CellState<
  V extends TableCells.Variant = TableCells.Variant,
  P extends object = UnknownRecord,
> extends z.infer<typeof cellState> {
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
  version: z.literal("0.0.0"),
  lastSelected: z.string().nullable(),
  editable: z.boolean(),
  layout: z.object({
    rows: z.array(rowLayout),
    columns: colLayout.array(),
  }),
  cells: z.record(z.string(), cellState),
});

export type State = z.infer<typeof stateZ>;

const cellOneKey = id.id();
const cellTwoKey = id.id();
const cellThreeKey = id.id();
const cellFourKey = id.id();

export const ZERO_STATE: State = {
  key: "",
  version: "0.0.0",
  lastSelected: null,
  editable: true,
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
  version: z.literal("0.0.0"),
  tables: z.record(z.string(), stateZ),
  copyBuffer: z.object({
    epicenter: z.string(),
    cells: z.record(z.string(), cellState),
    positions: z.record(z.string(), xy.xy),
  }),
});

export type SliceState = z.infer<typeof sliceStateZ>;

export const ZERO_SLICE_STATE: SliceState = {
  version: "0.0.0",
  tables: {},
  copyBuffer: {
    epicenter: "",
    cells: {},
    positions: {},
  },
};
