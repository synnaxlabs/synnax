// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { z } from "zod";

import { payload } from "@/payload";

describe("payload.Float", () => {
  describe("decode", () => {
    it.each([
      [1.5, 1.5],
      ["NaN", NaN],
      ["Infinity", Infinity],
      ["-Infinity", -Infinity],
    ])("should decode %s", (input, expected) => {
      expect(payload.floatZ.parse(input)).toBe(expected);
    });

    it.each(["inf", "1.5", true, null])("should reject %s", (input) => {
      expect(payload.floatZ.safeParse(input).success).toBe(false);
    });
  });

  describe("encode", () => {
    it.each([
      [1.5, 1.5],
      [NaN, "NaN"],
      [Infinity, "Infinity"],
      [-Infinity, "-Infinity"],
    ])("should encode %s", (input, expected) => {
      expect(z.encode(payload.floatZ, input)).toBe(expected);
    });

    it("should encode a non-finite fill inside a read", () => {
      const read = payload.floatReadZ.parse({ enumValues: [], fill: "-Infinity" });
      expect(read.fill).toBe(-Infinity);
      expect(z.encode(payload.floatReadZ, read)).toEqual({
        enumValues: [],
        fill: "-Infinity",
      });
    });
  });
});
