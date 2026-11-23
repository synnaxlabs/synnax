// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, test } from "vitest";

import { primitive } from "@/primitive";
import { TimeStamp } from "@/telem";
import { toString } from "@/testutil";

class ExampleStringer implements primitive.Stringer {
  readonly value: string;
  constructor(value: string) {
    this.value = value;
  }

  toString(): string {
    return this.value;
  }
}

describe("primitive", () => {
  describe("isZero and isNonZero", () => {
    interface Spec {
      value: primitive.Value;
      expected: boolean;
    }
    const SPECS: Spec[] = [
      { value: 0, expected: true },
      { value: 1, expected: false },
      { value: 0n, expected: true },
      { value: 12n, expected: false },
      { value: true, expected: false },
      { value: false, expected: true },
      { value: undefined, expected: true },
      { value: new ExampleStringer(""), expected: true },
      { value: new ExampleStringer("cat"), expected: false },
    ];
    SPECS.forEach(({ value, expected }) => {
      test(`isZero should return ${expected} for ${toString(value)}`, () => {
        expect(primitive.isZero(value)).toEqual(expected);
      });
      test(`isNonZero should return ${!expected} for ${toString(value)}`, () => {
        expect(primitive.isNonZero(value)).toEqual(!expected);
      });
    });
  });

  describe("isStringer", () => {
    it("should return true for a stringer", () => {
      expect(primitive.isStringer(new ExampleStringer("cat"))).toBe(true);
    });
    it("should return false for a non-stringer", () => {
      expect(primitive.isStringer(0)).toBe(false);
    });
    it("should return false for null", () => {
      expect(primitive.isStringer(null)).toBe(false);
    });
    it("should return false for a plain object", () => {
      expect(primitive.isStringer({})).toBe(false);
    });
    it("should return false for an object without a prototype", () => {
      expect(primitive.isStringer(Object.create(null))).toBe(false);
    });
    it("should return false for arrays and typed arrays", () => {
      expect(primitive.isStringer([])).toBe(false);
      expect(primitive.isStringer(new Float32Array([1]))).toBe(false);
    });
    it("should return false for a Map and a Set", () => {
      expect(primitive.isStringer(new Map())).toBe(false);
      expect(primitive.isStringer(new Set())).toBe(false);
    });
    it("should return true for a TimeStamp", () => {
      expect(primitive.isStringer(TimeStamp.now())).toBe(true);
    });
  });

  describe("is", () => {
    it("should return true for null and undefined", () => {
      expect(primitive.is(null)).toBe(true);
      expect(primitive.is(undefined)).toBe(true);
    });

    it("should return true for strings", () => {
      expect(primitive.is("")).toBe(true);
      expect(primitive.is("hello")).toBe(true);
    });

    it("should return true for numbers", () => {
      expect(primitive.is(0)).toBe(true);
      expect(primitive.is(3.14)).toBe(true);
      expect(primitive.is(NaN)).toBe(true);
    });

    it("should return true for booleans", () => {
      expect(primitive.is(true)).toBe(true);
      expect(primitive.is(false)).toBe(true);
    });

    it("should return true for bigints", () => {
      expect(primitive.is(42n)).toBe(true);
    });

    it("should return true for symbols", () => {
      expect(primitive.is(Symbol("x"))).toBe(true);
    });

    it("should return false for plain objects", () => {
      expect(primitive.is({})).toBe(false);
      expect(primitive.is({ a: 1 })).toBe(false);
    });

    it("should return false for arrays", () => {
      expect(primitive.is([])).toBe(false);
      expect(primitive.is([1, 2, 3])).toBe(false);
    });

    it("should return false for class instances", () => {
      expect(primitive.is(new Date())).toBe(false);
      expect(primitive.is(new Map())).toBe(false);
      expect(primitive.is(new Error())).toBe(false);
    });

    it("should return false for functions", () => {
      expect(primitive.is(() => 1)).toBe(false);
    });
  });

  describe("isHashable", () => {
    class HashableThing implements primitive.Hashable {
      constructor(private readonly v: string) {}
      hash(): string {
        return this.v;
      }
    }

    it("returns true for an object with a hash() method", () => {
      expect(primitive.isHashable(new HashableThing("x"))).toBe(true);
    });

    it("returns true for a plain object literal with a hash function", () => {
      expect(primitive.isHashable({ hash: () => "x" })).toBe(true);
    });

    it("narrows the type for downstream calls", () => {
      const v: unknown = new HashableThing("abc");
      if (primitive.isHashable(v)) expect(v.hash()).toEqual("abc");
      else throw new Error("expected isHashable to narrow");
    });

    it("returns false for null", () => {
      expect(primitive.isHashable(null)).toBe(false);
    });

    it("returns false for undefined", () => {
      expect(primitive.isHashable(undefined)).toBe(false);
    });

    it("returns false for primitives", () => {
      expect(primitive.isHashable("x")).toBe(false);
      expect(primitive.isHashable(42)).toBe(false);
      expect(primitive.isHashable(42n)).toBe(false);
      expect(primitive.isHashable(true)).toBe(false);
    });

    it("returns false for plain objects without a hash function", () => {
      expect(primitive.isHashable({})).toBe(false);
      expect(primitive.isHashable({ key: "x" })).toBe(false);
    });

    it("returns false when hash is not a function", () => {
      expect(primitive.isHashable({ hash: "not a function" })).toBe(false);
      expect(primitive.isHashable({ hash: 42 })).toBe(false);
      expect(primitive.isHashable({ hash: null })).toBe(false);
    });

    it("returns false for arrays", () => {
      expect(primitive.isHashable([1, 2, 3])).toBe(false);
    });
  });
});
