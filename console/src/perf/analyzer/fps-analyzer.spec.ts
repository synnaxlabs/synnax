// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeEach,describe, expect, it } from "vitest";

import { type FpsAnalysisContext,FpsAnalyzer } from "@/perf/analyzer/fps-analyzer";
import { THRESHOLDS } from "@/perf/constants";

describe("FpsAnalyzer", () => {
  let analyzer: FpsAnalyzer;

  beforeEach(() => {
    analyzer = new FpsAnalyzer();
  });

  describe("peak severity (based on min FPS)", () => {
    it("should report no severity for high min FPS", () => {
      const ctx: FpsAnalysisContext = {
        startFps: 60,
        endFps: 55,
        minFps: 50,
        avgFps: 58,
      };
      const report = analyzer.analyze(ctx);
      expect(report.peakSeverity).toBe("none");
    });

    it("should report warning when min FPS below warn threshold", () => {
      const ctx: FpsAnalysisContext = {
        startFps: 60,
        endFps: 15,
        minFps: THRESHOLDS.fps.warn - 1,
        avgFps: 30,
      };
      const report = analyzer.analyze(ctx);
      expect(report.peakSeverity).toBe("warning");
    });

    it("should report error when min FPS below error threshold", () => {
      const ctx: FpsAnalysisContext = {
        startFps: 60,
        endFps: 10,
        minFps: THRESHOLDS.fps.error - 1,
        avgFps: 25,
      };
      const report = analyzer.analyze(ctx);
      expect(report.peakSeverity).toBe("error");
    });

    it("should use endFps if lower than minFps", () => {
      const ctx: FpsAnalysisContext = {
        startFps: 60,
        endFps: 3,
        minFps: 20,
        avgFps: 40,
      };
      const report = analyzer.analyze(ctx);
      expect(report.peakSeverity).toBe("error");
    });
  });

  describe("avg severity (based on avg FPS)", () => {
    it("should report no severity for high avg FPS", () => {
      const ctx: FpsAnalysisContext = {
        startFps: 60,
        endFps: 55,
        minFps: 50,
        avgFps: 55,
      };
      const report = analyzer.analyze(ctx);
      expect(report.avgSeverity).toBe("none");
    });

    it("should report warning when avg FPS below warn threshold", () => {
      const ctx: FpsAnalysisContext = {
        startFps: 60,
        endFps: 20,
        minFps: 15,
        avgFps: THRESHOLDS.fpsAvg.warn - 1,
      };
      const report = analyzer.analyze(ctx);
      expect(report.avgSeverity).toBe("warning");
    });

    it("should report error when avg FPS below error threshold", () => {
      const ctx: FpsAnalysisContext = {
        startFps: 60,
        endFps: 8,
        minFps: 5,
        avgFps: THRESHOLDS.fpsAvg.error - 1,
      };
      const report = analyzer.analyze(ctx);
      expect(report.avgSeverity).toBe("error");
    });

    it("should handle null avg FPS", () => {
      const ctx: FpsAnalysisContext = {
        startFps: 60,
        endFps: 55,
        minFps: 50,
        avgFps: null,
      };
      const report = analyzer.analyze(ctx);
      expect(report.avgSeverity).toBe("none");
    });
  });

  describe("change percent calculation", () => {
    it("should calculate positive change (FPS drop)", () => {
      const ctx: FpsAnalysisContext = {
        startFps: 60,
        endFps: 30,
        minFps: 30,
        avgFps: 45,
      };
      const report = analyzer.analyze(ctx);
      expect(report.changePercent).toBe(50);
    });

    it("should calculate negative change (FPS improvement)", () => {
      const ctx: FpsAnalysisContext = {
        startFps: 30,
        endFps: 60,
        minFps: 30,
        avgFps: 45,
      };
      const report = analyzer.analyze(ctx);
      expect(report.changePercent).toBe(-100);
    });

    it("should handle null start FPS", () => {
      const ctx: FpsAnalysisContext = {
        startFps: null,
        endFps: 60,
        minFps: 50,
        avgFps: 55,
      };
      const report = analyzer.analyze(ctx);
      expect(report.changePercent).toBe(0);
      expect(report.startFps).toBe(0);
    });

    it("should handle null end FPS", () => {
      const ctx: FpsAnalysisContext = {
        startFps: 60,
        endFps: null,
        minFps: 50,
        avgFps: 55,
      };
      const report = analyzer.analyze(ctx);
      expect(report.changePercent).toBe(0);
      expect(report.endFps).toBe(0);
    });

    it("should handle zero start FPS", () => {
      const ctx: FpsAnalysisContext = {
        startFps: 0,
        endFps: 60,
        minFps: 0,
        avgFps: 30,
      };
      const report = analyzer.analyze(ctx);
      expect(report.changePercent).toBe(0);
    });
  });

  describe("report values", () => {
    it("should round FPS values to 1 decimal place", () => {
      const ctx: FpsAnalysisContext = {
        startFps: 59.876,
        endFps: 54.321,
        minFps: 50.555,
        avgFps: 55.999,
      };
      const report = analyzer.analyze(ctx);

      expect(report.startFps).toBe(59.9);
      expect(report.endFps).toBe(54.3);
    });

    it("should round change percent to 2 decimal places", () => {
      const ctx: FpsAnalysisContext = {
        startFps: 60,
        endFps: 47,
        minFps: 45,
        avgFps: 52,
      };
      const report = analyzer.analyze(ctx);
      // (60 - 47) / 60 * 100 = 21.666...
      expect(report.changePercent).toBe(21.67);
    });
  });

  describe("null handling", () => {
    it("should handle all null values", () => {
      const ctx: FpsAnalysisContext = {
        startFps: null,
        endFps: null,
        minFps: null,
        avgFps: null,
      };
      const report = analyzer.analyze(ctx);

      expect(report.peakSeverity).toBe("none");
      expect(report.avgSeverity).toBe("none");
      expect(report.startFps).toBe(0);
      expect(report.endFps).toBe(0);
      expect(report.changePercent).toBe(0);
    });
  });
});
