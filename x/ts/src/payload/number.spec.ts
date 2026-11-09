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

describe("payload", () => {
  describe("floatZ", () => {
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

    it.each([
      [1.5, 1.5],
      [NaN, "NaN"],
      [Infinity, "Infinity"],
      [-Infinity, "-Infinity"],
    ])("should encode %s", (input, expected) => {
      expect(z.encode(payload.floatZ, input)).toBe(expected);
    });

    it("should parse a decoded non-finite fill again", () => {
      const read = payload.floatReadZ.parse({ enumValues: [], fill: "-Infinity" });
      expect(payload.floatReadZ.parse(read).fill).toBe(-Infinity);
    });

    it("should encode a non-finite fill inside a read", () => {
      const read = payload.floatReadZ.parse({ enumValues: [], fill: "-Infinity" });
      expect(z.encode(payload.floatReadZ, read)).toEqual({
        enumValues: [],
        fill: "-Infinity",
      });
    });
  });

  describe("integerZ", () => {
    it.each(["0", "-9223372036854775808", "9223372036854775807"])(
      "should accept %s",
      (input) => {
        expect(payload.integerZ.parse(input)).toBe(input);
      },
    );

    it.each(["abc", "1.5", "", " 1", "+1", "9223372036854775808", 1])(
      "should reject %s",
      (input) => {
        expect(payload.integerZ.safeParse(input).success).toBe(false);
      },
    );
  });

  describe("unsignedZ", () => {
    it.each(["0", "18446744073709551615"])("should accept %s", (input) => {
      expect(payload.unsignedZ.parse(input)).toBe(input);
    });

    it.each(["-1", "-0", "+1", "18446744073709551616", "1e3"])(
      "should reject %s",
      (input) => {
        expect(payload.unsignedZ.safeParse(input).success).toBe(false);
      },
    );
  });
});
