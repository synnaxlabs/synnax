// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { roundResizeDims } from "@/schematic/symbol/Grid";

describe("roundResizeDims", () => {
  it("should round fractional width and height", () => {
    expect(roundResizeDims(100.7, 200.3)).toEqual({ width: 101, height: 200 });
  });

  it("should not change integer values", () => {
    expect(roundResizeDims(100, 200)).toEqual({ width: 100, height: 200 });
  });

  it("should round 0.5 up", () => {
    expect(roundResizeDims(99.5, 50.5)).toEqual({ width: 100, height: 51 });
  });

  it("should handle zero values", () => {
    expect(roundResizeDims(0, 0)).toEqual({ width: 0, height: 0 });
  });

  it("should handle negative values", () => {
    expect(roundResizeDims(-10.4, -20.6)).toEqual({ width: -10, height: -21 });
  });
});
