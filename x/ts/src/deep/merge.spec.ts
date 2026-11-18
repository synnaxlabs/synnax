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

import { deep } from "@/deep";

interface TestRecord {
  a: number;
  b?: {
    c?: number;
    d?: number;
  };
}

describe("deepMerge", () => {
  it("should deep two objects", () => {
    const a: TestRecord = {
      a: 1,
      b: {
        c: 2,
      },
    };
    const b: TestRecord = {
      a: 3,
      b: {
        d: 4,
      },
    };
    const c: TestRecord = {
      a: 3,
      b: {
        c: 2,
        d: 4,
      },
    };
    expect(deep.override(a, b)).toEqual(c);
  });
  it("Should set a value even when its parent is undefined", () => {
    const a: TestRecord = {
      a: 1,
    };
    const b: TestRecord = {
      a: 3,
      b: {
        d: 4,
      },
    };
    const c: TestRecord = {
      a: 3,
      b: {
        d: 4,
      },
    };
    expect(deep.override(a, b)).toEqual(c);
  });

  describe("overrideValidItems", () => {
    it("should override valid items", () => {
      const base = {
        a: 1,
        b: 2,
      };
      const override = {
        a: 3,
      };
      const schema = z.object({
        a: z.number(),
        b: z.number(),
      });
      expect(deep.overrideValidItems(base, override, schema)).toEqual({
        a: 3,
        b: 2,
      });
    });
    it("should ignore invalid items", () => {
      const base = {
        a: 1,
        b: 2,
      };
      const override = {
        a: "3",
      };
      const schema = z.object({
        a: z.number(),
        b: z.number(),
      });
      expect(deep.overrideValidItems(base, override, schema)).toEqual({
        a: 1,
        b: 2,
      });
    });
    it("should merge deeply nested objects", () => {
      const base = {
        a: 1,
        b: {
          c: 2,
        },
      };
      const override = {
        a: 3,
        b: {
          c: 4,
        },
      };
      const schema = z.object({
        a: z.number(),
        b: z.object({
          c: z.number(),
        }),
      });
      expect(deep.overrideValidItems(base, override, schema)).toEqual({
        a: 3,
        b: {
          c: 4,
        },
      });
    });
    it("should ignore invalid nested objects", () => {
      const base = {
        a: 1,
        b: {
          c: 2,
        },
      };
      const override = {
        a: 3,
        b: {
          c: "4",
        },
      };
      const schema = z.object({
        a: z.number(),
        b: z.object({
          c: z.number(),
        }),
      });
      expect(deep.overrideValidItems(base, override, schema)).toEqual({
        a: 3,
        b: {
          c: 2,
        },
      });
    });
    it("should ignore nested objects that don't exist in the schema", () => {
      const base = {
        a: 1,
        b: {
          c: 2,
        },
      };
      const override = {
        a: 3,
        f: {
          d: 4,
        },
      };
      const schema = z.object({
        a: z.number(),
        b: z.object({
          c: z.number(),
        }),
      });
      expect(deep.overrideValidItems(base, override, schema)).toEqual({
        a: 3,
        b: {
          c: 2,
        },
      });
    });
  });
  it("should work correctly with refinements", () => {
    const base = {
      a: 1,
      b: 2,
    };
    const override = {
      a: 3,
    };
    const schema = z
      .object({
        a: z.number(),
        b: z.number().refine((v) => v > 1),
      })
      .refine((v) => v.a > 0, {
        path: ["a"],
        message: "a must be greater than 0",
      })
      .refine((v) => v.b > 0, {
        path: ["b"],
        message: "b must be greater than 0",
      });
    expect(deep.overrideValidItems(base, override, schema)).toEqual({
      a: 3,
      b: 2,
    });
  });

  it("should work with zod unions", () => {
    const schema1 = z.object({
      a: z.number(),
    });
    const schema2 = z.object({
      b: z.number(),
    });
    const schema = z.union([schema1, schema2]);
    const base = {
      a: 1,
    };
    const override = {
      b: 2,
    };
    expect(deep.overrideValidItems(base, override, schema)).toEqual({
      a: 1,
      b: 2,
    });
  });

  it("should work with extensions", () => {
    const schema = z.object({
      a: z.number(),
    });
    const extendedSchema = schema.extend({
      b: z.string(),
    });
    const base = {
      a: 1,
    };
    const override = {
      b: "2",
    };
    expect(deep.overrideValidItems(base, override, extendedSchema)).toEqual({
      a: 1,
      b: "2",
    });
  });

  it("should work with intersection", () => {
    const schema1 = z.object({
      a: z.number(),
    });
    const schema2 = z.object({
      b: z.string(),
      c: z.number(),
    });
    const schema = schema1.and(schema2);
    const base = {};
    const override = {
      b: "2",
      c: 3,
    };
    expect(deep.overrideValidItems(base, override, schema)).toEqual({
      b: "2",
      c: 3,
    });
  });

  describe("override without mutation", () => {
    interface Nested {
      a: number;
      b: { c: number; d: { e: number; f?: number } };
      list: number[];
      g?: { h: number };
    }
    const createBase = (): Nested => ({
      a: 1,
      b: { c: 2, d: { e: 3 } },
      list: [1, 2],
    });

    it("should leave the base and its nested objects unchanged", () => {
      const base = createBase();
      const { b } = base;
      const { d } = b;
      const snapshot = deep.copy(base);
      const out = deep.override(base, { b: { d: { f: 4 } } });
      expect(base).toEqual(snapshot);
      expect(base.b).toBe(b);
      expect(base.b.d).toBe(d);
      expect(out).toEqual({ a: 1, b: { c: 2, d: { e: 3, f: 4 } }, list: [1, 2] });
    });

    it("should give each merged object a new identity and share the rest", () => {
      const base = createBase();
      const out = deep.override(base, { b: { c: 5 } });
      expect(out).not.toBe(base);
      expect(out.b).not.toBe(base.b);
      expect(out.b.d).toBe(base.b.d);
      expect(out.list).toBe(base.list);
    });

    it("should merge into a frozen base", () => {
      const base = deep.freeze(createBase());
      expect(deep.override(base, { a: 5, b: { d: { e: 6 } } })).toEqual({
        a: 5,
        b: { c: 2, d: { e: 6 } },
        list: [1, 2],
      });
    });

    it("should leave the overrides unchanged", () => {
      const first = { b: { c: 5 } };
      const second = { b: { d: { e: 6 } } };
      deep.override(createBase(), first, second);
      expect(first).toEqual({ b: { c: 5 } });
      expect(second).toEqual({ b: { d: { e: 6 } } });
    });

    it("should apply the overrides in order", () => {
      expect(deep.override(createBase(), { a: 2 }, { a: 3 }).a).toBe(3);
    });

    it("should return the base when there are no overrides", () => {
      const base = createBase();
      expect(deep.override(base)).toBe(base);
    });

    it("should replace an array whole", () => {
      expect(deep.override(createBase(), { list: [9] }).list).toEqual([9]);
    });

    it("should build a nested object the base lacks without sharing the override", () => {
      const source = { g: { h: 1 } };
      const out = deep.override(createBase(), source);
      expect(out.g).toEqual({ h: 1 });
      expect(out.g).not.toBe(source.g);
    });

    it("should drop a nested object over a value that is not an object", () => {
      const base: { a: number | null } = { a: null };
      expect(deep.override(base, { a: { b: 1 } } as never)).toEqual({ a: null });
    });
  });

  describe("overrideValidItems without mutation", () => {
    const schema = z.object({
      a: z.number(),
      b: z.object({ c: z.number(), d: z.number().optional() }),
    });

    it("should leave the base unchanged when a nested override is partly valid", () => {
      const base = { a: 1, b: { c: 2 } };
      const { b } = base;
      const snapshot = deep.copy(base);
      const out = deep.overrideValidItems(base, { b: { c: "bad", d: 5 } }, schema);
      expect(base).toEqual(snapshot);
      expect(base.b).toBe(b);
      expect(out).toEqual({ a: 1, b: { c: 2, d: 5 } });
    });

    it("should leave the base unchanged when the nested override is valid", () => {
      const base = { a: 1, b: { c: 2 } };
      const snapshot = deep.copy(base);
      const out = deep.overrideValidItems(base, { b: { c: 3 } }, schema);
      expect(base).toEqual(snapshot);
      expect(out.b).toEqual({ c: 3 });
    });

    it("should leave the override unchanged", () => {
      const override = { b: { c: "bad", d: 5 } };
      deep.overrideValidItems({ a: 1, b: { c: 2 } }, override, schema);
      expect(override).toEqual({ b: { c: "bad", d: 5 } });
    });

    it("should leave a frozen base unchanged", () => {
      const base = deep.freeze({ a: 1, b: { c: 2 } });
      expect(deep.overrideValidItems(base, { b: { c: "bad", d: 5 } }, schema)).toEqual({
        a: 1,
        b: { c: 2, d: 5 },
      });
    });

    it("should pass each union member's result to the next", () => {
      const union = z.union([z.object({ x: z.number() }), z.object({ y: z.number() })]);
      const base = { x: 0, y: 0 };
      const out = deep.overrideValidItems(base, { x: 1, y: 2 }, union);
      expect(out).toEqual({ x: 1, y: 2 });
      expect(base).toEqual({ x: 0, y: 0 });
    });

    it("should leave nested base objects unchanged across a union", () => {
      const union = z.union([
        z.object({ a: z.object({ x: z.number(), y: z.number().optional() }) }),
        z.object({ b: z.number() }),
      ]);
      const base = { a: { x: 0 } };
      const snapshot = deep.copy(base);
      const out = deep.overrideValidItems(base, { a: { x: "bad", y: 2 } }, union);
      expect(out).toEqual({ a: { x: 0, y: 2 } });
      expect(base).toEqual(snapshot);
    });

    it("should leave the base unchanged across an intersection", () => {
      const intersection = z
        .object({ a: z.object({ x: z.number() }) })
        .and(z.object({ b: z.number() }));
      const base = { a: { x: 0 }, b: 1 };
      const snapshot = deep.copy(base);
      const out = deep.overrideValidItems(base, { a: { x: 5 }, b: 2 }, intersection);
      expect(out).toEqual({ a: { x: 5 }, b: 2 });
      expect(base).toEqual(snapshot);
    });
  });
});
