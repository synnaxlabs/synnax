// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { labelZ, type Where } from "@/label/types.gen";
import { createLines, forIDs } from "@/label/where";
import { matches } from "@/label/where.gen";
import { type ontology } from "@/ontology";

const PUMP: ontology.ID = { type: "range", key: "pump" };
const VALVE: ontology.ID = { type: "range", key: "valve" };

describe("forIDs", () => {
  it.each<[string, Where, ontology.ID[]]>([
    ["no for line", { name: { eq: "Pressurized" } }, []],
    ["a for line", { for: PUMP }, [PUMP]],
    [
      "for lines under and and or",
      { and: [{ for: PUMP }], or: [{ for: VALVE }] },
      [PUMP, VALVE],
    ],
    ["a for line under not", { not: { for: PUMP } }, [PUMP]],
  ])("should return the resources of %s", (_, where, expected) => {
    expect(forIDs(where)).toEqual(expected);
  });
});

describe("createLines", () => {
  it("should match a for line through the labeled reader", () => {
    const l = labelZ.parse({ name: "Pressurized", color: "#000000" });
    const lines = createLines((id, key) => id.key === PUMP.key && key === l.key);
    expect(matches(lines, l, { for: PUMP })).toBe(true);
    expect(matches(lines, l, { for: VALVE })).toBe(false);
  });
});
