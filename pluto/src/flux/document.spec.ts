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
import { Status } from "@synnaxlabs/lyra/status";
import { id, record } from "@synnaxlabs/x";
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

const { useEnsure } = Flux.createRetrieve<{ key: task.Key }, task.Task>({
  name: "Task",
  retrieve: async ({ client, query: { key } }) => await client.tasks.retrieve(key),
  onChange: ({ client, query: { key } }, handler) =>
    client.tasks.onChange(key, handler),
  getCached: ({ client, query: { key } }) => client.tasks.getCached(key),
});

const useForm = Flux.createDocumentForm<task.Key, task.Task, typeof schema>({
  name: "Task",
  schema,
  docSchema: task.payloadZ(),
  domain: (client) => client.tasks,
  useEnsure,
  toValues: ({ payload: { name, config } }) => ({
    name,
    config: config as Values["config"],
  }),
});

// Remote edits come from this client. The form reads through its own client.
const client = createTestClient();
const formClient = createTestClient();

const fetchStored = async (key: task.Key): Promise<task.Task> =>
  await createTestClient().tasks.retrieve(key);

const countDispatches = (send: ReturnType<typeof spyOnSend>): number =>
  send.mock.calls.filter(([target]) => target === "/task/dispatch").length;

describe("createDocumentForm", () => {
  let testRack: rack.Rack;
  let wrapper: FC<PropsWithChildren>;

  beforeAll(async () => {
    testRack = await client.racks.create({ name: "document-form" });
  });

  beforeEach(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client: formClient });
    await waitForStreamLive(formClient.connection);
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  const createTask = async (routingKey = "dog"): Promise<task.Task> =>
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
        routingKey: "dog",
        alerts: [expect.objectContaining({ key: "a", status: "st-a" })],
      }),
    });
  });

  it("should show an edit at once and send it to the Core", async () => {
    const t = await createTask();
    const { result } = await renderForm(t.key);
    act(() => result.current.form.set("config.routingKey", "cat"));
    expect(result.current.form.get("config.routingKey").value).toBe("cat");
    await act(async () => expect(await result.current.flush()).toBe(true));
    expect((await fetchStored(t.key)).config).toMatchObject({ routingKey: "cat" });
  });

  it("should send the edits to one field as one dispatch", async () => {
    const t = await createTask();
    const { result } = await renderForm(t.key);
    const send = spyOnSend(formClient);
    act(() => {
      result.current.form.set("config.routingKey", "c");
      result.current.form.set("config.routingKey", "ca");
      result.current.form.set("config.routingKey", "cat");
    });
    await waitFor(() => expect(countDispatches(send)).toBe(1));
    await formClient.tasks.settled(t.key);
    expect((await fetchStored(t.key)).config).toMatchObject({ routingKey: "cat" });
  });

  it("should show a remote edit and keep the fields it does not reach", async () => {
    const t = await createTask();
    const { result } = await renderForm(t.key);
    const alerts = result.current.form.get("config.alerts").value;
    await act(async () => {
      await client.tasks.dispatch(
        t.key,
        task.set({ path: ["config", "routing_key"], value: "remote" }),
      );
    });
    await waitFor(() =>
      expect(result.current.form.get("config.routingKey").value).toBe("remote"),
    );
    expect(result.current.form.get("config.alerts").value).toBe(alerts);
  });

  it("should send nothing for a set to the value a field holds", async () => {
    const t = await createTask();
    const { result } = await renderForm(t.key);
    const send = spyOnSend(formClient);
    act(() => result.current.form.set("config.routingKey", "dog"));
    await act(async () => expect(await result.current.flush()).toBe(true));
    expect(countDispatches(send)).toBe(0);
    expect(formClient.tasks.hasUndo(t.key)).toBe(false);
  });

  it("should hold a value that fails its schema until the value passes", async () => {
    const t = await createTask();
    const { result } = await renderForm(t.key);
    const send = spyOnSend(formClient);
    act(() => result.current.form.set("config.routingKey", ""));
    await act(async () => expect(await result.current.flush()).toBe(false));
    expect(countDispatches(send)).toBe(0);
    expect(result.current.form.get("config.routingKey").value).toBe("");
    expect((await fetchStored(t.key)).config).toMatchObject({ routingKey: "dog" });
    act(() => result.current.form.set("config.routingKey", "cat"));
    await act(async () => expect(await result.current.flush()).toBe(true));
    expect(countDispatches(send)).toBe(1);
    expect((await fetchStored(t.key)).config).toMatchObject({ routingKey: "cat" });
  });

  it("should insert, move, and remove list entries by key", async () => {
    const t = await createTask();
    const { result } = await renderForm(t.key);
    act(() => {
      const { form } = result.current;
      form.insert("config.alerts", { key: "b", status: "st-b" }, "a");
      form.insert("config.alerts", { key: "c", status: "st-c" });
      form.move("config.alerts", "a", "b");
      form.remove("config.alerts.c");
    });
    const keys = (alerts: { key: string }[]): string[] => alerts.map(({ key }) => key);
    expect(keys(result.current.form.value().config.alerts)).toEqual(["b", "a"]);
    await act(async () => expect(await result.current.flush()).toBe(true));
    const stored = (await fetchStored(t.key)).config as Values["config"];
    expect(keys(stored.alerts)).toEqual(["b", "a"]);
  });

  it("should send a batch as one dispatch that one undo reverts", async () => {
    const t = await createTask();
    const { result } = await renderForm(t.key);
    const send = spyOnSend(formClient);
    act(() =>
      result.current.form.batch(() => {
        result.current.form.set("name", "renamed");
        result.current.form.set("config.routingKey", "cat");
      }),
    );
    await act(async () => expect(await result.current.flush()).toBe(true));
    expect(countDispatches(send)).toBe(1);
    const edited = await fetchStored(t.key);
    expect(edited.name).toBe("renamed");
    expect(edited.config).toMatchObject({ routingKey: "cat" });
    await act(async () => expect(await formClient.tasks.undo(t.key)).toBe(true));
    const reverted = await fetchStored(t.key);
    expect(reverted.name).toBe(t.name);
    expect(reverted.config).toMatchObject({ routingKey: "dog" });
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
    act(() => result.current.form.set("name", ""));
    await act(async () => expect(await result.current.flush()).toBe(false));
    await waitFor(() =>
      expect(result.current.statuses).toContainEqual(
        expect.objectContaining({ variant: "error", message: "Failed to save Task" }),
      ),
    );
    expect(result.current.form.get("name").value).toBe(t.name);
  });

  it("should report a refused send that was in flight when it flushes", async () => {
    const t = await createTask();
    const { result } = await renderForm(t.key);
    act(() => result.current.form.batch(() => result.current.form.set("name", "")));
    await act(async () => expect(await result.current.flush()).toBe(false));
    expect((await fetchStored(t.key)).name).toBe(t.name);
  });

  it("should flush a batch that changes nothing", async () => {
    const t = await createTask();
    const { result } = await renderForm(t.key);
    act(() =>
      result.current.form.batch(() =>
        result.current.form.set("config.routingKey", "dog"),
      ),
    );
    await act(async () => expect(await result.current.flush()).toBe(true));
  });

  it("should send the valid edits it holds when it unmounts", async () => {
    const t = await createTask();
    const { result, unmount } = await renderForm(t.key);
    act(() => result.current.form.set("config.routingKey", "cat"));
    unmount();
    await waitFor(async () =>
      expect((await fetchStored(t.key)).config).toMatchObject({ routingKey: "cat" }),
    );
  });

  it("should edit the new document when the key changes", async () => {
    const first = await createTask();
    const second = await createTask("bird");
    await formClient.tasks.retrieve(second.key);
    const { result, rerender } = await renderForm(first.key);
    rerender({ key: second.key });
    expect(result.current.form.value().config.routingKey).toBe("bird");
    act(() => result.current.form.set("config.routingKey", "owl"));
    await act(async () => expect(await result.current.flush()).toBe(true));
    expect((await fetchStored(second.key)).config).toMatchObject({ routingKey: "owl" });
    expect((await fetchStored(first.key)).config).toMatchObject({ routingKey: "dog" });
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
      docSchema: record.unknownZ(),
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
      expect(result.current.form.value()).toMatchObject({ routingKey: "dog" });
    });

    it("should send an edit to a field below the path", async () => {
      const t = await createTask();
      const { result } = await renderConfigForm(t.key);
      const send = spyOnSend(formClient);
      act(() => result.current.form.set("routingKey", "cat"));
      await act(async () => expect(await result.current.flush()).toBe(true));
      const [, req] =
        send.mock.calls.find(([target]) => target === "/task/dispatch") ?? [];
      expect(req).toMatchObject({
        actions: [{ type: "set", set: { path: ["config", "routing_key"] } }],
      });
      expect((await fetchStored(t.key)).config).toMatchObject({ routingKey: "cat" });
    });

    it("should keep its values and take no edits once another client removes the part", async () => {
      const useAlertForm = Flux.createDocumentForm<task.Key, task.Task, typeof alertZ>({
        name: "Task",
        schema: alertZ,
        docSchema: record.unknownZ(),
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
        const cached = formClient.tasks.getCached(t.key) as task.Task;
        expect((cached.config as Values["config"]).alerts).toEqual([]);
      });
      expect(result.current.form.value()).toMatchObject({ key: "a", status: "st-a" });
      const send = spyOnSend(formClient);
      act(() => result.current.form.set("status", "st-b"));
      await act(async () => expect(await result.current.flush()).toBe(true));
      expect(countDispatches(send)).toBe(0);
      expect((await fetchStored(t.key)).config).toMatchObject({ alerts: [] });
    });

    it("should insert a list entry below the path", async () => {
      const t = await createTask();
      const { result } = await renderConfigForm(t.key);
      act(() =>
        result.current.form.insert("alerts", { key: "b", status: "st-b" }, "a"),
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
