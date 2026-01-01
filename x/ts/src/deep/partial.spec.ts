// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { type deep } from "@/deep";

describe("Partial", () => {
  it("should make all properties optional at first level", () => {
    interface Test {
      a: string;
      b: number;
    }
    type PartialTest = deep.Partial<Test>;
    const obj: PartialTest = {};
    expect(obj).toEqual({});
  });

  it("should make nested object properties optional", () => {
    interface Test {
      a: {
        b: string;
        c: number;
      };
    }
    type PartialTest = deep.Partial<Test>;
    const obj1: PartialTest = {};
    const obj2: PartialTest = { a: {} };
    const obj3: PartialTest = { a: { b: "test" } };

    expect(obj1).toEqual({});
    expect(obj2).toEqual({ a: {} });
    expect(obj3).toEqual({ a: { b: "test" } });
  });

  it("should handle deeply nested objects", () => {
    interface Test {
      a: {
        b: {
          c: {
            d: string;
          };
        };
      };
    }
    type PartialTest = deep.Partial<Test>;
    const obj1: PartialTest = {};
    const obj2: PartialTest = { a: { b: {} } };
    const obj3: PartialTest = { a: { b: { c: { d: "test" } } } };

    expect(obj1).toEqual({});
    expect(obj2).toEqual({ a: { b: {} } });
    expect(obj3).toEqual({ a: { b: { c: { d: "test" } } } });
  });

  it("should handle arrays", () => {
    interface Test {
      items: string[];
      nested: {
        values: number[];
      };
    }
    type PartialTest = deep.Partial<Test>;
    const obj1: PartialTest = {};
    const obj2: PartialTest = { items: ["a", "b"] };
    const obj3: PartialTest = { nested: { values: [1, 2, 3] } };

    expect(obj1).toEqual({});
    expect(obj2).toEqual({ items: ["a", "b"] });
    expect(obj3).toEqual({ nested: { values: [1, 2, 3] } });
  });

  it("should handle primitive types", () => {
    type StringPartial = deep.Partial<string>;
    type NumberPartial = deep.Partial<number>;
    type BooleanPartial = deep.Partial<boolean>;

    const str: StringPartial = "test";
    const num: NumberPartial = 42;
    const bool: BooleanPartial = true;

    expect(str).toBe("test");
    expect(num).toBe(42);
    expect(bool).toBe(true);
  });

  it("should handle union types", () => {
    interface Test {
      value: string | number;
      nested: {
        prop: boolean | null;
      };
    }
    type PartialTest = deep.Partial<Test>;
    const obj1: PartialTest = {};
    const obj2: PartialTest = { value: "test" };
    const obj3: PartialTest = { value: 42 };
    const obj4: PartialTest = { nested: { prop: null } };

    expect(obj1).toEqual({});
    expect(obj2).toEqual({ value: "test" });
    expect(obj3).toEqual({ value: 42 });
    expect(obj4).toEqual({ nested: { prop: null } });
  });

  it("should handle optional properties", () => {
    interface Test {
      required: string;
      optional?: number;
      nested: {
        req: boolean;
        opt?: string;
      };
    }
    type PartialTest = deep.Partial<Test>;
    const obj: PartialTest = {
      nested: {
        req: true,
      },
    };

    expect(obj).toEqual({ nested: { req: true } });
  });

  it("should handle Record types", () => {
    interface Test {
      data: Record<string, number>;
    }
    type PartialTest = deep.Partial<Test>;
    const obj1: PartialTest = {};
    const obj2: PartialTest = { data: { a: 1, b: 2 } };

    expect(obj1).toEqual({});
    expect(obj2).toEqual({ data: { a: 1, b: 2 } });
  });

  it("should handle tuples", () => {
    interface Test {
      tuple: [string, number, boolean];
    }
    type PartialTest = deep.Partial<Test>;
    const obj1: PartialTest = {};
    const obj2: PartialTest = { tuple: ["a", 1, true] };

    expect(obj1).toEqual({});
    expect(obj2).toEqual({ tuple: ["a", 1, true] });
  });

  it("should compile with complex nested structures", () => {
    interface Complex {
      id: string;
      metadata: {
        created: Date;
        modified: Date;
        tags: string[];
      };
      content: {
        title: string;
        body: {
          paragraphs: Array<{
            text: string;
            style?: {
              bold?: boolean;
              italic?: boolean;
            };
          }>;
        };
      };
    }

    type PartialComplex = deep.Partial<Complex>;
    const obj: PartialComplex = {
      content: {
        body: {
          paragraphs: [{ text: "test" }],
        },
      },
    };

    expect(obj).toEqual({
      content: {
        body: {
          paragraphs: [{ text: "test" }],
        },
      },
    });
  });
});
