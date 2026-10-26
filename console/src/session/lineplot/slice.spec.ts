// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep } from "@synnaxlabs/x";
import { act } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";

import { LinePlot } from "@/session/lineplot";
import { createSliceStore, documentIn, inWindow } from "@/session/window/testutil";

const storeWith = (slice: LinePlot.SliceState) =>
  createSliceStore({
    name: LinePlot.SLICE_NAME,
    reducer: LinePlot.reducer,
    preloadedState: slice,
    middleware: LinePlot.MIDDLEWARE,
  });

const KEY = "plot-1";
const AUX_WINDOW = "aux-window";

describe("LinePlot Slice", () => {
  let store: ReturnType<typeof storeWith>;

  beforeEach(() => {
    store = storeWith(LinePlot.ZERO_SLICE_STATE);
  });

  const docOf = (key: string = KEY) =>
    documentIn(store.getState()[LinePlot.SLICE_NAME], key);

  // A window is a viewport: an edit in one window must not reach another window's
  // view of the same document.
  it("should keep each window's view of a document independent", () => {
    store.dispatch(LinePlot.create({ key: KEY }));
    store.dispatch(LinePlot.create({ key: KEY, windowKey: AUX_WINDOW }));
    store.dispatch(
      LinePlot.setActiveToolbarTab({ key: KEY, tab: "axes", windowKey: AUX_WINDOW }),
    );
    const slice = store.getState()[LinePlot.SLICE_NAME];
    expect(documentIn(slice, KEY, AUX_WINDOW)?.toolbar.activeTab).toBe("axes");
    expect(documentIn(slice, KEY)?.toolbar.activeTab).toBe("data");
  });

  describe("create", () => {
    it("should bootstrap session state from ZERO_STATE for the key", () => {
      act(() => void store.dispatch(LinePlot.create({ key: KEY })));
      expect(docOf()).toEqual(LinePlot.ZERO_STATE);
    });

    it("should create multiple plots independently", () => {
      act(() => {
        store.dispatch(LinePlot.create({ key: "plot-1" }));
        store.dispatch(LinePlot.create({ key: "plot-2" }));
        store.dispatch(LinePlot.setActiveToolbarTab({ key: "plot-1", tab: "lines" }));
        store.dispatch(LinePlot.setActiveToolbarTab({ key: "plot-2", tab: "axes" }));
      });
      expect(docOf("plot-1")?.toolbar.activeTab).toBe("lines");
      expect(docOf("plot-2")?.toolbar.activeTab).toBe("axes");
    });

    it("should fill annotations with the default for a key", () => {
      act(() => void store.dispatch(LinePlot.create({ key: KEY })));
      expect(docOf()?.annotations).toEqual(LinePlot.ZERO_STATE.annotations);
    });

    it("should not overwrite an existing entry", () => {
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(LinePlot.setActiveToolbarTab({ key: KEY, tab: "lines" }));
        store.dispatch(LinePlot.create({ key: KEY }));
      });
      expect(docOf()?.toolbar.activeTab).toBe("lines");
    });
  });

  describe("setActiveToolbarTab", () => {
    it("should set the active toolbar tab", () => {
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(LinePlot.setActiveToolbarTab({ key: KEY, tab: "axes" }));
      });
      expect(docOf()?.toolbar.activeTab).toBe("axes");
    });

    it("should lazily create the entry when the key does not exist", () => {
      act(
        () =>
          void store.dispatch(LinePlot.setActiveToolbarTab({ key: KEY, tab: "axes" })),
      );
      expect(docOf()?.toolbar.activeTab).toBe("axes");
    });
  });

  describe("setControlHold", () => {
    it("should set the hold state to true", () => {
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(LinePlot.setControlHold({ key: KEY, hold: true }));
      });
      expect(docOf()?.control.hold).toBe(true);
    });

    it("should set the hold state to false", () => {
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(LinePlot.setControlHold({ key: KEY, hold: false }));
      });
      expect(docOf()?.control.hold).toBe(false);
    });

    it("should toggle the hold state when value is undefined", () => {
      act(() => void store.dispatch(LinePlot.create({ key: KEY })));
      expect(docOf()?.control.hold).toBe(false);
      act(() => void store.dispatch(LinePlot.setControlHold({ key: KEY })));
      expect(docOf()?.control.hold).toBe(true);
      act(() => void store.dispatch(LinePlot.setControlHold({ key: KEY })));
      expect(docOf()?.control.hold).toBe(false);
    });
  });

  describe("toggleControlClickMode", () => {
    it("should set the click mode when it is currently null", () => {
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(LinePlot.toggleControlClickMode({ key: KEY, mode: "measure" }));
      });
      expect(docOf()?.control.clickMode).toBe("measure");
    });

    it("should clear the click mode when it already matches the mode", () => {
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(LinePlot.toggleControlClickMode({ key: KEY, mode: "measure" }));
        store.dispatch(LinePlot.toggleControlClickMode({ key: KEY, mode: "measure" }));
      });
      expect(docOf()?.control.clickMode).toBeNull();
    });

    it("should switch to the new mode when a different mode is active", () => {
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(LinePlot.toggleControlClickMode({ key: KEY, mode: "measure" }));
        store.dispatch(LinePlot.toggleControlClickMode({ key: KEY, mode: "annotate" }));
      });
      expect(docOf()?.control.clickMode).toBe("annotate");
    });
  });

  describe("setControlEnableTooltip", () => {
    it("should set the tooltip state to true", () => {
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(LinePlot.setControlEnableTooltip({ key: KEY, enabled: true }));
      });
      expect(docOf()?.control.enableTooltip).toBe(true);
    });

    it("should set the tooltip state to false", () => {
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(LinePlot.setControlEnableTooltip({ key: KEY, enabled: false }));
      });
      expect(docOf()?.control.enableTooltip).toBe(false);
    });

    it("should toggle the tooltip state when value is undefined", () => {
      act(() => void store.dispatch(LinePlot.create({ key: KEY })));
      const initial = docOf()?.control.enableTooltip;
      act(() => void store.dispatch(LinePlot.setControlEnableTooltip({ key: KEY })));
      expect(docOf()?.control.enableTooltip).toBe(!initial);
      act(() => void store.dispatch(LinePlot.setControlEnableTooltip({ key: KEY })));
      expect(docOf()?.control.enableTooltip).toBe(initial);
    });
  });

  describe("setViewportMode", () => {
    it("should set the viewport mode", () => {
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(LinePlot.setViewportMode({ key: KEY, mode: "pan" }));
      });
      expect(docOf()?.mode).toBe("pan");
    });
  });

  describe("viewport", () => {
    const viewportOf = () => {
      const viewport = docOf()?.viewport;
      if (viewport == null) throw new Error(`no line plot ${KEY}`);
      return viewport;
    };

    it("should reset the viewport and bump the render trigger on setViewport", () => {
      act(() => void store.dispatch(LinePlot.create({ key: KEY })));
      const before = viewportOf().renderTrigger;
      act(
        () =>
          void store.dispatch(
            LinePlot.setViewport({ key: KEY, zoom: { width: 2, height: 2 } }),
          ),
      );
      const viewport = viewportOf();
      expect(viewport.renderTrigger).toBe(before + 1);
      expect(viewport.zoom).toEqual({ width: 2, height: 2 });
    });

    it("should merge over the existing viewport on storeViewport", () => {
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
      const viewport = viewportOf();
      expect(viewport.zoom).toEqual({ width: 3, height: 3 });
      expect(viewport.pan).toEqual({ x: 1, y: 1 });
    });
  });

  describe("setSelectedRule", () => {
    it("should set the selected rules and switch to the annotations tab", () => {
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(LinePlot.setSelectedRule({ key: KEY, ruleKey: ["r1", "r2"] }));
      });
      expect(docOf()?.selectedRules).toEqual(["r1", "r2"]);
      expect(docOf()?.toolbar.activeTab).toBe("annotations");
    });

    it("should accept a single rule key", () => {
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(LinePlot.setSelectedRule({ key: KEY, ruleKey: "r1" }));
      });
      expect(docOf()?.selectedRules).toEqual(["r1"]);
    });
  });

  describe("setMeasureMode", () => {
    it("should set the measure mode", () => {
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(LinePlot.setMeasureMode({ key: KEY, mode: "two" }));
      });
      expect(docOf()?.measure.mode).toBe("two");
    });
  });

  describe("setRangeAnnotationsVisible", () => {
    it("should default to visible on a newly created plot", () => {
      act(() => void store.dispatch(LinePlot.create({ key: KEY })));
      expect(docOf()?.annotations.visible).toBe(true);
    });

    it("should hide and re-show range annotations", () => {
      act(() => void store.dispatch(LinePlot.create({ key: KEY })));
      act(
        () =>
          void store.dispatch(
            LinePlot.setRangeAnnotationsVisible({ key: KEY, visible: false }),
          ),
      );
      expect(docOf()?.annotations.visible).toBe(false);
      act(
        () =>
          void store.dispatch(
            LinePlot.setRangeAnnotationsVisible({ key: KEY, visible: true }),
          ),
      );
      expect(docOf()?.annotations.visible).toBe(true);
    });
  });

  describe("setLineVisible", () => {
    it("should hide a line by adding it to hiddenLines", () => {
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(
          LinePlot.setLineVisible({ key: KEY, lineKey: "l1", visible: false }),
        );
      });
      expect(docOf()?.hiddenLines).toEqual(["l1"]);
    });

    it("should show a hidden line by removing it from hiddenLines", () => {
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(
          LinePlot.setLineVisible({ key: KEY, lineKey: "l1", visible: false }),
        );
        store.dispatch(
          LinePlot.setLineVisible({ key: KEY, lineKey: "l1", visible: true }),
        );
      });
      expect(docOf()?.hiddenLines).toEqual([]);
    });
  });

  describe("remove", () => {
    it("should remove a plot by key", () => {
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(LinePlot.setActiveToolbarTab({ key: KEY, tab: "lines" }));
      });
      expect(docOf()?.toolbar.activeTab).toBe("lines");
      act(() => void store.dispatch(LinePlot.remove({ keys: [KEY] })));
      expect(docOf()).toBeUndefined();
    });

    it("should ignore keys that do not exist", () => {
      act(() => {
        store.dispatch(LinePlot.create({ key: KEY }));
        store.dispatch(LinePlot.remove({ keys: ["absent"] }));
      });
      expect(docOf()).toEqual(LinePlot.ZERO_STATE);
    });
  });

  it("should apply defaults to every nested object", () => {
    const zero = LinePlot.ZERO_STATE;
    expect(zero.control.hold).toBe(false);
    expect(zero.control.enableTooltip).toBe(true);
    expect(zero.toolbar.activeTab).toBe("data");
    expect(zero.measure.mode).toBe("one");
    expect(zero.annotations.visible).toBe(true);
    expect(zero.selectedRules).toEqual([]);
    expect(zero.hiddenLines).toEqual([]);
  });
});

describe("persistence", () => {
  const purge = (doc: Partial<LinePlot.State>): LinePlot.State | undefined => {
    const slice: LinePlot.SliceState = {
      version: 0,
      windows: inWindow({ [KEY]: { ...LinePlot.ZERO_STATE, ...doc } }),
    };
    const state = deep.copy(storeWith(slice).getState());
    LinePlot.PERSIST_EXCLUDE.forEach((exclude) => exclude(state));
    return documentIn(state[LinePlot.SLICE_NAME], KEY);
  };

  // A selection is the state of an edit in progress, so it does not outlive the
  // session that made it.
  it("should clear the selection on the way to disk", () => {
    expect(purge({ selectedRules: ["r1", "r2"] })?.selectedRules).toEqual([]);
  });

  it("should leave the rest of the document alone", () => {
    // Hidden lines are how the window looks at the plot, so they survive.
    expect(purge({ selectedRules: ["r1"], hiddenLines: ["l1"] })?.hiddenLines).toEqual([
      "l1",
    ]);
  });
});
