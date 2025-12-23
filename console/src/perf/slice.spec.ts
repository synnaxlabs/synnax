// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  ZERO_CPU_REPORT,
  ZERO_FPS_REPORT,
  ZERO_GPU_REPORT,
  ZERO_LEAK_REPORT,
} from "@/perf/analyzer/types";
import { type MacroResult } from "@/perf/macros/types";
import {
  addMacroResult,
  pause,
  reducer,
  reset,
  resume,
  setConfig,
  setCpuReport,
  setError,
  setFpsReport,
  setGpuReport,
  setLeakReport,
  setRangeKey,
  setRangeStartTime,
  type SliceState,
  start,
  stop,
  ZERO_SLICE_STATE,
} from "@/perf/slice";
import { type FpsReport, type LeakReport } from "@/perf/types";

describe("perf slice", () => {
  const mockNow = 1000;
  beforeEach(() => {
    vi.spyOn(performance, "now").mockReturnValue(mockNow);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe("start action", () => {
    it("should set status to running", () => {
      const state = reducer(ZERO_SLICE_STATE, start(undefined));
      expect(state.status).toBe("running");
    });

    it("should set start time", () => {
      const state = reducer(ZERO_SLICE_STATE, start(undefined));
      expect(state.startTime).toBe(mockNow);
    });

    it("should reset end time, range key, and results", () => {
      const initialState: SliceState = {
        ...ZERO_SLICE_STATE,
        endTime: 500,
        rangeKey: "old-key",
        macroResults: [{ macroType: "test", startTime: 0, endTime: 0, durationMs: 0 }],
      };
      const state = reducer(initialState, start(undefined));

      expect(state.endTime).toBeNull();
      expect(state.rangeKey).toBeNull();
      expect(state.rangeStartTime).toBeNull();
      expect(state.macroResults).toEqual([]);
    });

    it("should reset all reports", () => {
      const initialState: SliceState = {
        ...ZERO_SLICE_STATE,
        leakReport: { ...ZERO_LEAK_REPORT, severity: "error" },
        fpsReport: { ...ZERO_FPS_REPORT, peakSeverity: "warning" },
      };
      const state = reducer(initialState, start(undefined));

      expect(state.leakReport).toEqual(ZERO_LEAK_REPORT);
      expect(state.fpsReport).toEqual(ZERO_FPS_REPORT);
      expect(state.cpuReport).toEqual(ZERO_CPU_REPORT);
      expect(state.gpuReport).toEqual(ZERO_GPU_REPORT);
    });

    it("should clear error", () => {
      const initialState: SliceState = {
        ...ZERO_SLICE_STATE,
        error: "Previous error",
      };
      const state = reducer(initialState, start(undefined));
      expect(state.error).toBeNull();
    });

    it("should merge partial config", () => {
      const state = reducer(ZERO_SLICE_STATE, start({ durationMinutes: 60 }));
      expect(state.config.durationMinutes).toBe(60);
    });

    it("should merge nested config objects", () => {
      const state = reducer(
        ZERO_SLICE_STATE,
        start({
          metricsConfig: { sampleIntervalMs: 500 },
          macroConfig: { iterations: 5 },
        }),
      );
      expect(state.config.metricsConfig.sampleIntervalMs).toBe(500);
      expect(state.config.macroConfig.iterations).toBe(5);
    });
  });

  describe("stop action", () => {
    it("should set status to paused when running", () => {
      const initialState: SliceState = { ...ZERO_SLICE_STATE, status: "running" };
      const state = reducer(initialState, stop());
      expect(state.status).toBe("paused");
    });

    it("should not change status if already paused", () => {
      const initialState: SliceState = { ...ZERO_SLICE_STATE, status: "paused" };
      const state = reducer(initialState, stop());
      expect(state.status).toBe("paused");
    });

    it("should not change status if idle", () => {
      const state = reducer(ZERO_SLICE_STATE, stop());
      expect(state.status).toBe("idle");
    });
  });

  describe("pause action", () => {
    it("should set status to paused when running", () => {
      const initialState: SliceState = { ...ZERO_SLICE_STATE, status: "running" };
      const state = reducer(initialState, pause());
      expect(state.status).toBe("paused");
    });

    it("should not change status if not running", () => {
      const state = reducer(ZERO_SLICE_STATE, pause());
      expect(state.status).toBe("idle");
    });
  });

  describe("resume action", () => {
    it("should set status to running when paused", () => {
      const initialState: SliceState = { ...ZERO_SLICE_STATE, status: "paused" };
      const state = reducer(initialState, resume());
      expect(state.status).toBe("running");
    });

    it("should not change status if not paused", () => {
      const initialState: SliceState = { ...ZERO_SLICE_STATE, status: "running" };
      const state = reducer(initialState, resume());
      expect(state.status).toBe("running");
    });
  });

  describe("addMacroResult action", () => {
    it("should add macro result to array", () => {
      const result: MacroResult = {
        macroType: "createLinePlot",
        startTime: 100,
        endTime: 200,
        durationMs: 100,
      };
      const state = reducer(ZERO_SLICE_STATE, addMacroResult(result));
      expect(state.macroResults).toHaveLength(1);
      expect(state.macroResults[0]).toEqual(result);
    });

    it("should append to existing results", () => {
      const firstResult: MacroResult = {
        macroType: "createLinePlot",
        startTime: 100,
        endTime: 200,
        durationMs: 100,
      };
      const secondResult: MacroResult = {
        macroType: "panZoomPlot",
        startTime: 300,
        endTime: 400,
        durationMs: 100,
      };

      let state = reducer(ZERO_SLICE_STATE, addMacroResult(firstResult));
      state = reducer(state, addMacroResult(secondResult));

      expect(state.macroResults).toHaveLength(2);
      expect(state.macroResults[1]).toEqual(secondResult);
    });
  });

  describe("report setters", () => {
    it("should set leak report", () => {
      const report: LeakReport = {
        severity: "warning",
        heapStartMB: 100,
        heapEndMB: 150,
        heapGrowthMB: 50,
        heapGrowthPercent: 50,
        trend: "increasing",
        snapshotCount: 10,
      };
      const state = reducer(ZERO_SLICE_STATE, setLeakReport(report));
      expect(state.leakReport).toEqual(report);
    });

    it("should set FPS report", () => {
      const report: FpsReport = {
        peakSeverity: "warning",
        avgSeverity: "none",
        startFps: 60,
        endFps: 45,
        changePercent: 25,
      };
      const state = reducer(ZERO_SLICE_STATE, setFpsReport(report));
      expect(state.fpsReport).toEqual(report);
    });

    it("should set CPU report", () => {
      const report = {
        peakSeverity: "error" as const,
        avgSeverity: "warning" as const,
        avgPercent: 70,
        maxPercent: 98,
        startPercent: 50,
        endPercent: 90,
      };
      const state = reducer(ZERO_SLICE_STATE, setCpuReport(report));
      expect(state.cpuReport).toEqual(report);
    });

    it("should set GPU report", () => {
      const report = {
        peakSeverity: "none" as const,
        avgSeverity: "none" as const,
        avgPercent: 30,
        maxPercent: 50,
        startPercent: 25,
        endPercent: 35,
      };
      const state = reducer(ZERO_SLICE_STATE, setGpuReport(report));
      expect(state.gpuReport).toEqual(report);
    });
  });

  describe("range actions", () => {
    it("should set range key", () => {
      const state = reducer(ZERO_SLICE_STATE, setRangeKey("range-123"));
      expect(state.rangeKey).toBe("range-123");
    });

    it("should allow null range key", () => {
      const initialState: SliceState = { ...ZERO_SLICE_STATE, rangeKey: "old-key" };
      const state = reducer(initialState, setRangeKey(null));
      expect(state.rangeKey).toBeNull();
    });

    it("should set range start time", () => {
      const state = reducer(ZERO_SLICE_STATE, setRangeStartTime(12345));
      expect(state.rangeStartTime).toBe(12345);
    });

    it("should allow null range start time", () => {
      const initialState: SliceState = { ...ZERO_SLICE_STATE, rangeStartTime: 12345 };
      const state = reducer(initialState, setRangeStartTime(null));
      expect(state.rangeStartTime).toBeNull();
    });
  });

  describe("setError action", () => {
    it("should set error status and message", () => {
      const state = reducer(ZERO_SLICE_STATE, setError("Something went wrong"));
      expect(state.status).toBe("error");
      expect(state.error).toBe("Something went wrong");
    });

    it("should set end time", () => {
      const state = reducer(ZERO_SLICE_STATE, setError("Error"));
      expect(state.endTime).toBe(mockNow);
    });
  });

  describe("reset action", () => {
    it("should reset to initial state", () => {
      const initialState: SliceState = {
        ...ZERO_SLICE_STATE,
        status: "running",
        startTime: 1000,
        error: "Error",
        macroResults: [{ macroType: "test", startTime: 0, endTime: 0, durationMs: 0 }],
      };
      const state = reducer(initialState, reset());
      expect(state).toEqual(ZERO_SLICE_STATE);
    });
  });

  describe("setConfig action", () => {
    it("should merge partial config", () => {
      const state = reducer(ZERO_SLICE_STATE, setConfig({ durationMinutes: 120 }));
      expect(state.config.durationMinutes).toBe(120);
    });

    it("should merge nested metricsConfig", () => {
      const state = reducer(
        ZERO_SLICE_STATE,
        setConfig({
          metricsConfig: { sampleIntervalMs: 2000 },
        }),
      );
      expect(state.config.metricsConfig.sampleIntervalMs).toBe(2000);
      // Other metricsConfig properties should be preserved
      expect(state.config.metricsConfig.enableLongTaskObserver).toBeDefined();
    });

    it("should merge nested macroConfig", () => {
      const state = reducer(
        ZERO_SLICE_STATE,
        setConfig({
          macroConfig: { iterations: 10 },
        }),
      );
      expect(state.config.macroConfig.iterations).toBe(10);
      // Other macroConfig properties should be preserved
      expect(state.config.macroConfig.delayBetweenIterationsMs).toBeDefined();
    });
  });
});
