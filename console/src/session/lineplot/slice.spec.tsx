// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";
import { LinePlot as PLinePlot } from "@synnaxlabs/pluto";
import { act, renderHook } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { beforeEach, describe, expect, it } from "vitest";

import { LinePlot } from "@/session/lineplot";

const storeWith = (slice: LinePlot.SliceState) =>
  configureStore({
    reducer: { [LinePlot.SLICE_NAME]: LinePlot.reducer },
    preloadedState: { [LinePlot.SLICE_NAME]: slice },
  });

const KEY = "plot-1";

const wrapperFor = (
  store: ReturnType<typeof storeWith>,
  key: string,
): FC<PropsWithChildren> => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Provider store={store}>
      <PLinePlot.Scope.Provider value={key}>{children}</PLinePlot.Scope.Provider>
    </Provider>
  );
  Wrapper.displayName = "Wrapper";
  return Wrapper;
};

const renderGetters = (store: ReturnType<typeof storeWith>, key: string = KEY) =>
  renderHook(
    () => ({
      state: LinePlot.useGet(),
      toolbarTab: LinePlot.useGetActiveToolbarTab(),
      control: LinePlot.useGetControlState(),
      viewportMode: LinePlot.useGetViewportMode(),
      hiddenLines: LinePlot.useGetHiddenLines(),
      measureMode: LinePlot.useGetMeasureMode(),
      selectedRules: LinePlot.useGetSelectedRules(),
    }),
    { wrapper: wrapperFor(store, key) },
  ).result.current;

describe("LinePlot Slice", () => {
  let store: ReturnType<typeof storeWith>;

  beforeEach(() => {
    store = storeWith(LinePlot.ZERO_SLICE_STATE);
  });

  describe("create", () => {
    it("should bootstrap session state from ZERO_STATE for the key", () => {
      const get = renderGetters(store);
      act(() => store.dispatch(LinePlot.create({ key: KEY })));
      expect(get.state()).toEqual(LinePlot.ZERO_STATE);
    });

    it("should create multiple plots independently", () => {
      const get = renderGetters(store);
      act(() => {
        store.dispatch(LinePlot.create({ key: "plot-1" }));
        store.dispatch(LinePlot.create({ key: "plot-2" }));
        store.dispatch(LinePlot.setActiveToolbarTab({ key: "plot-1", tab: "lines" }));
        store.dispatch(LinePlot.setActiveToolbarTab({ key: "plot-2", tab: "axes" }));
      });
      expect(get.toolbarTab({ key: "plot-1" })).toBe("lines");
      expect(get.toolbarTab({ key: "plot-2" })).toBe("axes");
    });

    it("should fill annotations with the default for a key", () => {
      const get = renderGetters(store);
      act(() => store.dispatch(LinePlot.create({ key: KEY })));
      expect(get.state().annotations).toEqual(LinePlot.ZERO_ANNOTATIONS_STATE);
    });

    it("should not overwrite an existing entry", () => {
      const get = renderGetters(store);
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(LinePlot.setActiveToolbarTab({ key: KEY, tab: "lines" }));
        store.dispatch(LinePlot.create({ key: KEY }));
      });
      expect(get.toolbarTab()).toBe("lines");
    });
  });

  describe("setActiveToolbarTab", () => {
    it("should set the active toolbar tab", () => {
      const get = renderGetters(store);
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(LinePlot.setActiveToolbarTab({ key: KEY, tab: "axes" }));
      });
      expect(get.toolbarTab()).toBe("axes");
    });

    it("should lazily create the entry when the key does not exist", () => {
      const get = renderGetters(store);
      act(() =>
        store.dispatch(LinePlot.setActiveToolbarTab({ key: KEY, tab: "axes" })),
      );
      expect(get.toolbarTab()).toBe("axes");
    });
  });

  describe("setControlHold", () => {
    it("should set the hold state to true", () => {
      const get = renderGetters(store);
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(LinePlot.setControlHold({ key: KEY, hold: true }));
      });
      expect(get.control().hold).toBe(true);
    });

    it("should set the hold state to false", () => {
      const get = renderGetters(store);
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(LinePlot.setControlHold({ key: KEY, hold: false }));
      });
      expect(get.control().hold).toBe(false);
    });

    it("should toggle the hold state when value is undefined", () => {
      const get = renderGetters(store);
      act(() => store.dispatch(LinePlot.create({ key: KEY })));
      expect(get.control().hold).toBe(false);
      act(() => store.dispatch(LinePlot.setControlHold({ key: KEY })));
      expect(get.control().hold).toBe(true);
      act(() => store.dispatch(LinePlot.setControlHold({ key: KEY })));
      expect(get.control().hold).toBe(false);
    });
  });

  describe("toggleControlClickMode", () => {
    it("should set the click mode when it is currently null", () => {
      const get = renderGetters(store);
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(LinePlot.toggleControlClickMode({ key: KEY, mode: "measure" }));
      });
      expect(get.control().clickMode).toBe("measure");
    });

    it("should clear the click mode when it already matches the mode", () => {
      const get = renderGetters(store);
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(LinePlot.toggleControlClickMode({ key: KEY, mode: "measure" }));
        store.dispatch(LinePlot.toggleControlClickMode({ key: KEY, mode: "measure" }));
      });
      expect(get.control().clickMode).toBeNull();
    });

    it("should switch to the new mode when a different mode is active", () => {
      const get = renderGetters(store);
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(LinePlot.toggleControlClickMode({ key: KEY, mode: "measure" }));
        store.dispatch(LinePlot.toggleControlClickMode({ key: KEY, mode: "annotate" }));
      });
      expect(get.control().clickMode).toBe("annotate");
    });
  });

  describe("setControlEnableTooltip", () => {
    it("should set the tooltip state to true", () => {
      const get = renderGetters(store);
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(LinePlot.setControlEnableTooltip({ key: KEY, enabled: true }));
      });
      expect(get.control().enableTooltip).toBe(true);
    });

    it("should set the tooltip state to false", () => {
      const get = renderGetters(store);
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(LinePlot.setControlEnableTooltip({ key: KEY, enabled: false }));
      });
      expect(get.control().enableTooltip).toBe(false);
    });

    it("should toggle the tooltip state when value is undefined", () => {
      const get = renderGetters(store);
      act(() => store.dispatch(LinePlot.create({ key: KEY })));
      const initial = get.control().enableTooltip;
      act(() => store.dispatch(LinePlot.setControlEnableTooltip({ key: KEY })));
      expect(get.control().enableTooltip).toBe(!initial);
      act(() => store.dispatch(LinePlot.setControlEnableTooltip({ key: KEY })));
      expect(get.control().enableTooltip).toBe(initial);
    });
  });

  describe("setViewportMode", () => {
    it("should set the viewport mode", () => {
      const get = renderGetters(store);
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(LinePlot.setViewportMode({ key: KEY, mode: "pan" }));
      });
      expect(get.viewportMode()).toBe("pan");
    });
  });

  describe("viewport", () => {
    it("should reset the viewport and bump the render trigger on setViewport", () => {
      const get = renderGetters(store);
      act(() => store.dispatch(LinePlot.create({ key: KEY })));
      const before = get.state().viewport.renderTrigger;
      act(() =>
        store.dispatch(
          LinePlot.setViewport({ key: KEY, zoom: { width: 2, height: 2 } }),
        ),
      );
      const viewport = get.state().viewport;
      expect(viewport.renderTrigger).toBe(before + 1);
      expect(viewport.zoom).toEqual({ width: 2, height: 2 });
    });

    it("should merge over the existing viewport on storeViewport", () => {
      const get = renderGetters(store);
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(
          LinePlot.storeViewport({
            key: KEY,
            zoom: { width: 3, height: 3 },
            pan: { x: 1, y: 1 },
          }),
        );
      });
      const viewport = get.state().viewport;
      expect(viewport.zoom).toEqual({ width: 3, height: 3 });
      expect(viewport.pan).toEqual({ x: 1, y: 1 });
    });
  });

  describe("setSelectedRule", () => {
    it("should set the selected rules and switch to the annotations tab", () => {
      const get = renderGetters(store);
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(LinePlot.setSelectedRule({ key: KEY, ruleKey: ["r1", "r2"] }));
      });
      expect(get.selectedRules()).toEqual(["r1", "r2"]);
      expect(get.toolbarTab()).toBe("annotations");
    });

    it("should accept a single rule key", () => {
      const get = renderGetters(store);
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(LinePlot.setSelectedRule({ key: KEY, ruleKey: "r1" }));
      });
      expect(get.selectedRules()).toEqual(["r1"]);
    });
  });

  describe("setMeasureMode", () => {
    it("should set the measure mode", () => {
      const get = renderGetters(store);
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(LinePlot.setMeasureMode({ key: KEY, mode: "two" }));
      });
      expect(get.measureMode()).toBe("two");
    });
  });

  describe("setRangeAnnotationsVisible", () => {
    it("should default to visible on a newly created plot", () => {
      const get = renderGetters(store);
      act(() => store.dispatch(LinePlot.create({ key: KEY })));
      expect(get.state().annotations.visible).toBe(true);
    });

    it("should hide and re-show range annotations", () => {
      const get = renderGetters(store);
      act(() => store.dispatch(LinePlot.create({ key: KEY })));
      act(() =>
        store.dispatch(
          LinePlot.setRangeAnnotationsVisible({ key: KEY, visible: false }),
        ),
      );
      expect(get.state().annotations.visible).toBe(false);
      act(() =>
        store.dispatch(
          LinePlot.setRangeAnnotationsVisible({ key: KEY, visible: true }),
        ),
      );
      expect(get.state().annotations.visible).toBe(true);
    });
  });

  describe("setLineVisible", () => {
    it("should hide a line by adding it to hiddenLines", () => {
      const get = renderGetters(store);
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(
          LinePlot.setLineVisible({ key: KEY, lineKey: "l1", visible: false }),
        );
      });
      expect(get.hiddenLines()).toEqual(["l1"]);
    });

    it("should show a hidden line by removing it from hiddenLines", () => {
      const get = renderGetters(store);
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(
          LinePlot.setLineVisible({ key: KEY, lineKey: "l1", visible: false }),
        );
        store.dispatch(
          LinePlot.setLineVisible({ key: KEY, lineKey: "l1", visible: true }),
        );
      });
      expect(get.hiddenLines()).toEqual([]);
    });
  });

  describe("remove", () => {
    it("should remove a plot by key", () => {
      const get = renderGetters(store);
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(LinePlot.setActiveToolbarTab({ key: KEY, tab: "lines" }));
      });
      expect(get.toolbarTab()).toBe("lines");
      act(() => store.dispatch(LinePlot.remove({ keys: [KEY] })));
      expect(get.state()).toEqual(LinePlot.ZERO_STATE);
    });

    it("should ignore keys that do not exist", () => {
      const get = renderGetters(store);
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(LinePlot.remove({ keys: ["absent"] }));
      });
      expect(get.state()).toEqual(LinePlot.ZERO_STATE);
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
