// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { filter } from "@/filter";
import { TimeStamp } from "@/telem";

describe("filter.describe", () => {
  it("should describe each operator", () => {
    expect(filter.describe({ name: { eq: "press" } })).toBe('name = "press"');
    expect(filter.describe({ key: { in: [12, 14] } })).toBe("key in [12, 14]");
    expect(filter.describe({ name: { contains: "a" } })).toBe('name contains "a"');
    expect(filter.describe({ name: { startsWith: "a" } })).toBe('name starts with "a"');
    expect(filter.describe({ name: { regex: "pre.*" } })).toBe("name matches /pre.*/");
    expect(filter.describe({ count: { gt: 3 } })).toBe("count > 3");
    expect(filter.describe({ count: { lte: 3 } })).toBe("count <= 3");
    expect(filter.describe({ count: { gte: 3, lt: 10 } })).toBe(
      "count >= 3 and count < 10",
    );
    expect(filter.describe({ tags: { any: ["a", "b"] } })).toBe(
      'tags has any of ["a", "b"]',
    );
    expect(filter.describe({ active: { eq: true } })).toBe("active = true");
  });

  it("should mark each operator of a case-insensitive condition", () => {
    expect(filter.describe({ name: { eq: "Press", caseInsensitive: true } })).toBe(
      'name = "Press" (ignoring case)',
    );
    expect(
      filter.describe({
        name: { contains: "a", startsWith: "b", caseInsensitive: true },
      }),
    ).toBe(
      'name contains "a" (ignoring case) and name starts with "b" (ignoring case)',
    );
    expect(filter.describe({ name: { eq: "a", caseInsensitive: false } })).toBe(
      'name = "a"',
    );
  });

  it("should describe a value that is not an operator object", () => {
    expect(filter.describe({ virtual: true })).toBe("virtual = true");
    expect(filter.describe({ labels: ["a", "b"] })).toBe('labels = ["a", "b"]');
    expect(filter.describe({ span: { start: 1, end: 2 } })).toBe(
      'span = {"end":2,"start":1}',
    );
  });

  it("should describe a value by its JSON form", () => {
    expect(filter.describe({ count: { gte: new TimeStamp(5n) } })).toBe("count >= 5");
  });

  it("should escape a string", () => {
    expect(filter.describe({ name: { eq: 'a"b' } })).toBe('name = "a\\"b"');
  });

  it("should sort the fields of a node by name", () => {
    expect(
      filter.describe({ name: { eq: "a" }, active: { eq: true }, key: { eq: 1 } }),
    ).toBe('active = true and key = 1 and name = "a"');
  });

  it("should join the children of and", () => {
    expect(filter.describe({ and: [{ name: { eq: "a" } }, { key: { eq: 1 } }] })).toBe(
      'name = "a" and key = 1',
    );
  });

  it("should join the children of or", () => {
    expect(
      filter.describe({
        or: [{ name: { eq: "a" } }, { not: { active: { eq: true } } }],
      }),
    ).toBe('name = "a" or not (active = true)');
  });

  it("should wrap an or in parentheses beside other terms", () => {
    expect(
      filter.describe({
        key: { eq: 1 },
        or: [{ name: { eq: "a" } }, { name: { eq: "b" } }],
      }),
    ).toBe('key = 1 and (name = "a" or name = "b")');
  });

  it("should keep the grouping of a child of a single-child or", () => {
    expect(
      filter.describe({
        key: { eq: 1 },
        or: [{ or: [{ name: { eq: "a" } }, { name: { eq: "b" } }] }],
      }),
    ).toBe('key = 1 and (name = "a" or name = "b")');
  });

  it("should wrap a child of or that has more than one term", () => {
    expect(
      filter.describe({
        or: [{ key: { eq: 1 }, name: { eq: "a" } }, { key: { eq: 2 } }],
      }),
    ).toBe('(key = 1 and name = "a") or key = 2');
  });

  it("should describe not", () => {
    expect(filter.describe({ not: { key: { eq: 1 }, name: { eq: "a" } } })).toBe(
      'not (key = 1 and name = "a")',
    );
  });

  it("should describe an empty tree as an empty string", () => {
    expect(filter.describe({})).toBe("");
    expect(filter.describe({ and: [] })).toBe("");
    expect(filter.describe({ name: {} })).toBe("");
  });

  it("should describe nested empty trees", () => {
    expect(filter.describe({ or: [{}, { key: { eq: 1 } }] })).toBe("true or key = 1");
    expect(filter.describe({ or: [{ and: [] }, { key: { eq: 1 } }] })).toBe(
      "true or key = 1",
    );
    expect(filter.describe({ or: [] })).toBe("false");
    expect(filter.describe({ or: [{}] })).toBe("");
    expect(filter.describe({ not: {} })).toBe("not (true)");
  });
});
