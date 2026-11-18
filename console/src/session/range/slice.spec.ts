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

const PERSISTED: Range.PersistedState = { variant: "persisted", key: "persisted-1" };

const OTHER: Range.PersistedState = { variant: "persisted", key: "persisted-2" };

const emptyState = (): Range.SliceState => Range.sliceStateZ.parse({});

const stateWith = (ranges: Range.PersistedState[]): Range.SliceState => ({
  ...emptyState(),
  ranges,
});

describe("range slice", () => {
  describe("default state", () => {
    it("should store no ranges and open every toolbar group", () => {
      expect(Range.reducer(undefined, { type: "@@INIT" })).toEqual({
        version: 1,
        ranges: [],
        toolbar: { collapsed: [] },
        staticWindows: {},
      });
    });
  });

  describe("version 0", () => {
    it("should drop the ranges kept only in the Console and the selection", () => {
      const state = Range.sliceStateZ.parse({
        version: 0,
        selected: PERSISTED.key,
        ranges: [
          PERSISTED,
          {
            variant: "static",
            key: "local-1",
            name: "Local 1",
            timeRange: { start: 0, end: 1000 },
          },
        ],
        staticKeysRewritten: true,
      });
      expect(state).toEqual({ ...Range.ZERO_SLICE_STATE, ranges: [PERSISTED] });
    });

    it("should keep the times of those ranges while line plots may hold their keys", () => {
      const state = Range.sliceStateZ.parse({
        version: 0,
        ranges: [
          {
            variant: "static",
            key: "local-1",
            name: "Local 1",
            timeRange: { start: 0, end: 1000 },
          },
        ],
        staticKeysRewritten: false,
      });
      expect(state.ranges).toEqual([]);
      expect(state.staticWindows).toEqual({ "local-1": { start: 0, end: 1000 } });
    });
  });

  describe("add", () => {
    it("should add a range", () => {
      const next = Range.reducer(emptyState(), Range.add(PERSISTED));
      expect(next.ranges).toEqual([PERSISTED]);
    });

    it("should not add a second entry for the same key", () => {
      const next = Range.reducer(stateWith([PERSISTED]), Range.add(PERSISTED));
      expect(next.ranges).toEqual([PERSISTED]);
    });
  });

  describe("remove", () => {
    it("should remove the matching ranges", () => {
      const next = Range.reducer(
        stateWith([PERSISTED, OTHER]),
        Range.remove({ keys: [PERSISTED.key] }),
      );
      expect(next.ranges).toEqual([OTHER]);
    });
  });

  describe("setGroupCollapsed", () => {
    it("should collapse a group once and open it again", () => {
      const collapse = Range.setGroupCollapsed({ group: "recent", collapsed: true });
      const collapsed = Range.reducer(Range.reducer(emptyState(), collapse), collapse);
      expect(collapsed.toolbar.collapsed).toEqual(["recent"]);
      const opened = Range.reducer(
        collapsed,
        Range.setGroupCollapsed({ group: "recent", collapsed: false }),
      );
      expect(opened.toolbar.collapsed).toEqual([]);
    });
  });
});
