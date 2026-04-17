// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Rate, TimeSpan } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { fidelityFor, NATIVE_FIDELITY } from "@/telem/aether/fidelity";

describe("fidelityFor", () => {
  it("returns native fidelity when the raw sample count fits in the pixel budget", () => {
    // 1000 samples over 1000 pixels at oversample=2 fits comfortably.
    const fidelity = fidelityFor({
      timeSpan: TimeSpan.seconds(1),
      rate: Rate.hz(1000),
      pixelWidth: 1000,
      oversampleFactor: 2,
    });
    expect(fidelity).toEqual(NATIVE_FIDELITY);
  });

  it("computes a sensible fidelity for a 3-day 1kHz read on a 2000px plot", () => {
    const fidelity = fidelityFor({
      timeSpan: TimeSpan.seconds(3 * 24 * 60 * 60),
      rate: Rate.hz(1000),
      pixelWidth: 2000,
      oversampleFactor: 2,
    });
    // 259,200,000 samples / 4000 pixel-slots ~= 64,800.
    expect(fidelity).toEqual(64800n);
  });

  it("clamps pixel width <= 0 to native fidelity", () => {
    const fidelity = fidelityFor({
      timeSpan: TimeSpan.seconds(3600),
      rate: Rate.hz(1000),
      pixelWidth: 0,
    });
    expect(fidelity).toEqual(NATIVE_FIDELITY);
  });

  it("honors a custom oversampleFactor", () => {
    // 1M samples, 1000 pixels, oversample=4 -> 1M / 4000 = 250.
    const fidelity = fidelityFor({
      timeSpan: TimeSpan.seconds(1000),
      rate: Rate.hz(1000),
      pixelWidth: 1000,
      oversampleFactor: 4,
    });
    expect(fidelity).toEqual(250n);
  });
});
