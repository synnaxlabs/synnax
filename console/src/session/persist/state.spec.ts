// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { kv, TimeSpan } from "@synnaxlabs/x";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Persist } from "@/session/persist";
import { type SugaredKV } from "@/session/persist/kv";

interface MockState {
  mock: { value: string };
}

const ZERO_MOCK_STATE: MockState = {
  mock: { value: "0.0.0" },
};

// A minimal engine double matching the shape Persist.middleware consumes.
const makeEngine = () => ({
  persist: vi.fn().mockResolvedValue(undefined),
  revert: vi.fn().mockResolvedValue(undefined),
  clear: vi.fn().mockResolvedValue(undefined),
  initialState: undefined as MockState | undefined,
});

// Drives an action straight through the middleware chain, capturing what next saw.
const drive = (
  engine: ReturnType<typeof makeEngine>,
  state: MockState,
  action: { type: string },
  debounce = TimeSpan.ZERO,
) => {
  const next = vi.fn((a: unknown) => a);
  const store = { getState: () => state, dispatch: vi.fn() };
  const result = Persist.middleware<MockState>(engine, debounce)(store as never)(next)(
    action,
  );
  return { next, result };
};

describe("Persist", () => {
  describe("migration from v1 to v2", () => {
    it("should open a blank v2 store if the v1 store is empty", async () => {
      const v1Store = new kv.MockAsync();
      const v2Store = new kv.MockAsync();
      const openKV = (path: string): SugaredKV => {
        if (path === Persist.V1_STORE_PATH) return v1Store;
        if (path === Persist.V2_STORE_PATH) return v2Store;
        throw new Error(`Unknown path: ${path}`);
      };
      const engine = await Persist.open({ openKV, initial: ZERO_MOCK_STATE });
      await expect(v1Store.length()).resolves.toBe(0);
      await expect(v2Store.length()).resolves.toBe(1);
      expect(engine.initialState).toEqual(ZERO_MOCK_STATE);
    });
    it("should migrate a v1 store to a v2 store", async () => {
      const v1Store = new kv.MockAsync();
      const v2Store = new kv.MockAsync();
      const version = 12;
      const persistedStateKey = Persist.persistedStateKey(version);
      await v1Store.set(Persist.DB_VERSION_KEY, { version });
      const persistedState = {
        mock: {
          ...ZERO_MOCK_STATE.mock,
          value: "16.2.0",
        },
      };
      await v1Store.set(persistedStateKey, persistedState);
      const openKV = (path: string): SugaredKV => {
        if (path === Persist.V1_STORE_PATH) return v1Store;
        if (path === Persist.V2_STORE_PATH) return v2Store;
        throw new Error(`Unknown path: ${path}`);
      };
      await expect(v1Store.length()).resolves.toBe(2);
      const engine = await Persist.open({ openKV, initial: ZERO_MOCK_STATE });
      // Expect the v1 store to be cleared out.
      await expect(v1Store.length()).resolves.toBe(0);
      await expect(v2Store.length()).resolves.toBe(2);
      expect(engine.initialState).toEqual({
        mock: {
          ...ZERO_MOCK_STATE.mock,
          value: "16.2.0",
        },
      });
      const expectedPersistedStateKey = Persist.persistedStateKey(1);
      await expect(v2Store.get(expectedPersistedStateKey)).resolves.toEqual(
        persistedState,
      );
      await expect(v2Store.get(Persist.DB_VERSION_KEY)).resolves.toEqual({
        version: 1,
      });
    });
  });

  describe("engine.persist", () => {
    it("should correctly persist state", async () => {
      const store = new kv.MockAsync();
      const engine = await Persist.open({
        openKV: () => store,
        initial: ZERO_MOCK_STATE,
      });
      const state = {
        mock: {
          ...ZERO_MOCK_STATE.mock,
          value: "16.2.0",
        },
      };
      await engine.persist(state);
      await expect(store.get(Persist.DB_VERSION_KEY)).resolves.toEqual({
        version: 1,
      });
      await expect(store.get(Persist.persistedStateKey(1))).resolves.toEqual(state);
    });
    it("should maintain a maximum history of 4", async () => {
      const store = new kv.MockAsync();
      const openKV = () => store;
      const engine = await Persist.open({ openKV, initial: ZERO_MOCK_STATE });
      for (let i = 0; i < 10; i++)
        await engine.persist({
          mock: {
            ...ZERO_MOCK_STATE.mock,
            value: `16.2.${i}`,
          },
        });

      await expect(store.length()).resolves.toBe(5);
      // Open the engine again to make sure the initial state is correct
      const engine2 = await Persist.open({ openKV, initial: ZERO_MOCK_STATE });
      expect(engine2.initialState).toEqual({
        mock: {
          ...ZERO_MOCK_STATE.mock,
          value: "16.2.9",
        },
      });
    });
    it("should correctly revert state", async () => {
      const store = new kv.MockAsync();
      const engine = await Persist.open({
        openKV: () => store,
        initial: ZERO_MOCK_STATE,
      });
      const state = {
        mock: {
          ...ZERO_MOCK_STATE.mock,
          value: "16.2.0",
        },
      };
      await engine.persist(state);
      await engine.revert();
      const engine2 = await Persist.open({
        openKV: () => store,
        initial: ZERO_MOCK_STATE,
      });
      expect(engine2.initialState).toEqual(ZERO_MOCK_STATE);
    });
    it("should correctly revert the state when it has multiple versions", async () => {
      const store = new kv.MockAsync();
      const openKV = () => store;
      const engine = await Persist.open({ openKV, initial: ZERO_MOCK_STATE });
      const state = {
        mock: {
          ...ZERO_MOCK_STATE.mock,
          value: "16.2.0",
        },
      };
      await engine.persist(state);
      const state2 = {
        mock: {
          ...ZERO_MOCK_STATE.mock,
          value: "16.2.1",
        },
      };
      await engine.persist(state2);
      await engine.revert();
      const engine2 = await Persist.open({ openKV, initial: ZERO_MOCK_STATE });
      expect(engine2.initialState).toEqual(state);
    });
  });
  describe("engine.clear", () => {
    it("should correctly clear the store", async () => {
      const store = new kv.MockAsync();
      const openKV = () => store;
      const engine = await Persist.open({ openKV, initial: ZERO_MOCK_STATE });
      await engine.persist({
        mock: {
          ...ZERO_MOCK_STATE.mock,
          value: "16.2.0",
        },
      });
      await engine.clear();
      await expect(store.length()).resolves.toBe(1);
      await expect(store.get(Persist.DB_VERSION_KEY)).resolves.toEqual({
        version: 0,
      });
      await expect(store.get(Persist.persistedStateKey(1))).resolves.toBeNull();
      const engine2 = await Persist.open({ openKV, initial: ZERO_MOCK_STATE });
      expect(engine2.initialState).toEqual(ZERO_MOCK_STATE);
    });
  });

  describe("exclude", () => {
    it("should correctly exclude keys from the state", async () => {
      type State = MockState & { a: number; b: number; c: number };
      const state: State = {
        mock: {
          ...ZERO_MOCK_STATE.mock,
        },
        a: 1,
        b: 2,
        c: 3,
      };
      const store = new kv.MockAsync();
      const v = await Persist.open<State>({
        exclude: ["a"],
        initial: state,
        openKV: () => store,
      });
      await v.persist(state);
      await expect(store.get(Persist.persistedStateKey(1))).resolves.toEqual({
        mock: {
          ...ZERO_MOCK_STATE.mock,
        },
        b: 2,
        c: 3,
      });
    });
  });

  describe("migrator", () => {
    it("should correctly apply a migration function to the state", async () => {
      type State = { a: number; b: number; c: number } & MockState;
      const state: State = {
        mock: {
          ...ZERO_MOCK_STATE.mock,
        },
        a: 1,
        b: 2,
        c: 3,
      };
      const store = new kv.MockAsync();
      const engine1 = await Persist.open({ initial: state, openKV: () => store });
      await engine1.persist(state);
      const migrator = (state: State) => ({ ...state, c: 5 });
      await Persist.open({ initial: state, migrator, openKV: () => store });
      await expect(store.length()).resolves.toBe(3);
      await expect(store.get(Persist.persistedStateKey(2))).resolves.toEqual({
        ...state,
        c: 5,
      });
    });

    it("should fall back to the initial state when the migrator throws", async () => {
      const store = new kv.MockAsync();
      const engine1 = await Persist.open({
        initial: ZERO_MOCK_STATE,
        openKV: () => store,
      });
      await engine1.persist({ mock: { value: "1.2.3" } });
      const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
      const migrator = () => {
        throw new Error("migration failed");
      };
      const engine2 = await Persist.open({
        initial: ZERO_MOCK_STATE,
        migrator,
        openKV: () => store,
      });
      expect(engine2.initialState).toEqual(ZERO_MOCK_STATE);
      expect(errorSpy).toHaveBeenCalled();
      errorSpy.mockRestore();
    });
  });

  describe("exclude function", () => {
    it("should apply a function exclude to transform the persisted state", async () => {
      type State = MockState & { secret: string };
      const state: State = { mock: { ...ZERO_MOCK_STATE.mock }, secret: "hunter2" };
      const store = new kv.MockAsync();
      const stripSecret = (s: State): State => ({ ...s, secret: "" });
      const engine = await Persist.open<State>({
        exclude: [stripSecret],
        initial: state,
        openKV: () => store,
      });
      await engine.persist(state);
      await expect(store.get(Persist.persistedStateKey(1))).resolves.toEqual({
        mock: { ...ZERO_MOCK_STATE.mock },
        secret: "",
      });
    });
  });
});

describe("Persist.middleware", () => {
  it("should pass the action through to next and return its result", () => {
    const engine = makeEngine();
    const action = { type: "any/action" };
    const { next, result } = drive(engine, ZERO_MOCK_STATE, action);
    expect(next).toHaveBeenCalledWith(action);
    expect(result).toBe(action);
  });

  it("should persist the current store state after a normal action", () => {
    const engine = makeEngine();
    const state: MockState = { mock: { value: "9.9.9" } };
    drive(engine, state, { type: "any/action" });
    expect(engine.persist).toHaveBeenCalledWith(state);
    expect(engine.revert).not.toHaveBeenCalled();
    expect(engine.clear).not.toHaveBeenCalled();
  });

  // The revert/clear branches trigger a window reload, which jsdom cannot perform; the
  // production code swallows that failure via .catch, so we suppress the expected log.
  it("should revert instead of persisting on a revertState action", () => {
    const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const engine = makeEngine();
    drive(engine, ZERO_MOCK_STATE, Persist.revertState());
    expect(engine.revert).toHaveBeenCalledOnce();
    expect(engine.persist).not.toHaveBeenCalled();
    errorSpy.mockRestore();
  });

  it("should clear instead of persisting on a clearState action", () => {
    const errorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
    const engine = makeEngine();
    drive(engine, ZERO_MOCK_STATE, Persist.clearState());
    expect(engine.clear).toHaveBeenCalledOnce();
    expect(engine.persist).not.toHaveBeenCalled();
    errorSpy.mockRestore();
  });

  it("should coalesce rapid dispatches into a single persist when debounced", () => {
    vi.useFakeTimers();
    try {
      const engine = makeEngine();
      const state: MockState = { mock: { value: "1.0.0" } };
      const next = vi.fn((a: unknown) => a);
      const store = { getState: () => state, dispatch: vi.fn() };
      const dispatch = Persist.middleware<MockState>(
        engine,
        TimeSpan.milliseconds(250),
      )(store as never)(next);
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
  const V2_PREFIX = `${Persist.V2_STORE_PATH}:`;
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

  it("should clear the persisted store scoped to the v2 path", async () => {
    localStorage.setItem(`${V2_PREFIX}console-version`, JSON.stringify({ version: 1 }));
    localStorage.setItem(
      `${V2_PREFIX}${Persist.persistedStateKey(1)}`,
      JSON.stringify(ZERO_MOCK_STATE),
    );
    localStorage.setItem("unrelated:key", "keep-me");
    Persist.hardClearAndReload();
    await vi.waitFor(() => {
      expect(localStorage.getItem(`${V2_PREFIX}console-version`)).toBeNull();
    });
    expect(
      localStorage.getItem(`${V2_PREFIX}${Persist.persistedStateKey(1)}`),
    ).toBeNull();
    expect(localStorage.getItem("unrelated:key")).toBe("keep-me");
  });
});
