// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, type ranger } from "@synnaxlabs/client";
import { type record, TimeRange, TimeSpan, uuid } from "@synnaxlabs/x";
import { renderHook, waitFor } from "@testing-library/react";
import { act } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Flux } from "@/flux";
import { type ranger as aetherRanger } from "@/ranger/aether";
import { createSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();
const wrapper = createSynnaxWrapper({ client });

describe("list", () => {
  let controller: AbortController;
  beforeEach(() => {
    controller = new AbortController();
  });
  afterEach(() => {
    controller.abort();
  });
  describe("initial list", () => {
    it("should return a loading result as its initial state", () => {
      const { result } = renderHook(
        () =>
          Flux.createList({
            name: "Resource",
            retrieve: async () => [],
            retrieveByKey: async () => ({ key: 12 }),
          })(),
        { wrapper },
      );
      expect(result.current.variant).toEqual("loading");
      expect(result.current.data).toEqual([]);
    });

    it("should return a success result when the list is retrieved", async () => {
      const retrieve = vi.fn().mockResolvedValue([{ key: 1 }, { key: 2 }]);
      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, record.Keyed<number>>({
            name: "Resource",
            retrieve,
            retrieveByKey: async () => ({ key: 12 }),
          })(),
        { wrapper },
      );
      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });
      await waitFor(() => {
        expect(retrieve).toHaveBeenCalledTimes(1);
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toEqual([1, 2]);
      });
    });

    it("should return an error result when the query fails to execute", async () => {
      const retrieve = vi.fn().mockRejectedValue(new Error("Test Error"));
      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, record.Keyed<number>>({
            name: "Resource",
            retrieve,
            retrieveByKey: async () => ({ key: 12 }),
          })(),
        { wrapper },
      );
      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });
      await waitFor(() => {
        expect(retrieve).toHaveBeenCalledTimes(1);
        expect(result.current.variant).toEqual("error");
        expect(result.current.status.description).toEqual("Test Error");
      });
    });
  });

  describe("filter", () => {
    it("should allow the caller to provide a custom filter function", async () => {
      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, record.Keyed<number>>({
            name: "Resource",
            retrieve: async () => [{ key: 1 }, { key: 2 }],
            retrieveByKey: async ({ key }) => ({ key }),
          })({ filter: (item) => item.key === 1 }),
        { wrapper },
      );
      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });
      await waitFor(() => {
        expect(result.current.data).toEqual([1]);
      });
    });

    it("should respect the filter function when retrieving a list item", async () => {
      const { result } = renderHook(
        () => {
          const result = Flux.createList<{}, number, record.Keyed<number>>({
            name: "Resource",
            retrieve: async () => [{ key: 1 }, { key: 2 }],
            retrieveByKey: async ({ key }) => ({ key }),
          })({ filter: (item) => item.key === 1 });
          const value = Flux.useListItem<number, record.Keyed<number>>({
            subscribe: result.subscribe,
            getItem: result.getItem,
            key: 2,
          });
          return { ...result, value };
        },
        { wrapper },
      );
      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });
      await waitFor(() => {
        expect(result.current.value).toEqual(undefined);
      });
    });
  });

  describe("sort", () => {
    interface TestItem extends record.Keyed<number> {
      key: number;
      value: string;
      priority: number;
    }

    it("should sort items in ascending order by default", async () => {
      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, TestItem>({
            name: "Resource",
            retrieve: async () => [
              { key: 3, value: "c", priority: 3 },
              { key: 1, value: "a", priority: 1 },
              { key: 2, value: "b", priority: 2 },
            ],
            retrieveByKey: async ({ key }) => ({
              key,
              value: `item-${key}`,
              priority: key,
            }),
          })({ sort: (a, b) => a.key - b.key }),
        { wrapper },
      );
      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });
      await waitFor(() => {
        expect(result.current.data).toEqual([1, 2, 3]);
      });
    });

    it("should sort items in descending order", async () => {
      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, TestItem>({
            name: "Resource",
            retrieve: async () => [
              { key: 1, value: "a", priority: 1 },
              { key: 3, value: "c", priority: 3 },
              { key: 2, value: "b", priority: 2 },
            ],
            retrieveByKey: async ({ key }) => ({
              key,
              value: `item-${key}`,
              priority: key,
            }),
          })({ sort: (a, b) => b.key - a.key }),
        { wrapper },
      );
      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });
      await waitFor(() => {
        expect(result.current.data).toEqual([3, 2, 1]);
      });
    });

    it("should sort by a custom property", async () => {
      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, TestItem>({
            name: "Resource",
            retrieve: async () => [
              { key: 1, value: "zebra", priority: 1 },
              { key: 2, value: "apple", priority: 2 },
              { key: 3, value: "banana", priority: 3 },
            ],
            retrieveByKey: async ({ key }) => ({
              key,
              value: `item-${key}`,
              priority: key,
            }),
          })({ sort: (a, b) => a.value.localeCompare(b.value) }),
        { wrapper },
      );
      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });
      await waitFor(() => {
        expect(result.current.data).toEqual([2, 3, 1]);
      });
    });

    it("should combine sorting with filtering", async () => {
      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, TestItem>({
            name: "Resource",
            retrieve: async () => [
              { key: 1, value: "a", priority: 1 },
              { key: 2, value: "b", priority: 2 },
              { key: 3, value: "c", priority: 3 },
              { key: 4, value: "d", priority: 4 },
            ],
            retrieveByKey: async ({ key }) => ({
              key,
              value: `item-${key}`,
              priority: key,
            }),
          })({
            filter: (item) => item.key % 2 === 0, // Even keys only
            sort: (a, b) => b.key - a.key, // Descending order
          }),
        { wrapper },
      );
      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });
      await waitFor(() => {
        expect(result.current.data).toEqual([4, 2]);
      });
    });

    it("should maintain sort order when appending new items", async () => {
      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, TestItem>({
            name: "Resource",
            retrieve: async () => [
              { key: 1, value: "a", priority: 1 },
              { key: 3, value: "c", priority: 3 },
            ],
            retrieveByKey: async ({ key }) => ({
              key,
              value: `item-${key}`,
              priority: key,
            }),
          })({ sort: (a, b) => a.key - b.key }),
        { wrapper },
      );

      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });

      await waitFor(() => {
        expect(result.current.data).toEqual([1, 3]);
      });

      // Append more items
      act(() => {
        result.current.retrieve({}, { signal: controller.signal, mode: "append" });
      });

      await waitFor(() => {
        expect(result.current.data).toEqual([1, 3]);
      });
    });

    it("should work without a sort function", async () => {
      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, TestItem>({
            name: "Resource",
            retrieve: async () => [
              { key: 3, value: "c", priority: 3 },
              { key: 1, value: "a", priority: 1 },
              { key: 2, value: "b", priority: 2 },
            ],
            retrieveByKey: async ({ key }) => ({
              key,
              value: `item-${key}`,
              priority: key,
            }),
          })(), // No sort function provided
        { wrapper },
      );
      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });
      await waitFor(() => {
        expect(result.current.data).toEqual([3, 1, 2]); // Original order maintained
      });
    });

    it("should optimize updates when sort position doesn't change", async () => {
      interface TestItemWithPriority extends record.Keyed<number> {
        key: number;
        name: string;
        priority: number; // Used for sorting
        description: string; // Not used for sorting
      }

      // Mock data that will be updated
      let mockItem2 = { key: 2, name: "Item 2", priority: 2, description: "Original" };

      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, TestItemWithPriority>({
            name: "Resource",
            retrieve: async () => [
              { key: 1, name: "Item 1", priority: 1, description: "Original" },
              mockItem2, // This will change during the test
              { key: 3, name: "Item 3", priority: 3, description: "Original" },
            ],
            retrieveByKey: async ({ key }) => ({
              key,
              name: `Item ${key}`,
              priority: key,
              description: "Retrieved",
            }),
          })({ sort: (a, b) => a.priority - b.priority }),
        { wrapper },
      );

      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });

      await waitFor(() => {
        expect(result.current.data).toEqual([1, 2, 3]);
      });

      const initialDataRef = result.current.data;

      // Update the item with same priority (no position change)
      mockItem2 = { key: 2, name: "Item 2", priority: 2, description: "Updated" };

      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });

      await waitFor(() => {
        // Data reference should be the same since sort position didn't change
        expect(result.current.data).toBe(initialDataRef);
        expect(result.current.data).toEqual([1, 2, 3]);
      });

      // Now update with different priority (position should change)
      mockItem2 = { key: 2, name: "Item 2", priority: 4, description: "Updated" };

      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });

      await waitFor(() => {
        // Data reference should be different since sort position changed
        expect(result.current.data).not.toBe(initialDataRef);
        expect(result.current.data).toEqual([1, 3, 2]); // Item 2 moved to end
      });
    });
  });

  describe("useListItem", () => {
    it("should return a pre-retrieved list item", async () => {
      const { result } = renderHook(
        () => {
          const { retrieve, subscribe, getItem } = Flux.createList<
            {},
            number,
            record.Keyed<number>
          >({
            name: "Resource",
            retrieve: async () => [{ key: 1 }, { key: 2 }],
            retrieveByKey: async ({ key }) => ({ key }),
          })();
          const value = Flux.useListItem<number, record.Keyed<number>>({
            subscribe,
            getItem,
            key: 1,
          });
          return { retrieve, value };
        },
        { wrapper },
      );
      await waitFor(() => {
        expect(result.current.value).toEqual({ key: 1 });
      });
    });

    it("should move the query to an error state when the retrieveByKey fails to execute", async () => {
      const retrieveMock = vi.fn().mockResolvedValue([{ key: 1 }, { key: 2 }]);
      const retrieveByKeyMock = vi.fn().mockRejectedValue(new Error("Test Error"));
      const { result } = renderHook(
        () => {
          const result = Flux.createList<{}, number, record.Keyed<number>>({
            name: "Resource",
            retrieve: retrieveMock,
            retrieveByKey: retrieveByKeyMock,
          })();
          const value = Flux.useListItem<number, record.Keyed<number>>({
            subscribe: result.subscribe,
            getItem: result.getItem,
            key: 1,
          });
          return { ...result, value };
        },
        { wrapper },
      );
      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("error");
        expect(result.current.status.description).toEqual("Test Error");
      });
    });
  });

  interface SubStore extends Flux.Store {
    ranges: aetherRanger.FluxStore;
  }

  describe("listeners", () => {
    it("should correctly update a list item when the listener changes", async () => {
      const rng = await client.ranges.create({
        name: "Test Range",
        timeRange: new TimeRange({
          start: TimeSpan.seconds(12),
          end: TimeSpan.seconds(13),
        }),
      });

      const { result } = renderHook(
        () => {
          const { getItem, subscribe, retrieve } = Flux.createList<
            {},
            ranger.Key,
            ranger.Payload,
            SubStore
          >({
            name: "Resource",
            retrieve: async ({ client }) => [await client.ranges.retrieve(rng.key)],
            retrieveByKey: async ({ client, key }) => await client.ranges.retrieve(key),
            mountListeners: ({ store, onChange }) =>
              store.ranges.onSet((changed) => onChange(changed.key, () => changed)),
          })();
          const value = Flux.useListItem<ranger.Key, ranger.Payload>({
            subscribe,
            getItem,
            key: rng.key,
          });
          return { retrieve, value };
        },
        { wrapper },
      );

      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });

      await waitFor(() => {
        expect(result.current.value?.name).toEqual("Test Range");
      });

      await act(async () => await client.ranges.rename(rng.key, "Test Range 2"));

      await waitFor(() => {
        expect(result.current.value?.name).toEqual("Test Range 2");
      });
    });

    it("should correctly remove a list item when it gets deleted", async () => {
      const rng = await client.ranges.create({
        name: "Test Range",
        timeRange: new TimeRange({
          start: TimeSpan.seconds(12),
          end: TimeSpan.seconds(13),
        }),
      });
      const { result } = renderHook(
        () => {
          const { getItem, retrieveAsync } = Flux.createList<
            {},
            ranger.Key,
            ranger.Payload,
            SubStore
          >({
            name: "Resource",
            retrieve: async ({ client }) => [await client.ranges.retrieve(rng.key)],
            retrieveByKey: async ({ client, key }) => await client.ranges.retrieve(key),
            mountListeners: ({ store, onDelete }) =>
              store.ranges.onDelete(async (changed) => onDelete(changed)),
          })();
          return { retrieveAsync, value: getItem(rng.key) };
        },
        { wrapper },
      );

      await act(
        async () =>
          await result.current.retrieveAsync({}, { signal: controller.signal }),
      );
      await waitFor(() => {
        expect(result.current.value?.name).toEqual("Test Range");
      });
      await act(async () => await client.ranges.delete(rng.key));
      await waitFor(() => expect(result.current.value?.key).not.toEqual(rng.key));
    });

    it("should maintain sort order when items are updated through listeners", async () => {
      const rng1 = await client.ranges.create({
        name: "B Range",
        timeRange: new TimeRange({
          start: TimeSpan.seconds(10),
          end: TimeSpan.seconds(11),
        }),
      });

      const rng2 = await client.ranges.create({
        name: "A Range",
        timeRange: new TimeRange({
          start: TimeSpan.seconds(12),
          end: TimeSpan.seconds(13),
        }),
      });

      const { result } = renderHook(
        () =>
          Flux.createList<{}, ranger.Key, ranger.Payload, SubStore>({
            name: "Resource",
            retrieve: async ({ client }) => [
              await client.ranges.retrieve(rng1.key),
              await client.ranges.retrieve(rng2.key),
            ],
            retrieveByKey: async ({ client, key }) => await client.ranges.retrieve(key),
            mountListeners: ({ store, onChange }) =>
              store.ranges.onSet((changed) => onChange(changed.key, () => changed)),
          })({ sort: (a, b) => a.name.localeCompare(b.name) }),
        { wrapper },
      );

      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });

      await waitFor(() => {
        const indexOfRng1 = result.current.data.indexOf(rng1.key);
        const indexOfRng2 = result.current.data.indexOf(rng2.key);
        expect(indexOfRng2).toBeLessThan(indexOfRng1);
      });

      await act(async () => await client.ranges.rename(rng1.key, "Z Range"));

      await waitFor(() => {
        const indexOfRng1 = result.current.data.indexOf(rng1.key);
        const indexOfRng2 = result.current.data.indexOf(rng2.key);
        expect(indexOfRng2).toBeLessThan(indexOfRng1);
      });
    });

    it("should insert new items in correct sorted position through listeners", async () => {
      const rng1 = await client.ranges.create({
        name: "A Range",
        timeRange: new TimeRange({
          start: TimeSpan.seconds(10),
          end: TimeSpan.seconds(11),
        }),
      });

      const rng2 = await client.ranges.create({
        name: "C Range",
        timeRange: new TimeRange({
          start: TimeSpan.seconds(12),
          end: TimeSpan.seconds(13),
        }),
      });
      const rng3Key = uuid.create();
      const rangeKeys = new Set([rng1.key, rng2.key, rng3Key]);

      const { result } = renderHook(
        () =>
          Flux.createList<{}, ranger.Key, ranger.Payload, SubStore>({
            name: "Resource",
            retrieve: async ({ client }) => [
              await client.ranges.retrieve(rng1.key),
              await client.ranges.retrieve(rng2.key),
            ],
            retrieveByKey: async ({ client, key }) => await client.ranges.retrieve(key),
            mountListeners: ({ store, onChange }) =>
              store.ranges.onSet((changed) => {
                if (rangeKeys.has(changed.key)) onChange(changed.key, changed);
              }),
          })({ sort: (a, b) => a.name.localeCompare(b.name) }),
        { wrapper },
      );

      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });

      await waitFor(() => {
        expect(result.current.data).toEqual([rng1.key, rng2.key]); // A Range, C Range
      });

      const rng3 = await client.ranges.create({
        key: rng3Key,
        name: "B Range",
        timeRange: new TimeRange({
          start: TimeSpan.seconds(14),
          end: TimeSpan.seconds(15),
        }),
      });

      await waitFor(() => {
        expect(result.current.data).toEqual([rng1.key, rng3.key, rng2.key]); // A Range, B Range, C Range
      });
    });
  });
});
