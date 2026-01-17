// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import {
  formatAge,
  formatDelta,
  formatDuration,
  formatMB,
  formatPair,
  formatPercent,
  formatPercentChange,
  formatTime,
  NA,
  NO_DATA,
  truncateEndpoint,
} from "@/perf/utils/formatting";

describe("formatting", () => {
  describe("formatTime", () => {
    it("should format time as MM:SS", () => {
      expect(formatTime(0)).toBe("00:00");
      expect(formatTime(59)).toBe("00:59");
      expect(formatTime(60)).toBe("01:00");
      expect(formatTime(125)).toBe("02:05");
      expect(formatTime(3599)).toBe("59:59");
    });

    it("should handle negative values", () => {
      expect(formatTime(-10)).toBe("-1:-10");
    });
  });

  describe("formatDuration", () => {
    it("should format durations under 1 second as milliseconds", () => {
      expect(formatDuration(0)).toBe("0.0 ms");
      expect(formatDuration(1.5)).toBe("1.5 ms");
      expect(formatDuration(999)).toBe("999.0 ms");
    });

    it("should format durations over 1 second as seconds", () => {
      expect(formatDuration(1000)).toBe("1.0 s");
      expect(formatDuration(1500)).toBe("1.5 s");
      expect(formatDuration(5432)).toBe("5.4 s");
    });
  });

  describe("formatAge", () => {
    it("should format age under 1 second as milliseconds", () => {
      expect(formatAge(0)).toBe("0ms ago");
      expect(formatAge(500)).toBe("500ms ago");
      expect(formatAge(999)).toBe("999ms ago");
    });

    it("should format age under 1 minute as seconds", () => {
      expect(formatAge(1000)).toBe("1s ago");
      expect(formatAge(5500)).toBe("5s ago");
      expect(formatAge(59999)).toBe("59s ago");
    });

    it("should format age over 1 minute as minutes", () => {
      expect(formatAge(60000)).toBe("1m ago");
      expect(formatAge(125000)).toBe("2m ago");
      expect(formatAge(600000)).toBe("10m ago");
    });
  });

  describe("formatPercent", () => {
    it("should format numbers as percentages", () => {
      expect(formatPercent(0)).toBe("0.0%");
      expect(formatPercent(25.5)).toBe("25.5%");
      expect(formatPercent(100)).toBe("100.0%");
    });

    it("should handle null values", () => {
      expect(formatPercent(null)).toBe("N/A");
    });
  });

  describe("formatMB", () => {
    it("should format numbers as megabytes", () => {
      expect(formatMB(0)).toBe("0.0 MB");
      expect(formatMB(128.5)).toBe("128.5 MB");
      expect(formatMB(1024)).toBe("1024.0 MB");
    });

    it("should handle null values", () => {
      expect(formatMB(null)).toBe("N/A");
    });
  });

  describe("formatPair", () => {
    it("should format pairs of values with separator", () => {
      expect(formatPair(10.5, 20.3)).toBe("10.5 / 20.3");
      expect(formatPair(10.5, 20.3, "%")).toBe("10.5 / 20.3%");
      expect(formatPair(10.5, 20.3, " MB")).toBe("10.5 / 20.3 MB");
    });

    it("should handle null values individually", () => {
      expect(formatPair(null, 20.3)).toBe("— / 20.3");
      expect(formatPair(10.5, null)).toBe("10.5 / —");
      expect(formatPair(null, null)).toBe("—");
    });
  });

  describe("formatDelta", () => {
    it("should format positive deltas with plus sign", () => {
      expect(formatDelta(10, 15)).toBe("+5.0");
      expect(formatDelta(10, 15, "%")).toBe("+5.0%");
    });

    it("should format negative deltas with minus sign", () => {
      expect(formatDelta(15, 10)).toBe("-5.0");
      expect(formatDelta(15, 10, "%")).toBe("-5.0%");
    });

    it("should format zero delta with plus sign", () => {
      expect(formatDelta(10, 10)).toBe("+0.0");
    });

    it("should handle null values", () => {
      expect(formatDelta(null, 10)).toBe("—");
      expect(formatDelta(10, null)).toBe("—");
      expect(formatDelta(null, null)).toBe("—");
    });
  });

  describe("formatPercentChange", () => {
    it("should format positive changes with plus sign", () => {
      expect(formatPercentChange(5)).toBe("+5.0%");
      expect(formatPercentChange(10.5)).toBe("+10.5%");
    });

    it("should format negative changes with minus sign", () => {
      expect(formatPercentChange(-5)).toBe("-5.0%");
      expect(formatPercentChange(-10.5)).toBe("-10.5%");
    });

    it("should format zero with plus sign", () => {
      expect(formatPercentChange(0)).toBe("+0.0%");
    });

    it("should invert sign when invertSign is true", () => {
      expect(formatPercentChange(5, true)).toBe("-5.0%");
      expect(formatPercentChange(-5, true)).toBe("+5.0%");
      expect(formatPercentChange(0, true)).toBe("+0.0%");
    });

    it("should handle null values", () => {
      expect(formatPercentChange(null)).toBe("—");
      expect(formatPercentChange(null, true)).toBe("—");
    });
  });

  describe("truncateEndpoint", () => {
    it("should truncate endpoints to last N segments", () => {
      expect(truncateEndpoint("/api/v1/channels/123")).toBe("/channels/123");
      expect(truncateEndpoint("/api/v1/channels/123", 1)).toBe("/123");
      expect(truncateEndpoint("/api/v1/channels/123", 3)).toBe("/v1/channels/123");
    });

    it("should return full path if segments <= requested", () => {
      expect(truncateEndpoint("/api")).toBe("/api");
      expect(truncateEndpoint("/api/v1")).toBe("/api/v1");
    });

    it("should handle paths without leading slash", () => {
      expect(truncateEndpoint("api/v1/channels/123")).toBe("/channels/123");
    });
  });

  describe("constants", () => {
    it("should export NA constant", () => {
      expect(NA).toBe("N/A");
    });

    it("should export NO_DATA constant", () => {
      expect(NO_DATA).toBe("—");
    });
  });
});
