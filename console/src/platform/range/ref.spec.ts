// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Range } from "@/platform/range";

describe("Range.toRef", () => {
  it("should reference a range the Core holds by key", () => {
    expect(Range.toRef({ variant: "persisted", key: "k" })).toEqual({
      variant: "key",
      key: "k",
    });
  });

  it("should carry a rolling range as its window", () => {
    const span = Number(TimeSpan.hours(1));
    expect(Range.toRef({ variant: "dynamic", key: "r", name: "R", span })).toEqual({
      variant: "window",
      window: { variant: "dynamic", timeSpan: span },
    });
  });

  it("should carry a session-only fixed range as its window", () => {
    expect(
      Range.toRef({
        variant: "static",
        key: "s",
        name: "S",
        timeRange: { start: 1_000, end: 2_000.5 },
      }),
    ).toEqual({
      variant: "window",
      window: { variant: "static", timeRange: { start: "1000", end: "2000" } },
    });
  });
});

describe("Range.setBound", () => {
  const window = {
    variant: "static" as const,
    timeRange: { start: "1000", end: "2000" },
  };

  it("should move one bound", () => {
    expect(Range.setBound(window, "start", 1500).timeRange).toEqual({
      start: "1500",
      end: "2000",
    });
    expect(Range.setBound(window, "end", 2500).timeRange).toEqual({
      start: "1000",
      end: "2500",
    });
  });

  it("should carry the end along when the start crosses it", () => {
    expect(Range.setBound(window, "start", 3000).timeRange).toEqual({
      start: "3000",
      end: "4000",
    });
  });

  it("should carry the start along when the end crosses it", () => {
    expect(Range.setBound(window, "end", 500).timeRange).toEqual({
      start: "-500",
      end: "500",
    });
  });
});
