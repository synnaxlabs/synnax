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

import { json } from "@/json";

describe("json", () => {
  describe("pointerZ", () => {
    it("should accept an empty string", () => {
      expect(json.pointerZ.parse("")).toBe("");
    });

    it("should accept a simple pointer", () => {
      expect(json.pointerZ.parse("/foo")).toBe("/foo");
    });

    it("should accept a nested pointer", () => {
      expect(json.pointerZ.parse("/foo/bar/baz")).toBe("/foo/bar/baz");
    });

    it("should accept a pointer with array indices", () => {
      expect(json.pointerZ.parse("/items/0/name")).toBe("/items/0/name");
    });

    it("should accept a pointer with escaped tilde", () => {
      expect(json.pointerZ.parse("/a~0b")).toBe("/a~0b");
    });

    it("should accept a pointer with escaped slash", () => {
      expect(json.pointerZ.parse("/a~1b")).toBe("/a~1b");
    });

    it("should reject a pointer that does not start with /", () => {
      expect(z.validate(json.pointerZ, "foo")).toBe(false);
    });

    it("should reject a pointer with a bare tilde", () => {
      expect(z.validate(json.pointerZ, "/a~b")).toBe(false);
    });
  });

  describe("primitiveZ", () => {
    it("should accept a string", () => {
      expect(json.primitiveZ.parse("hello")).toBe("hello");
    });

    it("should accept a number", () => {
      expect(json.primitiveZ.parse(42)).toBe(42);
    });

    it("should accept a boolean", () => {
      expect(json.primitiveZ.parse(true)).toBe(true);
    });

    it("should accept null", () => {
      expect(json.primitiveZ.parse(null)).toBeNull();
    });

    it("should reject an object", () => {
      expect(z.validate(json.primitiveZ, {})).toBe(false);
    });

    it("should reject an array", () => {
      expect(z.validate(json.primitiveZ, [])).toBe(false);
    });
  });

  describe("primitiveTypeZ", () => {
    it("should accept 'string'", () => {
      expect(json.primitiveTypeZ.parse("string")).toBe("string");
    });

    it("should accept 'number'", () => {
      expect(json.primitiveTypeZ.parse("number")).toBe("number");
    });

    it("should accept 'boolean'", () => {
      expect(json.primitiveTypeZ.parse("boolean")).toBe("boolean");
    });

    it("should accept 'null'", () => {
      expect(json.primitiveTypeZ.parse("null")).toBe("null");
    });

    it("should reject an invalid type name", () => {
      expect(z.validate(json.primitiveTypeZ, "object")).toBe(false);
    });
  });
});
