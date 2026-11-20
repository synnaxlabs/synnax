// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { ranger } from "@synnaxlabs/client";
import { type NumericTimeRange, TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Ranger } from "@/ranger";

// A float64 holds the nanoseconds of this instant exactly.
const EXACT = new Date(Date.UTC(2026, 0, 1));

describe("getStage", () => {
  it("returns 'to_do' if now is before start", () => {
    const now = TimeStamp.now();
    const tr = { start: now.add(TimeSpan.HOUR), end: now.add(TimeSpan.HOUR.mult(2)) };
    expect(Ranger.getStage(tr)).toBe("to_do");
  });

  it("returns 'completed' if now is after end", () => {
    const now = TimeStamp.now();
    const tr = { start: now.sub(TimeSpan.HOUR.mult(2)), end: now.sub(TimeSpan.HOUR) };
    expect(Ranger.getStage(tr)).toBe("completed");
  });

  it("returns 'in_progress' if now is between start and end", () => {
    const now = TimeStamp.now();
    const tr = { start: now.sub(TimeSpan.HOUR), end: now.add(TimeSpan.HOUR) };
    expect(Ranger.getStage(tr)).toBe("in_progress");
  });

  it("returns 'in_progress' if now is at start", () => {
    const now = TimeStamp.now();
    expect(Ranger.getStage({ start: now, end: now.add(TimeSpan.HOUR) }, now)).toBe(
      "in_progress",
    );
  });

  it("returns 'completed' if now is at end", () => {
    const now = TimeStamp.now();
    expect(Ranger.getStage({ start: now.sub(TimeSpan.HOUR), end: now }, now)).toBe(
      "completed",
    );
  });
});

describe("moveToStage", () => {
  let original: NumericTimeRange;
  let now: TimeStamp;
  beforeEach(() => {
    now = TimeStamp.now();
  });
  describe("when now is before start", () => {
    beforeEach(() => {
      original = {
        start: now.add(TimeSpan.HOUR).nanoseconds,
        end: now.add(TimeSpan.HOUR.mult(2)).nanoseconds,
      };
    });
    it("changes nothing when changing to 'to_do'", () => {
      expect(Ranger.moveToStage(original, "to_do")).toEqual(original);
    });
    it("only moves start time to now when changing to 'in_progress'", () => {
      const modified = Ranger.moveToStage(original, "in_progress");
      expect(modified.end).toEqual(original.end);
      expect(new TimeStamp(modified.start).span(TimeStamp.now()).seconds).toBeLessThan(
        1,
      );
    });
    it("moves both start and end time to now when changing to 'completed'", () => {
      const modified = Ranger.moveToStage(original, "completed");
      expect(new TimeStamp(modified.start).span(TimeStamp.now()).seconds).toBeLessThan(
        1,
      );
      expect(new TimeStamp(modified.end).span(TimeStamp.now()).seconds).toBeLessThan(1);
    });
  });
  describe("when now is between start and end", () => {
    beforeEach(() => {
      original = {
        start: now.sub(TimeSpan.HOUR).nanoseconds,
        end: now.add(TimeSpan.HOUR).nanoseconds,
      };
    });
    it("moves start time to end time when changing stage to 'to_do'", () => {
      const modified = Ranger.moveToStage(original, "to_do");
      expect(modified.start).toEqual(original.end);
      expect(modified.end).toEqual(original.end);
    });
    it("changes nothing when changing stage to 'in_progress'", () => {
      expect(Ranger.moveToStage(original, "in_progress")).toEqual(original);
    });
    it("moves end time to now when changing to 'completed'", () => {
      const modified = Ranger.moveToStage(original, "completed");
      expect(modified.start).toEqual(original.start);
      expect(new TimeStamp(modified.end).span(TimeStamp.now()).seconds).toBeLessThan(1);
    });
  });
  describe("when now is after end", () => {
    beforeEach(() => {
      original = {
        start: now.sub(TimeSpan.HOUR.mult(2)).nanoseconds,
        end: now.sub(TimeSpan.HOUR).nanoseconds,
      };
    });
    it("moves both start and end times to TimeStamp.MAX when changing to 'to_do'", () => {
      const modified = Ranger.moveToStage(original, "to_do");
      expect(modified.start).toEqual(TimeStamp.MAX.nanoseconds);
      expect(modified.end).toEqual(TimeStamp.MAX.nanoseconds);
    });
    it("moves end time to TimeStamp.MAX when changing to 'in_progress'", () => {
      const modified = Ranger.moveToStage(original, "in_progress");
      expect(modified.start).toEqual(original.start);
      expect(modified.end).toEqual(TimeStamp.MAX.nanoseconds);
    });
    it("changes nothing when changing to 'completed'", () => {
      expect(Ranger.moveToStage(original, "completed")).toEqual(original);
    });
  });
  describe("when now is at a bound", () => {
    beforeEach(() => {
      vi.useFakeTimers({ now: EXACT });
      now = TimeStamp.now().sub(TimeSpan.MILLISECOND);
    });
    afterEach(() => {
      vi.useRealTimers();
    });
    it("moves a start at now to the end when changing to 'to_do'", () => {
      original = { start: now.nanoseconds, end: now.add(TimeSpan.HOUR).nanoseconds };
      const modified = Ranger.moveToStage(original, "to_do");
      expect(Ranger.getStage(modified)).toBe("to_do");
    });
    it("clears an end at now when changing to 'in_progress'", () => {
      original = { start: now.sub(TimeSpan.HOUR).nanoseconds, end: now.nanoseconds };
      const modified = Ranger.moveToStage(original, "in_progress");
      expect(modified.end).toEqual(TimeStamp.MAX.nanoseconds);
    });
  });
});

describe("useClock", () => {
  beforeEach(() => {
    vi.useFakeTimers({ now: EXACT });
  });
  afterEach(() => {
    vi.useRealTimers();
  });
  it("should catch up when a new range starts between its time and now", () => {
    const start = TimeStamp.now();
    const unset = TimeStamp.MAX.nanoseconds;
    const { result, rerender } = renderHook(
      ({ range }: { range: NumericTimeRange }) => Ranger.useClock(range),
      { initialProps: { range: { start: unset, end: unset } } },
    );
    vi.setSystemTime(start.add(TimeSpan.HOUR).date());
    const range = { start: start.add(TimeSpan.minutes(30)).nanoseconds, end: unset };
    rerender({ range });
    expect(Ranger.getStage(range, result.current)).toBe("in_progress");
  });
});

const AT = TimeStamp.seconds(1000);
const DAY = TimeSpan.days(1);

const rangeOf = (start: TimeStamp, end: TimeStamp): ranger.Range =>
  ({ timeRange: new TimeRange(start, end) }) as ranger.Range;

describe("nextCrossing", () => {
  it("should return null for a filter that reads no time", () => {
    const ranges = [rangeOf(AT.add(TimeSpan.SECOND), TimeStamp.MAX)];
    expect(Ranger.nextCrossing({ labeled: true }, ranges, AT)).toBeNull();
    expect(Ranger.nextCrossing(undefined, ranges, AT)).toBeNull();
  });

  it("should cross when a range in a time filter leaves at its end plus the span", () => {
    const end = AT.sub(TimeSpan.HOUR);
    const ranges = [rangeOf(AT.sub(TimeSpan.days(2)), end)];
    expect(Ranger.nextCrossing({ recent: DAY }, ranges, AT)).toEqual(end.add(DAY));
  });

  it("should cross at the first start or end after the instant for a stage filter", () => {
    const start = AT.add(TimeSpan.seconds(5));
    const end = AT.add(TimeSpan.seconds(3));
    const ranges = [rangeOf(start, TimeStamp.MAX), rangeOf(AT.sub(DAY), end)];
    expect(Ranger.nextCrossing({ stages: ["in_progress"] }, ranges, AT)).toEqual(end);
  });

  it("should not cross at an open end", () => {
    const ranges = [rangeOf(AT.sub(DAY), TimeStamp.MAX)];
    expect(Ranger.nextCrossing({ stages: ["in_progress"] }, ranges, AT)).toBeNull();
  });
});
