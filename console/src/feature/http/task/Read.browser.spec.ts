// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, http, icd, type Synnax, type task } from "@synnaxlabs/client";
import { createTestClient, holdRequests } from "@synnaxlabs/client/testutil";
import { DataType, type payload, uuid } from "@synnaxlabs/x";
import { assertDefined, uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { z } from "zod";

import { HTTP } from "@/feature/http";
import { createHTTPDevice } from "@/feature/http/testutil";
import { pickMessageFormat } from "@/platform/icd/testutil";
import {
  awaitEditableForm,
  awaitTare,
  clickDeploy,
  deployAndAwaitTask,
  preloadForms,
  renderTaskFormTab,
  type RenderTaskFormTabOptions,
  reportTaskRunning,
} from "@/platform/task/testutil";
import { findDialogTriggerByText, findIconButton, getLabeledInput } from "@/testutil";

// The form renders read-only until the update grant lands, and a preview field renders
// no input, so wait for it to become editable before querying fields.
const renderRead = async (options: RenderTaskFormTabOptions = {}) => {
  const rendered = await renderTaskFormTab(HTTP.Task.FORMS[HTTP.Task.READ_TYPE], {
    task: ZERO_DRAFT,
    ...options,
  });
  await awaitEditableForm();
  return rendered;
};

const addEndpoint = async (): Promise<void> => {
  fireEvent.click(await screen.findByRole("button", { name: "Add endpoint" }));
  await screen.findByText("Request");
};

const createReadConfig = (
  device: string,
  endpoints: HTTP.Task.ReadEndpoint[],
): z.infer<HTTP.Task.ReadSchemas["config"]> => ({
  ...HTTP.Task.READ_SCHEMAS.config.parse({}),
  device,
  endpoints,
});

// Drafts carry no key; the created row mints its own.
const ZERO_DRAFT: task.New<HTTP.Task.ReadSchemas> = {
  name: "HTTP read task",
  type: HTTP.Task.READ_TYPE,
  config: HTTP.Task.READ_SCHEMAS.config.parse({}),
};

const createDraft = async (
  client: Synnax,
  config: z.infer<HTTP.Task.ReadSchemas["config"]>,
) => await client.tasks.create({ ...ZERO_DRAFT, config }, HTTP.Task.READ_SCHEMAS);

preloadForms(HTTP.Task.FORMS);

const client = createTestClient();

const createField = (name: string, kind: payload.Kind = "float"): icd.JSONField =>
  icd.jsonFieldZ.parse({ name, pointer: `/${name}`, type: kind });

interface EndpointParams extends Partial<Omit<HTTP.Task.ReadEndpoint, "fields">> {
  /** The fields of the message that the endpoint reads, each bound to channel 0. */
  fields?: icd.JSONField[];
}

/** @returns an endpoint that reads a new message, in a new ICD, of its fields. */
const createEndpoint = async ({
  fields: messageFields = [],
  ...rest
}: EndpointParams = {}): Promise<HTTP.Task.ReadEndpoint> => {
  const ep = http.readEndpointZ.parse(rest);
  const message: icd.Message = {
    key: uuid.create(),
    name: `${ep.method} ${ep.path}`.trim(),
    payload: { format: "json", fields: messageFields },
  };
  const format = await client.icds.create({
    name: uniqueName("format"),
    messages: [message],
  });
  const bindings = messageFields.map(({ key }) =>
    icd.readBindingZ.parse({ field: key }),
  );
  return {
    ...ep,
    icd: format.key,
    messages: [icd.readMessageZ.parse({ message: message.key, bindings })],
  };
};

/** Adds a message of fields to the ICD of ep, and reads it after the others. */
const addMessage = async (
  ep: HTTP.Task.ReadEndpoint,
  name: string,
  fields: icd.JSONField[],
  match: icd.JSONMatch = { type: "string", pointer: "/kind", value: name },
): Promise<icd.Message> => {
  const icdKey = ep.icd;
  assertDefined(icdKey, "the endpoint reads no ICD");
  const message: icd.Message = {
    key: uuid.create(),
    name,
    payload: {
      format: "json",
      match,
      fields,
    },
  };
  await client.icds.dispatch(
    icdKey,
    icd.insert({
      path: ["messages"],
      value: message,
      after: ep.messages.at(-1)?.message,
    }),
  );
  const bindings = fields.map(({ key }) => icd.readBindingZ.parse({ field: key }));
  ep.messages.push(icd.readMessageZ.parse({ message: message.key, bindings }));
  return message;
};

// The form body renders against a device only. `as` is the client that renders the
// form, when not the one that saves the task.
const renderWithDevice = async (
  endpoints: HTTP.Task.ReadEndpoint[] = [],
  properties?: Partial<HTTP.Device.Properties>,
  as: Synnax = client,
) => {
  const dev = await createHTTPDevice(client, { properties });
  const draft = await createDraft(client, createReadConfig(dev.key, endpoints));
  const rendered = await renderRead({ client: as, taskKey: draft.key });
  return { ...rendered, dev, draft };
};

/**
 * Stops and then starts reading the field named name, once its checkbox enables. Waits
 * for bindings to hold none in between.
 */
const readAgain = async (
  name: string,
  bindings: () => Promise<unknown>,
): Promise<void> => {
  fireEvent.click(
    await screen.findByRole("treeitem", { name: new RegExp(`^${name}`) }),
  );
  const box = await screen.findByRole<HTMLInputElement>("checkbox", {
    name: `Read ${name}`,
  });
  await waitFor(() => expect(box.disabled).toBe(false));
  fireEvent.click(box);
  await waitFor(async () => expect(await bindings()).toEqual([]));
  fireEvent.click(box);
};

const retrieveEndpoints = async (key: string) =>
  (await client.tasks.retrieve({ key, schemas: HTTP.Task.READ_SCHEMAS })).config
    .endpoints;

const deploy = async (container: HTMLElement, key: string) =>
  await deployAndAwaitTask(client, container, key, HTTP.Task.READ_SCHEMAS);

describe("Read", () => {
  it("should ask for a device while the task has none", async () => {
    await createHTTPDevice(client);
    await renderRead();
    await screen.findByText("No device selected");
    expect(screen.queryByText("No endpoints")).toBeNull();
  });

  it("should show the empty state and add + select an endpoint", async () => {
    await renderWithDevice();
    await screen.findByText("Select an endpoint or field to configure");
    await screen.findByText("No endpoints");
    await addEndpoint();
    expect(screen.getByRole("button", { name: "GET" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "POST" })).toBeTruthy();
    expect(screen.getByPlaceholderText("/api/data")).toBeTruthy();
    expect(screen.getByText("Headers")).toBeTruthy();
    expect(screen.getByText("Query parameters")).toBeTruthy();
    expect(screen.queryByText("Select an endpoint or field to configure")).toBeNull();
  });

  it("should make an ICD named after the task for the message of a new endpoint", async () => {
    const { draft } = await renderWithDevice();
    await addEndpoint();
    const [ep] = await waitFor(async () => {
      const endpoints = await retrieveEndpoints(draft.key);
      expect(endpoints[0]?.icd).toBeDefined();
      return endpoints;
    });
    const icdKey = ep.icd;
    assertDefined(icdKey, "the endpoint reads no ICD");
    const format = await client.icds.retrieve(icdKey);
    expect(format.name).toBe("HTTP read task");
    expect(format.messages.map(({ name }) => name)).toEqual(["GET"]);
    expect(ep.messages).toEqual([{ message: format.messages[0].key, bindings: [] }]);
  });

  it("should add the message of a later endpoint to the ICD of the first", async () => {
    const first = await createEndpoint({ key: "ep1", path: "/a" });
    const { draft } = await renderWithDevice([first]);
    await screen.findByRole("treeitem", { name: /\/a/ });
    await addEndpoint();
    await waitFor(async () =>
      expect(await retrieveEndpoints(draft.key)).toHaveLength(2),
    );
    const [, added] = await retrieveEndpoints(draft.key);
    const icdKey = first.icd;
    assertDefined(icdKey, "the first endpoint reads no ICD");
    expect(added.icd).toBe(icdKey);
    await waitFor(async () =>
      expect(
        (await client.icds.retrieve(icdKey)).messages.map(({ name }) => name),
      ).toEqual(["GET /a", "GET"]),
    );
  });

  it("should number the name of a new message that the ICD already holds", async () => {
    const first = await createEndpoint({ key: "ep1" });
    const icdKey = first.icd;
    assertDefined(icdKey, "the first endpoint reads no ICD");
    const { draft } = await renderWithDevice([first]);
    await screen.findByRole("treeitem", { name: /GET/ });
    await addEndpoint();
    await waitFor(async () =>
      expect(await retrieveEndpoints(draft.key)).toHaveLength(2),
    );
    await waitFor(async () =>
      expect(
        (await client.icds.retrieve(icdKey)).messages.map(({ name }) => name),
      ).toEqual(["GET", "GET 2"]),
    );
  });

  it("should add a message to an endpoint from its details", async () => {
    const ep = await createEndpoint({ key: "ep1", path: "/a" });
    const icdKey = ep.icd;
    assertDefined(icdKey, "the endpoint reads no ICD");
    const { draft } = await renderWithDevice([ep]);
    fireEvent.click(await screen.findByRole("treeitem", { name: /\/a/ }));
    fireEvent.click(await screen.findByRole("button", { name: "Add message" }));
    await waitFor(async () => {
      const { messages } = await client.icds.retrieve(icdKey);
      expect(messages.map(({ name }) => name)).toEqual(["GET /a", "GET /a 2"]);
      const [{ messages: reads }] = await retrieveEndpoints(draft.key);
      expect(reads.map(({ message }) => message)).toEqual(
        messages.map(({ key }) => key),
      );
    });
  });

  it("should add a message from the context menu of an endpoint", async () => {
    const { draft } = await renderWithDevice([
      await createEndpoint({ key: "ep1", path: "/a" }),
    ]);
    fireEvent.contextMenu(await screen.findByRole("treeitem", { name: /\/a/ }));
    fireEvent.click(await screen.findByRole("menuitem", { name: "Add message" }));
    await screen.findByRole("treeitem", { name: "GET /a 2" });
    await waitFor(async () =>
      expect((await retrieveEndpoints(draft.key))[0].messages).toHaveLength(2),
    );
  });

  it("should read a new message in a new ICD for an endpoint that reads none", async () => {
    const { draft } = await renderWithDevice([
      http.readEndpointZ.parse({ key: "ep1", path: "/a" }),
    ]);
    fireEvent.click(await screen.findByRole("treeitem", { name: /\/a/ }));
    fireEvent.click(await screen.findByRole("button", { name: "Add message" }));
    const [ep] = await waitFor(async () => {
      const endpoints = await retrieveEndpoints(draft.key);
      expect(endpoints[0].icd).toBeDefined();
      return endpoints;
    });
    const icdKey = ep.icd;
    assertDefined(icdKey, "the endpoint reads no ICD");
    const format = await client.icds.retrieve(icdKey);
    expect(format.name).toBe("HTTP read task");
    expect(format.messages.map(({ name }) => name)).toEqual(["GET /a"]);
    expect(ep.messages).toEqual([{ message: format.messages[0].key, bindings: [] }]);
  });

  it("should add a field to the message of an endpoint and read it", async () => {
    const { draft } = await renderWithDevice([
      await createEndpoint({ key: "ep1", path: "/data", fields: [createField("v")] }),
    ]);
    fireEvent.click(await screen.findByRole("treeitem", { name: /\/data/ }));
    fireEvent.click(await screen.findByRole("button", { name: "Add field" }));
    await screen.findByDisplayValue("Field 2");
    await waitFor(async () =>
      expect((await retrieveEndpoints(draft.key))[0].messages[0].bindings).toHaveLength(
        2,
      ),
    );
  });

  it("should read a message of another ICD with each of its fields", async () => {
    const ep = await createEndpoint({
      key: "ep1",
      path: "/data",
      fields: [createField("temperature")],
    });
    const pressure = createField("pressure");
    const mode = createField("mode", "string");
    const message: icd.Message = {
      key: uuid.create(),
      name: uniqueName("status"),
      payload: { format: "json", fields: [pressure, mode] },
    };
    const shared = await client.icds.create({
      name: uniqueName("pump"),
      messages: [message],
    });
    const { draft } = await renderWithDevice([ep]);
    fireEvent.click(await screen.findByRole("treeitem", { name: /\/data/ }));
    fireEvent.click(await findDialogTriggerByText("GET /data"));
    const dialogs = await screen.findAllByRole("dialog");
    const dialog = within(dialogs[dialogs.length - 1]);
    fireEvent.change(await dialog.findByPlaceholderText("Search message formats..."), {
      target: { value: shared.name },
    });
    // waitFor resolves a timer tick after its match. A render in that tick can replace
    // the option. A click in the callback goes to the option on the page.
    await waitFor(() => {
      const [match] = dialog
        .getAllByText(message.name)
        .filter((n) => n.parentElement?.textContent?.includes(shared.name));
      if (match == null) throw new Error(`no ${message.name} of ${shared.name}`);
      fireEvent.click(match);
    });
    await waitFor(async () =>
      expect((await retrieveEndpoints(draft.key))[0]).toMatchObject({
        icd: shared.key,
        messages: [
          {
            message: message.key,
            bindings: [
              { field: pressure.key, channel: 0 },
              { field: mode.key, channel: 0 },
            ],
          },
        ],
      }),
    );
    await screen.findByRole("treeitem", { name: /^pressure/ });
    await screen.findByRole("treeitem", { name: /^mode/ });
  });

  it("should reveal the request body field when the method switches to POST", async () => {
    await renderWithDevice();
    await addEndpoint();
    expect(screen.queryByPlaceholderText('{"query": "latest"}')).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "POST" }));
    await screen.findByPlaceholderText('{"query": "latest"}');
    fireEvent.click(screen.getByRole("button", { name: "GET" }));
    await waitFor(() =>
      expect(screen.queryByPlaceholderText('{"query": "latest"}')).toBeNull(),
    );
  });

  it("should disable and enable an endpoint through the context menu", async () => {
    const { draft } = await renderWithDevice([
      await createEndpoint({ key: "ep1", path: "/data", fields: [createField("v")] }),
    ]);
    const item = () => screen.findByRole("treeitem", { name: /\/data/ });
    fireEvent.contextMenu(await item());
    fireEvent.click(await screen.findByText("Disable"));
    await waitFor(async () =>
      expect((await retrieveEndpoints(draft.key))[0].disabled).toBe(true),
    );
    fireEvent.contextMenu(await item());
    fireEvent.click(await screen.findByText("Enable"));
    await waitFor(async () =>
      expect((await retrieveEndpoints(draft.key))[0].disabled).toBe(false),
    );
  });

  it("should remove an endpoint through the context menu", async () => {
    const { draft } = await renderWithDevice([
      await createEndpoint({ key: "ep1", path: "/a" }),
      await createEndpoint({ key: "ep2", path: "/b" }),
    ]);
    fireEvent.contextMenu(await screen.findByRole("treeitem", { name: /\/a/ }));
    fireEvent.click(await screen.findByText("Remove"));
    await waitFor(async () =>
      expect((await retrieveEndpoints(draft.key)).map(({ key }) => key)).toEqual([
        "ep2",
      ]),
    );
  });

  it("should offer Reload Console from the endpoint context menu", async () => {
    await renderWithDevice([await createEndpoint({ key: "ep1", path: "/api/v1" })]);
    fireEvent.contextMenu(await screen.findByRole("treeitem", { name: /\/api\/v1/ }));
    expect(await screen.findByText("Reload Console")).toBeTruthy();
  });

  it("should toggle the NaN fill switch", async () => {
    await renderRead();
    const toggle = getLabeledInput("NaN fill");
    expect(toggle.checked).toBe(false);
    fireEvent.click(toggle);
    expect(toggle.checked).toBe(true);
  });

  it("should tare a float field from its button while the task runs", async () => {
    const float = await client.channels.create({
      name: uniqueName("tare"),
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    const ep = await createEndpoint({
      key: "ep1",
      path: "/data",
      fields: [createField("value")],
    });
    ep.messages[0].bindings[0].channel = float.key;
    const dev = await createHTTPDevice(client);
    const draft = await createDraft(client, createReadConfig(dev.key, [ep]));
    await reportTaskRunning(client, draft);
    const args = await awaitTare(client, draft.key, async () => {
      const { container } = await renderRead({ client, taskKey: draft.key });
      fireEvent.click(await findIconButton(container, "tare"));
    });
    expect(args).toEqual({ keys: [float.key] });
  });

  it("should open each endpoint of a saved task with the fields it reads", async () => {
    await renderWithDevice([
      await createEndpoint({
        key: "ep1",
        path: "/plant",
        fields: [createField("temperature"), createField("pressure")],
      }),
      await createEndpoint({
        key: "ep2",
        path: "/status",
        fields: [createField("mode")],
      }),
    ]);
    await screen.findByRole("treeitem", { name: /\/plant/ });
    await screen.findByRole("treeitem", { name: /\/status/ });
    const tree = screen.getByRole("tree");
    for (const name of ["temperature", "pressure", "mode"])
      expect(within(tree).getByText(name)).toBeTruthy();
  });

  it("should read a field again with the channel that the device stores for it", async () => {
    const ch = await client.channels.create({
      name: uniqueName("temperature"),
      dataType: DataType.FLOAT64,
      virtual: true,
    });
    const temperature = createField("temperature");
    const ep = await createEndpoint({
      key: "ep1",
      path: "/data",
      fields: [temperature],
    });
    const stored = { index: 0, channels: { "/temperature": ch.key } };
    const { draft } = await renderWithDevice([ep], {
      read: { [`/data#${ep.messages[0].message}`]: stored },
    });
    const bindings = async () =>
      (await retrieveEndpoints(draft.key))[0].messages[0].bindings;
    await readAgain("temperature", bindings);
    await waitFor(async () =>
      expect(await bindings()).toEqual([
        { field: temperature.key, channel: ch.key, dataType: "" },
      ]),
    );
  });

  it("should keep the stored channel of a pointer when the message format changes", async () => {
    const ch = await client.channels.create({
      name: uniqueName("temperature"),
      dataType: DataType.FLOAT64,
      virtual: true,
    });
    const ep = await createEndpoint({
      key: "ep1",
      path: "/data",
      fields: [createField("temperature")],
    });
    const next = createField("temperature");
    const message: icd.Message = {
      key: uuid.create(),
      name: uniqueName("status"),
      payload: { format: "json", fields: [next] },
    };
    const shared = await client.icds.create({
      name: uniqueName("pump"),
      messages: [message],
    });
    const stored = { index: 0, channels: { "/temperature": ch.key } };
    const { draft } = await renderWithDevice([ep], {
      read: { [`/data#${ep.messages[0].message}`]: stored },
    });
    fireEvent.click(await screen.findByRole("treeitem", { name: /\/data/ }));
    await pickMessageFormat("GET /data", message.name, shared.name);
    await waitFor(async () =>
      expect((await retrieveEndpoints(draft.key))[0].messages).toEqual([
        {
          message: message.key,
          bindings: [{ field: next.key, channel: ch.key, dataType: "" }],
        },
      ]),
    );
  });

  it("should reuse the channel that an older Console stored under the path", async () => {
    const ch = await client.channels.create({
      name: uniqueName("temperature"),
      dataType: DataType.FLOAT64,
      virtual: true,
    });
    const temperature = createField("temperature");
    const ep = await createEndpoint({
      key: "ep1",
      path: "/data",
      fields: [temperature],
    });
    const { draft } = await renderWithDevice([ep], {
      read: { "/data": { index: 0, channels: { "/temperature": ch.key } } },
    });
    const bindings = async () =>
      (await retrieveEndpoints(draft.key))[0].messages[0].bindings;
    await readAgain("temperature", bindings);
    await waitFor(async () =>
      expect(await bindings()).toEqual([
        { field: temperature.key, channel: ch.key, dataType: "" },
      ]),
    );
  });

  it("should reuse a channel stored under the path for a second endpoint on it", async () => {
    const [temperature, pressure] = await client.channels.create([
      { name: uniqueName("temperature"), dataType: DataType.FLOAT64, virtual: true },
      { name: uniqueName("pressure"), dataType: DataType.FLOAT64, virtual: true },
    ]);
    const first = await createEndpoint({
      key: "ep1",
      path: "/data",
      fields: [createField("temperature")],
    });
    const pressureField = createField("pressure");
    const second = await createEndpoint({
      key: "ep2",
      method: "POST",
      path: "/data",
      fields: [pressureField],
    });
    const channels = { "/temperature": temperature.key, "/pressure": pressure.key };
    const { draft } = await renderWithDevice([first, second], {
      read: { "/data": { index: 0, channels } },
    });
    const bindings = async () =>
      (await retrieveEndpoints(draft.key))[1].messages[0].bindings;
    await readAgain("pressure", bindings);
    await waitFor(async () =>
      expect(await bindings()).toEqual([
        { field: pressureField.key, channel: pressure.key, dataType: "" },
      ]),
    );
  });

  it("should not reuse a stored channel that another endpoint reads", async () => {
    const ch = await client.channels.create({
      name: uniqueName("temperature"),
      dataType: DataType.FLOAT64,
      virtual: true,
    });
    const temperature = createField("temperature");
    const ep = await createEndpoint({
      key: "ep1",
      path: "/data",
      fields: [temperature],
    });
    const other = await createEndpoint({
      key: "ep2",
      path: "/other",
      fields: [createField("reading")],
    });
    other.messages[0].bindings[0].channel = ch.key;
    const stored = { index: 0, channels: { "/temperature": ch.key } };
    const { draft } = await renderWithDevice([ep, other], {
      read: { [`/data#${ep.messages[0].message}`]: stored },
    });
    const bindings = async () =>
      (await retrieveEndpoints(draft.key))[0].messages[0].bindings;
    await readAgain("temperature", bindings);
    await waitFor(async () =>
      expect(await bindings()).toEqual([
        { field: temperature.key, channel: 0, dataType: "" },
      ]),
    );
  });

  it("should not reuse a stored channel of another endpoint with the same path", async () => {
    const ch = await client.channels.create({
      name: uniqueName("temperature"),
      dataType: DataType.FLOAT64,
      virtual: true,
    });
    const other = await createEndpoint({
      key: "ep1",
      method: "POST",
      path: "/query",
      disabled: true,
      fields: [
        icd.jsonFieldZ.parse({
          name: "reading",
          pointer: "/temperature",
          type: "float",
        }),
      ],
    });
    other.messages[0].bindings[0].channel = ch.key;
    const temperature = createField("temperature");
    const ep = await createEndpoint({
      key: "ep2",
      method: "POST",
      path: "/query",
      fields: [temperature],
    });
    const stored = { index: 0, channels: { "/temperature": ch.key } };
    const { draft } = await renderWithDevice([other, ep], {
      read: { [`/query#${other.messages[0].message}`]: stored },
    });
    const bindings = async () =>
      (await retrieveEndpoints(draft.key))[1].messages[0].bindings;
    await readAgain("temperature", bindings);
    await waitFor(async () =>
      expect(await bindings()).toEqual([
        { field: temperature.key, channel: 0, dataType: "" },
      ]),
    );
  });

  it("should not reuse a channel stored under the path that another endpoint binds", async () => {
    const ch = await client.channels.create({
      name: uniqueName("temperature"),
      dataType: DataType.FLOAT64,
      virtual: true,
    });
    const other = await createEndpoint({
      key: "ep1",
      method: "POST",
      path: "/query",
      disabled: true,
      fields: [
        icd.jsonFieldZ.parse({
          name: "reading",
          pointer: "/temperature",
          type: "float",
        }),
      ],
    });
    other.messages[0].bindings[0].channel = ch.key;
    const temperature = createField("temperature");
    const ep = await createEndpoint({
      key: "ep2",
      method: "POST",
      path: "/query",
      fields: [temperature],
    });
    const { draft } = await renderWithDevice([other, ep], {
      read: { "/query": { index: 0, channels: { "/temperature": ch.key } } },
    });
    const bindings = async () =>
      (await retrieveEndpoints(draft.key))[1].messages[0].bindings;
    await readAgain("temperature", bindings);
    await waitFor(async () =>
      expect(await bindings()).toEqual([
        { field: temperature.key, channel: 0, dataType: "" },
      ]),
    );
  });

  it("should keep the channel of the first endpoint that reads a path and message", async () => {
    const temperature = createField("temperature");
    const first = await createEndpoint({
      key: "ep1",
      method: "POST",
      path: "/query",
      fields: [temperature],
    });
    const later: HTTP.Task.ReadEndpoint = {
      ...http.readEndpointZ.parse({ key: "ep2", method: "POST", path: "/query" }),
      icd: first.icd,
      messages: [
        icd.readMessageZ.parse({
          message: first.messages[0].message,
          bindings: [icd.readBindingZ.parse({ field: temperature.key })],
        }),
      ],
    };
    const { container, draft, dev } = await renderWithDevice([first, later]);
    await screen.findAllByRole("treeitem", { name: /\/query/ });
    const created = await deploy(container, draft.key);
    const [kept, other] = created.config.endpoints.map(
      ({ messages }) => messages[0].bindings[0].channel,
    );
    expect(kept).not.toBe(other);
    const { properties } = await client.devices.retrieve({
      key: dev.key,
      schemas: HTTP.Device.SCHEMAS,
    });
    expect(
      properties.read[`/query#${first.messages[0].message}`].channels["/temperature"],
    ).toBe(kept);
    const channelOf = async () =>
      (await retrieveEndpoints(draft.key))[0].messages[0].bindings[0]?.channel;
    const [item] = await screen.findAllByRole("treeitem", { name: /^temperature/ });
    fireEvent.click(item);
    const box = await within(item).findByRole<HTMLInputElement>("checkbox", {
      name: "Read temperature",
    });
    await waitFor(() => expect(box.disabled).toBe(false));
    fireEvent.click(box);
    await waitFor(async () => expect(await channelOf()).toBeUndefined());
    fireEvent.click(box);
    await waitFor(async () => expect(await channelOf()).toBe(kept));
  });

  it("should not reuse a stored channel that the field does not convert to", async () => {
    const ch = await client.channels.create({
      name: uniqueName("mode"),
      dataType: DataType.FLOAT64,
      virtual: true,
    });
    const mode = createField("mode", "string");
    const ep = await createEndpoint({ key: "ep1", path: "/data", fields: [mode] });
    const stored = { index: 0, channels: { "/mode": ch.key } };
    const { draft } = await renderWithDevice([ep], {
      read: { [`/data#${ep.messages[0].message}`]: stored },
    });
    const bindings = async () =>
      (await retrieveEndpoints(draft.key))[0].messages[0].bindings;
    await readAgain("mode", bindings);
    await waitFor(async () =>
      expect(await bindings()).toEqual([{ field: mode.key, channel: 0, dataType: "" }]),
    );
  });

  it("should not reuse a stored channel that no longer exists", async () => {
    const ch = await client.channels.create({
      name: uniqueName("temperature"),
      dataType: DataType.FLOAT64,
      virtual: true,
    });
    await client.channels.delete(ch.key);
    const temperature = createField("temperature");
    const ep = await createEndpoint({
      key: "ep1",
      path: "/data",
      fields: [temperature],
    });
    const stored = { index: 0, channels: { "/temperature": ch.key } };
    const { draft } = await renderWithDevice([ep], {
      read: { [`/data#${ep.messages[0].message}`]: stored },
    });
    const bindings = async () =>
      (await retrieveEndpoints(draft.key))[0].messages[0].bindings;
    await readAgain("temperature", bindings);
    await waitFor(async () =>
      expect(await bindings()).toEqual([
        { field: temperature.key, channel: 0, dataType: "" },
      ]),
    );
  });

  it("should wait for the data types of the stored channels before it binds", async () => {
    const ch = await client.channels.create({
      name: uniqueName("temperature"),
      dataType: DataType.FLOAT64,
      virtual: true,
    });
    const ep = await createEndpoint({
      key: "ep1",
      path: "/data",
      fields: [createField("temperature")],
    });
    // A new client has no channels cached, so the form fetches the stored one.
    const viewer = createTestClient();
    await viewer.connect();
    const { release } = holdRequests(viewer, "/channel/retrieve");
    const stored = { index: 0, channels: { "/temperature": ch.key } };
    await renderWithDevice(
      [ep],
      { read: { [`/data#${ep.messages[0].message}`]: stored } },
      viewer,
    );
    fireEvent.click(await screen.findByRole("treeitem", { name: /\/data/ }));
    const addField = await screen.findByRole("button", { name: "Add field" });
    const format = await findDialogTriggerByText("GET /data");
    const read = await screen.findByRole<HTMLInputElement>("checkbox", {
      name: "Read temperature",
    });
    expect(addField.getAttribute("aria-disabled")).toBe("true");
    expect(format.getAttribute("aria-disabled")).toBe("true");
    expect(read.disabled).toBe(true);
    release();
    await waitFor(() => {
      expect(addField.getAttribute("aria-disabled")).toBeNull();
      expect(format.getAttribute("aria-disabled")).toBeNull();
      expect(read.disabled).toBe(false);
    });
  });
  it("should create and bind the channel of each field on deploy", async () => {
    const ep = await createEndpoint({
      key: "ep1",
      path: "/data",
      fields: [createField("v")],
    });
    const { container, draft, dev } = await renderWithDevice([ep]);
    await screen.findByRole("treeitem", { name: /\/data/ });
    const created = await deploy(container, draft.key);
    const [{ messages }] = created.config.endpoints;
    const [binding] = messages[0].bindings;
    expect(binding.channel).not.toBe(0);
    const ch = await client.channels.retrieve(binding.channel);
    expect(ch.name).toBe(`${channel.escapeInvalidName(dev.name)}_data_v`);
    const stored = await client.devices.retrieve({
      key: dev.key,
      schemas: HTTP.Device.SCHEMAS,
    });
    expect(Object.keys(stored.properties.read)).toEqual([
      `/data#${messages[0].message}`,
    ]);
  });

  it("should name the channels of a later message after the endpoint and store them by that message", async () => {
    const ep = await createEndpoint({
      key: "ep1",
      path: "/data",
      fields: [createField("v")],
    });
    const status = await addMessage(ep, "status", [createField("w")]);
    const { container, draft, dev } = await renderWithDevice([ep]);
    await screen.findAllByRole("treeitem", { name: /\/data/ });
    const created = await deploy(container, draft.key);
    const [, later] = created.config.endpoints[0].messages;
    const ch = await client.channels.retrieve(later.bindings[0].channel);
    expect(ch.name).toBe(`${channel.escapeInvalidName(dev.name)}_data_w`);
    const stored = await client.devices.retrieve({
      key: dev.key,
      schemas: HTTP.Device.SCHEMAS,
    });
    expect(stored.properties.read[`/data#${status.key}`].channels).toEqual({
      "/w": ch.key,
    });
  });

  it("should add the kind of a later message to a channel name that another message takes", async () => {
    const ep = await createEndpoint({
      key: "ep1",
      path: "/data",
      fields: [createField("v")],
    });
    await addMessage(ep, "code", [createField("code"), createField("v")], {
      type: "present",
      pointer: "/code",
    });
    const { container, draft, dev } = await renderWithDevice([ep]);
    await screen.findAllByRole("treeitem", { name: /\/data/ });
    const created = await deploy(container, draft.key);
    const [first, later] = created.config.endpoints[0].messages;
    const names = await client.channels.retrieve(
      [...first.bindings, ...later.bindings].map(({ channel }) => channel),
    );
    const prefix = `${channel.escapeInvalidName(dev.name)}_data`;
    expect(names.map(({ name }) => name)).toEqual([
      `${prefix}_v`,
      `${prefix}_code`,
      `${prefix}_code_v`,
    ]);
  });

  it("should deploy with the channels stored under the path and move them to the message", async () => {
    const time = await client.channels.create({
      name: uniqueName("time"),
      dataType: DataType.TIMESTAMP,
      isIndex: true,
    });
    const ch = await client.channels.create({
      name: uniqueName("temperature"),
      dataType: DataType.FLOAT64,
      index: time.key,
    });
    const ep = await createEndpoint({
      key: "ep1",
      path: "/data",
      fields: [createField("temperature")],
    });
    const stored = { index: time.key, channels: { "/temperature": ch.key } };
    const { container, draft, dev } = await renderWithDevice([ep], {
      read: { "/data": stored },
    });
    await screen.findByRole("treeitem", { name: /\/data/ });
    const created = await deploy(container, draft.key);
    expect(created.config.endpoints[0].messages[0].bindings[0].channel).toBe(ch.key);
    const { properties } = await client.devices.retrieve({
      key: dev.key,
      schemas: HTTP.Device.SCHEMAS,
    });
    expect(properties.read).toEqual({
      [`/data#${ep.messages[0].message}`]: stored,
    });
  });

  it("should deploy a new channel for a field whose stored channel a disabled endpoint binds", async () => {
    const ch = await client.channels.create({
      name: uniqueName("temperature"),
      dataType: DataType.FLOAT64,
      virtual: true,
    });
    const other = await createEndpoint({
      key: "ep1",
      method: "POST",
      path: "/query",
      disabled: true,
      fields: [createField("temperature")],
    });
    other.messages[0].bindings[0].channel = ch.key;
    const ep = await createEndpoint({
      key: "ep2",
      path: "/query",
      fields: [createField("temperature")],
    });
    const stored = { index: 0, channels: { "/temperature": ch.key } };
    const { container, draft, dev } = await renderWithDevice([other, ep], {
      read: { "/query": stored },
    });
    await screen.findAllByRole("treeitem", { name: /\/query/ });
    const created = await deploy(container, draft.key);
    const bound = created.config.endpoints[1].messages[0].bindings[0].channel;
    expect(bound).not.toBe(0);
    expect(bound).not.toBe(ch.key);
    const { properties } = await client.devices.retrieve({
      key: dev.key,
      schemas: HTTP.Device.SCHEMAS,
    });
    expect(properties.read[`/query#${other.messages[0].message}`]).toEqual(stored);
    expect(properties.read).not.toHaveProperty("/query");
  });

  it("should store the channels under a path for each endpoint that reads it, disabled ones too", async () => {
    const time = await client.channels.create({
      name: uniqueName("time"),
      dataType: DataType.TIMESTAMP,
      isIndex: true,
    });
    const [a, b] = await client.channels.create([
      { name: uniqueName("a"), dataType: DataType.FLOAT64, index: time.key },
      { name: uniqueName("b"), dataType: DataType.FLOAT64, index: time.key },
    ]);
    const first = await createEndpoint({
      key: "ep1",
      path: "/data",
      fields: [createField("a")],
    });
    first.messages[0].bindings[0].channel = a.key;
    const second = await createEndpoint({
      key: "ep2",
      method: "POST",
      path: "/data",
      disabled: true,
      fields: [createField("b")],
    });
    second.messages[0].bindings[0].channel = b.key;
    const stored = { index: time.key, channels: { "/a": a.key, "/b": b.key } };
    const { container, draft, dev } = await renderWithDevice([first, second], {
      read: { "/data": stored },
    });
    await screen.findAllByRole("treeitem", { name: /\/data/ });
    const created = await deploy(container, draft.key);
    expect(created.config.endpoints[0].messages[0].bindings[0].channel).toBe(a.key);
    const { properties } = await client.devices.retrieve({
      key: dev.key,
      schemas: HTTP.Device.SCHEMAS,
    });
    expect(properties.read).toEqual({
      [`/data#${first.messages[0].message}`]: stored,
      [`/data#${second.messages[0].message}`]: { index: 0, channels: { "/b": b.key } },
    });
  });

  it("should keep the channel and index of an enabled endpoint before a disabled one", async () => {
    const time = await client.channels.create({
      name: uniqueName("time"),
      dataType: DataType.TIMESTAMP,
      isIndex: true,
    });
    const [a, b] = await client.channels.create([
      { name: uniqueName("a"), dataType: DataType.FLOAT64, index: time.key },
      { name: uniqueName("b"), dataType: DataType.FLOAT64, index: time.key },
    ]);
    const disabled = await createEndpoint({
      key: "ep1",
      path: "/data",
      disabled: true,
      fields: [createField("a")],
    });
    disabled.messages[0].bindings[0].channel = a.key;
    const enabled = await createEndpoint({
      key: "ep2",
      method: "POST",
      path: "/data",
      fields: [createField("b")],
    });
    enabled.messages[0].bindings[0].channel = b.key;
    const stored = { index: time.key, channels: { "/a": a.key, "/b": b.key } };
    const { container, draft, dev } = await renderWithDevice([disabled, enabled], {
      read: { "/data": stored },
    });
    await screen.findAllByRole("treeitem", { name: /\/data/ });
    const created = await deploy(container, draft.key);
    expect(created.config.endpoints[1].messages[0].bindings[0].channel).toBe(b.key);
    const { properties } = await client.devices.retrieve({
      key: dev.key,
      schemas: HTTP.Device.SCHEMAS,
    });
    expect(properties.read).toEqual({
      [`/data#${disabled.messages[0].message}`]: {
        index: 0,
        channels: { "/a": a.key },
      },
      [`/data#${enabled.messages[0].message}`]: stored,
    });
  });

  it("should keep the channels of an earlier message when a later one fails", async () => {
    const time = await client.channels.create({
      name: uniqueName("time"),
      dataType: DataType.TIMESTAMP,
      isIndex: true,
    });
    const notIndex = await client.channels.create({
      name: uniqueName("not_index"),
      dataType: DataType.FLOAT64,
      index: time.key,
    });
    const ep = await createEndpoint({
      key: "ep1",
      path: "/data",
      fields: [createField("v")],
    });
    const failing = await addMessage(ep, "failing", [createField("w")]);
    const { container, dev } = await renderWithDevice([ep], {
      read: { [`/data#${failing.key}`]: { index: notIndex.key, channels: {} } },
    });
    await screen.findAllByRole("treeitem", { name: /\/data/ });
    await clickDeploy(container);
    const kept = await waitFor(async () => {
      const { properties } = await client.devices.retrieve({
        key: dev.key,
        schemas: HTTP.Device.SCHEMAS,
      });
      const channels = properties.read[`/data#${ep.messages[0].message}`]?.channels;
      expect(channels?.["/v"]).toBeGreaterThan(0);
      return channels["/v"];
    });
    const ch = await client.channels.retrieve(kept);
    expect(ch.name).toBe(`${channel.escapeInvalidName(dev.name)}_data_v`);
  });

  it("should skip a disabled endpoint on deploy", async () => {
    const ep = await createEndpoint({
      key: "ep1",
      path: "/data",
      disabled: true,
      fields: [createField("v")],
    });
    const { container, draft } = await renderWithDevice([ep]);
    await screen.findByRole("treeitem", { name: /\/data/ });
    const created = await deploy(container, draft.key);
    expect(created.config.endpoints[0].messages[0].bindings[0].channel).toBe(0);
  });

  it("should drop a request body left over from POST when the method switches to GET", async () => {
    const { container, draft } = await renderWithDevice([
      await createEndpoint({
        key: "ep1",
        path: "/switched",
        method: "POST",
        body: '{"query": "latest"}',
        fields: [createField("temperature")],
      }),
      await createEndpoint({
        key: "ep2",
        path: "/kept",
        method: "POST",
        body: '{"query": "kept"}',
        fields: [createField("pressure")],
      }),
    ]);
    fireEvent.click(await screen.findByRole("treeitem", { name: /\/switched/ }));
    await screen.findByDisplayValue('{"query": "latest"}');
    fireEvent.click(screen.getByRole("button", { name: "GET" }));
    await waitFor(() =>
      expect(screen.queryByDisplayValue('{"query": "latest"}')).toBeNull(),
    );
    const created = await deploy(container, draft.key);
    const endpoints = created.config.endpoints;
    expect(endpoints.find((e) => e.key === "ep1")?.body).toBe("");
    expect(endpoints.find((e) => e.key === "ep2")?.body).toBe('{"query": "kept"}');
  });
});

describe("plotChannels", () => {
  it("should find the enabled fields of every endpoint", () => {
    const plotChannels = HTTP.Task.INTEGRATION.plotChannels?.[HTTP.Task.READ_TYPE];
    assertDefined(plotChannels, "read tasks have no channels to plot");
    const config = {
      endpoints: [
        {
          fields: [
            { channel: 1, disabled: false },
            { channel: 2, disabled: true },
          ],
        },
        { fields: [{ channel: 3, disabled: false }] },
      ],
    };
    expect(plotChannels(config)).toEqual([1, 3]);
  });

  it("should find the bound channels of each enabled endpoint that reads an ICD", () => {
    const plotChannels = HTTP.Task.INTEGRATION.plotChannels?.[HTTP.Task.READ_TYPE];
    assertDefined(plotChannels, "read tasks have no channels to plot");
    const endpoint = (disabled: boolean, ...channels: number[][]) => ({
      disabled,
      icd: uuid.create(),
      messages: channels.map((keys) => ({
        message: uuid.create(),
        bindings: keys.map((channel) => ({ field: uuid.create(), channel })),
      })),
    });
    const config = {
      endpoints: [
        endpoint(false, [1, 0], [2]),
        endpoint(true, [3]),
        { disabled: true, fields: [{ channel: 4, disabled: false }] },
        { fields: [{ channel: 5, disabled: false }] },
      ],
    };
    expect(plotChannels(config)).toEqual([1, 2, 5]);
  });
});
