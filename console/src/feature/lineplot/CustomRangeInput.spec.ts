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
  describe("valid durations", () => {
    it("should parse every unit", () => {
      expect(parseSpan("5us")).toEqual(Number(TimeSpan.microseconds(5)));
      expect(parseSpan("5µs")).toEqual(Number(TimeSpan.microseconds(5)));
      expect(parseSpan("5ms")).toEqual(Number(TimeSpan.milliseconds(5)));
      expect(parseSpan("5s")).toEqual(Number(TimeSpan.seconds(5)));
      expect(parseSpan("5m")).toEqual(Number(TimeSpan.minutes(5)));
      expect(parseSpan("5h")).toEqual(Number(TimeSpan.hours(5)));
      expect(parseSpan("5d")).toEqual(Number(TimeSpan.days(5)));
      expect(parseSpan("5w")).toEqual(Number(TimeSpan.days(35)));
      expect(parseSpan("5mo")).toEqual(Number(TimeSpan.days(150)));
      expect(parseSpan("5y")).toEqual(Number(TimeSpan.days(1825)));
    });

    it("should sum multiple tokens", () => {
      expect(parseSpan("1h 30m")).toEqual(Number(TimeSpan.minutes(90)));
      expect(parseSpan("1d2h3m4s")).toEqual(
        Number(
          TimeSpan.days(1)
            .add(TimeSpan.hours(2))
            .add(TimeSpan.minutes(3))
            .add(TimeSpan.seconds(4)),
        ),
      );
    });

    it("should parse decimal values", () => {
      expect(parseSpan("1.5h")).toEqual(Number(TimeSpan.minutes(90)));
      expect(parseSpan("0.5s")).toEqual(Number(TimeSpan.milliseconds(500)));
      expect(parseSpan(".5s")).toEqual(Number(TimeSpan.milliseconds(500)));
    });

    it("should parse a long token sequence", () => {
      expect(parseSpan("1h ".repeat(50))).toEqual(Number(TimeSpan.hours(50)));
    });

    it("should parse mixed tokens with bare leading dots", () => {
      expect(parseSpan(".5h 30m")).toEqual(Number(TimeSpan.hours(1)));
      expect(parseSpan("1h .5m")).toEqual(
        Number(TimeSpan.hours(1).add(TimeSpan.seconds(30))),
      );
      expect(parseSpan(".5h.25m")).toEqual(
        Number(TimeSpan.minutes(30).add(TimeSpan.seconds(15))),
      );
    });

    it("should allow whitespace between tokens and around the input", () => {
      expect(parseSpan("1h   30m")).toEqual(Number(TimeSpan.minutes(90)));
      expect(parseSpan(" 1h30m 45s ")).toEqual(
        Number(TimeSpan.minutes(90).add(TimeSpan.seconds(45))),
      );
    });

    it("should ignore case and surrounding whitespace", () => {
      expect(parseSpan("  45M  ")).toEqual(Number(TimeSpan.minutes(45)));
      expect(parseSpan("10MS")).toEqual(Number(TimeSpan.milliseconds(10)));
    });

    it("should parse ms as milliseconds, not minutes and seconds", () => {
      expect(parseSpan("1ms")).toEqual(Number(TimeSpan.milliseconds(1)));
    });
  });

  describe("invalid durations", () => {
    it("should reject empty and blank input", () => {
      expect(parseSpan("")).toBeNull();
      expect(parseSpan("   ")).toBeNull();
    });

    it("should reject a number without a unit", () => {
      expect(parseSpan("45")).toBeNull();
      expect(parseSpan("1h 30")).toBeNull();
    });

    it("should reject a unit without a number", () => {
      expect(parseSpan("h")).toBeNull();
      expect(parseSpan("ms")).toBeNull();
    });

    it("should reject unknown units", () => {
      expect(parseSpan("5x")).toBeNull();
      expect(parseSpan("5ns")).toBeNull();
      expect(parseSpan("5 hours")).toBeNull();
      expect(parseSpan("5sec")).toBeNull();
    });

    it("should reject negative values", () => {
      expect(parseSpan("-5m")).toBeNull();
    });

    it("should reject zero durations", () => {
      expect(parseSpan("0s")).toBeNull();
      expect(parseSpan("0h 0m")).toBeNull();
    });

    it("should reject trailing garbage", () => {
      expect(parseSpan("5m!")).toBeNull();
      expect(parseSpan("5m foo")).toBeNull();
    });

    it("should reject a space between a number and its unit", () => {
      expect(parseSpan("5 m")).toBeNull();
      expect(parseSpan("1.5 h")).toBeNull();
      expect(parseSpan("1h 30 m")).toBeNull();
    });

    it("should reject a long valid prefix with an invalid suffix without hanging", () => {
      expect(parseSpan(`${"1h ".repeat(50)}!`)).toBeNull();
      expect(parseSpan(`${"1.5m  ".repeat(50)}5`)).toBeNull();
    });

    it("should reject bare dots and dotted tokens without units", () => {
      expect(parseSpan(".")).toBeNull();
      expect(parseSpan(".h")).toBeNull();
      expect(parseSpan("1h .5")).toBeNull();
      expect(parseSpan(".5 h")).toBeNull();
    });

    it("should reject malformed numbers", () => {
      expect(parseSpan("5.s")).toBeNull();
      expect(parseSpan("1..5s")).toBeNull();
      expect(parseSpan("1,5s")).toBeNull();
    });
  });
});
