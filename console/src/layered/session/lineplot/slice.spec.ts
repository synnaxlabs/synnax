// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";
import { beforeEach, describe, expect, it } from "vitest";

import { LinePlot } from "@/layered/session/lineplot";

const storeWith = (slice: LinePlot.SliceState) =>
  configureStore({
    reducer: { [LinePlot.SLICE_NAME]: LinePlot.reducer },
    preloadedState: { [LinePlot.SLICE_NAME]: slice },
  });

const KEY = "plot-1";

describe("LinePlot Slice", () => {
  let store: ReturnType<typeof storeWith>;

  beforeEach(() => {
    store = storeWith(LinePlot.ZERO_SLICE_STATE);
  });

  const select = <R>(
    selector: (params: LinePlot.KeyedSelectorParams) => R,
    key: string = KEY,
  ): R => selector({ state: store.getState(), key });

  describe("create", () => {
    it("should bootstrap session state from ZERO_STATE for the key", () => {
      store.dispatch(LinePlot.internalCreate({ key: KEY }));
      expect(select(LinePlot.selectState)).toEqual(LinePlot.ZERO_STATE);
    });

    it("should create multiple plots independently", () => {
      store.dispatch(LinePlot.internalCreate({ key: "plot-1" }));
      store.dispatch(LinePlot.internalCreate({ key: "plot-2" }));
      expect(
        Object.keys(LinePlot.selectSliceState(store.getState()).plots),
      ).toHaveLength(2);
    });

    it("should fill annotations with the default for a key", () => {
      store.dispatch(LinePlot.internalCreate({ key: KEY }));
      expect(select(LinePlot.selectState).annotations).toEqual(
        LinePlot.ZERO_ANNOTATIONS_STATE,
      );
    });

    it("should not overwrite an existing entry", () => {
      store.dispatch(LinePlot.internalCreate({ key: KEY }));
      store.dispatch(LinePlot.setActiveToolbarTab({ key: KEY, tab: "lines" }));
      store.dispatch(LinePlot.internalCreate({ key: KEY }));
      expect(select(LinePlot.selectActiveToolbarTab)).toBe("lines");
    });
  });

  describe("setActiveToolbarTab", () => {
    it("should set the active toolbar tab", () => {
      store.dispatch(LinePlot.internalCreate({ key: KEY }));
      store.dispatch(LinePlot.setActiveToolbarTab({ key: KEY, tab: "axes" }));
      expect(select(LinePlot.selectActiveToolbarTab)).toBe("axes");
    });

    it("should lazily create the entry when the key does not exist", () => {
      store.dispatch(LinePlot.setActiveToolbarTab({ key: KEY, tab: "axes" }));
      expect(select(LinePlot.selectActiveToolbarTab)).toBe("axes");
    });
  });

  describe("setControlHold", () => {
    it("should set the hold state to true", () => {
      store.dispatch(LinePlot.internalCreate({ key: KEY }));
      store.dispatch(LinePlot.setControlHold({ key: KEY, hold: true }));
      const control = select(LinePlot.selectControlState);
      expect(control.hold).toBe(true);
    });

    it("should set the hold state to false", () => {
      store.dispatch(LinePlot.internalCreate({ key: KEY }));
      store.dispatch(LinePlot.setControlHold({ key: KEY, hold: false }));
      expect(select(LinePlot.selectControlState).hold).toBe(false);
    });

    it("should toggle the hold state when value is undefined", () => {
      store.dispatch(LinePlot.internalCreate({ key: KEY }));
      expect(select(LinePlot.selectControlState).hold).toBe(false);
      store.dispatch(LinePlot.setControlHold({ key: KEY }));
      expect(select(LinePlot.selectControlState).hold).toBe(true);
      store.dispatch(LinePlot.setControlHold({ key: KEY }));
      expect(select(LinePlot.selectControlState).hold).toBe(false);
    });
  });

  describe("toggleControlClickMode", () => {
    it("should set the click mode when it is currently null", () => {
      store.dispatch(LinePlot.internalCreate({ key: KEY }));
      store.dispatch(LinePlot.toggleControlClickMode({ key: KEY, mode: "measure" }));
      expect(select(LinePlot.selectControlState).clickMode).toBe("measure");
    });

    it("should clear the click mode when it already matches the mode", () => {
      store.dispatch(LinePlot.internalCreate({ key: KEY }));
      store.dispatch(LinePlot.toggleControlClickMode({ key: KEY, mode: "measure" }));
      store.dispatch(LinePlot.toggleControlClickMode({ key: KEY, mode: "measure" }));
      expect(select(LinePlot.selectControlState).clickMode).toBeNull();
    });

    it("should switch to the new mode when a different mode is active", () => {
      store.dispatch(LinePlot.internalCreate({ key: KEY }));
      store.dispatch(LinePlot.toggleControlClickMode({ key: KEY, mode: "measure" }));
      store.dispatch(LinePlot.toggleControlClickMode({ key: KEY, mode: "annotate" }));
      expect(select(LinePlot.selectControlState).clickMode).toBe("annotate");
    });
  });

  describe("setControlEnableTooltip", () => {
    it("should set the tooltip state to true", () => {
      store.dispatch(LinePlot.internalCreate({ key: KEY }));
      store.dispatch(LinePlot.setControlEnableTooltip({ key: KEY, enabled: true }));
      expect(select(LinePlot.selectControlState).enableTooltip).toBe(true);
    });

    it("should set the tooltip state to false", () => {
      store.dispatch(LinePlot.internalCreate({ key: KEY }));
      store.dispatch(LinePlot.setControlEnableTooltip({ key: KEY, enabled: false }));
      expect(select(LinePlot.selectControlState).enableTooltip).toBe(false);
    });

    it("should toggle the tooltip state when value is undefined", () => {
      store.dispatch(LinePlot.internalCreate({ key: KEY }));
      const initial = select(LinePlot.selectControlState).enableTooltip;
      store.dispatch(LinePlot.setControlEnableTooltip({ key: KEY }));
      expect(select(LinePlot.selectControlState).enableTooltip).toBe(!initial);
      store.dispatch(LinePlot.setControlEnableTooltip({ key: KEY }));
      expect(select(LinePlot.selectControlState).enableTooltip).toBe(initial);
    });
  });

  describe("setViewportMode", () => {
    it("should set the viewport mode", () => {
      store.dispatch(LinePlot.internalCreate({ key: KEY }));
      store.dispatch(LinePlot.setViewportMode({ key: KEY, mode: "pan" }));
      expect(select(LinePlot.selectViewportMode)).toBe("pan");
    });
  });

  describe("viewport", () => {
    it("should reset the viewport and bump the render trigger on setViewport", () => {
      store.dispatch(LinePlot.internalCreate({ key: KEY }));
      const before = select(LinePlot.selectState).viewport.renderTrigger;
      store.dispatch(LinePlot.setViewport({ key: KEY, zoom: { width: 2, height: 2 } }));
      const viewport = select(LinePlot.selectState).viewport;
      expect(viewport.renderTrigger).toBe(before + 1);
      expect(viewport.zoom).toEqual({ width: 2, height: 2 });
    });

    it("should merge over the existing viewport on storeViewport", () => {
      store.dispatch(LinePlot.internalCreate({ key: KEY }));
      store.dispatch(
        LinePlot.storeViewport({
          key: KEY,
          zoom: { width: 3, height: 3 },
          pan: { x: 1, y: 1 },
        }),
      );
      const viewport = select(LinePlot.selectState).viewport;
      expect(viewport.zoom).toEqual({ width: 3, height: 3 });
      expect(viewport.pan).toEqual({ x: 1, y: 1 });
    });
  });

  describe("setSelectedRule", () => {
    it("should set the selected rules and switch to the annotations tab", () => {
      store.dispatch(LinePlot.internalCreate({ key: KEY }));
      store.dispatch(LinePlot.setSelectedRule({ key: KEY, ruleKey: ["r1", "r2"] }));
      expect(select(LinePlot.selectSelectedRules)).toEqual(["r1", "r2"]);
      expect(select(LinePlot.selectActiveToolbarTab)).toBe("annotations");
    });

    it("should accept a single rule key", () => {
      store.dispatch(LinePlot.internalCreate({ key: KEY }));
      store.dispatch(LinePlot.setSelectedRule({ key: KEY, ruleKey: "r1" }));
      expect(select(LinePlot.selectSelectedRules)).toEqual(["r1"]);
    });
  });

  describe("setMeasureMode", () => {
    it("should set the measure mode", () => {
      store.dispatch(LinePlot.internalCreate({ key: KEY }));
      store.dispatch(LinePlot.setMeasureMode({ key: KEY, mode: "two" }));
      expect(select(LinePlot.selectMeasureMode)).toBe("two");
    });
  });

  describe("setRangeAnnotationsVisible", () => {
    it("should default to visible on a newly created plot", () => {
      store.dispatch(LinePlot.internalCreate({ key: KEY }));
      expect(select(LinePlot.selectState).annotations.visible).toBe(true);
    });

    it("should hide and re-show range annotations", () => {
      store.dispatch(LinePlot.internalCreate({ key: KEY }));
      store.dispatch(LinePlot.setRangeAnnotationsVisible({ key: KEY, visible: false }));
      expect(select(LinePlot.selectState).annotations.visible).toBe(false);
      store.dispatch(LinePlot.setRangeAnnotationsVisible({ key: KEY, visible: true }));
      expect(select(LinePlot.selectState).annotations.visible).toBe(true);
    });
  });

  describe("setLineVisible", () => {
    it("should hide a line by adding it to hiddenLines", () => {
      store.dispatch(LinePlot.internalCreate({ key: KEY }));
      store.dispatch(
        LinePlot.setLineVisible({ key: KEY, lineKey: "l1", visible: false }),
      );
      expect(select(LinePlot.selectHiddenLines)).toEqual(["l1"]);
    });

    it("should show a hidden line by removing it from hiddenLines", () => {
      store.dispatch(LinePlot.internalCreate({ key: KEY }));
      store.dispatch(
        LinePlot.setLineVisible({ key: KEY, lineKey: "l1", visible: false }),
      );
      store.dispatch(
        LinePlot.setLineVisible({ key: KEY, lineKey: "l1", visible: true }),
      );
      expect(select(LinePlot.selectHiddenLines)).toEqual([]);
    });
  });

  describe("remove", () => {
    it("should remove a plot by key", () => {
      store.dispatch(LinePlot.internalCreate({ key: KEY }));
      store.dispatch(LinePlot.remove({ keys: [KEY] }));
      expect(LinePlot.selectSliceState(store.getState()).plots[KEY]).toBeUndefined();
    });

    it("should ignore keys that do not exist", () => {
      store.dispatch(LinePlot.internalCreate({ key: KEY }));
      store.dispatch(LinePlot.remove({ keys: ["absent"] }));
      expect(select(LinePlot.selectState)).toEqual(LinePlot.ZERO_STATE);
    });
  });

  describe("stateZ schema", () => {
    it("should accept the zero state", () => {
      expect(() => LinePlot.stateZ.parse(LinePlot.ZERO_STATE)).not.toThrow();
    });

    it("should apply prefaulted defaults when nested objects are missing", () => {
      const parsed = LinePlot.stateZ.parse({});
      expect(parsed.control.hold).toBe(false);
      expect(parsed.control.enableTooltip).toBe(true);
      expect(parsed.toolbar.activeTab).toBe("data");
      expect(parsed.measure.mode).toBe("one");
      expect(parsed.annotations.visible).toBe(true);
      expect(parsed.selectedRules).toEqual([]);
      expect(parsed.hiddenLines).toEqual([]);
    });
  });

  describe("purgeState", () => {
    it("should clear hidden lines", () => {
      const state = LinePlot.stateZ.parse({ hiddenLines: ["l1", "l2"] });
      expect(LinePlot.purgeState(state).hiddenLines).toEqual([]);
    });

    it("should leave other fields untouched", () => {
      const state = LinePlot.stateZ.parse({
        hiddenLines: ["l1"],
        toolbar: { activeTab: "axes" },
      });
      expect(LinePlot.purgeState(state).toolbar.activeTab).toBe("axes");
    });
  });

  describe("purgeSliceState", () => {
    it("should clear hidden lines on every plot in the slice", () => {
      const state = {
        [LinePlot.SLICE_NAME]: {
          version: 0 as const,
          plots: {
            "plot-1": LinePlot.stateZ.parse({ hiddenLines: ["l1"] }),
            "plot-2": LinePlot.stateZ.parse({ hiddenLines: ["l2"] }),
          },
        },
      };
      const purged = LinePlot.purgeSliceState(state);
      expect(purged[LinePlot.SLICE_NAME].plots["plot-1"].hiddenLines).toEqual([]);
      expect(purged[LinePlot.SLICE_NAME].plots["plot-2"].hiddenLines).toEqual([]);
    });
  });
});
