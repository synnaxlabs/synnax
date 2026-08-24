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

/** A store whose second window holds nav and panel state of its own. */
const createStore = async (): Promise<TestStore> => {
  const store = await createTestStore({
    preloadedState: { drift: createDriftStateWithWindow(AUX_KEY, AUX_LABEL) },
  });
  [MAIN_WINDOW, AUX_KEY].forEach((windowKey) => {
    store.dispatch(Session.Nav.showBottom({ windowKey }));
    store.dispatch(Session.Panel.select({ key: PANEL_KEY, windowKey }));
  });
  return store;
};

const windowKeys = (store: TestStore): { nav: string[]; panels: string[] } => {
  const state = store.getState();
  return {
    nav: Object.keys(state.nav.windows),
    panels: Object.keys(state.panels.windows),
  };
};

describe("Window.removalMiddleware", () => {
  it("should drop the closed window's nav and panel state", async () => {
    const store = await createStore();
    expect(windowKeys(store).nav).toContain(AUX_KEY);
    expect(windowKeys(store).panels).toContain(AUX_KEY);
    store.dispatch(Drift.closeWindow({ key: AUX_KEY }));
    // Window keys are minted fresh per open, so an entry left behind here is one the
    // session carries forever.
    expect(windowKeys(store).nav).not.toContain(AUX_KEY);
    expect(windowKeys(store).panels).not.toContain(AUX_KEY);
  });

  // Drift keeps a window with a process registered open, so a close it defers must
  // not take the state of a window the user is still looking at.
  it("should keep the state of a window whose close is deferred", async () => {
    const store = await createStore();
    store.dispatch(Drift.registerProcess({ key: AUX_KEY }));
    store.dispatch(Drift.closeWindow({ key: AUX_KEY }));
    expect(Drift.selectWindow(store.getState(), AUX_KEY)).not.toBeNull();
    expect(windowKeys(store).nav).toContain(AUX_KEY);
    expect(windowKeys(store).panels).toContain(AUX_KEY);
  });

  it("should drop the state once the deferred close lands", async () => {
    const store = await createStore();
    store.dispatch(Drift.registerProcess({ key: AUX_KEY }));
    store.dispatch(Drift.closeWindow({ key: AUX_KEY }));
    store.dispatch(Drift.completeProcess({ key: AUX_KEY }));
    expect(Drift.selectWindow(store.getState(), AUX_KEY)).toBeNull();
    expect(windowKeys(store).nav).not.toContain(AUX_KEY);
    expect(windowKeys(store).panels).not.toContain(AUX_KEY);
  });

  it("should leave the windows still open untouched", async () => {
    const store = await createStore();
    store.dispatch(Drift.closeWindow({ key: AUX_KEY }));
    expect(windowKeys(store).nav).toEqual([MAIN_WINDOW]);
    expect(windowKeys(store).panels).toEqual([MAIN_WINDOW]);
    expect(store.getState().panels.windows[MAIN_WINDOW].selected).toBe(PANEL_KEY);
  });
});
