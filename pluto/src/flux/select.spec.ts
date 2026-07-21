// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { schematic, type Synnax as Client } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uuid, type xy } from "@synnaxlabs/x";
import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Flux } from "@/flux";
import { createSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();
const wrapper = createSynnaxWrapper({ client });

interface SelectKeyArgs {
  key: schematic.Key;
}

const getSchem = (
  client: Client | null,
  key: schematic.Key,
): schematic.Schematic | undefined => {
  const cached = client?.schematics.getCached({ key });
  if (cached == null || cached.variant === "deleted") return undefined;
  return cached.data;
};

const subscribe = (
  { client, args: { key } }: Flux.SelectorParams<SelectKeyArgs>,
  notify: () => void,
) => (client == null ? () => {} : client.schematics.onChange({ key }, notify));

const ZERO_XY: xy.XY = { x: 0, y: 0 };

const getPosition = (client: Client | null, key: schematic.Key): xy.XY =>
  getSchem(client, key)?.nodes.find((n) => n.key === "n1")?.position ?? ZERO_XY;

interface CreateSchemParams {
  name?: string;
  position?: xy.XY;
  prime?: boolean;
}

// prime populates the answer cache via retrieve; without it a query only
// resolves once a subscriber mounts and seeds from the record table.
const createSchem = async ({
  name = "Alice",
  position = { x: 1, y: 2 },
  prime = true,
}: CreateSchemParams = {}): Promise<schematic.Schematic> => {
  const proj = await client.projects.create({
    name: `select_${uuid.create()}`,
    layout: {},
  });
  const schem = await client.schematics.create(proj.key, {
    name,
    nodes: [{ key: "n1", position }],
    edges: [],
    configs: {},
  });
  if (prime) await client.schematics.retrieve({ key: schem.key });
  return schem;
};

const rename = async (key: schematic.Key, name: string): Promise<void> => {
  await client.schematics.dispatch(key, schematic.rename({ name }));
};

const moveN1 = async (key: schematic.Key, position: xy.XY): Promise<void> => {
  await client.schematics.dispatch(
    key,
    schematic.setNodePosition({ key: "n1", position }),
  );
};

describe("createSelector", () => {
  describe("basic selection", () => {
    it("should return the selected value from the cache", async () => {
      const schem = await createSchem({ name: "Alice" });

      const [useSelectName] = Flux.createSelector<SelectKeyArgs, string>({
        subscribe,
        select: ({ client, args: { key } }) => getSchem(client, key)?.name ?? "",
      });

      const { result } = renderHook(() => useSelectName({ key: schem.key }), {
        wrapper,
      });
      expect(result.current).toBe("Alice");
    });

    it("should return a default when the document is not cached", async () => {
      const [useSelectDoc] = Flux.createSelector<
        SelectKeyArgs,
        schematic.Schematic | undefined
      >({
        subscribe,
        select: ({ client, args: { key } }) => getSchem(client, key),
      });

      const { result } = renderHook(() => useSelectDoc({ key: uuid.create() }), {
        wrapper,
      });
      expect(result.current).toBeUndefined();
    });
  });

  describe("reactivity", () => {
    it("should update when the subscribed key changes in the cache", async () => {
      const schem = await createSchem({ name: "Alice", prime: false });

      const [useSelectName] = Flux.createSelector<SelectKeyArgs, string>({
        subscribe,
        select: ({ client, args: { key } }) => getSchem(client, key)?.name ?? "",
      });

      const { result } = renderHook(() => useSelectName({ key: schem.key }), {
        wrapper,
      });
      expect(result.current).toBe("Alice");

      await act(async () => await rename(schem.key, "Bob"));

      expect(result.current).toBe("Bob");
    });

    it("should not re-render when a different key changes", async () => {
      const a = await createSchem({ name: "Alice" });
      const b = await createSchem({ name: "Bob" });
      let selectCount = 0;

      const [useSelectName] = Flux.createSelector<SelectKeyArgs, string>({
        subscribe,
        select: ({ client, args: { key } }) => {
          selectCount++;
          return getSchem(client, key)?.name ?? "";
        },
      });

      renderHook(() => useSelectName({ key: a.key }), { wrapper });
      const countAfterMount = selectCount;

      await act(async () => await rename(b.key, "Bobby"));

      expect(selectCount).toBe(countAfterMount);
    });
  });

  describe("equality and caching", () => {
    const shallowEqual = (a: xy.XY, b: xy.XY): boolean => a.x === b.x && a.y === b.y;

    it("should use custom equality function to prevent unnecessary updates", async () => {
      const schem = await createSchem({ position: { x: 1, y: 2 } });
      let renderCount = 0;

      const [useSelectNested] = Flux.createSelector<SelectKeyArgs, xy.XY>({
        subscribe,
        select: ({ client, args: { key } }) => ({ ...getPosition(client, key) }),
        equal: shallowEqual,
      });

      const { result } = renderHook(
        () => {
          renderCount++;
          return useSelectNested({ key: schem.key });
        },
        { wrapper },
      );

      const countAfterMount = renderCount;
      const firstRef = result.current;

      await act(async () => await rename(schem.key, "Bob"));

      expect(renderCount).toBe(countAfterMount);
      expect(result.current).toBe(firstRef);
    });

    it("should update when selected value actually changes per custom equality", async () => {
      const schem = await createSchem({ position: { x: 1, y: 2 } });

      const [useSelectNested] = Flux.createSelector<SelectKeyArgs, xy.XY>({
        subscribe,
        select: ({ client, args: { key } }) => ({ ...getPosition(client, key) }),
        equal: shallowEqual,
      });

      const { result } = renderHook(() => useSelectNested({ key: schem.key }), {
        wrapper,
      });
      expect(result.current).toEqual({ x: 1, y: 2 });

      await act(async () => await moveN1(schem.key, { x: 99, y: 2 }));

      expect(result.current).toEqual({ x: 99, y: 2 });
    });

    it("should skip re-render when primitive selection is unchanged", async () => {
      const schem = await createSchem({ position: { x: 5, y: 0 } });
      let renderCount = 0;

      const [useSelectX] = Flux.createSelector<SelectKeyArgs, number>({
        subscribe,
        select: ({ client, args: { key } }) => getPosition(client, key).x,
      });

      const { result } = renderHook(
        () => {
          renderCount++;
          return useSelectX({ key: schem.key });
        },
        { wrapper },
      );

      expect(result.current).toBe(5);
      const countAfterMount = renderCount;

      await act(async () => await rename(schem.key, "Bob"));

      expect(result.current).toBe(5);
      expect(renderCount).toBe(countAfterMount);

      await act(async () => await moveN1(schem.key, { x: 10, y: 0 }));

      expect(result.current).toBe(10);
    });
  });

  describe("derived selection stability", () => {
    it("should not infinite loop when select returns a new object reference from unchanged data", async () => {
      const schem = await createSchem({ name: "Alice" });
      let renderCount = 0;

      const [useSelectDerived] = Flux.createSelector<
        SelectKeyArgs,
        { name: string; count: number }
      >({
        subscribe,
        select: ({ client, args: { key } }) => {
          const doc = getSchem(client, key);
          return { name: doc?.name ?? "", count: doc?.nodes.length ?? 0 };
        },
      });

      const { result } = renderHook(
        () => {
          renderCount++;
          return useSelectDerived({ key: schem.key });
        },
        { wrapper },
      );

      expect(result.current).toEqual({ name: "Alice", count: 1 });
      expect(renderCount).toBeLessThan(5);
    });

    it("should still update derived selections when cached data changes", async () => {
      const schem = await createSchem({ name: "Alice" });

      const [useSelectDerived] = Flux.createSelector<
        SelectKeyArgs,
        { name: string; count: number }
      >({
        subscribe,
        select: ({ client, args: { key } }) => {
          const doc = getSchem(client, key);
          return { name: doc?.name ?? "", count: doc?.nodes.length ?? 0 };
        },
      });

      const { result } = renderHook(() => useSelectDerived({ key: schem.key }), {
        wrapper,
      });

      expect(result.current).toEqual({ name: "Alice", count: 1 });

      await act(async () => {
        await client.schematics.dispatch(schem.key, [
          schematic.rename({ name: "Bob" }),
          schematic.setNode({ node: { key: "n2", position: { x: 3, y: 3 } } }),
        ]);
      });

      expect(result.current).toEqual({ name: "Bob", count: 2 });
    });
  });

  describe("args memoization", () => {
    it("should not re-subscribe when args are deep-equal but referentially different", async () => {
      const subscribeSpy = vi.fn(subscribe);

      const [useSelectName] = Flux.createSelector<SelectKeyArgs, string>({
        subscribe: subscribeSpy,
        select: ({ client, args: { key } }) => getSchem(client, key)?.name ?? "",
      });

      const key = uuid.create();
      const { rerender } = renderHook(({ k }) => useSelectName({ key: k }), {
        wrapper,
        initialProps: { k: key },
      });

      const callCountAfterMount = subscribeSpy.mock.calls.length;

      rerender({ k: key });

      expect(subscribeSpy.mock.calls.length).toBe(callCountAfterMount);
    });

    it("should re-subscribe when args actually change", async () => {
      const subscribeSpy = vi.fn(subscribe);

      const [useSelectName] = Flux.createSelector<SelectKeyArgs, string>({
        subscribe: subscribeSpy,
        select: ({ client, args: { key } }) => getSchem(client, key)?.name ?? "",
      });

      const { rerender } = renderHook(({ k }) => useSelectName({ key: k }), {
        wrapper,
        initialProps: { k: uuid.create() },
      });

      const callCountAfterMount = subscribeSpy.mock.calls.length;

      rerender({ k: uuid.create() });

      expect(subscribeSpy.mock.calls.length).toBeGreaterThan(callCountAfterMount);
    });
  });

  describe("transform", () => {
    it("should apply transform to the raw selection", async () => {
      const schem = await createSchem({ position: { x: 1, y: 2 } });

      const [useSelectSum] = Flux.createSelector<SelectKeyArgs, number, xy.XY>({
        subscribe,
        select: ({ client, args: { key } }) => getPosition(client, key),
        transform: (raw) => raw.x + raw.y,
      });

      const { result } = renderHook(() => useSelectSum({ key: schem.key }), {
        wrapper,
      });
      expect(result.current).toBe(3);
    });

    it("should not re-run transform when the raw selection reference is unchanged", async () => {
      const schem = await createSchem({ position: { x: 1, y: 2 } });

      const transform = vi.fn((raw: xy.XY) => ({ sum: raw.x + raw.y }));
      const [useSelectNested] = Flux.createSelector<
        SelectKeyArgs,
        { sum: number },
        xy.XY
      >({
        subscribe,
        select: ({ client, args: { key } }) => getPosition(client, key),
        transform,
      });

      const { result } = renderHook(() => useSelectNested({ key: schem.key }), {
        wrapper,
      });
      // Sync the query cache with the record store so the next unrelated
      // change preserves the raw position reference via structural sharing.
      await act(async () => await moveN1(schem.key, { x: 1, y: 2 }));
      const firstOut = result.current;
      const callsAfterSync = transform.mock.calls.length;

      await act(async () => await rename(schem.key, "Bob"));

      expect(transform.mock.calls.length).toBe(callsAfterSync);
      expect(result.current).toBe(firstOut);
    });

    it("should re-run transform when the raw selection reference changes", async () => {
      const schem = await createSchem({ position: { x: 1, y: 2 } });

      const transform = vi.fn((raw: xy.XY) => raw.x + raw.y);
      const [useSelectSum] = Flux.createSelector<SelectKeyArgs, number, xy.XY>({
        subscribe,
        select: ({ client, args: { key } }) => getPosition(client, key),
        transform,
      });

      const { result } = renderHook(() => useSelectSum({ key: schem.key }), {
        wrapper,
      });
      expect(result.current).toBe(3);

      await act(async () => await moveN1(schem.key, { x: 10, y: 5 }));

      expect(result.current).toBe(15);
    });

    it("should re-run transform when an arg it depends on changes but the raw selection is unchanged", async () => {
      const schem = await createSchem({ position: { x: 2, y: 3 } });

      const [useSelectScaled] = Flux.createSelector<
        SelectKeyArgs & { multiplier: number },
        number,
        xy.XY
      >({
        subscribe,
        select: ({ client, args: { key } }) => getPosition(client, key),
        transform: (raw, { multiplier }) => (raw.x + raw.y) * multiplier,
      });

      const { result, rerender } = renderHook(
        ({ multiplier }) => useSelectScaled({ key: schem.key, multiplier }),
        { wrapper, initialProps: { multiplier: 1 } },
      );
      expect(result.current).toBe(5);

      rerender({ multiplier: 2 });

      expect(result.current).toBe(10);
    });

    it("should not re-run transform when args are deep-equal but referentially different", async () => {
      const schem = await createSchem({ position: { x: 2, y: 3 } });

      const transform = vi.fn(
        (raw: xy.XY, { multiplier }: { multiplier: number }) =>
          (raw.x + raw.y) * multiplier,
      );
      const [useSelectScaled] = Flux.createSelector<
        SelectKeyArgs & { multiplier: number },
        number,
        xy.XY
      >({
        subscribe,
        select: ({ client, args: { key } }) => getPosition(client, key),
        transform,
      });

      const { result, rerender } = renderHook(
        ({ multiplier }) => useSelectScaled({ key: schem.key, multiplier }),
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
      const unsubscribe = vi.fn();

      const [useSelectName] = Flux.createSelector<SelectKeyArgs, string>({
        subscribe: (_params, _notify) => unsubscribe,
        select: () => "",
      });

      const { unmount } = renderHook(() => useSelectName({ key: uuid.create() }), {
        wrapper,
      });

      expect(unsubscribe).not.toHaveBeenCalled();

      unmount();

      expect(unsubscribe).toHaveBeenCalled();
    });
  });

  describe("useGet", () => {
    it("should return the current value from the cache when invoked", async () => {
      const schem = await createSchem({ name: "Alice" });

      const [, useGetName] = Flux.createSelector<SelectKeyArgs, string>({
        subscribe,
        select: ({ client, args: { key } }) => getSchem(client, key)?.name ?? "",
      });

      const { result } = renderHook(() => useGetName(), { wrapper });
      expect(result.current({ key: schem.key })).toBe("Alice");
    });

    it("should read the latest value on each call without re-rendering", async () => {
      const schem = await createSchem({ name: "Alice" });
      // Keep the query mounted so cache writes stay visible to getCached.
      const off = client.schematics.onChange({ key: schem.key }, () => {});
      try {
        let renderCount = 0;
        const [, useGetName] = Flux.createSelector<SelectKeyArgs, string>({
          subscribe,
          select: ({ client, args: { key } }) => getSchem(client, key)?.name ?? "",
        });

        const { result } = renderHook(
          () => {
            renderCount++;
            return useGetName();
          },
          { wrapper },
        );

        expect(result.current({ key: schem.key })).toBe("Alice");
        const countAfterMount = renderCount;

        await act(async () => await rename(schem.key, "Bob"));

        expect(result.current({ key: schem.key })).toBe("Bob");
        expect(renderCount).toBe(countAfterMount);
      } finally {
        off();
      }
    });

    it("should never subscribe to the cache", () => {
      const subscribeSpy = vi.fn(subscribe);

      const [, useGetName] = Flux.createSelector<SelectKeyArgs, string>({
        subscribe: subscribeSpy,
        select: ({ client, args: { key } }) => getSchem(client, key)?.name ?? "",
      });

      const { result } = renderHook(() => useGetName(), { wrapper });
      result.current({ key: uuid.create() });

      expect(subscribeSpy).not.toHaveBeenCalled();
    });

    it("should apply the transform to the raw selection", async () => {
      const schem = await createSchem({ position: { x: 1, y: 2 } });

      const [, useGetSum] = Flux.createSelector<SelectKeyArgs, number, xy.XY>({
        subscribe,
        select: ({ client, args: { key } }) => getPosition(client, key),
        transform: (raw) => raw.x + raw.y,
      });

      const { result } = renderHook(() => useGetSum(), { wrapper });
      expect(result.current({ key: schem.key })).toBe(3);
    });

    it("should return a stable getter across re-renders with equal args", () => {
      const [, useGetName] = Flux.createSelector<SelectKeyArgs, string>({
        subscribe,
        select: ({ client, args: { key } }) => getSchem(client, key)?.name ?? "",
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
