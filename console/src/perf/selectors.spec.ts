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
  ZERO_CPU_REPORT,
  ZERO_FPS_REPORT,
  ZERO_GPU_REPORT,
  ZERO_LEAK_REPORT,
} from "@/perf/analyzer/types";
import {
  selectCpuReport,
  selectError,
  selectFpsReport,
  selectGpuReport,
  selectIsRunning,
  selectLeakReport,
  selectMacroResults,
  selectRangeKey,
  selectRangeStartTime,
  selectSlice,
  selectStatus,
} from "@/perf/selectors";
import { SLICE_NAME, ZERO_SLICE_STATE } from "@/perf/slice";
import { type RootState } from "@/store";

const createMockState = (overrides: Partial<typeof ZERO_SLICE_STATE> = {}): RootState =>
  ({
    [SLICE_NAME]: {
      ...ZERO_SLICE_STATE,
      ...overrides,
    },
  }) as unknown as RootState;

describe("selectors", () => {
  describe("selectSlice", () => {
    it("should return the perf slice", () => {
      const state = createMockState();
      const result = selectSlice(state);
      expect(result).toEqual(ZERO_SLICE_STATE);
    });
  });

  describe("selectStatus", () => {
    it("should return idle status by default", () => {
      const state = createMockState();
      expect(selectStatus(state)).toBe("idle");
    });

    it("should return running status when running", () => {
      const state = createMockState({ status: "running" });
      expect(selectStatus(state)).toBe("running");
    });

    it("should return paused status when paused", () => {
      const state = createMockState({ status: "paused" });
      expect(selectStatus(state)).toBe("paused");
    });

    it("should return error status when error", () => {
      const state = createMockState({ status: "error" });
      expect(selectStatus(state)).toBe("error");
    });
  });

  describe("selectIsRunning", () => {
    it("should return false when idle", () => {
      const state = createMockState({ status: "idle" });
      expect(selectIsRunning(state)).toBe(false);
    });

    it("should return true when running", () => {
      const state = createMockState({ status: "running" });
      expect(selectIsRunning(state)).toBe(true);
    });

    it("should return false when paused", () => {
      const state = createMockState({ status: "paused" });
      expect(selectIsRunning(state)).toBe(false);
    });
  });

  describe("selectMacroResults", () => {
    it("should return empty array by default", () => {
      const state = createMockState();
      expect(selectMacroResults(state)).toEqual([]);
    });

    it("should return macro results", () => {
      const results = [
        { macroType: "test", startTime: 0, endTime: 100, durationMs: 100 },
      ];
      const state = createMockState({ macroResults: results as any });
      expect(selectMacroResults(state)).toEqual(results);
    });
  });

  describe("selectLeakReport", () => {
    it("should return default leak report", () => {
      const state = createMockState();
      expect(selectLeakReport(state)).toEqual(ZERO_LEAK_REPORT);
    });

    it("should return updated leak report", () => {
      const report = { ...ZERO_LEAK_REPORT, severity: "warning" as const };
      const state = createMockState({ leakReport: report });
      expect(selectLeakReport(state).severity).toBe("warning");
    });
  });

  describe("selectFpsReport", () => {
    it("should return default FPS report", () => {
      const state = createMockState();
      expect(selectFpsReport(state)).toEqual(ZERO_FPS_REPORT);
    });

    it("should return updated FPS report", () => {
      const report = { ...ZERO_FPS_REPORT, startFps: 60 };
      const state = createMockState({ fpsReport: report });
      expect(selectFpsReport(state).startFps).toBe(60);
    });
  });

  describe("selectCpuReport", () => {
    it("should return default CPU report", () => {
      const state = createMockState();
      expect(selectCpuReport(state)).toEqual(ZERO_CPU_REPORT);
    });

    it("should return updated CPU report", () => {
      const report = { ...ZERO_CPU_REPORT, avgPercent: 50 };
      const state = createMockState({ cpuReport: report });
      expect(selectCpuReport(state).avgPercent).toBe(50);
    });
  });

  describe("selectGpuReport", () => {
    it("should return default GPU report", () => {
      const state = createMockState();
      expect(selectGpuReport(state)).toEqual(ZERO_GPU_REPORT);
    });

    it("should return updated GPU report", () => {
      const report = { ...ZERO_GPU_REPORT, maxPercent: 85 };
      const state = createMockState({ gpuReport: report });
      expect(selectGpuReport(state).maxPercent).toBe(85);
    });
  });

  describe("selectError", () => {
    it("should return null by default", () => {
      const state = createMockState();
      expect(selectError(state)).toBeNull();
    });

    it("should return error message when set", () => {
      const state = createMockState({ error: "Something went wrong" });
      expect(selectError(state)).toBe("Something went wrong");
    });
  });

  describe("selectRangeKey", () => {
    it("should return null by default", () => {
      const state = createMockState();
      expect(selectRangeKey(state)).toBeNull();
    });

    it("should return range key when set", () => {
      const state = createMockState({ rangeKey: "range-123" });
      expect(selectRangeKey(state)).toBe("range-123");
    });
  });

  describe("selectRangeStartTime", () => {
    it("should return null by default", () => {
      const state = createMockState();
      expect(selectRangeStartTime(state)).toBeNull();
    });

    it("should return range start time when set", () => {
      const state = createMockState({ rangeStartTime: 1234567890 });
      expect(selectRangeStartTime(state)).toBe(1234567890);
    });
  });
});
