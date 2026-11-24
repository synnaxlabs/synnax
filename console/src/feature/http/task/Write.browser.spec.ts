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
import { fireEvent, screen, waitFor } from "@testing-library/react";
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

describe("Write", () => {
  // This first test pays the one-time load of the lazy form, which takes over 10 s on
  // a loaded CI runner.
  it("should add an endpoint that sends a new message, in an ICD named after the task", async () => {
    const { draft } = await renderWrite();
    await screen.findByText("Select an endpoint to configure");
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

  it("should show the request and the message of an endpoint", async () => {
    const value = createField("value", "float");
    const { ic, message } = await createFormat([value]);
    await renderWrite([createEndpoint("ep1", "/setpoint", ic, message, [value])]);
    fireEvent.click(await screen.findByRole("option", { name: /\/setpoint/ }));
    await screen.findByText("Request");
    expect(screen.getByDisplayValue("/setpoint")).toBeTruthy();
    expect(await screen.findByText(message.name)).toBeTruthy();
    expect(await screen.findByText("value")).toBeTruthy();
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
    fireEvent.contextMenu(await screen.findByRole("option", { name: /\/setpoint/ }));
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
    fireEvent.click(await screen.findByRole("checkbox", { name: "Enabled" }));
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
    fireEvent.click(await screen.findByRole("option", { name: /\/setpoint/ }));
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
      [`/setpoint#${value.key}`]: keys[0],
      [`/setpoint#${open.key}`]: keys[1],
    });
  });
});
