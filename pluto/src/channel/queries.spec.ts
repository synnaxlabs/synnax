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
import { Ontology } from "@/ontology";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

describe("queries", () => {
  let controller: AbortController;
  const client = createTestClient();
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({
      client,
      excludeFluxStores: [Ontology.RESOURCES_FLUX_STORE_KEY],
    });
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
          () => Channel.useList({ initialParams: { searchTerm: "cached" } }),
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
          () => Channel.useList({ initialParams: { internal: true } }),
          { wrapper },
        );
        expect(secondResult.current.data).not.toContain(normalCh.key);
      });

      it("should filter by calculated channels", async () => {
        const idxCh = await client.channels.create({
          name: "idx_for_calc",
          dataType: DataType.TIMESTAMP,
          isIndex: true,
        });
        const calcCh = await client.channels.create({
          name: "calculated_ch",
          dataType: DataType.FLOAT32,
          virtual: true,
          expression: "return 1",
          requires: [idxCh.key],
        });
        const normalCh = await client.channels.create({
          name: "normal_virtual",
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
        unmount();

        const { result: secondResult } = renderHook(
          () => Channel.useList({ initialParams: { calculated: true } }),
          {
            wrapper,
          },
        );
        expect(secondResult.current.variant).toEqual("loading");
        expect(secondResult.current.data).toContain(calcCh.key);
        expect(secondResult.current.data).not.toContain(normalCh.key);
      });

      it("should filter by dataTypes inclusion", async () => {
        const float32Ch = await client.channels.create({
          name: "float32_ch",
          dataType: DataType.FLOAT32,
          virtual: true,
        });
        const float64Ch = await client.channels.create({
          name: "float64_ch",
          dataType: DataType.FLOAT64,
          virtual: true,
        });

        const { result: firstResult, unmount } = renderHook(() => Channel.useList(), {
          wrapper,
        });
        act(() => {
          firstResult.current.retrieve({}, { signal: controller.signal });
        });
        await waitFor(() => expect(firstResult.current.variant).toEqual("success"));
        unmount();

        const { result: secondResult } = renderHook(
          () => Channel.useList({ initialParams: { dataTypes: [DataType.FLOAT32] } }),
          { wrapper },
        );
        expect(secondResult.current.data).toContain(float32Ch.key);
        expect(secondResult.current.data).not.toContain(float64Ch.key);
      });

      it("should filter by notDataTypes exclusion", async () => {
        const float32Ch = await client.channels.create({
          name: "float32_exclude",
          dataType: DataType.FLOAT32,
          virtual: true,
        });
        const int32Ch = await client.channels.create({
          name: "int32_include",
          dataType: DataType.INT32,
          virtual: true,
        });

        const { result: firstResult, unmount } = renderHook(() => Channel.useList(), {
          wrapper,
        });
        act(() => {
          firstResult.current.retrieve({}, { signal: controller.signal });
        });
        await waitFor(() => expect(firstResult.current.variant).toEqual("success"));
        unmount();

        const { result: secondResult } = renderHook(
          () =>
            Channel.useList({
              initialParams: { notDataTypes: [DataType.FLOAT32] },
            }),
          { wrapper },
        );
        expect(secondResult.current.variant).toEqual("loading");
        expect(secondResult.current.data).not.toContain(float32Ch.key);
        expect(secondResult.current.data).toContain(int32Ch.key);
      });

      it("should filter by isIndex", async () => {
        const indexCh = await client.channels.create({
          name: "index_filter",
          dataType: DataType.TIMESTAMP,
          isIndex: true,
        });
        const dataCh = await client.channels.create({
          name: "data_filter",
          dataType: DataType.FLOAT32,
          index: indexCh.key,
        });

        const { result: firstResult, unmount } = renderHook(() => Channel.useList(), {
          wrapper,
        });
        act(() => {
          firstResult.current.retrieve({}, { signal: controller.signal });
        });
        await waitFor(() => expect(firstResult.current.variant).toEqual("success"));
        unmount();

        const { result: secondResult } = renderHook(
          () => Channel.useList({ initialParams: { isIndex: true } }),
          { wrapper },
        );
        expect(secondResult.current.variant).toEqual("loading");
        expect(secondResult.current.data).toContain(indexCh.key);
        expect(secondResult.current.data).not.toContain(dataCh.key);
      });

      it("should filter by virtual", async () => {
        const indexCh = await client.channels.create({
          name: "index_virt",
          dataType: DataType.TIMESTAMP,
          isIndex: true,
        });
        const virtualCh = await client.channels.create({
          name: "virtual_filter",
          dataType: DataType.FLOAT32,
          virtual: true,
        });
        const persistedCh = await client.channels.create({
          name: "persisted_filter",
          dataType: DataType.FLOAT32,
          index: indexCh.key,
          virtual: false,
        });

        const { result: firstResult, unmount } = renderHook(() => Channel.useList(), {
          wrapper,
        });
        act(() => {
          firstResult.current.retrieve({}, { signal: controller.signal });
        });
        await waitFor(() => expect(firstResult.current.variant).toEqual("success"));
        unmount();

        const { result: secondResult } = renderHook(
          () => Channel.useList({ initialParams: { virtual: true } }),
          { wrapper },
        );
        expect(secondResult.current.variant).toEqual("loading");
        expect(secondResult.current.data).toContain(virtualCh.key);
        expect(secondResult.current.data).not.toContain(persistedCh.key);
      });

      it("should handle combined filters", async () => {
        const indexCh = await client.channels.create({
          name: "idx_combined",
          dataType: DataType.TIMESTAMP,
          isIndex: true,
        });
        const virtualFloat32Ch = await client.channels.create({
          name: "virtual_float32",
          dataType: DataType.FLOAT32,
          virtual: true,
        });
        const virtualInt32Ch = await client.channels.create({
          name: "virtual_int32",
          dataType: DataType.INT32,
          virtual: true,
        });
        const persistedFloat32Ch = await client.channels.create({
          name: "persisted_float32",
          dataType: DataType.FLOAT32,
          index: indexCh.key,
          virtual: false,
        });

        const { result: firstResult, unmount } = renderHook(() => Channel.useList(), {
          wrapper,
        });
        act(() => {
          firstResult.current.retrieve({}, { signal: controller.signal });
        });
        await waitFor(() => expect(firstResult.current.variant).toEqual("success"));
        unmount();

        const { result: secondResult } = renderHook(
          () =>
            Channel.useList({
              initialParams: {
                virtual: true,
                dataTypes: [DataType.FLOAT32],
                internal: false,
              },
            }),
          { wrapper },
        );
        expect(secondResult.current.variant).toEqual("loading");
        expect(secondResult.current.data).toContain(virtualFloat32Ch.key);
        expect(secondResult.current.data).not.toContain(virtualInt32Ch.key);
        expect(secondResult.current.data).not.toContain(persistedFloat32Ch.key);
      });
    });
  });

  describe("useForm", () => {
    it("should create a new virtual channel", async () => {
      const { result } = renderHook(() => Channel.useForm({ params: {} }), {
        wrapper,
      });

      act(() => {
        result.current.form.set("name", "newFormChannel");
        result.current.form.set("dataType", DataType.FLOAT32.toString());
        result.current.form.set("virtual", true);
        result.current.save({ signal: controller.signal });
      });

      await waitFor(() => {
        expect(result.current.form.value().name).toEqual("newFormChannel");
        expect(result.current.form.value().dataType).toEqual(
          DataType.FLOAT32.toString(),
        );
        expect(result.current.form.value().virtual).toBe(true);
        expect(result.current.form.value().key).toBeDefined();
        expect(result.current.form.value().key).toBeGreaterThan(0);
      });
    });

    it("should create a new index channel", async () => {
      const { result } = renderHook(() => Channel.useForm({ params: {} }), {
        wrapper,
      });

      act(() => {
        result.current.form.set("name", "newIndexChannel");
        result.current.form.set("dataType", DataType.TIMESTAMP.toString());
        result.current.form.set("isIndex", true);
        result.current.save({ signal: controller.signal });
      });

      await waitFor(() => {
        expect(result.current.form.value().name).toEqual("newIndexChannel");
        expect(result.current.form.value().dataType).toEqual(
          DataType.TIMESTAMP.toString(),
        );
        expect(result.current.form.value().isIndex).toBe(true);
        expect(result.current.form.value().key).toBeDefined();
        expect(result.current.form.value().key).toBeGreaterThan(0);
      });
    });

    it("should create a new data channel with index", async () => {
      const indexChannel = await client.channels.create({
        name: "test_index",
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });

      const { result } = renderHook(() => Channel.useForm({ params: {} }), {
        wrapper,
      });

      act(() => {
        result.current.form.set("name", "newDataChannel");
        result.current.form.set("dataType", DataType.FLOAT32.toString());
        result.current.form.set("index", indexChannel.key);
        result.current.save({ signal: controller.signal });
      });

      await waitFor(() => {
        expect(result.current.form.value().name).toEqual("newDataChannel");
        expect(result.current.form.value().dataType).toEqual(
          DataType.FLOAT32.toString(),
        );
        expect(result.current.form.value().index).toEqual(indexChannel.key);
        expect(result.current.form.value().key).toBeDefined();
        expect(result.current.form.value().key).toBeGreaterThan(0);
      });
    });

    it("should retrieve and edit existing channel", async () => {
      const existingChannel = await client.channels.create({
        name: "existingChannel",
        dataType: DataType.FLOAT64,
        virtual: true,
      });

      const { result } = renderHook(
        () => Channel.useForm({ params: { key: existingChannel.key } }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      expect(result.current.form.value().name).toEqual("existingChannel");
      expect(result.current.form.value().dataType).toEqual(DataType.FLOAT64.toString());
      expect(result.current.form.value().virtual).toBe(true);

      act(() => {
        result.current.form.set("name", "editedChannel");
        result.current.save({ signal: controller.signal });
      });

      await waitFor(() => {
        expect(result.current.form.value().name).toEqual("editedChannel");
      });
    });

    it("should update form when channel is updated externally", async () => {
      const testChannel = await client.channels.create({
        name: "externalUpdate",
        dataType: DataType.FLOAT32,
        virtual: true,
      });

      const { result } = renderHook(
        () => {
          const form = Channel.useForm({ params: { key: testChannel.key } });
          const rename = Channel.useRename();
          return { form, rename };
        },
        { wrapper },
      );
      await waitFor(() => expect(result.current.form.variant).toEqual("success"));
      expect(result.current.form.form.value().name).toEqual("externalUpdate");

      await act(async () => {
        await result.current.rename.updateAsync({
          key: testChannel.key,
          name: "externallyUpdated",
        });
      });

      await waitFor(() => {
        expect(result.current.form.form.value().name).toEqual("externallyUpdated");
      });
    });

    it("should handle form with default values", async () => {
      const { result } = renderHook(() => Channel.useForm({ params: {} }), {
        wrapper,
      });

      expect(result.current.form.value().name).toEqual("");
      expect(result.current.form.value().dataType).toEqual(DataType.FLOAT32.toString());
      expect(result.current.form.value().virtual).toBe(false);
      expect(result.current.form.value().isIndex).toBe(false);
      expect(result.current.form.value().index).toEqual(0);
    });

    it("should validate that index channels have timestamp data type", async () => {
      const { result } = renderHook(() => Channel.useForm({ params: {} }), {
        wrapper,
      });

      act(() => {
        result.current.form.set("name", "invalidIndex");
        result.current.form.set("dataType", DataType.FLOAT32.toString());
        result.current.form.set("isIndex", true);
      });

      expect(result.current.form.validate()).toBe(false);
      expect(result.current.form.get("dataType").status.message).toContain(
        "Index channel must have data type TIMESTAMP",
      );
    });

    it("should validate that data channels have an index or are virtual", async () => {
      const { result } = renderHook(() => Channel.useForm({ params: {} }), {
        wrapper,
      });

      act(() => {
        result.current.form.set("name", "invalidDataChannel");
        result.current.form.set("dataType", DataType.FLOAT32.toString());
        result.current.form.set("isIndex", false);
        result.current.form.set("index", 0);
        result.current.form.set("virtual", false);
      });

      expect(result.current.form.validate()).toBe(false);
      expect(result.current.form.get("index").status.message).toContain(
        "Data channel must have an index",
      );
    });

    it("should validate that persisted channels have fixed-size data types", async () => {
      const { result } = renderHook(() => Channel.useForm({ params: {} }), {
        wrapper,
      });

      act(() => {
        result.current.form.set("name", "invalidPersistedChannel");
        result.current.form.set("dataType", DataType.STRING.toString());
        result.current.form.set("virtual", false);
        result.current.form.set("isIndex", false);
      });

      expect(result.current.form.validate()).toBe(false);
      expect(result.current.form.get("dataType").status.message).toContain(
        "Persisted channels must have a fixed-size data type",
      );
    });
  });

  describe("useCalculatedForm", () => {
    it("should create a new calculated channel", async () => {
      const sourceChannel = await client.channels.create({
        name: "sourceChannel",
        dataType: DataType.FLOAT32,
        virtual: true,
      });

      const { result } = renderHook(() => Channel.useCalculatedForm({ params: {} }), {
        wrapper,
      });

      act(() => {
        result.current.form.set("name", "calculatedChannel");
        result.current.form.set("dataType", DataType.FLOAT32.toString());
        result.current.form.set("virtual", true);
        result.current.form.set("expression", "return sourceChannel * 2;");
        result.current.form.set("requires", [sourceChannel.key]);
        result.current.save({ signal: controller.signal });
      });

      await waitFor(() => {
        expect(result.current.form.value().name).toEqual("calculatedChannel");
        expect(result.current.form.value().expression).toEqual(
          "return sourceChannel * 2;",
        );
        expect(result.current.form.value().requires).toEqual([sourceChannel.key]);
        expect(result.current.form.value().key).toBeDefined();
        expect(result.current.form.value().key).toBeGreaterThan(0);
      });
    });

    it("should retrieve and edit existing calculated channel", async () => {
      const sourceChannel = await client.channels.create({
        name: "existingSource",
        dataType: DataType.FLOAT32,
        virtual: true,
      });

      const existingCalculated = await client.channels.create({
        name: "existingCalculated",
        dataType: DataType.FLOAT32,
        virtual: true,
        expression: "return existingSource + 1;",
        requires: [sourceChannel.key],
      });

      const { result } = renderHook(
        () => Channel.useCalculatedForm({ params: { key: existingCalculated.key } }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      expect(result.current.form.value().name).toEqual("existingCalculated");
      expect(result.current.form.value().expression).toEqual(
        "return existingSource + 1;",
      );
      expect(result.current.form.value().requires).toEqual([sourceChannel.key]);

      act(() => {
        result.current.form.set("expression", "return existingSource * 3;");
        result.current.save({ signal: controller.signal });
      });

      await waitFor(() => {
        expect(result.current.form.value().expression).toEqual(
          "return existingSource * 3;",
        );
      });
    });

    it("should validate that expression is not empty", async () => {
      const { result } = renderHook(() => Channel.useCalculatedForm({ params: {} }), {
        wrapper,
      });

      act(() => {
        result.current.form.set("name", "invalidCalculated");
        result.current.form.set("expression", "");
      });

      expect(result.current.form.validate()).toBe(false);
      expect(result.current.form.get("expression").status.message).toContain(
        "Expression must contain a return statement",
      );
    });

    it("should validate that expression contains return statement", async () => {
      const { result } = renderHook(() => Channel.useCalculatedForm({ params: {} }), {
        wrapper,
      });

      act(() => {
        result.current.form.set("name", "invalidCalculated");
        result.current.form.set("expression", "sourceChannel * 2");
      });

      expect(result.current.form.validate()).toBe(false);
      expect(result.current.form.get("expression").status.message).toContain(
        "Expression must contain a return statement",
      );
    });

    it("should validate that expression uses at least one channel", async () => {
      const { result } = renderHook(() => Channel.useCalculatedForm({ params: {} }), {
        wrapper,
      });

      act(() => {
        result.current.form.set("name", "invalidCalculated");
        result.current.form.set("expression", "return 42;");
        result.current.form.set("requires", []);
      });

      expect(result.current.form.validate()).toBe(false);
      expect(result.current.form.get("requires").status.message).toContain(
        "Expression must use at least one channel",
      );
    });

    it("should handle form with default values", async () => {
      const { result } = renderHook(() => Channel.useCalculatedForm({ params: {} }), {
        wrapper,
      });

      expect(result.current.form.value().name).toEqual("");
      expect(result.current.form.value().expression).toEqual("");
      expect(result.current.form.value().requires).toEqual([]);
      expect(result.current.form.value().dataType).toEqual(DataType.FLOAT32.toString());
      expect(result.current.form.value().virtual).toBe(false);
    });

    it("should update form when calculated channel is updated externally", async () => {
      const sourceChannel = await client.channels.create({
        name: "updateSource",
        dataType: DataType.FLOAT32,
        virtual: true,
      });

      const testCalculated = await client.channels.create({
        name: "updateCalculated",
        dataType: DataType.FLOAT32,
        virtual: true,
        expression: "return updateSource;",
        requires: [sourceChannel.key],
      });

      const { result } = renderHook(
        () => {
          const form = Channel.useCalculatedForm({
            params: { key: testCalculated.key },
          });
          const rename = Channel.useRename();
          return { form, rename };
        },
        { wrapper },
      );
      await waitFor(() => {
        expect(result.current.form.variant).toEqual("success");
      });
      expect(result.current.form.form.value().name).toEqual("updateCalculated");

      await act(async () => {
        await result.current.rename.updateAsync({
          key: testCalculated.key,
          name: "externallyUpdatedCalculated",
        });
      });

      await waitFor(() => {
        expect(result.current.form.form.value().name).toEqual(
          "externallyUpdatedCalculated",
        );
      });
    });
  });
});
