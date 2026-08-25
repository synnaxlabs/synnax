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

import { query } from "@/query";

const noopError = (_: Error) => {};

describe("indexes", () => {
  interface Entry extends record.Keyed<string> {
    key: string;
    group: string;
  }

  const newIndexed = () => {
    const byGroup = new query.LookupIndex<string, Entry>((e) => e.group);
    const table = new query.Table<string, Entry>({
      onError: noopError,
      indexes: [byGroup],
    });
    return { table, byGroup };
  };

  it("should index entries as they are set", () => {
    const { table, byGroup } = newIndexed();
    table.set([
      { key: "k1", group: "a" },
      { key: "k2", group: "a" },
      { key: "k3", group: "b" },
    ]);
    expect(byGroup.get("a")).toEqual([
      { key: "k1", group: "a" },
      { key: "k2", group: "a" },
    ]);
    expect(byGroup.get("b")).toEqual([{ key: "k3", group: "b" }]);
    expect(byGroup.get("c")).toEqual([]);
  });

  it("should relocate an entry whose indexed value changes", () => {
    const { table, byGroup } = newIndexed();
    table.set("k1", { key: "k1", group: "a" });
    table.set("k1", { key: "k1", group: "b" });
    expect(byGroup.get("a")).toEqual([]);
    expect(byGroup.get("b")).toEqual([{ key: "k1", group: "b" }]);
  });

  it("should serve the latest value after an update within the same bucket", () => {
    interface Wide extends record.Keyed<string> {
      key: string;
      group: string;
      name: string;
    }
    const byGroup = new query.LookupIndex<string, Wide>((e) => e.group);
    const table = new query.Table<string, Wide>({
      onError: noopError,
      indexes: [byGroup],
    });
    table.set("k1", { key: "k1", group: "a", name: "one" });
    table.set("k1", { key: "k1", group: "a", name: "two" });
    expect(byGroup.get("a")).toEqual([{ key: "k1", group: "a", name: "two" }]);
  });

  it("should unindex deleted and evicted entries", () => {
    const { table, byGroup } = newIndexed();
    table.set([
      { key: "k1", group: "a" },
      { key: "k2", group: "a" },
    ]);
    table.delete("k1");
    table.evict("k2");
    expect(byGroup.get("a")).toEqual([]);
  });

  it("should unindex a new entry when its set is rolled back", () => {
    const { table, byGroup } = newIndexed();
    const rollback = table.set("k1", { key: "k1", group: "a" });
    rollback();
    expect(byGroup.get("a")).toEqual([]);
  });

  it("should restore the prior bucket when an overwrite is rolled back", () => {
    const { table, byGroup } = newIndexed();
    table.set("k1", { key: "k1", group: "a" });
    const rollback = table.set("k1", { key: "k1", group: "b" });
    rollback();
    expect(byGroup.get("a")).toEqual([{ key: "k1", group: "a" }]);
    expect(byGroup.get("b")).toEqual([]);
  });

  it("should reindex a deleted entry when the delete is rolled back", () => {
    const { table, byGroup } = newIndexed();
    table.set("k1", { key: "k1", group: "a" });
    const rollback = table.delete("k1");
    expect(byGroup.get("a")).toEqual([]);
    rollback();
    expect(byGroup.get("a")).toEqual([{ key: "k1", group: "a" }]);
  });

  it("should clear every index on reset", () => {
    const { table, byGroup } = newIndexed();
    table.set([
      { key: "k1", group: "a" },
      { key: "k2", group: "b" },
    ]);
    table.reset();
    expect(byGroup.get("a")).toEqual([]);
    expect(byGroup.get("b")).toEqual([]);
  });

  it("should keep multiple indexes current", () => {
    const byGroup = new query.LookupIndex<string, Entry>((e) => e.group);
    const byKey = new query.LookupIndex<string, Entry>((e) => e.key);
    const table = new query.Table<string, Entry>({
      onError: noopError,
      indexes: [byGroup, byKey],
    });
    table.set("k1", { key: "k1", group: "a" });
    expect(byGroup.get("a")).toEqual([{ key: "k1", group: "a" }]);
    expect(byKey.get("k1")).toEqual([{ key: "k1", group: "a" }]);
    table.delete("k1");
    expect(byGroup.get("a")).toEqual([]);
    expect(byKey.get("k1")).toEqual([]);
  });
});

describe("partial indexes", () => {
  interface Entry extends record.Keyed<string> {
    key: string;
    group: string | null;
  }

  const newIndexed = () => {
    const byGroup = new query.LookupIndex<string, Entry>((e) => e.group);
    const table = new query.Table<string, Entry>({
      onError: noopError,
      indexes: [byGroup],
    });
    return { table, byGroup };
  };

  it("should leave entries whose extract returns null out of the index", () => {
    const { table, byGroup } = newIndexed();
    table.set([
      { key: "k1", group: "a" },
      { key: "k2", group: null },
    ]);
    expect(byGroup.get("a")).toEqual([{ key: "k1", group: "a" }]);
  });

  it("should unindex an entry whose extracted value becomes null", () => {
    const { table, byGroup } = newIndexed();
    table.set("k1", { key: "k1", group: "a" });
    table.set("k1", { key: "k1", group: null });
    expect(byGroup.get("a")).toEqual([]);
  });

  it("should index an entry whose extracted value appears", () => {
    const { table, byGroup } = newIndexed();
    table.set("k1", { key: "k1", group: null });
    table.set("k1", { key: "k1", group: "a" });
    expect(byGroup.get("a")).toEqual([{ key: "k1", group: "a" }]);
  });
});
