// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { getAvgPeakStatus, getThresholdStatus } from "@/perf/utils/status";

describe("status", () => {
  describe("getThresholdStatus", () => {
    it("should return error when value exceeds error threshold", () => {
      expect(getThresholdStatus(100, 50, 80)).toBe("error");
      expect(getThresholdStatus(81, 50, 80)).toBe("error");
    });

    it("should return warning when value exceeds warning threshold", () => {
      expect(getThresholdStatus(75, 50, 80)).toBe("warning");
      expect(getThresholdStatus(51, 50, 80)).toBe("warning");
    });

    it("should return undefined when value is below thresholds", () => {
      expect(getThresholdStatus(25, 50, 80)).toBeUndefined();
      expect(getThresholdStatus(0, 50, 80)).toBeUndefined();
    });

    it("should return undefined for null values", () => {
      expect(getThresholdStatus(null, 50, 80)).toBeUndefined();
    });

    describe("inverted thresholds", () => {
      it("should return error when value is below error threshold", () => {
        expect(getThresholdStatus(20, 50, 30, true)).toBe("error");
        expect(getThresholdStatus(29, 50, 30, true)).toBe("error");
      });

      it("should return warning when value is below warning threshold", () => {
        expect(getThresholdStatus(40, 50, 30, true)).toBe("warning");
        expect(getThresholdStatus(49, 50, 30, true)).toBe("warning");
      });

      it("should return undefined when value is above thresholds", () => {
        expect(getThresholdStatus(60, 50, 30, true)).toBeUndefined();
        expect(getThresholdStatus(100, 50, 30, true)).toBeUndefined();
      });

      it("should return undefined for null values", () => {
        expect(getThresholdStatus(null, 50, 30, true)).toBeUndefined();
      });
    });
  });

  describe("getAvgPeakStatus", () => {
    it("should return warning when avg exceeds threshold", () => {
      expect(getAvgPeakStatus(60, 40, 50, 80)).toBe("warning");
      expect(getAvgPeakStatus(51, 40, 50, 80)).toBe("warning");
    });

    it("should return warning when peak exceeds threshold", () => {
      expect(getAvgPeakStatus(40, 90, 50, 80)).toBe("warning");
      expect(getAvgPeakStatus(40, 81, 50, 80)).toBe("warning");
    });

    it("should return warning when both exceed thresholds", () => {
      expect(getAvgPeakStatus(60, 90, 50, 80)).toBe("warning");
    });

    it("should return undefined when both are below thresholds", () => {
      expect(getAvgPeakStatus(40, 70, 50, 80)).toBeUndefined();
      expect(getAvgPeakStatus(0, 0, 50, 80)).toBeUndefined();
    });

    it("should handle null values as zero", () => {
      expect(getAvgPeakStatus(null, null, 50, 80)).toBeUndefined();
      expect(getAvgPeakStatus(null, 90, 50, 80)).toBe("warning");
      expect(getAvgPeakStatus(60, null, 50, 80)).toBe("warning");
    });
  });
});
