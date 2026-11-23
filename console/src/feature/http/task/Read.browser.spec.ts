// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, http, icd, type Synnax, type task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { DataType, type payload, uuid } from "@synnaxlabs/x";
import { assertDefined, uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import type { z } from "zod";

import { HTTP } from "@/feature/http";
import { SCHEMAS } from "@/feature/http/device/types";
import { createHTTPDevice } from "@/feature/http/testutil";
import {
  awaitEditableForm,
  deployAndAwaitTask,
  preloadForms,
  renderTaskFormTab,
  type RenderTaskFormTabOptions,
} from "@/platform/task/testutil";
import { getLabeledInput } from "@/testutil";

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

// The form body renders against a device only.
const renderWithDevice = async (
  endpoints: HTTP.Task.ReadEndpoint[] = [],
  properties?: Partial<HTTP.Device.Properties>,
) => {
  const dev = await createHTTPDevice(client, { properties });
  const draft = await createDraft(client, createReadConfig(dev.key, endpoints));
  const rendered = await renderRead({ client, taskKey: draft.key });
  return { ...rendered, dev, draft };
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
    fireEvent.click(await screen.findByRole("treeitem", { name: /^temperature/ }));
    const box = await screen.findByRole("checkbox", { name: "Read temperature" });
    fireEvent.click(box);
    await waitFor(async () => expect(await bindings()).toEqual([]));
    fireEvent.click(box);
    await waitFor(async () =>
      expect(await bindings()).toEqual([
        { field: temperature.key, channel: ch.key, dataType: "" },
      ]),
    );
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
    const stored = await client.devices.retrieve({ key: dev.key, schemas: SCHEMAS });
    expect(Object.keys(stored.properties.read)).toEqual([
      `/data#${messages[0].message}`,
    ]);
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
});
