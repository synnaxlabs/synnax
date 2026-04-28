// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { incompleteRangeTelemSpec, rangeIsIncomplete } from "@/channel/LinePlot";

describe("rangeIsIncomplete", () => {
  it("should return true when the range end is in the future", () => {
    const futureEnd = Number(TimeStamp.now()) + 60_000_000_000;
    const timeRange = { start: Number(TimeStamp.now()), end: futureEnd };
    expect(rangeIsIncomplete(timeRange)).toBe(true);
  });

  it("should return false when the range end is in the past", () => {
    const pastEnd = Number(TimeStamp.now()) - 60_000_000_000;
    const pastStart = pastEnd - 60_000_000_000;
    const timeRange = { start: pastStart, end: pastEnd };
    expect(rangeIsIncomplete(timeRange)).toBe(false);
  });
});

describe("incompleteRangeTelemSpec", () => {
  it("should return a streaming telem spec", () => {
    const now = Number(TimeStamp.now());
    const start = now - 60_000_000_000;
    const end = now + 60_000_000_000;
    const timeRange = { start, end };
    const spec = incompleteRangeTelemSpec(timeRange, 1);
    expect(spec.type).toBe("dynamic-series-source");
    expect(spec.variant).toBe("source");
  });

  it("should set timeSpan to the elapsed time since range start", () => {
    const now = Number(TimeStamp.now());
    const start = now - 60_000_000_000;
    const end = now + 60_000_000_000;
    const timeRange = { start, end };
    const spec = incompleteRangeTelemSpec(timeRange, 1);
    const timeSpan = Number(spec.props.timeSpan);
    // timeSpan should be approximately now - start (60s), allow 1s tolerance
    expect(timeSpan).toBeGreaterThan(59_000_000_000);
    expect(timeSpan).toBeLessThan(61_000_000_000);
  });

  it("should set keepFor to the full range duration", () => {
    const now = Number(TimeStamp.now());
    const start = now - 60_000_000_000;
    const end = now + 60_000_000_000;
    const timeRange = { start, end };
    const spec = incompleteRangeTelemSpec(timeRange, 1);
    expect(Number(spec.props.keepFor)).toBe(end - start);
  });

  it("should pass through useIndexOfChannel", () => {
    const now = Number(TimeStamp.now());
    const timeRange = { start: now - 60_000_000_000, end: now + 60_000_000_000 };
    const spec = incompleteRangeTelemSpec(timeRange, 1, true);
    expect(spec.props.useIndexOfChannel).toBe(true);
  });
});
