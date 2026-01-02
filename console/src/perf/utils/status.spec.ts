// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { getThresholdStatus } from "@/perf/utils/status";

describe("status", () => {
  describe("getThresholdStatus", () => {
    // Tests use CPU-like thresholds: warn=85, error=95
    it("should return error when value exceeds error threshold", () => {
      expect(getThresholdStatus(100, 85, 95)).toBe("error");
      expect(getThresholdStatus(96, 85, 95)).toBe("error");
    });

    it("should return warning when value exceeds warning threshold", () => {
      expect(getThresholdStatus(90, 85, 95)).toBe("warning");
      expect(getThresholdStatus(86, 85, 95)).toBe("warning");
    });

    it("should return undefined when value is below thresholds", () => {
      expect(getThresholdStatus(80, 85, 95)).toBeUndefined();
      expect(getThresholdStatus(0, 85, 95)).toBeUndefined();
    });

    it("should return undefined for null values", () => {
      expect(getThresholdStatus(null, 85, 95)).toBeUndefined();
    });

    describe("inverted thresholds", () => {
      // Tests use FPS-like thresholds: warn=59, error=10
      it("should return error when value is below error threshold", () => {
        expect(getThresholdStatus(5, 59, 10, true)).toBe("error");
        expect(getThresholdStatus(9, 59, 10, true)).toBe("error");
      });

      it("should return warning when value is below warning threshold", () => {
        expect(getThresholdStatus(30, 59, 10, true)).toBe("warning");
        expect(getThresholdStatus(58, 59, 10, true)).toBe("warning");
      });

      it("should return undefined when value is above thresholds", () => {
        expect(getThresholdStatus(60, 59, 10, true)).toBeUndefined();
        expect(getThresholdStatus(120, 59, 10, true)).toBeUndefined();
      });

      it("should return undefined for null values", () => {
        expect(getThresholdStatus(null, 59, 10, true)).toBeUndefined();
      });
    });
  });
});
