// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { newTestClient, ranger } from "@synnaxlabs/client";
import { type record, TimeRange, TimeSpan } from "@synnaxlabs/x";
import { renderHook, waitFor } from "@testing-library/react";
import { act } from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { Flux } from "@/flux";
import { Sync } from "@/flux/sync";
import { newSynnaxWrapper } from "@/testutil/Synnax";

const client = newTestClient();

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
        { wrapper: newSynnaxWrapper(client) },
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
        { wrapper: newSynnaxWrapper(client) },
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
        { wrapper: newSynnaxWrapper(client) },
      );
      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });
      await waitFor(() => {
        expect(retrieve).toHaveBeenCalledTimes(1);
        expect(result.current.variant).toEqual("error");
        expect(result.current.description).toEqual("Test Error");
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
        { wrapper: newSynnaxWrapper(client) },
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
        { wrapper: newSynnaxWrapper(client) },
      );
      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });
      await waitFor(() => {
        expect(result.current.value).toEqual(undefined);
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
        { wrapper: newSynnaxWrapper(client) },
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
        { wrapper: newSynnaxWrapper(client) },
      );
      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("error");
        expect(result.current.description).toEqual("Test Error");
      });
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
            ranger.Payload
          >({
            name: "Resource",
            retrieve: async ({ client }) => [await client.ranges.retrieve(rng.key)],
            retrieveByKey: async ({ client, key }) => await client.ranges.retrieve(key),
            listeners: [
              {
                channel: ranger.SET_CHANNEL_NAME,
                onChange: Sync.parsedHandler(
                  ranger.payloadZ,
                  async ({ onChange, changed }) => onChange(changed.key, () => changed),
                ),
              },
            ],
          })();
          const value = Flux.useListItem<ranger.Key, ranger.Payload>({
            subscribe,
            getItem,
            key: rng.key,
          });
          return { retrieve, value };
        },
        { wrapper: newSynnaxWrapper(client) },
      );

      act(() => {
        result.current.retrieve({});
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
      const { result, unmount } = renderHook(
        () => {
          const { getItem, retrieve } = Flux.createList<{}, ranger.Key, ranger.Payload>(
            {
              name: "Resource",
              retrieve: async ({ client }) => [await client.ranges.retrieve(rng.key)],
              retrieveByKey: async ({ client, key }) =>
                await client.ranges.retrieve(key),
              listeners: [
                {
                  channel: ranger.DELETE_CHANNEL_NAME,
                  onChange: Sync.parsedHandler(
                    ranger.keyZ,
                    async ({ onDelete, changed }) => onDelete(changed),
                  ),
                },
              ],
            },
          )();
          return { retrieve, value: getItem(rng.key) };
        },
        { wrapper: newSynnaxWrapper(client) },
      );

      act(() => {
        result.current.retrieve({});
      });

      await waitFor(() => {
        expect(result.current.value?.name).toEqual("Test Range");
      });
      await act(async () => await client.ranges.delete(rng.key));
      await waitFor(() => {
        expect(result.current.value?.key).not.toEqual(rng.key);
      });
    });
  });
});
