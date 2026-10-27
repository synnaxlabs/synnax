// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { type z } from "zod";

import { filter } from "@/filter";

interface Where {
  and?: Where[];
  or?: Where[];
  not?: Where;
  key?: filter.Set<z.ZodNumber>;
  name?: filter.String;
  active?: filter.Bool;
  count?: filter.Ordered<z.ZodNumber>;
  tags?: filter.Contains<z.ZodString>;
  virtual?: boolean;
  labels?: string[];
  window?: { start: number; end: number };
}

const b = filter.builder<"thing", Where>();
const key = b.set("key");
const name = b.string("name");

describe("builder", () => {
  describe("lines", () => {
    it("should build a set condition", () => {
      expect(key.eq(1).tree).toEqual({ key: { eq: 1 } });
      expect(key.in([1, 2]).tree).toEqual({ key: { in: [1, 2] } });
    });

    it("should build a bool condition", () => {
      expect(b.bool("active").eq(true).tree).toEqual({ active: { eq: true } });
    });

    it("should build an ordered condition", () => {
      const count = b.ordered("count");
      expect(count.gt(3).tree).toEqual({ count: { gt: 3 } });
      expect(count.lte(3).tree).toEqual({ count: { lte: 3 } });
    });

    it("should build a text condition", () => {
      expect(name.eq("a").tree).toEqual({ name: { eq: "a" } });
      expect(name.in(["a"]).tree).toEqual({ name: { in: ["a"] } });
      expect(name.contains("a").tree).toEqual({ name: { contains: "a" } });
      expect(name.startsWith("a").tree).toEqual({ name: { startsWith: "a" } });
      expect(name.regex("^a").tree).toEqual({ name: { regex: "^a" } });
    });

    it("should set caseInsensitive on a text condition", () => {
      expect(name.contains("a", { caseInsensitive: true }).tree).toEqual({
        name: { contains: "a", caseInsensitive: true },
      });
    });

    it("should build a contains condition", () => {
      const tags = b.contains("tags");
      expect(tags.has("a").tree).toEqual({ tags: { any: ["a"] } });
      expect(tags.hasAny(["a", "b"]).tree).toEqual({ tags: { any: ["a", "b"] } });
    });

    it("should put a typed bool line's value on the line", () => {
      expect(b.flag("virtual").eq(false).tree).toEqual({ virtual: false });
    });

    it("should put a typed list line's values on the line", () => {
      const labels = b.list("labels");
      expect(labels.has("a").tree).toEqual({ labels: ["a"] });
      expect(labels.hasAny(["a", "b"]).tree).toEqual({ labels: ["a", "b"] });
    });

    it("should put a typed line's value on the line", () => {
      const window = { start: 1, end: 2 };
      expect(b.value("window")(window).tree).toEqual({ window });
    });

    it("should reject a value of the wrong type", () => {
      // @ts-expect-error a key is a number
      key.eq("1");
      // @ts-expect-error a value cannot be absent
      key.eq(undefined);
      // @ts-expect-error a name is a string
      name.eq(1);
    });
  });

  describe("tree", () => {
    it("should match every child with and", () => {
      expect(b.and(key.eq(1), name.eq("a")).tree).toEqual({
        and: [{ key: { eq: 1 } }, { name: { eq: "a" } }],
      });
    });

    it("should match any child with or", () => {
      expect(b.or(key.eq(1), key.eq(2)).tree).toEqual({
        or: [{ key: { eq: 1 } }, { key: { eq: 2 } }],
      });
    });

    it("should negate a child with not", () => {
      expect(b.not(key.eq(1)).tree).toEqual({ not: { key: { eq: 1 } } });
    });

    it("should turn a stored tree into a condition", () => {
      const tree: Where = { or: [{ key: { eq: 1 } }] };
      expect(b.where(tree).tree).toBe(tree);
    });

    it("should reject a condition of another entity", () => {
      const other = filter.builder<"other", Where>();
      // @ts-expect-error an other condition is not a thing condition
      b.and(other.set("key").eq(1));
    });
  });

  describe("toRequest", () => {
    it("should make an empty request from no parts", () => {
      expect(filter.toRequest([])).toEqual({});
    });

    it("should use one condition as the tree", () => {
      expect(filter.toRequest([key.eq(1)])).toEqual({ where: { key: { eq: 1 } } });
    });

    it("should join several conditions with and", () => {
      expect(filter.toRequest([key.eq(1), name.eq("a")])).toEqual({
        where: { and: [{ key: { eq: 1 } }, { name: { eq: "a" } }] },
      });
    });

    it("should set a field for each option", () => {
      const includeLabels = filter.include("includeLabels");
      expect(
        filter.toRequest([
          filter.limit(5),
          filter.offset(10),
          filter.search("a"),
          includeLabels(),
        ]),
      ).toEqual({ limit: 5, offset: 10, searchTerm: "a", includeLabels: true });
    });

    it("should throw if two options set the same field", () => {
      expect(() => filter.toRequest([filter.limit(5), filter.limit(6)])).toThrow(
        "limit is set twice",
      );
    });
  });

  describe("isPart", () => {
    it("should accept conditions, options, and one", () => {
      expect(filter.isPart(key.eq(1))).toBe(true);
      expect(filter.isPart(filter.limit(1))).toBe(true);
      expect(filter.isPart(filter.one())).toBe(true);
    });

    it("should reject anything else", () => {
      expect(filter.isPart({ where: {} })).toBe(false);
      expect(filter.isPart("a")).toBe(false);
    });
  });
});
