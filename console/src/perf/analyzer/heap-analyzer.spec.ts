// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeEach,describe, expect, it } from "vitest";

import { HeapAnalyzer } from "@/perf/analyzer/heap-analyzer";
import { ZERO_LEAK_REPORT } from "@/perf/analyzer/types";
import { type HeapSnapshot } from "@/perf/metrics/types";

const createSnapshot = (heapUsedMB: number, timestamp = Date.now()): HeapSnapshot => ({
  timestamp,
  heapUsedMB,
  heapTotalMB: heapUsedMB * 1.5,
});

describe("HeapAnalyzer", () => {
  let analyzer: HeapAnalyzer;

  beforeEach(() => {
    analyzer = new HeapAnalyzer();
  });

  describe("edge cases", () => {
    it("should return zero report for empty snapshots", () => {
      const report = analyzer.analyze([]);
      expect(report).toEqual({ ...ZERO_LEAK_REPORT, snapshotCount: 0 });
    });

    it("should return zero report for single snapshot", () => {
      const report = analyzer.analyze([createSnapshot(100)]);
      expect(report).toEqual({ ...ZERO_LEAK_REPORT, snapshotCount: 1 });
    });
  });

  describe("trend detection", () => {
    it("should detect increasing trend", () => {
      const snapshots = Array.from({ length: 10 }, (_, i) =>
        createSnapshot(100 + i * 20),
      );
      const report = analyzer.analyze(snapshots);
      expect(report.trend).toBe("increasing");
    });

    it("should detect decreasing trend", () => {
      const snapshots = Array.from({ length: 10 }, (_, i) =>
        createSnapshot(300 - i * 50),
      );
      const report = analyzer.analyze(snapshots);
      expect(report.trend).toBe("decreasing");
    });

    it("should detect stable trend", () => {
      const snapshots = Array.from({ length: 10 }, (_, i) =>
        createSnapshot(100 + (i % 2)),
      );
      const report = analyzer.analyze(snapshots);
      expect(report.trend).toBe("stable");
    });
  });

  describe("severity calculation", () => {
    it("should report no severity for stable memory", () => {
      const snapshots = Array.from({ length: 10 }, () => createSnapshot(100));
      const report = analyzer.analyze(snapshots);
      expect(report.severity).toBe("none");
    });

    it("should report error severity for significant growth with increasing trend", () => {
      const snapshots = Array.from({ length: 10 }, (_, i) =>
        createSnapshot(100 + i * 20),
      );
      const report = analyzer.analyze(snapshots);
      expect(report.trend).toBe("increasing");
      expect(report.severity).toBe("error");
    });

    it("should not report severity for growth without increasing trend", () => {
      // Create data that shows growth but not an increasing trend
      // (e.g., jump and stabilize)
      const snapshots = [
        ...Array.from({ length: 5 }, () => createSnapshot(100)),
        ...Array.from({ length: 5 }, () => createSnapshot(200)), // 100% growth but stable after
      ];
      const report = analyzer.analyze(snapshots);
      if (report.trend !== "increasing") expect(report.severity).toBe("none");
    });
  });

  describe("report values", () => {
    it("should calculate correct heap values", () => {
      const snapshots = [
        ...Array.from({ length: 5 }, () => createSnapshot(100)),
        ...Array.from({ length: 5 }, () => createSnapshot(150)),
      ];
      const report = analyzer.analyze(snapshots);

      expect(report.heapStartMB).toBe(100);
      expect(report.heapEndMB).toBe(150);
      expect(report.heapGrowthMB).toBe(50);
      expect(report.heapGrowthPercent).toBe(50);
      expect(report.snapshotCount).toBe(10);
    });

    it("should round values appropriately", () => {
      const snapshots = [
        createSnapshot(100.123),
        createSnapshot(100.456),
        createSnapshot(150.789),
        createSnapshot(150.111),
      ];
      const report = analyzer.analyze(snapshots);

      expect(report.heapStartMB).toBeCloseTo(100.3, 1);
      expect(report.heapEndMB).toBeCloseTo(150.5, 1);
    });
  });

  describe("window size handling", () => {
    it("should handle small snapshot counts", () => {
      const snapshots = [createSnapshot(100), createSnapshot(200)];
      const report = analyzer.analyze(snapshots);

      expect(report.snapshotCount).toBe(2);
      expect(report.heapStartMB).toBe(100);
      expect(report.heapEndMB).toBe(200);
    });

    it("should use correct window size for large datasets", () => {
      const snapshots = Array.from({ length: 100 }, (_, i) =>
        createSnapshot(100 + i),
      );
      const report = analyzer.analyze(snapshots);

      expect(report.snapshotCount).toBe(100);
      expect(report.heapStartMB).toBeCloseTo(114.5, 0);
      expect(report.heapEndMB).toBeCloseTo(184.5, 0);
    });
  });
});
