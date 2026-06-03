// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";

import { type AxisKey } from "@/lineplot/axis";
import {
  select,
  selectActiveToolbarTab,
  selectAxes,
  selectAxis,
  selectAxisBounds,
  selectControlState,
  selectControlStateOptional,
  selectExists,
  selectIsRemoteCreated,
  selectLine,
  selectLineKeys,
  selectLines,
  selectMeasureMode,
  selectMultiple,
  selectOptional,
  selectRanges,
  selectRule,
  selectRules,
  selectSelection,
  selectSliceState,
  selectToolbar,
  selectVersion,
  selectViewportMode,
} from "@/lineplot/selectors";
import {
  SLICE_NAME,
  type State,
  type StoreState,
  ZERO_SLICE_STATE,
  ZERO_STATE,
} from "@/lineplot/slice";
import { Range } from "@/range";

const KEY = "plot-1";
const AXIS = Object.keys(ZERO_STATE.axes.axes)[0] as AxisKey;

const entry: State = {
  ...ZERO_STATE,
  key: KEY,
  remoteCreated: true,
  toolbar: { activeTab: "annotations" },
};

const state: StoreState = {
  [SLICE_NAME]: { ...ZERO_SLICE_STATE, plots: { [KEY]: entry } },
};

const empty: StoreState = { [SLICE_NAME]: ZERO_SLICE_STATE };

describe("lineplot selectors", () => {
  describe("selectSliceState", () => {
    it("returns the slice state", () => {
      expect(selectSliceState(state)).toBe(state[SLICE_NAME]);
    });
  });

  describe("select / selectOptional / selectMultiple", () => {
    it("returns the entry when present", () => {
      expect(select(state, KEY)).toBe(entry);
      expect(selectOptional(state, KEY)).toBe(entry);
      expect(selectMultiple(state, [KEY])).toEqual([entry]);
    });

    it("returns undefined from selectOptional when absent", () => {
      expect(selectOptional(empty, "absent")).toBeUndefined();
    });
  });

  describe("selectExists", () => {
    it("reports whether the entry is present", () => {
      expect(selectExists(state, KEY)).toBe(true);
      expect(selectExists(empty, "absent")).toBe(false);
    });
  });

  describe("toolbar selectors", () => {
    it("selectActiveToolbarTab reads the active tab", () => {
      expect(selectActiveToolbarTab(state, KEY)).toBe("annotations");
    });

    it("selectToolbar returns the toolbar, undefined when absent", () => {
      expect(selectToolbar(state, KEY)).toBe(entry.toolbar);
      expect(selectToolbar(empty, "absent")).toBeUndefined();
    });
  });

  describe("control / viewport / measure / selection", () => {
    it("selectControlState and its optional variant", () => {
      expect(selectControlState(state, KEY)).toBe(entry.control);
      expect(selectControlStateOptional(state, KEY)).toBe(entry.control);
      expect(selectControlStateOptional(empty, "absent")).toBeUndefined();
    });

    it("selectViewportMode reads the mode", () => {
      expect(selectViewportMode(state, KEY)).toBe(entry.mode);
    });

    it("selectMeasureMode reads the measure mode", () => {
      expect(selectMeasureMode(state, KEY)).toBe(entry.measure.mode);
    });

    it("selectSelection reads the selection", () => {
      expect(selectSelection(state, KEY)).toBe(entry.selection);
    });
  });

  describe("axes selectors", () => {
    it("selectAxes returns the axes record", () => {
      expect(selectAxes(state, KEY)).toBe(entry.axes.axes);
    });

    it("selectAxis returns a single axis", () => {
      expect(selectAxis(state, KEY, AXIS)).toBe(entry.axes.axes[AXIS]);
    });

    it("selectAxisBounds returns the axis bounds", () => {
      expect(selectAxisBounds(state, KEY, AXIS)).toBe(entry.axes.axes[AXIS].bounds);
    });
  });

  describe("rules selectors", () => {
    it("selectRules returns the rules", () => {
      expect(selectRules(state, KEY)).toBe(entry.rules);
    });

    it("selectRule returns undefined without a rule key or on miss", () => {
      expect(selectRule(state, KEY)).toBeUndefined();
      expect(selectRule(state, KEY, "missing")).toBeUndefined();
    });
  });

  describe("lines selectors", () => {
    it("selectLines returns the lines", () => {
      expect(selectLines(state, KEY)).toBe(entry.lines);
    });

    it("selectLineKeys maps the line keys", () => {
      expect(selectLineKeys(state, KEY)).toEqual(entry.lines.map(({ key }) => key));
    });

    it("selectLine returns undefined without a line key or on miss", () => {
      expect(selectLine(state, KEY)).toBeUndefined();
      expect(selectLine(state, KEY, "missing")).toBeUndefined();
    });
  });

  describe("metadata selectors", () => {
    it("selectVersion reads the version, undefined when absent", () => {
      expect(selectVersion(state, KEY)).toBe(ZERO_STATE.version);
      expect(selectVersion(empty, "absent")).toBeUndefined();
    });

    it("selectIsRemoteCreated reads remoteCreated, undefined when absent", () => {
      expect(selectIsRemoteCreated(state, KEY)).toBe(true);
      expect(selectIsRemoteCreated(empty, "absent")).toBeUndefined();
    });
  });

  describe("selectRanges", () => {
    it("resolves the plot's range keys against the range slice", () => {
      const combined = {
        ...state,
        [Range.SLICE_NAME]: Range.ZERO_SLICE_STATE,
      } as StoreState & Range.StoreState;
      expect(selectRanges(combined, KEY)).toEqual({ x1: [], x2: [] });
    });
  });
});
