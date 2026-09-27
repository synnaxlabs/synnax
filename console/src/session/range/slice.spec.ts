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

const STATIC: Range.StaticState = {
  variant: "static",
  key: "static-1",
  name: "Static 1",
  timeRange: { start: 0, end: 1000 },
};

const PERSISTED: Range.PersistedState = { variant: "persisted", key: "persisted-1" };

const OTHER: Range.StaticState = {
  variant: "static",
  key: "static-2",
  name: "Static 2",
  timeRange: { start: 1000, end: 2000 },
};

const emptyState = (): Range.SliceState => Range.sliceStateZ.parse({ ranges: [] });

const stateWith = (ranges: Range.State[], selected?: string): Range.SliceState => ({
  ...emptyState(),
  ranges,
  selected,
});

describe("range slice", () => {
  describe("default state", () => {
    it("should store no ranges of its own", () => {
      const state = Range.reducer(undefined, { type: "@@INIT" });
      expect(state.ranges).toEqual([]);
      expect(state.selected).toBeUndefined();
    });
  });

  describe("add", () => {
    it("should add a range and select it by default", () => {
      const next = Range.reducer(emptyState(), Range.add(STATIC));
      expect(next.ranges).toEqual([STATIC]);
      expect(next.selected).toEqual(STATIC.key);
    });

    it("should replace an existing range with the same key", () => {
      const renamed: Range.StaticState = { ...STATIC, name: "Replaced" };
      const next = Range.reducer(stateWith([STATIC]), Range.add(renamed));
      expect(next.ranges).toEqual([renamed]);
    });
  });

  describe("remove", () => {
    it("should remove the matching ranges", () => {
      const next = Range.reducer(
        stateWith([STATIC, OTHER]),
        Range.remove({ keys: [STATIC.key] }),
      );
      expect(next.ranges).toEqual([OTHER]);
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
        stateWith([STATIC, OTHER], OTHER.key),
        Range.remove({ keys: [STATIC.key] }),
      );
      expect(next.selected).toEqual(OTHER.key);
    });
  });

  describe("restore", () => {
    it("should put a range back at the index it was removed from", () => {
      const next = Range.reducer(
        stateWith([OTHER]),
        Range.restore({ ranges: [{ index: 0, range: STATIC }] }),
      );
      expect(next.ranges).toEqual([STATIC, OTHER]);
    });

    it("should put the selection back", () => {
      const next = Range.reducer(
        stateWith([]),
        Range.restore({ ranges: [{ index: 0, range: STATIC }], selected: STATIC.key }),
      );
      expect(next.selected).toEqual(STATIC.key);
    });

    it("should keep a selection made while the delete was in flight", () => {
      const next = Range.reducer(
        stateWith([OTHER], OTHER.key),
        Range.restore({ ranges: [{ index: 0, range: STATIC }] }),
      );
      expect(next.selected).toEqual(OTHER.key);
    });

    it("should skip a range the slice already holds", () => {
      const next = Range.reducer(
        stateWith([STATIC]),
        Range.restore({ ranges: [{ index: 0, range: STATIC }] }),
      );
      expect(next.ranges).toEqual([STATIC]);
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
      expect(next.ranges[0]).toEqual({ ...STATIC, name: "Renamed" });
    });

    it("should be a no-op when the key does not match", () => {
      const next = Range.reducer(
        stateWith([STATIC]),
        Range.rename({ key: "missing", name: "Renamed" }),
      );
      expect(next.ranges).toEqual([STATIC]);
    });

    // The Core owns a persisted range's name, so renaming one here would only invent
    // a second answer for it.
    it("should be a no-op for a range the Core holds", () => {
      const next = Range.reducer(
        stateWith([PERSISTED]),
        Range.rename({ key: PERSISTED.key, name: "Renamed" }),
      );
      expect(next.ranges).toEqual([PERSISTED]);
    });
  });
});
