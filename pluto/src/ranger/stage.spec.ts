// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type NumericTimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { beforeEach, describe, expect, it } from "vitest";

import { Ranger } from "@/ranger";

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
});

describe("wrapNumericTimeRangeToStage", () => {
  let original: NumericTimeRange;
  let now: TimeStamp;
  let modified: NumericTimeRange;
  let onChange: (v: NumericTimeRange) => void;
  let onStageChange: (v: Ranger.Stage) => void = () => {};
  let stage: Ranger.Stage | undefined;
  beforeEach(() => {
    now = TimeStamp.now();
    onChange = (v) => {
      modified = v;
    };
  });
  describe("when now is before start", () => {
    beforeEach(() => {
      original = {
        start: now.add(TimeSpan.HOUR).nanoseconds,
        end: now.add(TimeSpan.HOUR.mult(2)).nanoseconds,
      };
      ({ value: stage, onChange: onStageChange } = Ranger.wrapNumericTimeRangeToStage({
        value: original,
        onChange,
      }));
    });
    it("correctly interprets the value as 'to_do'", () => {
      expect(stage).toBe("to_do");
    });
    it("changes nothing when changing to 'to_do'", () => {
      onStageChange("to_do");
      expect(modified).toEqual(original);
    });
    it("only moves start time to now when changing to 'in_progress'", () => {
      onStageChange("in_progress");
      expect(modified.end).toEqual(original.end);
      expect(new TimeStamp(modified.start).span(TimeStamp.now()).seconds).toBeLessThan(
        1,
      );
    });
    it("moves both start and end time to now when changing to 'completed'", () => {
      onStageChange("completed");
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
      ({ value: stage, onChange: onStageChange } = Ranger.wrapNumericTimeRangeToStage({
        value: original,
        onChange,
      }));
    });
    it("correctly interprets the value as 'in_progress'", () => {
      expect(stage).toBe("in_progress");
    });
    it("moves start time to end time when changing stage to 'to_do'", () => {
      onStageChange("to_do");
      expect(modified.start).toEqual(original.end);
      expect(modified.end).toEqual(original.end);
    });
    it("changes nothing when changing stage to 'in_progress'", () => {
      onStageChange("in_progress");
      expect(modified).toEqual(original);
    });
    it("moves end time to now when changing to 'completed'", () => {
      onStageChange("completed");
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
      ({ value: stage, onChange: onStageChange } = Ranger.wrapNumericTimeRangeToStage({
        value: original,
        onChange,
      }));
    });
    it("correctly interprets the value as 'completed'", () => {
      expect(stage).toBe("completed");
    });
    it("moves both start and end times to TimeStamp.MAX when changing to 'to_do'", () => {
      onStageChange("to_do");
      expect(modified.start).toEqual(TimeStamp.MAX.nanoseconds);
      expect(modified.end).toEqual(TimeStamp.MAX.nanoseconds);
    });
    it("moves end time to TimeStamp.MAX when changing to 'in_progress'", () => {
      onStageChange("in_progress");
      expect(modified.start).toEqual(original.start);
      expect(modified.end).toEqual(TimeStamp.MAX.nanoseconds);
    });
    it("changes nothing when changing to 'completed'", () => {
      onStageChange("completed");
      expect(modified).toEqual(original);
    });
  });
});
