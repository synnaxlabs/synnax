// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeRange, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { renderHook } from "@testing-library/react";
import { beforeAll, describe, expect, it } from "vitest";

import { Input } from "@/input";
import { preloadTimeLanguage } from "@/testutil";

const NOW = new TimeStamp(new Date(2026, 7, 23, 14, 5, 0, 0));

const local = (day: number, hour: number, minute: number = 0): TimeStamp =>
  new TimeStamp(new Date(2026, 7, day, hour, minute));

describe("Input.suggestTimeRange", () => {
  preloadTimeLanguage();
  let language: Input.Language;
  beforeAll(() => {
    const { result } = renderHook(() => Input.useLanguage());
    if (result.current == null) throw new Error("time language not loaded");
    language = result.current;
  });

  it("should read a day and a span of times", () => {
    expect(Input.suggestTimeRange("Yesterday from 1pm to 2pm", language, NOW)).toEqual(
      new TimeRange(local(22, 13), local(22, 14)),
    );
    expect(Input.suggestTimeRange("today 9:30-10:15am", language, NOW)).toEqual(
      new TimeRange(local(23, 9, 30), local(23, 10, 15)),
    );
  });

  it("should cover all of a day it names, up to the next midnight", () => {
    expect(Input.suggestTimeRange("yesterday", language, NOW)).toEqual(
      new TimeRange(local(22, 0), local(23, 0)),
    );
  });

  it("should read a start and a length", () => {
    expect(Input.suggestTimeRange("2h ago for 1h", language, NOW)).toEqual(
      new TimeRange(NOW.sub(TimeSpan.hours(2)), NOW.sub(TimeSpan.hours(1))),
    );
    expect(
      Input.suggestTimeRange("yesterday 1pm for half an hour", language, NOW),
    ).toEqual(new TimeRange(local(22, 13), local(22, 13, 30)));
  });

  it("should read a start until now", () => {
    expect(Input.suggestTimeRange("since 1pm", language, NOW)).toEqual(
      new TimeRange(local(23, 13), NOW),
    );
  });

  it("should read two ends", () => {
    expect(Input.suggestTimeRange("3h ago to 1h ago", language, NOW)).toEqual(
      new TimeRange(NOW.sub(TimeSpan.hours(3)), NOW.sub(TimeSpan.hours(1))),
    );
    expect(Input.suggestTimeRange("30m ago to now", language, NOW)).toEqual(
      new TimeRange(NOW.sub(TimeSpan.minutes(30)), NOW),
    );
  });

  it("should read a range pasted from a window's tooltip", () => {
    const text = "2026-08-22 13:00:00.000 → 2026-08-22 14:00:00.000";
    expect(Input.suggestTimeRange(text, language, NOW)).toEqual(
      new TimeRange(local(22, 13), local(22, 14)),
    );
  });

  it("should place a time of day alone on the day of the other end", () => {
    expect(Input.suggestTimeRange("13:00 to 14:00 yesterday", language, NOW)).toEqual(
      new TimeRange(local(22, 13), local(22, 14)),
    );
    expect(Input.suggestTimeRange("yesterday 10pm to 2am", language, NOW)).toEqual(
      new TimeRange(local(22, 22), local(23, 2)),
    );
    expect(Input.suggestTimeRange("10pm to 2am yesterday", language, NOW)).toEqual(
      new TimeRange(local(21, 22), local(22, 2)),
    );
  });

  it("should place a time of day alone before now", () => {
    expect(Input.suggestTimeRange("since 3pm", language, NOW)).toEqual(
      new TimeRange(local(22, 15), NOW),
    );
    expect(Input.suggestTimeRange("10pm to 2am", language, NOW)).toEqual(
      new TimeRange(local(22, 22), local(23, 2)),
    );
    expect(Input.suggestTimeRange("1pm to 3pm", language, NOW)).toEqual(
      new TimeRange(local(23, 13), local(23, 15)),
    );
    expect(Input.suggestTimeRange("22:00", language, NOW)).toEqual(
      new TimeRange(local(22, 22), local(22, 23)),
    );
  });

  it("should end on the last day of a span of days", () => {
    expect(Input.suggestTimeRange("aug 17 through aug 19", language, NOW)).toEqual(
      new TimeRange(local(17, 0), local(20, 0)),
    );
  });

  it("should read nothing when other words name a range", () => {
    expect(Input.suggestTimeRange("monday test", language, NOW)).toBeNull();
  });

  it("should read an instant alone as the hour after it", () => {
    expect(Input.suggestTimeRange("3h ago", language, NOW)).toEqual(
      new TimeRange(NOW.sub(TimeSpan.hours(3)), NOW.sub(TimeSpan.hours(2))),
    );
    expect(Input.suggestTimeRange("yesterday 1pm", language, NOW)).toEqual(
      new TimeRange(local(22, 13), local(22, 14)),
    );
  });

  it("should end the hour after an instant at now", () => {
    expect(Input.suggestTimeRange("10m ago", language, NOW)).toEqual(
      new TimeRange(NOW.sub(TimeSpan.minutes(10)), NOW),
    );
  });

  it("should read nothing for an instant after now", () => {
    expect(Input.suggestTimeRange("in 2 hours", language, NOW)).toBeNull();
  });

  it("should read nothing for text that names no time", () => {
    expect(Input.suggestTimeRange("purple", language, NOW)).toBeNull();
    expect(Input.suggestTimeRange("", language, NOW)).toBeNull();
  });
});
