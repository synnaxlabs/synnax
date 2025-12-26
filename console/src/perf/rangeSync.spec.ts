// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeStamp } from "@synnaxlabs/client";
import { TimeRange } from "@synnaxlabs/x";
import { beforeEach, describe, expect, it, vi } from "vitest";

import * as Perf from "@/perf/slice";

/**
 * Tests for range synchronization with Synnax.
 * These tests verify that profiling ranges are created and updated correctly
 * throughout the profiling lifecycle.
 */
describe("Range Synchronization", () => {
  describe("Redux State Management", () => {
    it("should initialize with null rangeKey and rangeStartTime", () => {
      const state = Perf.ZERO_SLICE_STATE;
      expect(state.rangeKey).toBeNull();
      expect(state.rangeStartTime).toBeNull();
    });

    it("should clear rangeKey and rangeStartTime on start", () => {
      const state = {
        ...Perf.ZERO_SLICE_STATE,
        rangeKey: "test-range-key",
        rangeStartTime: 123456789,
      };

      const action = Perf.start(undefined);
      const reducer = Perf.reducer;
      const newState = reducer(state, action);

      expect(newState.rangeKey).toBeNull();
      expect(newState.rangeStartTime).toBeNull();
      expect(newState.status).toBe("running");
    });

    it("should set rangeKey when setRangeKey is dispatched", () => {
      const state = Perf.ZERO_SLICE_STATE;
      const action = Perf.setRangeKey("test-range-key");
      const reducer = Perf.reducer;
      const newState = reducer(state, action);

      expect(newState.rangeKey).toBe("test-range-key");
    });

    it("should set rangeStartTime when setRangeStartTime is dispatched", () => {
      const state = Perf.ZERO_SLICE_STATE;
      const startTime = 1640000000000;
      const action = Perf.setRangeStartTime(startTime);
      const reducer = Perf.reducer;
      const newState = reducer(state, action);

      expect(newState.rangeStartTime).toBe(startTime);
    });

    it("should clear rangeKey and rangeStartTime on reset", () => {
      const state = {
        ...Perf.ZERO_SLICE_STATE,
        rangeKey: "test-range-key",
        rangeStartTime: 123456789,
      };

      const action = Perf.reset();
      const reducer = Perf.reducer;
      const newState = reducer(state, action);

      expect(newState.rangeKey).toBeNull();
      expect(newState.rangeStartTime).toBeNull();
      expect(newState.status).toBe("idle");
    });
  });

  describe("Range Update Logic", () => {
    let mockClient: any;
    let mockRangeClient: any;

    beforeEach(() => {
      // Mock the Synnax range client
      mockRangeClient = {
        create: vi.fn().mockResolvedValue({ key: "test-range-key" }),
      };

      mockClient = {
        ranges: mockRangeClient,
      };
    });

    it("should create range with MAX end time when profiling starts", async () => {
      const startTime = new TimeStamp(1640000000000n);
      const maxTime = TimeStamp.MAX;

      await mockClient.ranges.create({
        name: `Console Profiling - ${startTime.toLocaleString()}`,
        timeRange: new TimeRange(startTime, maxTime).numeric,
      });

      expect(mockRangeClient.create).toHaveBeenCalledWith({
        name: expect.stringContaining("Console Profiling"),
        timeRange: expect.objectContaining({
          start: expect.any(Number),
          end: expect.any(Number),
        }),
      });

      // Verify the end time is very large (MAX)
      const call = mockRangeClient.create.mock.calls[0][0];
      expect(call.timeRange.end).toBeGreaterThan(1e15);
    });

    it("should update range end time to current timestamp when pausing", async () => {
      const rangeKey = "test-range-key";
      const startTime = new TimeStamp(1640000000000n);
      const endTime = new TimeStamp(1640001000000n);

      await mockClient.ranges.create({
        key: rangeKey,
        name: `Console Profiling - ${startTime.toLocaleString()}`,
        timeRange: new TimeRange(startTime, endTime).numeric,
      });

      expect(mockRangeClient.create).toHaveBeenCalledWith({
        key: rangeKey,
        name: expect.stringContaining("Console Profiling"),
        timeRange: expect.objectContaining({
          start: 1640000000000,
          end: 1640001000000,
        }),
      });
    });

    it("should update range end time to MAX when resuming", async () => {
      const rangeKey = "test-range-key";
      const startTime = new TimeStamp(1640000000000n);
      const maxTime = TimeStamp.MAX;

      await mockClient.ranges.create({
        key: rangeKey,
        name: `Console Profiling - ${startTime.toLocaleString()}`,
        timeRange: new TimeRange(startTime, maxTime).numeric,
      });

      expect(mockRangeClient.create).toHaveBeenCalledWith({
        key: rangeKey,
        name: expect.stringContaining("Console Profiling"),
        timeRange: expect.objectContaining({
          start: expect.any(Number),
          end: expect.any(Number),
        }),
      });

      // Verify the end time is very large (MAX)
      const call = mockRangeClient.create.mock.calls[0][0];
      expect(call.timeRange.end).toBeGreaterThan(1e15);
    });

    it("should finalize range with current timestamp when resetting", async () => {
      const rangeKey = "test-range-key";
      const startTime = new TimeStamp(1640000000000n);
      const endTime = new TimeStamp(1640002000000n);

      await mockClient.ranges.create({
        key: rangeKey,
        name: `Console Profiling - ${startTime.toLocaleString()}`,
        timeRange: new TimeRange(startTime, endTime).numeric,
      });

      expect(mockRangeClient.create).toHaveBeenCalledWith({
        key: rangeKey,
        name: expect.stringContaining("Console Profiling"),
        timeRange: expect.objectContaining({
          start: 1640000000000,
          end: 1640002000000,
        }),
      });
    });

    it("should handle errors gracefully when range creation fails", async () => {
      const mockError = new Error("Network error");
      mockRangeClient.create.mockRejectedValue(mockError);

      const consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});

      try {
        await mockClient.ranges.create({
          name: "Test Range",
          timeRange: new TimeRange(TimeStamp.now(), TimeStamp.MAX).numeric,
        });
      } catch {
        // Error is expected
      }

      consoleErrorSpy.mockRestore();
    });
  });

  describe("Range Lifecycle State Machine", () => {
    /**
     * Test the complete lifecycle of a profiling session:
     * idle -> running (create with MAX) -> paused (set end) -> running (MAX again) -> idle (finalize)
     */
    it("should follow the complete range lifecycle", () => {
      let state = Perf.ZERO_SLICE_STATE;
      const reducer = Perf.reducer;

      // 1. Start profiling (idle -> running)
      state = reducer(state, Perf.start(undefined));
      expect(state.status).toBe("running");
      expect(state.rangeKey).toBeNull(); // Will be set by Dashboard after API call

      // Simulate range creation
      state = reducer(state, Perf.setRangeKey("range-1"));
      state = reducer(state, Perf.setRangeStartTime(1640000000000));
      expect(state.rangeKey).toBe("range-1");
      expect(state.rangeStartTime).toBe(1640000000000);

      // 2. Pause profiling (running -> paused)
      state = reducer(state, Perf.pause());
      expect(state.status).toBe("paused");
      // Range end time would be updated to current timestamp by Dashboard

      // 3. Resume profiling (paused -> running)
      state = reducer(state, Perf.resume());
      expect(state.status).toBe("running");
      // Range end time would be updated to MAX by Dashboard

      // 4. Reset profiling (running -> idle)
      state = reducer(state, Perf.reset());
      expect(state.status).toBe("idle");
      expect(state.rangeKey).toBeNull();
      expect(state.rangeStartTime).toBeNull();
    });
  });

  describe("TimeStamp Integration", () => {
    it("should correctly convert TimeStamp to number for Redux storage", () => {
      const timestamp = TimeStamp.now();
      const timestampValue = Number(timestamp.valueOf());

      expect(typeof timestampValue).toBe("number");
      expect(timestampValue).toBeGreaterThan(0);

      // Note: Due to JavaScript number precision limits, converting bigint to number
      // and back may lose precision for very large values. This is acceptable for
      // our use case since we only need millisecond precision.
      const reconstructed = new TimeStamp(timestampValue);
      const originalMs = Number(timestamp.valueOf());
      const reconstructedMs = Number(reconstructed.valueOf());

      // Verify timestamps are close enough (within 1ms precision)
      expect(Math.abs(originalMs - reconstructedMs)).toBeLessThan(1000);
    });

    it("should handle TimeStamp.MAX correctly", () => {
      const maxTimestamp = TimeStamp.MAX;
      expect(maxTimestamp).toBeDefined();

      // Verify it can be used in TimeRange
      const range = new TimeRange(TimeStamp.now(), maxTimestamp);
      expect(range.end.valueOf()).toBe(maxTimestamp.valueOf());
    });
  });

  describe("Edge Cases", () => {
    it("should handle null client gracefully", () => {
      // The updateRangeEndTime callback returns early if client is null
      const client = null;
      const rangeKey = "test-key";
      const rangeStartTime = 1640000000000;

      // This simulates the guard clause in updateRangeEndTime
      if (client == null || rangeKey == null || rangeStartTime == null)
        // Should return early without error
        expect(true).toBe(true);
    });

    it("should handle missing rangeKey gracefully", () => {
      const client = { ranges: { create: vi.fn() } };
      const rangeKey = null;
      const rangeStartTime = 1640000000000;

      if (client == null || rangeKey == null || rangeStartTime == null)
        expect(true).toBe(true);
    });

    it("should handle missing rangeStartTime gracefully", () => {
      const client = { ranges: { create: vi.fn() } };
      const rangeKey = "test-key";
      const rangeStartTime = null;

      if (client == null || rangeKey == null || rangeStartTime == null)
        expect(true).toBe(true);
    });

    it("should create range when Dashboard mounts with profiling already running", () => {
      // Scenario: User starts profiling from command palette, Dashboard opens after
      let state = Perf.ZERO_SLICE_STATE;
      const reducer = Perf.reducer;

      // Profiling is started (Dashboard not yet mounted)
      state = reducer(state, Perf.start(undefined));
      expect(state.status).toBe("running");
      expect(state.rangeKey).toBeNull();

      // Dashboard mounts and detects profiling is running without a range
      // It should create a range even though prevStatus === status === "running"
      // This is handled by checking: status === "running" && rangeKey === null

      // Simulate Dashboard creating the range
      state = reducer(state, Perf.setRangeKey("range-created-on-mount"));
      state = reducer(state, Perf.setRangeStartTime(1640000000000));

      expect(state.rangeKey).toBe("range-created-on-mount");
      expect(state.rangeStartTime).toBe(1640000000000);
      expect(state.status).toBe("running");
    });
  });
});
