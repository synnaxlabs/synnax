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
  configZ,
  type Page,
  PAGE_TYPES,
  parsePage,
} from "@/schematic/node/general/offPageReference/config";

describe("parsePage", () => {
  it.each(PAGE_TYPES)("should return a valid %s page unchanged", (type) => {
    const page: Page = { type, key: "abc" };
    expect(parsePage(page)).toEqual(page);
  });

  it("should parse a legacy bare string as a schematic key", () => {
    expect(parsePage("abc")).toEqual({ type: "schematic", key: "abc" });
  });

  it("should parse a legacy bare UUID as a schematic key", () => {
    const key = "3f1c9a2e-9d1b-4a5e-8f0a-6b7c8d9e0f1a";
    expect(parsePage(key)).toEqual({ type: "schematic", key });
  });

  it("should treat a string with colons as a legacy schematic key", () => {
    expect(parsePage("lineplot:abc")).toEqual({
      type: "schematic",
      key: "lineplot:abc",
    });
  });

  it("should parse an empty or missing page as an empty schematic key", () => {
    expect(parsePage("")).toEqual({ type: "schematic", key: "" });
    expect(parsePage()).toEqual({ type: "schematic", key: "" });
  });

  it("should normalize an object with an invalid type to an empty schematic key", () => {
    expect(parsePage({ type: "bogus", key: "abc" })).toEqual({
      type: "schematic",
      key: "",
    });
  });

  it("should normalize a malformed value to an empty schematic key", () => {
    expect(parsePage(null)).toEqual({ type: "schematic", key: "" });
    expect(parsePage(42)).toEqual({ type: "schematic", key: "" });
    expect(parsePage({ key: "abc" })).toEqual({ type: "schematic", key: "" });
  });
});

describe("configZ", () => {
  const base = { variant: "offPageReference", label: { label: "ref" } } as const;

  it.each(PAGE_TYPES)("should accept a %s page object", (type) => {
    const res = configZ.safeParse({ ...base, page: { type, key: "abc" } });
    expect(res.success).toBe(true);
  });

  it("should accept a legacy bare string page", () => {
    const res = configZ.safeParse({ ...base, page: "abc" });
    expect(res.success).toBe(true);
  });

  it("should accept a missing page", () => {
    const res = configZ.safeParse(base);
    expect(res.success).toBe(true);
  });

  it("should reject a page object with an invalid type", () => {
    const res = configZ.safeParse({ ...base, page: { type: "bogus", key: "abc" } });
    expect(res.success).toBe(false);
  });
});
