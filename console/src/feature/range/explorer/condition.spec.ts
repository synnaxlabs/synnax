// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { type Condition, decode, encode } from "@/feature/range/explorer/condition";

const A = "4a1f0f4e-2b8f-4d3c-9b8e-3f6c2a1d0e01";
const B = "4a1f0f4e-2b8f-4d3c-9b8e-3f6c2a1d0e02";

describe("condition", () => {
  describe("decode", () => {
    it.each<[string, Condition]>([
      ["any labels", { property: "labels", match: "any", keys: [A, B] }],
      ["all labels", { property: "labels", match: "all", keys: [A, B] }],
      ["no listed labels", { property: "labels", match: "none", keys: [A] }],
      ["no labels", { property: "labels", match: "empty", keys: [] }],
      ["stages", { property: "stage", stages: ["in_progress", "completed"] }],
      ["a span", { property: "time", window: TimeSpan.days(7) }],
      [
        "a time range",
        {
          property: "time",
          window: new TimeRange(TimeStamp.seconds(1), TimeStamp.seconds(2)),
        },
      ],
      ["parents", { property: "parent", keys: [A] }],
    ])("should read back the chip for %s", (_, condition) => {
      expect(decode(encode([condition]))).toEqual([condition]);
    });

    it("should read a sole labels filter from 0.61 as one chip", () => {
      expect(decode({ labels: [A] })).toEqual([
        { property: "labels", match: "any", keys: [A] },
      ]);
    });

    it("should keep the chips of a filter in order", () => {
      const conditions: Condition[] = [
        { property: "stage", stages: ["to_do"] },
        { property: "labels", match: "any", keys: [A] },
      ];
      expect(decode(encode(conditions))).toEqual(conditions);
    });

    it("should show a filter with two fields as one other chip", () => {
      const where: ranger.Where = { labels: [A], stages: ["to_do"] };
      expect(decode(where)).toEqual([{ property: "other", where }]);
    });

    it("should read no chips from an absent or empty filter", () => {
      expect(decode()).toEqual([]);
      expect(decode({})).toEqual([]);
    });
  });
});
