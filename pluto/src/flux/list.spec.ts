// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient, type ranger } from "@synnaxlabs/client";
import { type record, testutil, TimeRange, TimeSpan, uuid } from "@synnaxlabs/x";
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

    it("should return undefined for zero value keys", async () => {
      const { result } = renderHook(
        () => {
          const { retrieve, getItem } = Flux.createList<
            {},
            number,
            record.Keyed<number>
          >({
            name: "Resource",
            retrieve: async () => [{ key: 1 }, { key: 2 }],
            retrieveByKey: async ({ key }) => ({ key }),
          })();
          // Test various zero values
          const zeroValue = getItem(0);
          const nullValue = getItem(null as unknown as number);
          const undefinedValue = getItem(undefined as unknown as number);
          const emptyStringValue = getItem("" as unknown as number);
          return { retrieve, zeroValue, nullValue, undefinedValue, emptyStringValue };
        },
        { wrapper },
      );

      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });

      await waitFor(() => {
        expect(result.current.zeroValue).toBeUndefined();
        expect(result.current.nullValue).toBeUndefined();
        expect(result.current.undefinedValue).toBeUndefined();
        expect(result.current.emptyStringValue).toBeUndefined();
      });
    });

    it("should return undefined for zero value keys with string keys", async () => {
      const { result } = renderHook(
        () => {
          const { retrieve, getItem } = Flux.createList<
            {},
            string,
            record.Keyed<string>
          >({
            name: "Resource",
            retrieve: async () => [{ key: "key1" }, { key: "key2" }],
            retrieveByKey: async ({ key }) => ({ key }),
          })();
          // Test various zero values for string keys
          const emptyStringValue = getItem("");
          const nullValue = getItem(null as unknown as string);
          const undefinedValue = getItem(undefined as unknown as string);
          return { retrieve, emptyStringValue, nullValue, undefinedValue };
        },
        { wrapper },
      );

      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });

      await waitFor(() => {
        expect(result.current.emptyStringValue).toBeUndefined();
        expect(result.current.nullValue).toBeUndefined();
        expect(result.current.undefinedValue).toBeUndefined();
      });
    });

    it("should handle array of keys with zero values correctly", async () => {
      const { result } = renderHook(
        () => {
          const { retrieve, getItem } = Flux.createList<
            {},
            number,
            record.Keyed<number>
          >({
            name: "Resource",
            retrieve: async () => [{ key: 1 }, { key: 2 }, { key: 3 }],
            retrieveByKey: async ({ key }) => ({ key }),
          })();
          // Test array with mixed valid and zero values
          const mixedArray = getItem([1, 0, 2, null as unknown as number, 3]);
          const allZeroArray = getItem([
            0,
            null as unknown as number,
            undefined as unknown as number,
          ]);
          return { retrieve, mixedArray, allZeroArray };
        },
        { wrapper },
      );

      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });

      await waitFor(() => {
        // Should filter out zero values and return only valid items
        expect(result.current.mixedArray).toEqual([{ key: 1 }, { key: 2 }, { key: 3 }]);
        // All zero values should result in empty array
        expect(result.current.allZeroArray).toEqual([]);
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

  interface FluxStore extends Flux.Store {
    ranges: aetherRanger.FluxStore;
  }

  describe("retrieveCached", () => {
    it("should use cached data as initial state when available", () => {
      const cachedItems = [
        { key: 1, value: "cached-1" },
        { key: 2, value: "cached-2" },
      ];
      const retrieveCached = vi.fn().mockReturnValue(cachedItems);
      const retrieve = vi.fn().mockResolvedValue([
        { key: 1, value: "fresh-1" },
        { key: 2, value: "fresh-2" },
      ]);

      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, record.Keyed<number> & { value: string }>({
            name: "Resource",
            retrieve,
            retrieveByKey: async ({ key }) => ({ key, value: `item-${key}` }),
            retrieveCached,
          })(),
        { wrapper },
      );

      expect(result.current.variant).toEqual("loading");
      expect(result.current.data).toEqual([1, 2]);
      expect(retrieveCached).toHaveBeenCalledTimes(1);
    });

    it("should not use cached data when empty array is returned", () => {
      const retrieveCached = vi.fn().mockReturnValue([]);
      const retrieve = vi.fn().mockResolvedValue([{ key: 1 }, { key: 2 }]);

      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, record.Keyed<number>>({
            name: "Resource",
            retrieve,
            retrieveByKey: async ({ key }) => ({ key }),
            retrieveCached,
          })(),
        { wrapper },
      );

      expect(result.current.variant).toEqual("loading");
      expect(result.current.data).toEqual([]);
    });

    it("should apply filter to cached data", () => {
      const cachedItems = [
        { key: 1, value: "odd" },
        { key: 2, value: "even" },
        { key: 3, value: "odd" },
        { key: 4, value: "even" },
      ];
      const retrieveCached = vi.fn().mockReturnValue(cachedItems);

      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, record.Keyed<number> & { value: string }>({
            name: "Resource",
            retrieve: async () => [],
            retrieveByKey: async ({ key }) => ({ key, value: `item-${key}` }),
            retrieveCached,
          })({ filter: (item) => item.key % 2 === 0 }), // Only even keys
        { wrapper },
      );
      expect(result.current.data).toEqual([2, 4]);
    });

    it("should handle params correctly with cached retrieval", () => {
      interface TestParams {
        searchTerm?: string;
      }
      const cachedItems = [{ key: 1 }, { key: 2 }];
      const retrieveCached = vi.fn().mockReturnValue(cachedItems);

      renderHook(
        () =>
          Flux.createList<TestParams, number, record.Keyed<number>>({
            name: "Resource",
            retrieve: async () => [],
            retrieveByKey: async ({ key }) => ({ key }),
            retrieveCached,
          })({ initialQuery: { searchTerm: "test" } }),
        { wrapper },
      );

      expect(retrieveCached).toHaveBeenCalledWith({
        query: { searchTerm: "test" },
        store: expect.any(Object),
      });
    });

    it("should replace cached data when fresh data arrives", async () => {
      const cachedItems = [
        { key: 1, value: "cached-1" },
        { key: 2, value: "cached-2" },
      ];
      const freshItems = [
        { key: 1, value: "fresh-1" },
        { key: 2, value: "fresh-2" },
        { key: 3, value: "fresh-3" },
      ];
      const retrieveCached = vi.fn().mockReturnValue(cachedItems);
      const retrieve = vi.fn().mockResolvedValue(freshItems);

      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, record.Keyed<number> & { value: string }>({
            name: "Resource",
            retrieve,
            retrieveByKey: async ({ key }) => ({ key, value: `item-${key}` }),
            retrieveCached,
          })(),
        { wrapper },
      );

      // Initially should have cached data
      expect(result.current.data).toEqual([1, 2]);

      // Retrieve fresh data
      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });

      await waitFor(() => {
        expect(result.current.data).toEqual([1, 2, 3]);
        expect(result.current.getItem(3)?.value).toEqual("fresh-3");
      });
    });

    it("should work without retrieveCached defined", async () => {
      const retrieve = vi.fn().mockResolvedValue([{ key: 1 }, { key: 2 }]);

      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, record.Keyed<number>>({
            name: "Resource",
            retrieve,
            retrieveByKey: async ({ key }) => ({ key }),
            // No retrieveCached provided
          })(),
        { wrapper },
      );

      // Should start with empty data
      expect(result.current.variant).toEqual("loading");
      expect(result.current.data).toEqual([]);

      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });

      await waitFor(() => {
        expect(result.current.data).toEqual([1, 2]);
      });
    });

    it("should apply sort to cached data", () => {
      interface TestItem extends record.Keyed<number> {
        key: number;
        priority: number;
      }

      const cachedItems: TestItem[] = [
        { key: 3, priority: 3 },
        { key: 1, priority: 1 },
        { key: 2, priority: 2 },
      ];
      const retrieveCached = vi.fn().mockReturnValue(cachedItems);

      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, TestItem>({
            name: "Resource",
            retrieve: async () => [],
            retrieveByKey: async ({ key }) => ({ key, priority: key }),
            retrieveCached,
          })({ sort: (a, b) => a.priority - b.priority }),
        { wrapper },
      );

      // Should apply sort to cached data
      expect(result.current.data).toEqual([1, 2, 3]);
    });

    it("should apply sort in descending order to cached data", () => {
      interface TestItem extends record.Keyed<number> {
        key: number;
        name: string;
      }

      const cachedItems: TestItem[] = [
        { key: 1, name: "Alpha" },
        { key: 2, name: "Charlie" },
        { key: 3, name: "Bravo" },
      ];
      const retrieveCached = vi.fn().mockReturnValue(cachedItems);

      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, TestItem>({
            name: "Resource",
            retrieve: async () => [],
            retrieveByKey: async ({ key }) => ({ key, name: `item-${key}` }),
            retrieveCached,
          })({ sort: (a, b) => b.name.localeCompare(a.name) }), // Descending order
        { wrapper },
      );

      // Should sort cached data by name in descending order
      expect(result.current.data).toEqual([2, 3, 1]); // Charlie, Bravo, Alpha
    });

    it("should combine filter and sort with cached data", () => {
      interface TestItem extends record.Keyed<number> {
        key: number;
        value: number;
        active: boolean;
      }

      const cachedItems: TestItem[] = [
        { key: 1, value: 100, active: true },
        { key: 2, value: 50, active: false },
        { key: 3, value: 75, active: true },
        { key: 4, value: 25, active: true },
      ];
      const retrieveCached = vi.fn().mockReturnValue(cachedItems);

      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, TestItem>({
            name: "Resource",
            retrieve: async () => [],
            retrieveByKey: async ({ key }) => ({ key, value: key * 10, active: true }),
            retrieveCached,
          })({
            filter: (item) => item.active,
            sort: (a, b) => a.value - b.value,
          }),
        { wrapper },
      );

      // Should filter for active items and sort by value
      expect(result.current.data).toEqual([4, 3, 1]); // 25, 75, 100
    });
  });

  describe("listener synchronization", () => {
    it("should mount listeners on first retrieve", async () => {
      const mountListeners = vi.fn();
      const retrieve = vi.fn().mockResolvedValue([{ key: 1 }]);

      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, record.Keyed<number>, FluxStore>({
            name: "Resource",
            retrieve,
            retrieveByKey: async ({ key }) => ({ key }),
            mountListeners,
          })(),
        { wrapper },
      );

      expect(mountListeners).not.toHaveBeenCalled();

      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });

      await waitFor(() => {
        expect(mountListeners).toHaveBeenCalledTimes(1);
      });
    });

    it("should mount listeners when retrieving single item before list", async () => {
      const mountListeners = vi.fn();
      const retrieveByKey = vi.fn().mockResolvedValue({ key: 1 });

      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, record.Keyed<number>, FluxStore>({
            name: "Resource",
            retrieve: async () => [],
            retrieveByKey,
            mountListeners,
          })(),
        { wrapper },
      );

      expect(mountListeners).not.toHaveBeenCalled();

      act(() => {
        result.current.getItem(1);
      });

      await waitFor(() => {
        expect(mountListeners).toHaveBeenCalledTimes(1);
        expect(retrieveByKey).toHaveBeenCalled();
      });
    });

    it("should not remount listeners on subsequent calls to getItem", async () => {
      const mountListeners = vi.fn();
      const retrieveByKey = vi.fn().mockResolvedValue({ key: 1 });

      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, record.Keyed<number>, FluxStore>({
            name: "Resource",
            retrieve: async () => [],
            retrieveByKey,
            mountListeners,
          })(),
        { wrapper },
      );
      act(() => {
        result.current.getItem(1);
      });
      await waitFor(() => {
        expect(mountListeners).toHaveBeenCalledTimes(1);
      });
      act(() => {
        result.current.getItem(1);
      });
      await testutil.expectAlways(() => {
        expect(mountListeners).toHaveBeenCalledTimes(1);
      });
    });

    it("should not remount listeners when getItem is called AFTER retrieve", async () => {
      const mountListeners = vi.fn();
      const retrieve = vi.fn().mockResolvedValue([{ key: 1 }]);
      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, record.Keyed<number>, FluxStore>({
            name: "Resource",
            retrieve,
            retrieveByKey: async ({ key }) => ({ key }),
            mountListeners,
          })(),
        { wrapper },
      );
      await act(async () => {
        await result.current.retrieveAsync({}, { signal: controller.signal });
      });
      await waitFor(() => {
        expect(mountListeners).toHaveBeenCalledTimes(1);
      });
      act(() => {
        result.current.getItem(1);
      });
      await testutil.expectAlways(() => {
        expect(mountListeners).toHaveBeenCalledTimes(1);
      });
    });

    it("should remount listeners on subsequent retrieves", async () => {
      const mountListeners = vi.fn();
      const retrieve = vi.fn().mockResolvedValue([{ key: 1 }]);

      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, record.Keyed<number>, FluxStore>({
            name: "Resource",
            retrieve,
            retrieveByKey: async ({ key }) => ({ key }),
            mountListeners,
          })(),
        { wrapper },
      );

      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });

      await waitFor(() => {
        expect(mountListeners).toHaveBeenCalledTimes(1);
      });

      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });

      await waitFor(() => {
        expect(mountListeners).toHaveBeenCalledTimes(2);
      });
    });

    it("should pass correct params to mountListeners", async () => {
      interface TestParams {
        filter?: string;
      }
      const mountListeners = vi.fn();
      const retrieve = vi.fn().mockResolvedValue([{ key: 1 }]);

      const { result } = renderHook(
        () =>
          Flux.createList<TestParams, number, record.Keyed<number>, FluxStore>({
            name: "Resource",
            retrieve,
            retrieveByKey: async ({ key }) => ({ key }),
            mountListeners,
          })({ initialQuery: { filter: "active" } }),
        { wrapper },
      );

      act(() => {
        result.current.getItem(1);
      });

      await waitFor(() => {
        const firstCall = mountListeners.mock.calls[0];
        expect(firstCall[0].query).toEqual({ filter: "active" });
      });
    });

    it("should not mount listeners immediately when retrieveCached returns data", () => {
      const mountListeners = vi.fn();
      const cachedItems = [{ key: 1 }, { key: 2 }];
      const retrieveCached = vi.fn().mockReturnValue(cachedItems);

      renderHook(
        () =>
          Flux.createList<{}, number, record.Keyed<number>, FluxStore>({
            name: "Resource",
            retrieve: async () => [],
            retrieveByKey: async ({ key }) => ({ key }),
            retrieveCached,
            mountListeners,
          })(),
        { wrapper },
      );

      expect(mountListeners).not.toHaveBeenCalled();
    });

    it("should mount listeners when getItem is called before retrieve AND the result of getItem is cached", () => {
      const mountListeners = vi.fn();
      const useList = Flux.createList<{}, number, record.Keyed<number>, FluxStore>({
        name: "Resource",
        retrieve: async () => [],
        retrieveByKey: async ({ key }) => ({ key }),
        retrieveCached: () => [{ key: 1 }],
        mountListeners,
      });

      const { result } = renderHook(useList, { wrapper });
      act(() => {
        result.current.getItem(1);
      });
      expect(mountListeners).toHaveBeenCalledTimes(1);
    });
  });

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
            FluxStore
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

    it("should accept a keyed record as the argument to onChange", async () => {
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
            FluxStore
          >({
            name: "Resource",
            retrieve: async ({ client }) => [await client.ranges.retrieve(rng.key)],
            retrieveByKey: async ({ client, key }) => await client.ranges.retrieve(key),
            mountListeners: ({ store, onChange }) => store.ranges.onSet(onChange),
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
            FluxStore
          >({
            name: "Resource",
            retrieve: async ({ client }) => [await client.ranges.retrieve(rng.key)],
            retrieveByKey: async ({ client, key }) => await client.ranges.retrieve(key),
            mountListeners: ({ store, onDelete }) => store.ranges.onDelete(onDelete),
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
          Flux.createList<{}, ranger.Key, ranger.Payload, FluxStore>({
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
          Flux.createList<{}, ranger.Key, ranger.Payload, FluxStore>({
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
