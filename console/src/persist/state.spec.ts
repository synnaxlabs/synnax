// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { kv } from "@synnaxlabs/x";
import { describe, expect, it } from "vitest";

import { Persist } from "@/persist";
import { type SugaredKV } from "@/persist/kv";
import { Version } from "@/version";

type MockState = Version.StoreState;

const ZERO_MOCK_STATE: MockState = {
  [Version.SLICE_NAME]: Version.ZERO_SLICE_STATE,
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
        [Version.SLICE_NAME]: {
          ...ZERO_MOCK_STATE[Version.SLICE_NAME],
          consoleVersion: "16.2.0",
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
        [Version.SLICE_NAME]: {
          ...ZERO_MOCK_STATE[Version.SLICE_NAME],
          consoleVersion: "16.2.0",
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
        [Version.SLICE_NAME]: {
          ...ZERO_MOCK_STATE[Version.SLICE_NAME],
          consoleVersion: "16.2.0",
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
          [Version.SLICE_NAME]: {
            ...ZERO_MOCK_STATE[Version.SLICE_NAME],
            consoleVersion: `16.2.${i}`,
          },
        });

      await expect(store.length()).resolves.toBe(5);
      // Open the engine again to make sure the initial state is correct
      const engine2 = await Persist.open({ openKV, initial: ZERO_MOCK_STATE });
      expect(engine2.initialState).toEqual({
        [Version.SLICE_NAME]: {
          ...ZERO_MOCK_STATE[Version.SLICE_NAME],
          consoleVersion: "16.2.9",
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
        [Version.SLICE_NAME]: {
          ...ZERO_MOCK_STATE[Version.SLICE_NAME],
          consoleVersion: "16.2.0",
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
        [Version.SLICE_NAME]: {
          ...ZERO_MOCK_STATE[Version.SLICE_NAME],
          consoleVersion: "16.2.0",
        },
      };
      await engine.persist(state);
      const state2 = {
        [Version.SLICE_NAME]: {
          ...ZERO_MOCK_STATE[Version.SLICE_NAME],
          consoleVersion: "16.2.1",
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
        [Version.SLICE_NAME]: {
          ...ZERO_MOCK_STATE[Version.SLICE_NAME],
          consoleVersion: "16.2.0",
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
        [Version.SLICE_NAME]: {
          ...ZERO_MOCK_STATE[Version.SLICE_NAME],
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
        [Version.SLICE_NAME]: {
          ...ZERO_MOCK_STATE[Version.SLICE_NAME],
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
        [Version.SLICE_NAME]: {
          ...ZERO_MOCK_STATE[Version.SLICE_NAME],
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
  });
});
