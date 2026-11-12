// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack, task } from "@synnaxlabs/client";
import {
  createTestClient,
  spyOnSend,
  waitForStreamLive,
} from "@synnaxlabs/client/testutil";
import { Form } from "@synnaxlabs/lyra/form";
import { Status } from "@synnaxlabs/lyra/status";
import { deep, id, uuid } from "@synnaxlabs/x";
import { act, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren } from "react";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Flux } from "@/flux";
import { renderHookSuspended } from "@/testutil/render";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const alertZ = z.object({ key: z.string(), status: z.string() });

const schema = z.object({
  name: z.string(),
  config: z.object({
    routingKey: z.string().min(1, "Routing key is required"),
    alerts: alertZ.array(),
  }),
});

type Values = z.infer<typeof schema>;

const path = deep.path<Values>();
const config = deep.path<Values["config"]>();
const alert = deep.path<Values["config"]["alerts"][number]>();

const { useEnsure } = Flux.createRetrieve<{ key: task.Key }, task.Task>({
  name: "Task",
  retrieve: async ({ client, query: { key } }) => await client.tasks.retrieve(key),
  onChange: ({ client, query: { key } }, handler) =>
    client.tasks.watch(key).onChange(handler),
  getCached: ({ client, query: { key } }) => client.tasks.watch(key).getCached(),
});

const createForm = <S extends z.ZodType<Values>>(formSchema: S) =>
  Flux.createDocumentForm<task.Key, task.Task, S>({
    name: "Task",
    schema: formSchema,
    domain: (client) => client.tasks,
    useEnsure,
    toValues: ({ payload: { name, config } }) => ({ name, config }) as z.infer<S>,
  });

const useForm = createForm(schema);

const useEntryForm = createForm(
  z.object({
    name: z.string(),
    config: z.object({
      routingKey: z.string(),
      alerts: z.object({ key: z.string(), status: z.string().min(1) }).array(),
    }),
  }),
);

// Remote edits come from this client. The form reads through its own client.
const client = createTestClient();
const formClient = createTestClient();

// A routing key is the key of a secret, which must exist when a config names it.
const createRoutingKey = async (): Promise<string> =>
  (await client.secrets.set({ name: "Routing key", source: "stored", value: "rk" }))
    .key;
const ROUTING_KEY = await createRoutingKey();
const OTHER_ROUTING_KEY = await createRoutingKey();

const fetchStored = async (key: task.Key): Promise<task.Task> =>
  await createTestClient().tasks.retrieve(key);

const countDispatches = (send: ReturnType<typeof spyOnSend>): number =>
  send.mock.calls.filter(([target]) => target === "/task/dispatch").length;

const dispatchedActions = (send: ReturnType<typeof spyOnSend>): unknown[] =>
  send.mock.calls
    .filter(([target]) => target === "/task/dispatch")
    .map(([, req]) => (req as { actions: unknown[] }).actions);

const setRoutingKey = (value: string) => ({
  type: "set",
  set: expect.objectContaining({ path: ["config", "routing_key"], value }),
});

describe("createDocumentForm", () => {
  let testRack: rack.Rack;
  let wrapper: FC<PropsWithChildren>;

  beforeAll(async () => {
    testRack = await client.racks.create({ name: "document-form" });
    // The Core refuses an alert config that names a status it does not hold.
    await client.statuses.set(
      ["st-a", "st-b", "st-c", "st-0", "st-1"].map((key) => ({
        key,
        name: key,
        variant: "info" as const,
        message: key,
      })),
    );
  });

  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client: formClient });
    await waitForStreamLive(formClient.connection);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  const createTask = async (routingKey = ROUTING_KEY): Promise<task.Task> =>
    await testRack.createTask({
      name: `document-${id.create()}`,
      type: "pagerduty_alert",
      config: { routingKey, alerts: [{ key: "a", status: "st-a" }] },
    });

  const renderForm = async (key: task.Key) =>
    await renderHookSuspended(
      ({ key }: { key: task.Key }) => useForm({ query: { key } }),
      { wrapper, initialProps: { key } },
    );

  it("should show the cached document", async () => {
    const t = await createTask();
    const { result } = await renderForm(t.key);
    expect(result.current.form.value()).toEqual({
      name: t.name,
      config: expect.objectContaining({
        routingKey: ROUTING_KEY,
        alerts: [expect.objectContaining({ key: "a", status: "st-a" })],
      }),
    });
  });

  it("should show an edit at once and send it to the Core", async () => {
    const t = await createTask();
    const { result } = await renderForm(t.key);
    act(() => result.current.form.set(path.config.routingKey, OTHER_ROUTING_KEY));
    expect(result.current.form.get(path.config.routingKey).value).toBe(
      OTHER_ROUTING_KEY,
    );
    await act(async () => expect(await result.current.flush()).toBe(true));
    expect((await fetchStored(t.key)).config).toMatchObject({
      routingKey: OTHER_ROUTING_KEY,
    });
  });

  it("should send the edits to one field as one dispatch", async () => {
    const t = await createTask();
    const { result } = await renderForm(t.key);
    const send = spyOnSend(formClient);
    act(() => {
      result.current.form.set(path.config.routingKey, uuid.create());
      result.current.form.set(path.config.routingKey, uuid.create());
      result.current.form.set(path.config.routingKey, OTHER_ROUTING_KEY);
    });
    await waitFor(() => expect(countDispatches(send)).toBe(1));
    expect(dispatchedActions(send)).toEqual([[setRoutingKey(OTHER_ROUTING_KEY)]]);
    await formClient.tasks.settled(t.key);
    expect((await fetchStored(t.key)).config).toMatchObject({
      routingKey: OTHER_ROUTING_KEY,
    });
  });

  it("should show a remote edit and keep the fields it does not reach", async () => {
    const t = await createTask();
    const { result } = await renderForm(t.key);
    const alerts = result.current.form.get(path.config.alerts).value;
    await act(async () => {
      await client.tasks.dispatch(
        t.key,
        task.set({ path: ["config", "routingKey"], value: OTHER_ROUTING_KEY }),
      );
    });
    await waitFor(() =>
      expect(result.current.form.get(path.config.routingKey).value).toBe(
        OTHER_ROUTING_KEY,
      ),
    );
    expect(result.current.form.get(path.config.alerts).value).toBe(alerts);
  });

  it("should send nothing for a set to the value a field holds", async () => {
    const t = await createTask();
    const { result } = await renderForm(t.key);
    const send = spyOnSend(formClient);
    act(() => result.current.form.set(path.config.routingKey, ROUTING_KEY));
    await act(async () => expect(await result.current.flush()).toBe(true));
    expect(countDispatches(send)).toBe(0);
  });

  it("should send only the fields that a write of the whole form changes", async () => {
    const t = await createTask();
    const { result } = await renderForm(t.key);
    const send = spyOnSend(formClient);
    const values = result.current.form.value();
    act(() =>
      result.current.form.set(path, {
        ...values,
        config: { ...values.config, routingKey: OTHER_ROUTING_KEY },
      }),
    );
    expect(result.current.form.get(path.config.routingKey).value).toBe(
      OTHER_ROUTING_KEY,
    );
    await act(async () => expect(await result.current.flush()).toBe(true));
    const [, req] =
      send.mock.calls.find(([target]) => target === "/task/dispatch") ?? [];
    expect(req).toMatchObject({
      actions: [{ type: "set", set: { path: ["config", "routing_key"] } }],
    });
    expect((await fetchStored(t.key)).config).toMatchObject({
      routingKey: OTHER_ROUTING_KEY,
    });
  });

  it("should send writes of the whole form to one field as one dispatch", async () => {
    const t = await createTask();
    const { result } = await renderForm(t.key);
    const send = spyOnSend(formClient);
    const write = (routingKey: string) => {
      const values = result.current.form.value();
      const config = { ...values.config, routingKey };
      act(() => result.current.form.set(path, { ...values, config }));
    };
    write(uuid.create());
    write(uuid.create());
    write(OTHER_ROUTING_KEY);
    await act(async () => expect(await result.current.flush()).toBe(true));
    expect(countDispatches(send)).toBe(1);
    expect((await fetchStored(t.key)).config).toMatchObject({
      routingKey: OTHER_ROUTING_KEY,
    });
  });

  it("should hold a write of the whole form that fails a field's schema", async () => {
    const t = await createTask();
    const { result } = await renderForm(t.key);
    const values = result.current.form.value();
    const config = { ...values.config, routingKey: "" };
    act(() => result.current.form.set(path, { ...values, config }));
    await act(async () => expect(await result.current.flush()).toBe(false));
    expect((await fetchStored(t.key)).config).toMatchObject({
      routingKey: ROUTING_KEY,
    });
  });

  it("should send nothing for a write of the whole form that changes nothing", async () => {
    const t = await createTask();
    const { result } = await renderForm(t.key);
    const send = spyOnSend(formClient);
    act(() => result.current.form.set(path, { ...result.current.form.value() }));
    await act(async () => expect(await result.current.flush()).toBe(true));
    expect(countDispatches(send)).toBe(0);
  });

  it("should hold a value that fails its schema until the value passes", async () => {
    const t = await createTask();
    const { result } = await renderForm(t.key);
    const send = spyOnSend(formClient);
    act(() => result.current.form.set(path.config.routingKey, ""));
    await act(async () => expect(await result.current.flush()).toBe(false));
    expect(countDispatches(send)).toBe(0);
    expect(result.current.form.get(path.config.routingKey).value).toBe("");
    expect((await fetchStored(t.key)).config).toMatchObject({
      routingKey: ROUTING_KEY,
    });
    act(() => result.current.form.set(path.config.routingKey, OTHER_ROUTING_KEY));
    await act(async () => expect(await result.current.flush()).toBe(true));
    expect(dispatchedActions(send)).toEqual([[setRoutingKey(OTHER_ROUTING_KEY)]]);
    expect((await fetchStored(t.key)).config).toMatchObject({
      routingKey: OTHER_ROUTING_KEY,
    });
  });

  it("should send a batch at once when a value fails its schema", async () => {
    const t = await createTask();
    const { result } = await renderHookSuspended(
      () => useEntryForm({ query: { key: t.key } }),
      { wrapper },
    );
    const send = spyOnSend(formClient);
    act(() =>
      result.current.form.batch(() =>
        result.current.form.set(path.config.alerts.a.status, ""),
      ),
    );
    await act(async () => expect(await result.current.flush()).toBe(true));
    expect(countDispatches(send)).toBe(1);
    expect((await fetchStored(t.key)).config).toMatchObject({
      alerts: [{ key: "a", status: "" }],
    });
  });

  it("should drop a held value when a batch replaces it", async () => {
    const t = await createTask();
    const { result } = await renderForm(t.key);
    act(() => result.current.form.set(path.config.routingKey, ""));
    act(() =>
      result.current.form.batch(() =>
        result.current.form.set(path.config.routingKey, OTHER_ROUTING_KEY),
      ),
    );
    await act(async () => expect(await result.current.flush()).toBe(true));
    await waitFor(async () =>
      expect((await fetchStored(t.key)).config).toMatchObject({
        routingKey: OTHER_ROUTING_KEY,
      }),
    );
    await waitFor(() =>
      expect(result.current.form.get(path.config.routingKey).value).toBe(
        OTHER_ROUTING_KEY,
      ),
    );
  });

  describe("a held value inside a list entry", () => {
    const renderEntryForm = async () => {
      const t = await testRack.createTask({
        name: `document-${id.create()}`,
        type: "pagerduty_alert",
        config: {
          routingKey: ROUTING_KEY,
          alerts: [
            { key: "a", status: "st-a" },
            { key: "b", status: "st-b" },
          ],
        },
      });
      const { result } = await renderHookSuspended(
        ({ key }: { key: task.Key }) => useEntryForm({ query: { key } }),
        { wrapper, initialProps: { key: t.key } },
      );
      act(() => result.current.form.set(path.config.alerts.a.status, ""));
      await act(async () => expect(await result.current.flush()).toBe(false));
      return { t, result };
    };

    it("should stop blocking a flush when the form removes the entry", async () => {
      const { t, result } = await renderEntryForm();
      act(() => result.current.form.remove(path.config.alerts.a));
      await act(async () => expect(await result.current.flush()).toBe(true));
      expect((await fetchStored(t.key)).config).toMatchObject({
        alerts: [{ key: "b", status: "st-b" }],
      });
    });

    it("should edit an entry with a dot in its key", async () => {
      const t = await testRack.createTask({
        name: `document-${id.create()}`,
        type: "pagerduty_alert",
        config: {
          routingKey: ROUTING_KEY,
          alerts: [
            { key: "ns=2;s=a.b", status: "st-a" },
            { key: "ns=2;s=a", status: "st-b" },
          ],
        },
      });
      const { result } = await renderHookSuspended(
        ({ key }: { key: task.Key }) => useEntryForm({ query: { key } }),
        { wrapper, initialProps: { key: t.key } },
      );
      act(() =>
        result.current.form.set(path.config.alerts["ns=2;s=a.b"].status, "st-c"),
      );
      await act(async () => expect(await result.current.flush()).toBe(true));
      expect((await fetchStored(t.key)).config).toMatchObject({
        alerts: [
          { key: "ns=2;s=a.b", status: "st-c" },
          { key: "ns=2;s=a", status: "st-b" },
        ],
      });
    });

    it("should remove the entry with a key that reads as an index", async () => {
      const t = await testRack.createTask({
        name: `document-${id.create()}`,
        type: "pagerduty_alert",
        config: {
          routingKey: ROUTING_KEY,
          alerts: [
            { key: "1", status: "st-1" },
            { key: "0", status: "st-0" },
          ],
        },
      });
      const { result } = await renderHookSuspended(
        ({ key }: { key: task.Key }) => useEntryForm({ query: { key } }),
        { wrapper, initialProps: { key: t.key } },
      );
      act(() => {
        Form.fieldListUtils(result.current.form, path.config.alerts).remove("1");
      });
      await act(async () => expect(await result.current.flush()).toBe(true));
      expect((await fetchStored(t.key)).config).toMatchObject({
        alerts: [{ key: "0", status: "st-0" }],
      });
    });

    it("should stop blocking a flush when another client removes the entry", async () => {
      const { t, result } = await renderEntryForm();
      await act(async () => {
        await client.tasks.dispatch(
          t.key,
          task.remove({ path: ["config", "alerts"], key: "a" }),
        );
      });
      await waitFor(() =>
        expect(result.current.form.value().config.alerts).toHaveLength(1),
      );
      await act(async () => expect(await result.current.flush()).toBe(true));
    });

    it("should keep blocking a flush when another client edits another entry", async () => {
      const { t, result } = await renderEntryForm();
      await act(async () => {
        await client.tasks.dispatch(
          t.key,
          task.set({ path: ["config", "alerts", "b", "status"], value: "st-c" }),
        );
      });
      await waitFor(() =>
        expect(result.current.form.get(path.config.alerts.b.status).value).toBe("st-c"),
      );
      await act(async () => expect(await result.current.flush()).toBe(false));
      expect(result.current.form.get(path.config.alerts.a.status).value).toBe("");
    });
  });

  it("should insert, move, and remove list entries by key", async () => {
    const t = await createTask();
    const { result } = await renderForm(t.key);
    act(() => {
      const { form } = result.current;
      form.insert(path.config.alerts, { key: "b", status: "st-b" }, "a");
      form.insert(path.config.alerts, { key: "c", status: "st-c" });
      form.move(path.config.alerts, "a", "b");
      form.remove(path.config.alerts.c);
    });
    const keys = (alerts: { key: string }[]): string[] => alerts.map(({ key }) => key);
    expect(keys(result.current.form.value().config.alerts)).toEqual(["b", "a"]);
    await act(async () => expect(await result.current.flush()).toBe(true));
    const stored = (await fetchStored(t.key)).config as Values["config"];
    expect(keys(stored.alerts)).toEqual(["b", "a"]);
  });

  it("should send nothing for edits through an entry that another client removed", async () => {
    const t = await testRack.createTask({
      name: `document-${id.create()}`,
      type: "pagerduty_alert",
      config: {
        routingKey: ROUTING_KEY,
        alerts: [
          { key: "a", status: "st-a" },
          { key: "b", status: "st-b" },
        ],
      },
    });
    const { result } = await renderForm(t.key);
    await act(async () => {
      await client.tasks.dispatch(
        t.key,
        task.remove({ path: ["config", "alerts"], key: "a" }),
      );
    });
    await waitFor(() =>
      expect(result.current.form.value().config.alerts).toHaveLength(1),
    );
    const send = spyOnSend(formClient);
    act(() => {
      result.current.form.set(path.config.alerts.a.status, "st-c");
      result.current.form.remove(path.config.alerts.a);
    });
    await act(async () => expect(await result.current.flush()).toBe(true));
    expect(countDispatches(send)).toBe(0);
    expect((await fetchStored(t.key)).config).toMatchObject({
      alerts: [{ key: "b", status: "st-b" }],
    });
  });

  // Task channels are often a union of channel types.
  const alertUnionZ = z.discriminatedUnion("status", [
    alertZ.extend({ status: z.literal("st-a") }),
    alertZ.extend({ status: z.literal("st-b") }),
  ]);

  it.each([
    ["objects", alertZ],
    ["a union", alertUnionZ],
  ])(
    "should send nothing for edits through an entry after another client empties a list of %s",
    async (_, entryZ) => {
      const useEmptiedForm = createForm(
        schema.extend({
          config: schema.shape.config.extend({ alerts: entryZ.array() }),
        }),
      );
      const t = await createTask();
      const { result } = await renderHookSuspended(
        () => useEmptiedForm({ query: { key: t.key } }),
        { wrapper },
      );
      await act(async () => {
        await client.tasks.dispatch(
          t.key,
          task.remove({ path: ["config", "alerts"], key: "a" }),
        );
      });
      await waitFor(() =>
        expect(result.current.form.value().config.alerts).toEqual([]),
      );
      const send = spyOnSend(formClient);
      act(() => {
        result.current.form.set(path.config.alerts.a.status, "st-b");
        result.current.form.set(path.config.alerts[0].status, "st-b");
        result.current.form.remove(path.config.alerts.a);
      });
      await act(async () => expect(await result.current.flush()).toBe(true));
      expect(countDispatches(send)).toBe(0);
      expect((await fetchStored(t.key)).config).toMatchObject({ alerts: [] });
    },
  );

  it("should guard a write inside a union on the value that picks its option", async () => {
    // A Modbus write channel is a union on its type.
    const unionSchema = z.object({
      name: z.string(),
      config: z.object({
        channels: z
          .discriminatedUnion("type", [
            z.object({ type: z.literal("coil"), key: z.string(), address: z.number() }),
            z.object({
              type: z.literal("holding_register"),
              key: z.string(),
              address: z.number(),
            }),
          ])
          .array(),
      }),
    });
    type UnionValues = z.infer<typeof unionSchema>;
    const unionPath = deep.path<UnionValues>();
    const useUnionForm = Flux.createDocumentForm<
      task.Key,
      task.Task,
      typeof unionSchema
    >({
      name: "Task",
      schema: unionSchema,
      domain: (client) => client.tasks,
      useEnsure,
      toValues: ({ payload: { name, config } }) => ({ name, config }) as UnionValues,
    });
    const t = await testRack.createTask({
      name: `document-${id.create()}`,
      type: "modbus_write",
      config: { channels: [{ type: "coil", key: "a", address: 1 }] },
    });
    const { result } = await renderHookSuspended(
      () => useUnionForm({ query: { key: t.key } }),
      { wrapper },
    );
    const send = spyOnSend(formClient);
    act(() => result.current.form.set(unionPath.config.channels.a.address, 2));
    await act(async () => expect(await result.current.flush()).toBe(true));
    const [, req] =
      send.mock.calls.find(([target]) => target === "/task/dispatch") ?? [];
    expect(req).toMatchObject({
      actions: [
        {
          type: "set",
          set: {
            path: ["config", "channels", "a", "address"],
            guards: [{ path: ["config", "channels", "a", "type"], value: "coil" }],
          },
        },
      ],
    });
    expect((await fetchStored(t.key)).config).toMatchObject({
      channels: [{ key: "a", address: 2 }],
    });
  });

  it("should send a batch as one dispatch", async () => {
    const t = await createTask();
    const { result } = await renderForm(t.key);
    const send = spyOnSend(formClient);
    act(() =>
      result.current.form.batch(() => {
        result.current.form.set(path.name, "renamed");
        result.current.form.set(path.config.routingKey, OTHER_ROUTING_KEY);
      }),
    );
    await act(async () => expect(await result.current.flush()).toBe(true));
    expect(countDispatches(send)).toBe(1);
    const edited = await fetchStored(t.key);
    expect(edited.name).toBe("renamed");
    expect(edited.config).toMatchObject({ routingKey: OTHER_ROUTING_KEY });
  });

  it("should report an edit that the Core refuses and drop it", async () => {
    const t = await createTask();
    const { result } = await renderHookSuspended(
      () => ({
        ...useForm({ query: { key: t.key } }),
        statuses: Status.useNotifications().statuses,
      }),
      { wrapper },
    );
    act(() => result.current.form.set(path.name, ""));
    await act(async () => expect(await result.current.flush()).toBe(false));
    await waitFor(() =>
      expect(result.current.statuses).toContainEqual(
        expect.objectContaining({ variant: "error", message: "Failed to save Task" }),
      ),
    );
    expect(result.current.form.get(path.name).value).toBe(t.name);
  });

  it("should report a refused send that was in flight when it flushes", async () => {
    const t = await createTask();
    const { result } = await renderForm(t.key);
    act(() => result.current.form.batch(() => result.current.form.set(path.name, "")));
    await act(async () => expect(await result.current.flush()).toBe(false));
    expect((await fetchStored(t.key)).name).toBe(t.name);
  });

  it("should not fail a flush for an edit made while it waits", async () => {
    const t = await createTask();
    const { result } = await renderForm(t.key);
    act(() => result.current.form.set(path.config.routingKey, OTHER_ROUTING_KEY));
    let flushed = Promise.resolve(false);
    act(() => {
      flushed = result.current.flush();
      result.current.form.set(path.name, "renamed");
    });
    await act(async () => expect(await flushed).toBe(true));
  });

  it("should flush a batch that changes nothing", async () => {
    const t = await createTask();
    const { result } = await renderForm(t.key);
    act(() =>
      result.current.form.batch(() =>
        result.current.form.set(path.config.routingKey, ROUTING_KEY),
      ),
    );
    await act(async () => expect(await result.current.flush()).toBe(true));
  });

  it("should send the valid edits it holds when it unmounts", async () => {
    const t = await createTask();
    const { result, unmount } = await renderForm(t.key);
    act(() => result.current.form.set(path.config.routingKey, OTHER_ROUTING_KEY));
    unmount();
    await waitFor(async () =>
      expect((await fetchStored(t.key)).config).toMatchObject({
        routingKey: OTHER_ROUTING_KEY,
      }),
    );
  });

  it("should drop an edit that fails its schema when it unmounts", async () => {
    const t = await createTask();
    const { result, unmount } = await renderForm(t.key);
    act(() => result.current.form.set(path.config.routingKey, ""));
    unmount();
    await formClient.tasks.settled(t.key);
    const cached = formClient.tasks.watch(t.key).getCached() as task.Task;
    expect(cached.config).toMatchObject({ routingKey: ROUTING_KEY });
    expect((await fetchStored(t.key)).config).toMatchObject({
      routingKey: ROUTING_KEY,
    });
  });

  it("should edit the new document when the key changes", async () => {
    const first = await createTask();
    const secondKey = await createRoutingKey();
    const editedKey = await createRoutingKey();
    const second = await createTask(secondKey);
    await formClient.tasks.retrieve(second.key);
    const { result, rerender } = await renderForm(first.key);
    rerender({ key: second.key });
    expect(result.current.form.value().config.routingKey).toBe(secondKey);
    act(() => result.current.form.set(path.config.routingKey, editedKey));
    await act(async () => expect(await result.current.flush()).toBe(true));
    expect((await fetchStored(second.key)).config).toMatchObject({
      routingKey: editedKey,
    });
    expect((await fetchStored(first.key)).config).toMatchObject({
      routingKey: ROUTING_KEY,
    });
  });

  describe("pathOf", () => {
    const configSchema = schema.shape.config;
    const useConfigForm = Flux.createDocumentForm<
      task.Key,
      task.Task,
      typeof configSchema
    >({
      name: "Task",
      schema: configSchema,
      domain: (client) => client.tasks,
      useEnsure,
      toValues: ({ payload: { config } }) => config as Values["config"],
      pathOf: () => ["config"],
    });

    const renderConfigForm = async (key: task.Key) =>
      await renderHookSuspended(() => useConfigForm({ query: { key } }), { wrapper });

    it("should show the part of the document at the path", async () => {
      const t = await createTask();
      const { result } = await renderConfigForm(t.key);
      expect(result.current.form.value()).toMatchObject({ routingKey: ROUTING_KEY });
    });

    it("should send an edit to a field below the path", async () => {
      const t = await createTask();
      const { result } = await renderConfigForm(t.key);
      const send = spyOnSend(formClient);
      act(() => result.current.form.set(config.routingKey, OTHER_ROUTING_KEY));
      await act(async () => expect(await result.current.flush()).toBe(true));
      const [, req] =
        send.mock.calls.find(([target]) => target === "/task/dispatch") ?? [];
      expect(req).toMatchObject({
        actions: [{ type: "set", set: { path: ["config", "routing_key"] } }],
      });
      expect((await fetchStored(t.key)).config).toMatchObject({
        routingKey: OTHER_ROUTING_KEY,
      });
    });

    it("should keep its values and take no edits after a remote removal", async () => {
      const useAlertForm = Flux.createDocumentForm<task.Key, task.Task, typeof alertZ>({
        name: "Task",
        schema: alertZ,
        domain: (client) => client.tasks,
        useEnsure,
        toValues: ({ payload: { config } }) =>
          (config as Values["config"]).alerts.find(({ key }) => key === "a"),
        pathOf: () => ["config", "alerts", "a"],
      });
      const t = await createTask();
      const { result } = await renderHookSuspended(
        () => useAlertForm({ query: { key: t.key } }),
        { wrapper },
      );
      await act(async () => {
        await client.tasks.dispatch(
          t.key,
          task.remove({ path: ["config", "alerts"], key: "a" }),
        );
      });
      await waitFor(() => {
        const cached = formClient.tasks.watch(t.key).getCached() as task.Task;
        expect((cached.config as Values["config"]).alerts).toEqual([]);
      });
      expect(result.current.form.value()).toMatchObject({ key: "a", status: "st-a" });
      const send = spyOnSend(formClient);
      act(() => result.current.form.set(alert.status, "st-b"));
      await act(async () => expect(await result.current.flush()).toBe(true));
      expect(countDispatches(send)).toBe(0);
      expect((await fetchStored(t.key)).config).toMatchObject({ alerts: [] });
    });

    it("should insert a list entry below the path", async () => {
      const t = await createTask();
      const { result } = await renderConfigForm(t.key);
      act(() =>
        result.current.form.insert(config.alerts, { key: "b", status: "st-b" }, "a"),
      );
      await act(async () => expect(await result.current.flush()).toBe(true));
      expect((await fetchStored(t.key)).config).toMatchObject({
        alerts: [
          { key: "a", status: "st-a" },
          { key: "b", status: "st-b" },
        ],
      });
    });
  });
});
