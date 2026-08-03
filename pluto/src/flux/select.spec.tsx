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

import { aetherTest } from "@/aether/test";
import { Flux } from "@/flux";
import { flux } from "@/flux/aether";
import { base } from "@/flux/base";
import { status } from "@/status/aether";
import { Status } from "@/status/base";
import { Synnax } from "@/synnax";
import { synnax } from "@/synnax/aether";

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
  fluxClient: base.Client<TestStore>;
} => {
  const handleError = status.createErrorHandler(console.error);
  const handleAsyncError = status.createAsyncErrorHandler(console.error);
  const fluxClient = new base.Client<TestStore>({
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

const setDoc = (fluxClient: base.Client<TestStore>, doc: Doc): void => {
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

      const [useSelectName] = Flux.createSelector<TestStore, { key: string }, string>({
        subscribe: (store, { key }, notify) => store.docs.onSet(notify, key),
        select: (store, { key }) => store.docs.get(key)?.name ?? "",
      });

      const { result } = renderHook(() => useSelectName({ key: "k1" }), {
        wrapper,
      });
      expect(result.current).toBe("Alice");
    });

    it("should return a default when document is not in store", () => {
      const { wrapper } = createTestWrapper();

      const [useSelectDoc] = Flux.createSelector<
        TestStore,
        { key: string },
        Doc | undefined
      >({
        subscribe: (store, { key }, notify) => store.docs.onSet(notify, key),
        select: (store, { key }) => store.docs.get(key),
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

      const [useSelectName] = Flux.createSelector<TestStore, { key: string }, string>({
        subscribe: (store, { key }, notify) => store.docs.onSet(notify, key),
        select: (store, { key }) => store.docs.get(key)?.name ?? "",
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

      const [useSelectName] = Flux.createSelector<TestStore, { key: string }, string>({
        subscribe: (store, { key }, notify) => store.docs.onSet(notify, key),
        select: (store, { key }) => {
          renderCount++;
          return store.docs.get(key)?.name ?? "";
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

      const [useSelectNested] = Flux.createSelector<
        TestStore,
        { key: string },
        { x: number; y: number }
      >({
        subscribe: (store, { key }, notify) => store.docs.onSet(notify, key),
        select: (store, { key }) => store.docs.get(key)?.nested ?? { x: 0, y: 0 },
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

      act(() =>
        setDoc(fluxClient, {
          key: "k1",
          name: "Bob",
          count: 2,
          nested: { x: 1, y: 2 },
        }),
      );

      expect(renderCount).toBe(countAfterMount);
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

      const [useSelectNested] = Flux.createSelector<
        TestStore,
        { key: string },
        { x: number; y: number }
      >({
        subscribe: (store, { key }, notify) => store.docs.onSet(notify, key),
        select: (store, { key }) => store.docs.get(key)?.nested ?? { x: 0, y: 0 },
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

      const [useSelectCount] = Flux.createSelector<TestStore, { key: string }, number>({
        subscribe: (store, { key }, notify) => store.docs.onSet(notify, key),
        select: (store, { key }) => store.docs.get(key)?.count ?? 0,
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

      const [useSelectDerived] = Flux.createSelector<
        TestStore,
        { key: string },
        { name: string; count: number }
      >({
        subscribe: (store, { key }, notify) => store.docs.onSet(notify, key),
        select: (store, { key }) => {
          const doc = store.docs.get(key);
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

      const [useSelectDerived] = Flux.createSelector<
        TestStore,
        { key: string },
        { name: string; count: number }
      >({
        subscribe: (store, { key }, notify) => store.docs.onSet(notify, key),
        select: (store, { key }) => {
          const doc = store.docs.get(key);
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

      const [useSelectName] = Flux.createSelector<TestStore, { key: string }, string>({
        subscribe: subscribeSpy,
        select: (store, { key }) => store.docs.get(key)?.name ?? "",
      });

      const { rerender } = renderHook(({ k }) => useSelectName({ key: k }), {
        wrapper,
        initialProps: { k: "k1" },
      });

      const callCountAfterMount = subscribeSpy.mock.calls.length;

      rerender({ k: "k1" });

      expect(subscribeSpy.mock.calls.length).toBe(callCountAfterMount);
    });

    it("should re-subscribe when args actually change", () => {
      const { wrapper } = createTestWrapper();
      const subscribeSpy = vi.fn(
        (store: TestStore, { key }: { key: string }, notify: () => void) =>
          store.docs.onSet(notify, key),
      );

      const [useSelectName] = Flux.createSelector<TestStore, { key: string }, string>({
        subscribe: subscribeSpy,
        select: (store, { key }) => store.docs.get(key)?.name ?? "",
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

  describe("transform", () => {
    it("should apply transform to the raw selection", () => {
      const { wrapper, fluxClient } = createTestWrapper();
      setDoc(fluxClient, {
        key: "k1",
        name: "Alice",
        count: 1,
        nested: { x: 1, y: 2 },
      });

      const [useSelectSum] = Flux.createSelector<
        TestStore,
        { key: string },
        number,
        { x: number; y: number }
      >({
        subscribe: (store, { key }, notify) => store.docs.onSet(notify, key),
        select: (store, { key }) => store.docs.get(key)?.nested ?? { x: 0, y: 0 },
        transform: (raw) => raw.x + raw.y,
      });

      const { result } = renderHook(() => useSelectSum({ key: "k1" }), { wrapper });
      expect(result.current).toBe(3);
    });

    it("should not re-run transform when the raw selection reference is unchanged", () => {
      const { wrapper, fluxClient } = createTestWrapper();
      const nested = { x: 1, y: 2 };
      setDoc(fluxClient, { key: "k1", name: "Alice", count: 1, nested });

      const transform = vi.fn((raw: { x: number; y: number }) => ({
        sum: raw.x + raw.y,
      }));
      const [useSelectNested] = Flux.createSelector<
        TestStore,
        { key: string },
        { sum: number },
        { x: number; y: number }
      >({
        subscribe: (store, { key }, notify) => store.docs.onSet(notify, key),
        select: (store, { key }) => store.docs.get(key)?.nested ?? { x: 0, y: 0 },
        transform,
      });

      const { result } = renderHook(() => useSelectNested({ key: "k1" }), { wrapper });
      const firstOut = result.current;
      const callsAfterMount = transform.mock.calls.length;

      act(() => setDoc(fluxClient, { key: "k1", name: "Bob", count: 2, nested }));

      expect(transform.mock.calls.length).toBe(callsAfterMount);
      expect(result.current).toBe(firstOut);
    });

    it("should re-run transform when the raw selection reference changes", () => {
      const { wrapper, fluxClient } = createTestWrapper();
      setDoc(fluxClient, {
        key: "k1",
        name: "Alice",
        count: 1,
        nested: { x: 1, y: 2 },
      });

      const transform = vi.fn((raw: { x: number; y: number }) => raw.x + raw.y);
      const [useSelectSum] = Flux.createSelector<
        TestStore,
        { key: string },
        number,
        { x: number; y: number }
      >({
        subscribe: (store, { key }, notify) => store.docs.onSet(notify, key),
        select: (store, { key }) => store.docs.get(key)?.nested ?? { x: 0, y: 0 },
        transform,
      });

      const { result } = renderHook(() => useSelectSum({ key: "k1" }), { wrapper });
      expect(result.current).toBe(3);

      act(() =>
        setDoc(fluxClient, {
          key: "k1",
          name: "Alice",
          count: 1,
          nested: { x: 10, y: 5 },
        }),
      );

      expect(result.current).toBe(15);
    });

    it("should re-run transform when an arg it depends on changes but the raw selection is unchanged", () => {
      const { wrapper, fluxClient } = createTestWrapper();
      setDoc(fluxClient, {
        key: "k1",
        name: "Alice",
        count: 1,
        nested: { x: 2, y: 3 },
      });

      const [useSelectScaled] = Flux.createSelector<
        TestStore,
        { key: string; multiplier: number },
        number,
        { x: number; y: number }
      >({
        subscribe: (store, { key }, notify) => store.docs.onSet(notify, key),
        select: (store, { key }) => store.docs.get(key)?.nested ?? { x: 0, y: 0 },
        transform: (raw, { multiplier }) => (raw.x + raw.y) * multiplier,
      });

      const { result, rerender } = renderHook(
        ({ multiplier }) => useSelectScaled({ key: "k1", multiplier }),
        { wrapper, initialProps: { multiplier: 1 } },
      );
      expect(result.current).toBe(5);

      rerender({ multiplier: 2 });

      expect(result.current).toBe(10);
    });

    it("should not re-run transform when args are deep-equal but referentially different", () => {
      const { wrapper, fluxClient } = createTestWrapper();
      const nested = { x: 2, y: 3 };
      setDoc(fluxClient, { key: "k1", name: "Alice", count: 1, nested });

      const transform = vi.fn(
        (raw: { x: number; y: number }, { multiplier }: { multiplier: number }) =>
          (raw.x + raw.y) * multiplier,
      );
      const [useSelectScaled] = Flux.createSelector<
        TestStore,
        { key: string; multiplier: number },
        number,
        { x: number; y: number }
      >({
        subscribe: (store, { key }, notify) => store.docs.onSet(notify, key),
        select: (store, { key }) => store.docs.get(key)?.nested ?? { x: 0, y: 0 },
        transform,
      });

      const { result, rerender } = renderHook(
        ({ multiplier }) => useSelectScaled({ key: "k1", multiplier }),
        { wrapper, initialProps: { multiplier: 2 } },
      );
      expect(result.current).toBe(10);
      const callsAfterMount = transform.mock.calls.length;

      rerender({ multiplier: 2 });

      expect(transform.mock.calls.length).toBe(callsAfterMount);
      expect(result.current).toBe(10);
    });
  });

  describe("cleanup", () => {
    it("should unsubscribe when the component unmounts", () => {
      const { wrapper } = createTestWrapper();
      const unsubscribe = vi.fn();

      const [useSelectName] = Flux.createSelector<TestStore, { key: string }, string>({
        subscribe: (_store, _args, _notify) => unsubscribe,
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

  describe("useGet", () => {
    it("should return the current value from the store when invoked", () => {
      const { wrapper, fluxClient } = createTestWrapper();
      setDoc(fluxClient, {
        key: "k1",
        name: "Alice",
        count: 1,
        nested: { x: 0, y: 0 },
      });

      const [, useGetName] = Flux.createSelector<TestStore, { key: string }, string>({
        subscribe: (store, { key }, notify) => store.docs.onSet(notify, key),
        select: (store, { key }) => store.docs.get(key)?.name ?? "",
      });

      const { result } = renderHook(() => useGetName(), { wrapper });
      expect(result.current({ key: "k1" })).toBe("Alice");
    });

    it("should read the latest value on each call without re-rendering", () => {
      const { wrapper, fluxClient } = createTestWrapper();
      setDoc(fluxClient, {
        key: "k1",
        name: "Alice",
        count: 1,
        nested: { x: 0, y: 0 },
      });

      let renderCount = 0;
      const [, useGetName] = Flux.createSelector<TestStore, { key: string }, string>({
        subscribe: (store, { key }, notify) => store.docs.onSet(notify, key),
        select: (store, { key }) => store.docs.get(key)?.name ?? "",
      });

      const { result } = renderHook(
        () => {
          renderCount++;
          return useGetName();
        },
        { wrapper },
      );

      expect(result.current({ key: "k1" })).toBe("Alice");
      const countAfterMount = renderCount;

      act(() =>
        setDoc(fluxClient, {
          key: "k1",
          name: "Bob",
          count: 2,
          nested: { x: 0, y: 0 },
        }),
      );

      expect(result.current({ key: "k1" })).toBe("Bob");
      expect(renderCount).toBe(countAfterMount);
    });

    it("should never subscribe to the store", () => {
      const { wrapper } = createTestWrapper();
      const subscribeSpy = vi.fn(
        (store: TestStore, { key }: { key: string }, notify: () => void) =>
          store.docs.onSet(notify, key),
      );

      const [, useGetName] = Flux.createSelector<TestStore, { key: string }, string>({
        subscribe: subscribeSpy,
        select: (store, { key }) => store.docs.get(key)?.name ?? "",
      });

      const { result } = renderHook(() => useGetName(), { wrapper });
      result.current({ key: "k1" });

      expect(subscribeSpy).not.toHaveBeenCalled();
    });

    it("should apply the transform to the raw selection", () => {
      const { wrapper, fluxClient } = createTestWrapper();
      setDoc(fluxClient, {
        key: "k1",
        name: "Alice",
        count: 1,
        nested: { x: 1, y: 2 },
      });

      const [, useGetSum] = Flux.createSelector<
        TestStore,
        { key: string },
        number,
        { x: number; y: number }
      >({
        subscribe: (store, { key }, notify) => store.docs.onSet(notify, key),
        select: (store, { key }) => store.docs.get(key)?.nested ?? { x: 0, y: 0 },
        transform: (raw) => raw.x + raw.y,
      });

      const { result } = renderHook(() => useGetSum(), { wrapper });
      expect(result.current({ key: "k1" })).toBe(3);
    });

    it("should return a stable getter across re-renders with equal args", () => {
      const { wrapper } = createTestWrapper();

      const [, useGetName] = Flux.createSelector<TestStore, { key: string }, string>({
        subscribe: (store, { key }, notify) => store.docs.onSet(notify, key),
        select: (store, { key }) => store.docs.get(key)?.name ?? "",
      });

      const { result, rerender } = renderHook(() => useGetName(), {
        wrapper,
      });
      const firstGetter = result.current;

      rerender();

      expect(result.current).toBe(firstGetter);
    });
  });
});
