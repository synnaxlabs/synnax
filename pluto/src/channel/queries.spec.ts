// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, createTestClient, DataType } from "@synnaxlabs/client";
import { act, renderHook, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { afterEach, beforeAll, beforeEach, describe, expect, it } from "vitest";

import { Channel } from "@/channel";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

describe("queries", () => {
  let controller: AbortController;
  const client = createTestClient();
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });
  beforeEach(() => {
    controller = new AbortController();
  });
  afterEach(() => {
    controller.abort();
  });

  describe("useList", () => {
    it("should return a list of channel keys", async () => {
      const indexCh = await client.channels.create({
        name: "time_index",
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });
      const ch1 = await client.channels.create({
        name: "channel1",
        dataType: DataType.FLOAT32,
        index: indexCh.key,
      });
      const ch2 = await client.channels.create({
        name: "channel2",
        dataType: DataType.FLOAT32,
        index: indexCh.key,
      });

      const { result } = renderHook(() => Channel.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      expect(result.current.data.length).toBeGreaterThanOrEqual(3);
      expect(result.current.data).toContain(ch1.key);
      expect(result.current.data).toContain(ch2.key);
    });

    it("should get individual channels using getItem", async () => {
      const testChannel = await client.channels.create({
        name: "testChannel",
        dataType: DataType.FLOAT32,
        virtual: true,
      });

      const { result } = renderHook(() => Channel.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const retrievedChannel = result.current.getItem(testChannel.key);
      expect(retrievedChannel?.key).toEqual(testChannel.key);
      expect(retrievedChannel?.name).toEqual("testChannel");
    });

    it("should filter channels by search term", async () => {
      await client.channels.create({
        name: "ordinary_channel",
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      await client.channels.create({
        name: "special_channel",
        dataType: DataType.FLOAT32,
        virtual: true,
      });

      const { result } = renderHook(() => Channel.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve(
          { searchTerm: "special" },
          { signal: controller.signal },
        );
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data.length).toBeGreaterThanOrEqual(1);
      expect(
        result.current.data
          .map((key: channel.Key) => result.current.getItem(key)?.name)
          .includes("special_channel"),
      ).toBe(true);
    });

    it("should handle pagination with limit and offset", async () => {
      for (let i = 0; i < 5; i++)
        await client.channels.create({
          name: `paginationChannel${i}`,
          dataType: DataType.FLOAT32,
          virtual: true,
        });

      const { result } = renderHook(() => Channel.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({ limit: 2, offset: 1 }, { signal: controller.signal });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data).toHaveLength(2);
    });

    it("should update the list when a channel is created", async () => {
      const { result } = renderHook(() => Channel.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      const initialLength = result.current.data.length;

      const newChannel = await client.channels.create({
        name: "newChannel",
        dataType: DataType.FLOAT32,
        virtual: true,
      });

      await waitFor(() => {
        expect(result.current.data).toHaveLength(initialLength + 1);
        expect(result.current.data).toContain(newChannel.key);
      });
    });

    it("should update the list when a channel is updated", async () => {
      const testChannel = await client.channels.create({
        name: "original",
        dataType: DataType.FLOAT32,
        virtual: true,
      });

      const { result } = renderHook(
        () => {
          const list = Channel.useList();
          const rename = Channel.useRename();
          return { list, rename };
        },
        { wrapper },
      );
      act(() => {
        result.current.list.retrieve({}, { signal: controller.signal });
      });
      await waitFor(() => {
        expect(result.current.list.variant).toEqual("success");
      });
      expect(result.current.list.getItem(testChannel.key)?.name).toEqual("original");

      await act(async () => {
        await result.current.rename.updateAsync({
          key: testChannel.key,
          name: "updated",
        });
      });

      await waitFor(() => {
        expect(result.current.list.getItem(testChannel.key)?.name).toEqual("updated");
      });
    });

    it("should remove channel from list when deleted", async () => {
      const testChannel = await client.channels.create({
        name: "toDelete",
        dataType: DataType.FLOAT32,
        virtual: true,
      });

      const { result } = renderHook(() => Channel.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });
      await waitFor(() => {
        expect(result.current.variant).toEqual("success");
      });
      expect(result.current.data).toContain(testChannel.key);

      await client.channels.delete(testChannel.key);

      await waitFor(() => {
        expect(result.current.data).not.toContain(testChannel.key);
      });
    });

    it("should handle index channels correctly", async () => {
      const indexChannel = await client.channels.create({
        name: "index_channel",
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });

      const { result } = renderHook(() => Channel.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const retrievedChannel = result.current.getItem(indexChannel.key);
      expect(retrievedChannel?.isIndex).toBe(true);
      expect(retrievedChannel?.dataType.toString()).toEqual("timestamp");
    });

    it("should handle virtual channels correctly", async () => {
      const virtualChannel = await client.channels.create({
        name: "virtual_channel",
        dataType: DataType.FLOAT32,
        isIndex: false,
        virtual: true,
      });

      const { result } = renderHook(() => Channel.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({}, { signal: controller.signal });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const retrievedChannel = result.current.getItem(virtualChannel.key);
      expect(retrievedChannel?.virtual).toBe(true);
    });

    it("should update the channel alias when a range alias is set", async () => {
      const range = await client.ranges.create({
        name: "range",
        timeRange: { start: 1n, end: 1000n },
      });
      const channel = await client.channels.create({
        name: "channel",
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      await client.ranges.setAlias(range.key, channel.key, "alias");
      const { result } = renderHook(() => Channel.useList(), {
        wrapper,
      });
      act(() => {
        result.current.retrieve({ rangeKey: range.key }, { signal: controller.signal });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.getItem(channel.key)?.alias).toEqual("alias");

      await act(async () => {
        await client.ranges.setAlias(range.key, channel.key, "new_alias");
      });
      await waitFor(() =>
        expect(result.current.getItem(channel.key)?.alias).toEqual("new_alias"),
      );
    });

    it("should correctly retrieve the alias when an initial query is provided, and getItem is called but not retrieve", async () => {
      const range = await client.ranges.create({
        name: "range",
        timeRange: { start: 1n, end: 1000n },
      });
      const channel = await client.channels.create({
        name: "channel",
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      await client.ranges.setAlias(range.key, channel.key, "alias");
      const { result } = renderHook(
        () =>
          Channel.useList({
            initialQuery: { rangeKey: range.key },
            useCachedList: false,
          }),
        { wrapper },
      );
      await waitFor(() =>
        expect(result.current.getItem(channel.key)?.alias).toEqual("alias"),
      );
    });

    describe("retrieveCached", () => {
      it("should use cached data on initial mount when no searchTerm", async () => {
        const ch = await client.channels.create({
          name: "cached_test",
          dataType: DataType.FLOAT32,
          virtual: true,
        });
        const { result: firstResult, unmount } = renderHook(() => Channel.useList(), {
          wrapper,
        });
        act(() => {
          firstResult.current.retrieve({}, { signal: controller.signal });
        });
        await waitFor(() => expect(firstResult.current.variant).toEqual("success"));
        expect(firstResult.current.data).toContain(ch.key);
        unmount();

        const { result: secondResult } = renderHook(() => Channel.useList(), {
          wrapper,
        });
        expect(secondResult.current.variant).toEqual("loading");
        expect(secondResult.current.data).toContain(ch.key);
      });

      it("should not use cached data on initial mount when searchTerm provided", async () => {
        await client.channels.create({
          name: "cached_test_search",
          dataType: DataType.FLOAT32,
          virtual: true,
        });

        const { result: firstResult } = renderHook(() => Channel.useList(), {
          wrapper,
        });
        act(() => {
          firstResult.current.retrieve({}, { signal: controller.signal });
        });
        await waitFor(() => expect(firstResult.current.variant).toEqual("success"));

        const { result: secondResult } = renderHook(
          () => Channel.useList({ initialQuery: { searchTerm: "cached" } }),
          { wrapper },
        );
        expect(secondResult.current.variant).toEqual("loading");
        expect(secondResult.current.data).toEqual([]);
      });

      it("should filter cached data by internal flag", async () => {
        const normalCh = await client.channels.create({
          name: "normal_ch",
          dataType: DataType.FLOAT32,
          virtual: true,
        });

        const { result: firstResult } = renderHook(() => Channel.useList(), {
          wrapper,
        });
        act(() => {
          firstResult.current.retrieve({}, { signal: controller.signal });
        });
        await waitFor(() => expect(firstResult.current.variant).toEqual("success"));

        const { result: secondResult } = renderHook(
          () => Channel.useList({ initialQuery: { internal: true } }),
          { wrapper },
        );
        expect(secondResult.current.data).not.toContain(normalCh.key);
      });

      //     it("should filter by calculated channels", async () => {
      //       const idxCh = await client.channels.create({
      //         name: "idx_for_calc",
      //         dataType: DataType.TIMESTAMP,
      //         isIndex: true,
      //       });
      //       const calcCh = await client.channels.create({
      //         name: "calculated_ch",
      //         dataType: DataType.FLOAT32,
      //         virtual: true,
      //         expression: "return 1",
      //         requires: [idxCh.key],
      //       });
      //       const normalCh = await client.channels.create({
      //         name: "normal_virtual",
      //         dataType: DataType.FLOAT32,
      //         virtual: true,
      //       });

      //       const { result: firstResult, unmount } = renderHook(() => Channel.useList(), {
      //         wrapper,
      //       });
      //       act(() => {
      //         firstResult.current.retrieve({}, { signal: controller.signal });
      //       });
      //       await waitFor(() => expect(firstResult.current.variant).toEqual("success"));
      //       unmount();

      //       const { result: secondResult } = renderHook(
      //         () => Channel.useList({ initialQuery: { calculated: true } }),
      //         {
      //           wrapper,
      //         },
      //       );
      //       expect(secondResult.current.variant).toEqual("loading");
      //       expect(secondResult.current.data).toContain(calcCh.key);
      //       expect(secondResult.current.data).not.toContain(normalCh.key);
      //     });

      //     it("should filter by dataTypes inclusion", async () => {
      //       const float32Ch = await client.channels.create({
      //         name: "float32_ch",
      //         dataType: DataType.FLOAT32,
      //         virtual: true,
      //       });
      //       const float64Ch = await client.channels.create({
      //         name: "float64_ch",
      //         dataType: DataType.FLOAT64,
      //         virtual: true,
      //       });

      //       const { result: firstResult, unmount } = renderHook(() => Channel.useList(), {
      //         wrapper,
      //       });
      //       act(() => {
      //         firstResult.current.retrieve({}, { signal: controller.signal });
      //       });
      //       await waitFor(() => expect(firstResult.current.variant).toEqual("success"));
      //       unmount();

      //       const { result: secondResult } = renderHook(
      //         () => Channel.useList({ initialQuery: { dataTypes: [DataType.FLOAT32] } }),
      //         { wrapper },
      //       );
      //       expect(secondResult.current.data).toContain(float32Ch.key);
      //       expect(secondResult.current.data).not.toContain(float64Ch.key);
      //     });

      //     it("should filter by notDataTypes exclusion", async () => {
      //       const float32Ch = await client.channels.create({
      //         name: "float32_exclude",
      //         dataType: DataType.FLOAT32,
      //         virtual: true,
      //       });
      //       const int32Ch = await client.channels.create({
      //         name: "int32_include",
      //         dataType: DataType.INT32,
      //         virtual: true,
      //       });

      //       const { result: firstResult, unmount } = renderHook(() => Channel.useList(), {
      //         wrapper,
      //       });
      //       act(() => {
      //         firstResult.current.retrieve({}, { signal: controller.signal });
      //       });
      //       await waitFor(() => expect(firstResult.current.variant).toEqual("success"));
      //       unmount();

      //       const { result: secondResult } = renderHook(
      //         () =>
      //           Channel.useList({
      //             initialQuery: { notDataTypes: [DataType.FLOAT32] },
      //           }),
      //         { wrapper },
      //       );
      //       expect(secondResult.current.variant).toEqual("loading");
      //       expect(secondResult.current.data).not.toContain(float32Ch.key);
      //       expect(secondResult.current.data).toContain(int32Ch.key);
      //     });

      //     it("should filter by isIndex", async () => {
      //       const indexCh = await client.channels.create({
      //         name: "index_filter",
      //         dataType: DataType.TIMESTAMP,
      //         isIndex: true,
      //       });
      //       const dataCh = await client.channels.create({
      //         name: "data_filter",
      //         dataType: DataType.FLOAT32,
      //         index: indexCh.key,
      //       });

      //       const { result: firstResult, unmount } = renderHook(() => Channel.useList(), {
      //         wrapper,
      //       });
      //       act(() => {
      //         firstResult.current.retrieve({}, { signal: controller.signal });
      //       });
      //       await waitFor(() => expect(firstResult.current.variant).toEqual("success"));
      //       unmount();

      //       const { result: secondResult } = renderHook(
      //         () => Channel.useList({ initialQuery: { isIndex: true } }),
      //         { wrapper },
      //       );
      //       expect(secondResult.current.variant).toEqual("loading");
      //       expect(secondResult.current.data).toContain(indexCh.key);
      //       expect(secondResult.current.data).not.toContain(dataCh.key);
      //     });

      //     it("should filter by virtual", async () => {
      //       const indexCh = await client.channels.create({
      //         name: "index_virt",
      //         dataType: DataType.TIMESTAMP,
      //         isIndex: true,
      //       });
      //       const virtualCh = await client.channels.create({
      //         name: "virtual_filter",
      //         dataType: DataType.FLOAT32,
      //         virtual: true,
      //       });
      //       const persistedCh = await client.channels.create({
      //         name: "persisted_filter",
      //         dataType: DataType.FLOAT32,
      //         index: indexCh.key,
      //         virtual: false,
      //       });

      //       const { result: firstResult, unmount } = renderHook(() => Channel.useList(), {
      //         wrapper,
      //       });
      //       act(() => {
      //         firstResult.current.retrieve({}, { signal: controller.signal });
      //       });
      //       await waitFor(() => expect(firstResult.current.variant).toEqual("success"));
      //       unmount();

      //       const { result: secondResult } = renderHook(
      //         () => Channel.useList({ initialQuery: { virtual: true } }),
      //         { wrapper },
      //       );
      //       expect(secondResult.current.variant).toEqual("loading");
      //       expect(secondResult.current.data).toContain(virtualCh.key);
      //       expect(secondResult.current.data).not.toContain(persistedCh.key);
      //     });

      //     it("should handle combined filters", async () => {
      //       const indexCh = await client.channels.create({
      //         name: "idx_combined",
      //         dataType: DataType.TIMESTAMP,
      //         isIndex: true,
      //       });
      //       const virtualFloat32Ch = await client.channels.create({
      //         name: "virtual_float32",
      //         dataType: DataType.FLOAT32,
      //         virtual: true,
      //       });
      //       const virtualInt32Ch = await client.channels.create({
      //         name: "virtual_int32",
      //         dataType: DataType.INT32,
      //         virtual: true,
      //       });
      //       const persistedFloat32Ch = await client.channels.create({
      //         name: "persisted_float32",
      //         dataType: DataType.FLOAT32,
      //         index: indexCh.key,
      //         virtual: false,
      //       });

      //       const { result: firstResult, unmount } = renderHook(() => Channel.useList(), {
      //         wrapper,
      //       });
      //       act(() => {
      //         firstResult.current.retrieve({}, { signal: controller.signal });
      //       });
      //       await waitFor(() => expect(firstResult.current.variant).toEqual("success"));
      //       unmount();

      //       const { result: secondResult } = renderHook(
      //         () =>
      //           Channel.useList({
      //             initialQuery: {
      //               virtual: true,
      //               dataTypes: [DataType.FLOAT32],
      //               internal: false,
      //             },
      //           }),
      //         { wrapper },
      //       );
      //       expect(secondResult.current.variant).toEqual("loading");
      //       expect(secondResult.current.data).toContain(virtualFloat32Ch.key);
      //       expect(secondResult.current.data).not.toContain(virtualInt32Ch.key);
      //       expect(secondResult.current.data).not.toContain(persistedFloat32Ch.key);
      //     });
      //   });
      // });

      // describe("useForm", () => {
      //   it("should create a new virtual channel", async () => {
      //     const { result } = renderHook(() => Channel.useForm({ query: {} }), {
      //       wrapper,
      //     });

      //     act(() => {
      //       result.current.form.set("name", "newFormChannel");
      //       result.current.form.set("dataType", DataType.FLOAT32.toString());
      //       result.current.form.set("virtual", true);
      //       result.current.save({ signal: controller.signal });
      //     });

      //     await waitFor(() => {
      //       expect(result.current.form.value().name).toEqual("newFormChannel");
      //       expect(result.current.form.value().dataType).toEqual(
      //         DataType.FLOAT32.toString(),
      //       );
      //       expect(result.current.form.value().virtual).toBe(true);
      //       expect(result.current.form.value().key).toBeDefined();
      //       expect(result.current.form.value().key).toBeGreaterThan(0);
      //     });
      //   });

      //   it("should create a new index channel", async () => {
      //     const { result } = renderHook(() => Channel.useForm({ query: {} }), {
      //       wrapper,
      //     });

      //     act(() => {
      //       result.current.form.set("name", "newIndexChannel");
      //       result.current.form.set("dataType", DataType.TIMESTAMP.toString());
      //       result.current.form.set("isIndex", true);
      //       result.current.save({ signal: controller.signal });
      //     });

      //     await waitFor(() => {
      //       expect(result.current.form.value().name).toEqual("newIndexChannel");
      //       expect(result.current.form.value().dataType).toEqual(
      //         DataType.TIMESTAMP.toString(),
      //       );
      //       expect(result.current.form.value().isIndex).toBe(true);
      //       expect(result.current.form.value().key).toBeDefined();
      //       expect(result.current.form.value().key).toBeGreaterThan(0);
      //     });
      //   });

      //   it("should create a new data channel with index", async () => {
      //     const indexChannel = await client.channels.create({
      //       name: "test_index",
      //       dataType: DataType.TIMESTAMP,
      //       isIndex: true,
      //     });

      //     const { result } = renderHook(() => Channel.useForm({ query: {} }), {
      //       wrapper,
      //     });

      //     act(() => {
      //       result.current.form.set("name", "newDataChannel");
      //       result.current.form.set("dataType", DataType.FLOAT32.toString());
      //       result.current.form.set("index", indexChannel.key);
      //       result.current.save({ signal: controller.signal });
      //     });

      //     await waitFor(() => {
      //       expect(result.current.form.value().name).toEqual("newDataChannel");
      //       expect(result.current.form.value().dataType).toEqual(
      //         DataType.FLOAT32.toString(),
      //       );
      //       expect(result.current.form.value().index).toEqual(indexChannel.key);
      //       expect(result.current.form.value().key).toBeDefined();
      //       expect(result.current.form.value().key).toBeGreaterThan(0);
      //     });
      //   });

      //   it("should retrieve and edit existing channel", async () => {
      //     const existingChannel = await client.channels.create({
      //       name: "existingChannel",
      //       dataType: DataType.FLOAT64,
      //       virtual: true,
      //     });

      //     const { result } = renderHook(
      //       () => Channel.useForm({ query: { key: existingChannel.key } }),
      //       { wrapper },
      //     );
      //     await waitFor(() => expect(result.current.variant).toEqual("success"));

      //     expect(result.current.form.value().name).toEqual("existingChannel");
      //     expect(result.current.form.value().dataType).toEqual(DataType.FLOAT64.toString());
      //     expect(result.current.form.value().virtual).toBe(true);

      //     act(() => {
      //       result.current.form.set("name", "editedChannel");
      //       result.current.save({ signal: controller.signal });
      //     });

      //     await waitFor(() => {
      //       expect(result.current.form.value().name).toEqual("editedChannel");
      //     });
      //   });

      //   it("should update form when channel is updated externally", async () => {
      //     const testChannel = await client.channels.create({
      //       name: "externalUpdate",
      //       dataType: DataType.FLOAT32,
      //       virtual: true,
      //     });

      //     const { result } = renderHook(
      //       () => {
      //         const form = Channel.useForm({ query: { key: testChannel.key } });
      //         const rename = Channel.useRename();
      //         return { form, rename };
      //       },
      //       { wrapper },
      //     );
      //     await waitFor(() => expect(result.current.form.variant).toEqual("success"));
      //     expect(result.current.form.form.value().name).toEqual("externalUpdate");

      //     await act(async () => {
      //       await result.current.rename.updateAsync({
      //         key: testChannel.key,
      //         name: "externallyUpdated",
      //       });
      //     });

      //     await waitFor(() => {
      //       expect(result.current.form.form.value().name).toEqual("externallyUpdated");
      //     });
      //   });

      //   it("should handle form with default values", async () => {
      //     const { result } = renderHook(() => Channel.useForm({ query: {} }), {
      //       wrapper,
      //     });

      //     expect(result.current.form.value().name).toEqual("");
      //     expect(result.current.form.value().dataType).toEqual(DataType.FLOAT32.toString());
      //     expect(result.current.form.value().virtual).toBe(false);
      //     expect(result.current.form.value().isIndex).toBe(false);
      //     expect(result.current.form.value().index).toEqual(0);
      //   });

      //   it("should validate that index channels have timestamp data type", async () => {
      //     const { result } = renderHook(() => Channel.useForm({ query: {} }), {
      //       wrapper,
      //     });

      //     act(() => {
      //       result.current.form.set("name", "invalidIndex");
      //       result.current.form.set("dataType", DataType.FLOAT32.toString());
      //       result.current.form.set("isIndex", true);
      //     });

      //     expect(result.current.form.validate()).toBe(false);
      //     expect(result.current.form.get("dataType").status.message).toContain(
      //       "Index channel must have data type TIMESTAMP",
      //     );
      //   });

      //   it("should validate that data channels have an index or are virtual", async () => {
      //     const { result } = renderHook(() => Channel.useForm({ query: {} }), {
      //       wrapper,
      //     });

      //     act(() => {
      //       result.current.form.set("name", "invalidDataChannel");
      //       result.current.form.set("dataType", DataType.FLOAT32.toString());
      //       result.current.form.set("isIndex", false);
      //       result.current.form.set("index", 0);
      //       result.current.form.set("virtual", false);
      //     });

      //     expect(result.current.form.validate()).toBe(false);
      //     expect(result.current.form.get("index").status.message).toContain(
      //       "Data channel must have an index",
      //     );
      //   });

      //   it("should validate that persisted channels have fixed-size data types", async () => {
      //     const { result } = renderHook(() => Channel.useForm({ query: {} }), {
      //       wrapper,
      //     });

      //     act(() => {
      //       result.current.form.set("name", "invalidPersistedChannel");
      //       result.current.form.set("dataType", DataType.STRING.toString());
      //       result.current.form.set("virtual", false);
      //       result.current.form.set("isIndex", false);
      //     });

      //     expect(result.current.form.validate()).toBe(false);
      //     expect(result.current.form.get("dataType").status.message).toContain(
      //       "Persisted channels must have a fixed-size data type",
      //     );
      //   });
      // });

      // describe("useRetrieve", () => {
      //   it("should retrieve a channel by key", async () => {
      //     const ch = await client.channels.create({
      //       name: "retrieve_test",
      //       dataType: DataType.FLOAT32,
      //       virtual: true,
      //     });
      //     const { result } = renderHook(() => Channel.useRetrieve({ key: ch.key }), {
      //       wrapper,
      //     });
      //     await waitFor(() => expect(result.current.variant).toEqual("success"));
      //     expect(result.current.data?.key).toEqual(ch.key);
      //     expect(result.current.data?.name).toEqual("retrieve_test");
      //   });

      //   it("should retrieve channel with range alias", async () => {
      //     const indexCh = await client.channels.create({
      //       name: "alias_index",
      //       dataType: DataType.TIMESTAMP,
      //       isIndex: true,
      //     });
      //     const ch = await client.channels.create({
      //       name: "alias_channel",
      //       dataType: DataType.FLOAT32,
      //       index: indexCh.key,
      //     });
      //     const range = await client.ranges.create({
      //       name: "alias_range",
      //       timeRange: { start: 1n, end: 1000n },
      //     });
      //     await client.ranges.setAlias(range.key, ch.key, "custom_alias");

      //     const { result } = renderHook(
      //       () => Channel.useRetrieve({ key: ch.key, rangeKey: range.key }),
      //       { wrapper },
      //     );
      //     await waitFor(() => expect(result.current.variant).toEqual("success"));
      //     expect(result.current.data?.alias).toEqual("custom_alias");
      //   });

      //   it("should update when channel is renamed", async () => {
      //     const ch = await client.channels.create({
      //       name: "original_retrieve",
      //       dataType: DataType.FLOAT32,
      //       virtual: true,
      //     });
      //     const { result } = renderHook(
      //       () => {
      //         const retrieve = Channel.useRetrieve({ key: ch.key });
      //         const rename = Channel.useRename();
      //         return { retrieve, rename };
      //       },
      //       { wrapper },
      //     );
      //     await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      //     expect(result.current.retrieve.data?.name).toEqual("original_retrieve");

      //     await act(async () => {
      //       await result.current.rename.updateAsync({
      //         key: ch.key,
      //         name: "renamed_retrieve",
      //       });
      //     });
      //     await waitFor(() => {
      //       expect(result.current.retrieve.data?.name).toEqual("renamed_retrieve");
      //     });
      //   });
      // });

      // describe("useRetrieveMultiple", () => {
      //   it("should retrieve multiple channels by keys", async () => {
      //     const ch1 = await client.channels.create({
      //       name: "retrieve_many_1",
      //       dataType: DataType.FLOAT32,
      //       virtual: true,
      //     });
      //     const ch2 = await client.channels.create({
      //       name: "retrieve_many_2",
      //       dataType: DataType.INT32,
      //       virtual: true,
      //     });
      //     const { result } = renderHook(
      //       () => Channel.useRetrieveMultiple({ keys: [ch1.key, ch2.key] }),
      //       { wrapper },
      //     );
      //     await waitFor(() => expect(result.current.variant).toEqual("success"));
      //     expect(result.current.data).toHaveLength(2);
      //     expect(result.current.data?.map((c) => c.key)).toContain(ch1.key);
      //     expect(result.current.data?.map((c) => c.key)).toContain(ch2.key);
      //   });

      //   it("should retrieve channels with range aliases", async () => {
      //     const indexCh = await client.channels.create({
      //       name: "many_alias_index",
      //       dataType: DataType.TIMESTAMP,
      //       isIndex: true,
      //     });
      //     const ch1 = await client.channels.create({
      //       name: "many_alias_1",
      //       dataType: DataType.FLOAT32,
      //       index: indexCh.key,
      //     });
      //     const ch2 = await client.channels.create({
      //       name: "many_alias_2",
      //       dataType: DataType.FLOAT64,
      //       index: indexCh.key,
      //     });
      //     const range = await client.ranges.create({
      //       name: "many_alias_range",
      //       timeRange: { start: 1n, end: 2000n },
      //     });
      //     await client.ranges.setAlias(range.key, ch1.key, "alias_1");
      //     await client.ranges.setAlias(range.key, ch2.key, "alias_2");

      //     const { result } = renderHook(
      //       () =>
      //         Channel.useRetrieveMultiple({
      //           keys: [ch1.key, ch2.key],
      //           rangeKey: range.key,
      //         }),
      //       { wrapper },
      //     );
      //     await waitFor(() => expect(result.current.variant).toEqual("success"));
      //     const ch1Result = result.current.data?.find((c) => c.key === ch1.key);
      //     const ch2Result = result.current.data?.find((c) => c.key === ch2.key);
      //     expect(ch1Result?.alias).toEqual("alias_1");
      //     expect(ch2Result?.alias).toEqual("alias_2");
      //   });

      //   it("should update when a channel in the list is renamed", async () => {
      //     const ch1 = await client.channels.create({
      //       name: "many_original_1",
      //       dataType: DataType.FLOAT32,
      //       virtual: true,
      //     });
      //     const ch2 = await client.channels.create({
      //       name: "many_original_2",
      //       dataType: DataType.INT64,
      //       virtual: true,
      //     });
      //     const { result } = renderHook(
      //       () => {
      //         const retrieve = Channel.useRetrieveMultiple({ keys: [ch1.key, ch2.key] });
      //         const rename = Channel.useRename();
      //         return { retrieve, rename };
      //       },
      //       { wrapper },
      //     );
      //     await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));

      //     await act(async () => {
      //       await result.current.rename.updateAsync({
      //         key: ch1.key,
      //         name: "many_renamed_1",
      //       });
      //     });
      //     await waitFor(() => {
      //       const updated = result.current.retrieve.data?.find((c) => c.key === ch1.key);
      //       expect(updated?.name).toEqual("many_renamed_1");
      //     });
      //     const unchanged = result.current.retrieve.data?.find((c) => c.key === ch2.key);
      //     expect(unchanged?.name).toEqual("many_original_2");
      //   });
      // });

      // describe("useDelete", () => {
      //   it("should delete a single channel", async () => {
      //     const ch = await client.channels.create({
      //       name: "delete_single",
      //       dataType: DataType.FLOAT32,
      //       virtual: true,
      //     });
      //     const { result } = renderHook(
      //       () => {
      //         const list = Channel.useList();
      //         const del = Channel.useDelete();
      //         return { list, del };
      //       },
      //       { wrapper },
      //     );
      //     act(() => {
      //       result.current.list.retrieve({}, { signal: controller.signal });
      //     });
      //     await waitFor(() => expect(result.current.list.variant).toEqual("success"));
      //     expect(result.current.list.data).toContain(ch.key);

      //     await act(async () => {
      //       await result.current.del.updateAsync(ch.key);
      //     });
      //     await waitFor(() => {
      //       expect(result.current.list.data).not.toContain(ch.key);
      //     });
      //   });

      //   it("should delete multiple channels", async () => {
      //     const ch1 = await client.channels.create({
      //       name: "delete_multi_1",
      //       dataType: DataType.FLOAT32,
      //       virtual: true,
      //     });
      //     const ch2 = await client.channels.create({
      //       name: "delete_multi_2",
      //       dataType: DataType.INT32,
      //       virtual: true,
      //     });
      //     const { result } = renderHook(
      //       () => {
      //         const list = Channel.useList();
      //         const del = Channel.useDelete();
      //         return { list, del };
      //       },
      //       { wrapper },
      //     );
      //     act(() => {
      //       result.current.list.retrieve({}, { signal: controller.signal });
      //     });
      //     await waitFor(() => expect(result.current.list.variant).toEqual("success"));
      //     expect(result.current.list.data).toContain(ch1.key);
      //     expect(result.current.list.data).toContain(ch2.key);

      //     await act(async () => {
      //       await result.current.del.updateAsync([ch1.key, ch2.key]);
      //     });
      //     await waitFor(() => {
      //       expect(result.current.list.data).not.toContain(ch1.key);
      //       expect(result.current.list.data).not.toContain(ch2.key);
      //     });
      //   });

      //   it("should handle delete errors gracefully", async () => {
      //     const nonExistentKey = 999999999;
      //     const { result } = renderHook(() => Channel.useDelete(), { wrapper });

      //     await act(async () => {
      //       try {
      //         await result.current.updateAsync(nonExistentKey);
      //       } catch (error) {
      //         expect(error).toBeDefined();
      //       }
      //     });
      //     expect(result.current.variant).toEqual("error");
      //   });
      // });

      // describe("useUpdateAlias", () => {
      //   it("should update channel alias in a range", async () => {
      //     const indexCh = await client.channels.create({
      //       name: "alias_update_index",
      //       dataType: DataType.TIMESTAMP,
      //       isIndex: true,
      //     });
      //     const ch = await client.channels.create({
      //       name: "alias_update_channel",
      //       dataType: DataType.FLOAT32,
      //       index: indexCh.key,
      //     });
      //     const range = await client.ranges.create({
      //       name: "alias_update_range",
      //       timeRange: { start: 1n, end: 3000n },
      //     });

      //     const { result } = renderHook(
      //       () => {
      //         const retrieve = Channel.useRetrieve({ key: ch.key, rangeKey: range.key });
      //         const updateAlias = Channel.useUpdateAlias();
      //         return { retrieve, updateAlias };
      //       },
      //       { wrapper },
      //     );
      //     await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));

      //     await act(async () => {
      //       await result.current.updateAlias.updateAsync({
      //         range: range.key,
      //         channel: ch.key,
      //         alias: "new_alias",
      //       });
      //     });
      //     await waitFor(() => {
      //       expect(result.current.retrieve.data?.alias).toEqual("new_alias");
      //     });

      //     await act(async () => {
      //       await result.current.updateAlias.updateAsync({
      //         range: range.key,
      //         channel: ch.key,
      //         alias: "updated_alias",
      //       });
      //     });
      //     await waitFor(() => {
      //       expect(result.current.retrieve.data?.alias).toEqual("updated_alias");
      //     });
      //   });
      // });

      // describe("useDeleteAlias", () => {
      //   it("should delete a single channel alias", async () => {
      //     const indexCh = await client.channels.create({
      //       name: "alias_delete_index",
      //       dataType: DataType.TIMESTAMP,
      //       isIndex: true,
      //     });
      //     const ch = await client.channels.create({
      //       name: "alias_delete_channel",
      //       dataType: DataType.FLOAT32,
      //       index: indexCh.key,
      //     });
      //     const range = await client.ranges.create({
      //       name: "alias_delete_range",
      //       timeRange: { start: 1n, end: 4000n },
      //     });
      //     await client.ranges.setAlias(range.key, ch.key, "to_delete");

      //     const { result } = renderHook(
      //       () => {
      //         const retrieve = Channel.useRetrieve({ key: ch.key, rangeKey: range.key });
      //         const deleteAlias = Channel.useDeleteAlias();
      //         return { retrieve, deleteAlias };
      //       },
      //       { wrapper },
      //     );
      //     await waitFor(() => expect(result.current.retrieve.variant).toEqual("success"));
      //     expect(result.current.retrieve.data?.alias).toEqual("to_delete");

      //     await act(async () => {
      //       await result.current.deleteAlias.updateAsync({
      //         range: range.key,
      //         channels: ch.key,
      //       });
      //     });
      //     await waitFor(() => {
      //       expect(result.current.retrieve.data?.alias).toBeUndefined();
      //     });
      //   });

      //   it("should delete multiple channel aliases", async () => {
      //     const indexCh = await client.channels.create({
      //       name: "alias_delete_multi_index",
      //       dataType: DataType.TIMESTAMP,
      //       isIndex: true,
      //     });
      //     const ch1 = await client.channels.create({
      //       name: "alias_delete_multi_1",
      //       dataType: DataType.FLOAT32,
      //       index: indexCh.key,
      //     });
      //     const ch2 = await client.channels.create({
      //       name: "alias_delete_multi_2",
      //       dataType: DataType.INT32,
      //       index: indexCh.key,
      //     });
      //     const range = await client.ranges.create({
      //       name: "alias_delete_multi_range",
      //       timeRange: { start: 1n, end: 5000n },
      //     });
      //     await client.ranges.setAlias(range.key, ch1.key, "multi_alias_1");
      //     await client.ranges.setAlias(range.key, ch2.key, "multi_alias_2");

      //     const { result } = renderHook(
      //       () => {
      //         const retrieve = Channel.useRetrieveMultiple({
      //           keys: [ch1.key, ch2.key],
      //           rangeKey: range.key,
      //         });
      //         const deleteAlias = Channel.useDeleteAlias();
      //         return { retrieve, deleteAlias };
      //       },
      //       { wrapper },
      //     );
      //     await waitFor(() => {
      //       expect(result.current.retrieve.variant).toEqual("success");
      //     });
      //     const ch1Before = result.current.retrieve.data?.find((c) => c.key === ch1.key);
      //     const ch2Before = result.current.retrieve.data?.find((c) => c.key === ch2.key);
      //     expect(ch1Before?.alias).toEqual("multi_alias_1");
      //     expect(ch2Before?.alias).toEqual("multi_alias_2");

      //     await act(async () => {
      //       await result.current.deleteAlias.updateAsync({
      //         range: range.key,
      //         channels: [ch1.key, ch2.key],
      //       });
      //     });
      //     await waitFor(() => {
      //       const ch1After = result.current.retrieve.data?.find((c) => c.key === ch1.key);
      //       const ch2After = result.current.retrieve.data?.find((c) => c.key === ch2.key);
      //       expect(ch1After?.alias).toBeUndefined();
      //       expect(ch2After?.alias).toBeUndefined();
      //     });
      //   });
      // });

      // describe("useRetrieveGroup", () => {
      //   it("should retrieve the channel group", async () => {
      //     const { result } = renderHook(() => Channel.useRetrieveGroup({}), { wrapper });
      //     await waitFor(() => expect(result.current.variant).toEqual("success"));
      //     expect(result.current.data).toBeDefined();
      //     expect(result.current.data?.key).toBeDefined();
      //     expect(result.current.data?.name).toEqual("Channels");
      //   });

      //   it("should cache the group after first retrieval", async () => {
      //     const { result: result1 } = renderHook(() => Channel.useRetrieveGroup({}), {
      //       wrapper,
      //     });
      //     await waitFor(() => expect(result1.current.variant).toEqual("success"));
      //     const firstGroup = result1.current.data;

      //     const { result: result2 } = renderHook(() => Channel.useRetrieveGroup({}), {
      //       wrapper,
      //     });
      //     await waitFor(() => expect(result2.current.variant).toEqual("success"));
      //     expect(result2.current.data?.key).toEqual(firstGroup?.key);
      //   });
      // });

      // describe("useCalculatedForm", () => {
      //   it("should create a new calculated channel", async () => {
      //     const sourceChannel = await client.channels.create({
      //       name: "sourceChannel",
      //       dataType: DataType.FLOAT32,
      //       virtual: true,
      //     });

      //     const { result } = renderHook(() => Channel.useCalculatedForm({ query: {} }), {
      //       wrapper,
      //     });

      //     act(() => {
      //       result.current.form.set("name", "calculatedChannel");
      //       result.current.form.set("dataType", DataType.FLOAT32.toString());
      //       result.current.form.set("virtual", true);
      //       result.current.form.set("expression", "return sourceChannel * 2;");
      //       result.current.form.set("requires", [sourceChannel.key]);
      //       result.current.save({ signal: controller.signal });
      //     });

      //     await waitFor(() => {
      //       expect(result.current.form.value().name).toEqual("calculatedChannel");
      //       expect(result.current.form.value().expression).toEqual(
      //         "return sourceChannel * 2;",
      //       );
      //       expect(result.current.form.value().requires).toEqual([sourceChannel.key]);
      //       expect(result.current.form.value().key).toBeDefined();
      //       expect(result.current.form.value().key).toBeGreaterThan(0);
      //     });
      //   });

      //   it("should retrieve and edit existing calculated channel", async () => {
      //     const sourceChannel = await client.channels.create({
      //       name: "existingSource",
      //       dataType: DataType.FLOAT32,
      //       virtual: true,
      //     });

      //     const existingCalculated = await client.channels.create({
      //       name: "existingCalculated",
      //       dataType: DataType.FLOAT32,
      //       virtual: true,
      //       expression: "return existingSource + 1;",
      //       requires: [sourceChannel.key],
      //     });

      //     const { result } = renderHook(
      //       () => Channel.useCalculatedForm({ query: { key: existingCalculated.key } }),
      //       { wrapper },
      //     );
      //     await waitFor(() => expect(result.current.variant).toEqual("success"));

      //     expect(result.current.form.value().name).toEqual("existingCalculated");
      //     expect(result.current.form.value().expression).toEqual(
      //       "return existingSource + 1;",
      //     );
      //     expect(result.current.form.value().requires).toEqual([sourceChannel.key]);

      //     act(() => {
      //       result.current.form.set("expression", "return existingSource * 3;");
      //       result.current.save({ signal: controller.signal });
      //     });

      //     await waitFor(() => {
      //       expect(result.current.form.value().expression).toEqual(
      //         "return existingSource * 3;",
      //       );
      //     });
      //   });

      //   it("should validate that expression is not empty", async () => {
      //     const { result } = renderHook(() => Channel.useCalculatedForm({ query: {} }), {
      //       wrapper,
      //     });

      //     act(() => {
      //       result.current.form.set("name", "invalidCalculated");
      //       result.current.form.set("expression", "");
      //     });

      //     expect(result.current.form.validate()).toBe(false);
      //     expect(result.current.form.get("expression").status.message).toContain(
      //       "Expression must contain a return statement",
      //     );
      //   });

      //   it("should validate that expression contains return statement", async () => {
      //     const { result } = renderHook(() => Channel.useCalculatedForm({ query: {} }), {
      //       wrapper,
      //     });

      //     act(() => {
      //       result.current.form.set("name", "invalidCalculated");
      //       result.current.form.set("expression", "sourceChannel * 2");
      //     });

      //     expect(result.current.form.validate()).toBe(false);
      //     expect(result.current.form.get("expression").status.message).toContain(
      //       "Expression must contain a return statement",
      //     );
      //   });

      //   it("should validate that expression uses at least one channel", async () => {
      //     const { result } = renderHook(() => Channel.useCalculatedForm({ query: {} }), {
      //       wrapper,
      //     });

      //     act(() => {
      //       result.current.form.set("name", "invalidCalculated");
      //       result.current.form.set("expression", "return 42;");
      //       result.current.form.set("requires", []);
      //     });

      //     expect(result.current.form.validate()).toBe(false);
      //     expect(result.current.form.get("requires").status.message).toContain(
      //       "Expression must use at least one channel",
      //     );
      //   });

      //   it("should handle form with default values", async () => {
      //     const { result } = renderHook(() => Channel.useCalculatedForm({ query: {} }), {
      //       wrapper,
      //     });

      //     expect(result.current.form.value().name).toEqual("");
      //     expect(result.current.form.value().expression).toEqual("");
      //     expect(result.current.form.value().requires).toEqual([]);
      //     expect(result.current.form.value().dataType).toEqual(DataType.FLOAT32.toString());
      //     expect(result.current.form.value().virtual).toBe(false);
      //   });

      //   it("should update form when calculated channel is updated externally", async () => {
      //     const sourceChannel = await client.channels.create({
      //       name: "updateSource",
      //       dataType: DataType.FLOAT32,
      //       virtual: true,
      //     });

      //     const testCalculated = await client.channels.create({
      //       name: "updateCalculated",
      //       dataType: DataType.FLOAT32,
      //       virtual: true,
      //       expression: "return updateSource;",
      //       requires: [sourceChannel.key],
      //     });

      //     const { result } = renderHook(
      //       () => {
      //         const form = Channel.useCalculatedForm({
      //           query: { key: testCalculated.key },
      //         });
      //         const rename = Channel.useRename();
      //         return { form, rename };
      //       },
      //       { wrapper },
      //     );
      //     await waitFor(() => {
      //       expect(result.current.form.variant).toEqual("success");
      //     });
      //     expect(result.current.form.form.value().name).toEqual("updateCalculated");

      //     await act(async () => {
      //       await result.current.rename.updateAsync({
      //         key: testCalculated.key,
      //         name: "externallyUpdatedCalculated",
      //       });
      //     });

      //     await waitFor(() => {
      //       expect(result.current.form.form.value().name).toEqual(
      //         "externallyUpdatedCalculated",
      //       );
      //     });
    });
  });
});
