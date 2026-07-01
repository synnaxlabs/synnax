// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { parseImport } from "@/session/table/migrations";

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
      const out = parseImport(LEGACY_V0, undefined);
      expect(out.rows).toHaveLength(1);
      expect(out.columns).toHaveLength(2);
      expect(Object.keys(cellsOf(out))).toEqual(["c1", "c2"]);
    });

    it("should not silently drop the layout by parsing a legacy file as a typed one", () => {
      const out = parseImport(LEGACY_V0, undefined);
      expect(out.rows).not.toHaveLength(0);
      expect(out.columns).not.toHaveLength(0);
    });

    it("should import a typed table export directly, preserving structure", () => {
      const out = parseImport(TYPED_EXPORT, undefined);
      expect(out.rows).toHaveLength(1);
      expect(out.columns).toHaveLength(2);
      expect(Object.keys(cellsOf(out))).toEqual(["c1", "c2"]);
    });
  });
});
