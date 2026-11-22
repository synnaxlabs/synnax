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

const parseExpectingError = (
  schema: z.ZodType,
  value: unknown,
  opts?: zod.ParseOptions,
): zod.ParseError => {
  try {
    zod.parse(schema, value, opts);
    throw new Error("expected parse to throw");
  } catch (e) {
    if (!zod.ParseError.matches(e))
      throw new Error("expected ParseError", { cause: e });
    return e as zod.ParseError;
  }
};

const renderedInput = (input: unknown): unknown =>
  parseExpectingError(z.never(), input).toStatus().details.input;

const nest = (depth: number, wrap: (v: unknown) => unknown, leaf: unknown): unknown => {
  let v = leaf;
  for (let i = 0; i < depth; i++) v = wrap(v);
  return v;
};

describe("parse error input rendering", () => {
  describe("primitives", () => {
    it("should pass strings, numbers, booleans, and null through unchanged", () => {
      ["hello", "", 42, 3.14, 0, Infinity, true, false, null].forEach((v) =>
        expect(renderedInput(v)).toBe(v),
      );
      expect(renderedInput(NaN)).toBeNaN();
    });

    it("should render undefined as its bracketed tag", () => {
      expect(renderedInput(undefined)).toBe("[undefined]");
    });

    it("should render a bigint with an n suffix", () => {
      expect(renderedInput(42n)).toBe("42n");
      expect(renderedInput(0n)).toBe("0n");
    });

    it("should render functions and symbols as bracketed tags", () => {
      expect(renderedInput(() => 1)).toBe("[function]");
      expect(renderedInput(Symbol("x"))).toBe("[symbol]");
    });
  });

  describe("well-known objects", () => {
    it("should render a Date as its ISO string", () => {
      expect(renderedInput(new Date("2026-01-01T00:00:00Z"))).toBe(
        "2026-01-01T00:00:00.000Z",
      );
    });

    it("should render an Error by its message", () => {
      expect(renderedInput(new Error("boom"))).toBe("[Error: boom]");
    });

    it("should render a class instance by its object tag", () => {
      class Custom {}
      expect(renderedInput(new Custom())).toBe("[[object Object]]");
    });
  });

  describe("arrays", () => {
    it("should recurse into elements, keeping their order and type", () => {
      expect(renderedInput([])).toEqual([]);
      expect(renderedInput([1, "two", true, null])).toEqual([1, "two", true, null]);
    });

    it("should truncate an array longer than ten elements", () => {
      const input = Array.from({ length: 15 }, (_, i) => i);
      expect(renderedInput(input)).toEqual([...input.slice(0, 10), "…(+5 more)"]);
    });

    it("should not truncate an array of ten elements", () => {
      const input = Array.from({ length: 10 }, (_, i) => i);
      expect(renderedInput(input)).toEqual(input);
    });

    it("should name an array nested past the depth limit without its elements", () => {
      const wrap = (v: unknown): unknown => [v];
      expect(renderedInput(nest(8, wrap, [1, 2]))).toEqual(nest(8, wrap, "[Array(2)]"));
    });
  });

  describe("plain objects", () => {
    it("should recurse into nested values", () => {
      expect(renderedInput({})).toEqual({});
      expect(renderedInput({ a: 1, b: { c: "deep" } })).toEqual({
        a: 1,
        b: { c: "deep" },
      });
    });

    it("should name an object nested past the depth limit", () => {
      const wrap = (v: unknown): unknown => ({ a: v });
      expect(renderedInput(nest(8, wrap, { e: "x" }))).toEqual(
        nest(8, wrap, "[Object]"),
      );
    });
  });

  describe("strings", () => {
    it("should truncate a string longer than 200 characters", () => {
      expect(renderedInput("a".repeat(206))).toBe(`${"a".repeat(200)}…(+6 chars)`);
    });

    it("should not truncate a string of 200 characters", () => {
      expect(renderedInput("a".repeat(200))).toBe("a".repeat(200));
    });
  });
});

describe("parse error context rendering", () => {
  const contextOf = (context: Record<string, unknown>): string => {
    const { message } = parseExpectingError(z.string(), 1, { context });
    return message.slice(message.indexOf("  context: ") + "  context: ".length);
  };

  it("should pretty print a context that holds a structure", () => {
    expect(contextOf({ a: { b: [1, 2] } })).toBe(
      `{
  "a": {
    "b": [
      1,
      2
    ]
  }
}`,
    );
  });

  it("should render undefined in a context as its bracketed tag", () => {
    expect(contextOf({ a: undefined, b: {} })).toBe(
      `{
  "a": "[undefined]",
  "b": {}
}`,
    );
  });

  it("should tolerate a circular context", () => {
    const circular: Record<string, unknown> = {};
    circular.self = circular;
    expect(() => contextOf({ circular })).not.toThrow();
  });
});

describe("parse error path rendering", () => {
  it("should join keys with dots and render indices in brackets", () => {
    const schema = z.object({
      tasks: z.array(z.object({ status: z.object({ code: z.number() }) })),
    });
    const { message } = parseExpectingError(schema, {
      tasks: [{ status: { code: "x" } }],
    });
    expect(message).toContain("  at tasks[0].status\n");
  });

  it("should render consecutive indices as adjacent brackets", () => {
    const schema = z.object({ m: z.array(z.array(z.object({ v: z.number() }))) });
    const { message } = parseExpectingError(schema, {
      m: [[{ v: 1 }], [{ v: 1 }, { v: 1 }, { v: "x" }]],
    });
    expect(message).toContain("  at m[1][2]\n");
  });

  it("should render a leading index without a dot after it", () => {
    const schema = z.array(z.object({ inner: z.object({ v: z.number() }) }));
    const { message } = parseExpectingError(schema, [{ inner: { v: "x" } }]);
    expect(message).toContain("  at [0].inner\n");
  });
});
