// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";
import { act, renderHook } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { Provider } from "react-redux";
import { describe, expect, it } from "vitest";

import { Cluster } from "@/session/cluster";
import { createCluster } from "@/session/cluster/testutil";

const CLUSTER_A = createCluster("a", { name: "Alpha", host: "a.example.com" });
const CLUSTER_B = createCluster("b", {
  name: "Beta",
  host: "b.example.com",
  secure: true,
});

const EMPTY_STATE: Cluster.SliceState = {
  version: 0,
  selected: undefined,
  clusters: {},
};

const stateWith = (
  clusters: Cluster.Cluster[],
  selected?: string,
): Cluster.SliceState => ({
  version: 0,
  selected,
  clusters: Object.fromEntries(clusters.map((c) => [c.key, c])),
});

const storeWith = (
  clusters: Cluster.Cluster[],
  selected?: string,
): Cluster.StoreState => ({ [Cluster.SLICE_NAME]: stateWith(clusters, selected) });

const createStore = (initial: Cluster.SliceState = EMPTY_STATE) =>
  configureStore({
    reducer: { [Cluster.SLICE_NAME]: Cluster.reducer },
    preloadedState: { [Cluster.SLICE_NAME]: initial },
  });

const createWrapper = (
  store: ReturnType<typeof createStore>,
): FC<PropsWithChildren> => {
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <Provider store={store}>{children}</Provider>
  );
  Wrapper.displayName = "Wrapper";
  return Wrapper;
};

describe("cluster selectors", () => {
  describe("raw selectors", () => {
    describe("selectSelectedKey", () => {
      it("should return the selected key", () => {
        expect(Cluster.selectSelectedKey(storeWith([CLUSTER_A], "a"))).toBe("a");
      });

      it("should return undefined when nothing is selected", () => {
        expect(Cluster.selectSelectedKey(storeWith([CLUSTER_A]))).toBeUndefined();
      });
    });

    describe("selectState", () => {
      it("should resolve a cluster by explicit key", () => {
        expect(Cluster.selectState(storeWith([CLUSTER_A, CLUSTER_B]), "b")).toEqual(
          CLUSTER_B,
        );
      });

      it("should fall back to the selected cluster when no key is given", () => {
        expect(Cluster.selectState(storeWith([CLUSTER_A, CLUSTER_B], "a"))).toEqual(
          CLUSTER_A,
        );
      });

      it("should return undefined when neither a key nor a selection resolves", () => {
        expect(Cluster.selectState(storeWith([CLUSTER_A]))).toBeUndefined();
      });
    });
  });

  describe("getters", () => {
    it("should read the whole slice state on demand across dispatches", () => {
      const store = createStore();
      const { result } = renderHook(() => Cluster.useGetSliceState(), {
        wrapper: createWrapper(store),
      });
      const get = result.current;
      expect(get().clusters).toEqual({});
      act(() => {
        store.dispatch(Cluster.set(CLUSTER_A));
      });
      expect(get().clusters).toEqual({ a: CLUSTER_A });
    });

    it("should read the current selected key on demand across dispatches", () => {
      const store = createStore(stateWith([CLUSTER_A, CLUSTER_B]));
      const { result } = renderHook(() => Cluster.useGetSelectedKey(), {
        wrapper: createWrapper(store),
      });
      const get = result.current;
      expect(get()).toBeUndefined();
      act(() => {
        store.dispatch(Cluster.select("a"));
      });
      expect(get()).toBe("a");
      act(() => {
        store.dispatch(Cluster.select("b"));
      });
      expect(get()).toBe("b");
    });

    it("should resolve a cluster by key and fall back to the selection on demand", () => {
      const store = createStore();
      const { result } = renderHook(() => Cluster.useGetState(), {
        wrapper: createWrapper(store),
      });
      const get = result.current;
      expect(get("b")).toBeUndefined();
      act(() => {
        store.dispatch(Cluster.set(CLUSTER_A));
        store.dispatch(Cluster.set(CLUSTER_B));
      });
      expect(get("b")).toEqual(CLUSTER_B);
      act(() => {
        store.dispatch(Cluster.select("a"));
      });
      expect(get()).toEqual(CLUSTER_A);
    });

    it("should report whether any cluster is selected on demand", () => {
      const store = createStore(stateWith([CLUSTER_A]));
      const { result } = renderHook(() => Cluster.useGetIsAnySelected(), {
        wrapper: createWrapper(store),
      });
      const get = result.current;
      expect(get()).toBe(false);
      act(() => {
        store.dispatch(Cluster.select("a"));
      });
      expect(get()).toBe(true);
    });
  });
});
