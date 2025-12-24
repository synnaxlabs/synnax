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
import { THRESHOLDS } from "@/perf/constants";
import { ZERO_AGGREGATES } from "@/perf/metrics/buffer";
import { compileReport } from "@/perf/report/compiler";
import { type CompileInput } from "@/perf/report/types";

const createBaseInput = (overrides: Partial<CompileInput> = {}): CompileInput => ({
  samples: [],
  captured: {
    initialFPS: 60,
    finalFPS: 60,
    initialCPU: 10,
    finalCPU: 10,
    initialGPU: 10,
    finalGPU: 10,
    initialHeap: 100,
    finalHeap: 100,
  },
  aggregates: {
    ...ZERO_AGGREGATES,
    avgFps: 60,
    minFps: 60,
    maxFps: 60,
    avgCpu: 10,
    maxCpu: 10,
    avgGpu: 10,
    maxGpu: 10,
    minHeap: 100,
    maxHeap: 100,
  },
  analysisResults: {
    leak: ZERO_LEAK_REPORT,
    fps: ZERO_FPS_REPORT,
    cpu: ZERO_CPU_REPORT,
    gpu: ZERO_GPU_REPORT,
  },
  startTime: 0,
  endTime: 10000,
  ...overrides,
});

describe("compileReport", () => {
  describe("verdict determination", () => {
    it("should return Passed when no issues are detected", () => {
      const input = createBaseInput();
      const report = compileReport(input);
      expect(report.summary.verdict).toBe("Passed");
      expect(report.issues).toHaveLength(0);
    });

    it("should return Failed when a critical issue exists", () => {
      const input = createBaseInput({
        aggregates: {
          ...createBaseInput().aggregates,
          avgFps: THRESHOLDS.fps.error - 1,
        },
      });
      const report = compileReport(input);
      expect(report.summary.verdict).toBe("Failed");
    });

    it("should return Passed when only warnings exist", () => {
      const input = createBaseInput({
        aggregates: {
          ...createBaseInput().aggregates,
          avgFps: THRESHOLDS.fps.warn - 1,
        },
      });
      const report = compileReport(input);
      expect(report.summary.verdict).toBe("Passed");
      expect(report.issues.some((i) => i.severity === "warning")).toBe(true);
    });
  });

  describe("FPS issue detection", () => {
    it("should detect critical issue when avgFps is below error threshold", () => {
      const input = createBaseInput({
        aggregates: {
          ...createBaseInput().aggregates,
          avgFps: THRESHOLDS.fps.error - 1,
        },
      });
      const report = compileReport(input);
      const fpsIssue = report.issues.find((i) => i.category === "fps");
      expect(fpsIssue).toBeDefined();
      expect(fpsIssue?.severity).toBe("critical");
    });

    it("should detect warning when avgFps is below warn but above error threshold", () => {
      const input = createBaseInput({
        aggregates: {
          ...createBaseInput().aggregates,
          avgFps: THRESHOLDS.fps.warn - 1,
        },
      });
      const report = compileReport(input);
      const fpsIssue = report.issues.find((i) => i.category === "fps");
      expect(fpsIssue).toBeDefined();
      expect(fpsIssue?.severity).toBe("warning");
    });

    it("should detect critical FPS change when degradation exceeds error threshold", () => {
      const input = createBaseInput({
        analysisResults: {
          ...createBaseInput().analysisResults,
          fps: {
            ...ZERO_FPS_REPORT,
            peakSeverity: "error",
            changePercent: THRESHOLDS.fpsChange.error + 1,
          },
        },
      });
      const report = compileReport(input);
      const changeIssue = report.issues.find(
        (i) => i.category === "fps" && i.message.includes("dropped"),
      );
      expect(changeIssue).toBeDefined();
      expect(changeIssue?.severity).toBe("critical");
    });

    it("should detect warning FPS change when degradation exceeds warn threshold", () => {
      const input = createBaseInput({
        analysisResults: {
          ...createBaseInput().analysisResults,
          fps: {
            ...ZERO_FPS_REPORT,
            avgSeverity: "warning",
            changePercent: THRESHOLDS.fpsChange.warn + 1,
          },
        },
      });
      const report = compileReport(input);
      const changeIssue = report.issues.find(
        (i) => i.category === "fps" && i.message.includes("dropped"),
      );
      expect(changeIssue).toBeDefined();
      expect(changeIssue?.severity).toBe("warning");
    });
  });

  describe("CPU issue detection", () => {
    it("should detect critical issue when avgCpu exceeds error threshold", () => {
      const input = createBaseInput({
        aggregates: {
          ...createBaseInput().aggregates,
          avgCpu: THRESHOLDS.cpu.error + 1,
        },
      });
      const report = compileReport(input);
      const cpuIssue = report.issues.find((i) => i.category === "cpu");
      expect(cpuIssue).toBeDefined();
      expect(cpuIssue?.severity).toBe("critical");
    });

    it("should detect warning when avgCpu exceeds warn but not error threshold", () => {
      const input = createBaseInput({
        aggregates: {
          ...createBaseInput().aggregates,
          avgCpu: THRESHOLDS.cpu.warn + 1,
        },
      });
      const report = compileReport(input);
      const cpuIssue = report.issues.find((i) => i.category === "cpu");
      expect(cpuIssue).toBeDefined();
      expect(cpuIssue?.severity).toBe("warning");
    });
  });

  describe("GPU issue detection", () => {
    it("should detect critical issue when avgGpu exceeds error threshold", () => {
      const input = createBaseInput({
        aggregates: {
          ...createBaseInput().aggregates,
          avgGpu: THRESHOLDS.gpu.error + 1,
        },
      });
      const report = compileReport(input);
      const gpuIssue = report.issues.find((i) => i.category === "gpu");
      expect(gpuIssue).toBeDefined();
      expect(gpuIssue?.severity).toBe("critical");
    });

    it("should detect warning when avgGpu exceeds warn but not error threshold", () => {
      const input = createBaseInput({
        aggregates: {
          ...createBaseInput().aggregates,
          avgGpu: THRESHOLDS.gpu.warn + 1,
        },
      });
      const report = compileReport(input);
      const gpuIssue = report.issues.find((i) => i.category === "gpu");
      expect(gpuIssue).toBeDefined();
      expect(gpuIssue?.severity).toBe("warning");
    });
  });

  describe("memory issue detection", () => {
    it("should detect critical issue when heap growth exceeds error threshold", () => {
      const input = createBaseInput({
        analysisResults: {
          ...createBaseInput().analysisResults,
          leak: {
            ...ZERO_LEAK_REPORT,
            severity: "error",
            heapGrowthPercent: THRESHOLDS.heapGrowth.error + 1,
          },
        },
      });
      const report = compileReport(input);
      const memoryIssue = report.issues.find((i) => i.category === "memory");
      expect(memoryIssue).toBeDefined();
      expect(memoryIssue?.severity).toBe("critical");
    });

    it("should detect warning when heap growth exceeds warn but not error threshold", () => {
      const input = createBaseInput({
        analysisResults: {
          ...createBaseInput().analysisResults,
          leak: {
            ...ZERO_LEAK_REPORT,
            severity: "warning",
            heapGrowthPercent: THRESHOLDS.heapGrowth.warn + 1,
          },
        },
      });
      const report = compileReport(input);
      const memoryIssue = report.issues.find((i) => i.category === "memory");
      expect(memoryIssue).toBeDefined();
      expect(memoryIssue?.severity).toBe("warning");
    });

    it("should not detect memory issue when leak severity is none", () => {
      const input = createBaseInput({
        analysisResults: {
          ...createBaseInput().analysisResults,
          leak: {
            ...ZERO_LEAK_REPORT,
            severity: "none",
            heapGrowthPercent: THRESHOLDS.heapGrowth.error + 1,
          },
        },
      });
      const report = compileReport(input);
      const memoryIssue = report.issues.find((i) => i.category === "memory");
      expect(memoryIssue).toBeUndefined();
    });
  });

  describe("metrics report building", () => {
    it("should calculate FPS change percent correctly", () => {
      const input = createBaseInput({
        captured: {
          ...createBaseInput().captured,
          initialFPS: 60,
          finalFPS: 30,
        },
      });
      const report = compileReport(input);
      expect(report.metrics.fps.changePercent).toBe(50);
    });

    it("should calculate CPU change percent correctly", () => {
      const input = createBaseInput({
        captured: {
          ...createBaseInput().captured,
          initialCPU: 20,
          finalCPU: 40,
        },
      });
      const report = compileReport(input);
      expect(report.metrics.cpu.changePercent).toBe(100);
    });

    it("should calculate GPU change percent correctly", () => {
      const input = createBaseInput({
        captured: {
          ...createBaseInput().captured,
          initialGPU: 25,
          finalGPU: 50,
        },
      });
      const report = compileReport(input);
      expect(report.metrics.gpu.changePercent).toBe(100);
    });

    it("should handle null initial values gracefully", () => {
      const input = createBaseInput({
        captured: {
          ...createBaseInput().captured,
          initialFPS: null,
          initialCPU: null,
          initialGPU: null,
        },
      });
      const report = compileReport(input);
      expect(report.metrics.fps.changePercent).toBeNull();
      expect(report.metrics.cpu.changePercent).toBeNull();
      expect(report.metrics.gpu.changePercent).toBeNull();
    });

    it("should handle zero initial values gracefully", () => {
      const input = createBaseInput({
        captured: {
          ...createBaseInput().captured,
          initialFPS: 0,
          initialCPU: 0,
          initialGPU: 0,
        },
      });
      const report = compileReport(input);
      expect(report.metrics.fps.changePercent).toBeNull();
      expect(report.metrics.cpu.changePercent).toBeNull();
      expect(report.metrics.gpu.changePercent).toBeNull();
    });

    it("should round metrics to one decimal place", () => {
      const input = createBaseInput({
        aggregates: {
          ...createBaseInput().aggregates,
          avgFps: 59.456,
          minFps: 30.123,
          maxFps: 61.999,
        },
      });
      const report = compileReport(input);
      expect(report.metrics.fps.avg).toBe(59.5);
      expect(report.metrics.fps.min).toBe(30.1);
      expect(report.metrics.fps.max).toBe(62);
    });
  });

  describe("summary calculations", () => {
    it("should calculate duration correctly", () => {
      const input = createBaseInput({
        startTime: 1000,
        endTime: 5000,
      });
      const report = compileReport(input);
      expect(report.summary.durationMs).toBe(4000);
    });

    it("should count total samples", () => {
      const input = createBaseInput({
        samples: [
          { timestamp: 0 } as any,
          { timestamp: 1 } as any,
          { timestamp: 2 } as any,
        ],
      });
      const report = compileReport(input);
      expect(report.summary.totalSamples).toBe(3);
    });

    it("should count issues correctly", () => {
      const input = createBaseInput({
        aggregates: {
          ...createBaseInput().aggregates,
          avgFps: THRESHOLDS.fps.warn - 1,
          avgCpu: THRESHOLDS.cpu.warn + 1,
        },
      });
      const report = compileReport(input);
      expect(report.summary.issueCount).toBe(2);
    });
  });
});
