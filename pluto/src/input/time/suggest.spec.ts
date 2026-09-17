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

import { suggestTimeSpans, suggestTimeStamps } from "@/input/time/suggest";

const NOW = new TimeStamp(new Date(2026, 7, 23, 14, 5, 0, 0));
const START = new TimeStamp(new Date(2026, 7, 20, 9, 0, 0, 0));

describe("suggestTimeStamps", () => {
  it("should lead with the grammar's exact reading", () => {
    const [first] = suggestTimeStamps("now - 5m", { anchors: { now: NOW } });
    expect(first.value.equals(NOW.sub(TimeSpan.minutes(5)))).toBe(true);
    expect(first.reading).toBe("5m before now");
  });

  it("should offer every plausible epoch unit", () => {
    const ms = Number(NOW.valueOf() / 1000000n);
    const res = suggestTimeStamps(ms.toString(), { anchors: { now: NOW } });
    expect(res[0].reading).toBe("Unix milliseconds");
    expect(res[0].value.equals(NOW)).toBe(true);
    expect(res.map((r) => r.reading)).not.toContain("Unix seconds");
  });

  it("should place a bare time on the other end's day for an end cell", () => {
    const res = suggestTimeStamps("11:30", {
      anchors: { now: NOW, start: START },
      role: "end",
    });
    expect(res[0].reading).toBe("today");
    expect(res[1].reading).toBe("on the day of the start");
    expect(res[1].value.equals(new TimeStamp(new Date(2026, 7, 20, 11, 30)))).toBe(
      true,
    );
  });

  it("should read a bare duration relative to the range", () => {
    const res = suggestTimeStamps("2h", {
      anchors: { now: NOW, start: START },
      role: "end",
    });
    expect(res[0].reading).toBe("2h after the start");
    expect(res[0].value.equals(START.add(TimeSpan.hours(2)))).toBe(true);
    expect(res.map((r) => r.reading)).toContain("2h from now");
  });

  it("should shift the current value by a signed duration", () => {
    const res = suggestTimeStamps("+30s", { anchors: { now: NOW }, current: START });
    expect(res[0].reading).toBe("30s after the current value");
    expect(res[0].value.equals(START.add(TimeSpan.seconds(30)))).toBe(true);
  });

  it("should read natural language", () => {
    const res = suggestTimeStamps("tomorrow at 3pm", { anchors: { now: NOW } });
    expect(res[0].value.equals(new TimeStamp(new Date(2026, 7, 24, 15, 0)))).toBe(true);
    const ago = suggestTimeStamps("5m ago", { anchors: { now: NOW } });
    expect(ago[0].value.equals(NOW.sub(TimeSpan.minutes(5)))).toBe(true);
  });

  it("should offer the end of a whole day to an end cell", () => {
    const res = suggestTimeStamps("tomorrow", { anchors: { now: NOW }, role: "end" });
    expect(res[0].reading).toBe("start of tomorrow");
    expect(res[1].reading).toBe("end of tomorrow");
  });

  it("should return nothing for nonsense", () => {
    expect(suggestTimeStamps("purple", { anchors: { now: NOW } })).toEqual([]);
  });
});

describe("suggestTimeSpans", () => {
  it("should lead with the grammar's reading", () => {
    const [first] = suggestTimeSpans("2h 30m");
    expect(first.value.equals(TimeSpan.minutes(150))).toBe(true);
  });

  it("should read natural language durations", () => {
    expect(
      suggestTimeSpans("an hour and a half")[0].value.equals(TimeSpan.minutes(90)),
    ).toBe(true);
    expect(suggestTimeSpans("half an hour")[0].value.equals(TimeSpan.minutes(30))).toBe(
      true,
    );
    expect(
      suggestTimeSpans("2 hours and 30 minutes")[0].value.equals(TimeSpan.minutes(150)),
    ).toBe(true);
    expect(suggestTimeSpans("five minutes")[0].value.equals(TimeSpan.minutes(5))).toBe(
      true,
    );
  });
});
