// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { id } from "@/id";

describe("id", () => {
  describe("generate", () => {
    it("should generate a string", () => {
      const newID = id.generate();
      expect(typeof newID).toBe("string");
    });

    it("should generate a string of length 11", () => {
      const newID = id.generate();
      expect(newID.length).toBe(11);
    });

    it("should only contain alphanumeric characters", () => {
      const newID = id.generate();
      expect(newID).toMatch(/^[0-9A-Za-z]+$/);
    });

    it("should generate unique IDs", () => {
      const ids = new Set<string>();
      for (let i = 0; i < 100; i++) {
        const newID = id.generate();
        expect(ids.has(newID)).toBe(false);
        ids.add(newID);
      }
    });
  });

  describe("validate", () => {
    it("should return true for valid IDs", () => {
      const validID = "abc123DEF45";
      expect(id.validate(validID)).toBe(true);
    });

    it("should return false for IDs with invalid length", () => {
      expect(id.validate("abc")).toBe(false);
      expect(id.validate("abcdefghijkl")).toBe(false);
    });

    it("should return false for IDs with non-alphanumeric characters", () => {
      expect(id.validate("abc!@#defghi")).toBe(false);
      expect(id.validate("abc def ghi")).toBe(false);
    });

    it("should return false for empty string", () => {
      expect(id.validate("")).toBe(false);
    });

    it("should accept strings with only numbers", () => {
      expect(id.validate("12345678901")).toBe(true);
    });

    it("should accept strings with only lowercase letters", () => {
      expect(id.validate("abcdefghijk")).toBe(true);
    });

    it("should accept strings with only uppercase letters", () => {
      expect(id.validate("ABCDEFGHIJK")).toBe(true);
    });

    it("should return false for strings with whitespace padding", () => {
      expect(id.validate(" ABCDEFGHIJ ")).toBe(false);
      expect(id.validate("\tABCDEFGHIJ")).toBe(false);
      expect(id.validate("ABCDEFGHIJ\n")).toBe(false);
    });
  });

  describe("schema", () => {
    it("should validate valid IDs", () => {
      const validID = id.generate();
      expect(id.schema.safeParse(validID).success).toBe(true);
    });

    it("should reject invalid IDs", () => {
      const invalidID = "invalid-id";
      expect(id.schema.safeParse(invalidID).success).toBe(false);
    });
  });
});
