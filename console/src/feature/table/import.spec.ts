// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { table as clientTable } from "@synnaxlabs/client";
import { type record } from "@synnaxlabs/x";
import { describe, expect, it, vi } from "vitest";

import { Table } from "@/feature/table";
import { client, project } from "@/feature/table/testutil";
import { createFileIngesterContext } from "@/platform/import/testutil";
import { type Panel } from "@/platform/panel";
import { assertDefined, awaitGranted, uniqueName } from "@/testutil";

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
    const migrated = Table.anyStateZ.parse(state);
    expect(migrated.version).toBe("1.0.0");
    expect(migrated.editable).toBeDefined();
    expect(migrated.selectedCells).toEqual([]);
  });

  it("should not produce a pendingUpload when v0 was remoteCreated", () => {
    const migrated = Table.anyStateZ.parse(populatedV0State({ remoteCreated: true }));
    expect(migrated.pendingUpload).toBeUndefined();
  });

  it("should project v0 layout + cells into pendingUpload when not remoteCreated", () => {
    const upload = Table.anyStateZ.parse(
      populatedV0State({ remoteCreated: false }),
    ).pendingUpload;
    assertDefined(upload, "expected pendingUpload to be defined");
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
    const upload = Table.anyStateZ.parse(
      populatedV0State({
        remoteCreated: false,
        cells: {
          a: { key: "a", variant: "text", selected: true, props: {} },
        },
      }),
    ).pendingUpload;
    assertDefined(upload, "expected pendingUpload to be defined");
    const cell = upload.cells.a as Record<string, unknown>;
    expect(cell.selected).toBeUndefined();
  });

  it("should preserve editable and lastSelected across migration", () => {
    const migrated = Table.anyStateZ.parse(
      populatedV0State({ editable: false, lastSelected: "b" }),
    );
    expect(migrated.editable).toBe(false);
    expect(migrated.lastSelected).toEqual("b");
  });
});

// A legacy v0 console export parks its structural model under layout.rows /
// layout.columns and stores a per-cell `selected` flag. It carries a valid uuid key and
// (in the swallow-prone case) a name, so the lenient typed tableZ would accept it
// directly — dropping the layout and yielding empty rows/columns.
const LEGACY_V0 = {
  version: "0.0.0",
  type: "table",
  key: "88aee41e-53b7-4a76-9df9-aceccc220089",
  name: "My Table",
  lastSelected: null,
  editable: true,
  remoteCreated: false,
  layout: {
    rows: [{ size: 36, cells: [{ key: "c1" }, { key: "c2" }] }],
    columns: [{ size: 72 }, { size: 72 }],
  },
  cells: {
    c1: { key: "c1", variant: "text", selected: false, props: { value: "hello" } },
    c2: { key: "c2", variant: "value", selected: false, props: {} },
  },
};

const TYPED_EXPORT = {
  key: "88aee41e-53b7-4a76-9df9-aceccc220089",
  name: "My Table",
  type: "table",
  version: "1.0.0",
  rows: [{ size: 36, cells: ["c1", "c2"] }],
  columns: [{ size: 72 }, { size: 72 }],
  cells: {
    c1: { key: "c1", variant: "text", props: { value: "hello" } },
    c2: { key: "c2", variant: "value", props: {} },
  },
};

const cellsOf = (t: { cells?: unknown }): record.Unknown =>
  typeof t.cells === "object" && t.cells != null ? { ...t.cells } : {};

describe("table import", () => {
  describe("parseImport", () => {
    it("should migrate a legacy console export, preserving rows, columns, and cells", () => {
      const out = Table.parseImport(LEGACY_V0, undefined);
      expect(out.rows).toHaveLength(1);
      expect(out.columns).toHaveLength(2);
      expect(Object.keys(cellsOf(out))).toEqual(["c1", "c2"]);
    });

    it("should not silently drop the layout by parsing a legacy file as a typed one", () => {
      const out = Table.parseImport(LEGACY_V0, undefined);
      expect(out.rows).not.toHaveLength(0);
      expect(out.columns).not.toHaveLength(0);
    });

    it("should import a typed table export directly, preserving structure", () => {
      const out = Table.parseImport(TYPED_EXPORT, undefined);
      expect(out.rows).toHaveLength(1);
      expect(out.columns).toHaveLength(2);
      expect(Object.keys(cellsOf(out))).toEqual(["c1", "c2"]);
    });
  });
});

describe("table ingest", () => {
  it("creates the table on the cluster and opens it as a tab", async () => {
    await awaitGranted(client, clientTable.TYPE_ONTOLOGY_ID, "update");
    const openTab = vi.fn<Panel.OpenTab>();
    const name = uniqueName("imported");
    const id = await Table.ingest(
      TYPED_EXPORT,
      createFileIngesterContext({
        name,
        openTab,
        client,
        projectKey: await project(),
      }),
    );
    expect(openTab).toHaveBeenCalledTimes(1);
    assertDefined(id, "ingest returned no ontology id");
    const created = await client.tables.retrieve({ key: id.key });
    expect(created.name).toBe(name);
    expect(created.rows).toHaveLength(1);
    const cached = client.tables.getCached({ key: id.key });
    if (cached?.variant !== "changed") throw new Error("expected a cached table");
    expect(cached.data.name).toBe(name);
  });

  it("rejects the import when the permission cache has no grant", async () => {
    await expect(
      Table.ingest(
        TYPED_EXPORT,
        createFileIngesterContext({ name: "denied", client: null }),
      ),
    ).rejects.toThrow("You do not have permission to import tables");
  });

  it("throws when a legacy state carries no structural data", () => {
    expect(() => Table.parseImport(V1_ZERO, undefined)).toThrow(
      "Imported table has no structural data",
    );
  });
});
