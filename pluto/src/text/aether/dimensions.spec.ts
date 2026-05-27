// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { text } from "@/text/aether";

describe("dimensionsFromMetrics", () => {
  it("should calculate dimensions from positive metrics", () => {
    const metrics = {
      actualBoundingBoxLeft: 10,
      actualBoundingBoxRight: 20,
      actualBoundingBoxAscent: 15,
      actualBoundingBoxDescent: 5,
    } as TextMetrics;
    expect(text.dimensionsFromMetrics(metrics)).toEqual({ width: 30, height: 20 });
  });

  it("should calculate dimensions from negative metrics", () => {
    const metrics = {
      actualBoundingBoxLeft: -10,
      actualBoundingBoxRight: -20,
      actualBoundingBoxAscent: -15,
      actualBoundingBoxDescent: -5,
    } as TextMetrics;
    expect(text.dimensionsFromMetrics(metrics)).toEqual({ width: 30, height: 20 });
  });

  it("should calculate dimensions from mixed positive and negative metrics", () => {
    const metrics = {
      actualBoundingBoxLeft: -10,
      actualBoundingBoxRight: 20,
      actualBoundingBoxAscent: 15,
      actualBoundingBoxDescent: -5,
    } as TextMetrics;
    expect(text.dimensionsFromMetrics(metrics)).toEqual({ width: 30, height: 20 });
  });

  it("should handle zero metrics", () => {
    const metrics = {
      actualBoundingBoxLeft: 0,
      actualBoundingBoxRight: 0,
      actualBoundingBoxAscent: 0,
      actualBoundingBoxDescent: 0,
    } as TextMetrics;
    expect(text.dimensionsFromMetrics(metrics)).toEqual({ width: 0, height: 0 });
  });
});
