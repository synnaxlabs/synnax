// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { Visibility } from "@/feature/lineplot/visibility";

const SCOPE: Visibility.Scope = { ranges: ["a", "b", "c"], lines: ["l1", "l2"] };

const NOTHING: Visibility.Visibility = {
  hidden: Visibility.NONE_HIDDEN,
  isolation: null,
};

const RANGE_A: Visibility.Item = { variant: "range", range: "a" };
const RANGE_B: Visibility.Item = { variant: "range", range: "b" };
const LINE_1: Visibility.Item = { variant: "line", line: "l1" };
const TRACE_1A: Visibility.Item = { variant: "trace", line: "l1", range: "a" };

const hide = (
  state: Visibility.Visibility,
  item: Visibility.Item,
): Visibility.Visibility => Visibility.setItemVisible(state, item, false, SCOPE);

const show = (
  state: Visibility.Visibility,
  item: Visibility.Item,
): Visibility.Visibility => Visibility.setItemVisible(state, item, true, SCOPE);

const toggle = (
  state: Visibility.Visibility,
  item: Visibility.Item,
): Visibility.Visibility => Visibility.toggleIsolation(state, item, SCOPE);

describe("Visibility", () => {
  describe("setItemVisible", () => {
    it("should hide and show a range", () => {
      const hidden = hide(NOTHING, RANGE_A);
      expect(hidden.hidden).toEqual({ ...Visibility.NONE_HIDDEN, ranges: ["a"] });
      expect(show(hidden, RANGE_A)).toEqual(NOTHING);
    });

    it("should hide a line of one range by its trace key", () => {
      expect(hide(NOTHING, TRACE_1A).hidden).toEqual({
        ...Visibility.NONE_HIDDEN,
        traces: ["l1/a"],
      });
    });

    it("should hide a line in every range", () => {
      expect(hide(NOTHING, LINE_1).hidden).toEqual({
        ...Visibility.NONE_HIDDEN,
        lines: ["l1"],
      });
    });

    it("should show a line in every range, also where its eye hid it", () => {
      const hidden = hide(hide(NOTHING, TRACE_1A), LINE_1);
      expect(hidden.hidden).toEqual({ ...Visibility.NONE_HIDDEN, lines: ["l1"] });
      expect(show(hidden, LINE_1)).toEqual(NOTHING);
    });

    it("should show a line hidden in every range only in the range picked", () => {
      expect(show(hide(NOTHING, LINE_1), TRACE_1A).hidden).toEqual({
        ...Visibility.NONE_HIDDEN,
        traces: ["l1/b", "l1/c"],
      });
    });

    it("should show a hidden range with only the line picked", () => {
      expect(show(hide(NOTHING, RANGE_A), TRACE_1A).hidden).toEqual({
        ...Visibility.NONE_HIDDEN,
        traces: ["l2/a"],
      });
    });

    it("should not hide an item twice", () => {
      expect(hide(hide(NOTHING, RANGE_A), RANGE_A).hidden.ranges).toEqual(["a"]);
    });

    it("should keep the isolation when an item changes after it", () => {
      const isolated = toggle(NOTHING, RANGE_A);
      expect(show(isolated, RANGE_B)).toEqual({
        hidden: { ...Visibility.NONE_HIDDEN, ranges: ["c"] },
        isolation: isolated.isolation,
      });
    });
  });

  describe("isolate", () => {
    it("should show only a range and keep what was hidden to restore", () => {
      const before = hide(NOTHING, LINE_1);
      expect(Visibility.isolate(before, RANGE_A, SCOPE)).toEqual({
        hidden: { ...Visibility.NONE_HIDDEN, ranges: ["b", "c"] },
        isolation: { item: "range:a", restore: before.hidden },
      });
    });

    it("should show only one line of one range", () => {
      expect(Visibility.isolate(NOTHING, TRACE_1A, SCOPE).hidden).toEqual({
        ...Visibility.NONE_HIDDEN,
        ranges: ["b", "c"],
        traces: ["l2/a"],
      });
    });

    it("should show only a line in every range", () => {
      expect(Visibility.isolate(NOTHING, LINE_1, SCOPE).hidden).toEqual({
        ...Visibility.NONE_HIDDEN,
        lines: ["l2"],
      });
    });

    it("should show only the item again when it is the one already shown alone", () => {
      const isolated = Visibility.isolate(NOTHING, RANGE_A, SCOPE);
      expect(Visibility.isolate(isolated, RANGE_A, SCOPE)).toEqual(isolated);
    });
  });

  describe("toggleIsolation", () => {
    it("should restore what was hidden on a second toggle of the same item", () => {
      const before = hide(NOTHING, LINE_1);
      expect(toggle(toggle(before, RANGE_A), RANGE_A)).toEqual(before);
    });

    it("should show only another item and keep the first restore point", () => {
      const before = hide(NOTHING, LINE_1);
      const next = toggle(toggle(before, RANGE_A), RANGE_B);
      expect(next).toEqual({
        hidden: { ...Visibility.NONE_HIDDEN, ranges: ["a", "c"] },
        isolation: { item: "range:b", restore: before.hidden },
      });
      expect(toggle(next, RANGE_B)).toEqual(before);
    });

    it("should restore only the items still in scope", () => {
      const isolated = toggle(hide(NOTHING, RANGE_A), LINE_1);
      const smaller: Visibility.Scope = { ranges: ["b", "c"], lines: SCOPE.lines };
      expect(Visibility.toggleIsolation(isolated, LINE_1, smaller)).toEqual(NOTHING);
    });
  });

  describe("inScope", () => {
    it("should drop the ranges, lines, and traces that are not in scope", () => {
      const hidden = {
        lines: ["l1", "gone"],
        ranges: ["a", "gone"],
        traces: ["l2/b", "l2/gone", "gone/b"],
      };
      expect(Visibility.inScope(hidden, SCOPE)).toEqual({
        lines: ["l1"],
        ranges: ["a"],
        traces: ["l2/b"],
      });
    });
  });

  describe("isLineHidden", () => {
    it("should be true for a line hidden in every range", () => {
      expect(Visibility.isLineHidden(hide(NOTHING, LINE_1).hidden, "l1", SCOPE)).toBe(
        true,
      );
    });

    it("should be true for a line hidden in each range one at a time", () => {
      const hidden = { ...Visibility.NONE_HIDDEN, traces: ["l1/a", "l1/b", "l1/c"] };
      expect(Visibility.isLineHidden(hidden, "l1", SCOPE)).toBe(true);
    });

    it("should be false for a line shown in one range", () => {
      const hidden = { ...Visibility.NONE_HIDDEN, traces: ["l1/a", "l1/b"] };
      expect(Visibility.isLineHidden(hidden, "l1", SCOPE)).toBe(false);
    });

    it("should be false with no ranges in scope", () => {
      expect(
        Visibility.isLineHidden(Visibility.NONE_HIDDEN, "l1", { ...SCOPE, ranges: [] }),
      ).toBe(false);
    });
  });

  describe("countHidden", () => {
    it("should count the hidden ranges, lines, and traces", () => {
      expect(
        Visibility.countHidden({ lines: ["l1"], ranges: ["a", "b"], traces: ["l2/c"] }),
      ).toBe(4);
    });
  });
});
