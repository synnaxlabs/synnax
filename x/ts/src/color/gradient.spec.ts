// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { color } from "@/color";

const STOPS: color.Stop[] = [
  { position: 10, color: [0, 0, 0, 1] },
  { position: 0, color: [255, 0, 0, 1] },
  { position: 20, color: [0, 0, 255, 1] },
];

describe("fromGradient", () => {
  it("should return transparent for a gradient with no stops", () => {
    expect(color.fromGradient([], 5)).toEqual(color.TRANSPARENT);
  });

  it("should clamp to the first and last stops", () => {
    expect(color.fromGradient(STOPS, -5)).toEqual([255, 0, 0, 1]);
    expect(color.fromGradient(STOPS, 25)).toEqual([0, 0, 255, 1]);
  });

  it("should return the color of a stop at its position", () => {
    expect(color.fromGradient(STOPS, 10)).toEqual([0, 0, 0, 1]);
  });

  it("should mix the colors of the stops around the position", () => {
    expect(color.fromGradient(STOPS, 5)).toEqual([128, 0, 0, 1]);
    expect(color.fromGradient(STOPS, 15)).toEqual([0, 0, 128, 1]);
  });
});
