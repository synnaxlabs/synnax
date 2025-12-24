// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeEach, describe, expect, it } from "vitest";

import { CpuAnalyzer } from "@/perf/analyzer/cpu-analyzer";
import { GpuAnalyzer } from "@/perf/analyzer/gpu-analyzer";
import {
  ResourceAnalyzer,
  type ResourceContext,
} from "@/perf/analyzer/resource-analyzer";

describe("ResourceAnalyzer", () => {
  const testPeakThresholds = { warn: 80, error: 95 };
  const testAvgThresholds = { warn: 50, error: 75 };

  let analyzer: ResourceAnalyzer;

  beforeEach(() => {
    analyzer = new ResourceAnalyzer(testPeakThresholds, testAvgThresholds);
  });

  describe("peak severity (based on max)", () => {
    it("should report no severity when below thresholds", () => {
      const ctx: ResourceContext = {
        startPercent: 30,
        endPercent: 40,
        avgPercent: 35,
        maxPercent: 50, // Below warn (80)
      };
      const report = analyzer.analyze(ctx);
      expect(report.peakSeverity).toBe("none");
    });

    it("should report warning when max exceeds warn threshold", () => {
      const ctx: ResourceContext = {
        startPercent: 30,
        endPercent: 40,
        avgPercent: 35,
        maxPercent: 85,
      };
      const report = analyzer.analyze(ctx);
      expect(report.peakSeverity).toBe("warning");
    });

    it("should report error when max exceeds error threshold", () => {
      const ctx: ResourceContext = {
        startPercent: 30,
        endPercent: 40,
        avgPercent: 35,
        maxPercent: 98,
      };
      const report = analyzer.analyze(ctx);
      expect(report.peakSeverity).toBe("error");
    });

    it("should use endPercent if higher than maxPercent", () => {
      const ctx: ResourceContext = {
        startPercent: 30,
        endPercent: 96,
        avgPercent: 50,
        maxPercent: 70,
      };
      const report = analyzer.analyze(ctx);
      expect(report.peakSeverity).toBe("error");
    });

    it("should handle null maxPercent", () => {
      const ctx: ResourceContext = {
        startPercent: 30,
        endPercent: 85,
        avgPercent: 35,
        maxPercent: null,
      };
      const report = analyzer.analyze(ctx);
      expect(report.peakSeverity).toBe("warning");
    });
  });

  describe("avg severity", () => {
    it("should report no severity when avg below thresholds", () => {
      const ctx: ResourceContext = {
        startPercent: 30,
        endPercent: 40,
        avgPercent: 40,
        maxPercent: 50,
      };
      const report = analyzer.analyze(ctx);
      expect(report.avgSeverity).toBe("none");
    });

    it("should report warning when avg exceeds warn threshold", () => {
      const ctx: ResourceContext = {
        startPercent: 30,
        endPercent: 40,
        avgPercent: 60,
        maxPercent: 70,
      };
      const report = analyzer.analyze(ctx);
      expect(report.avgSeverity).toBe("warning");
    });

    it("should report error when avg exceeds error threshold", () => {
      const ctx: ResourceContext = {
        startPercent: 30,
        endPercent: 40,
        avgPercent: 80,
        maxPercent: 90,
      };
      const report = analyzer.analyze(ctx);
      expect(report.avgSeverity).toBe("error");
    });

    it("should handle null avgPercent", () => {
      const ctx: ResourceContext = {
        startPercent: 30,
        endPercent: 40,
        avgPercent: null,
        maxPercent: 50,
      };
      const report = analyzer.analyze(ctx);
      expect(report.avgSeverity).toBe("none");
    });
  });

  describe("report values", () => {
    it("should return all values rounded to 1 decimal", () => {
      const ctx: ResourceContext = {
        startPercent: 25.56,
        endPercent: 35.54,
        avgPercent: 30.55,
        maxPercent: 45.59,
      };
      const report = analyzer.analyze(ctx);

      expect(report.startPercent).toBe(25.6);
      expect(report.endPercent).toBe(35.5);
      expect(report.avgPercent).toBe(30.6);
      expect(report.maxPercent).toBe(45.6);
    });

    it("should handle all null values", () => {
      const ctx: ResourceContext = {
        startPercent: null,
        endPercent: null,
        avgPercent: null,
        maxPercent: null,
      };
      const report = analyzer.analyze(ctx);

      expect(report.peakSeverity).toBe("none");
      expect(report.avgSeverity).toBe("none");
      expect(report.startPercent).toBeNull();
      expect(report.endPercent).toBeNull();
      expect(report.avgPercent).toBeNull();
      expect(report.maxPercent).toBeNull();
    });
  });
});

describe("CpuAnalyzer", () => {
  let analyzer: CpuAnalyzer;

  beforeEach(() => {
    analyzer = new CpuAnalyzer();
  });

  it("should use CPU thresholds", () => {
    // CPU peak thresholds: warn=85, error=95
    // CPU avg thresholds: warn=50, error=75

    const ctx: ResourceContext = {
      startPercent: 30,
      endPercent: 40,
      avgPercent: 60,
      maxPercent: 90,
    };
    const report = analyzer.analyze(ctx);

    expect(report.peakSeverity).toBe("warning");
    expect(report.avgSeverity).toBe("warning");
  });

  it("should report error for high CPU usage", () => {
    const ctx: ResourceContext = {
      startPercent: 50,
      endPercent: 90,
      avgPercent: 80,
      maxPercent: 98,
    };
    const report = analyzer.analyze(ctx);

    expect(report.peakSeverity).toBe("error");
    expect(report.avgSeverity).toBe("error");
  });
});

describe("GpuAnalyzer", () => {
  let analyzer: GpuAnalyzer;

  beforeEach(() => {
    analyzer = new GpuAnalyzer();
  });

  it("should use GPU thresholds", () => {
    const ctx: ResourceContext = {
      startPercent: 30,
      endPercent: 40,
      avgPercent: 60,
      maxPercent: 90,
    };
    const report = analyzer.analyze(ctx);

    expect(report.peakSeverity).toBe("warning");
    expect(report.avgSeverity).toBe("warning");
  });

  it("should report error for high GPU usage", () => {
    const ctx: ResourceContext = {
      startPercent: 50,
      endPercent: 90,
      avgPercent: 80,
      maxPercent: 98,
    };
    const report = analyzer.analyze(ctx);

    expect(report.peakSeverity).toBe("error");
    expect(report.avgSeverity).toBe("error");
  });
});
