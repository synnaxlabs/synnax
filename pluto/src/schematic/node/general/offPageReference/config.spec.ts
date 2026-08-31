// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import {
  formatPage,
  type Page,
  PAGE_TYPES,
  parsePage,
} from "@/schematic/node/general/offPageReference/config";

describe("parsePage", () => {
  it.each(PAGE_TYPES)("should parse a typed %s page string", (type) => {
    expect(parsePage(`${type}:abc`)).toEqual({ type, key: "abc" });
  });

  it("should parse a legacy bare string as a schematic key", () => {
    expect(parsePage("abc")).toEqual({ type: "schematic", key: "abc" });
  });

  it("should parse a legacy bare UUID as a schematic key", () => {
    const key = "3f1c9a2e-9d1b-4a5e-8f0a-6b7c8d9e0f1a";
    expect(parsePage(key)).toEqual({ type: "schematic", key });
  });

  it("should treat a string with an invalid type prefix as a schematic key", () => {
    expect(parsePage("bogus:abc")).toEqual({ type: "schematic", key: "bogus:abc" });
  });

  it("should treat a wrongly cased type prefix as a schematic key", () => {
    expect(parsePage("Lineplot:abc")).toEqual({
      type: "schematic",
      key: "Lineplot:abc",
    });
  });

  it("should treat a string with an empty prefix as a schematic key", () => {
    expect(parsePage(":abc")).toEqual({ type: "schematic", key: ":abc" });
  });

  it("should keep colons after the first separator in the key", () => {
    expect(parsePage("log:a:b")).toEqual({ type: "log", key: "a:b" });
  });

  it("should parse a typed string with an empty key", () => {
    expect(parsePage("table:")).toEqual({ type: "table", key: "" });
  });

  it("should parse an empty or missing page as an empty schematic key", () => {
    expect(parsePage("")).toEqual({ type: "schematic", key: "" });
    expect(parsePage()).toEqual({ type: "schematic", key: "" });
  });
});

describe("formatPage", () => {
  it.each(PAGE_TYPES)("should format a %s page with its prefix", (type) => {
    expect(formatPage({ type, key: "abc" })).toBe(`${type}:abc`);
  });

  it.each(PAGE_TYPES)("should format an empty %s key as an empty string", (type) => {
    expect(formatPage({ type, key: "" })).toBe("");
  });

  it("should migrate a legacy bare string to canonical form via a round trip", () => {
    expect(formatPage(parsePage("abc"))).toBe("schematic:abc");
  });

  it.each(PAGE_TYPES)(
    "should leave a canonical %s string unchanged through a round trip",
    (type) => {
      expect(formatPage(parsePage(`${type}:abc`))).toBe(`${type}:abc`);
    },
  );

  it("should leave an empty page unchanged through a round trip", () => {
    expect(formatPage(parsePage(""))).toBe("");
  });

  it.each(PAGE_TYPES)("should invert parsePage for a %s page", (type) => {
    const page: Page = { type, key: "abc" };
    expect(parsePage(formatPage(page))).toEqual(page);
  });
});
