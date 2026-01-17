// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeEach, describe, expect, it } from "vitest";

import { SampleBuffer, ZERO_AGGREGATES, ZERO_SAMPLE } from "@/perf/metrics/buffer";
import { type MetricSample } from "@/perf/metrics/types";

const createSample = (overrides: Partial<MetricSample> = {}): MetricSample => ({
  ...ZERO_SAMPLE,
  timestamp: Date.now(),
  ...overrides,
});

describe("SampleBuffer", () => {
  let buffer: SampleBuffer;

  beforeEach(() => {
    buffer = new SampleBuffer(3, 3);
  });

  describe("initial state", () => {
    it("should start with empty buffers", () => {
      expect(buffer.getBaselineSamples()).toEqual([]);
      expect(buffer.getRecentSamples()).toEqual([]);
      expect(buffer.getAllSamples()).toEqual([]);
      expect(buffer.getTotalSampleCount()).toBe(0);
    });

    it("should return zero aggregates when empty", () => {
      expect(buffer.getAggregates()).toEqual(ZERO_AGGREGATES);
    });
  });

  describe("baseline buffer", () => {
    it("should fill baseline buffer first", () => {
      const s1 = createSample({ frameRate: 60 });
      const s2 = createSample({ frameRate: 55 });
      const s3 = createSample({ frameRate: 50 });

      buffer.push(s1);
      buffer.push(s2);
      buffer.push(s3);

      expect(buffer.getBaselineSamples()).toHaveLength(3);
      expect(buffer.getBaselineSamples()).toEqual([s1, s2, s3]);
      expect(buffer.getRecentSamples()).toEqual([]);
    });

    it("should not overflow baseline buffer", () => {
      const samples = [1, 2, 3, 4].map((i) => createSample({ frameRate: i * 10 }));
      samples.forEach((s) => buffer.push(s));

      expect(buffer.getBaselineSamples()).toHaveLength(3);
      expect(buffer.getBaselineSamples()).toEqual(samples.slice(0, 3));
    });
  });

  describe("recent buffer (circular)", () => {
    it("should start filling recent after baseline is full", () => {
      const samples = [1, 2, 3, 4].map((i) => createSample({ frameRate: i * 10 }));
      samples.forEach((s) => buffer.push(s));

      expect(buffer.getRecentSamples()).toHaveLength(1);
      expect(buffer.getRecentSamples()[0]).toEqual(samples[3]);
    });

    it("should implement circular buffer correctly", () => {
      const samples = [1, 2, 3, 4, 5, 6, 7, 8].map((i) =>
        createSample({ frameRate: i * 10 }),
      );
      samples.forEach((s) => buffer.push(s));

      const recent = buffer.getRecentSamples();
      expect(recent).toHaveLength(3);
      expect(recent[0].frameRate).toBe(60);
      expect(recent[1].frameRate).toBe(70);
      expect(recent[2].frameRate).toBe(80);
    });

    it("should maintain correct order after multiple wrap-arounds", () => {
      const samples = Array.from({ length: 12 }, (_, i) =>
        createSample({ frameRate: i + 1 }),
      );
      samples.forEach((s) => buffer.push(s));

      const recent = buffer.getRecentSamples();
      expect(recent).toHaveLength(3);
      expect(recent[0].frameRate).toBe(10);
      expect(recent[1].frameRate).toBe(11);
      expect(recent[2].frameRate).toBe(12);
    });
  });

  describe("getAllSamples", () => {
    it("should combine baseline and recent samples", () => {
      const samples = [1, 2, 3, 4, 5].map((i) => createSample({ frameRate: i * 10 }));
      samples.forEach((s) => buffer.push(s));

      const all = buffer.getAllSamples();
      expect(all).toHaveLength(5);
      expect(all.map((s) => s.frameRate)).toEqual([10, 20, 30, 40, 50]);
    });
  });

  describe("aggregates", () => {
    it("should calculate running averages correctly", () => {
      buffer.push(createSample({ frameRate: 60 }));
      buffer.push(createSample({ frameRate: 40 }));
      buffer.push(createSample({ frameRate: 50 }));

      const agg = buffer.getAggregates();
      expect(agg.avgFps).toBe(50);
      expect(agg.maxFps).toBe(60);
    });

    it("should track min/max correctly after warmup period", () => {
      const largeBuffer = new SampleBuffer(10, 10);

      for (let i = 0; i < 5; i++) largeBuffer.push(createSample({ cpuPercent: 10 }));

      largeBuffer.push(createSample({ cpuPercent: 25 }));
      largeBuffer.push(createSample({ cpuPercent: 75 }));
      largeBuffer.push(createSample({ cpuPercent: 50 }));

      const agg = largeBuffer.getAggregates();
      expect(agg.avgCpu).toBe(25);
      expect(agg.maxCpu).toBe(75);
    });

    it("should handle null values gracefully", () => {
      buffer.push(createSample({ frameRate: 60, cpuPercent: null }));
      buffer.push(createSample({ frameRate: null, cpuPercent: 50 }));

      const agg = buffer.getAggregates();
      expect(agg.avgFps).toBe(60);
      expect(agg.avgCpu).toBe(50);
    });

    it("should track heap min/max correctly after warmup period", () => {
      const largeBuffer = new SampleBuffer(10, 10);

      for (let i = 0; i < 5; i++) largeBuffer.push(createSample({ heapUsedMB: 50 }));

      largeBuffer.push(createSample({ heapUsedMB: 100 }));
      largeBuffer.push(createSample({ heapUsedMB: 200 }));
      largeBuffer.push(createSample({ heapUsedMB: 150 }));

      const agg = largeBuffer.getAggregates();
      expect(agg.minHeap).toBe(50);
      expect(agg.maxHeap).toBe(200);
    });
  });

  describe("getTotalSampleCount", () => {
    it("should track total samples pushed", () => {
      expect(buffer.getTotalSampleCount()).toBe(0);

      buffer.push(createSample());
      expect(buffer.getTotalSampleCount()).toBe(1);

      for (let i = 0; i < 10; i++) buffer.push(createSample());
      expect(buffer.getTotalSampleCount()).toBe(11);
    });
  });

  describe("reset", () => {
    it("should clear all buffers and aggregates", () => {
      for (let i = 0; i < 10; i++)
        buffer.push(createSample({ frameRate: 60, cpuPercent: 50 }));

      buffer.reset();

      expect(buffer.getBaselineSamples()).toEqual([]);
      expect(buffer.getRecentSamples()).toEqual([]);
      expect(buffer.getAllSamples()).toEqual([]);
      expect(buffer.getTotalSampleCount()).toBe(0);
      expect(buffer.getAggregates()).toEqual(ZERO_AGGREGATES);
    });

    it("should allow reuse after reset", () => {
      buffer.push(createSample({ frameRate: 60 }));
      buffer.reset();
      buffer.push(createSample({ frameRate: 30 }));

      expect(buffer.getTotalSampleCount()).toBe(1);
      expect(buffer.getAggregates().avgFps).toBe(30);
    });
  });

  describe("warmup period", () => {
    it("should skip min tracking during warmup (first 5 samples)", () => {
      const largeBuffer = new SampleBuffer(10, 10);

      largeBuffer.push(createSample({ frameRate: 5 }));
      largeBuffer.push(createSample({ frameRate: 5 }));
      largeBuffer.push(createSample({ frameRate: 5 }));
      largeBuffer.push(createSample({ frameRate: 5 }));
      largeBuffer.push(createSample({ frameRate: 5 }));

      largeBuffer.push(createSample({ frameRate: 60 }));
      largeBuffer.push(createSample({ frameRate: 50 }));

      const agg = largeBuffer.getAggregates();
      expect(agg.minFps).toBe(50);
    });
  });
});
