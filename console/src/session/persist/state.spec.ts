// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type MiddlewareAPI } from "@reduxjs/toolkit";
import { TimeSpan } from "@synnaxlabs/x";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { z } from "zod";

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
  core: { selected?: string };
  project: { selected?: string };
  work: { value: string; transient?: string };
}

const ZERO_MOCK_STATE: MockState = {
  core: {},
  project: {},
  work: { value: "0.0.0", transient: "zero" },
};

const selectedZ = z.object({ selected: z.string().optional() });
const workZ = z.object({
  value: z.string(),
  transient: z.string().optional(),
});

const SCOPES: Persist.Scopes<MockState> = {
  global: { core: selectedZ },
  core: { project: selectedZ },
  project: { work: workZ },
  transient: [],
};

const getContext = (state: MockState): Persist.Context => ({
  core: state.core.selected,
  project: state.project.selected,
});

const CTX: Persist.Context = { core: "c1", project: "p1" };

const STATE: MockState = {
  core: { selected: "c1" },
  project: { selected: "p1" },
  work: { value: "16.2.0", transient: "drag" },
};

// One state key plus one slot pointer for the global partition.
const GLOBAL_KEYS = 2;

/** A store whose writes to keys matching failOn reject, standing in for a full disk. */
class FailingKV extends Persist.MemoryKV {
  failOn: RegExp | null = null;

  override async setMany(entries: Persist.Entry[]): Promise<void> {
    if (entries.some(({ key }) => this.failOn?.test(key) === true))
      throw new Error("disk full");
    await super.setMany(entries);
  }

  override async delete(key: string): Promise<void> {
    if (this.failOn?.test(key) === true) throw new Error("disk full");
    await super.delete(key);
  }
}

const openPersist = async (
  store: Persist.MemoryKV,
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
  store: Persist.MemoryKV,
  overrides: Partial<Persist.Config<MockState>> = {},
) => {
  const { initialState, middleware } = await openPersist(store, overrides);
  let state = initialState ?? ZERO_MOCK_STATE;
  let settled = 0;
  let hydrates = 0;
  const next = vi.fn((action: unknown) => action);
  const dispatched = vi.fn((action: unknown) => {
    if (!Persist.hydrate.match(action)) return;
    hydrates++;
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
      await vi.waitFor(() => expect(hydrates).toBeGreaterThan(settled));
      settled = hydrates;
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
 * Walks into a context the way production does: the Core is chosen first, and only
 * once its partition has hydrated does the project it names become selectable.
 */
const enter = async (driver: Driver, { core, project }: Persist.Context) => {
  if (driver.getState().core.selected !== core) {
    driver.dispatch(
      { type: "Core/select" },
      { ...driver.getState(), core: { selected: core } },
    );
    await driver.settle();
  }
  // The Core's partition may already name the project, in which case nothing moved.
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
      const { initialState } = await openPersist(new Persist.MemoryKV());
      expect(initialState).toEqual(ZERO_MOCK_STATE);
    });

    it("should compose the global, selected Core, and active project partitions", async () => {
      const store = new Persist.MemoryKV();
      const driver = await createDriver(store);
      await enter(driver, CTX);
      await edit(driver, "16.2.0");
      expect(await driver.composed()).toEqual(STATE);
    });

    it("should stop at the global partition when no Core was selected", async () => {
      const store = new Persist.MemoryKV();
      const driver = await createDriver(store);
      driver.dispatch({ type: "work/edit" }, { ...ZERO_MOCK_STATE, work: STATE.work });
      await vi.waitFor(async () => expect(await store.length()).toBe(GLOBAL_KEYS));
      const composed = await driver.composed();
      expect(composed?.project).toEqual(ZERO_MOCK_STATE.project);
      expect(composed?.work).toEqual(ZERO_MOCK_STATE.work);
    });
  });

  describe("partitions", () => {
    it("should write only the global partition when no Core is in context", async () => {
      const store = new Persist.MemoryKV();
      const driver = await createDriver(store);
      driver.dispatch({ type: "work/edit" }, { ...ZERO_MOCK_STATE, work: STATE.work });
      await vi.waitFor(async () => expect(await store.length()).toBe(GLOBAL_KEYS));
    });

    it("should skip the project partition when no project is in context", async () => {
      const store = new Persist.MemoryKV();
      const driver = await createDriver(store);
      await enter(driver, { core: "c1" });
      driver.dispatch(
        { type: "work/edit" },
        { ...driver.getState(), work: { value: "c1-work" } },
      );
      await driver.flushed((s) => s?.core.selected === "c1");
      // The project-scoped work never reached disk, so it composes from zero.
      expect((await driver.composed())?.work).toEqual(ZERO_MOCK_STATE.work);
    });

    it("should bound a partition to four slots", async () => {
      const store = new Persist.MemoryKV();
      const driver = await createDriver(store);
      await enter(driver, CTX);
      for (let i = 0; i < 10; i++) await edit(driver, `16.2.${i}`);
      // Four ring entries plus the slot pointer.
      expect(
        (await store.keys()).filter((k) => k.startsWith("project.c1.p1.")),
      ).toHaveLength(5);
    });

    it("should leave a partition alone while its slices hold still", async () => {
      const store = new Persist.MemoryKV();
      const driver = await createDriver(store);
      await enter(driver, CTX);
      for (let i = 0; i < 10; i++) await edit(driver, `16.2.${i}`);
      // Every edit is project-scoped, so the global ring still holds its one write.
      expect(
        (await store.keys()).filter((k) => k.startsWith("global.")).sort(),
      ).toEqual(["global.0", "global.slot"]);
    });

    it("should fall back to the first slot when the slot pointer is corrupt", async () => {
      const store = new Persist.MemoryKV();
      const driver = await createDriver(store);
      await enter(driver, CTX);
      await edit(driver, "16.2.0");
      // A pointer naming no slot in the ring reads slot 0, so the partition keeps
      // its state instead of composing from zero off a key that does not exist.
      for (const slot of ["one", 4, -1, 1.5, null]) {
        await store.set("global.slot", { slot });
        expect((await openPersist(store)).initialState?.core).toEqual(STATE.core);
      }
    });

    it("should keep the last version loadable when a state write fails", async () => {
      const store = new FailingKV();
      const driver = await createDriver(store);
      await enter(driver, CTX);
      await edit(driver, "16.2.0");
      const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
      // Every ring entry is keyed on its slot number. The pointer rides in the same
      // batch, so rejecting the state write leaves it naming the last good slot.
      store.failOn = /\.\d$/;
      driver.dispatch(
        { type: "work/edit" },
        { ...driver.getState(), work: { value: "16.3.0", transient: "drag" } },
      );
      await vi.waitFor(() => expect(errorSpy).toHaveBeenCalled());
      expect((await driver.composed())?.work.value).toBe("16.2.0");
      errorSpy.mockRestore();
    });

    it("should leave the pointer where it was when a write is rejected", async () => {
      const store = new FailingKV();
      const driver = await createDriver(store);
      await enter(driver, CTX);
      await edit(driver, "16.2.0");
      const before = await store.get(`project.c1.p1.slot`);
      const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
      store.failOn = /^project\./;
      driver.dispatch(
        { type: "work/edit" },
        { ...driver.getState(), work: { value: "16.3.0" } },
      );
      await vi.waitFor(() => expect(errorSpy).toHaveBeenCalled());
      expect(await store.get(`project.c1.p1.slot`)).toEqual(before);
      errorSpy.mockRestore();
    });

    it("should scope slices to the context they were written under", async () => {
      const store = new Persist.MemoryKV();
      const driver = await createDriver(store);
      await enter(driver, CTX);
      await edit(driver, "p1-work");
      await enter(driver, { core: "c2", project: "p2" });
      await edit(driver, "p2-work");
      await enter(driver, CTX);
      expect(driver.getState().work.value).toEqual("p1-work");
    });
  });

  describe("secondary windows", () => {
    it("should neither compose nor persist state outside the main window", async () => {
      mocks.engine = "tauri";
      mocks.label = "child-window";
      const store = new Persist.MemoryKV();
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
      const store = new Persist.MemoryKV();
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

  describe("schemas", () => {
    const createPersisted = async (store: Persist.MemoryKV) => {
      const driver = await createDriver(store);
      await enter(driver, CTX);
      await edit(driver, "16.2.0");
    };

    it("should drop fields the slice's schema does not declare", async () => {
      const store = new Persist.MemoryKV();
      await createPersisted(store);
      const { initialState } = await openPersist(store, {
        scopes: { ...SCOPES, project: { work: z.object({ value: z.string() }) } },
      });
      expect(initialState?.work).toEqual({ value: "16.2.0" });
    });

    it("should fall back to the slice's initial state when the stored bytes fail its schema", async () => {
      const store = new Persist.MemoryKV();
      await createPersisted(store);
      const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
      const { initialState } = await openPersist(store, {
        scopes: { ...SCOPES, project: { work: workZ.refine(() => false) } },
      });
      expect(initialState?.work).toEqual(ZERO_MOCK_STATE.work);
      expect(initialState?.core).toEqual(STATE.core);
      expect(errorSpy).toHaveBeenCalled();
      errorSpy.mockRestore();
    });

    it("should leave the other slices of a partition alone when one fails", async () => {
      const store = new Persist.MemoryKV();
      await createPersisted(store);
      const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
      const { initialState } = await openPersist(store, {
        scopes: {
          ...SCOPES,
          global: { core: selectedZ.refine(() => false) },
        },
      });
      expect(initialState?.core).toEqual(ZERO_MOCK_STATE.core);
      errorSpy.mockRestore();
    });
  });

  describe("an unusable store", () => {
    const broken = (): Persist.SugaredKV => {
      const fail = async () => {
        throw new Error("storage is blocked");
      };
      return {
        get: fail,
        set: fail,
        setMany: fail,
        delete: fail,
        length: fail,
        keys: fail,
        clear: fail,
      };
    };

    it("should still compose the initial state", async () => {
      const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
      const { initialState, middleware } = await Persist.open<MockState>({
        initial: ZERO_MOCK_STATE,
        scopes: SCOPES,
        getContext,
        openKV: broken,
      });
      expect(initialState).toEqual(ZERO_MOCK_STATE);
      expect(middleware).toBeDefined();
      errorSpy.mockRestore();
    });

    it("should announce that the store is unavailable", async () => {
      const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
      const { middleware } = await Persist.open<MockState>({
        initial: ZERO_MOCK_STATE,
        scopes: SCOPES,
        getContext,
        openKV: broken,
      });
      const dispatch = vi.fn();
      middleware({
        getState: () => ZERO_MOCK_STATE,
        dispatch,
      } as never)((a) => a)({ type: "work/edit" });
      expect(dispatch).toHaveBeenCalledWith(Persist.storeUnavailable());
      errorSpy.mockRestore();
    });

    it("should announce it only once", async () => {
      const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
      const { middleware } = await Persist.open<MockState>({
        initial: ZERO_MOCK_STATE,
        scopes: SCOPES,
        getContext,
        openKV: broken,
      });
      const dispatch = vi.fn();
      const next = middleware({
        getState: () => ZERO_MOCK_STATE,
        dispatch,
      } as never)((a) => a);
      next({ type: "work/edit" });
      next({ type: "work/edit" });
      const announcements = dispatch.mock.calls.filter(
        ([action]) =>
          (action as { type: string }).type === Persist.storeUnavailable.type,
      );
      expect(announcements).toHaveLength(1);
      errorSpy.mockRestore();
    });
  });

  describe("scope coverage", () => {
    it("should throw when a slice is in no scope and not transient", async () => {
      await expect(
        openPersist(new Persist.MemoryKV(), {
          scopes: { ...SCOPES, project: {} },
        }),
      ).rejects.toThrow("work");
    });

    it("should throw when a slice is declared in two scopes", async () => {
      await expect(
        openPersist(new Persist.MemoryKV(), {
          scopes: { ...SCOPES, global: { core: selectedZ, work: workZ } },
        }),
      ).rejects.toThrow("more than one scope");
    });

    it("should accept a slice declared transient instead of scoped", async () => {
      await expect(
        openPersist(new Persist.MemoryKV(), {
          scopes: { ...SCOPES, project: {}, transient: ["work"] },
        }),
      ).resolves.toBeDefined();
    });
  });
});

describe("Persist.middleware", () => {
  beforeEach(() => {
    mocks.engine = "web";
    mocks.label = "main";
  });

  it("should pass the action through to next and return its result", async () => {
    const driver = await createDriver(new Persist.MemoryKV());
    const action = { type: "any/action" };
    expect(driver.dispatch(action)).toBe(action);
    expect(driver.next).toHaveBeenCalledWith(action);
  });

  it("should persist the current store state under the current context", async () => {
    const store = new Persist.MemoryKV();
    const driver = await createDriver(store);
    await enter(driver, CTX);
    await edit(driver, "edited");
  });

  it("should delete a Core's partitions on a purge action", async () => {
    const store = new Persist.MemoryKV();
    const driver = await createDriver(store);
    await enter(driver, CTX);
    await edit(driver, "16.2.0");
    driver.dispatch(Persist.purge("c1"));
    await vi.waitFor(async () =>
      expect((await store.keys()).filter((k) => k.includes("c1"))).toHaveLength(0),
    );
    // The Core the global partition names outlives the state stored under it.
    expect(await store.get("global.0")).toEqual({ core: { selected: "c1" } });
  });

  it("should purge behind the swap that leaves the purged Core", async () => {
    const store = new Persist.MemoryKV();
    const driver = await createDriver(store);
    await enter(driver, CTX);
    await edit(driver, "16.2.0");
    // Production dispatches both in one tick: the switch flushes c1's partitions on
    // its way out, so a purge that ran first would find the keys written back.
    driver.dispatch(
      { type: "Core/select" },
      { ...driver.getState(), core: { selected: "c2" }, project: {} },
    );
    driver.dispatch(Persist.purge("c1"));
    await driver.settle();
    await vi.waitFor(async () =>
      expect((await store.keys()).filter((k) => k.includes("c1"))).toHaveLength(0),
    );
  });

  it("should report a failed purge instead of throwing", async () => {
    const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const store = new FailingKV();
    const driver = await createDriver(store);
    await enter(driver, CTX);
    await edit(driver, "16.2.0");
    store.failOn = /^core\.c1\./;
    driver.dispatch(Persist.purge("c1"));
    await vi.waitFor(() =>
      expect(errorSpy).toHaveBeenCalledWith(
        "failed to purge stored Core state",
        expect.anything(),
      ),
    );
    errorSpy.mockRestore();
  });

  // The revert and clear branches reload the window, which jsdom cannot perform; the
  // production code swallows that failure via .catch, so we suppress the expected log.
  it("should revert the innermost partition one version on a revertState action", async () => {
    const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const store = new Persist.MemoryKV();
    const driver = await createDriver(store);
    await enter(driver, CTX);
    await edit(driver, "16.2.0");
    await edit(driver, "16.2.1");
    driver.dispatch(Persist.revertState());
    await driver.flushed(workIs("16.2.0"));
    errorSpy.mockRestore();
  });

  it("should report a failed revert instead of reloading", async () => {
    const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const store = new FailingKV();
    const driver = await createDriver(store);
    await enter(driver, CTX);
    await edit(driver, "16.2.0");
    await edit(driver, "16.2.1");
    store.failOn = /\.slot$/;
    driver.dispatch(Persist.revertState());
    await vi.waitFor(() =>
      expect(errorSpy).toHaveBeenCalledWith(
        "failed to revert state",
        expect.anything(),
      ),
    );
    expect((await driver.composed())?.work.value).toBe("16.2.1");
    errorSpy.mockRestore();
  });

  it("should fall back to the initial state when reverting past the first version", async () => {
    const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const store = new Persist.MemoryKV();
    const driver = await createDriver(store);
    await enter(driver, CTX);
    await edit(driver, "16.2.0");
    driver.dispatch(Persist.revertState());
    await driver.flushed(workIs(ZERO_MOCK_STATE.work.value));
    errorSpy.mockRestore();
  });

  it("should clear the entire store on a clearState action", async () => {
    const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const store = new Persist.MemoryKV();
    const driver = await createDriver(store);
    await enter(driver, CTX);
    await edit(driver, "16.2.0");
    driver.dispatch(Persist.clearState());
    await vi.waitFor(async () => expect(await store.length()).toBe(0));
    errorSpy.mockRestore();
  });

  it("should load the target context and dispatch hydrate on a context switch", async () => {
    const driver = await createDriver(new Persist.MemoryKV());
    await enter(driver, CTX);
    await edit(driver, "c1-work");
    await enter(driver, { core: "c2", project: "p2" });
    await enter(driver, { core: "c1" });
    // The Core's partition names the project whose partition holds the work.
    expect(driver.dispatched).toHaveBeenLastCalledWith(
      Persist.hydrate({
        project: { selected: "p1" },
        work: { value: "c1-work", transient: "drag" },
      }),
    );
  });

  it("should mark the swap window with beginSwap before hydrate", async () => {
    const driver = await createDriver(new Persist.MemoryKV());
    await enter(driver, CTX);
    const calls = driver.dispatched.mock.calls.map(
      ([action]) => (action as { type: string }).type,
    );
    const begin = calls.indexOf(Persist.beginSwap.type);
    const hydrated = calls.indexOf(Persist.hydrate.type);
    expect(begin).toBeGreaterThanOrEqual(0);
    expect(hydrated).toBeGreaterThan(begin);
  });

  it("should swap only the project partition when the Core is unchanged", async () => {
    const driver = await createDriver(new Persist.MemoryKV());
    await enter(driver, CTX);
    await edit(driver, "p1-work");
    await enter(driver, { core: "c1", project: "p2" });
    await enter(driver, CTX);
    // The Core-scoped project slice stayed put; only the project's work swapped.
    expect(driver.dispatched).toHaveBeenLastCalledWith(
      Persist.hydrate({ work: { value: "p1-work", transient: "drag" } }),
    );
  });

  it("should hydrate zero slices for a never-visited context", async () => {
    const driver = await createDriver(new Persist.MemoryKV());
    await enter(driver, { core: "fresh" });
    expect(driver.dispatched).toHaveBeenCalledWith(
      Persist.hydrate({
        project: ZERO_MOCK_STATE.project,
        work: ZERO_MOCK_STATE.work,
      }),
    );
  });

  it("should adopt the hydrated context instead of re-swapping on hydrate", async () => {
    const store = new Persist.MemoryKV();
    const driver = await createDriver(store);
    await enter(driver, CTX);
    const swaps = driver.dispatched.mock.calls.length;
    driver.dispatch(Persist.hydrate({ work: STATE.work }), STATE);
    await driver.flushed(workIs("16.2.0"));
    expect(driver.dispatched.mock.calls.length).toBe(swaps);
  });

  it("should discard a stale swap when a newer context switch supersedes it", async () => {
    const store = new Persist.MemoryKV();
    const seed = await createDriver(store);
    await enter(seed, CTX);
    await edit(seed, "fresh");
    let releaseStale: (() => void) | undefined;
    const staleGate = new Promise<void>((resolve) => (releaseStale = resolve));
    const gateHit = vi.fn();
    const gateDone = vi.fn();
    // Holds the slow Core's state read so its swap is still in flight when the next
    // switch starts and finishes. Its slot pointer stays readable so persists pass.
    const gated: Persist.SugaredKV = {
      get: async <V>(key: string): Promise<V | null> => {
        if (/^core\.slow\.\d+$/.test(key)) {
          gateHit();
          await staleGate;
          const value = await store.get<V>(key);
          gateDone();
          return value;
        }
        return await store.get<V>(key);
      },
      set: async (key, value) => await store.set(key, value),
      setMany: async (entries) => await store.setMany(entries),
      delete: async (key) => await store.delete(key),
      length: async () => await store.length(),
      keys: async () => await store.keys(),
      clear: async () => await store.clear(),
    };
    const driver = await createDriver(store, { openKV: () => gated });
    driver.dispatch(
      { type: "Core/select" },
      { ...driver.getState(), core: { selected: "slow" }, project: {} },
    );
    await vi.waitFor(() => expect(gateHit).toHaveBeenCalled());
    driver.dispatch(
      { type: "Core/select" },
      { ...driver.getState(), core: { selected: "c2" }, project: {} },
    );
    await driver.settle();
    const hydrates = (): number =>
      driver.dispatched.mock.calls.filter(
        ([action]) => (action as { type: string }).type === Persist.hydrate.type,
      ).length;
    expect(hydrates()).toBe(1);
    releaseStale?.();
    // The released read is the stale swap's last async hop, so once it lands the swap
    // has fully resolved. Its hydrate would clobber the newer Core's slices.
    await vi.waitFor(() => expect(gateDone).toHaveBeenCalled());
    expect(hydrates()).toBe(1);
  });

  it("should coalesce rapid dispatches into a single persist when debounced", async () => {
    const store = new Persist.MemoryKV();
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
  let errorSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    // The reload after clearing is a no-op under jsdom and its failure is swallowed by
    // the production .catch; suppress the expected error log.
    errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
  });

  afterEach(() => {
    errorSpy.mockRestore();
  });

  it("should clear the session store", async () => {
    const db = Persist.openSugaredKV(Persist.STORE_NAME);
    await db.set("global.slot", { slot: 1 });
    Persist.hardClearAndReload();
    await vi.waitFor(async () => {
      expect(await db.get("global.slot")).toBeNull();
    });
  });
});
