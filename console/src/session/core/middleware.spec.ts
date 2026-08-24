// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore, type Middleware, type Store } from "@reduxjs/toolkit";
import { describe, expect, it, vi } from "vitest";

import { Core } from "@/session/core";
import { Persist } from "@/session/persist";

const CLUSTER_A = "cluster-a";
const CLUSTER_B = "cluster-b";

const BASE: Omit<Core.Core, "key"> = {
  name: "My Core",
  host: "example.com",
  port: 9090,
  username: "",
  password: "",
  secure: false,
};

const ALPHA: Core.Core = { ...BASE, key: "alpha", name: "Alpha" };
const BRAVO: Core.Core = { ...BASE, key: "bravo", name: "Bravo" };

/**
 * A store holding the Core slice and its middleware. Purges land as dispatches rather
 * than deletions, since the engine that acts on them belongs to the persist layer, so
 * the spec records what reaches the chain. A middleware re-dispatches through the top
 * of the chain, which is why the recorder sits there.
 */
const createStore = (
  ...cores: Core.Core[]
): { store: Store<Core.StoreState>; purged: () => string[] } => {
  const record = vi.fn();
  const recorder: Middleware = () => (next) => (action) => {
    record(action);
    return next(action);
  };
  const store = configureStore({
    reducer: { [Core.SLICE_NAME]: Core.reducer },
    preloadedState: {
      [Core.SLICE_NAME]: {
        ...Core.ZERO_SLICE_STATE,
        cores: Object.fromEntries(cores.map((c) => [c.key, c])),
      },
    },
    middleware: (getDefault) => getDefault().concat(recorder, ...Core.MIDDLEWARE),
  });
  const purged = (): string[] =>
    record.mock.calls
      .map(([action]) => action)
      .filter((action) => Persist.purge.match(action))
      .map(({ payload }) => payload);
  return { store, purged };
};

describe("purgeOrphanedClusters", () => {
  describe("on remove", () => {
    it("should purge the cluster the removed Core was the last to name", () => {
      const { store, purged } = createStore({ ...ALPHA, clusterKey: CLUSTER_A });
      store.dispatch(Core.remove(ALPHA.key));
      expect(purged()).toEqual([CLUSTER_A]);
    });

    it("should keep a cluster another Core still names", () => {
      const { store, purged } = createStore(
        { ...ALPHA, clusterKey: CLUSTER_A },
        { ...BRAVO, clusterKey: CLUSTER_A },
      );
      store.dispatch(Core.remove(ALPHA.key));
      expect(purged()).toEqual([]);
    });

    it("should purge a cluster once when every Core naming it goes at once", () => {
      const { store, purged } = createStore(
        { ...ALPHA, clusterKey: CLUSTER_A },
        { ...BRAVO, clusterKey: CLUSTER_A },
      );
      store.dispatch(Core.remove([ALPHA.key, BRAVO.key]));
      expect(purged()).toEqual([CLUSTER_A]);
    });

    it("should purge every cluster the removed Cores named", () => {
      const { store, purged } = createStore(
        { ...ALPHA, clusterKey: CLUSTER_A },
        { ...BRAVO, clusterKey: CLUSTER_B },
      );
      store.dispatch(Core.remove([ALPHA.key, BRAVO.key]));
      expect(purged().sort()).toEqual([CLUSTER_A, CLUSTER_B]);
    });

    it("should purge nothing for a Core that never connected", () => {
      const { store, purged } = createStore(ALPHA);
      store.dispatch(Core.remove(ALPHA.key));
      expect(purged()).toEqual([]);
    });
  });

  describe("on set", () => {
    // The reducer drops the cached cluster when the address moves, so without this
    // the old partition would outlive every Core that could open it.
    it("should purge the cluster a repointed Core was the last to name", () => {
      const cached = { ...ALPHA, clusterKey: CLUSTER_A };
      const { store, purged } = createStore(cached);
      store.dispatch(Core.set({ ...cached, host: "elsewhere.com" }));
      expect(purged()).toEqual([CLUSTER_A]);
    });

    it("should keep the cluster when the edit leaves the address alone", () => {
      const cached = { ...ALPHA, clusterKey: CLUSTER_A };
      const { store, purged } = createStore(cached);
      store.dispatch(Core.set({ ...cached, name: "Renamed" }));
      expect(purged()).toEqual([]);
    });

    it("should keep the cluster another Core still names after the move", () => {
      const cached = { ...ALPHA, clusterKey: CLUSTER_A };
      const { store, purged } = createStore(cached, {
        ...BRAVO,
        clusterKey: CLUSTER_A,
      });
      store.dispatch(Core.set({ ...cached, host: "elsewhere.com" }));
      expect(purged()).toEqual([]);
    });

    it("should purge nothing when a Core is added", () => {
      const { store, purged } = createStore();
      store.dispatch(Core.set(BASE));
      expect(purged()).toEqual([]);
    });
  });

  it("should leave the cluster alone for an action that names no Core", () => {
    const { store, purged } = createStore({ ...ALPHA, clusterKey: CLUSTER_A });
    store.dispatch(Core.select(ALPHA.key));
    store.dispatch(Core.rename({ key: ALPHA.key, name: "Renamed" }));
    expect(purged()).toEqual([]);
    expect(Core.selectClusterKey(store.getState())).toBe(CLUSTER_A);
  });
});
