// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Range } from "@/session/range";

const STATIC: Range.Static = {
  key: "static-1",
  name: "Static 1",
  persisted: true,
  variant: "static",
  timeRange: { start: 0, end: 1000 },
};

const DYNAMIC: Range.Dynamic = {
  key: "dynamic-1",
  name: "Dynamic 1",
  persisted: false,
  variant: "dynamic",
  span: 1000,
};

const emptyState = (): Range.SliceState => Range.sliceStateZ.parse({ ranges: [] });

const stateWith = (ranges: Range.Range[], selected?: string): Range.SliceState => ({
  ...emptyState(),
  ranges,
  selected,
});

describe("range slice", () => {
  describe("default state", () => {
    it("should seed the rolling ranges including the recent range", () => {
      const state = Range.reducer(undefined, { type: "@@INIT" });
      expect(state.ranges.map((r) => r.key)).toContain(Range.RECENT_KEY);
      expect(state.selected).toBeUndefined();
    });
  });

  describe("add", () => {
    it("should add a range and select it by default", () => {
      const next = Range.reducer(emptyState(), Range.add(STATIC));
      expect(next.ranges).toEqual([STATIC]);
      expect(next.selected).toEqual(STATIC.key);
    });

    it("should not select the range when switchActive is false", () => {
      const next = Range.reducer(
        emptyState(),
        Range.add({ ...DYNAMIC, switchActive: false }),
      );
      expect(next.ranges).toEqual([DYNAMIC]);
      expect(next.selected).toBeUndefined();
    });

    it("should replace an existing range with the same key", () => {
      const renamed: Range.Static = { ...STATIC, name: "Replaced" };
      const next = Range.reducer(stateWith([STATIC]), Range.add(renamed));
      expect(next.ranges).toEqual([renamed]);
    });
  });

  describe("remove", () => {
    it("should remove the matching ranges", () => {
      const next = Range.reducer(
        stateWith([STATIC, DYNAMIC]),
        Range.remove({ keys: [STATIC.key] }),
      );
      expect(next.ranges).toEqual([DYNAMIC]);
    });

    it("should clear the selection when the selected range is removed", () => {
      const next = Range.reducer(
        stateWith([STATIC], STATIC.key),
        Range.remove({ keys: [STATIC.key] }),
      );
      expect(next.selected).toBeUndefined();
    });

    it("should keep the selection when a different range is removed", () => {
      const next = Range.reducer(
        stateWith([STATIC, DYNAMIC], DYNAMIC.key),
        Range.remove({ keys: [STATIC.key] }),
      );
      expect(next.selected).toEqual(DYNAMIC.key);
    });
  });

  describe("select", () => {
    it("should set the selected key", () => {
      const next = Range.reducer(emptyState(), Range.select(STATIC.key));
      expect(next.selected).toEqual(STATIC.key);
    });
  });

  describe("rename", () => {
    it("should rename the matching range", () => {
      const next = Range.reducer(
        stateWith([STATIC]),
        Range.rename({ key: STATIC.key, name: "Renamed" }),
      );
      expect(next.ranges[0].name).toEqual("Renamed");
    });

    it("should be a no-op when the key does not match", () => {
      const next = Range.reducer(
        stateWith([STATIC]),
        Range.rename({ key: "missing", name: "Renamed" }),
      );
      expect(next.ranges).toEqual([STATIC]);
    });
  });

  describe("updateRemote", () => {
    it("should update the name and time range of a static range", () => {
      const next = Range.reducer(
        stateWith([STATIC]),
        Range.updateRemote({
          key: STATIC.key,
          name: "Remote",
          timeRange: { start: 5, end: 50 },
        }),
      );
      expect(next.ranges[0]).toEqual({
        ...STATIC,
        name: "Remote",
        timeRange: { start: 5, end: 50 },
      });
    });

    it("should be a no-op for a dynamic range", () => {
      const next = Range.reducer(
        stateWith([DYNAMIC]),
        Range.updateRemote({
          key: DYNAMIC.key,
          name: "Remote",
          timeRange: { start: 5, end: 50 },
        }),
      );
      expect(next.ranges).toEqual([DYNAMIC]);
    });

    it("should be a no-op when the key does not match", () => {
      const next = Range.reducer(
        stateWith([STATIC]),
        Range.updateRemote({
          key: "missing",
          name: "Remote",
          timeRange: { start: 5, end: 50 },
        }),
      );
      expect(next.ranges).toEqual([STATIC]);
    });
  });
});
