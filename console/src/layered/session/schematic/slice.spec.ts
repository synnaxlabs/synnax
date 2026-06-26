// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";
import { color } from "@synnaxlabs/x";
import { beforeEach, describe, expect, it } from "vitest";

import { Schematic } from "@/layered/session/schematic";

const storeWith = (slice: Schematic.SliceState) =>
  configureStore({
    reducer: { [Schematic.SLICE_NAME]: Schematic.reducer },
    preloadedState: { [Schematic.SLICE_NAME]: slice },
  });

const KEY = "schematic-1";

describe("Schematic Slice", () => {
  let store: ReturnType<typeof storeWith>;

  beforeEach(() => {
    store = storeWith(Schematic.ZERO_SLICE_STATE);
  });

  const select = <R>(
    selector: (params: Schematic.KeyedSelectorParams) => R,
    key: string = KEY,
  ): R => selector({ state: store.getState(), key });

  describe("create", () => {
    it("should bootstrap session state from ZERO_STATE for the key", () => {
      store.dispatch(Schematic.internalCreate({ key: KEY }));
      expect(select(Schematic.selectState)).toEqual(Schematic.ZERO_STATE);
    });

    it("should create multiple schematics independently", () => {
      store.dispatch(Schematic.internalCreate({ key: "schematic-1" }));
      store.dispatch(Schematic.internalCreate({ key: "schematic-2" }));
      expect(
        Object.keys(Schematic.selectSliceState(store.getState()).schematics),
      ).toHaveLength(2);
    });

    it("should apply provided fields over the defaults", () => {
      store.dispatch(Schematic.internalCreate({ key: KEY, editable: true }));
      expect(select(Schematic.selectEditable)).toBe(true);
    });

    it("should not overwrite an existing entry", () => {
      store.dispatch(Schematic.internalCreate({ key: KEY }));
      store.dispatch(Schematic.setEditable({ key: KEY, editable: true }));
      store.dispatch(Schematic.internalCreate({ key: KEY }));
      expect(select(Schematic.selectEditable)).toBe(true);
    });
  });

  describe("setSelected", () => {
    it("should set the selected elements", () => {
      store.dispatch(Schematic.internalCreate({ key: KEY }));
      store.dispatch(Schematic.setSelected({ key: KEY, selected: ["a", "b"] }));
      expect(select(Schematic.selectSelected)).toEqual(["a", "b"]);
    });

    it("should switch the toolbar to properties when something is selected", () => {
      store.dispatch(Schematic.internalCreate({ key: KEY }));
      store.dispatch(Schematic.setSelected({ key: KEY, selected: ["a"] }));
      expect(select(Schematic.selectActiveToolbarTab)).toBe("properties");
    });

    it("should switch the toolbar back to symbols when nothing is selected", () => {
      store.dispatch(Schematic.internalCreate({ key: KEY }));
      store.dispatch(Schematic.setSelected({ key: KEY, selected: ["a"] }));
      store.dispatch(Schematic.setSelected({ key: KEY, selected: [] }));
      expect(select(Schematic.selectActiveToolbarTab)).toBe("symbols");
    });

    it("should lazily create the entry when the key does not exist", () => {
      store.dispatch(Schematic.setSelected({ key: KEY, selected: ["a"] }));
      expect(select(Schematic.selectSelected)).toEqual(["a"]);
    });
  });

  describe("setControlStatus", () => {
    it("should set the control status", () => {
      store.dispatch(Schematic.internalCreate({ key: KEY }));
      store.dispatch(Schematic.setControlStatus({ key: KEY, status: "acquired" }));
      expect(select(Schematic.selectControlStatus)).toBe("acquired");
    });

    it("should clear the selection and disable editing when control is acquired", () => {
      store.dispatch(Schematic.internalCreate({ key: KEY }));
      store.dispatch(Schematic.setSelected({ key: KEY, selected: ["a"] }));
      store.dispatch(Schematic.setEditable({ key: KEY, editable: true }));
      store.dispatch(Schematic.setControlStatus({ key: KEY, status: "acquired" }));
      expect(select(Schematic.selectSelected)).toEqual([]);
      expect(select(Schematic.selectEditable)).toBe(false);
    });

    it("should not touch selection or editing for non-acquired statuses", () => {
      store.dispatch(Schematic.internalCreate({ key: KEY }));
      store.dispatch(Schematic.setSelected({ key: KEY, selected: ["a"] }));
      store.dispatch(Schematic.setEditable({ key: KEY, editable: true }));
      store.dispatch(Schematic.setControlStatus({ key: KEY, status: "released" }));
      expect(select(Schematic.selectSelected)).toEqual(["a"]);
      expect(select(Schematic.selectEditable)).toBe(true);
    });
  });

  describe("setControlAuthority", () => {
    it("should set the control authority", () => {
      store.dispatch(Schematic.internalCreate({ key: KEY }));
      store.dispatch(Schematic.setControlAuthority({ key: KEY, authority: 200 }));
      expect(select(Schematic.selectAuthority)).toBe(200);
    });
  });

  describe("legend", () => {
    it("should move the legend", () => {
      const position = {
        x: 5,
        y: 6,
        root: { x: "left", y: "top" },
        units: { x: "px", y: "px" },
      } as const;
      store.dispatch(Schematic.internalCreate({ key: KEY }));
      store.dispatch(Schematic.moveLegend({ key: KEY, position }));
      expect(select(Schematic.selectLegend).position).toEqual(position);
    });

    it("should set the legend colors", () => {
      const colors = { a: color.ZERO };
      store.dispatch(Schematic.internalCreate({ key: KEY }));
      store.dispatch(Schematic.setLegendColors({ key: KEY, colors }));
      expect(select(Schematic.selectLegend).colors).toEqual(colors);
    });

    it("should set the legend visibility", () => {
      store.dispatch(Schematic.internalCreate({ key: KEY }));
      store.dispatch(Schematic.setLegendVisible({ key: KEY, visible: false }));
      expect(select(Schematic.selectLegendVisible)).toBe(false);
    });
  });

  describe("toolbar", () => {
    it("should set the active toolbar tab", () => {
      store.dispatch(Schematic.internalCreate({ key: KEY }));
      store.dispatch(Schematic.setActiveToolbarTab({ key: KEY, tab: "properties" }));
      expect(select(Schematic.selectActiveToolbarTab)).toBe("properties");
    });

    it("should set the selected symbol group", () => {
      store.dispatch(Schematic.internalCreate({ key: KEY }));
      store.dispatch(Schematic.setSelectedSymbolGroup({ key: KEY, group: "valves" }));
      expect(select(Schematic.selectSelectedSymbolGroup)).toBe("valves");
    });
  });

  describe("setEditable", () => {
    it("should enable editing", () => {
      store.dispatch(Schematic.internalCreate({ key: KEY }));
      store.dispatch(Schematic.setEditable({ key: KEY, editable: true }));
      expect(select(Schematic.selectEditable)).toBe(true);
    });

    it("should clear the selection when editing is disabled", () => {
      store.dispatch(Schematic.internalCreate({ key: KEY }));
      store.dispatch(Schematic.setSelected({ key: KEY, selected: ["a"] }));
      store.dispatch(Schematic.setEditable({ key: KEY, editable: false }));
      expect(select(Schematic.selectSelected)).toEqual([]);
    });
  });

  describe("setFitViewOnResize", () => {
    it("should set fit view on resize", () => {
      store.dispatch(Schematic.internalCreate({ key: KEY }));
      store.dispatch(Schematic.setFitViewOnResize({ key: KEY, fitViewOnResize: true }));
      expect(select(Schematic.selectFitViewOnResize)).toBe(true);
    });
  });

  describe("viewport", () => {
    it("should merge the viewport over the existing one", () => {
      store.dispatch(Schematic.internalCreate({ key: KEY }));
      store.dispatch(
        Schematic.setViewport({
          key: KEY,
          viewport: { position: { x: 10, y: 20 }, zoom: 3 },
        }),
      );
      const viewport = select(Schematic.selectViewport);
      expect(viewport.position).toEqual({ x: 10, y: 20 });
      expect(viewport.zoom).toBe(3);
      expect(viewport.mode).toBe(Schematic.ZERO_STATE.viewport.mode);
    });

    it("should set the viewport mode without touching position or zoom", () => {
      store.dispatch(Schematic.internalCreate({ key: KEY }));
      store.dispatch(Schematic.setViewportMode({ key: KEY, mode: "pan" }));
      const viewport = select(Schematic.selectViewport);
      expect(viewport.mode).toBe("pan");
      expect(viewport.position).toEqual(Schematic.ZERO_STATE.viewport.position);
    });
  });

  describe("remove", () => {
    it("should remove a schematic by key", () => {
      store.dispatch(Schematic.internalCreate({ key: KEY }));
      store.dispatch(Schematic.remove({ keys: [KEY] }));
      expect(
        Schematic.selectSliceState(store.getState()).schematics[KEY],
      ).toBeUndefined();
    });

    it("should remove multiple schematics at once", () => {
      store.dispatch(Schematic.internalCreate({ key: "schematic-1" }));
      store.dispatch(Schematic.internalCreate({ key: "schematic-2" }));
      store.dispatch(Schematic.remove({ keys: ["schematic-1", "schematic-2"] }));
      expect(
        Object.keys(Schematic.selectSliceState(store.getState()).schematics),
      ).toHaveLength(0);
    });

    it("should ignore keys that do not exist", () => {
      store.dispatch(Schematic.internalCreate({ key: KEY }));
      store.dispatch(Schematic.remove({ keys: ["absent"] }));
      expect(select(Schematic.selectState)).toEqual(Schematic.ZERO_STATE);
    });
  });

  describe("stateZ schema", () => {
    it("should accept the zero state", () => {
      expect(() => Schematic.stateZ.parse(Schematic.ZERO_STATE)).not.toThrow();
    });

    it("should apply prefaulted defaults when nested objects are missing", () => {
      const parsed = Schematic.stateZ.parse({});
      expect(parsed.control.authority).toBe(1);
      expect(parsed.control.status).toBe("released");
      expect(parsed.legend.visible).toBe(true);
      expect(parsed.toolbar.selectedTab).toBe("symbols");
      expect(parsed.viewport.zoom).toBe(1);
      expect(parsed.editable).toBe(false);
      expect(parsed.fitViewOnResize).toBe(false);
      expect(parsed.selected).toEqual([]);
    });
  });

  describe("purgeState", () => {
    it("should reset the control status to released", () => {
      const state = Schematic.stateZ.parse({ control: { status: "acquired" } });
      expect(Schematic.purgeState(state).control.status).toBe("released");
    });

    it("should leave other fields untouched", () => {
      const state = Schematic.stateZ.parse({
        control: { status: "acquired" },
        editable: true,
      });
      expect(Schematic.purgeState(state).editable).toBe(true);
    });
  });

  describe("purgeSliceState", () => {
    it("should release control on every schematic in the slice", () => {
      const state = {
        [Schematic.SLICE_NAME]: {
          schematics: {
            "schematic-1": Schematic.stateZ.parse({ control: { status: "acquired" } }),
            "schematic-2": Schematic.stateZ.parse({ control: { status: "acquired" } }),
          },
        },
      };
      const purged = Schematic.purgeSliceState(state);
      expect(
        purged[Schematic.SLICE_NAME].schematics["schematic-1"].control.status,
      ).toBe("released");
      expect(
        purged[Schematic.SLICE_NAME].schematics["schematic-2"].control.status,
      ).toBe("released");
    });
  });
});
