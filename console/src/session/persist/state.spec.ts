// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { MiddlewareAPI } from "@reduxjs/toolkit";
import { sleep, TimeSpan, unique } from "@synnaxlabs/x";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { z } from "zod";

const mocks = vi.hoisted((): { engine: "web" | "tauri"; label: string } => ({
  engine: "web",
  label: "main",
}));

vi.mock("@/session/runtime", async (importOriginal) => {
  const { mockRuntimeEngine } = await import("@/testutil/runtime");
  return await mockRuntimeEngine(importOriginal, mocks);
});

vi.mock("@tauri-apps/api/window", () => ({
  getCurrentWindow: () => ({ label: mocks.label }),
}));

import { Persist } from "@/session/persist";
import { Window } from "@/session/window";
import { MemoryKV } from "@/testutil";

interface MockState {
  core: { selected?: string };
  project: { selected?: string };
  work: { value: string; transient?: string };
  /**
   * A window-keyed slice, shaped the way every real one is: per-window state under
   * windows, and a shared field beside it.
   */
  view: { label: string; windows: Record<string, { value: string }> };
  /** Stands in for the drift bookkeeping the project partition holds. */
  windows: string[];
}

const ZERO_MOCK_STATE: MockState = {
  core: {},
  project: {},
  work: { value: "0.0.0", transient: "zero" },
  view: { label: "none", windows: {} },
  windows: [],
};

const selectedZ = z.object({ selected: z.string().optional() });
const workZ = z.object({
  value: z.string(),
  transient: z.string().optional(),
});

const viewZ = z.object({
  label: z.string().default("none"),
  windows: z.record(z.string(), z.object({ value: z.string() })),
});

const SCOPES: Persist.Scopes<MockState> = {
  global: { core: selectedZ },
  core: { project: selectedZ },
  project: { work: workZ, windows: z.string().array() },
  window: { view: viewZ },
  transient: [],
};

const getContext = (state: MockState): Persist.Context => ({
  core: state.core.selected,
  project: state.project.selected,
});

const CTX: Persist.Context = { core: "c1", project: "p1" };

const WINDOW = "w1";

const STATE: MockState = {
  core: { selected: "c1" },
  project: { selected: "p1" },
  work: { value: "16.2.0", transient: "drag" },
  view: { label: "none", windows: {} },
  windows: [],
};

// One state key plus one slot pointer for the global partition.
const GLOBAL_KEYS = 2;

/** A store whose writes to keys matching failOn reject, standing in for a full disk. */
class FailingKV extends MemoryKV {
  failOn: RegExp | null = null;

  override async setMany(entries: Persist.Entry[]): Promise<void> {
    if (entries.some(({ key }) => this.failOn?.test(key) === true))
      throw new Error("disk full");
    await super.setMany(entries);
  }

  override async deleteMany(keys: string[]): Promise<void> {
    if (keys.some((key) => this.failOn?.test(key) === true))
      throw new Error("disk full");
    await super.deleteMany(keys);
  }
}

/** Counts the batched calls production makes, so a spec can assert it writes once. */
class CountingKV extends MemoryKV {
  readonly writes: Persist.Entry[][] = [];
  readonly deletes: string[][] = [];

  override async setMany(entries: Persist.Entry[]): Promise<void> {
    this.writes.push(entries);
    await super.setMany(entries);
  }

  override async deleteMany(keys: string[]): Promise<void> {
    this.deletes.push(keys);
    await super.deleteMany(keys);
  }
}

/** A store whose reads of keys matching failOn reject, standing in for a lost disk. */
class ReadFailingKV extends CountingKV {
  failOn: RegExp | null = null;

  override async get<V>(key: string): Promise<V | null> {
    if (this.failOn?.test(key) === true) throw new Error("read failed");
    return await super.get<V>(key);
  }
}

/** Holds reads of the keys matching pattern, standing in for a slow disk. */
class GatedKV implements Persist.SugaredKV {
  held = false;
  readonly hit = vi.fn();
  private readonly gate = Promise.withResolvers<void>();

  constructor(
    private readonly store: MemoryKV,
    private readonly pattern: RegExp,
  ) {}

  release(): void {
    this.gate.resolve();
  }

  async get<V>(key: string): Promise<V | null> {
    if (this.held && this.pattern.test(key)) {
      this.hit();
      await this.gate.promise;
    }
    return await this.store.get<V>(key);
  }

  set: Persist.SugaredKV["set"] = async (key, value) =>
    await this.store.set(key, value);
  setMany: Persist.SugaredKV["setMany"] = async (entries) =>
    await this.store.setMany(entries);
  delete: Persist.SugaredKV["delete"] = async (key) => await this.store.delete(key);
  deleteMany: Persist.SugaredKV["deleteMany"] = async (keys) =>
    await this.store.deleteMany(keys);
  length: Persist.SugaredKV["length"] = async () => await this.store.length();
  keys: Persist.SugaredKV["keys"] = async () => await this.store.keys();
  clear: Persist.SugaredKV["clear"] = async () => await this.store.clear();
}

/** Lists every work value written to a project's ring in c1, in write order. */
const works = (store: CountingKV, project: string): Array<string | undefined> =>
  store.writes
    .flat()
    .filter(({ key }) => new RegExp(`^project\\.c1\\.${project}\\.\\d+$`).test(key))
    .map(({ value }) => z.object({ work: workZ }).safeParse(value).data?.work.value);

const openPersist = async (
  store: MemoryKV,
  overrides: Partial<Persist.Config<MockState>> = {},
) =>
  await Persist.open<MockState>({
    initial: ZERO_MOCK_STATE,
    scopes: SCOPES,
    getContext,
    getWindows: (state) => state.windows,
    lens: Window.LENS,
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
  store: MemoryKV,
  overrides: Partial<Persist.Config<MockState>> = {},
) => {
  const { initialState, middleware } = await openPersist(store, overrides);
  let state = initialState ?? ZERO_MOCK_STATE;
  let settled = 0;
  let hydrates = 0;
  const next = vi.fn((action: unknown) => action);
  const dispatched = vi.fn((action: unknown) => {
    // The swap's end rides back through the chain, as it would in the redux store.
    if (Persist.endSwap.match(action)) {
      run(action);
      return;
    }
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
    {
      ...driver.getState(),
      work: { value, transient: "drag" },
    },
  );
  await driver.flushed(workIs(value));
};

/** Edits the work slice and flushes it, for a driver whose debounce may not fire. */
const commit = async (driver: Driver, value: string) => {
  driver.dispatch(
    { type: "work/edit" },
    { ...driver.getState(), work: { value, transient: "drag" } },
  );
  await Persist.flush((action) => driver.dispatch(action));
};

const P2_STATE = /^project\.c1\.p2\.\d+$/;

/**
 * Leaves p2 holding p2-work for p1 holding p1-work, then starts a swap back to p2 that
 * stays in flight until p2 is released.
 */
const swapToHeldP2 = async (driver: Driver, p2: GatedKV) => {
  await enter(driver, { core: "c1", project: "p2" });
  await commit(driver, "p2-work");
  await enter(driver, CTX);
  await commit(driver, "p1-work");
  p2.held = true;
  driver.dispatch(
    { type: "project/select" },
    { ...driver.getState(), project: { selected: "p2" } },
  );
  await vi.waitFor(() => expect(p2.hit).toHaveBeenCalled());
};

/** Edits one window's view of the project, the way a window-scoped slice changes. */
const editView = async (driver: Driver, key: string, value: string) => {
  const state = driver.getState();
  driver.dispatch(
    { type: "view/edit" },
    {
      ...state,
      view: { ...state.view, windows: { ...state.view.windows, [key]: { value } } },
      windows: unique.unique([...state.windows, key]),
    },
  );
  await driver.flushed((state) => state?.view.windows[key]?.value === value);
};

/** Edits the view slice's shared field, the part outside its windows record. */
const editLabel = async (driver: Driver, value: string) => {
  const state = driver.getState();
  driver.dispatch(
    { type: "view/label" },
    { ...state, view: { ...state.view, label: value } },
  );
  await driver.flushed((state) => state?.view.label === value);
};

describe("Persist.open", () => {
  beforeEach(() => {
    mocks.engine = "web";
    mocks.label = "main";
  });

  describe("composition", () => {
    it("should start from the initial state when nothing has been persisted", async () => {
      const { initialState } = await openPersist(new MemoryKV());
      expect(initialState).toEqual(ZERO_MOCK_STATE);
    });

    it("should apply the migrated slices to an empty store", async () => {
      const { initialState } = await openPersist(new MemoryKV(), {
        migrate: async () => ({ work: { value: "0.56.0" } }),
      });
      expect(initialState?.work.value).toBe("0.56.0");
    });

    it("should not consult the migration once the store holds a session", async () => {
      const store = new MemoryKV();
      const driver = await createDriver(store);
      await enter(driver, CTX);
      await edit(driver, "16.2.0");
      const migrate = vi.fn(async () => ({ work: { value: "0.56.0" } }));
      const { initialState } = await openPersist(store, { migrate });
      expect(migrate).not.toHaveBeenCalled();
      expect(initialState?.work.value).toBe("16.2.0");
    });

    it("should still open when the migration throws", async () => {
      const onError = vi.fn<(error: Error) => void>();
      const { initialState } = await openPersist(new MemoryKV(), {
        onError,
        migrate: async () => {
          throw new Error("unreadable legacy store");
        },
      });
      expect(initialState).toEqual(ZERO_MOCK_STATE);
      expect(onError).toHaveBeenCalledWith(
        expect.objectContaining({
          message: "failed to carry the previous release's state over",
        }),
      );
    });

    it("should compose the global, selected Core, and active project partitions", async () => {
      const store = new MemoryKV();
      const driver = await createDriver(store);
      await enter(driver, CTX);
      await edit(driver, "16.2.0");
      expect(await driver.composed()).toEqual(STATE);
    });

    it("should stop at the global partition when no Core was selected", async () => {
      const store = new MemoryKV();
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
      const store = new MemoryKV();
      const driver = await createDriver(store);
      driver.dispatch({ type: "work/edit" }, { ...ZERO_MOCK_STATE, work: STATE.work });
      await vi.waitFor(async () => expect(await store.length()).toBe(GLOBAL_KEYS));
    });

    it("should skip the project partition when no project is in context", async () => {
      const store = new MemoryKV();
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

    it("should store a window's slices under its own partition", async () => {
      const store = new MemoryKV();
      const driver = await createDriver(store);
      await enter(driver, CTX);
      await editView(driver, WINDOW, "left");
      await editView(driver, "w2", "right");
      const keys = await store.keys();
      expect(keys).toContain(`window.c1.p1.${WINDOW}.slot`);
      expect(keys).toContain("window.c1.p1.w2.slot");
      // Each partition holds the slice narrowed to its own window, so the bytes still
      // parse through the slice's schema.
      expect(await store.get(`window.c1.p1.${WINDOW}.0`)).toEqual({
        view: { windows: { [WINDOW]: { value: "left" } } },
      });
    });

    it("should compose every window's slices on open", async () => {
      const store = new MemoryKV();
      const driver = await createDriver(store);
      await enter(driver, CTX);
      await editView(driver, WINDOW, "left");
      await editView(driver, "w2", "right");
      expect((await driver.composed())?.view.windows).toEqual({
        [WINDOW]: { value: "left" },
        w2: { value: "right" },
      });
    });

    it("should delete a window's partition once the window is gone", async () => {
      const store = new MemoryKV();
      const driver = await createDriver(store);
      await enter(driver, CTX);
      await editView(driver, WINDOW, "left");
      await editView(driver, "w2", "right");
      // Window keys are minted fresh per open, so a partition left behind is one the
      // store carries forever.
      driver.dispatch(
        { type: "window/closed" },
        { ...driver.getState(), windows: [WINDOW] },
      );
      await vi.waitFor(async () =>
        expect((await store.keys()).filter((key) => key.includes("w2"))).toHaveLength(
          0,
        ),
      );
      expect(await store.keys()).toContain(`window.c1.p1.${WINDOW}.slot`);
    });

    it("should leave a window with no entry in a slice alone across sessions", async () => {
      const store = new CountingKV();
      const driver = await createDriver(store);
      await enter(driver, CTX);
      // The window list comes from drift, not from the slice, so a window that never
      // touched the view slice still gets a partition staged and written.
      driver.dispatch(
        { type: "window/open" },
        { ...driver.getState(), windows: [WINDOW] },
      );
      await vi.waitFor(async () =>
        expect(await store.keys()).toContain(`window.c1.p1.${WINDOW}.slot`),
      );
      const onError = vi.fn<(error: Error) => void>();
      const sessionOneWrites = store.writes.length;
      const driver2 = await createDriver(store, { onError });
      await edit(driver2, "16.3.0");
      // The reread bytes must compose back to what re-staging narrows, or every
      // launch rewrites the partition and burns a revert slot.
      const rewrites = store.writes
        .slice(sessionOneWrites)
        .flat()
        .filter(({ key }) => key.startsWith("window."));
      expect(rewrites).toHaveLength(0);
      expect(onError).not.toHaveBeenCalled();
    });

    it("should bound a partition to four slots", async () => {
      const store = new MemoryKV();
      const driver = await createDriver(store);
      await enter(driver, CTX);
      for (let i = 0; i < 10; i++) await edit(driver, `16.2.${i}`);
      // Four ring entries plus the slot pointer.
      expect(
        (await store.keys()).filter((k) => k.startsWith("project.c1.p1.")),
      ).toHaveLength(5);
    });

    it("should write every partition it touches in one batch", async () => {
      const store = new CountingKV();
      const driver = await createDriver(store);
      await enter(driver, CTX);
      await edit(driver, "16.2.0");
      store.writes.length = 0;
      // Leaving a project flushes the Core and project partitions together, so a
      // batched write commits both in one call.
      driver.dispatch(
        { type: "project/select" },
        {
          ...driver.getState(),
          project: { selected: "p2" },
          work: { value: "16.2.1" },
        },
      );
      await driver.settle();
      const keys = store.writes[0].map(({ key }) => key);
      expect(keys.filter((key) => key.startsWith("core.c1."))).toHaveLength(2);
      expect(keys.filter((key) => key.startsWith("project.c1.p1."))).toHaveLength(2);
    });

    it("should leave a partition alone while its slices hold still", async () => {
      const store = new MemoryKV();
      const driver = await createDriver(store);
      await enter(driver, CTX);
      for (let i = 0; i < 10; i++) await edit(driver, `16.2.${i}`);
      // Every edit is project-scoped, so the global ring still holds its one write.
      expect(
        (await store.keys()).filter((k) => k.startsWith("global.")).sort(),
      ).toEqual(["global.0", "global.slot"]);
    });

    it("should fall back to the first slot when the slot pointer is corrupt", async () => {
      const store = new MemoryKV();
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
      const onError = vi.fn<(error: Error) => void>();
      const driver = await createDriver(store, { onError });
      await enter(driver, CTX);
      await edit(driver, "16.2.0");
      // Every ring entry is keyed on its slot number. The pointer rides in the same
      // batch, so rejecting the state write leaves it naming the last good slot.
      store.failOn = /\.\d$/;
      driver.dispatch(
        { type: "work/edit" },
        { ...driver.getState(), work: { value: "16.3.0", transient: "drag" } },
      );
      await vi.waitFor(() => expect(onError).toHaveBeenCalled());
      expect((await driver.composed())?.work.value).toBe("16.2.0");
    });

    it("should leave the pointer where it was when a write is rejected", async () => {
      const store = new FailingKV();
      const onError = vi.fn<(error: Error) => void>();
      const driver = await createDriver(store, { onError });
      await enter(driver, CTX);
      await edit(driver, "16.2.0");
      const before = await store.get(`project.c1.p1.slot`);
      store.failOn = /^project\./;
      driver.dispatch(
        { type: "work/edit" },
        { ...driver.getState(), work: { value: "16.3.0" } },
      );
      await vi.waitFor(() => expect(onError).toHaveBeenCalled());
      expect(await store.get(`project.c1.p1.slot`)).toEqual(before);
    });

    it("should scope slices to the context they were written under", async () => {
      const store = new MemoryKV();
      const driver = await createDriver(store);
      await enter(driver, CTX);
      await edit(driver, "p1-work");
      await enter(driver, { core: "c2", project: "p2" });
      await edit(driver, "p2-work");
      await enter(driver, CTX);
      expect(driver.getState().work.value).toEqual("p1-work");
    });
  });

  describe("shared fields", () => {
    it("should store a window slice's shared fields in the project partition", async () => {
      const store = new MemoryKV();
      const driver = await createDriver(store);
      await enter(driver, CTX);
      await editView(driver, WINDOW, "left");
      await editLabel(driver, "tagged");
      const slot = (await store.get<{ slot: number }>("project.c1.p1.slot"))?.slot ?? 0;
      const stored = await store.get<Partial<MockState>>(`project.c1.p1.${slot}`);
      expect(stored?.view).toEqual({ label: "tagged", windows: {} });
      // The window's partition holds its windows entry alone, no shared fields.
      expect(await store.get(`window.c1.p1.${WINDOW}.0`)).toEqual({
        view: { windows: { [WINDOW]: { value: "left" } } },
      });
    });

    it("should compose shared fields with every window's entries", async () => {
      const store = new MemoryKV();
      const driver = await createDriver(store);
      await enter(driver, CTX);
      await editLabel(driver, "tagged");
      await editView(driver, WINDOW, "left");
      await editView(driver, "w2", "right");
      expect((await driver.composed())?.view).toEqual({
        label: "tagged",
        windows: { [WINDOW]: { value: "left" }, w2: { value: "right" } },
      });
    });

    it("should leave window partitions alone when only a shared field changes", async () => {
      const store = new CountingKV();
      const driver = await createDriver(store);
      await enter(driver, CTX);
      await editView(driver, WINDOW, "left");
      const baseline = store.writes.length;
      await editLabel(driver, "one");
      await editLabel(driver, "two");
      const windowWrites = store.writes
        .slice(baseline)
        .flat()
        .filter(({ key }) => key.startsWith("window."));
      expect(windowWrites).toHaveLength(0);
    });

    it("should leave the project ring alone when only a window's entry changes", async () => {
      const store = new CountingKV();
      const driver = await createDriver(store);
      await enter(driver, CTX);
      // The first view edit registers the window, which is project bookkeeping.
      await editView(driver, WINDOW, "one");
      const baseline = store.writes.length;
      await editView(driver, WINDOW, "two");
      await editView(driver, WINDOW, "three");
      const projectWrites = store.writes
        .slice(baseline)
        .flat()
        .filter(({ key }) => key.startsWith("project."));
      expect(projectWrites).toHaveLength(0);
    });

    it("should scope shared fields to the project they were written under", async () => {
      const store = new MemoryKV();
      const driver = await createDriver(store);
      await enter(driver, CTX);
      await editLabel(driver, "p1-tag");
      await enter(driver, { core: "c1", project: "p2" });
      expect(driver.getState().view.label).toBe("none");
      await editLabel(driver, "p2-tag");
      await enter(driver, CTX);
      expect(driver.getState().view.label).toBe("p1-tag");
    });
  });

  describe("secondary windows", () => {
    it("should neither compose nor persist state outside the main window", async () => {
      mocks.engine = "tauri";
      mocks.label = "child-window";
      const store = new MemoryKV();
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
      const store = new MemoryKV();
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
    const createPersisted = async (store: MemoryKV) => {
      const driver = await createDriver(store);
      await enter(driver, CTX);
      await edit(driver, "16.2.0");
    };

    it("should drop fields the slice's schema does not declare", async () => {
      const store = new MemoryKV();
      await createPersisted(store);
      const { initialState } = await openPersist(store, {
        scopes: {
          ...SCOPES,
          project: {
            work: z.object({ value: z.string() }),
            windows: z.string().array(),
          },
        },
      });
      expect(initialState?.work).toEqual({ value: "16.2.0" });
    });

    it("should fall back to the slice's initial state when the stored bytes fail its schema", async () => {
      const store = new MemoryKV();
      await createPersisted(store);
      const onError = vi.fn<(error: Error) => void>();
      const { initialState } = await openPersist(store, {
        onError,
        scopes: {
          ...SCOPES,
          project: { work: workZ.refine(() => false), windows: z.string().array() },
        },
      });
      expect(initialState?.work).toEqual(ZERO_MOCK_STATE.work);
      expect(initialState?.core).toEqual(STATE.core);
      expect(onError).toHaveBeenCalled();
    });

    it("should leave the other slices of a partition alone when one fails", async () => {
      const store = new MemoryKV();
      await createPersisted(store);
      const onError = vi.fn<(error: Error) => void>();
      const { initialState } = await openPersist(store, {
        onError,
        scopes: {
          ...SCOPES,
          global: { core: selectedZ.refine(() => false) },
        },
      });
      expect(initialState?.core).toEqual(ZERO_MOCK_STATE.core);
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
        deleteMany: fail,
        length: fail,
        keys: fail,
        clear: fail,
      };
    };

    it("should still compose the initial state", async () => {
      const onError = vi.fn<(error: Error) => void>();
      const { initialState, middleware } = await Persist.open<MockState>({
        onError,
        initial: ZERO_MOCK_STATE,
        scopes: SCOPES,
        getContext,
        openKV: broken,
      });
      expect(initialState).toEqual(ZERO_MOCK_STATE);
      expect(middleware).toBeDefined();
    });

    it("should announce that the store is unavailable", async () => {
      const onError = vi.fn<(error: Error) => void>();
      const { middleware } = await Persist.open<MockState>({
        onError,
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
    });

    it("should announce it only once", async () => {
      const onError = vi.fn<(error: Error) => void>();
      const { middleware } = await Persist.open<MockState>({
        onError,
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
    });

    it("should try a failed save once instead of retrying it", async () => {
      const onError = vi.fn<(error: Error) => void>();
      const { middleware } = await Persist.open<MockState>({
        onError,
        initial: ZERO_MOCK_STATE,
        scopes: SCOPES,
        getContext,
        openKV: broken,
        debounceInterval: TimeSpan.milliseconds(10),
      });
      // Dispatches from the middleware go back through it, as they do in redux.
      const chain: (action: unknown) => unknown = middleware({
        getState: () => ZERO_MOCK_STATE,
        dispatch: (action: unknown) => chain(action),
      } as never)((a) => a);
      const failedSaves = () =>
        onError.mock.calls.filter(([e]) => e.message === "failed to persist state");
      chain({ type: "work/edit" });
      await vi.waitFor(() => expect(failedSaves()).toHaveLength(1));
      await sleep.sleep(TimeSpan.milliseconds(100));
      expect(failedSaves()).toHaveLength(1);
    });
  });

  describe("scope coverage", () => {
    it("should throw when a slice is in no scope and not transient", async () => {
      await expect(
        openPersist(new MemoryKV(), {
          scopes: { ...SCOPES, project: {} },
        }),
      ).rejects.toThrow("work");
    });

    it("should throw when a slice is declared in two scopes", async () => {
      await expect(
        openPersist(new MemoryKV(), {
          scopes: { ...SCOPES, global: { core: selectedZ, work: workZ } },
        }),
      ).rejects.toThrow("more than one scope");
    });

    it("should accept a slice declared transient instead of scoped", async () => {
      await expect(
        openPersist(new MemoryKV(), {
          scopes: { ...SCOPES, project: {}, transient: ["work", "windows"] },
        }),
      ).resolves.toBeDefined();
    });

    // A window partition holds only the windows record, so a field without a default
    // would make every window partition unreadable on the next launch.
    it("should throw when a window-scoped schema rejects a bare windows record", async () => {
      await expect(
        openPersist(new MemoryKV(), {
          scopes: {
            ...SCOPES,
            window: { view: viewZ.extend({ label: z.string() }) },
          },
        }),
      ).rejects.toThrow("reject a bare windows record: view");
    });
  });
});

describe("Persist.middleware", () => {
  beforeEach(() => {
    mocks.engine = "web";
    mocks.label = "main";
  });

  it("should pass the action through to next and return its result", async () => {
    const driver = await createDriver(new MemoryKV());
    const action = { type: "any/action" };
    expect(driver.dispatch(action)).toBe(action);
    expect(driver.next).toHaveBeenCalledWith(action);
  });

  it("should persist the current store state under the current context", async () => {
    const store = new MemoryKV();
    const driver = await createDriver(store);
    await enter(driver, CTX);
    await edit(driver, "edited");
  });

  it("should delete a Core's partitions on a purge action", async () => {
    const store = new MemoryKV();
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

  it("should delete a purged Core's keys in one batch", async () => {
    const store = new CountingKV();
    const driver = await createDriver(store);
    await enter(driver, CTX);
    await edit(driver, "16.2.0");
    driver.dispatch(Persist.purge("c1"));
    await vi.waitFor(() => expect(store.deletes).toHaveLength(1));
    expect(store.deletes[0].length).toBeGreaterThan(1);
  });

  it("should purge behind the swap that leaves the purged Core", async () => {
    const store = new MemoryKV();
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
    const onError = vi.fn<(error: Error) => void>();
    const store = new FailingKV();
    const driver = await createDriver(store, { onError });
    await enter(driver, CTX);
    await edit(driver, "16.2.0");
    store.failOn = /^core\.c1\./;
    driver.dispatch(Persist.purge("c1"));
    await vi.waitFor(() =>
      expect(onError).toHaveBeenCalledWith(
        expect.objectContaining({ message: "failed to purge stored Core state" }),
      ),
    );
  });

  it("should announce the store unavailable when a mid-session write fails", async () => {
    const onError = vi.fn<(error: Error) => void>();
    const store = new FailingKV();
    const driver = await createDriver(store, { onError });
    await enter(driver, CTX);
    store.failOn = /./;
    driver.dispatch(
      { type: "work/edit" },
      { ...driver.getState(), work: { value: "doomed" } },
    );
    await vi.waitFor(() =>
      expect(driver.dispatched).toHaveBeenCalledWith(Persist.storeUnavailable()),
    );
  });

  // The revert and clear branches reload the window, which jsdom cannot perform; the
  // production code swallows that failure via .catch, so we suppress the expected log.
  it("should revert the innermost partition one version on a revertState action", async () => {
    const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const store = new MemoryKV();
    const driver = await createDriver(store);
    await enter(driver, CTX);
    await edit(driver, "16.2.0");
    await edit(driver, "16.2.1");
    driver.dispatch(Persist.revertState());
    await driver.flushed(workIs("16.2.0"));
    errorSpy.mockRestore();
  });

  it("should report a failed revert instead of reloading", async () => {
    const onError = vi.fn<(error: Error) => void>();
    const store = new FailingKV();
    const driver = await createDriver(store, { onError });
    await enter(driver, CTX);
    await edit(driver, "16.2.0");
    await edit(driver, "16.2.1");
    store.failOn = /\.slot$/;
    driver.dispatch(Persist.revertState());
    await vi.waitFor(() =>
      expect(onError).toHaveBeenCalledWith(
        expect.objectContaining({ message: "failed to revert state" }),
      ),
    );
    expect((await driver.composed())?.work.value).toBe("16.2.1");
  });

  it("should fall back to the initial state when reverting past the first version", async () => {
    const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const store = new MemoryKV();
    const driver = await createDriver(store);
    await enter(driver, CTX);
    await edit(driver, "16.2.0");
    driver.dispatch(Persist.revertState());
    await driver.flushed(workIs(ZERO_MOCK_STATE.work.value));
    errorSpy.mockRestore();
  });

  it("should revert the window partition before the project partition", async () => {
    const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const store = new MemoryKV();
    const driver = await createDriver(store);
    await enter(driver, CTX);
    await edit(driver, "16.2.0");
    await editView(driver, WINDOW, "one");
    await edit(driver, "16.2.1");
    await editView(driver, WINDOW, "two");
    driver.dispatch(Persist.revertState());
    // Only the innermost partition steps back: the window's view reverts while the
    // project's work stays put.
    await driver.flushed((state) => state?.view.windows[WINDOW]?.value === "one");
    expect((await driver.composed())?.work.value).toBe("16.2.1");
    errorSpy.mockRestore();
  });

  it("should clear the entire store on a clearState action", async () => {
    const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const store = new MemoryKV();
    const driver = await createDriver(store);
    await enter(driver, CTX);
    await edit(driver, "16.2.0");
    driver.dispatch(Persist.clearState());
    await vi.waitFor(async () => expect(await store.length()).toBe(0));
    errorSpy.mockRestore();
  });

  it("should load the target context and dispatch hydrate on a context switch", async () => {
    const driver = await createDriver(new MemoryKV());
    await enter(driver, CTX);
    await edit(driver, "c1-work");
    await enter(driver, { core: "c2", project: "p2" });
    await enter(driver, { core: "c1" });
    // The Core's partition names the project whose partition holds the work.
    expect(driver.dispatched).toHaveBeenLastCalledWith(
      Persist.hydrate({
        project: { selected: "p1" },
        work: { value: "c1-work", transient: "drag" },
        windows: [],
        view: ZERO_MOCK_STATE.view,
      }),
    );
  });

  it("should end the swap when loading the target context fails", async () => {
    const onError = vi.fn<(error: Error) => void>();
    const store = new MemoryKV();
    const failing: Persist.SugaredKV = {
      get: async <V>(key: string): Promise<V | null> => {
        if (key.startsWith("core.c2.")) throw new Error("read failed");
        return await store.get<V>(key);
      },
      set: async (key, value) => await store.set(key, value),
      setMany: async (entries) => await store.setMany(entries),
      delete: async (key) => await store.delete(key),
      deleteMany: async (keys) => await store.deleteMany(keys),
      length: async () => await store.length(),
      keys: async () => await store.keys(),
      clear: async () => await store.clear(),
    };
    const driver = await createDriver(store, { onError, openKV: () => failing });
    await enter(driver, CTX);
    driver.dispatch(
      { type: "Core/select" },
      { ...driver.getState(), core: { selected: "c2" }, project: {} },
    );
    await vi.waitFor(() =>
      expect(driver.dispatched).toHaveBeenCalledWith(Persist.endSwap()),
    );
  });

  describe("after a failed swap", () => {
    const P2_SWAP = { core: "c1", project: "p2" };
    const C2_SWAP = { core: "c2", project: "p1" };

    /**
     * Leaves p1 loaded with p1-work, then swaps to the target, failing the swap's first
     * read of a key matching failOn.
     */
    const failSwap = async (
      target: Persist.Context,
      failOn: RegExp,
      scopes: Persist.Scopes<MockState> = SCOPES,
    ) => {
      const store = new ReadFailingKV();
      const driver = await createDriver(store, {
        onError: vi.fn(),
        debounceInterval: TimeSpan.hours(1),
        scopes,
      });
      await enter(driver, P2_SWAP);
      await commit(driver, "p2-work");
      await enter(driver, CTX);
      await commit(driver, "p1-work");
      store.failOn = failOn;
      driver.dispatch(
        { type: "context/select" },
        {
          ...driver.getState(),
          core: { selected: target.core },
          project: { selected: target.project },
        },
      );
      await vi.waitFor(() =>
        expect(driver.dispatched).toHaveBeenCalledWith(Persist.endSwap()),
      );
      return { store, driver, written: store.writes.length };
    };

    /** Lists the work values written under the key pattern, in write order. */
    const worksAt = (store: CountingKV, pattern: RegExp): Array<string | undefined> =>
      store.writes
        .flat()
        .filter(({ key }) => pattern.test(key))
        .map(
          ({ value }) => z.object({ work: workZ }).safeParse(value).data?.work.value,
        );

    it("should save no project slice until a later swap hydrates", async () => {
      const { store, driver, written } = await failSwap(P2_SWAP, P2_STATE);
      await commit(driver, "late");
      expect(store.writes.length).toBe(written);
    });

    it("should save a global edit but not a project edit", async () => {
      const scopes: Persist.Scopes<MockState> = {
        global: { core: selectedZ, work: workZ },
        core: { project: selectedZ },
        project: { view: viewZ, windows: z.string().array() },
        window: {},
        transient: [],
      };
      const { store, driver, written } = await failSwap(P2_SWAP, P2_STATE, scopes);
      driver.dispatch(
        { type: "edit" },
        {
          ...driver.getState(),
          work: { value: "global-edit" },
          view: { label: "project-edit", windows: {} },
        },
      );
      await Persist.flush((action) => driver.dispatch(action));
      const entries = store.writes.slice(written).flat();
      expect(entries.map(({ key }) => key)).toEqual(
        expect.arrayContaining([expect.stringMatching(/^global\./)]),
      );
      entries.forEach(({ key }) => expect(key).toMatch(/^global\./));
      store.failOn = null;
      const { initialState } = await openPersist(store, { scopes });
      expect(initialState?.work.value).toBe("global-edit");
    });

    it("should save again once a later swap hydrates", async () => {
      const { store, driver } = await failSwap(P2_SWAP, P2_STATE);
      store.failOn = null;
      await enter(driver, { core: "c1", project: "p3" });
      await commit(driver, "p3-work");
      expect(works(store, "p3")).toContain("p3-work");
    });

    it("should reload the context the store held on a return to it", async () => {
      const { store, driver } = await failSwap(P2_SWAP, P2_STATE);
      store.failOn = null;
      await commit(driver, "late");
      await enter(driver, CTX);
      expect(driver.getState().work.value).toBe("p1-work");
      await Persist.flush((action) => driver.dispatch(action));
      expect(worksAt(store, /^project\.c1\.p\d\.\d+$/)).not.toContain("late");
    });

    it("should reload the old Core on a return after a failed Core swap", async () => {
      const scopes: Persist.Scopes<MockState> = {
        ...SCOPES,
        core: { project: selectedZ, work: workZ },
        project: { windows: z.string().array() },
      };
      const { store, driver } = await failSwap(C2_SWAP, /^core\.c2\./, scopes);
      // The target Core's synchronizers change the store after the failure.
      await commit(driver, "c2-sync");
      store.failOn = null;
      driver.dispatch(
        { type: "Core/select" },
        { ...driver.getState(), core: { selected: "c1" } },
      );
      await driver.settle();
      expect(driver.getState().work.value).toBe("p1-work");
      await Persist.flush((action) => driver.dispatch(action));
      expect(worksAt(store, /^core\.c\d\.\d+$/)).not.toContain("c2-sync");
    });

    it("should keep a project picked in the target Core after a failed Core swap", async () => {
      const { store, driver } = await failSwap(C2_SWAP, /^core\.c2\./);
      store.failOn = null;
      driver.dispatch(
        { type: "project/select" },
        { ...driver.getState(), project: { selected: "p3" } },
      );
      await driver.settle();
      expect(driver.getState().project.selected).toBe("p3");
      await Persist.flush((action) => driver.dispatch(action));
      const c1Projects = store.writes
        .flat()
        .filter(({ key }) => /^core\.c1\.\d+$/.test(key))
        .map(
          ({ value }) => z.object({ project: selectedZ }).parse(value).project.selected,
        );
      expect(c1Projects).not.toContain("p3");
    });

    it("should reload the old Core after a retry of a failed Core swap fails", async () => {
      const { store, driver } = await failSwap(C2_SWAP, /^core\.c2\./);
      // The project remover clears a project the new Core does not have.
      driver.dispatch({ type: "project/clear" }, { ...driver.getState(), project: {} });
      await vi.waitFor(() =>
        expect(
          driver.dispatched.mock.calls.filter(([a]) => Persist.endSwap.match(a)),
        ).toHaveLength(2),
      );
      store.failOn = null;
      driver.dispatch(
        { type: "Core/select" },
        { ...driver.getState(), core: { selected: "c1" } },
      );
      await driver.settle();
      expect(driver.getState().project.selected).toBe("p1");
      expect(driver.getState().work.value).toBe("p1-work");
      await Persist.flush((action) => driver.dispatch(action));
      const { initialState } = await openPersist(store);
      expect(initialState?.project.selected).toBe("p1");
    });
  });

  it("should mark the swap window with beginSwap before hydrate", async () => {
    const driver = await createDriver(new MemoryKV());
    await enter(driver, CTX);
    const calls = driver.dispatched.mock.calls.map(
      ([action]) => (action as { type: string }).type,
    );
    const begin = calls.indexOf(Persist.beginSwap.type);
    const hydrated = calls.indexOf(Persist.hydrate.type);
    expect(begin).toBeGreaterThanOrEqual(0);
    expect(hydrated).toBeGreaterThan(begin);
  });

  // A write and a purge used to run on separate chains, so a write in flight could
  // put back the keys the purge had just deleted.
  it("should not write a purged Core's partition back", async () => {
    const store = new MemoryKV();
    const driver = await createDriver(store);
    await enter(driver, CTX);
    await edit(driver, "doomed");
    driver.dispatch(
      { type: "work/edit" },
      { ...driver.getState(), work: { value: "racing", transient: "drag" } },
    );
    driver.dispatch(Persist.purge("c1"));
    await vi.waitFor(async () =>
      expect((await store.keys()).filter((key) => key.includes("c1"))).toHaveLength(0),
    );
  });

  it("should swap only the project partition when the Core is unchanged", async () => {
    const driver = await createDriver(new MemoryKV());
    await enter(driver, CTX);
    await edit(driver, "p1-work");
    await enter(driver, { core: "c1", project: "p2" });
    await enter(driver, CTX);
    // The Core-scoped project slice stayed put; only the project's work swapped.
    expect(driver.dispatched).toHaveBeenLastCalledWith(
      Persist.hydrate({
        work: { value: "p1-work", transient: "drag" },
        windows: [],
        view: ZERO_MOCK_STATE.view,
      }),
    );
  });

  it("should hydrate zero slices for a never-visited context", async () => {
    const driver = await createDriver(new MemoryKV());
    await enter(driver, { core: "fresh" });
    expect(driver.dispatched).toHaveBeenCalledWith(
      Persist.hydrate({
        project: ZERO_MOCK_STATE.project,
        work: ZERO_MOCK_STATE.work,
        windows: ZERO_MOCK_STATE.windows,
        view: ZERO_MOCK_STATE.view,
      }),
    );
  });

  it("should adopt the hydrated context instead of re-swapping on hydrate", async () => {
    const store = new MemoryKV();
    const driver = await createDriver(store);
    await enter(driver, CTX);
    const swaps = driver.dispatched.mock.calls.length;
    driver.dispatch(Persist.hydrate({ work: STATE.work }), STATE);
    await driver.flushed(workIs("16.2.0"));
    expect(driver.dispatched.mock.calls.length).toBe(swaps);
  });

  it("should discard a stale swap when a newer context switch supersedes it", async () => {
    const store = new MemoryKV();
    const written = await createDriver(store);
    await enter(written, CTX);
    await edit(written, "fresh");
    const { promise: staleGate, resolve: releaseStale } = Promise.withResolvers<void>();
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
      deleteMany: async (keys) => await store.deleteMany(keys),
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
    releaseStale();
    // The released read is the stale swap's last async hop, so once it lands the
    // swap has fully resolved. Its hydrate would clobber the newer Core's slices.
    await vi.waitFor(() => expect(gateDone).toHaveBeenCalled());
    await driver.settle();
    const hydrates = (): number =>
      driver.dispatched.mock.calls.filter(
        ([action]) => (action as { type: string }).type === Persist.hydrate.type,
      ).length;
    expect(hydrates()).toBe(1);
  });

  describe("while a swap is in flight", () => {
    it("should not save an edit under the context the swap moves to", async () => {
      const store = new CountingKV();
      const p2 = new GatedKV(store, P2_STATE);
      const driver = await createDriver(store, { openKV: () => p2 });
      await swapToHeldP2(driver, p2);
      driver.dispatch(
        { type: "work/edit" },
        { ...driver.getState(), work: { value: "p1-late", transient: "drag" } },
      );
      p2.release();
      await driver.settle();
      await Persist.flush((action) => driver.dispatch(action));
      expect(works(store, "p2")).not.toContain("p1-late");
      expect(driver.getState().work.value).toBe("p2-work");
    });

    it("should drop a save still waiting on the debounce when the swap starts", async () => {
      const store = new CountingKV();
      const p2 = new GatedKV(store, P2_STATE);
      const driver = await createDriver(store, {
        openKV: () => p2,
        debounceInterval: TimeSpan.milliseconds(250),
      });
      await enter(driver, { core: "c1", project: "p2" });
      await commit(driver, "p2-work");
      await enter(driver, CTX);
      await commit(driver, "p1-work");
      vi.useFakeTimers();
      try {
        driver.dispatch(
          { type: "work/edit" },
          { ...driver.getState(), work: { value: "p1-late", transient: "drag" } },
        );
        p2.held = true;
        driver.dispatch(
          { type: "project/select" },
          { ...driver.getState(), project: { selected: "p2" } },
        );
        await vi.advanceTimersByTimeAsync(250);
      } finally {
        vi.useRealTimers();
      }
      await vi.waitFor(() => expect(p2.hit).toHaveBeenCalled());
      p2.release();
      await driver.settle();
      await Persist.flush((action) => driver.dispatch(action));
      expect(works(store, "p1")).toContain("p1-late");
      expect(works(store, "p2")).not.toContain("p1-late");
    });

    it("should resolve a flush without saving under the swap's target", async () => {
      const store = new CountingKV();
      const p2 = new GatedKV(store, P2_STATE);
      const driver = await createDriver(store, {
        openKV: () => p2,
        debounceInterval: TimeSpan.hours(1),
      });
      await swapToHeldP2(driver, p2);
      driver.dispatch(
        { type: "work/edit" },
        { ...driver.getState(), work: { value: "p1-late", transient: "drag" } },
      );
      const flush = Persist.flush((action) => driver.dispatch(action));
      p2.release();
      await flush;
      await driver.settle();
      await Persist.flush((action) => driver.dispatch(action));
      expect(works(store, "p2")).not.toContain("p1-late");
    });

    it("should reload without stepping back the swap target's ring on a revert", async () => {
      const store = new CountingKV();
      const p2 = new GatedKV(store, P2_STATE);
      const reload = vi.fn<() => void>();
      const driver = await createDriver(store, { openKV: () => p2, reload });
      await swapToHeldP2(driver, p2);
      const written = store.writes.length;
      driver.dispatch(Persist.revertState());
      p2.release();
      await driver.settle();
      await vi.waitFor(() => expect(reload).toHaveBeenCalled());
      const pointers = store.writes
        .slice(written)
        .flat()
        .filter(({ key }) => key === "project.c1.p2.slot");
      expect(pointers).toEqual([]);
    });

    it("should not flush the store's slices under the target a newer swap supersedes", async () => {
      const store = new CountingKV();
      const p2 = new GatedKV(store, P2_STATE);
      const driver = await createDriver(store, { openKV: () => p2 });
      await swapToHeldP2(driver, p2);
      driver.dispatch(
        { type: "project/select" },
        { ...driver.getState(), project: { selected: "p3" } },
      );
      p2.release();
      await driver.settle();
      await Persist.flush((action) => driver.dispatch(action));
      expect(works(store, "p2")).not.toContain("p1-work");
      await enter(driver, { core: "c1", project: "p2" });
      expect(driver.getState().work.value).toBe("p2-work");
    });

    it("should load the new Core's slices when a newer swap supersedes a Core swap", async () => {
      const store = new MemoryKV();
      const c2 = new GatedKV(store, /^core\.c2\./);
      const scopes: Persist.Scopes<MockState> = {
        ...SCOPES,
        core: { project: selectedZ, work: workZ },
        project: { windows: z.string().array() },
      };
      const driver = await createDriver(store, { openKV: () => c2, scopes });
      await enter(driver, { core: "c2", project: "p9" });
      await commit(driver, "c2-work");
      await enter(driver, CTX);
      await commit(driver, "c1-work");
      c2.held = true;
      driver.dispatch(
        { type: "Core/select" },
        { ...driver.getState(), core: { selected: "c2" } },
      );
      await vi.waitFor(() => expect(c2.hit).toHaveBeenCalled());
      // The project remover clears a project the new Core does not have.
      driver.dispatch({ type: "project/clear" }, { ...driver.getState(), project: {} });
      c2.release();
      await driver.settle();
      expect(driver.getState().project.selected).toBe("p9");
      expect(driver.getState().work.value).toBe("c2-work");
      await Persist.flush((action) => driver.dispatch(action));
      const { initialState } = await openPersist(store, { scopes });
      expect(initialState?.work.value).toBe("c2-work");
    });

    it("should keep a project picked while a Core swap loads", async () => {
      const store = new MemoryKV();
      const c2 = new GatedKV(store, /^core\.c2\./);
      const scopes: Persist.Scopes<MockState> = {
        ...SCOPES,
        core: { project: selectedZ, work: workZ },
        project: { windows: z.string().array() },
      };
      const driver = await createDriver(store, { openKV: () => c2, scopes });
      await enter(driver, { core: "c2", project: "p9" });
      await commit(driver, "c2-work");
      await enter(driver, CTX);
      await commit(driver, "c1-work");
      c2.held = true;
      driver.dispatch(
        { type: "Core/select" },
        { ...driver.getState(), core: { selected: "c2" } },
      );
      await vi.waitFor(() => expect(c2.hit).toHaveBeenCalled());
      driver.dispatch(
        { type: "project/select" },
        { ...driver.getState(), project: { selected: "p3" } },
      );
      c2.release();
      await driver.settle();
      expect(driver.getState().project.selected).toBe("p3");
      expect(driver.getState().work.value).toBe("c2-work");
      await Persist.flush((action) => driver.dispatch(action));
      const { initialState } = await openPersist(store, { scopes });
      expect(initialState?.project.selected).toBe("p3");
      expect(initialState?.work.value).toBe("c2-work");
    });
  });

  it("should queue a swap's flush behind a write already in flight", async () => {
    const store = new MemoryKV();
    let armed = false;
    let inFlight = 0;
    let maxInFlight = 0;
    const gateHit = vi.fn();
    const { promise: gate, resolve: release } = Promise.withResolvers<void>();
    const gated: Persist.SugaredKV = {
      get: async <V>(key: string): Promise<V | null> => await store.get<V>(key),
      set: async (key, value) => await store.set(key, value),
      setMany: async (entries) => {
        if (!armed) return await store.setMany(entries);
        inFlight++;
        maxInFlight = Math.max(maxInFlight, inFlight);
        gateHit();
        await gate;
        await store.setMany(entries);
        inFlight--;
      },
      delete: async (key) => await store.delete(key),
      deleteMany: async (keys) => await store.deleteMany(keys),
      length: async () => await store.length(),
      keys: async () => await store.keys(),
      clear: async () => await store.clear(),
    };
    const driver = await createDriver(store, { openKV: () => gated });
    await enter(driver, CTX);
    armed = true;
    driver.dispatch(
      { type: "work/edit" },
      { ...driver.getState(), work: { value: "mid-flight" } },
    );
    await vi.waitFor(() => expect(gateHit).toHaveBeenCalled());
    driver.dispatch(
      { type: "Core/select" },
      { ...driver.getState(), core: { selected: "c2" }, project: {} },
    );
    // A concurrent flush would reach the store within this window; a queued one
    // cannot move until the gate opens.
    await sleep.sleep(TimeSpan.milliseconds(20));
    release();
    await driver.settle();
    expect(maxInFlight).toBe(1);
  });

  it("should write a change still waiting on the debounce when flushed", async () => {
    const store = new MemoryKV();
    const driver = await createDriver(store, { debounceInterval: TimeSpan.hours(1) });
    driver.dispatch({ type: "a" }, ZERO_MOCK_STATE);
    await expect(store.length()).resolves.toBe(0);
    await Persist.flush((action) => driver.dispatch(action));
    expect(await store.length()).toBe(GLOBAL_KEYS);
  });

  it("should reject a flush whose write fails", async () => {
    const store = new FailingKV();
    const onError = vi.fn<(error: Error) => void>();
    const driver = await createDriver(store, {
      onError,
      debounceInterval: TimeSpan.hours(1),
    });
    store.failOn = /./;
    driver.dispatch({ type: "a" }, ZERO_MOCK_STATE);
    const flush = Persist.flush((action) => driver.dispatch(action));
    await expect(flush).rejects.toThrow("failed to write the session state");
    expect(onError).toHaveBeenCalledWith(new Error("failed to persist state"));
  });

  it("should coalesce rapid dispatches into a single persist when debounced", async () => {
    const store = new MemoryKV();
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
    const db = Persist.openSugaredKV("session");
    await db.set("global.slot", { slot: 1 });
    Persist.hardClearAndReload();
    await vi.waitFor(async () => {
      expect(await db.get("global.slot")).toBeNull();
    });
  });
});
