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

const messageOf = (schema: z.ZodType, value: unknown): string => {
  try {
    zod.parse(schema, value);
    throw new Error("expected parse to throw");
  } catch (e) {
    if (!zod.ParseError.matches(e))
      throw new Error("expected ParseError", { cause: e });
    return (e as zod.ParseError).message;
  }
};

describe("parse error key presence", () => {
  it("should mark a missing key as missing", () => {
    expect(messageOf(z.object({ a: z.string() }), {})).toContain(
      '✗ "a": <missing>  × expected string',
    );
  });

  it("should show a key present with null as its value", () => {
    expect(messageOf(z.object({ a: z.string() }), { a: null })).toContain(
      '✗ "a": null  × expected string',
    );
  });

  it("should show a key present with undefined as its value", () => {
    const message = messageOf(z.object({ a: z.string() }), { a: undefined });
    expect(message).toContain('✗ "a": "[undefined]"  × expected string');
    expect(message).not.toContain("<missing>");
  });

  it("should find a nested key present with null", () => {
    const schema = z.object({ a: z.object({ b: z.string() }) });
    expect(messageOf(schema, { a: { b: null } })).toContain(
      '✗ "b": null  × expected string',
    );
  });

  it("should treat a key holding a literal dot as one key", () => {
    expect(messageOf(z.object({ "foo.bar": z.string() }), { "foo.bar": 42 })).toContain(
      '✗ "foo.bar": 42  × expected string',
    );
  });

  it("should walk into arrays by index", () => {
    const schema = z.object({ items: z.array(z.number()) });
    expect(messageOf(schema, { items: [10, "x", 30] })).toContain(
      '✗ "x",  × expected number',
    );
  });

  it("should fall back to a flat line when the parent is absent from the input", () => {
    const schema = z.object({
      a: z.preprocess(() => ({ b: 1 }), z.object({ b: z.string() })),
    });
    expect(messageOf(schema, {})).toContain(
      "  at a.b\n    × expected string, received {}",
    );
  });
});
