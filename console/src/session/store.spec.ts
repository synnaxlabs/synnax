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
import { beforeEach, describe, expect, it, vi } from "vitest";

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
 * Every payload production has written into the store the given KV backs.
 * Partitioning is the engine's business, so this asserts over all of them rather
 * than naming keys.
 */
const readPersisted = (db: Session.Persist.MemoryKV): Array<Partial<Session.State>> =>
  [...db.store.entries()]
    .filter(([key]) => !key.endsWith(".slot"))
    .map(([, value]) => value as Partial<Session.State>);

const waitForPersisted = async (
  db: Session.Persist.MemoryKV,
  predicate: (payload: Partial<Session.State>) => boolean,
  message: string,
): Promise<void> =>
  await waitFor(() => {
    if (!readPersisted(db).some(predicate)) throw new Error(message);
  });

// State partitions by the cluster a Core connects to, which a spec has to supply
// itself: nothing here connects.
const CLUSTER_KEY = "9a1f7b2c-0d3e-4f56-8a9b-0c1d2e3f4a5b";

const PANEL = { key: "0c9d1a3b-6f8e-4d21-9b77-2f5e8c4a1d60", name: "Overview" };

describe("createStore", () => {
  let db: Session.Persist.MemoryKV;

  beforeEach(() => {
    db = new Session.Persist.MemoryKV();
  });

  const createStore = async (opts: Session.CreateStoreOptions = {}) =>
    await Session.createStore({ enablePrerender: false, openKV: () => db, ...opts });

  it("initializes every slice to its zero state", async () => {
    const store = await createStore({ enablePersistence: false });
    expect(Object.keys(store.getState()).sort()).toEqual(
      Object.keys(Session.ZERO_STATE).sort(),
    );
    expect(withoutDrift(store.getState())).toStrictEqual(
      withoutDrift(Session.ZERO_STATE),
    );
  });

  it("routes dispatched actions to their owning slice", async () => {
    const store = await createStore({ enablePersistence: false });
    store.dispatch(Session.Core.select(Session.Core.DEMO_KEY));
    store.dispatch(Session.Nav.showBottom({ windowKey: MAIN_WINDOW }));
    expect(Session.Core.selectSelectedKey(store.getState())).toBe(
      Session.Core.DEMO_KEY,
    );
    expect(Session.Nav.selectWindowState(store.getState()).bottom.visible).toBe(true);
  });

  it("honors an explicit preloadedState", async () => {
    const store = await createStore({
      enablePersistence: false,
      preloadedState: deep.copy({
        ...Session.ZERO_STATE,
        [Session.Core.SLICE_NAME]: {
          ...Session.Core.ZERO_SLICE_STATE,
          selected: Session.Core.DEMO_KEY,
        },
      }),
    });
    expect(Session.Core.selectSelectedKey(store.getState())).toBe(
      Session.Core.DEMO_KEY,
    );
  });

  it("persists state and reloads it into a fresh store", async () => {
    const store = await createStore();
    store.dispatch(Session.Core.select(Session.Core.DEMO_KEY));
    await waitForPersisted(
      db,
      (p) => p.core?.selected === Session.Core.DEMO_KEY,
      "Core selection not persisted yet",
    );
    const reloaded = await createStore();
    expect(Session.Core.selectSelectedKey(reloaded.getState())).toBe(
      Session.Core.DEMO_KEY,
    );
  });

  it("round-trips a Core saved before its first login", async () => {
    const key = "3c1d9b2e-5a47-4f08-9d16-7e2b8c4a1f53";
    const store = await createStore();
    // The connect modal saves a new Core with no credentials; login fills them in
    // later.
    store.dispatch(
      Session.Core.set({
        key,
        name: "Staging",
        host: "somewhere.com",
        port: 9090,
        secure: false,
        username: "",
        password: "",
      }),
    );
    await waitForPersisted(db, (p) => p.core?.cores[key] != null, "Core not persisted");
    const reloaded = await createStore();
    expect(Session.Core.selectState(reloaded.getState(), key)?.name).toBe("Staging");
  });

  it("should carry the previous release's Cores out of the browser legacy store", async () => {
    const legacyKey = "5b8e2f1a-9c47-4d03-b6a2-1f0d9e8c7b64";
    // The literal keys pin the 0.56 on-disk format the reader must keep accepting.
    localStorage.setItem(
      "persisted-state.json:console-version",
      JSON.stringify({ version: 3 }),
    );
    localStorage.setItem(
      "persisted-state.json:console-persisted-state.3",
      JSON.stringify({
        cluster: {
          activeCluster: legacyKey,
          clusters: {
            [legacyKey]: {
              key: legacyKey,
              name: "Legacy",
              host: "legacy.example.com",
              port: 9090,
              username: "synnax",
              password: "seldon",
              secure: false,
            },
          },
        },
      }),
    );
    try {
      const store = await createStore();
      expect(Session.Core.selectState(store.getState(), legacyKey)?.name).toBe(
        "Legacy",
      );
      expect(Session.Core.selectSelectedKey(store.getState())).toBe(legacyKey);
    } finally {
      localStorage.removeItem("persisted-state.json:console-version");
      localStorage.removeItem("persisted-state.json:console-persisted-state.3");
    }
  });

  it("boots when storage is unreadable and announces it", async () => {
    const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const fail = async (): Promise<never> => {
      throw new Error("storage is blocked");
    };
    const store = await createStore({
      openKV: () => ({
        get: fail,
        set: fail,
        setMany: fail,
        delete: fail,
        deleteMany: fail,
        length: fail,
        keys: fail,
        clear: fail,
      }),
    });
    // The announcement is the one departure a blocked platform leaves from a
    // fresh session.
    const { [Session.Persist.SLICE_NAME]: _persist, ...rest } = withoutDrift(
      store.getState(),
    );
    const { [Session.Persist.SLICE_NAME]: _, ...zero } = withoutDrift(
      Session.ZERO_STATE,
    );
    expect(rest).toStrictEqual(zero);
    expect(Session.Persist.selectStoreUnavailable(store.getState())).toBe(true);
    errorSpy.mockRestore();
  });

  /** Walks into a Core and project the way production does, and edits window state. */
  const enterAndEdit = async (
    store: Session.Store,
    core: string,
    clusterKey: string,
  ): Promise<void> => {
    store.dispatch(Session.Core.select(core));
    store.dispatch(Session.Core.setClusterKey({ key: core, clusterKey }));
    store.dispatch(Session.Project.select("6f7cd5f4-4b93-4a35-a55c-72ba9dae2c9d"));
    await waitForPersisted(db, (p) => p.project != null, "Core swap has not settled");
    store.dispatch(Session.Nav.showBottom({ windowKey: MAIN_WINDOW }));
    await waitForPersisted(db, (p) => p.nav != null, "project state not persisted");
  };

  const clusterKeys = async (clusterKey: string): Promise<string[]> =>
    (await db.keys()).filter((key) => key.includes(clusterKey));

  it("stores a window's view of the project under the window", async () => {
    const store = await createStore();
    await enterAndEdit(store, Session.Core.DEMO_KEY, CLUSTER_KEY);
    const keys = await db.keys();
    const prefix = `window.${CLUSTER_KEY}.`;
    const windowed = keys.filter((key) => key.startsWith(prefix));
    expect(windowed.length).toBeGreaterThan(0);
    expect(windowed.every((key) => key.includes(MAIN_WINDOW))).toBe(true);
    // The project partition holds what the project owns, not how a window looks at it.
    const project = readPersisted(db).filter((p) => p.range != null);
    expect(project.length).toBeGreaterThan(0);
    project.forEach((p) => expect(p.nav).toBeUndefined());
  });

  it("splits the panel strip's order from each window's view of it", async () => {
    const store = await createStore();
    await enterAndEdit(store, Session.Core.DEMO_KEY, CLUSTER_KEY);
    store.dispatch(Session.Panel.reconcileOrder({ panels: [PANEL] }));
    store.dispatch(Session.Panel.select({ key: PANEL.key, windowKey: MAIN_WINDOW }));
    await waitForPersisted(
      db,
      (p) => (p.panels?.order?.length ?? 0) > 0,
      "order not persisted yet",
    );
    await waitForPersisted(
      db,
      (p) => Object.values(p.panels?.windows ?? {})[0]?.selected === PANEL.key,
      "panel selection not persisted yet",
    );
    // The order belongs to the project; the selection belongs to the window looking
    // at it. The shared remainder carries no window's state, and a window's
    // partition carries no shared fields.
    readPersisted(db).forEach((p) => {
      if (p.panels == null) return;
      if (p.panels.order != null) expect(p.panels.windows).toEqual({});
      if (Object.keys(p.panels.windows ?? {}).length > 0)
        expect(p.panels.order).toBeUndefined();
    });
  });

  it("keeps the main window's partition when the main window closes", async () => {
    const store = await createStore();
    await enterAndEdit(store, Session.Core.DEMO_KEY, CLUSTER_KEY);
    const mainKeys = async (): Promise<string[]> =>
      (await db.keys()).filter(
        (key) => key.startsWith(`window.${CLUSTER_KEY}.`) && key.includes(MAIN_WINDOW),
      );
    expect((await mainKeys()).length).toBeGreaterThan(0);
    store.dispatch(Drift.closeWindow({ key: MAIN_WINDOW }));
    // A later write must not prune the partition main will reuse on relaunch.
    store.dispatch(Session.Range.select(Session.Range.RECENT_KEY));
    await waitForPersisted(
      db,
      (p) => p.range?.selected === Session.Range.RECENT_KEY,
      "range selection not persisted yet",
    );
    expect((await mainKeys()).length).toBeGreaterThan(0);
  });

  // Guards every window-scoped slice at once: a slice whose narrowed bytes fail its
  // own schema, or fail to round-trip, shows up here as a discard or a rewrite.
  it("leaves the partition of an untouched window alone across launches", async () => {
    const store = await createStore();
    // Enter a context without touching any window-keyed slice, so main's partition
    // holds every slice narrowed to a window none of them has an entry for.
    store.dispatch(Session.Core.select(Session.Core.DEMO_KEY));
    store.dispatch(
      Session.Core.setClusterKey({
        key: Session.Core.DEMO_KEY,
        clusterKey: CLUSTER_KEY,
      }),
    );
    store.dispatch(Session.Project.select("6f7cd5f4-4b93-4a35-a55c-72ba9dae2c9d"));
    await waitForPersisted(db, (p) => p.project != null, "Core swap has not settled");
    store.dispatch(Session.Range.select(Session.Range.RECENT_KEY));
    await waitForPersisted(
      db,
      (p) => p.range?.selected === Session.Range.RECENT_KEY,
      "range selection not persisted yet",
    );
    const slotKey = (await db.keys()).find(
      (key) => key.startsWith(`window.${CLUSTER_KEY}.`) && key.endsWith(".slot"),
    );
    expect(slotKey).toBeDefined();
    const slotBefore = await db.get(slotKey as string);
    const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const reloaded = await createStore();
    reloaded.dispatch(Session.Panel.reconcileOrder({ panels: [PANEL] }));
    await waitForPersisted(
      db,
      (p) => (p.panels?.order?.length ?? 0) > 0,
      "order not persisted yet",
    );
    // A rewrite advances the slot pointer, so four launches would erase the ring.
    expect(await db.get(slotKey as string)).toEqual(slotBefore);
    expect(errorSpy).not.toHaveBeenCalledWith(
      expect.stringContaining("discarding stored slice"),
      expect.anything(),
    );
    errorSpy.mockRestore();
  });

  it("purges a removed Core's stored state", async () => {
    const store = await createStore();
    await enterAndEdit(store, Session.Core.DEMO_KEY, CLUSTER_KEY);
    expect((await clusterKeys(CLUSTER_KEY)).length).toBeGreaterThan(0);
    store.dispatch(Session.Core.remove(Session.Core.DEMO_KEY));
    await waitFor(async () => expect(await clusterKeys(CLUSTER_KEY)).toHaveLength(0));
  });

  /** The partition keys holding the given slice. */
  const holdersOf = (slice: keyof Session.State): string[] =>
    [...db.store.entries()]
      .filter(([key]) => !key.endsWith(".slot"))
      .filter(([, value]) => (value as Partial<Session.State>)[slice] != null)
      .map(([key]) => key);

  // Statuses and ranges belong to the cluster, so a favorite made in one project is
  // still a favorite in the next.
  it("stores status favorites under the Core, not the project", async () => {
    const store = await createStore();
    await enterAndEdit(store, Session.Core.DEMO_KEY, CLUSTER_KEY);
    store.dispatch(Session.Status.toggleFavorite("status-1"));
    await waitForPersisted(db, (p) => p.status != null, "favorite not persisted yet");
    const holders = holdersOf("status");
    expect(holders.length).toBeGreaterThan(0);
    expect(holders.every((key) => key.startsWith(`core.${CLUSTER_KEY}`))).toBe(true);
  });

  it("stores ranges under the Core, not the project", async () => {
    const store = await createStore();
    await enterAndEdit(store, Session.Core.DEMO_KEY, CLUSTER_KEY);
    store.dispatch(Session.Range.select(Session.Range.RECENT_KEY));
    await waitForPersisted(db, (p) => p.range != null, "range not persisted yet");
    const holders = holdersOf("range");
    expect(holders.length).toBeGreaterThan(0);
    expect(holders.every((key) => key.startsWith(`core.${CLUSTER_KEY}`))).toBe(true);
  });

  it("keeps a repointed Core's stored state", async () => {
    const store = await createStore();
    await enterAndEdit(store, Session.Core.DEMO_KEY, CLUSTER_KEY);
    const before = await clusterKeys(CLUSTER_KEY);
    expect(before.length).toBeGreaterThan(0);
    const demo = Session.Core.selectState(store.getState(), Session.Core.DEMO_KEY);
    // The new address may still reach the same cluster, and only a connection can
    // answer that. Dropping the state here loses a workspace to a retyped hostname.
    store.dispatch(Session.Core.set({ ...demo!, host: "elsewhere.com" }));
    await waitForPersisted(
      db,
      (p) => p.core?.cores[Session.Core.DEMO_KEY]?.host === "elsewhere.com",
      "the repoint has not been persisted yet",
    );
    expect(await clusterKeys(CLUSTER_KEY)).toEqual(before);
  });

  it("keeps a cluster's state while another Core still names it", async () => {
    const store = await createStore();
    store.dispatch(
      Session.Core.setClusterKey({
        key: Session.Core.LOCAL_KEY,
        clusterKey: CLUSTER_KEY,
      }),
    );
    await enterAndEdit(store, Session.Core.DEMO_KEY, CLUSTER_KEY);
    expect((await clusterKeys(CLUSTER_KEY)).length).toBeGreaterThan(0);
    store.dispatch(Session.Core.remove(Session.Core.DEMO_KEY));
    // The local Core reaches the same cluster, so its state outlives the removal.
    await waitFor(async () =>
      expect((await clusterKeys(CLUSTER_KEY)).length).toBeGreaterThan(0),
    );
  });

  it("never writes transient haul state", async () => {
    const projectKey = "6f7cd5f4-4b93-4a35-a55c-72ba9dae2c9d";
    const store = await createStore();
    store.dispatch(Session.Core.select(Session.Core.DEMO_KEY));
    store.dispatch(
      Session.Core.setClusterKey({
        key: Session.Core.DEMO_KEY,
        clusterKey: CLUSTER_KEY,
      }),
    );
    store.dispatch(Session.Project.select(projectKey));
    // The switch hydrates the project's stored slices over whatever is in the store, so
    // let it settle before making changes that have to survive. Project-scoped slices
    // are only written once the hydrate lands, so nav appearing marks the end of it.
    await waitForPersisted(
      db,
      (p) => p.nav != null,
      "project swap has not settled yet",
    );
    store.dispatch(Session.Haul.setHauled(HAULED));
    store.dispatch(Session.Nav.showBottom({ windowKey: MAIN_WINDOW }));
    await waitForPersisted(
      db,
      (p) => Object.values(p.nav?.windows ?? {})[0]?.bottom.visible === true,
      "nav change not persisted yet",
    );
    expect(store.getState().haul.state).toStrictEqual(HAULED);
    // Haul belongs to no partition scope, so the drag never reached any payload.
    readPersisted(db).forEach((p) => expect(p.haul).toBeUndefined());
  });
});

// A pre-render webview's label never resolves a window key, so window-scoped dispatches
// from it must die locally instead of mutating state or leaking to other windows.
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
        [Session.Core.SLICE_NAME]: {
          ...Session.Core.ZERO_SLICE_STATE,
          selected: Session.Core.DEMO_KEY,
        },
      }),
    );
    expect(Session.Core.selectSelectedKey(next)).toBe(Session.Core.DEMO_KEY);
  });
});
