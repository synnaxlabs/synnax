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

const LEVELS: Persist.Levels<MockState> = {
  l0: ["cluster"],
  l1: ["project"],
  l2: ["work"],
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

const openEngine = async (
  store: kv.MockAsync,
  overrides: Partial<Persist.Config<MockState>> = {},
) =>
  await Persist.open<MockState>({
    initial: ZERO_MOCK_STATE,
    levels: LEVELS,
    getContext,
    openKV: () => store,
    ...overrides,
  });

describe("Persist", () => {
  describe("engine.persist", () => {
    it("should write each partition under its own versioned keys", async () => {
      const store = new kv.MockAsync();
      const engine = await openEngine(store);
      await engine.persist(STATE, CTX);
      await expect(
        store.get(Persist.partitionStateKey(Persist.l0PartitionBase(), 1)),
      ).resolves.toEqual({ cluster: { selected: "c1" } });
      await expect(
        store.get(Persist.partitionStateKey(Persist.l1PartitionBase("c1"), 1)),
      ).resolves.toEqual({ project: { selected: "p1" } });
      await expect(
        store.get(Persist.partitionStateKey(Persist.l2PartitionBase("c1", "p1"), 1)),
      ).resolves.toEqual({ work: STATE.work });
      await expect(
        store.get(Persist.partitionVersionKey(Persist.l0PartitionBase())),
      ).resolves.toEqual({ version: 1 });
    });

    it("should skip L1 and L2 when no cluster is in context", async () => {
      const store = new kv.MockAsync();
      const engine = await openEngine(store);
      await engine.persist(STATE, {});
      await expect(
        store.get(Persist.partitionStateKey(Persist.l0PartitionBase(), 1)),
      ).resolves.toEqual({ cluster: { selected: "c1" } });
      await expect(
        store.get(Persist.partitionStateKey(Persist.l1PartitionBase("c1"), 1)),
      ).resolves.toBeNull();
    });

    it("should skip L2 when no project is in context", async () => {
      const store = new kv.MockAsync();
      const engine = await openEngine(store);
      await engine.persist(STATE, { cluster: "c1" });
      await expect(
        store.get(Persist.partitionStateKey(Persist.l1PartitionBase("c1"), 1)),
      ).resolves.toEqual({ project: { selected: "p1" } });
      await expect(
        store.get(Persist.partitionStateKey(Persist.l2PartitionBase("c1", "p1"), 1)),
      ).resolves.toBeNull();
    });

    it("should maintain a bounded history ring per partition", async () => {
      const store = new kv.MockAsync();
      const openKV = () => store;
      const engine = await openEngine(store, { openKV });
      for (let i = 0; i < 10; i++)
        await engine.persist({ ...STATE, work: { value: `16.2.${i}` } }, CTX);
      // 4 ring entries + 1 version pointer for each of the three partitions.
      await expect(store.length()).resolves.toBe(15);
      const engine2 = await openEngine(store, { openKV });
      expect(engine2.initialState?.work.value).toEqual("16.2.9");
    });
  });

  describe("engine.revert", () => {
    it("should step every active partition back one version", async () => {
      const store = new kv.MockAsync();
      const engine = await openEngine(store);
      await engine.persist(STATE, CTX);
      await engine.persist({ ...STATE, work: { value: "16.2.1" } }, CTX);
      await engine.revert(CTX);
      const engine2 = await openEngine(store);
      expect(engine2.initialState?.work.value).toEqual("16.2.0");
    });

    it("should fall back to the initial state when reverting past the first version", async () => {
      const store = new kv.MockAsync();
      const engine = await openEngine(store);
      await engine.persist(STATE, CTX);
      await engine.revert(CTX);
      const engine2 = await openEngine(store);
      expect(engine2.initialState?.work.value).toEqual(ZERO_MOCK_STATE.work.value);
    });
  });

  describe("engine.clear", () => {
    it("should clear the entire store", async () => {
      const store = new kv.MockAsync();
      const engine = await openEngine(store);
      await engine.persist(STATE, CTX);
      await engine.clear();
      await expect(store.length()).resolves.toBe(0);
      const engine2 = await openEngine(store);
      expect(engine2.initialState).toEqual(ZERO_MOCK_STATE);
    });
  });

  describe("startup composition", () => {
    it("should compose L0, the selected cluster's L1, and its project's L2", async () => {
      const store = new kv.MockAsync();
      const engine = await openEngine(store);
      await engine.persist(STATE, CTX);
      const engine2 = await openEngine(store);
      expect(engine2.initialState).toEqual({
        ...STATE,
        work: { ...STATE.work, transient: STATE.work.transient },
      });
      expect(engine2.context).toEqual(CTX);
    });

    it("should stop at L0 when no cluster was selected", async () => {
      const store = new kv.MockAsync();
      const engine = await openEngine(store);
      await engine.persist({ ...STATE, cluster: {} }, { cluster: "c1", project: "p1" });
      const engine2 = await openEngine(store);
      expect(engine2.initialState?.project).toEqual(ZERO_MOCK_STATE.project);
      expect(engine2.initialState?.work).toEqual(ZERO_MOCK_STATE.work);
    });
  });

  describe("engine.loadSwap", () => {
    it("should load the target cluster's L1 and the L2 its project points at", async () => {
      const store = new kv.MockAsync();
      const engine = await openEngine(store);
      await engine.persist(
        {
          cluster: { selected: "c2" },
          project: { selected: "p2" },
          work: { value: "c2-work" },
        },
        { cluster: "c2", project: "p2" },
      );
      const loaded = await engine.loadSwap(STATE, { cluster: "c2" }, true);
      expect(loaded.project?.selected).toEqual("p2");
      expect(loaded.work?.value).toEqual("c2-work");
    });

    it("should return zero slices for a never-visited context", async () => {
      const store = new kv.MockAsync();
      const engine = await openEngine(store);
      const loaded = await engine.loadSwap(STATE, { cluster: "fresh" }, true);
      expect(loaded.project).toEqual(ZERO_MOCK_STATE.project);
      expect(loaded.work).toEqual(ZERO_MOCK_STATE.work);
    });

    it("should swap only L2 when the cluster is unchanged", async () => {
      const store = new kv.MockAsync();
      const engine = await openEngine(store);
      await engine.persist(
        { ...STATE, project: { selected: "p2" }, work: { value: "p2-work" } },
        { cluster: "c1", project: "p2" },
      );
      const loaded = await engine.loadSwap(
        STATE,
        { cluster: "c1", project: "p2" },
        false,
      );
      expect(loaded.project).toBeUndefined();
      expect(loaded.work?.value).toEqual("p2-work");
    });
  });

  describe("exclude", () => {
    it("should strip excluded deep keys from writes and restore their defaults on load", async () => {
      const store = new kv.MockAsync();
      const openKV = () => store;
      const exclude: Persist.Config<MockState>["exclude"] = ["work.transient"];
      const engine = await openEngine(store, { openKV, exclude });
      await engine.persist(STATE, CTX);
      await expect(
        store.get(Persist.partitionStateKey(Persist.l2PartitionBase("c1", "p1"), 1)),
      ).resolves.toEqual({ work: { value: "16.2.0" } });
      const engine2 = await openEngine(store, { openKV, exclude });
      expect(engine2.initialState?.work).toEqual({
        value: "16.2.0",
        transient: "zero",
      });
    });

    it("should apply a function exclude to transform the persisted state", async () => {
      const store = new kv.MockAsync();
      const stripWork = (s: MockState): MockState => ({
        ...s,
        work: { value: "" },
      });
      const engine = await openEngine(store, { exclude: [stripWork] });
      await engine.persist(STATE, CTX);
      await expect(
        store.get(Persist.partitionStateKey(Persist.l2PartitionBase("c1", "p1"), 1)),
      ).resolves.toEqual({ work: { value: "" } });
    });
  });

  describe("migrators", () => {
    it("should apply a slice migrator as its partition loads", async () => {
      const store = new kv.MockAsync();
      const engine = await openEngine(store);
      await engine.persist(STATE, CTX);
      const engine2 = await openEngine(store, {
        migrators: {
          work: (raw) => ({ ...(raw as MockState["work"]), value: "migrated" }),
        },
      });
      expect(engine2.initialState?.work.value).toEqual("migrated");
    });

    it("should fall back to the slice's initial state when its migrator throws", async () => {
      const store = new kv.MockAsync();
      const engine = await openEngine(store);
      await engine.persist(STATE, CTX);
      const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
      const engine2 = await openEngine(store, {
        migrators: {
          work: () => {
            throw new Error("migration failed");
          },
        },
      });
      expect(engine2.initialState?.work).toEqual(ZERO_MOCK_STATE.work);
      expect(engine2.initialState?.cluster).toEqual(STATE.cluster);
      expect(errorSpy).toHaveBeenCalled();
      errorSpy.mockRestore();
    });
  });
});

describe("Persist.middleware", () => {
  const createEngine = (): Persist.Engine<MockState> => ({
    context: {},
    initialState: undefined,
    persist: vi.fn().mockResolvedValue(undefined),
    loadSwap: vi.fn().mockResolvedValue({ work: { value: "swapped" } }),
    revert: vi.fn().mockResolvedValue(undefined),
    clear: vi.fn().mockResolvedValue(undefined),
  });

  // Drives an action straight through the middleware chain, capturing what next saw.
  const drive = (
    engine: Persist.Engine<MockState>,
    state: MockState,
    action: { type: string },
    debounceInterval = TimeSpan.ZERO,
  ) => {
    const next = vi.fn((a: unknown) => a);
    const dispatch = vi.fn();
    const store: MiddlewareAPI = { getState: () => state, dispatch };
    const result = Persist.middleware<MockState>({
      engine,
      getContext,
      debounceInterval,
    })(store)(next)(action);
    return { next, dispatch, result };
  };

  it("should pass the action through to next and return its result", () => {
    const engine = createEngine();
    const action = { type: "any/action" };
    const { next, result } = drive(engine, ZERO_MOCK_STATE, action);
    expect(next).toHaveBeenCalledWith(action);
    expect(result).toBe(action);
  });

  it("should persist the current store state under the current context", () => {
    const engine = createEngine();
    drive(engine, ZERO_MOCK_STATE, { type: "any/action" });
    expect(engine.persist).toHaveBeenCalledWith(ZERO_MOCK_STATE, {});
    expect(engine.revert).not.toHaveBeenCalled();
    expect(engine.clear).not.toHaveBeenCalled();
  });

  // The revert/clear branches trigger a window reload, which jsdom cannot perform; the
  // production code swallows that failure via .catch, so we suppress the expected log.
  it("should revert instead of persisting on a revertState action", () => {
    const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const engine = createEngine();
    drive(engine, ZERO_MOCK_STATE, Persist.revertState());
    expect(engine.revert).toHaveBeenCalledOnce();
    expect(engine.persist).not.toHaveBeenCalled();
    errorSpy.mockRestore();
  });

  it("should clear instead of persisting on a clearState action", () => {
    const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const engine = createEngine();
    drive(engine, ZERO_MOCK_STATE, Persist.clearState());
    expect(engine.clear).toHaveBeenCalledOnce();
    expect(engine.persist).not.toHaveBeenCalled();
    errorSpy.mockRestore();
  });

  it("should flush the old context, load the target, and dispatch hydrate on a context switch", async () => {
    const engine = createEngine();
    const { dispatch } = drive(engine, STATE, { type: "cluster/select" });
    expect(engine.persist).toHaveBeenCalledWith(STATE, {});
    await vi.waitFor(() => {
      expect(engine.loadSwap).toHaveBeenCalledWith(STATE, CTX, true);
      expect(dispatch).toHaveBeenCalledWith(
        Persist.hydrate({ work: { value: "swapped" } }),
      );
    });
  });

  it("should swap only L2 when the project changes within the same cluster", async () => {
    const engine = createEngine();
    engine.context = { cluster: "c1", project: "p0" };
    drive(engine, STATE, { type: "project/select" });
    await vi.waitFor(() => {
      expect(engine.loadSwap).toHaveBeenCalledWith(STATE, CTX, false);
    });
  });

  it("should let onSwap rewrite the loaded slices before hydrating", async () => {
    const engine = createEngine();
    const next = vi.fn((a: unknown) => a);
    const dispatch = vi.fn();
    const store: MiddlewareAPI = { getState: () => STATE, dispatch };
    Persist.middleware<MockState>({
      engine,
      getContext,
      debounceInterval: TimeSpan.ZERO,
      onSwap: (loaded) => ({ ...loaded, work: { value: "rewritten" } }),
    })(store)(next)({ type: "cluster/select" });
    await vi.waitFor(() => {
      expect(dispatch).toHaveBeenCalledWith(
        Persist.hydrate({ work: { value: "rewritten" } }),
      );
    });
  });

  it("should discard a stale swap when a newer context switch supersedes it", async () => {
    const engine = createEngine();
    let releaseStale: (() => void) | undefined;
    const staleGate = new Promise<void>((resolve) => (releaseStale = resolve));
    engine.loadSwap = vi.fn(async (_state, context: Persist.Context) => {
      if (context.project == null) {
        await staleGate;
        return { project: {}, work: { value: "stale" } };
      }
      return { work: { value: "fresh" } };
    });
    const next = vi.fn((a: unknown) => a);
    const dispatch = vi.fn();
    let state: MockState = {
      cluster: { selected: "c1" },
      project: {},
      work: { value: "w" },
    };
    const store: MiddlewareAPI = { getState: () => state, dispatch };
    const mw = Persist.middleware<MockState>({
      engine,
      getContext,
      debounceInterval: TimeSpan.ZERO,
    })(store)(next);
    mw({ type: "cluster/select" });
    await vi.waitFor(() => expect(engine.loadSwap).toHaveBeenCalledTimes(1));
    state = { ...state, project: { selected: "p1" } };
    mw({ type: "project/select" });
    await vi.waitFor(() => {
      expect(dispatch).toHaveBeenCalledWith(
        Persist.hydrate({ work: { value: "fresh" } }),
      );
    });
    releaseStale?.();
    await vi.waitFor(() => expect(engine.loadSwap).toHaveBeenCalledTimes(2));
    // The slow cluster swap resolved last; hydrating it would wipe the newer
    // project selection with its stale project slice.
    expect(dispatch).toHaveBeenCalledOnce();
  });

  it("should adopt the hydrated context instead of re-swapping on hydrate", () => {
    const engine = createEngine();
    const next = vi.fn((a: unknown) => a);
    const dispatch = vi.fn();
    const store: MiddlewareAPI = { getState: () => STATE, dispatch };
    const mw = Persist.middleware<MockState>({
      engine,
      getContext,
      debounceInterval: TimeSpan.ZERO,
    })(store)(next);
    mw(Persist.hydrate({ work: STATE.work }));
    expect(engine.loadSwap).not.toHaveBeenCalled();
    expect(engine.persist).toHaveBeenCalledWith(STATE, CTX);
  });

  it("should coalesce rapid dispatches into a single persist when debounced", () => {
    vi.useFakeTimers();
    try {
      const engine = createEngine();
      const next = vi.fn((a: unknown) => a);
      const store: MiddlewareAPI = {
        getState: () => ZERO_MOCK_STATE,
        dispatch: vi.fn(),
      };
      const dispatch = Persist.middleware<MockState>({
        engine,
        getContext,
        debounceInterval: TimeSpan.milliseconds(250),
      })(store)(next);
      dispatch({ type: "a" });
      dispatch({ type: "b" });
      dispatch({ type: "c" });
      expect(engine.persist).not.toHaveBeenCalled();
      vi.advanceTimersByTime(250);
      expect(engine.persist).toHaveBeenCalledOnce();
    } finally {
      vi.useRealTimers();
    }
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
    const versionKey = Persist.partitionVersionKey(Persist.l0PartitionBase());
    localStorage.setItem(`${PREFIX}${versionKey}`, JSON.stringify({ version: 1 }));
    localStorage.setItem("unrelated:key", "keep-me");
    Persist.hardClearAndReload();
    await vi.waitFor(() => {
      expect(localStorage.getItem(`${PREFIX}${versionKey}`)).toBeNull();
    });
    expect(localStorage.getItem("unrelated:key")).toBe("keep-me");
  });
});
