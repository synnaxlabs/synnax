// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { compile, find, invalid } from "@/log/aether/search";

describe("log/aether/search", () => {
  describe("compile", () => {
    it("should return null for empty text", () => {
      expect(compile({ text: "", mode: "text" })).toBeNull();
      expect(compile({ text: "", mode: "regex" })).toBeNull();
    });

    it("should match plain text in any case", () => {
      expect(compile({ text: "err", mode: "text" })?.test("An ERROR")).toBe(true);
    });

    it("should match the special characters of plain text as written", () => {
      const pattern = compile({ text: "a.b(", mode: "text" });
      expect(pattern?.test("a.b(")).toBe(true);
      expect(pattern?.test("axb(")).toBe(false);
    });

    it("should match a regex as written", () => {
      const pattern = compile({ text: "^err", mode: "regex" });
      expect(pattern?.test("err 1")).toBe(true);
      expect(pattern?.test("ERR 1")).toBe(false);
    });

    it("should return null for an invalid regex", () => {
      expect(compile({ text: "(", mode: "regex" })).toBeNull();
    });
  });

  describe("invalid", () => {
    it("should be true only for a regex that does not compile", () => {
      expect(invalid({ text: "(", mode: "regex" })).toBe(true);
      expect(invalid({ text: "(", mode: "text" })).toBe(false);
      expect(invalid({ text: "", mode: "regex" })).toBe(false);
    });
  });

  describe("find", () => {
    it("should return the start and end of each match", () => {
      const pattern = compile({ text: "\\d+", mode: "regex" }, "g");
      expect(find(pattern!, "a1 b22 c333")).toEqual([
        [1, 2],
        [4, 6],
        [8, 11],
      ]);
    });

    it("should count an empty match", () => {
      const pattern = compile({ text: "^(?!DEBUG)", mode: "regex" }, "g");
      expect(find(pattern!, "INFO up")).toEqual([[0, 0]]);
      expect(find(pattern!, "DEBUG up")).toEqual([]);
    });
  });
});
