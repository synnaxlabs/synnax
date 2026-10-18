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

import { parseSpan } from "@/feature/lineplot/CustomRangeInput";

describe("parseSpan", () => {
  it("should read back every unit the input displays", () => {
    const spans = [
      TimeSpan.days(2),
      TimeSpan.hours(3),
      TimeSpan.minutes(45),
      TimeSpan.seconds(12),
      TimeSpan.milliseconds(500),
      TimeSpan.microseconds(25),
      TimeSpan.nanoseconds(7),
      TimeSpan.days(1).add(TimeSpan.minutes(46)).add(TimeSpan.microseconds(3)),
    ];
    spans.forEach((span) => expect(parseSpan(span.toString())).toEqual(Number(span)));
  });

  it("should parse short minute forms typed by hand", () => {
    expect(parseSpan("1h 30m")).toEqual(Number(TimeSpan.minutes(90)));
  });

  it("should reject zero durations", () => {
    expect(parseSpan("0s")).toBeNull();
    expect(parseSpan("0h 0min")).toBeNull();
  });

  it("should reject text that is not a duration", () => {
    expect(parseSpan("")).toBeNull();
    expect(parseSpan("45")).toBeNull();
    expect(parseSpan("-5min")).toBeNull();
    expect(parseSpan("5min!")).toBeNull();
  });
});
