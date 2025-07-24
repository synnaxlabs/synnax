import { type channel, DataType, newTestClient } from "@synnaxlabs/client";
import { act, renderHook, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Channel } from "@/channel";
import { newSynnaxWrapper } from "@/testutil/Synnax";

const client = newTestClient();

describe("queries", () => {
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
        wrapper: newSynnaxWrapper(client),
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
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
        wrapper: newSynnaxWrapper(client),
      });
      act(() => {
        result.current.retrieve({});
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
        wrapper: newSynnaxWrapper(client),
      });
      act(() => {
        result.current.retrieve({ term: "special" });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data.length).toBeGreaterThanOrEqual(1);
      expect(
        result.current.data
          .map((key: channel.Key) => result.current.getItem(key)?.name)
          .includes("special_channel"),
      ).toBeTruthy();
    });

    it("should handle pagination with limit and offset", async () => {
      for (let i = 0; i < 5; i++)
        await client.channels.create({
          name: `paginationChannel${i}`,
          dataType: DataType.FLOAT32,
          virtual: true,
        });

      const { result } = renderHook(() => Channel.useList(), {
        wrapper: newSynnaxWrapper(client),
      });
      act(() => {
        result.current.retrieve({ limit: 2, offset: 1 });
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.data).toHaveLength(2);
    });

    it("should update the list when a channel is created", async () => {
      const { result } = renderHook(() => Channel.useList(), {
        wrapper: newSynnaxWrapper(client),
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
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

      const { result } = renderHook(() => Channel.useList(), {
        wrapper: newSynnaxWrapper(client),
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.getItem(testChannel.key)?.name).toEqual("original");

      await client.channels.rename(testChannel.key, "updated");

      await waitFor(() => {
        expect(result.current.getItem(testChannel.key)?.name).toEqual("updated");
      });
    });

    it("should remove channel from list when deleted", async () => {
      const testChannel = await client.channels.create({
        name: "toDelete",
        dataType: DataType.FLOAT32,
        virtual: true,
      });

      const { result } = renderHook(() => Channel.useList(), {
        wrapper: newSynnaxWrapper(client),
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));
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
        wrapper: newSynnaxWrapper(client),
      });
      act(() => {
        result.current.retrieve({});
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
        wrapper: newSynnaxWrapper(client),
      });
      act(() => {
        result.current.retrieve({});
      });
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      const retrievedChannel = result.current.getItem(virtualChannel.key);
      expect(retrievedChannel?.virtual).toBe(true);
    });
  });

  describe("useForm", () => {
    it("should create a new virtual channel", async () => {
      const { result } = renderHook(() => Channel.useForm({ params: {} }), {
        wrapper: newSynnaxWrapper(client),
      });

      act(() => {
        result.current.form.set("name", "newFormChannel");
        result.current.form.set("dataType", DataType.FLOAT32.toString());
        result.current.form.set("virtual", true);
        result.current.save();
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
        wrapper: newSynnaxWrapper(client),
      });

      act(() => {
        result.current.form.set("name", "newIndexChannel");
        result.current.form.set("dataType", DataType.TIMESTAMP.toString());
        result.current.form.set("isIndex", true);
        result.current.save();
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
        wrapper: newSynnaxWrapper(client),
      });

      act(() => {
        result.current.form.set("name", "newDataChannel");
        result.current.form.set("dataType", DataType.FLOAT32.toString());
        result.current.form.set("index", indexChannel.key);
        result.current.save();
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
        { wrapper: newSynnaxWrapper(client) },
      );
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      expect(result.current.form.value().name).toEqual("existingChannel");
      expect(result.current.form.value().dataType).toEqual(DataType.FLOAT64.toString());
      expect(result.current.form.value().virtual).toBe(true);

      act(() => {
        result.current.form.set("name", "editedChannel");
        result.current.save();
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
        () => Channel.useForm({ params: { key: testChannel.key } }),
        { wrapper: newSynnaxWrapper(client) },
      );
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.form.value().name).toEqual("externalUpdate");

      await client.channels.rename(testChannel.key, "externallyUpdated");

      await waitFor(() => {
        expect(result.current.form.value().name).toEqual("externallyUpdated");
      });
    });

    it("should handle form with default values", async () => {
      const { result } = renderHook(() => Channel.useForm({ params: {} }), {
        wrapper: newSynnaxWrapper(client),
      });

      expect(result.current.form.value().name).toEqual("");
      expect(result.current.form.value().dataType).toEqual(DataType.FLOAT32.toString());
      expect(result.current.form.value().virtual).toBe(false);
      expect(result.current.form.value().isIndex).toBe(false);
      expect(result.current.form.value().index).toEqual(0);
    });

    it("should validate that index channels have timestamp data type", async () => {
      const { result } = renderHook(() => Channel.useForm({ params: {} }), {
        wrapper: newSynnaxWrapper(client),
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
        wrapper: newSynnaxWrapper(client),
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
        wrapper: newSynnaxWrapper(client),
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
        wrapper: newSynnaxWrapper(client),
      });

      act(() => {
        result.current.form.set("name", "calculatedChannel");
        result.current.form.set("dataType", DataType.FLOAT32.toString());
        result.current.form.set("virtual", true);
        result.current.form.set("expression", "return sourceChannel * 2;");
        result.current.form.set("requires", [sourceChannel.key]);
        result.current.save();
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
        { wrapper: newSynnaxWrapper(client) },
      );
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      expect(result.current.form.value().name).toEqual("existingCalculated");
      expect(result.current.form.value().expression).toEqual(
        "return existingSource + 1;",
      );
      expect(result.current.form.value().requires).toEqual([sourceChannel.key]);

      act(() => {
        result.current.form.set("expression", "return existingSource * 3;");
        result.current.save();
      });

      await waitFor(() => {
        expect(result.current.form.value().expression).toEqual(
          "return existingSource * 3;",
        );
      });
    });

    it("should validate that expression is not empty", async () => {
      const { result } = renderHook(() => Channel.useCalculatedForm({ params: {} }), {
        wrapper: newSynnaxWrapper(client),
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
        wrapper: newSynnaxWrapper(client),
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
        wrapper: newSynnaxWrapper(client),
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
        wrapper: newSynnaxWrapper(client),
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
        () => Channel.useCalculatedForm({ params: { key: testCalculated.key } }),
        { wrapper: newSynnaxWrapper(client) },
      );
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      expect(result.current.form.value().name).toEqual("updateCalculated");

      await client.channels.rename(testCalculated.key, "externallyUpdatedCalculated");

      await waitFor(() => {
        expect(result.current.form.value().name).toEqual("externallyUpdatedCalculated");
      });
    });
  });
});
