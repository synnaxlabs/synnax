// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { json } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { ICD } from "@/platform/icd";

interface Item {
  name: string;
  shape: json.Shape;
}

const createItem = (name: string, count: number): Item => ({
  name,
  shape: json.shapeZ.parse({
    sources: [name],
    fields: [],
    index: -1,
    present: -1,
    count,
  }),
});

const shapeOf = (item: Item): json.Shape => item.shape;

describe("ICD.Write.mostCommon", () => {
  it("should return the item whose shape holds the most documents", () => {
    const items = [createItem("a", 1), createItem("b", 3), createItem("c", 2)];
    expect(ICD.Write.mostCommon(items, shapeOf)).toBe(items[1]);
  });

  it("should return the first item of a tie", () => {
    const items = [createItem("a", 2), createItem("b", 2)];
    expect(ICD.Write.mostCommon(items, shapeOf)).toBe(items[0]);
  });

  it("should return null for no items", () => {
    expect(ICD.Write.mostCommon([], shapeOf)).toBeNull();
  });
});
