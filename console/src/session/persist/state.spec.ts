// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type MiddlewareAPI } from "@reduxjs/toolkit";
import { kv, TimeSpan } from "@synnaxlabs/x";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted((): { engine: "web" | "tauri"; label: string } => ({
  engine: "web",
  label: "main",
}));

vi.mock("@/session/runtime/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

vi.mock("@tauri-apps/api/window", () => ({
  getCurrentWindow: () => ({ label: mocks.label }),
}));

import { Persist } from "@/session/persist";

interface MockState {
  cluster: { selected?: string };
  project: { selected?: string };
  work: { value: string; transient?: string };
}

const ZERO_MOCK_STATE: MockState = {
  cluster: {},
  project: {},
  work: { value: "0.0.0", transient: "zero" },
};

const SCOPES: Persist.Scopes<MockState> = {
  global: ["cluster"],
  cluster: ["project"],
  project: ["work"],
};

const getContext = (state: MockState): Persist.Context => ({
  cluster: state.cluster.selected,
  project: state.project.selected,
});

const CTX: Persist.Context = { cluster: "c1", project: "p1" };

const STATE: MockState = {
  cluster: { selected: "c1" },
  project: { selected: "p1" },
  work: { value: "16.2.0", transient: "drag" },
};

// One state key plus one version key for the global partition.
const GLOBAL_KEYS = 2;

const openPersist = async (
  store: kv.MockAsync,
  overrides: Partial<Persist.Config<MockState>> = {},
) =>
  await Persist.open<MockState>({
    initial: ZERO_MOCK_STATE,
    scopes: SCOPES,
    getContext,
    openKV: () => store,
    debounceInterval: TimeSpan.ZERO,
    ...overrides,
  });

/**
 * Drives actions through the persistence middleware the way the redux store does,
 * standing in for the root reducer: the caller supplies the state each action reduced
 * to, and a hydrate the middleware dispatches swaps its slices in and rides back
 * through the chain.
 */
const createDriver = async (
  store: kv.MockAsync,
  overrides: Partial<Persist.Config<MockState>> = {},
) => {
  const { initialState, middleware } = await openPersist(store, overrides);
  let state = initialState ?? ZERO_MOCK_STATE;
  let settled = 0;
  const next = vi.fn((action: unknown) => action);
  const dispatched = vi.fn((action: unknown) => {
    if (!Persist.hydrate.match(action)) return;
    state = { ...state, ...(action.payload as Partial<MockState>) };
    run(action);
  });
  const api = { getState: () => state, dispatch: dispatched } as MiddlewareAPI;
  const run = middleware(api)(next);
  return {
    initialState,
    next,
    dispatched,
    getState: () => state,
    dispatch: (action: { type: string }, reduced?: MockState) => {
      if (reduced != null) state = reduced;
      return run(action);
    },
    /** Waits for the swap in flight to hydrate. */
    settle: async () => {
      await vi.waitFor(() =>
        expect(dispatched.mock.calls.length).toBeGreaterThan(settled),
      );
      settled = dispatched.mock.calls.length;
    },
    /** Reopens the store to see what production has written so far. */
    composed: async () => (await openPersist(store)).initialState,
    /** Waits until a reopened engine composes state the predicate accepts. */
    flushed: async (predicate: (state?: MockState) => boolean) =>
      await vi.waitFor(async () => {
        const { initialState } = await openPersist(store);
        if (!predicate(initialState)) throw new Error("state not persisted yet");
      }),
  };
};

const workIs =
  (value: string) =>
  (state?: MockState): boolean =>
    state?.work.value === value;

type Driver = Awaited<ReturnType<typeof createDriver>>;

/**
 * Walks into a context the way production does: the cluster is chosen first, and only
 * once its partition has hydrated does the project it names become selectable.
 */
const enter = async (driver: Driver, { cluster, project }: Persist.Context) => {
  if (driver.getState().cluster.selected !== cluster) {
    driver.dispatch(
      { type: "cluster/select" },
      { ...driver.getState(), cluster: { selected: cluster } },
    );
    await driver.settle();
  }
  // The cluster's partition may already name the project, in which case nothing moved.
  if (project == null || driver.getState().project.selected === project) return;
  driver.dispatch(
    { type: "project/select" },
    { ...driver.getState(), project: { selected: project } },
  );
  await driver.settle();
};

/** Edits the work slice and waits for the edit to reach disk. */
const edit = async (driver: Driver, value: string) => {
  driver.dispatch(
    { type: "work/edit" },
    { ...driver.getState(), work: { value, transient: "drag" } },
  );
  await driver.flushed(workIs(value));
};

describe("Persist.open", () => {
  beforeEach(() => {
    mocks.engine = "web";
    mocks.label = "main";
  });

  describe("composition", () => {
    it("should start from the initial state when nothing has been persisted", async () => {
      const { initialState } = await openPersist(new kv.MockAsync());
      expect(initialState).toEqual(ZERO_MOCK_STATE);
    });

    it("should compose the global, selected cluster, and active project partitions", async () => {
      const store = new kv.MockAsync();
      const driver = await createDriver(store);
      await enter(driver, CTX);
      await edit(driver, "16.2.0");
      expect(await driver.composed()).toEqual(STATE);
    });

    it("should stop at the global partition when no cluster was selected", async () => {
      const store = new kv.MockAsync();
      const driver = await createDriver(store);
      driver.dispatch({ type: "work/edit" }, { ...ZERO_MOCK_STATE, work: STATE.work });
      await vi.waitFor(async () => expect(await store.length()).toBe(GLOBAL_KEYS));
      const composed = await driver.composed();
      expect(composed?.project).toEqual(ZERO_MOCK_STATE.project);
      expect(composed?.work).toEqual(ZERO_MOCK_STATE.work);
    });
  });

  describe("partitions", () => {
    it("should write only the global partition when no cluster is in context", async () => {
      const store = new kv.MockAsync();
      const driver = await createDriver(store);
      driver.dispatch({ type: "work/edit" }, { ...ZERO_MOCK_STATE, work: STATE.work });
      await vi.waitFor(async () => expect(await store.length()).toBe(GLOBAL_KEYS));
    });

    it("should skip the project partition when no project is in context", async () => {
      const store = new kv.MockAsync();
      const driver = await createDriver(store);
      await enter(driver, { cluster: "c1" });
      driver.dispatch(
        { type: "work/edit" },
        { ...driver.getState(), work: { value: "c1-work" } },
      );
      await driver.flushed((s) => s?.cluster.selected === "c1");
      // The project-scoped work never reached disk, so it composes from zero.
      expect((await driver.composed())?.work).toEqual(ZERO_MOCK_STATE.work);
    });

    it("should bound each partition to four versions", async () => {
      const store = new kv.MockAsync();
      const driver = await createDriver(store);
      await enter(driver, CTX);
      for (let i = 0; i < 10; i++) await edit(driver, `16.2.${i}`);
      // Four ring entries plus a version pointer for each of the three partitions.
      expect(await store.length()).toBe(15);
    });

    it("should scope slices to the context they were written under", async () => {
      const store = new kv.MockAsync();
      const driver = await createDriver(store);
      await enter(driver, CTX);
      await edit(driver, "p1-work");
      await enter(driver, { cluster: "c2", project: "p2" });
      await edit(driver, "p2-work");
      await enter(driver, CTX);
      expect(driver.getState().work.value).toEqual("p1-work");
    });
  });

  describe("secondary windows", () => {
    it("should neither compose nor persist state outside the main window", async () => {
      mocks.engine = "tauri";
      mocks.label = "child-window";
      const store = new kv.MockAsync();
      const driver = await createDriver(store);
      expect(driver.initialState).toBeUndefined();
      const action = { type: "work/edit" };
      expect(driver.dispatch(action, STATE)).toBe(action);
      expect(driver.next).toHaveBeenCalledWith(action);
      await expect(store.length()).resolves.toBe(0);
    });
  });

  describe("exclude", () => {
    it("should strip excluded state from writes", async () => {
      const store = new kv.MockAsync();
      const stripTransient = (s: MockState): MockState => ({
        ...s,
        work: { value: s.work.value },
      });
      const driver = await createDriver(store, { exclude: [stripTransient] });
      await enter(driver, CTX);
      await edit(driver, "16.2.0");
      expect((await driver.composed())?.work).toEqual({ value: "16.2.0" });
    });
  });

  describe("migrators", () => {
    const createPersisted = async (store: kv.MockAsync) => {
      const driver = await createDriver(store);
      await enter(driver, CTX);
      await edit(driver, "16.2.0");
    };

    it("should apply a slice migrator as its partition loads", async () => {
      const store = new kv.MockAsync();
      await createPersisted(store);
      const { initialState } = await openPersist(store, {
        migrators: {
          work: (raw) => ({ ...(raw as MockState["work"]), value: "migrated" }),
        },
      });
      expect(initialState?.work.value).toEqual("migrated");
    });

    it("should fall back to the slice's initial state when its migrator throws", async () => {
      const store = new kv.MockAsync();
      await createPersisted(store);
      const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
      const { initialState } = await openPersist(store, {
        migrators: {
          work: () => {
            throw new Error("migration failed");
          },
        },
      });
      expect(initialState?.work).toEqual(ZERO_MOCK_STATE.work);
      expect(initialState?.cluster).toEqual(STATE.cluster);
      expect(errorSpy).toHaveBeenCalled();
      errorSpy.mockRestore();
    });
  });
});

describe("Persist.middleware", () => {
  beforeEach(() => {
    mocks.engine = "web";
    mocks.label = "main";
  });

  it("should pass the action through to next and return its result", async () => {
    const driver = await createDriver(new kv.MockAsync());
    const action = { type: "any/action" };
    expect(driver.dispatch(action)).toBe(action);
    expect(driver.next).toHaveBeenCalledWith(action);
  });

  it("should persist the current store state under the current context", async () => {
    const store = new kv.MockAsync();
    const driver = await createDriver(store);
    await enter(driver, CTX);
    await edit(driver, "edited");
  });

  // The revert and clear branches reload the window, which jsdom cannot perform; the
  // production code swallows that failure via .catch, so we suppress the expected log.
  it("should revert every active partition one version on a revertState action", async () => {
    const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const store = new kv.MockAsync();
    const driver = await createDriver(store);
    await enter(driver, CTX);
    await edit(driver, "16.2.0");
    await edit(driver, "16.2.1");
    driver.dispatch(Persist.revertState());
    await driver.flushed(workIs("16.2.0"));
    errorSpy.mockRestore();
  });

  it("should fall back to the initial state when reverting past the first version", async () => {
    const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const store = new kv.MockAsync();
    const driver = await createDriver(store);
    await enter(driver, CTX);
    await edit(driver, "16.2.0");
    driver.dispatch(Persist.revertState());
    await driver.flushed(workIs(ZERO_MOCK_STATE.work.value));
    errorSpy.mockRestore();
  });

  it("should clear the entire store on a clearState action", async () => {
    const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const store = new kv.MockAsync();
    const driver = await createDriver(store);
    await enter(driver, CTX);
    await edit(driver, "16.2.0");
    driver.dispatch(Persist.clearState());
    await vi.waitFor(async () => expect(await store.length()).toBe(0));
    errorSpy.mockRestore();
  });

  it("should load the target context and dispatch hydrate on a context switch", async () => {
    const driver = await createDriver(new kv.MockAsync());
    await enter(driver, CTX);
    await edit(driver, "c1-work");
    await enter(driver, { cluster: "c2", project: "p2" });
    await enter(driver, { cluster: "c1" });
    // The cluster's partition names the project whose partition holds the work.
    expect(driver.dispatched).toHaveBeenLastCalledWith(
      Persist.hydrate({
        project: { selected: "p1" },
        work: { value: "c1-work", transient: "drag" },
      }),
    );
  });

  it("should swap only the project partition when the cluster is unchanged", async () => {
    const driver = await createDriver(new kv.MockAsync());
    await enter(driver, CTX);
    await edit(driver, "p1-work");
    await enter(driver, { cluster: "c1", project: "p2" });
    await enter(driver, CTX);
    // The cluster-scoped project slice stayed put; only the project's work swapped.
    expect(driver.dispatched).toHaveBeenLastCalledWith(
      Persist.hydrate({ work: { value: "p1-work", transient: "drag" } }),
    );
  });

  it("should hydrate zero slices for a never-visited context", async () => {
    const driver = await createDriver(new kv.MockAsync());
    await enter(driver, { cluster: "fresh" });
    expect(driver.dispatched).toHaveBeenCalledWith(
      Persist.hydrate({
        project: ZERO_MOCK_STATE.project,
        work: ZERO_MOCK_STATE.work,
      }),
    );
  });

  it("should adopt the hydrated context instead of re-swapping on hydrate", async () => {
    const store = new kv.MockAsync();
    const driver = await createDriver(store);
    await enter(driver, CTX);
    const swaps = driver.dispatched.mock.calls.length;
    driver.dispatch(Persist.hydrate({ work: STATE.work }), STATE);
    await driver.flushed(workIs("16.2.0"));
    expect(driver.dispatched.mock.calls.length).toBe(swaps);
  });

  it("should discard a stale swap when a newer context switch supersedes it", async () => {
    const store = new kv.MockAsync();
    const seed = await createDriver(store);
    await enter(seed, CTX);
    await edit(seed, "fresh");
    let releaseStale: (() => void) | undefined;
    const staleGate = new Promise<void>((resolve) => (releaseStale = resolve));
    // Holds the cluster partition read so its swap is still in flight when the project
    // swap starts and finishes.
    let gateArmed = false;
    const gated: Persist.SugaredKV = {
      get: async <V>(key: string): Promise<V | null> => {
        if (gateArmed && key.startsWith("cluster.c1")) await staleGate;
        return await store.get<V>(key);
      },
      set: async (key, value) => await store.set(key, value),
      delete: async (key) => await store.delete(key),
      length: async () => await store.length(),
      clear: async () => await store.clear(),
    };
    const driver = await createDriver(store, { openKV: () => gated });
    gateArmed = true;
    driver.dispatch(
      { type: "cluster/select" },
      { ...ZERO_MOCK_STATE, cluster: { selected: "c1" } },
    );
    driver.dispatch({ type: "project/select" }, STATE);
    await driver.settle();
    expect(driver.getState().work.value).toEqual("fresh");
    releaseStale?.();
    // The slow cluster swap resolved last; hydrating it would wipe the newer project
    // selection with its stale slices.
    await expect.poll(() => driver.dispatched.mock.calls.length).toBe(1);
  });

  it("should coalesce rapid dispatches into a single persist when debounced", async () => {
    const store = new kv.MockAsync();
    const driver = await createDriver(store, {
      debounceInterval: TimeSpan.milliseconds(250),
    });
    vi.useFakeTimers();
    try {
      driver.dispatch({ type: "a" }, ZERO_MOCK_STATE);
      driver.dispatch({ type: "b" }, ZERO_MOCK_STATE);
      driver.dispatch({ type: "c" }, ZERO_MOCK_STATE);
      await expect(store.length()).resolves.toBe(0);
      await vi.advanceTimersByTimeAsync(250);
    } finally {
      vi.useRealTimers();
    }
    await vi.waitFor(async () => expect(await store.length()).toBe(GLOBAL_KEYS));
  });
});

describe("Persist.hardClearAndReload", () => {
  const PREFIX = `${Persist.STORE_PATH}:`;
  let errorSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    localStorage.clear();
    // The reload after clearing is a no-op under jsdom and its failure is swallowed by
    // the production .catch; suppress the expected error log.
    errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
  });

  afterEach(() => {
    errorSpy.mockRestore();
    localStorage.clear();
  });

  it("should clear the persisted store scoped to the store path", async () => {
    await Persist.openSugaredKV(Persist.STORE_PATH).set("global.version", {
      version: 1,
    });
    localStorage.setItem("unrelated:key", "keep-me");
    Persist.hardClearAndReload();
    await vi.waitFor(() => {
      expect(localStorage.getItem(`${PREFIX}global.version`)).toBeNull();
    });
    expect(localStorage.getItem("unrelated:key")).toBe("keep-me");
  });
});
