// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { Ranger } from "@synnaxlabs/pluto";
import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { nextCrossing, pin } from "@/feature/range/explorer/time";

const AT = TimeStamp.seconds(1000);
const DAY = TimeSpan.days(1);

const rangeOf = (start: TimeStamp, end: TimeStamp): ranger.Range =>
  ({ timeRange: new TimeRange(start, end) }) as ranger.Range;

describe("pin", () => {
  it("should read a recent span back from the instant", () => {
    expect(pin({ recent: DAY }, AT)).toEqual({
      and: [{ overlaps: new TimeRange(AT.sub(DAY), AT) }],
    });
  });

  it("should keep the other fields beside the pinned span", () => {
    expect(pin({ labeled: true, recent: DAY }, AT)).toEqual({
      and: [{ labeled: true }, { overlaps: new TimeRange(AT.sub(DAY), AT) }],
    });
  });

  it("should pin the stages and spans inside and, or, and not", () => {
    const inProgress = { and: [{ or: [Ranger.stageWhere("in_progress", AT)] }] };
    const recent = { and: [{ overlaps: new TimeRange(AT.sub(DAY), AT) }] };
    expect(
      pin(
        {
          and: [{ stages: ["in_progress"] }, { labeled: true }],
          or: [{ recent: DAY }],
          not: { stages: ["in_progress"] },
        },
        AT,
      ),
    ).toEqual({
      and: [inProgress, { labeled: true }],
      or: [recent],
      not: inProgress,
    });
  });

  it("should return a filter that reads no time as it is", () => {
    expect(pin({ labeled: true }, AT)).toEqual({ labeled: true });
  });
});

describe("nextCrossing", () => {
  it("should return null for a filter that reads no time", () => {
    const ranges = [rangeOf(AT.add(TimeSpan.SECOND), TimeStamp.MAX)];
    expect(nextCrossing({ labeled: true }, ranges, AT)).toBeNull();
    expect(nextCrossing(undefined, ranges, AT)).toBeNull();
  });

  it("should cross when a range in a time filter leaves at its end plus the span", () => {
    const end = AT.sub(TimeSpan.HOUR);
    const ranges = [rangeOf(AT.sub(TimeSpan.days(2)), end)];
    expect(nextCrossing({ recent: DAY }, ranges, AT)).toEqual(end.add(DAY));
  });

  it("should cross at the first start or end after the instant for a stage filter", () => {
    const start = AT.add(TimeSpan.seconds(5));
    const end = AT.add(TimeSpan.seconds(3));
    const ranges = [rangeOf(start, TimeStamp.MAX), rangeOf(AT.sub(DAY), end)];
    expect(nextCrossing({ stages: ["in_progress"] }, ranges, AT)).toEqual(end);
  });

  it("should not cross at an open end", () => {
    const ranges = [rangeOf(AT.sub(DAY), TimeStamp.MAX)];
    expect(nextCrossing({ stages: ["in_progress"] }, ranges, AT)).toBeNull();
  });
});
