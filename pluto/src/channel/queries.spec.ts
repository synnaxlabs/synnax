// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType, project, type Synnax } from "@synnaxlabs/client";
import {
  builtInsFetches,
  createTestClient,
  holdResponses,
  isBuiltInsFetch,
  spyOnSend,
  waitForStreamLive,
} from "@synnaxlabs/client/testutil";
import { Icon } from "@synnaxlabs/lyra/icon";
import { id, TimeSpan } from "@synnaxlabs/x";
import { act, renderHook, waitFor } from "@testing-library/react";
import type { FC, PropsWithChildren } from "react";
import { afterEach, beforeAll, beforeEach, describe, expect, it } from "vitest";

import { Channel } from "@/channel";
import { createAsyncSynnaxWrapper, renderHookSuspended } from "@/testutil";

const path = Channel.formPath;
const calculatedPath = Channel.calculatedFormPath;

const SCROLL_COUNT = 1010;

// Scrolls a channel list of fresh channels to its end, as a channel select does.
const scrollToEnd = async (client: Synnax) => {
  const prefix = `scroll_${id.create()}_`;
  await createTestClient().channels.create(
    Array.from({ length: SCROLL_COUNT }, (_, i) => ({
      name: `${prefix}${i}`,
      dataType: DataType.FLOAT32,
      virtual: true,
    })),
  );
  const wrapper = await createAsyncSynnaxWrapper({ client });
  const { result } = renderHook(
    () => Channel.useList({ query: { where: { name: { startsWith: prefix } } } }),
    { wrapper },
  );
  for (
    let i = 0;
    i < 40 && !(result.current.answered && result.current.complete);
    i++
  ) {
    act(() => result.current.more());
    await waitFor(() => expect(result.current.variant).toBe("success"));
  }
  return { length: result.current.data.length, complete: result.current.complete };
};

describe("useList with the built-in keys", () => {
  it("should scroll to the end once the built-in keys arrived", async () => {
    const client = createTestClient();
    const send = spyOnSend(client);
    await client.connect();
    await waitForStreamLive(client.connection);
    await expect.poll(() => builtInsFetches(send).length).toBeGreaterThan(0);
    await builtInsFetches(send)[0];
    expect(await scrollToEnd(client)).toEqual({ length: SCROLL_COUNT, complete: true });
  }, 60000);

  it("should scroll to the end before the built-in keys arrive", async () => {
    const client = createTestClient();
    holdResponses(client, isBuiltInsFetch);
    await client.connect();
    await waitForStreamLive(client.connection);
    expect(await scrollToEnd(client)).toEqual({ length: SCROLL_COUNT, complete: true });
  }, 60000);
});

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
    it("should leave out internal channels unless the query sets where.internal", async () => {
      const ch = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      const name = { in: [ch.name, project.ACTION_CHANNEL_NAME] };
      const { result } = renderHook(
        () => ({
          visible: Channel.useList({ query: { where: { name } } }),
          internal: Channel.useList({
            query: { where: { name, internal: { eq: true } } },
          }),
        }),
        { wrapper },
      );
      act(() => {
        result.current.visible.more();
        result.current.internal.more();
      });
      await waitFor(() => {
        expect(result.current.visible.data).toEqual([ch.key]);
        expect(result.current.internal.data.length).toBeGreaterThan(0);
      });
      expect(result.current.internal.data).not.toContain(ch.key);
    });
  });

  describe("useResultCountLimited", () => {
    // Other specs change channels on the same Core, so compare with a fresh count.
    const expectCount = async (data: number | undefined) =>
      expect(data).toBe(await client.channels.countLimited());

    const createIndex = async () =>
      await client.channels.create({
        name: id.create(),
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });

    it("should fetch the count again when a channel is created or deleted", async () => {
      const { result } = renderHook(() => Channel.useResultCountLimited(), { wrapper });
      await waitFor(async () => await expectCount(result.current.data));
      const ch = await createIndex();
      await waitFor(async () => await expectCount(result.current.data));
      await client.channels.delete(ch.key);
      await waitFor(async () => await expectCount(result.current.data));
    });

    it("should fetch the count again on a new mount", async () => {
      const first = renderHook(() => Channel.useResultCountLimited(), { wrapper });
      await waitFor(async () => await expectCount(first.result.current.data));
      first.unmount();
      await createIndex();
      const { result } = renderHook(() => Channel.useResultCountLimited(), { wrapper });
      await waitFor(async () => await expectCount(result.current.data));
    });
  });

  describe("useForm", () => {
    it("should create a new virtual channel", async () => {
      const { result } = renderHook(() => Channel.useForm({ query: null }), {
        wrapper,
      });
      const name = id.create();

      act(() => {
        result.current.form.set(path.name, name);
        result.current.form.set(path.dataType, DataType.FLOAT32.toString());
        result.current.form.set(path.virtual, true);
        result.current.save({ signal: controller.signal });
      });

      await waitFor(() => {
        expect(result.current.form.value().name).toEqual(name);
        expect(result.current.form.value().dataType).toEqual(
          DataType.FLOAT32.toString(),
        );
        expect(result.current.form.value().virtual).toBe(true);
        expect(result.current.form.value().key).toBeDefined();
        expect(result.current.form.value().key).toBeGreaterThan(0);
      });
    });

    it("should create a new index channel", async () => {
      const { result } = renderHook(() => Channel.useForm({ query: null }), {
        wrapper,
      });

      const name = id.create();
      act(() => {
        result.current.form.set(path.name, name);
        result.current.form.set(path.dataType, DataType.TIMESTAMP.toString());
        result.current.form.set(path.isIndex, true);
        result.current.save({ signal: controller.signal });
      });

      await waitFor(() => {
        expect(result.current.form.value().name).toEqual(name);
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
        name: id.create(),
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });

      const { result } = renderHook(() => Channel.useForm({ query: null }), {
        wrapper,
      });

      const name = id.create();
      act(() => {
        result.current.form.set(path.name, name);
        result.current.form.set(path.dataType, DataType.FLOAT32.toString());
        result.current.form.set(path.index, indexChannel.key);
        result.current.save({ signal: controller.signal });
      });

      await waitFor(() => {
        expect(result.current.form.value().name).toEqual(name);
        expect(result.current.form.value().dataType).toEqual(
          DataType.FLOAT32.toString(),
        );
        expect(result.current.form.value().index).toEqual(indexChannel.key);
        expect(result.current.form.value().key).toBeDefined();
        expect(result.current.form.value().key).toBeGreaterThan(0);
      });
    });

    it("should retrieve and edit existing channel", async () => {
      const name = id.create();
      const existingChannel = await client.channels.create({
        name,
        dataType: DataType.FLOAT64,
        virtual: true,
      });

      const { result } = renderHook(
        () => Channel.useForm({ query: { key: existingChannel.key } }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      expect(result.current.form.value().name).toEqual(name);
      expect(result.current.form.value().dataType).toEqual(DataType.FLOAT64.toString());
      expect(result.current.form.value().virtual).toBe(true);

      const updatedName = id.create();

      act(() => {
        result.current.form.set(path.name, updatedName);
        result.current.save({ signal: controller.signal });
      });

      await waitFor(() => {
        expect(result.current.form.value().name).toEqual(updatedName);
      });
    });

    it("should update form when channel is updated externally", async () => {
      const name = id.create();
      const testChannel = await client.channels.create({
        name,
        dataType: DataType.FLOAT32,
        virtual: true,
      });

      const { result } = renderHook(
        () => {
          const form = Channel.useForm({ query: { key: testChannel.key } });
          const rename = Channel.useRename();
          return { form, rename };
        },
        { wrapper },
      );
      await waitFor(() => expect(result.current.form.variant).toEqual("success"));
      expect(result.current.form.form.value().name).toEqual(testChannel.name);

      const updatedName = id.create();
      await act(async () => {
        await result.current.rename.updateAsync({
          key: testChannel.key,
          name: updatedName,
        });
      });

      await waitFor(() => {
        expect(result.current.form.form.value().name).toEqual(updatedName);
      });
    });

    it("should handle form with default values", async () => {
      const { result } = renderHook(() => Channel.useForm({ query: null }), {
        wrapper,
      });

      expect(result.current.form.value().name).toEqual("");
      expect(result.current.form.value().dataType).toEqual(DataType.FLOAT32.toString());
      expect(result.current.form.value().virtual).toBe(false);
      expect(result.current.form.value().isIndex).toBe(false);
      expect(result.current.form.value().index).toEqual(0);
    });

    it("should validate that index channels have timestamp data type", async () => {
      const { result } = renderHook(() => Channel.useForm({ query: null }), {
        wrapper,
      });

      act(() => {
        result.current.form.set(path.name, id.create());
        result.current.form.set(path.dataType, DataType.FLOAT32.toString());
        result.current.form.set(path.isIndex, true);
      });

      expect(result.current.form.validate()).toBe(false);
      expect(result.current.form.get(path.dataType).status.message).toContain(
        "Index channel must have data type TIMESTAMP",
      );
    });

    it("should validate that data channels have an index or are virtual", async () => {
      const { result } = renderHook(() => Channel.useForm({ query: null }), {
        wrapper,
      });

      act(() => {
        result.current.form.set(path.name, id.create());
        result.current.form.set(path.dataType, DataType.FLOAT32.toString());
        result.current.form.set(path.isIndex, false);
        result.current.form.set(path.index, 0);
        result.current.form.set(path.virtual, false);
      });

      expect(result.current.form.validate()).toBe(false);
      expect(result.current.form.get(path.index).status.message).toContain(
        "Data channel must have an index",
      );
    });

    it("should validate that name cannot be empty", async () => {
      const { result } = renderHook(() => Channel.useForm({ query: null }), {
        wrapper,
      });

      act(() => {
        result.current.form.set(path.name, "");
        result.current.form.set(path.virtual, true);
      });

      expect(result.current.form.validate()).toBe(false);
      expect(result.current.form.get(path.name).status.message).toContain(
        "Name is required",
      );
    });

    it("should accept valid names with letters, digits, and underscores", async () => {
      const indexChannel = await client.channels.create({
        name: id.create(),
        dataType: DataType.TIMESTAMP,
        isIndex: true,
      });

      const { result } = renderHook(() => Channel.useForm({ query: null }), {
        wrapper,
      });

      act(() => {
        result.current.form.set(path.name, "sensor_temp_123");
        result.current.form.set(path.dataType, DataType.FLOAT32.toString());
        result.current.form.set(path.index, indexChannel.key);
      });

      expect(result.current.form.validate()).toBe(true);
      expect(result.current.form.get(path.name).status.variant).not.toBe("error");
    });
  });

  describe("use", () => {
    it("should retrieve a channel by key", async () => {
      const name = id.create();
      const ch = await client.channels.create({
        name,
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      const { result } = await renderHookSuspended(() => Channel.use({ key: ch.key }), {
        wrapper,
      });
      await waitFor(() => expect(result.current).not.toBeNull());
      expect(result.current?.key).toEqual(ch.key);
      expect(result.current?.name).toEqual(name);
    });

    it("should update when channel is renamed", async () => {
      const name = id.create();
      const ch = await client.channels.create({
        name,
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      const { result } = await renderHookSuspended(
        () => {
          const retrieve = Channel.use({ key: ch.key });
          const rename = Channel.useRename();
          return { retrieve, rename };
        },
        { wrapper },
      );
      await waitFor(() => expect(result.current.retrieve).not.toBeNull());
      expect(result.current.retrieve?.name).toEqual(name);

      const updatedName = id.create();

      await act(async () => {
        await result.current.rename.updateAsync({
          key: ch.key,
          name: updatedName,
        });
      });
      await waitFor(() => {
        expect(result.current.retrieve?.name).toEqual(updatedName);
      });
    });

    it("should not fetch for plain channels", async () => {
      const plain = await Promise.all(
        [0, 1, 2].map(
          async () =>
            await client.channels.create({
              name: id.create(),
              dataType: DataType.FLOAT32,
              virtual: true,
            }),
        ),
      );
      const local = createTestClient();
      await local.channels.retrieve(plain.map(({ key }) => key));
      const localWrapper = await createAsyncSynnaxWrapper({ client: local });
      const send = spyOnSend(local);
      const { result } = renderHook(
        () => plain.map(({ key }) => Channel.use({ key })),
        { wrapper: localWrapper },
      );
      expect(result.current.map((ch) => ch?.key)).toEqual(plain.map(({ key }) => key));
      const unaryTargets = send.mock.calls.map(([target]) => target);
      expect(unaryTargets).not.toContain("/channel/retrieve");
      expect(unaryTargets).not.toContain("/status/retrieve");
    });
  });

  describe("useKind", () => {
    it("should read the icon and flags of a channel and keep them across a rename", async () => {
      const ch = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      const { result } = renderHook(
        () => ({
          kind: Channel.useKind({ key: ch.key }),
          name: Channel.useResultName({ key: ch.key }),
        }),
        { wrapper },
      );
      await waitFor(() =>
        expect(result.current.kind).toEqual({
          icon: Channel.resolveIcon(ch.payload),
          internal: false,
          virtual: true,
        }),
      );
      const before = result.current.kind;
      const name = id.create();
      await act(async () => {
        await client.channels.rename(ch.key, name);
      });
      await waitFor(() => expect(result.current.name.data).toEqual(name));
      expect(result.current.kind).toBe(before);
    });

    it("should show a calculation icon for a channel that a calculation writes", async () => {
      const calc = await client.calculations.create({
        name: id.create(),
        expression: "return 2",
      });
      const { result } = renderHook(() => Channel.useKind({ key: calc.output }), {
        wrapper,
      });
      await waitFor(() => expect(result.current?.icon).toBe(Icon.Calculation));
    });
  });

  describe("useResultList", () => {
    it("should search channels and follow a new match", async () => {
      const token = `rl${id.create()}`;
      const first = await client.channels.create({
        name: `${token}_a`,
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      const { result } = renderHook(
        () => Channel.useResultList({ searchTerm: token, limit: 10 }),
        { wrapper },
      );
      await waitFor(() =>
        expect(result.current.data?.map((c) => c.key)).toEqual([first.key]),
      );
      const second = await client.channels.create({
        name: `${token}_b`,
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      await waitFor(() =>
        expect(result.current.data?.map((c) => c.key).sort()).toEqual(
          [first.key, second.key].sort(),
        ),
      );
    });
  });

  describe("useMultiple", () => {
    it("should retrieve multiple channels by keys", async () => {
      const ch1 = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      const ch2 = await client.channels.create({
        name: id.create(),
        dataType: DataType.INT32,
        virtual: true,
      });
      const { result } = await renderHookSuspended(
        () => Channel.useMultiple({ where: { key: { in: [ch1.key, ch2.key] } } }),
        { wrapper },
      );
      await waitFor(() => expect(result.current).not.toBeNull());
      expect(result.current).toHaveLength(2);
      expect(result.current?.map((c) => c.key)).toContain(ch1.key);
      expect(result.current?.map((c) => c.key)).toContain(ch2.key);
    });

    it("should fetch the authoritative answer when only part of it is cached", async () => {
      const ch1 = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      // A second client creates ch2 so this client's record store holds only ch1: a
      // cached partial answer must not suppress the fetch of the full one.
      const ch2 = await createTestClient().channels.create({
        name: id.create(),
        dataType: DataType.INT32,
        virtual: true,
      });
      const { result } = await renderHookSuspended(
        () => Channel.useMultiple({ where: { key: { in: [ch1.key, ch2.key] } } }),
        { wrapper },
      );
      await waitFor(() => expect(result.current).toHaveLength(2));
    });

    it("should update when a channel in the list is renamed", async () => {
      const ch1 = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      const orig2 = id.create();
      const ch2 = await client.channels.create({
        name: orig2,
        dataType: DataType.INT64,
        virtual: true,
      });
      const { result } = await renderHookSuspended(
        () => {
          const retrieve = Channel.useMultiple({
            where: { key: { in: [ch1.key, ch2.key] } },
          });
          const rename = Channel.useRename();
          return { retrieve, rename };
        },
        { wrapper },
      );
      await waitFor(() => expect(result.current.retrieve).not.toBeNull());

      const renamed1 = id.create();

      await act(async () => {
        await result.current.rename.updateAsync({
          key: ch1.key,
          name: renamed1,
        });
      });
      await waitFor(() => {
        const updated = result.current.retrieve?.find((c) => c.key === ch1.key);
        expect(updated?.name).toEqual(renamed1);
      });
      const unchanged = result.current.retrieve?.find((c) => c.key === ch2.key);
      expect(unchanged?.name).toEqual(orig2);
    });
  });

  describe("useDelete", () => {
    it("should delete a single channel", async () => {
      const ch = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      const { result } = renderHook(
        () => {
          const list = Channel.useList({ query: { where: { name: { eq: ch.name } } } });
          const del = Channel.useDelete();
          return { list, del };
        },
        { wrapper },
      );
      act(() => result.current.list.more());
      await waitFor(() => expect(result.current.list.data).toEqual([ch.key]));

      await act(async () => {
        await result.current.del.updateAsync(ch.key);
      });
      await waitFor(() => {
        expect(result.current.list.data).not.toContain(ch.key);
      });
    });

    it("should delete multiple channels", async () => {
      const ch1 = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      const ch2 = await client.channels.create({
        name: id.create(),
        dataType: DataType.INT32,
        virtual: true,
      });
      const name = { in: [ch1.name, ch2.name] };
      const { result } = renderHook(
        () => {
          const list = Channel.useList({ query: { where: { name } } });
          const del = Channel.useDelete();
          return { list, del };
        },
        { wrapper },
      );
      act(() => result.current.list.more());
      await waitFor(() => expect(result.current.list.data).toHaveLength(2));

      await act(async () => {
        await result.current.del.updateAsync([ch1.key, ch2.key]);
      });
      await waitFor(() => {
        expect(result.current.list.data).not.toContain(ch1.key);
        expect(result.current.list.data).not.toContain(ch2.key);
      });
    });

    it("should fail to delete a built-in channel", async () => {
      const [internal] = await client.channels.retrieve({ internal: true });
      const { result } = renderHook(() => Channel.useDelete(), { wrapper });
      await act(async () => {
        expect(await result.current.updateAsync(internal.key)).toBe(false);
      });
      expect(result.current.variant).toEqual("error");
      expect(result.current.status.description).toContain(
        `can't delete built-in channel "${internal.name}"`,
      );
      await expect(client.channels.retrieve(internal.key)).resolves.toBeDefined();
    });
  });

  describe("useCalculatedForm", () => {
    it("should create a new calculated channel", async () => {
      const sourceChannel = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT32,
        virtual: true,
      });

      const { result } = renderHook(() => Channel.useCalculatedForm({ query: null }), {
        wrapper,
      });

      const calculatedName = id.create();

      act(() => {
        result.current.form.set(path.name, calculatedName);
        result.current.form.set(path.dataType, DataType.FLOAT32.toString());
        result.current.form.set(path.virtual, true);
        result.current.form.set(
          calculatedPath.expression,
          `return ${sourceChannel.name} * 2`,
        );
        result.current.save({ signal: controller.signal });
      });

      await waitFor(() => {
        expect(result.current.form.value().name).toEqual(calculatedName);
        expect(result.current.form.value().expression).toEqual(
          `return ${sourceChannel.name} * 2`,
        );
        expect(result.current.form.value().key).toBeDefined();
        expect(result.current.form.value().key).toBeGreaterThan(0);
      });
    });

    it("should retrieve and edit existing calculated channel", async () => {
      const sourceChannel = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT32,
        virtual: true,
      });

      const existingName = id.create();
      const existingCalculated = await client.channels.retrieve(
        (
          await client.calculations.create({
            name: existingName,
            expression: `return ${sourceChannel.name} + 1`,
          })
        ).output,
      );

      const { result } = await renderHookSuspended(
        () => Channel.useCalculatedForm({ query: { key: existingCalculated.key } }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toEqual("success"));

      expect(result.current.form.value().name).toEqual(existingName);
      expect(result.current.form.value().expression).toEqual(
        `return ${sourceChannel.name} + 1`,
      );
      expect(result.current.form.value().operations).toEqual([
        { type: "none", duration: TimeSpan.ZERO },
      ]);

      act(() => {
        result.current.form.set(
          calculatedPath.expression,
          `return ${sourceChannel.name} * 3`,
        );
        result.current.save({ signal: controller.signal });
      });

      await waitFor(async () => {
        expect(result.current.form.value().expression).toEqual(
          `return ${sourceChannel.name} * 3`,
        );
        const [stored] = await client.calculations.retrieve({
          output: existingCalculated.key,
        });
        expect(stored.expression).toEqual(`return ${sourceChannel.name} * 3`);
      });
    });

    it("should save a rename and new operations of a calculated channel", async () => {
      const sourceChannel = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT32,
        virtual: true,
      });
      const resetChannel = await client.channels.create({
        name: id.create(),
        dataType: DataType.BOOLEAN,
        virtual: true,
      });
      const existing = await client.channels.retrieve(
        (
          await client.calculations.create({
            name: id.create(),
            expression: `return ${sourceChannel.name}`,
          })
        ).output,
      );
      const { result } = await renderHookSuspended(
        () => Channel.useCalculatedForm({ query: { key: existing.key } }),
        { wrapper },
      );
      await waitFor(() => expect(result.current.variant).toEqual("success"));
      const name = id.create();
      act(() => {
        result.current.form.set(path.name, name);
        result.current.form.set(calculatedPath.operations, [
          {
            type: "avg",
            reset: resetChannel.key,
            duration: TimeSpan.seconds(5),
          },
        ]);
      });
      await act(async () => {
        await result.current.saveAsync();
      });
      expect(result.current.variant).toEqual("success");
      await waitFor(async () => {
        expect((await client.channels.retrieve(existing.key)).name).toEqual(name);
        const [stored] = await client.calculations.retrieve({ output: existing.key });
        expect(stored.expression).toEqual(`return ${sourceChannel.name}`);
        expect(stored.operations).toEqual([
          { type: "avg", reset: resetChannel.key, duration: TimeSpan.seconds(5) },
        ]);
      });
    });

    it("should validate that expression is not empty", async () => {
      const { result } = renderHook(() => Channel.useCalculatedForm({ query: null }), {
        wrapper,
      });

      act(() => {
        result.current.form.set(path.name, id.create());
        result.current.form.set(calculatedPath.expression, "");
      });

      expect(result.current.form.validate()).toBe(false);
      expect(
        result.current.form.get(calculatedPath.expression).status.message,
      ).toContain("Expression must contain a return statement");
    });

    it("should validate that expression contains return statement", async () => {
      const { result } = renderHook(() => Channel.useCalculatedForm({ query: null }), {
        wrapper,
      });

      act(() => {
        result.current.form.set(path.name, id.create());
        result.current.form.set(calculatedPath.expression, "sourceChannel * 2");
      });

      expect(result.current.form.validate()).toBe(false);
      expect(
        result.current.form.get(calculatedPath.expression).status.message,
      ).toContain("Expression must contain a return statement");
    });

    it("should handle form with default values", async () => {
      const { result } = renderHook(() => Channel.useCalculatedForm({ query: null }), {
        wrapper,
      });

      expect(result.current.form.value().name).toEqual("");
      expect(result.current.form.value().expression).toEqual("");
      expect(result.current.form.value().dataType).toEqual(DataType.FLOAT32.toString());
      expect(result.current.form.value().virtual).toBe(false);
    });

    it("should update form when calculated channel is updated externally", async () => {
      const sourceChannel = await client.channels.create({
        name: id.create(),
        dataType: DataType.FLOAT32,
        virtual: true,
      });

      const testCalculated = await client.channels.retrieve(
        (
          await client.calculations.create({
            name: id.create(),
            expression: `return ${sourceChannel.name}`,
          })
        ).output,
      );

      const { result } = await renderHookSuspended(
        () => {
          const form = Channel.useCalculatedForm({
            query: { key: testCalculated.key },
          });
          const rename = Channel.useRename();
          return { form, rename };
        },
        { wrapper },
      );
      await waitFor(() => {
        expect(result.current.form.variant).toEqual("success");
      });
      expect(result.current.form.form.value().name).toEqual(testCalculated.name);

      const updatedName = id.create();
      await act(async () => {
        await result.current.rename.updateAsync({
          key: testCalculated.key,
          name: updatedName,
        });
      });

      await waitFor(() => {
        expect(result.current.form.form.value().name).toEqual(updatedName);
      });
    });
  });
});
