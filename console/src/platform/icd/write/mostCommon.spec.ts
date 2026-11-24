// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { ICD } from "@/platform/icd";

interface Item {
  name: string;
  count: number;
}

const countOf = (item: Item): number => item.count;

describe("ICD.Write.mostCommon", () => {
  it("should return the item with the largest count", () => {
    const items = [
      { name: "a", count: 1 },
      { name: "b", count: 3 },
      { name: "c", count: 2 },
    ];
    expect(ICD.Write.mostCommon(items, countOf)).toBe(items[1]);
  });

  it("should return the first item of a tie", () => {
    const items = [
      { name: "a", count: 2 },
      { name: "b", count: 2 },
    ];
    expect(ICD.Write.mostCommon(items, countOf)).toBe(items[0]);
  });

  it("should return null for no items", () => {
    expect(ICD.Write.mostCommon<Item>([], countOf)).toBeNull();
  });
});
