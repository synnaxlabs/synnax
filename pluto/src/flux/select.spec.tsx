// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import { act, renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Flux } from "@/flux";
import { type base } from "@/flux/base";
import { Client } from "@/flux/base/client";
import { createSelector } from "@/flux/select";
import { status } from "@/status/aether";
import { aetherTest } from "@/aether/test";
import { Status } from "@/status/base";
import { Synnax } from "@/synnax";
import { synnax } from "@/synnax/aether";
import { flux } from "@/flux/aether";

interface Doc extends record.Keyed<string> {
  key: string;
  name: string;
  count: number;
  nested: { x: number; y: number };
}

interface TestStore extends base.Store {
  docs: base.UnaryStore<string, Doc>;
}

const STORE_CONFIG: base.StoreConfig<TestStore> = {
  docs: { listeners: [] },
};

const AetherProvider = aetherTest.createProvider({
  ...synnax.REGISTRY,
  ...status.REGISTRY,
  ...flux.createRegistry({ storeConfig: {} }),
});

const createTestWrapper = (): {
  wrapper: React.FC<PropsWithChildren>;
  fluxClient: Client<TestStore>;
} => {
  const handleError = status.createErrorHandler(console.error);
  const handleAsyncError = status.createAsyncErrorHandler(console.error);
  const fluxClient = new Client<TestStore>({
    client: null,
    storeConfig: STORE_CONFIG,
    handleError,
    handleAsyncError,
  });
  const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
    <AetherProvider>
      <Status.Aggregator>
        <Synnax.TestProvider client={null}>
          <Flux.Provider client={fluxClient}>{children}</Flux.Provider>
        </Synnax.TestProvider>
      </Status.Aggregator>
    </AetherProvider>
  );
  return { wrapper: Wrapper, fluxClient };
};

const setDoc = (fluxClient: Client<TestStore>, doc: Doc): void => {
  const store = fluxClient.scopedStore<TestStore>("test-writer");
  store.docs.set(doc.key, doc);
};

describe("createSelector", () => {
  describe("basic selection", () => {
    it("should return the selected value from the store", () => {
      const { wrapper, fluxClient } = createTestWrapper();
      setDoc(fluxClient, {
        key: "k1",
        name: "Alice",
        count: 1,
        nested: { x: 0, y: 0 },
      });

      const useSelectName = createSelector<TestStore, { key: string }, string>({
        subscribe: (store, { key }, notify) => store.docs.onSet(notify, key),
        select: (store, { key }) =>
          (store.docs.get(key) as Doc | undefined)?.name ?? "",
      });

      const { result } = renderHook(() => useSelectName({ key: "k1" }), {
        wrapper,
      });
      expect(result.current).toBe("Alice");
    });

    it("should return a default when document is not in store", () => {
      const { wrapper } = createTestWrapper();

      const useSelectDoc = createSelector<TestStore, { key: string }, Doc | undefined>({
        subscribe: (store, { key }, notify) => store.docs.onSet(notify, key),
        select: (store, { key }) => store.docs.get(key) as Doc | undefined,
      });

      const { result } = renderHook(() => useSelectDoc({ key: "missing" }), {
        wrapper,
      });
      expect(result.current).toBeUndefined();
    });
  });

  describe("reactivity", () => {
    it("should update when the subscribed key changes in the store", () => {
      const { wrapper, fluxClient } = createTestWrapper();

      const useSelectName = createSelector<TestStore, { key: string }, string>({
        subscribe: (store, { key }, notify) => store.docs.onSet(notify, key),
        select: (store, { key }) =>
          (store.docs.get(key) as Doc | undefined)?.name ?? "",
      });

      const { result } = renderHook(() => useSelectName({ key: "k1" }), {
        wrapper,
      });
      expect(result.current).toBe("");

      act(() =>
        setDoc(fluxClient, {
          key: "k1",
          name: "Alice",
          count: 1,
          nested: { x: 0, y: 0 },
        }),
      );

      expect(result.current).toBe("Alice");
    });

    it("should not re-render when a different key changes", () => {
      const { wrapper, fluxClient } = createTestWrapper();
      let renderCount = 0;

      const useSelectName = createSelector<TestStore, { key: string }, string>({
        subscribe: (store, { key }, notify) => store.docs.onSet(notify, key),
        select: (store, { key }) => {
          renderCount++;
          return (store.docs.get(key) as Doc | undefined)?.name ?? "";
        },
      });

      renderHook(() => useSelectName({ key: "k1" }), { wrapper });
      const countAfterMount = renderCount;

      act(() =>
        setDoc(fluxClient, {
          key: "k2",
          name: "Bob",
          count: 2,
          nested: { x: 0, y: 0 },
        }),
      );

      expect(renderCount).toBe(countAfterMount);
    });
  });

  describe("equality and caching", () => {
    it("should use custom equality function to prevent unnecessary updates", () => {
      const { wrapper, fluxClient } = createTestWrapper();
      setDoc(fluxClient, {
        key: "k1",
        name: "Alice",
        count: 1,
        nested: { x: 1, y: 2 },
      });

      let renderCount = 0;
      const shallowEqual = (
        a: { x: number; y: number },
        b: { x: number; y: number },
      ): boolean => a.x === b.x && a.y === b.y;

      const useSelectNested = createSelector<
        TestStore,
        { key: string },
        { x: number; y: number }
      >({
        subscribe: (store, { key }, notify) => store.docs.onSet(notify, key),
        select: (store, { key }) =>
          (store.docs.get(key) as Doc | undefined)?.nested ?? { x: 0, y: 0 },
        equal: shallowEqual,
      });

      const { result } = renderHook(
        () => {
          renderCount++;
          return useSelectNested({ key: "k1" });
        },
        { wrapper },
      );

      const countAfterMount = renderCount;
      const firstRef = result.current;

      // Update name only - nested stays structurally equal but new reference
      act(() =>
        setDoc(fluxClient, {
          key: "k1",
          name: "Bob",
          count: 2,
          nested: { x: 1, y: 2 },
        }),
      );

      // Custom equality should prevent re-render
      expect(renderCount).toBe(countAfterMount);
      // Should return the same cached reference
      expect(result.current).toBe(firstRef);
    });

    it("should update when selected value actually changes per custom equality", () => {
      const { wrapper, fluxClient } = createTestWrapper();
      setDoc(fluxClient, {
        key: "k1",
        name: "Alice",
        count: 1,
        nested: { x: 1, y: 2 },
      });

      const shallowEqual = (
        a: { x: number; y: number },
        b: { x: number; y: number },
      ): boolean => a.x === b.x && a.y === b.y;

      const useSelectNested = createSelector<
        TestStore,
        { key: string },
        { x: number; y: number }
      >({
        subscribe: (store, { key }, notify) => store.docs.onSet(notify, key),
        select: (store, { key }) =>
          (store.docs.get(key) as Doc | undefined)?.nested ?? { x: 0, y: 0 },
        equal: shallowEqual,
      });

      const { result } = renderHook(() => useSelectNested({ key: "k1" }), {
        wrapper,
      });
      expect(result.current).toEqual({ x: 1, y: 2 });

      act(() =>
        setDoc(fluxClient, {
          key: "k1",
          name: "Alice",
          count: 1,
          nested: { x: 99, y: 2 },
        }),
      );

      expect(result.current).toEqual({ x: 99, y: 2 });
    });

    it("should skip re-render when primitive selection is unchanged", () => {
      const { wrapper, fluxClient } = createTestWrapper();
      setDoc(fluxClient, {
        key: "k1",
        name: "Alice",
        count: 5,
        nested: { x: 0, y: 0 },
      });

      let renderCount = 0;

      const useSelectCount = createSelector<TestStore, { key: string }, number>({
        subscribe: (store, { key }, notify) => store.docs.onSet(notify, key),
        select: (store, { key }) =>
          (store.docs.get(key) as Doc | undefined)?.count ?? 0,
      });

      const { result } = renderHook(
        () => {
          renderCount++;
          return useSelectCount({ key: "k1" });
        },
        { wrapper },
      );

      expect(result.current).toBe(5);
      const countAfterMount = renderCount;

      // Change name but not count
      act(() =>
        setDoc(fluxClient, {
          key: "k1",
          name: "Bob",
          count: 5,
          nested: { x: 0, y: 0 },
        }),
      );

      expect(result.current).toBe(5);
      expect(renderCount).toBe(countAfterMount);

      // Now actually change count
      act(() =>
        setDoc(fluxClient, {
          key: "k1",
          name: "Bob",
          count: 10,
          nested: { x: 0, y: 0 },
        }),
      );

      expect(result.current).toBe(10);
    });
  });

  describe("derived selection stability", () => {
    it("should not infinite loop when select returns a new object reference from unchanged data", () => {
      const { wrapper, fluxClient } = createTestWrapper();
      setDoc(fluxClient, {
        key: "k1",
        name: "Alice",
        count: 1,
        nested: { x: 0, y: 0 },
      });

      let renderCount = 0;

      // This selector creates a NEW object every call - the exact pattern
      // that would cause infinite re-renders without version tracking.
      const useSelectDerived = createSelector<
        TestStore,
        { key: string },
        { name: string; count: number }
      >({
        subscribe: (store, { key }, notify) => store.docs.onSet(notify, key),
        select: (store, { key }) => {
          const doc = store.docs.get(key) as Doc | undefined;
          return { name: doc?.name ?? "", count: doc?.count ?? 0 };
        },
      });

      const { result } = renderHook(
        () => {
          renderCount++;
          return useSelectDerived({ key: "k1" });
        },
        { wrapper },
      );

      expect(result.current).toEqual({ name: "Alice", count: 1 });
      // Should have rendered a bounded number of times, not infinite
      expect(renderCount).toBeLessThan(5);
    });

    it("should still update derived selections when store data changes", () => {
      const { wrapper, fluxClient } = createTestWrapper();
      setDoc(fluxClient, {
        key: "k1",
        name: "Alice",
        count: 1,
        nested: { x: 0, y: 0 },
      });

      const useSelectDerived = createSelector<
        TestStore,
        { key: string },
        { name: string; count: number }
      >({
        subscribe: (store, { key }, notify) => store.docs.onSet(notify, key),
        select: (store, { key }) => {
          const doc = store.docs.get(key) as Doc | undefined;
          return { name: doc?.name ?? "", count: doc?.count ?? 0 };
        },
      });

      const { result } = renderHook(() => useSelectDerived({ key: "k1" }), {
        wrapper,
      });

      expect(result.current).toEqual({ name: "Alice", count: 1 });

      act(() =>
        setDoc(fluxClient, {
          key: "k1",
          name: "Bob",
          count: 99,
          nested: { x: 0, y: 0 },
        }),
      );

      expect(result.current).toEqual({ name: "Bob", count: 99 });
    });
  });

  describe("args memoization", () => {
    it("should not re-subscribe when args are deep-equal but referentially different", () => {
      const { wrapper } = createTestWrapper();
      const subscribeSpy = vi.fn(
        (store: TestStore, { key }: { key: string }, notify: () => void) =>
          store.docs.onSet(notify, key),
      );

      const useSelectName = createSelector<TestStore, { key: string }, string>({
        subscribe: subscribeSpy,
        select: (store, { key }) =>
          (store.docs.get(key) as Doc | undefined)?.name ?? "",
      });

      const { rerender } = renderHook(({ k }) => useSelectName({ key: k }), {
        wrapper,
        initialProps: { k: "k1" },
      });

      const callCountAfterMount = subscribeSpy.mock.calls.length;

      // Re-render with a new object that is deep-equal
      rerender({ k: "k1" });

      expect(subscribeSpy.mock.calls.length).toBe(callCountAfterMount);
    });

    it("should re-subscribe when args actually change", () => {
      const { wrapper } = createTestWrapper();
      const subscribeSpy = vi.fn(
        (store: TestStore, { key }: { key: string }, notify: () => void) =>
          store.docs.onSet(notify, key),
      );

      const useSelectName = createSelector<TestStore, { key: string }, string>({
        subscribe: subscribeSpy,
        select: (store, { key }) =>
          (store.docs.get(key) as Doc | undefined)?.name ?? "",
      });

      const { rerender } = renderHook(({ k }) => useSelectName({ key: k }), {
        wrapper,
        initialProps: { k: "k1" },
      });

      const callCountAfterMount = subscribeSpy.mock.calls.length;

      rerender({ k: "k2" });

      expect(subscribeSpy.mock.calls.length).toBeGreaterThan(callCountAfterMount);
    });
  });

  describe("cleanup", () => {
    it("should unsubscribe when the component unmounts", () => {
      const { wrapper } = createTestWrapper();
      const unsubscribe = vi.fn();

      const useSelectName = createSelector<TestStore, { key: string }, string>({
        subscribe: (_store, _args, notify) => {
          notify;
          return unsubscribe;
        },
        select: () => "",
      });

      const { unmount } = renderHook(() => useSelectName({ key: "k1" }), {
        wrapper,
      });

      expect(unsubscribe).not.toHaveBeenCalled();

      unmount();

      expect(unsubscribe).toHaveBeenCalled();
    });
  });
});
