// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { table } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import {
  migrateSlice,
  migrateState,
  ZERO_SLICE_STATE,
  ZERO_STATE,
} from "@/table/types";
import * as v0 from "@/table/types/v0";
import * as v1 from "@/table/types/v1";

const populatedV0State = (overrides: Partial<v0.State> = {}): v0.State => ({
  key: "11111111-1111-1111-1111-111111111111",
  version: v0.VERSION,
  lastSelected: null,
  editable: true,
  remoteCreated: true,
  layout: {
    rows: [
      { size: 36, cells: [{ key: "a" }, { key: "b" }] },
      { size: 36, cells: [{ key: "c" }, { key: "d" }] },
    ],
    columns: [{ size: 72 }, { size: 72 }],
  },
  cells: {
    a: { ...v0.ZERO_CELL_STATE, key: "a", props: { value: "A" } },
    b: { ...v0.ZERO_CELL_STATE, key: "b", props: { value: "B" } },
    c: { ...v0.ZERO_CELL_STATE, key: "c", props: { value: "C" } },
    d: { ...v0.ZERO_CELL_STATE, key: "d", props: { value: "D" } },
  },
  ...overrides,
});

describe("table migrations", () => {
  describe("state", () => {
    const STATES = [v0.ZERO_STATE, v1.ZERO_STATE];
    STATES.forEach((state) => {
      it(`should migrate state from ${state.version} to latest`, () => {
        const migrated = migrateState(state);
        expect(migrated.version).toBe(ZERO_STATE.version);
        expect(migrated.editable).toBeDefined();
        expect(migrated.selectedCells).toEqual([]);
      });
    });

    it("should not produce a pendingUpload when v0 was remoteCreated", () => {
      const migrated = migrateState(populatedV0State({ remoteCreated: true }));
      expect(migrated.pendingUpload).toBeUndefined();
    });

    it("should project v0 layout + cells into pendingUpload when not remoteCreated", () => {
      const upload = migrateState(
        populatedV0State({ remoteCreated: false }),
      ).pendingUpload;
      if (upload == null) throw new Error("expected pendingUpload to be defined");
      expect(upload.key).toEqual("11111111-1111-1111-1111-111111111111");
      expect(upload.rows).toHaveLength(2);
      expect(upload.rows[0].cells).toEqual(["a", "b"]);
      expect(upload.rows[1].cells).toEqual(["c", "d"]);
      expect(upload.columns).toEqual([{ size: 72 }, { size: 72 }]);
      expect(Object.keys(upload.cells).sort()).toEqual(["a", "b", "c", "d"]);
      expect(upload.cells.a).toEqual(
        table.cellConfigZ.parse({ variant: "text", value: "A" }),
      );
    });

    it("should extract value cell telem args from the stored pipeline spec", () => {
      const upload = migrateState(
        populatedV0State({
          remoteCreated: false,
          cells: {
            a: {
              key: "a",
              variant: "value",
              selected: false,
              props: {
                telem: {
                  type: "source-pipeline",
                  variant: "source",
                  valueType: "string",
                  props: {
                    segments: {
                      valueStream: { props: { channel: 42 } },
                      rollingAverage: { props: { windowSize: 5 } },
                      stringifier: { props: { precision: 3, notation: "scientific" } },
                    },
                    outlet: "stringifier",
                  },
                },
                units: "psi",
              },
            },
          },
        }),
      ).pendingUpload;
      if (upload == null) throw new Error("expected pendingUpload to be defined");
      expect(upload.cells.a).toMatchObject({
        variant: "value",
        channel: 42,
        rollingAverage: 5,
        precision: 3,
        notation: "scientific",
        units: "psi",
      });
      expect(upload.cells.a).not.toHaveProperty("telem");
    });

    it("should leave the channel at the zero sentinel for a legacy zero channel", () => {
      const upload = migrateState(
        populatedV0State({
          remoteCreated: false,
          cells: {
            a: {
              key: "a",
              variant: "value",
              selected: false,
              props: {
                telem: {
                  props: { segments: { valueStream: { props: { channel: 0 } } } },
                },
              },
            },
          },
        }),
      ).pendingUpload;
      if (upload == null) throw new Error("expected pendingUpload to be defined");
      expect(upload.cells.a).toMatchObject({ variant: "value", channel: 0 });
    });

    it("should map legacy x-location alignments and named weights", () => {
      const upload = migrateState(
        populatedV0State({
          remoteCreated: false,
          cells: {
            a: {
              key: "a",
              variant: "text",
              selected: false,
              props: { value: "hi", align: "left", weight: "bold" },
            },
          },
        }),
      ).pendingUpload;
      if (upload == null) throw new Error("expected pendingUpload to be defined");
      expect(upload.cells.a).toMatchObject({
        variant: "text",
        align: "start",
        weight: 700,
      });
    });

    it("should degrade cells with an unknown variant to an empty text cell", () => {
      const upload = migrateState(
        populatedV0State({
          remoteCreated: false,
          cells: {
            a: {
              key: "a",
              variant: "hologram",
              selected: false,
              props: { value: "hi" },
            },
          },
        }),
      ).pendingUpload;
      if (upload == null) throw new Error("expected pendingUpload to be defined");
      expect(upload.cells.a).toEqual(table.cellConfigZ.parse({ variant: "text" }));
    });

    it("should drop the per-cell selected flag from pendingUpload", () => {
      const upload = migrateState(
        populatedV0State({
          remoteCreated: false,
          cells: {
            a: { ...v0.ZERO_CELL_STATE, key: "a", selected: true, props: {} },
          },
        }),
      ).pendingUpload;
      if (upload == null) throw new Error("expected pendingUpload to be defined");
      const cell = upload.cells.a as Record<string, unknown>;
      expect(cell.selected).toBeUndefined();
      expect(cell.variant).toEqual("text");
    });

    it("should preserve editable and lastSelected across migration", () => {
      const migrated = migrateState(
        populatedV0State({
          editable: false,
          lastSelected: "b",
        }),
      );
      expect(migrated.editable).toBe(false);
      expect(migrated.lastSelected).toEqual("b");
    });
  });

  describe("slice", () => {
    const SLICES = [v0.ZERO_SLICE_STATE, v1.ZERO_SLICE_STATE];
    SLICES.forEach((slice) => {
      it(`should migrate slice from ${slice.version} to latest`, () => {
        const migrated = migrateSlice(slice);
        expect(migrated.version).toBe(ZERO_SLICE_STATE.version);
        expect(migrated.tables).toEqual({});
      });
    });

    it("should run stateMigration on every table in a v0 slice", () => {
      const sliceState: v0.SliceState = {
        version: v0.VERSION,
        tables: {
          remote: populatedV0State({ remoteCreated: true }),
          local: populatedV0State({
            key: "22222222-2222-2222-2222-222222222222",
            remoteCreated: false,
          }),
        },
        copyBuffer: { epicenter: "", cells: {}, positions: {} },
      };
      const migrated = migrateSlice(sliceState);
      expect(migrated.tables.remote.pendingUpload).toBeUndefined();
      const localUpload = migrated.tables.local.pendingUpload;
      if (localUpload == null) throw new Error("expected pendingUpload to be defined");
      expect(localUpload.key).toEqual("22222222-2222-2222-2222-222222222222");
    });
  });
});
