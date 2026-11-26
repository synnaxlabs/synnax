// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, icd, type Synnax, type task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { z } from "zod";

import { HTTP } from "@/feature/http";
import { createHTTPDevice } from "@/feature/http/testutil";
import { pickMessageFormat } from "@/platform/icd/testutil";
import {
  deployAndAwaitTask,
  preloadForms,
  renderTaskFormTab,
} from "@/platform/task/testutil";

const client = createTestClient();

preloadForms(HTTP.Task.FORMS);

type WriteConfig = z.infer<HTTP.Task.WriteSchemas["config"]>;

const ZERO_DRAFT: task.New<HTTP.Task.WriteSchemas> = {
  name: uniqueName("http_write"),
  type: HTTP.Task.WRITE_TYPE,
  config: HTTP.Task.WRITE_SCHEMAS.config.parse({}),
};

const createDraft = async (client: Synnax, config: WriteConfig) =>
  await client.tasks.create({ ...ZERO_DRAFT, config }, HTTP.Task.WRITE_SCHEMAS);

const retrieveEndpoints = async (key: task.Key) =>
  (await client.tasks.retrieve({ key, schemas: HTTP.Task.WRITE_SCHEMAS })).config
    .endpoints;

const createField = (name: string, type: icd.JSONField["type"]): icd.JSONField =>
  icd.jsonFieldZ.parse({ name, pointer: `/${name}`, type });

/** Creates an ICD that holds one JSON message of the fields, with unique names. */
const createFormat = async (fields: icd.JSONField[]) => {
  const message = icd.messageZ.parse({
    name: uniqueName("setpoint"),
    payload: { format: "json", fields },
  });
  const ic = await client.icds.create({
    name: uniqueName("valve"),
    messages: [message],
  });
  return { ic, message };
};

/** @returns an endpoint that sends message of ic, with each field on a new channel. */
const createEndpoint = (
  key: string,
  path: string,
  ic: icd.ICD,
  message: icd.Message,
  fields: icd.JSONField[],
): HTTP.Task.WriteEndpoint => ({
  ...HTTP.Task.writeEndpointZ.parse({}),
  key,
  method: "PUT",
  path,
  icd: ic.key,
  message: {
    message: message.key,
    bindings: fields.map((f) => ({ type: "channel", field: f.key, channel: 0 })),
  },
});

const renderWrite = async (endpoints: HTTP.Task.WriteEndpoint[] = []) => {
  const dev = await createHTTPDevice(client);
  const draft = await createDraft(client, {
    ...HTTP.Task.WRITE_SCHEMAS.config.parse({}),
    device: dev.key,
    endpoints,
  });
  const result = await renderTaskFormTab(HTTP.Task.FORMS[HTTP.Task.WRITE_TYPE], {
    client,
    taskKey: draft.key,
  });
  return { ...result, dev, draft };
};

const addEndpoint = async (): Promise<void> => {
  fireEvent.click(await screen.findByRole("button", { name: "Add endpoint" }));
  await screen.findByText("Request");
};

/** Selects a field in the tree, and returns the section that fills it. */
const selectField = async (name: string) => {
  fireEvent.click(
    await screen.findByRole("treeitem", { name: new RegExp(`^${name}`) }),
  );
  return within(await screen.findByRole("group", { name: "Source" }));
};

describe("Write", () => {
  // This first test pays the one-time load of the lazy form, which takes over 10 s on
  // a loaded CI runner.
  it("should add an endpoint that sends a new message, in an ICD named after the task", async () => {
    const { draft } = await renderWrite();
    await screen.findByText("Add an endpoint to send requests to");
    await addEndpoint();
    let endpoints: HTTP.Task.WriteEndpoint[] = [];
    await waitFor(async () => {
      endpoints = await retrieveEndpoints(draft.key);
      expect(endpoints[0]?.icd).toBeDefined();
    });
    const ic = await client.icds.retrieve(endpoints[0].icd as icd.Key);
    expect(ic.name).toBe(ZERO_DRAFT.name);
    expect(ic.messages.map(({ name }) => name)).toEqual(["POST"]);
    expect(endpoints[0].message).toEqual({ message: ic.messages[0].key, bindings: [] });
  }, 30_000);

  it("should add the message of a second endpoint to the ICD of the first", async () => {
    const { draft } = await renderWrite();
    await addEndpoint();
    await addEndpoint();
    let endpoints: HTTP.Task.WriteEndpoint[] = [];
    await waitFor(async () => {
      endpoints = await retrieveEndpoints(draft.key);
      expect(endpoints.map((ep) => ep.icd != null)).toEqual([true, true]);
    });
    expect(endpoints[1].icd).toBe(endpoints[0].icd);
    await waitFor(async () => {
      const ic = await client.icds.retrieve(endpoints[0].icd as icd.Key);
      expect(ic.messages.map(({ name }) => name)).toEqual(["POST", "POST 2"]);
    });
  });

  it("should rename the message of an endpoint after its new method and path", async () => {
    const { draft } = await renderWrite();
    await addEndpoint();
    await addEndpoint();
    const [first] = await screen.findAllByRole("treeitem");
    fireEvent.click(first);
    await waitFor(() =>
      expect(screen.getByRole("treeitem", { selected: true })).toBe(first),
    );
    fireEvent.click(screen.getByText("PATCH"));
    fireEvent.change(screen.getByPlaceholderText("/api/control"), {
      target: { value: "/speed" },
    });
    await waitFor(async () => {
      const [ep] = await retrieveEndpoints(draft.key);
      if (ep?.icd == null) throw new Error("the endpoint sends no ICD");
      const ic = await client.icds.retrieve(ep.icd);
      expect(ic.messages.map(({ name }) => name)).toEqual(["PATCH /speed", "POST 2"]);
    });
  });

  it("should keep the name of a message that the form did not create, though its request names it", async () => {
    const value = createField("value", "float");
    const message = icd.messageZ.parse({
      name: "PUT /setpoint",
      payload: { format: "json", fields: [value] },
    });
    const ic = await client.icds.create({
      name: uniqueName("valve"),
      messages: [message],
    });
    await renderWrite([createEndpoint("ep1", "/setpoint", ic, message, [value])]);
    fireEvent.click(await screen.findByRole("treeitem", { name: /\/setpoint/ }));
    fireEvent.change(await screen.findByDisplayValue("/setpoint"), {
      target: { value: "/speed" },
    });
    await screen.findByDisplayValue("/speed");
    expect(screen.getByText("PUT /setpoint")).toBeTruthy();
  });

  it("should keep the name of a message that two endpoints send", async () => {
    await renderWrite();
    await addEndpoint();
    fireEvent.contextMenu(
      await screen.findByRole("treeitem", { name: /New endpoint/ }),
    );
    fireEvent.click(await screen.findByText("Duplicate"));
    await waitFor(() =>
      expect(screen.getAllByRole("treeitem", { name: /New endpoint/ })).toHaveLength(2),
    );
    fireEvent.change(screen.getByPlaceholderText("/api/control"), {
      target: { value: "/speed" },
    });
    await screen.findByDisplayValue("/speed");
    expect(screen.queryByText("POST /speed")).toBeNull();
  });

  it("should show the request and the message of an endpoint", async () => {
    const value = createField("value", "float");
    const { ic, message } = await createFormat([value]);
    await renderWrite([createEndpoint("ep1", "/setpoint", ic, message, [value])]);
    fireEvent.click(await screen.findByRole("treeitem", { name: /\/setpoint/ }));
    await screen.findByText("Request");
    expect(screen.getByDisplayValue("/setpoint")).toBeTruthy();
    expect(await screen.findByText(message.name)).toBeTruthy();
  });

  it("should open a field under its endpoint with the name of its new command channel", async () => {
    const value = createField("value", "float");
    const { ic, message } = await createFormat([value]);
    const { dev } = await renderWrite([
      createEndpoint("ep1", "/setpoint", ic, message, [value]),
    ]);
    const source = await selectField("value");
    expect(
      screen.getByDisplayValue(
        `${channel.escapeInvalidName(dev.name)}_setpoint_value_cmd`,
      ),
    ).toBeTruthy();
    expect(source.getByRole("img", { name: "New channel" })).toBeTruthy();
  });

  it("should mark no new command channel for a field of a disabled endpoint", async () => {
    const value = createField("value", "float");
    const { ic, message } = await createFormat([value]);
    const { dev } = await renderWrite([
      { ...createEndpoint("ep1", "/setpoint", ic, message, [value]), disabled: true },
    ]);
    const source = await selectField("value");
    screen.getByDisplayValue(
      `${channel.escapeInvalidName(dev.name)}_setpoint_value_cmd`,
    );
    expect(source.queryByRole("img", { name: "New channel" })).toBeNull();
  });

  it("should show the numbered name that deploying gives a field channel whose name another endpoint takes", async () => {
    const value = createField("value", "float");
    const { ic, message } = await createFormat([value]);
    const { dev } = await renderWrite([
      createEndpoint("ep1", "/valve-1", ic, message, [value]),
      createEndpoint("ep2", "/valve_1", ic, message, [value]),
    ]);
    const name = `${channel.escapeInvalidName(dev.name)}_valve_1_value_cmd`;
    const fields = await screen.findAllByRole("treeitem", { name: /^value/ });
    fireEvent.click(fields[1]);
    expect(await screen.findByDisplayValue(`${name}_2`)).toBeTruthy();
  });

  it("should give each of two endpoints of one path and message its own command channels", async () => {
    const value = createField("value", "float");
    const { ic, message } = await createFormat([value]);
    const { container, dev, draft } = await renderWrite([
      createEndpoint("ep1", "/setpoint", ic, message, [value]),
      createEndpoint("ep2", "/setpoint", ic, message, [value]),
    ]);
    const name = `${channel.escapeInvalidName(dev.name)}_setpoint_value_cmd`;
    fireEvent.click((await screen.findAllByRole("treeitem", { name: /^value/ }))[1]);
    expect(await screen.findByDisplayValue(`${name}_2`)).toBeTruthy();
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      HTTP.Task.WRITE_SCHEMAS,
    );
    const keys = created.config.endpoints.map(({ message }) => {
      const binding = message?.bindings[0];
      return binding?.type === "channel" ? binding.channel : 0;
    });
    const channels = await client.channels.retrieve(keys);
    expect(channels.map((ch) => ch.name)).toEqual([name, `${name}_2`]);
  });

  it("should send a fixed value for a field of an endpoint set to Value", async () => {
    const value = createField("value", "float");
    const mode = createField("mode", "string");
    const { ic, message } = await createFormat([value, mode]);
    const { draft } = await renderWrite([
      createEndpoint("ep1", "/setpoint", ic, message, [value, mode]),
    ]);
    const source = await selectField("mode");
    fireEvent.click(source.getByRole("button", { name: "Value" }));
    const input = await source.findByRole("textbox");
    fireEvent.change(input, { target: { value: "auto" } });
    fireEvent.blur(input);
    await waitFor(async () =>
      expect((await retrieveEndpoints(draft.key))[0].message?.bindings).toEqual([
        { type: "channel", field: value.key, channel: 0 },
        { type: "value", field: mode.key, value: { type: "string", value: "auto" } },
      ]),
    );
  });

  it("should add a field to the message of an endpoint from its pane, and open it", async () => {
    const value = createField("value", "float");
    const { ic, message } = await createFormat([value]);
    await renderWrite([createEndpoint("ep1", "/setpoint", ic, message, [value])]);
    fireEvent.click(await screen.findByRole("treeitem", { name: /\/setpoint/ }));
    fireEvent.click(await screen.findByRole("button", { name: "Add field" }));
    const name = await screen.findByDisplayValue("Field 2");
    await waitFor(() => expect(document.activeElement).toBe(name));
    await waitFor(async () => {
      const [stored] = (await client.icds.retrieve(ic.key)).messages;
      if (stored.payload.format !== "json") throw new Error("expected a JSON message");
      expect(stored.payload.fields.map((f) => f.name)).toEqual(["value", "Field 2"]);
    });
  });

  it("should duplicate an endpoint with its message, but no command channels", async () => {
    const value = createField("value", "float");
    const { ic, message } = await createFormat([value]);
    const ep = createEndpoint("ep1", "/setpoint", ic, message, [value]);
    const cmd = await client.channels.create({
      name: uniqueName("http_cmd"),
      dataType: "float64",
      virtual: true,
    });
    ep.message = {
      message: message.key,
      bindings: [{ type: "channel", field: value.key, channel: cmd.key }],
    };
    const { draft } = await renderWrite([ep]);
    fireEvent.contextMenu(await screen.findByRole("treeitem", { name: /\/setpoint/ }));
    fireEvent.click(await screen.findByText("Duplicate"));
    await waitFor(async () => {
      const endpoints = await retrieveEndpoints(draft.key);
      expect(endpoints).toMatchObject([
        { key: "ep1", message: ep.message },
        {
          path: "/setpoint",
          icd: ic.key,
          message: {
            message: message.key,
            bindings: [{ type: "channel", field: value.key, channel: 0 }],
          },
        },
      ]);
      expect(endpoints[1].key).not.toBe("ep1");
    });
  });

  it("should switch an endpoint off through its checkbox", async () => {
    const value = createField("value", "float");
    const { ic, message } = await createFormat([value]);
    const { draft } = await renderWrite([
      createEndpoint("ep1", "/setpoint", ic, message, [value]),
    ]);
    const item = await screen.findByRole("treeitem", { name: /\/setpoint/ });
    fireEvent.click(within(item).getByRole("checkbox", { name: "Enabled" }));
    await waitFor(async () => {
      expect((await retrieveEndpoints(draft.key))[0].disabled).toBe(true);
    });
  });

  it("should send the message picked for an endpoint that sends none", async () => {
    const value = createField("value", "float");
    const { ic, message } = await createFormat([value]);
    const { draft } = await renderWrite([
      { ...HTTP.Task.writeEndpointZ.parse({}), key: "ep1", path: "/setpoint" },
    ]);
    fireEvent.click(await screen.findByRole("treeitem", { name: /\/setpoint/ }));
    await pickMessageFormat("Select message format", message.name, ic.name);
    await waitFor(async () => {
      const [ep] = await retrieveEndpoints(draft.key);
      expect(ep).toMatchObject({
        icd: ic.key,
        message: {
          message: message.key,
          bindings: [{ type: "channel", field: value.key, channel: 0 }],
        },
      });
    });
  });

  it("should create a command channel for each field of an enabled endpoint, and store it on the device", async () => {
    const value = createField("value", "float");
    const open = createField("open", "boolean");
    const { ic, message } = await createFormat([value, open]);
    const { container, dev, draft } = await renderWrite([
      createEndpoint("ep1", "/setpoint", ic, message, [value, open]),
      { ...createEndpoint("ep2", "/off", ic, message, [value, open]), disabled: true },
    ]);
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      HTTP.Task.WRITE_SCHEMAS,
    );
    const [on, off] = created.config.endpoints;
    const keys = (on.message?.bindings ?? []).map((b) =>
      b.type === "channel" ? b.channel : 0,
    );
    const channels = await client.channels.retrieve(keys);
    expect(channels.map(({ name }) => name)).toEqual([
      `${channel.escapeInvalidName(dev.name)}_setpoint_value_cmd`,
      `${channel.escapeInvalidName(dev.name)}_setpoint_open_cmd`,
    ]);
    expect(channels.map((ch) => ch.dataType.toString())).toEqual(["float64", "uint8"]);
    expect(off.message?.bindings).toEqual([
      { type: "channel", field: value.key, channel: 0 },
      { type: "channel", field: open.key, channel: 0 },
    ]);
    const updated = await client.devices.retrieve({
      key: dev.key,
      schemas: HTTP.Device.SCHEMAS,
    });
    expect(updated.properties.write).toEqual({
      [`ep1#${value.key}`]: keys[0],
      [`ep1#${open.key}`]: keys[1],
    });
  });
});
