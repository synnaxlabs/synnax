// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type query, type ranger, type Synnax as Client } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { type record, testutil, TimeRange, TimeSpan, uuid } from "@synnaxlabs/x";
import { renderHook, waitFor } from "@testing-library/react";
import { act, type PropsWithChildren, type ReactElement } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { aetherTest } from "@/aether/test";
import { Flux } from "@/flux";
import { status } from "@/status/aether";
import { Status } from "@/status/base";
import { Synnax } from "@/synnax";
import { synnax } from "@/synnax/aether";
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

  const changed = <E,>(data: E[]): query.Cached<E[]> => ({ variant: "changed", data });

  describe("getCached", () => {
    it("should use cached data as initial state when available", () => {
      const cachedItems = [
        { key: 1, value: "cached-1" },
        { key: 2, value: "cached-2" },
      ];
      const getCached = vi.fn().mockReturnValue(changed(cachedItems));
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
            getCached,
          })(),
        { wrapper },
      );

      expect(result.current.variant).toEqual("loading");
      expect(result.current.data).toEqual([1, 2]);
      expect(getCached).toHaveBeenCalledTimes(1);
    });

    it("should not use cached data when the cached answer is empty", () => {
      const getCached = vi.fn().mockReturnValue(changed([]));
      const retrieve = vi.fn().mockResolvedValue([{ key: 1 }, { key: 2 }]);

      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, record.Keyed<number>>({
            name: "Resource",
            retrieve,
            retrieveByKey: async ({ key }) => ({ key }),
            getCached,
          })(),
        { wrapper },
      );

      expect(result.current.variant).toEqual("loading");
      expect(result.current.data).toEqual([]);
    });

    it("should not use cached data when nothing is cached", () => {
      const getCached = vi.fn().mockReturnValue(undefined);

      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, record.Keyed<number>>({
            name: "Resource",
            retrieve: async () => [],
            retrieveByKey: async ({ key }) => ({ key }),
            getCached,
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
      const getCached = vi.fn().mockReturnValue(changed(cachedItems));

      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, record.Keyed<number> & { value: string }>({
            name: "Resource",
            retrieve: async () => [],
            retrieveByKey: async ({ key }) => ({ key, value: `item-${key}` }),
            getCached,
          })({ filter: (item) => item.key % 2 === 0 }), // Only even keys
        { wrapper },
      );
      expect(result.current.data).toEqual([2, 4]);
    });

    it("should handle params correctly with cached retrieval", () => {
      type TestParams = {
        searchTerm?: string;
      };
      const cachedItems = [{ key: 1 }, { key: 2 }];
      const getCached = vi.fn().mockReturnValue(changed(cachedItems));

      renderHook(
        () =>
          Flux.createList<TestParams, number, record.Keyed<number>>({
            name: "Resource",
            retrieve: async () => [],
            retrieveByKey: async ({ key }) => ({ key }),
            getCached,
          })({ initialQuery: { searchTerm: "test" } }),
        { wrapper },
      );

      expect(getCached).toHaveBeenCalledWith({
        query: { searchTerm: "test" },
        client: expect.any(Object),
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
      const getCached = vi.fn().mockReturnValue(changed(cachedItems));
      const retrieve = vi.fn().mockResolvedValue(freshItems);

      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, record.Keyed<number> & { value: string }>({
            name: "Resource",
            retrieve,
            retrieveByKey: async ({ key }) => ({ key, value: `item-${key}` }),
            getCached,
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

    it("should work without getCached defined", async () => {
      const retrieve = vi.fn().mockResolvedValue([{ key: 1 }, { key: 2 }]);

      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, record.Keyed<number>>({
            name: "Resource",
            retrieve,
            retrieveByKey: async ({ key }) => ({ key }),
            // No getCached provided
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
      const getCached = vi.fn().mockReturnValue(changed(cachedItems));

      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, TestItem>({
            name: "Resource",
            retrieve: async () => [],
            retrieveByKey: async ({ key }) => ({ key, priority: key }),
            getCached,
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
      const getCached = vi.fn().mockReturnValue(changed(cachedItems));

      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, TestItem>({
            name: "Resource",
            retrieve: async () => [],
            retrieveByKey: async ({ key }) => ({ key, name: `item-${key}` }),
            getCached,
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
      const getCached = vi.fn().mockReturnValue(changed(cachedItems));

      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, TestItem>({
            name: "Resource",
            retrieve: async () => [],
            retrieveByKey: async ({ key }) => ({ key, value: key * 10, active: true }),
            getCached,
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

  describe("subscription lifecycle", () => {
    it("should subscribe to the query on first retrieve", async () => {
      const subscribe = vi.fn().mockReturnValue(() => {});
      const retrieve = vi.fn().mockResolvedValue([{ key: 1 }]);

      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, record.Keyed<number>>({
            name: "Resource",
            retrieve,
            retrieveByKey: async ({ key }) => ({ key }),
            subscribe,
          })(),
        { wrapper },
      );

      expect(subscribe).not.toHaveBeenCalled();

      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });

      await waitFor(() => {
        expect(subscribe).toHaveBeenCalledTimes(1);
      });
    });

    it("should subscribe to an item when retrieving it before any list", async () => {
      const subscribeByKey = vi.fn().mockReturnValue(() => {});
      const retrieveByKey = vi.fn().mockResolvedValue({ key: 1 });

      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, record.Keyed<number>>({
            name: "Resource",
            retrieve: async () => [],
            retrieveByKey,
            subscribeByKey,
          })(),
        { wrapper },
      );

      expect(subscribeByKey).not.toHaveBeenCalled();

      act(() => {
        result.current.getItem(1);
      });

      await waitFor(() => {
        expect(subscribeByKey).toHaveBeenCalledTimes(1);
        expect(retrieveByKey).toHaveBeenCalled();
      });
    });

    it("should not re-subscribe on subsequent calls to getItem", async () => {
      const subscribeByKey = vi.fn().mockReturnValue(() => {});
      const retrieveByKey = vi.fn().mockResolvedValue({ key: 1 });

      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, record.Keyed<number>>({
            name: "Resource",
            retrieve: async () => [],
            retrieveByKey,
            subscribeByKey,
          })(),
        { wrapper },
      );
      act(() => {
        result.current.getItem(1);
      });
      await waitFor(() => {
        expect(subscribeByKey).toHaveBeenCalledTimes(1);
      });
      act(() => {
        result.current.getItem(1);
      });
      await testutil.expectAlways(() => {
        expect(subscribeByKey).toHaveBeenCalledTimes(1);
      });
    });

    it("should not open an item subscription for a page member served by getItem", async () => {
      const subscribe = vi.fn().mockReturnValue(() => {});
      const subscribeByKey = vi.fn().mockReturnValue(() => {});
      const retrieve = vi.fn().mockResolvedValue([{ key: 1 }]);
      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, record.Keyed<number>>({
            name: "Resource",
            retrieve,
            retrieveByKey: async ({ key }) => ({ key }),
            subscribe,
            subscribeByKey,
          })(),
        { wrapper },
      );
      await act(async () => {
        await result.current.retrieveAsync({}, { signal: controller.signal });
      });
      await waitFor(() => {
        expect(subscribe).toHaveBeenCalledTimes(1);
      });
      act(() => {
        result.current.getItem(1);
      });
      await testutil.expectAlways(() => {
        expect(subscribe).toHaveBeenCalledTimes(1);
        expect(subscribeByKey).not.toHaveBeenCalled();
      });
    });

    it("should re-subscribe on subsequent replace-mode retrieves", async () => {
      const subscribe = vi.fn().mockReturnValue(() => {});
      const retrieve = vi.fn().mockResolvedValue([{ key: 1 }]);

      const { result } = renderHook(
        () =>
          Flux.createList<{}, number, record.Keyed<number>>({
            name: "Resource",
            retrieve,
            retrieveByKey: async ({ key }) => ({ key }),
            subscribe,
          })(),
        { wrapper },
      );

      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });

      await waitFor(() => {
        expect(subscribe).toHaveBeenCalledTimes(1);
      });

      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });

      await waitFor(() => {
        expect(subscribe).toHaveBeenCalledTimes(2);
      });
    });

    it("should pass the current query to subscribeByKey", async () => {
      type TestParams = {
        filter?: string;
      };
      const subscribeByKey = vi.fn().mockReturnValue(() => {});
      const retrieve = vi.fn().mockResolvedValue([{ key: 1 }]);

      const { result } = renderHook(
        () =>
          Flux.createList<TestParams, number, record.Keyed<number>>({
            name: "Resource",
            retrieve,
            retrieveByKey: async ({ key }) => ({ key }),
            subscribeByKey,
          })({ initialQuery: { filter: "active" } }),
        { wrapper },
      );

      act(() => {
        result.current.getItem(1);
      });

      await waitFor(() => {
        const firstCall = subscribeByKey.mock.calls[0];
        expect(firstCall[0].query).toEqual({ filter: "active" });
      });
    });

    it("should not subscribe immediately when getCached returns data", () => {
      const subscribe = vi.fn().mockReturnValue(() => {});
      const getCached = vi.fn().mockReturnValue(changed([{ key: 1 }, { key: 2 }]));

      renderHook(
        () =>
          Flux.createList<{}, number, record.Keyed<number>>({
            name: "Resource",
            retrieve: async () => [],
            retrieveByKey: async ({ key }) => ({ key }),
            getCached,
            subscribe,
          })(),
        { wrapper },
      );

      expect(subscribe).not.toHaveBeenCalled();
    });

    it("should serve getItem from cached initial data without fetching", async () => {
      const retrieveByKey = vi.fn().mockResolvedValue({ key: 1 });
      const useList = Flux.createList<{}, number, record.Keyed<number>>({
        name: "Resource",
        retrieve: async () => [],
        retrieveByKey,
        getCached: () => changed([{ key: 1 }]),
      });

      const { result } = renderHook(useList, { wrapper });
      let item: record.Keyed<number> | undefined;
      act(() => {
        item = result.current.getItem(1);
      });
      expect(item).toEqual({ key: 1 });
      await testutil.expectAlways(() => {
        expect(retrieveByKey).not.toHaveBeenCalled();
      });
    });
  });

  describe("subscriptions against a live client", () => {
    it("should correctly update a list item when the subscribed answer changes", async () => {
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
            ranger.Range
          >({
            name: "Resource",
            retrieve: async ({ client }) =>
              await client.ranges.retrieve({ keys: [rng.key] }),
            retrieveByKey: async ({ client, key }) => await client.ranges.retrieve(key),
            subscribe: ({ client }, handler) =>
              client.ranges.onChange({ keys: [rng.key] }, handler),
          })();
          const value = Flux.useListItem<ranger.Key, ranger.Range>({
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

    it("should update an item fetched by key when its subscription changes", async () => {
      const rng = await client.ranges.create({
        name: "Test Range",
        timeRange: new TimeRange({
          start: TimeSpan.seconds(12),
          end: TimeSpan.seconds(13),
        }),
      });

      const { result } = renderHook(
        () => {
          const { getItem, subscribe } = Flux.createList<{}, ranger.Key, ranger.Range>({
            name: "Resource",
            retrieve: async () => [],
            retrieveByKey: async ({ client, key }) => await client.ranges.retrieve(key),
            subscribeByKey: ({ client, key }, handler) =>
              client.ranges.onChange(key, handler),
          })();
          const value = Flux.useListItem<ranger.Key, ranger.Range>({
            subscribe,
            getItem,
            key: rng.key,
          });
          return { value };
        },
        { wrapper },
      );

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
            ranger.Range
          >({
            name: "Resource",
            retrieve: async ({ client }) =>
              await client.ranges.retrieve({ keys: [rng.key] }),
            retrieveByKey: async ({ client, key }) => await client.ranges.retrieve(key),
            subscribe: ({ client }, handler) =>
              client.ranges.onChange({ keys: [rng.key] }, handler),
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

    it("should maintain sort order when items are updated through subscriptions", async () => {
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
      const keys = [rng1.key, rng2.key];

      const { result } = renderHook(
        () =>
          Flux.createList<{}, ranger.Key, ranger.Range>({
            name: "Resource",
            retrieve: async ({ client }) => await client.ranges.retrieve({ keys }),
            retrieveByKey: async ({ client, key }) => await client.ranges.retrieve(key),
            subscribe: ({ client }, handler) =>
              client.ranges.onChange({ keys }, handler),
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

    it("should insert new items in correct sorted position through subscriptions", async () => {
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
      const keySet = new Set([rng1.key, rng2.key, rng3Key]);
      // rng3 does not exist yet, so the query is a permissive overlap window
      // and the hook-level filter narrows the answer to the test's ranges.
      const query = {
        overlapsWith: new TimeRange({
          start: TimeSpan.seconds(9),
          end: TimeSpan.seconds(16),
        }),
      };

      const { result } = renderHook(
        () =>
          Flux.createList<{}, ranger.Key, ranger.Range>({
            name: "Resource",
            retrieve: async ({ client }) => await client.ranges.retrieve(query),
            retrieveByKey: async ({ client, key }) => await client.ranges.retrieve(key),
            subscribe: ({ client }, handler) => client.ranges.onChange(query, handler),
          })({
            sort: (a, b) => a.name.localeCompare(b.name),
            filter: (r) => keySet.has(r.key),
          }),
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

  describe("client reconnect", () => {
    interface Doc extends record.Keyed<string> {
      key: string;
      name: string;
    }
    const ReconnectAetherProvider = aetherTest.createProvider({
      ...synnax.REGISTRY,
      ...status.REGISTRY,
    });

    // The hooks only read `key` off the client; a minimal stub models the swap.
    const clientWithKey = (key: string): Client => ({ key }) as unknown as Client;

    /** A fake per-client domain: answers by client key, push notifies subs. */
    interface FakeDomain {
      answers: Record<string, Doc[]>;
      push: (clientKey: string, docs: Doc[]) => void;
      pushItem: (clientKey: string, doc: Doc) => void;
      subscribe: (clientKey: string, handler: query.ChangeHandler<Doc[]>) => () => void;
      subscribeItem: (
        clientKey: string,
        key: string,
        handler: query.ChangeHandler<Doc>,
      ) => () => void;
    }

    const createFakeDomain = (): FakeDomain => {
      const answers: Record<string, Doc[]> = { a: [], b: [] };
      const subs = new Map<string, Set<query.ChangeHandler<Doc[]>>>();
      const itemSubs = new Map<string, Set<query.ChangeHandler<Doc>>>();
      return {
        answers,
        push: (clientKey, docs) => {
          answers[clientKey] = docs;
          subs.get(clientKey)?.forEach((h) => h({ variant: "changed", data: docs }));
        },
        pushItem: (clientKey, doc) => {
          itemSubs
            .get(`${clientKey}:${doc.key}`)
            ?.forEach((h) => h({ variant: "changed", data: doc }));
        },
        subscribe: (clientKey, handler) => {
          const set = subs.get(clientKey) ?? new Set();
          set.add(handler);
          subs.set(clientKey, set);
          return () => set.delete(handler);
        },
        subscribeItem: (clientKey, key, handler) => {
          const mapKey = `${clientKey}:${key}`;
          const set = itemSubs.get(mapKey) ?? new Set();
          set.add(handler);
          itemSubs.set(mapKey, set);
          return () => set.delete(handler);
        },
      };
    };

    interface ReconnectSetup {
      domain: FakeDomain;
      wrapper: React.FC<PropsWithChildren>;
      reconnect: () => void;
    }

    const createReconnectSetup = (): ReconnectSetup => {
      const domain = createFakeDomain();
      let activeSynnax = clientWithKey("a");
      const wrapper = ({ children }: PropsWithChildren): ReactElement => (
        <ReconnectAetherProvider>
          <Status.Aggregator>
            <Synnax.TestProvider client={activeSynnax}>{children}</Synnax.TestProvider>
          </Status.Aggregator>
        </ReconnectAetherProvider>
      );
      const reconnect = (): void => {
        activeSynnax = clientWithKey("b");
      };
      return { domain, wrapper, reconnect };
    };

    it("should refetch and re-subscribe onto the new client after the client changes", async () => {
      const { domain, wrapper, reconnect } = createReconnectSetup();
      const useList = Flux.createList<{}, string, Doc>({
        name: "Doc",
        retrieve: async ({ client }) => domain.answers[client.key],
        retrieveByKey: async ({ key }) => ({ key, name: key }),
        subscribe: ({ client }, handler) => domain.subscribe(client.key, handler),
      });

      const { result, rerender } = renderHook(() => useList(), { wrapper });

      await act(
        async () =>
          await result.current.retrieveAsync({}, { signal: controller.signal }),
      );

      act(() => domain.push("a", [{ key: "before", name: "Before" }]));
      await waitFor(() => expect(result.current.data).toContain("before"));

      reconnect();
      rerender();

      // Refetch against the new (empty) client replaces the old data.
      await waitFor(() => expect(result.current.data).not.toContain("before"));

      act(() => domain.push("b", [{ key: "after", name: "After" }]));
      await waitFor(() => expect(result.current.data).toContain("after"));
    });

    it("should re-subscribe item lookups without running a list retrieve when no query was issued", async () => {
      const { domain, wrapper, reconnect } = createReconnectSetup();
      const retrieve = vi.fn(async () => [] as Doc[]);
      const useList = Flux.createList<{}, string, Doc>({
        name: "Doc",
        retrieve,
        retrieveByKey: async ({ key }) => ({ key, name: key }),
        subscribeByKey: ({ client, key }, handler) =>
          domain.subscribeItem(client.key, key, handler),
      });

      const { result, rerender } = renderHook(
        () => {
          const list = useList();
          const value = Flux.useListItem<string, Doc>({
            subscribe: list.subscribe,
            getItem: list.getItem,
            key: "k",
          });
          return { list, value };
        },
        { wrapper },
      );

      await waitFor(() => expect(result.current.value?.name).toBe("k"));
      act(() => domain.pushItem("a", { key: "k", name: "A" }));
      await waitFor(() => expect(result.current.value?.name).toBe("A"));

      reconnect();
      rerender();

      // getItem-only: re-subscribe onto the new client without a list fetch.
      act(() => domain.pushItem("b", { key: "k", name: "B" }));
      await waitFor(() => expect(result.current.value?.name).toBe("B"));
      expect(result.current.list.data).toEqual([]);
      expect(retrieve).not.toHaveBeenCalled();
    });
  });
});
