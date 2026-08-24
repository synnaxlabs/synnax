// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { moveEnd, moveStart, resolutionFor } from "@/ranger/Timeline";

const UNSET = TimeStamp.MAX.nanoseconds;

describe("resolutionFor", () => {
  it("should coarsen with the span", () => {
    expect(resolutionFor(TimeSpan.days(2)).equals(TimeSpan.MINUTE)).toBe(true);
    expect(resolutionFor(TimeSpan.hours(2)).equals(TimeSpan.SECOND)).toBe(true);
    expect(resolutionFor(TimeSpan.minutes(5)).equals(TimeSpan.SECOND)).toBe(true);
    expect(resolutionFor(TimeSpan.seconds(15)).equals(TimeSpan.MILLISECOND)).toBe(true);
    expect(resolutionFor(TimeSpan.milliseconds(3)).equals(TimeSpan.MICROSECOND)).toBe(
      true,
    );
  });
});

describe("moveStart", () => {
  it("should keep the end when the edit stays before it", () => {
    expect(moveStart({ start: 100, end: 200 }, 150)).toEqual({ start: 150, end: 200 });
  });
  it("should slide the end to keep the duration when the edit crosses it", () => {
    expect(moveStart({ start: 100, end: 200 }, 300)).toEqual({ start: 300, end: 400 });
  });
  it("should leave an open end alone", () => {
    expect(moveStart({ start: 100, end: UNSET }, 500)).toEqual({
      start: 500,
      end: UNSET,
    });
  });
  it("should clear the end when unscheduling", () => {
    expect(moveStart({ start: 100, end: 200 }, UNSET)).toEqual({
      start: UNSET,
      end: UNSET,
    });
  });
});

describe("moveEnd", () => {
  it("should keep the start when the edit stays after it", () => {
    expect(moveEnd({ start: 100, end: 200 }, 150)).toEqual({ start: 100, end: 150 });
  });
  it("should slide the start to keep the duration when the edit crosses it", () => {
    expect(moveEnd({ start: 100, end: 200 }, 50)).toEqual({ start: -50, end: 50 });
  });
  it("should clamp the start when the range was open", () => {
    expect(moveEnd({ start: 100, end: UNSET }, 50)).toEqual({ start: 50, end: 50 });
  });
  it("should keep the start when clearing the end", () => {
    expect(moveEnd({ start: 100, end: 200 }, UNSET)).toEqual({
      start: 100,
      end: UNSET,
    });
  });
});
