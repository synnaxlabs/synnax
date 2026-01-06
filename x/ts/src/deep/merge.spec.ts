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
});
