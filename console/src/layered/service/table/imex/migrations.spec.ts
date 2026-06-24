// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { anyStateZ } from "@/layered/service/table/imex/import";

const populatedV0State = (overrides: Record<string, unknown> = {}) => ({
  key: "11111111-1111-1111-1111-111111111111",
  version: "0.0.0",
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
    a: { key: "a", variant: "text", selected: false, props: { value: "A" } },
    b: { key: "b", variant: "text", selected: false, props: { value: "B" } },
    c: { key: "c", variant: "text", selected: false, props: { value: "C" } },
    d: { key: "d", variant: "text", selected: false, props: { value: "D" } },
  },
  ...overrides,
});

const V0_ZERO = {
  key: "00000000-0000-0000-0000-000000000000",
  version: "0.0.0",
  lastSelected: null,
  editable: true,
  remoteCreated: true,
  layout: { rows: [], columns: [] },
  cells: {},
};

const V1_ZERO = {
  key: "00000000-0000-0000-0000-000000000000",
  version: "1.0.0",
  lastSelected: null,
  editable: true,
  selectedCells: [],
  hideIndicators: false,
};

describe("table state migrations", () => {
  it.each([
    ["0.0.0", V0_ZERO],
    ["1.0.0", V1_ZERO],
  ])("should migrate state from %s to latest", (_version, state) => {
    const migrated = anyStateZ.parse(state);
    expect(migrated.version).toBe("1.0.0");
    expect(migrated.editable).toBeDefined();
    expect(migrated.selectedCells).toEqual([]);
  });

  it("should not produce a pendingUpload when v0 was remoteCreated", () => {
    const migrated = anyStateZ.parse(populatedV0State({ remoteCreated: true }));
    expect(migrated.pendingUpload).toBeUndefined();
  });

  it("should project v0 layout + cells into pendingUpload when not remoteCreated", () => {
    const upload = anyStateZ.parse(
      populatedV0State({ remoteCreated: false }),
    ).pendingUpload;
    if (upload == null) throw new Error("expected pendingUpload to be defined");
    expect(upload.key).toEqual("11111111-1111-1111-1111-111111111111");
    expect(upload.rows).toHaveLength(2);
    expect(upload.rows[0].cells).toEqual(["a", "b"]);
    expect(upload.rows[1].cells).toEqual(["c", "d"]);
    expect(upload.columns).toEqual([{ size: 72 }, { size: 72 }]);
    expect(Object.keys(upload.cells).sort()).toEqual(["a", "b", "c", "d"]);
    expect(upload.cells.a.variant).toEqual("text");
    expect((upload.cells.a.props as Record<string, unknown>).value).toEqual("A");
  });

  it("should drop the per-cell selected flag from pendingUpload", () => {
    const upload = anyStateZ.parse(
      populatedV0State({
        remoteCreated: false,
        cells: {
          a: { key: "a", variant: "text", selected: true, props: {} },
        },
      }),
    ).pendingUpload;
    if (upload == null) throw new Error("expected pendingUpload to be defined");
    const cell = upload.cells.a as Record<string, unknown>;
    expect(cell.selected).toBeUndefined();
  });

  it("should preserve editable and lastSelected across migration", () => {
    const migrated = anyStateZ.parse(
      populatedV0State({ editable: false, lastSelected: "b" }),
    );
    expect(migrated.editable).toBe(false);
    expect(migrated.lastSelected).toEqual("b");
  });
});
