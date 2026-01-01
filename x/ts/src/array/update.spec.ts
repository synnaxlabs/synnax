// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { array } from "@/array";
import { type record } from "@/record";

describe("upsertKeyed", () => {
  it("should update a keyed item in the array", () => {
    const arr: record.KeyedNamed[] = [
      { key: "1", name: "First" },
      { key: "2", name: "Second" },
    ];
    const updated = array.upsertKeyed(arr, {
      key: "1",
      name: "First Updated",
    });
    expect(updated).toHaveLength(2);
    expect(updated).toEqual([
      { key: "1", name: "First Updated" },
      { key: "2", name: "Second" },
    ]);
  });

  it("should append entry when updating non-existent key", () => {
    const arr: record.KeyedNamed[] = [
      { key: "1", name: "First" },
      { key: "2", name: "Second" },
    ];
    const updated = array.upsertKeyed(arr, {
      key: "3",
      name: "Third",
    });
    expect(updated).toHaveLength(3);
    expect(updated).toEqual([
      { key: "1", name: "First" },
      { key: "2", name: "Second" },
      { key: "3", name: "Third" },
    ]);
  });

  it("should handle empty array", () => {
    const arr: record.KeyedNamed[] = [];
    const updated = array.upsertKeyed(arr, { key: "1", name: "Test" });
    expect(updated).toHaveLength(1);
    expect(updated).toEqual([{ key: "1", name: "Test" }]);
  });
});

describe("deleteKeyed", () => {
  it("should delete a single keyed item", () => {
    const arr: record.KeyedNamed[] = [
      { key: "1", name: "First" },
      { key: "2", name: "Second" },
      { key: "3", name: "Third" },
    ];
    const result = array.removeKeyed<string, record.KeyedNamed>(arr, "2");
    expect(result).toHaveLength(2);
    expect(result).toEqual([
      { key: "1", name: "First" },
      { key: "3", name: "Third" },
    ]);
  });

  it("should delete multiple keyed items", () => {
    const arr: record.KeyedNamed[] = [
      { key: "1", name: "First" },
      { key: "2", name: "Second" },
      { key: "3", name: "Third" },
      { key: "4", name: "Fourth" },
    ];
    const result = array.removeKeyed<string, record.KeyedNamed>(arr, ["2", "4"]);
    expect(result).toHaveLength(2);
    expect(result).toEqual([
      { key: "1", name: "First" },
      { key: "3", name: "Third" },
    ]);
  });

  it("should handle deleting non-existent keys", () => {
    const arr: record.KeyedNamed[] = [
      { key: "1", name: "First" },
      { key: "2", name: "Second" },
    ];
    const result = array.removeKeyed<string, record.KeyedNamed>(arr, "3");
    expect(result).toHaveLength(2);
    expect(result).toEqual(arr);
  });

  it("should handle mixed existing and non-existing keys", () => {
    const arr: record.KeyedNamed[] = [
      { key: "1", name: "First" },
      { key: "2", name: "Second" },
      { key: "3", name: "Third" },
    ];
    const result = array.removeKeyed<string, record.KeyedNamed>(arr, ["2", "5", "3"]);
    expect(result).toHaveLength(1);
    expect(result).toEqual([{ key: "1", name: "First" }]);
  });

  it("should handle empty array", () => {
    const arr: record.KeyedNamed[] = [];
    const result = array.removeKeyed<string, record.KeyedNamed>(arr, "1");
    expect(result).toHaveLength(0);
    expect(result).toEqual([]);
  });

  it("should handle empty keys to delete", () => {
    const arr: record.KeyedNamed[] = [
      { key: "1", name: "First" },
      { key: "2", name: "Second" },
    ];
    const result = array.removeKeyed<string, record.KeyedNamed>(arr, []);
    expect(result).toEqual(arr);
  });

  it("should delete all items when all keys match", () => {
    const arr: record.KeyedNamed[] = [
      { key: "1", name: "First" },
      { key: "2", name: "Second" },
    ];
    const result = array.removeKeyed<string, record.KeyedNamed>(arr, ["1", "2"]);
    expect(result).toHaveLength(0);
    expect(result).toEqual([]);
  });
});
