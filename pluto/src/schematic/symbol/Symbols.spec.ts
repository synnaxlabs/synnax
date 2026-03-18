// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { offPageReferenceTooltip } from "@/schematic/symbol/Symbols";

describe("offPageReferenceTooltip", () => {
  it("should return double-click tooltip by default", () => {
    expect(offPageReferenceTooltip("page-key")).toBe(
      "Double-click to navigate",
    );
  });

  it("should return double-click tooltip when dblClickNav is true", () => {
    expect(offPageReferenceTooltip("page-key", true)).toBe(
      "Double-click to navigate",
    );
  });

  it("should return single-click tooltip when dblClickNav is false", () => {
    expect(offPageReferenceTooltip("page-key", false)).toBe(
      "Single-click to navigate",
    );
  });

  it("should return undefined when page is undefined", () => {
    expect(offPageReferenceTooltip(undefined)).toBeUndefined();
  });

  it("should return undefined when page is empty string", () => {
    expect(offPageReferenceTooltip("")).toBeUndefined();
  });
});
