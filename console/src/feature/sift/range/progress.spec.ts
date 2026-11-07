// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type sift, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/client";
import { describe, expect, it } from "vitest";

import { Sift } from "@/feature/sift";

const START = TimeStamp.seconds(10);

const details = (...positions: TimeSpan[]): sift.UploadDetails => ({
  device: "dev",
  range: "00000000-0000-0000-0000-000000000000",
  timeRange: new TimeRange(START, START.add(TimeSpan.seconds(10))),
  channels: [],
  asset: "stand",
  positions: positions.map((p, i) => ({ index: i + 1, time: START.add(p) })),
  samples: 0,
});

describe("Sift.Range.progress", () => {
  it("should be zero before the first checkpoint", () => {
    expect(Sift.Range.progress(details())).toBe(0);
  });

  it("should follow the index that has reached the least far", () => {
    expect(
      Sift.Range.progress(details(TimeSpan.seconds(8), TimeSpan.seconds(4))),
    ).toBeCloseTo(0.4);
  });

  it("should stop at one", () => {
    expect(Sift.Range.progress(details(TimeSpan.seconds(20)))).toBe(1);
  });
});
