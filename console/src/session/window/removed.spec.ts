// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift, MAIN_WINDOW } from "@synnaxlabs/drift";
import { describe, expect, it } from "vitest";

import { Session } from "@/session";
import { createDriftStateWithWindow } from "@/session/window/testutil";
import { createTestStore, type TestStore } from "@/testutil";

const AUX_KEY = "aux-key";
const AUX_LABEL = "aux";
const PANEL_KEY = "panel-1";
const DOC_KEY = "doc-1";

/** The window-keyed slices, every one of which must drop a removed window's entry. */
const SLICES = [
  Session.Arc.SLICE_NAME,
  Session.LinePlot.SLICE_NAME,
  Session.Log.SLICE_NAME,
  Session.Nav.SLICE_NAME,
  Session.Panel.SLICE_NAME,
  Session.Schematic.SLICE_NAME,
  Session.Table.SLICE_NAME,
] as const;

/** A store whose second window holds state of its own in every window-keyed slice. */
const createStore = async (): Promise<TestStore> => {
  const store = await createTestStore({
    preloadedState: { drift: createDriftStateWithWindow(AUX_KEY, AUX_LABEL) },
  });
  [MAIN_WINDOW, AUX_KEY].forEach((windowKey) => {
    store.dispatch(Session.Nav.showBottom({ windowKey }));
    store.dispatch(Session.Panel.select({ key: PANEL_KEY, windowKey }));
    store.dispatch(Session.Arc.create({ key: DOC_KEY, windowKey }));
    store.dispatch(Session.LinePlot.create({ key: DOC_KEY, windowKey }));
    store.dispatch(Session.Log.create({ key: DOC_KEY, windowKey }));
    store.dispatch(Session.Schematic.create({ key: DOC_KEY, windowKey }));
    store.dispatch(Session.Table.create({ key: DOC_KEY, windowKey }));
  });
  return store;
};

const windowKeys = (store: TestStore, slice: (typeof SLICES)[number]): string[] =>
  Object.keys(store.getState()[slice].windows);

describe("Window.removalMiddleware", () => {
  it("should drop the closed window's state from every window-keyed slice", async () => {
    const store = await createStore();
    SLICES.forEach((slice) =>
      expect(windowKeys(store, slice), slice).toContain(AUX_KEY),
    );
    store.dispatch(Drift.closeWindow({ key: AUX_KEY }));
    // Window keys are minted fresh per open, so an entry left behind here is one the
    // session carries forever.
    SLICES.forEach((slice) =>
      expect(windowKeys(store, slice), slice).not.toContain(AUX_KEY),
    );
  });

  // Drift keeps a window with a process registered open, so a close it defers must
  // not take the state of a window the user is still looking at.
  it("should keep the state of a window whose close is deferred", async () => {
    const store = await createStore();
    store.dispatch(Drift.registerProcess({ key: AUX_KEY }));
    store.dispatch(Drift.closeWindow({ key: AUX_KEY }));
    expect(Drift.selectWindow(store.getState(), AUX_KEY)).not.toBeNull();
    SLICES.forEach((slice) =>
      expect(windowKeys(store, slice), slice).toContain(AUX_KEY),
    );
  });

  it("should drop the state once the deferred close lands", async () => {
    const store = await createStore();
    store.dispatch(Drift.registerProcess({ key: AUX_KEY }));
    store.dispatch(Drift.closeWindow({ key: AUX_KEY }));
    store.dispatch(Drift.completeProcess({ key: AUX_KEY }));
    expect(Drift.selectWindow(store.getState(), AUX_KEY)).toBeNull();
    SLICES.forEach((slice) =>
      expect(windowKeys(store, slice), slice).not.toContain(AUX_KEY),
    );
  });

  // Main's key is the one key that recurs across launches, so quitting the app must
  // not take its state with it.
  it("should keep the main window's state when the main window closes", async () => {
    const store = await createStore();
    store.dispatch(Drift.closeWindow({ key: MAIN_WINDOW }));
    SLICES.forEach((slice) =>
      expect(windowKeys(store, slice), slice).toContain(MAIN_WINDOW),
    );
  });

  it("should leave the windows still open untouched", async () => {
    const store = await createStore();
    store.dispatch(Drift.closeWindow({ key: AUX_KEY }));
    SLICES.forEach((slice) =>
      expect(windowKeys(store, slice), slice).toEqual([MAIN_WINDOW]),
    );
    expect(store.getState().panels.windows[MAIN_WINDOW].selected).toBe(PANEL_KEY);
  });
});
