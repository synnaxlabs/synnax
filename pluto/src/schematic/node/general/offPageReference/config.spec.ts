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
  parsePage,
} from "@/schematic/node/general/offPageReference/config";

describe("parsePage", () => {
  it("should parse a typed page string", () => {
    expect(parsePage("lineplot:abc")).toEqual({ type: "lineplot", key: "abc" });
    expect(parsePage("schematic:abc")).toEqual({ type: "schematic", key: "abc" });
  });

  it("should parse a legacy bare string as a schematic key", () => {
    expect(parsePage("abc")).toEqual({ type: "schematic", key: "abc" });
  });

  it("should treat a string with an invalid type prefix as a schematic key", () => {
    expect(parsePage("bogus:abc")).toEqual({ type: "schematic", key: "bogus:abc" });
  });

  it("should parse an empty or missing page as an empty schematic key", () => {
    expect(parsePage("")).toEqual({ type: "schematic", key: "" });
    expect(parsePage()).toEqual({ type: "schematic", key: "" });
  });
});

describe("formatPage", () => {
  it("should format every page type with its prefix", () => {
    expect(formatPage({ type: "schematic", key: "abc" })).toBe("schematic:abc");
    expect(formatPage({ type: "table", key: "abc" })).toBe("table:abc");
  });

  it("should format an empty key as an empty string", () => {
    expect(formatPage({ type: "log", key: "" })).toBe("");
  });

  it("should migrate a legacy bare string to canonical form via a round trip", () => {
    expect(formatPage(parsePage("abc"))).toBe("schematic:abc");
  });

  it("should leave a canonical string unchanged through a round trip", () => {
    expect(formatPage(parsePage("table:abc"))).toBe("table:abc");
  });
});
