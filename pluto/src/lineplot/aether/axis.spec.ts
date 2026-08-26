// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { afterEach, describe, expect, it, vi } from "vitest";

import { autoBounds, emptyBounds } from "@/lineplot/aether/axis";

describe("axis", () => {
  afterEach(() => {
    vi.useRealTimers();
  });

  describe("emptyBounds", () => {
    it("should return decimal bounds for a linear axis", () => {
      expect(emptyBounds("linear")).toStrictEqual({ lower: 0, upper: 1 });
    });

    it("should compute the time window at call time", () => {
      vi.useFakeTimers();
      vi.setSystemTime(new Date("2030-01-02T03:04:05Z"));
      const now = TimeStamp.now();
      const b = emptyBounds("time");
      expect(b.lower).toBe(Number(now.valueOf()));
      expect(b.upper).toBe(Number(now.add(TimeSpan.HOUR).valueOf()));
    });

    it("should track a moving clock across calls", () => {
      vi.useFakeTimers();
      vi.setSystemTime(new Date("2030-01-02T03:04:05Z"));
      const first = emptyBounds("time");
      vi.setSystemTime(new Date("2030-01-02T04:04:05Z"));
      const second = emptyBounds("time");
      expect(second.lower).toBeGreaterThan(first.lower);
      expect(second.lower).toBe(Number(TimeStamp.now().valueOf()));
    });
  });

  describe("autoBounds", () => {
    it("should pad bounds by the given padding", () => {
      expect(autoBounds([{ lower: 0, upper: 10 }], 0.1, "linear")).toStrictEqual({
        lower: -1,
        upper: 11,
      });
    });

    it("should expand equal bounds by one", () => {
      expect(autoBounds([{ lower: 5, upper: 5 }], 0.1, "linear")).toStrictEqual({
        lower: 4,
        upper: 6,
      });
    });

    it("should ignore non-finite bounds", () => {
      const b = [
        { lower: Infinity, upper: -Infinity },
        { lower: 0, upper: 10 },
      ];
      expect(autoBounds(b, 0, "linear")).toStrictEqual({ lower: 0, upper: 10 });
    });

    it("should fall back to decimal bounds for an empty linear axis", () => {
      const b = [{ lower: Infinity, upper: -Infinity }];
      expect(autoBounds(b, 0.1, "linear")).toStrictEqual({ lower: 0, upper: 1 });
    });

    it("should fall back to a now-anchored hour for an empty time axis", () => {
      vi.useFakeTimers();
      vi.setSystemTime(new Date("2030-01-02T03:04:05Z"));
      const now = TimeStamp.now();
      const b = autoBounds([], 0.1, "time");
      expect(b.lower).toBe(Number(now.valueOf()));
      expect(b.upper).toBe(Number(now.add(TimeSpan.HOUR).valueOf()));
    });
  });
});
