// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift, MAIN_WINDOW } from "@synnaxlabs/drift";
import { type Haul } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import { waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it } from "vitest";

import { Session } from "@/session";
import { createTestStore, type TestStore } from "@/testutil";

const HAULED: Haul.DraggingState = {
  source: { key: "file", type: "file" },
  items: [{ key: "a", type: "b" }],
};

const withoutDrift = (state: Session.State): Partial<Session.State> => {
  const copy: Partial<Session.State> = { ...state };
  delete copy[Drift.SLICE_NAME];
  return copy;
};

/**
 * Every payload production has written under the persisted store's path. Partitioning
 * is the engine's business, so this asserts over all of them rather than naming keys.
 */
const readPersisted = (): Array<Partial<Session.State>> =>
  Object.entries(localStorage)
    .filter(([key]) => key.startsWith(`${Session.Persist.STORE_PATH}:`))
    .map(([, value]) => JSON.parse(value) as Partial<Session.State>);

const waitForPersisted = async (
  predicate: (payload: Partial<Session.State>) => boolean,
  message: string,
): Promise<void> =>
  await waitFor(() => {
    if (!readPersisted().some(predicate)) throw new Error(message);
  });

describe("createStore", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  it("initializes every slice to its zero state", async () => {
    const store = await Session.createStore({
      enablePersistence: false,
      enablePrerender: false,
    });
    expect(Object.keys(store.getState()).sort()).toEqual(
      Object.keys(Session.ZERO_STATE).sort(),
    );
    expect(withoutDrift(store.getState())).toStrictEqual(
      withoutDrift(Session.ZERO_STATE),
    );
  });

  it("routes dispatched actions to their owning slice", async () => {
    const store = await Session.createStore({
      enablePersistence: false,
      enablePrerender: false,
    });
    store.dispatch(Session.Cluster.select("DEMO"));
    store.dispatch(Session.Nav.showBottom({ windowKey: MAIN_WINDOW }));
    expect(Session.Cluster.selectSelectedKey(store.getState())).toBe("DEMO");
    expect(Session.Nav.selectWindowState(store.getState()).bottom.visible).toBe(true);
  });

  it("honors an explicit preloadedState", async () => {
    const store = await Session.createStore({
      enablePersistence: false,
      enablePrerender: false,
      preloadedState: deep.copy({
        ...Session.ZERO_STATE,
        [Session.Cluster.SLICE_NAME]: {
          ...Session.Cluster.ZERO_SLICE_STATE,
          selected: "DEMO",
        },
      }),
    });
    expect(Session.Cluster.selectSelectedKey(store.getState())).toBe("DEMO");
  });

  it("persists state and reloads it into a fresh store", async () => {
    const store = await Session.createStore({ enablePrerender: false });
    store.dispatch(Session.Cluster.select("DEMO"));
    await waitForPersisted(
      (p) => p.cluster?.selected === "DEMO",
      "cluster selection not persisted yet",
    );
    const reloaded = await Session.createStore({ enablePrerender: false });
    expect(Session.Cluster.selectSelectedKey(reloaded.getState())).toBe("DEMO");
  });

  it("excludes transient haul state from persistence", async () => {
    const projectKey = "6f7cd5f4-4b93-4a35-a55c-72ba9dae2c9d";
    const store = await Session.createStore({ enablePrerender: false });
    store.dispatch(Session.Cluster.select("DEMO"));
    store.dispatch(Session.Project.select(projectKey));
    // The switch hydrates the project's stored slices over whatever is in the store, so
    // let it settle before making changes that have to survive. Project-scoped slices
    // are only written once the hydrate lands, so nav appearing marks the end of it.
    await waitForPersisted((p) => p.nav != null, "project swap has not settled yet");
    store.dispatch(Session.Haul.setHauled(HAULED));
    store.dispatch(Session.Nav.showBottom({ windowKey: MAIN_WINDOW }));
    await waitForPersisted(
      (p) => Object.values(p.nav?.windows ?? {})[0]?.bottom.visible === true,
      "nav change not persisted yet",
    );
    expect(store.getState().haul.state).toStrictEqual(HAULED);
    // The drag rode to disk alongside the nav change that just landed.
    readPersisted().forEach((p) => {
      if (p.haul != null) expect(p.haul).toStrictEqual(Session.Haul.ZERO_SLICE_STATE);
    });
  });
});

// A pre-render webview's label never resolves a window key, so window-scoped
// dispatches from it must die locally instead of mutating state or leaking to
// other windows.
describe("pre-rendered windows", () => {
  const PRERENDER_LABEL = "prerender-label";

  const createPrerenderStore = async (): Promise<TestStore> =>
    await createTestStore({ windowLabel: PRERENDER_LABEL });

  it("never becomes visible from the shell's self-show dispatch", async () => {
    const store = await createPrerenderStore();
    store.dispatch(
      Drift.setWindowProps({ visible: true, minimized: false, decorations: true }),
    );
    expect(
      Drift.selectSliceState(store.getState()).windows[PRERENDER_LABEL],
    ).toBeUndefined();
  });

  it("drops window-scoped actions instead of applying them elsewhere", async () => {
    const store = await createPrerenderStore();
    store.dispatch(Session.Panel.select({ key: "some-panel" }));
    expect(store.getState().panels.windows).toEqual({});
  });

  it("still shows a claimed window from the same dispatch", async () => {
    const store = await createTestStore();
    store.dispatch(
      Drift.setWindowProps({ visible: true, minimized: false, decorations: true }),
    );
    expect(
      Drift.selectSliceState(store.getState()).windows[MAIN_WINDOW].visible,
    ).toEqual(true);
  });
});

const createWindow = (key: string): Drift.WindowState => ({
  ...Drift.ZERO_SLICE_STATE.windows[MAIN_WINDOW],
  key,
});

const createDriftState = (
  label: string,
  windows: Record<string, Drift.WindowState>,
): Drift.SliceState => ({ ...Drift.ZERO_SLICE_STATE, label, windows });

describe("reducer", () => {
  it("hydrates stored windows without disturbing the running one", () => {
    const live: Session.State = {
      ...Session.ZERO_STATE,
      [Drift.SLICE_NAME]: createDriftState("live", {
        [MAIN_WINDOW]: createWindow(MAIN_WINDOW),
        outgoing: createWindow("outgoing"),
      }),
    };
    const stored = createDriftState("stale", {
      [MAIN_WINDOW]: createWindow(MAIN_WINDOW),
      incoming: createWindow("incoming"),
    });
    const next = Session.reducer(
      live,
      Session.Persist.hydrate({ [Drift.SLICE_NAME]: stored }),
    );
    const drift = next[Drift.SLICE_NAME];
    expect(drift.label).toEqual("live");
    expect(drift.windows[MAIN_WINDOW]).toEqual(
      live[Drift.SLICE_NAME].windows[MAIN_WINDOW],
    );
    expect(drift.windows.outgoing).toBeUndefined();
    expect(drift.windows.incoming?.key).toEqual("incoming");
  });

  it("replaces non-drift slices wholesale", () => {
    const next = Session.reducer(
      Session.ZERO_STATE,
      Session.Persist.hydrate({
        [Session.Cluster.SLICE_NAME]: {
          ...Session.Cluster.ZERO_SLICE_STATE,
          selected: "DEMO",
        },
      }),
    );
    expect(Session.Cluster.selectSelectedKey(next)).toBe("DEMO");
  });
});
