// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, test } from "vitest";
import { z } from "zod";

import { binary } from "@/binary";
import { primitive } from "@/primitive";
import { testutil } from "@/testutil";

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
      test(`isZero should return ${expected} for ${testutil.toString(value)}`, () => {
        expect(primitive.isZero(value)).toEqual(expected);
      });
      test(`isNonZero should return ${!expected} for ${testutil.toString(value)}`, () => {
        expect(primitive.isNonZero(value)).toEqual(!expected);
      });
    });
  });

  describe("isStringer", () => {
    it("should return true for a stringer", () => {
      expect(primitive.isStringer(new ExampleStringer("cat"))).toEqual(true);
    });
    it("should return false for a non-stringer", () => {
      expect(primitive.isStringer(0)).toEqual(false);
    });
    it("should return false for null", () => {
      expect(primitive.isStringer(null)).toEqual(false);
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

  describe("ValueExtension", () => {
    class MyValueExtension extends primitive.ValueExtension<bigint> {
      valueOf(): bigint {
        return this.value;
      }
      toJSON(): bigint {
        return this.value;
      }
    }

    describe("valueOf", () => {
      it("should return the value", () => {
        const value = new MyValueExtension(12n);
        expect(value.valueOf()).toEqual(12n);
      });
    });

    describe("toJSON", () => {
      it("should return the value", () => {
        const value = new MyValueExtension(12n);
        expect(value.toJSON()).toEqual(12n);
      });
    });

    describe("toString", () => {
      it("should return the value", () => {
        const value = new MyValueExtension(12n);
        expect(value.toString()).toEqual("12");
      });
    });

    describe("encode + decode", () => {
      it("should encode + decode a value extension", () => {
        const v = {
          myDog: new MyValueExtension(12n),
        };
        const encoded = binary.JSON_CODEC.encode(v);
        const decoded = binary.JSON_CODEC.decode(encoded);
        expect(decoded).toEqual({
          myDog: "12",
        });
      });

      it("should encode + decode a value extension with a schema", () => {
        const v = {
          myDog: new MyValueExtension(12n),
        };
        const schema = z.object({
          myDog: z
            .bigint()
            .or(z.string())
            .transform((v) => new MyValueExtension(BigInt(v))),
        });
        const encoded = binary.JSON_CODEC.encode(v);
        const decoded = binary.JSON_CODEC.decode(encoded, schema);
        expect(decoded).toEqual({
          myDog: new MyValueExtension(12n),
        });
        expect(decoded.myDog).toBeInstanceOf(MyValueExtension);
      });

      describe("isCrudeValueExtension", () => {
        it("should return true for a CrudeValueExtension", () => {
          expect(primitive.isCrudeValueExtension({ value: 12n })).toEqual(true);
        });
        it("should return false for a non-CrudeValueExtension", () => {
          expect(primitive.isCrudeValueExtension(12n)).toEqual(false);
        });
      });
    });
  });
});
