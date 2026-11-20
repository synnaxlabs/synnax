// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import {
  countHidden,
  type Item,
  NONE_HIDDEN,
  type Scope,
  setItemVisible,
  type Visibility,
} from "@/feature/lineplot/visibility";

const SCOPE: Scope = { ranges: ["a", "b", "c"], lines: ["l1", "l2"] };

const NOTHING: Visibility = { hidden: NONE_HIDDEN, soloRestore: null };

const RANGE_A: Item = { variant: "range", range: "a" };
const LINE_1: Item = { variant: "line", line: "l1" };
const TRACE_1A: Item = { variant: "trace", line: "l1", range: "a" };

const hide = (state: Visibility, item: Item): Visibility =>
  setItemVisible(state, item, false, false, SCOPE);

const show = (state: Visibility, item: Item): Visibility =>
  setItemVisible(state, item, true, false, SCOPE);

const solo = (state: Visibility, item: Item): Visibility =>
  setItemVisible(state, item, true, true, SCOPE);

describe("lineplot visibility", () => {
  describe("setItemVisible", () => {
    it("should hide and show a range", () => {
      const hidden = hide(NOTHING, RANGE_A);
      expect(hidden.hidden).toEqual({ ...NONE_HIDDEN, ranges: ["a"] });
      expect(show(hidden, RANGE_A)).toEqual(NOTHING);
    });

    it("should hide a line of one range by its trace key", () => {
      expect(hide(NOTHING, TRACE_1A).hidden).toEqual({
        ...NONE_HIDDEN,
        traces: ["l1/a"],
      });
    });

    it("should hide a line in every range", () => {
      expect(hide(NOTHING, LINE_1).hidden).toEqual({ ...NONE_HIDDEN, lines: ["l1"] });
    });

    it("should show a line hidden in every range only in the range picked", () => {
      expect(show(hide(NOTHING, LINE_1), TRACE_1A).hidden).toEqual({
        ...NONE_HIDDEN,
        traces: ["l1/b", "l1/c"],
      });
    });

    it("should not hide an item twice", () => {
      expect(hide(hide(NOTHING, RANGE_A), RANGE_A).hidden.ranges).toEqual(["a"]);
    });

    it("should show only a range and keep what was hidden to restore", () => {
      const before = hide(NOTHING, LINE_1);
      expect(solo(before, RANGE_A)).toEqual({
        hidden: { ...NONE_HIDDEN, ranges: ["b", "c"] },
        soloRestore: before.hidden,
      });
    });

    it("should show only one line of one range", () => {
      expect(solo(NOTHING, TRACE_1A).hidden).toEqual({
        ...NONE_HIDDEN,
        ranges: ["b", "c"],
        traces: ["l2/a"],
      });
    });

    it("should show only a line in every range", () => {
      expect(solo(NOTHING, LINE_1).hidden).toEqual({ ...NONE_HIDDEN, lines: ["l2"] });
    });

    it("should restore what was hidden on a second solo", () => {
      const before = hide(NOTHING, LINE_1);
      expect(solo(solo(before, RANGE_A), TRACE_1A)).toEqual(before);
    });

    it("should keep the restore point when an item changes after a solo", () => {
      const soloed = solo(NOTHING, RANGE_A);
      expect(show(soloed, { variant: "range", range: "b" })).toEqual({
        hidden: { ...NONE_HIDDEN, ranges: ["c"] },
        soloRestore: NONE_HIDDEN,
      });
    });
  });

  describe("countHidden", () => {
    it("should count the hidden ranges, lines, and traces", () => {
      expect(countHidden({ lines: ["l1"], ranges: ["a", "b"], traces: ["l2/c"] })).toBe(
        4,
      );
    });
  });
});
