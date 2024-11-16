import { z } from "zod";

const cellLayout = z.object({
  key: z.string(),
});

const rowLayout = z.object({
  cells: z.array(cellLayout),
});

const cellState = z.object({
  key: z.string(),
  type: z.string(),
  selected: z.boolean(),
  props: z.object({}),
});

export type CellState = z.infer<typeof cellState>;

export const stateZ = z.object({
  key: z.string(),
  version: z.literal("0.0.0"),
  lastSelected: z.string().nullable(),
  layout: z.object({
    rows: z.array(rowLayout),
  }),
  cells: z.record(z.string(), cellState),
});

export type State = z.infer<typeof stateZ>;

export const ZERO_STATE: State = {
  key: "",
  version: "0.0.0",
  lastSelected: null,
  layout: {
    rows: [
      {
        cells: [{ key: "123" }],
      },
      {
        cells: [{ key: "456" }],
      },
    ],
  },
  cells: {
    "123": {
      key: "123",
      type: "text",
      props: {},
      selected: true,
    },
    "456": {
      key: "456",
      type: "text",
      props: {},
      selected: false,
    },
  },
};

export const sliceStateZ = z.object({
  version: z.literal("0.0.0"),
  tables: z.record(z.string(), stateZ),
});

export type SliceState = z.infer<typeof sliceStateZ>;

export const ZERO_SLICE_STATE: SliceState = {
  version: "0.0.0",
  tables: {},
};
