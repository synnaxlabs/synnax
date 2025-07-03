// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { newTestClient, ranger, type Synnax } from "@synnaxlabs/client";
import { type record, TimeRange, TimeSpan } from "@synnaxlabs/x";
import { renderHook, waitFor } from "@testing-library/react";
import { act, type FC, type PropsWithChildren } from "react";
import { describe, expect, it, vi } from "vitest";

import { Flux } from "@/flux";
import { Sync } from "@/flux/sync";
import { Synnax as PSynnax } from "@/synnax";

const client = newTestClient();

const newWrapper =
  (client: Synnax | null): FC<PropsWithChildren> =>
  // eslint-disable-next-line react/display-name
  (props) => (
    <PSynnax.TestProvider client={client}>
      <Sync.Provider {...props} />
    </PSynnax.TestProvider>
  );

describe("list", () => {
  describe("initial list", () => {
    it("should return a loading result as its initial state", () => {
      const { result } = renderHook(
        () =>
          Flux.createList({
            name: "Resource",
            retrieve: async () => [],
            retrieveByKey: async () => ({ key: 12 }),
          })(),
        { wrapper: newWrapper(client) },
      );
      expect(result.current.variant).toEqual("loading");
      expect(result.current.data).toEqual([]);
      expect(result.current.error).toEqual(null);
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
        { wrapper: newWrapper(client) },
      );
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => {
        expect(retrieve).toHaveBeenCalledTimes(1);
        expect(result.current.variant).toEqual("success");
        expect(result.current.data).toEqual([1, 2]);
        expect(result.current.error).toEqual(null);
      });
    });
  });

  describe("useListItem", () => {
    it("should return a pre-retrieved list item", async () => {
      const { result } = renderHook(
        () => {
          const { useListItem, retrieve } = Flux.createList<
            {},
            number,
            record.Keyed<number>
          >({
            name: "Resource",
            retrieve: async () => [{ key: 1 }, { key: 2 }],
            retrieveByKey: async ({ key }) => ({ key }),
          })();
          return { retrieve, value: useListItem(1) };
        },
        { wrapper: newWrapper(client) },
      );
      await waitFor(() => {
        expect(result.current.value).toEqual({ key: 1 });
      });
    });
  });

  describe("listeners", () => {
    interface RangeParams extends Flux.Params {
      key: ranger.Key;
    }
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
          const { useListItem, retrieve } = Flux.createList<
            RangeParams,
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
          return { retrieve, value: useListItem(rng.key) };
        },
        { wrapper: newWrapper(client) },
      );

      await waitFor(() => {
        expect(result.current.value?.name).toEqual("Test Range");
      });

      await act(async () => {
        await client.ranges.rename(rng.key, "Test Range 2");
      });

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
          const { useListItem, retrieve } = Flux.createList<
            RangeParams,
            ranger.Key,
            ranger.Payload
          >({
            name: "Resource",
            retrieve: async ({ client }) => [await client.ranges.retrieve(rng.key)],
            retrieveByKey: async ({ client, key }) => await client.ranges.retrieve(key),
            listeners: [
              {
                channel: ranger.DELETE_CHANNEL_NAME,
                onChange: Sync.parsedHandler(
                  ranger.keyZ,
                  async ({ onDelete, changed }) => onDelete(changed),
                ),
              },
            ],
          })();
          return { retrieve, value: useListItem(rng.key) };
        },
        { wrapper: newWrapper(client) },
      );

      await waitFor(() => {
        expect(result.current.value?.name).toEqual("Test Range");
      });

      await act(async () => {
        await client.ranges.delete(rng.key);
      });

      await waitFor(() => {
        expect(result.current.value).toBeUndefined();
      });

      unmount();
    });
  });
});
