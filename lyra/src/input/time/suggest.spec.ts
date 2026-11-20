// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { beforeAll, describe, expect, it } from "vitest";

import {
  type Language,
  loadLanguage,
  suggestTimeRange,
  suggestTimeSpans,
  suggestTimeStamps,
} from "@/input/time/suggest";

const NOW = new TimeStamp(new Date(2026, 7, 23, 14, 5, 0, 0));
const START = new TimeStamp(new Date(2026, 7, 20, 9, 0, 0, 0));

let language: Language;
beforeAll(async () => {
  language = await loadLanguage();
});

describe("suggestTimeStamps", () => {
  it("should lead with the grammar's exact reading", () => {
    const [first] = suggestTimeStamps("now - 5m", { language, anchors: { now: NOW } });
    expect(first.value.equals(NOW.sub(TimeSpan.minutes(5)))).toBe(true);
    expect(first.reading).toBe("5min before now");
  });

  it("should name the anchor and the side of it an offset lands on", () => {
    const anchors = { now: NOW, parent: START };
    const [after] = suggestTimeStamps("T+3.2s", { language, anchors });
    expect(after.reading).toBe("3s 200ms after the parent start");
    const [bare] = suggestTimeStamps("T", { language, anchors });
    expect(bare.reading).toBe("the parent start");
  });

  it("should read an epoch in its one plausible unit", () => {
    const us = NOW.valueOf() / 1000n;
    const res = suggestTimeStamps(us.toString(), { language, anchors: { now: NOW } });
    expect(res.map((r) => r.reading)).toEqual(["Unix microseconds"]);
    expect(res[0].value.equals(NOW)).toBe(true);
  });

  it("should place a bare time on the other end's day for an end cell", () => {
    const res = suggestTimeStamps("11:30", {
      language,
      anchors: { now: NOW, start: START },
      bound: "end",
    });
    expect(res[0].reading).toBe("today");
    expect(res[1].reading).toBe("on the day of the start");
    expect(res[1].value.equals(new TimeStamp(new Date(2026, 7, 20, 11, 30)))).toBe(
      true,
    );
  });

  it("should read a bare duration relative to the range", () => {
    const res = suggestTimeStamps("2h", {
      language,
      anchors: { now: NOW, start: START },
      bound: "end",
    });
    expect(res[0].reading).toBe("2h after the start");
    expect(res[0].value.equals(START.add(TimeSpan.hours(2)))).toBe(true);
    expect(res.map((r) => r.reading)).toContain("2h from now");
  });

  it("should shift the current value by a signed duration", () => {
    const res = suggestTimeStamps("+30s", {
      language,
      anchors: { now: NOW },
      current: START,
    });
    expect(res[0].reading).toBe("30s after the current value");
    expect(res[0].value.equals(START.add(TimeSpan.seconds(30)))).toBe(true);
  });

  it("should read natural language", () => {
    const res = suggestTimeStamps("tomorrow at 3pm", {
      language,
      anchors: { now: NOW },
    });
    expect(res[0].value.equals(new TimeStamp(new Date(2026, 7, 24, 15, 0)))).toBe(true);
    const ago = suggestTimeStamps("5m ago", { language, anchors: { now: NOW } });
    expect(ago[0].value.equals(NOW.sub(TimeSpan.minutes(5)))).toBe(true);
  });

  it("should offer the end of a whole day to an end cell", () => {
    const res = suggestTimeStamps("tomorrow", {
      language,
      anchors: { now: NOW },
      bound: "end",
    });
    expect(res[0].reading).toBe("start of tomorrow");
    expect(res[1].reading).toBe("end of tomorrow");
  });

  it("should return nothing for nonsense", () => {
    expect(suggestTimeStamps("purple", { language, anchors: { now: NOW } })).toEqual(
      [],
    );
  });
});

describe("suggestTimeSpans", () => {
  it("should lead with the grammar's reading", () => {
    const [first] = suggestTimeSpans("2h 30m", language);
    expect(first.value.equals(TimeSpan.minutes(150))).toBe(true);
  });

  it("should read a bare number in the given unit and name the unit", () => {
    const suggestions = suggestTimeSpans("1000", language, TimeSpan.MILLISECOND);
    expect(suggestions).toHaveLength(1);
    expect(suggestions[0].value.equals(TimeSpan.SECOND)).toBe(true);
    expect(suggestions[0].reading).toBe("milliseconds");
  });

  it("should throw on a unit it cannot name", () => {
    expect(() => suggestTimeSpans("5", language, TimeSpan.seconds(2))).toThrow(
      "2s is not a unit of time",
    );
  });

  it("should read Ms only as megaseconds", () => {
    const suggestions = suggestTimeSpans("5Ms", language);
    expect(suggestions.length).toBeGreaterThan(0);
    for (const { value } of suggestions)
      expect(value.equals(TimeSpan.seconds(5e6))).toBe(true);
  });

  it("should read natural language durations", () => {
    expect(
      suggestTimeSpans("an hour and a half", language)[0].value.equals(
        TimeSpan.minutes(90),
      ),
    ).toBe(true);
    expect(
      suggestTimeSpans("half an hour", language)[0].value.equals(TimeSpan.minutes(30)),
    ).toBe(true);
    expect(
      suggestTimeSpans("2 hours and 30 minutes", language)[0].value.equals(
        TimeSpan.minutes(150),
      ),
    ).toBe(true);
    expect(
      suggestTimeSpans("five minutes", language)[0].value.equals(TimeSpan.minutes(5)),
    ).toBe(true);
  });
});

const local = (day: number, hour: number, minute: number = 0): TimeStamp =>
  new TimeStamp(new Date(2026, 7, day, hour, minute));

describe("suggestTimeRange", () => {
  it("should read a day and a span of times", () => {
    expect(suggestTimeRange("Yesterday from 1pm to 2pm", language, NOW)).toEqual(
      new TimeRange(local(22, 13), local(22, 14)),
    );
    expect(suggestTimeRange("today 9:30-10:15am", language, NOW)).toEqual(
      new TimeRange(local(23, 9, 30), local(23, 10, 15)),
    );
  });

  it("should cover all of a day it names, up to the next midnight", () => {
    expect(suggestTimeRange("yesterday", language, NOW)).toEqual(
      new TimeRange(local(22, 0), local(23, 0)),
    );
  });

  it("should read a start and a length", () => {
    expect(suggestTimeRange("2h ago for 1h", language, NOW)).toEqual(
      new TimeRange(NOW.sub(TimeSpan.hours(2)), NOW.sub(TimeSpan.hours(1))),
    );
    expect(suggestTimeRange("yesterday 1pm for half an hour", language, NOW)).toEqual(
      new TimeRange(local(22, 13), local(22, 13, 30)),
    );
  });

  it("should read a start until now", () => {
    expect(suggestTimeRange("since 1pm", language, NOW)).toEqual(
      new TimeRange(local(23, 13), NOW),
    );
  });

  it("should read two ends", () => {
    expect(suggestTimeRange("3h ago to 1h ago", language, NOW)).toEqual(
      new TimeRange(NOW.sub(TimeSpan.hours(3)), NOW.sub(TimeSpan.hours(1))),
    );
    expect(suggestTimeRange("30m ago to now", language, NOW)).toEqual(
      new TimeRange(NOW.sub(TimeSpan.minutes(30)), NOW),
    );
  });

  it("should read a range pasted from a window's tooltip", () => {
    const text = "2026-08-22 13:00:00.000 → 2026-08-22 14:00:00.000";
    expect(suggestTimeRange(text, language, NOW)).toEqual(
      new TimeRange(local(22, 13), local(22, 14)),
    );
  });

  it("should place a time of day alone on the day of the other end", () => {
    expect(suggestTimeRange("13:00 to 14:00 yesterday", language, NOW)).toEqual(
      new TimeRange(local(22, 13), local(22, 14)),
    );
    expect(suggestTimeRange("yesterday 10pm to 2am", language, NOW)).toEqual(
      new TimeRange(local(22, 22), local(23, 2)),
    );
  });

  it("should end on the last day of a span of days", () => {
    expect(suggestTimeRange("aug 17 through aug 19", language, NOW)).toEqual(
      new TimeRange(local(17, 0), local(20, 0)),
    );
  });

  it("should read nothing when other words name a range", () => {
    expect(suggestTimeRange("monday test", language, NOW)).toBeNull();
  });

  it("should read an instant alone as the hour after it", () => {
    expect(suggestTimeRange("3h ago", language, NOW)).toEqual(
      new TimeRange(NOW.sub(TimeSpan.hours(3)), NOW.sub(TimeSpan.hours(2))),
    );
    expect(suggestTimeRange("yesterday 1pm", language, NOW)).toEqual(
      new TimeRange(local(22, 13), local(22, 14)),
    );
  });

  it("should end the hour after an instant at now", () => {
    expect(suggestTimeRange("10m ago", language, NOW)).toEqual(
      new TimeRange(NOW.sub(TimeSpan.minutes(10)), NOW),
    );
  });

  it("should read nothing for an instant after now", () => {
    expect(suggestTimeRange("in 2 hours", language, NOW)).toBeNull();
  });

  it("should read nothing for text that names no time", () => {
    expect(suggestTimeRange("purple", language, NOW)).toBeNull();
    expect(suggestTimeRange("", language, NOW)).toBeNull();
  });
});
