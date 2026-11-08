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

import { zod } from "@/zod";

describe("length", () => {
  describe("minLength", () => {
    const schema = z.string().check(zod.minLength(4));

    it.each([
      ["ASCII letters", "abcd"],
      ["accented letters", "éééé"],
      ["emoji", "😀😀😀😀"],
    ])("should accept %s at the minimum", (_, value) => {
      expect(schema.parse(value)).toEqual(value);
    });

    it.each([
      ["ASCII letters", "abc"],
      ["accented letters", "ééé"],
      ["emoji", "😀😀😀"],
    ])("should reject %s below the minimum", (_, value) => {
      const result = schema.safeParse(value);
      expect(result.error?.issues[0].message).toEqual(
        "must be at least 4 characters long",
      );
    });
  });

  describe("maxLength", () => {
    const schema = z.string().check(zod.maxLength(4));

    it.each([
      ["ASCII letters", "abcd"],
      ["accented letters", "éééé"],
      ["emoji", "😀😀😀😀"],
    ])("should accept %s at the maximum", (_, value) => {
      expect(schema.parse(value)).toEqual(value);
    });

    it.each([
      ["ASCII letters", "abcde"],
      ["accented letters", "ééééé"],
      ["emoji", "😀😀😀😀😀"],
      ["a family emoji, which is five code points", "👨‍👩‍👧"],
    ])("should reject %s above the maximum", (_, value) => {
      const result = schema.safeParse(value);
      expect(result.error?.issues[0].message).toEqual(
        "must be at most 4 characters long",
      );
    });

    it("should stay chainable with string methods and a default", () => {
      const chained = z.string().check(zod.maxLength(4)).regex(/^a/).default("abc");
      expect(chained.parse(undefined)).toEqual("abc");
      expect(chained.safeParse("abcde").success).toBe(false);
      expect(chained.safeParse("bcd").success).toBe(false);
    });
  });
});
