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
  inScope,
  type Item,
  NONE_HIDDEN,
  type Scope,
  setItemVisible,
  showOnly,
  toggleSolo,
  type Visibility,
} from "@/feature/lineplot/visibility";

const SCOPE: Scope = { ranges: ["a", "b", "c"], lines: ["l1", "l2"] };

const NOTHING: Visibility = { hidden: NONE_HIDDEN, solo: null };

const RANGE_A: Item = { variant: "range", range: "a" };
const RANGE_B: Item = { variant: "range", range: "b" };
const LINE_1: Item = { variant: "line", line: "l1" };
const TRACE_1A: Item = { variant: "trace", line: "l1", range: "a" };

const hide = (state: Visibility, item: Item): Visibility =>
  setItemVisible(state, item, false, SCOPE);

const show = (state: Visibility, item: Item): Visibility =>
  setItemVisible(state, item, true, SCOPE);

const solo = (state: Visibility, item: Item): Visibility =>
  toggleSolo(state, item, SCOPE);

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

    it("should show a hidden range with only the line picked", () => {
      expect(show(hide(NOTHING, RANGE_A), TRACE_1A).hidden).toEqual({
        ...NONE_HIDDEN,
        traces: ["l2/a"],
      });
    });

    it("should not hide an item twice", () => {
      expect(hide(hide(NOTHING, RANGE_A), RANGE_A).hidden.ranges).toEqual(["a"]);
    });

    it("should keep the solo when an item changes after it", () => {
      const soloed = solo(NOTHING, RANGE_A);
      expect(show(soloed, RANGE_B)).toEqual({
        hidden: { ...NONE_HIDDEN, ranges: ["c"] },
        solo: soloed.solo,
      });
    });
  });

  describe("showOnly", () => {
    it("should show only a range and keep what was hidden to restore", () => {
      const before = hide(NOTHING, LINE_1);
      expect(showOnly(before, RANGE_A, SCOPE)).toEqual({
        hidden: { ...NONE_HIDDEN, ranges: ["b", "c"] },
        solo: { item: "range:a", restore: before.hidden },
      });
    });

    it("should show only one line of one range", () => {
      expect(showOnly(NOTHING, TRACE_1A, SCOPE).hidden).toEqual({
        ...NONE_HIDDEN,
        ranges: ["b", "c"],
        traces: ["l2/a"],
      });
    });

    it("should show only a line in every range", () => {
      expect(showOnly(NOTHING, LINE_1, SCOPE).hidden).toEqual({
        ...NONE_HIDDEN,
        lines: ["l2"],
      });
    });

    it("should show only the item again when it is the one already shown alone", () => {
      const soloed = showOnly(NOTHING, RANGE_A, SCOPE);
      expect(showOnly(soloed, RANGE_A, SCOPE)).toEqual(soloed);
    });
  });

  describe("toggleSolo", () => {
    it("should restore what was hidden on a second solo of the same item", () => {
      const before = hide(NOTHING, LINE_1);
      expect(solo(solo(before, RANGE_A), RANGE_A)).toEqual(before);
    });

    it("should show only another item and keep the first restore point", () => {
      const before = hide(NOTHING, LINE_1);
      const next = solo(solo(before, RANGE_A), RANGE_B);
      expect(next).toEqual({
        hidden: { ...NONE_HIDDEN, ranges: ["a", "c"] },
        solo: { item: "range:b", restore: before.hidden },
      });
      expect(solo(next, RANGE_B)).toEqual(before);
    });

    it("should restore only the items still in scope", () => {
      const soloed = solo(hide(NOTHING, RANGE_A), LINE_1);
      const smaller: Scope = { ranges: ["b", "c"], lines: SCOPE.lines };
      expect(toggleSolo(soloed, LINE_1, smaller)).toEqual(NOTHING);
    });
  });

  describe("inScope", () => {
    it("should drop the ranges, lines, and traces that are not in scope", () => {
      const hidden = {
        lines: ["l1", "gone"],
        ranges: ["a", "gone"],
        traces: ["l2/b", "l2/gone", "gone/b"],
      };
      expect(inScope(hidden, SCOPE)).toEqual({
        lines: ["l1"],
        ranges: ["a"],
        traces: ["l2/b"],
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
