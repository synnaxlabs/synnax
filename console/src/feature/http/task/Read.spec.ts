// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { http, type Synnax, type task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { DataType } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { type z } from "zod";

import { HTTP } from "@/feature/http";
import { createHTTPDevice } from "@/feature/http/testutil";
import { Device } from "@/platform/device";
import {
  awaitEditableForm,
  clickDeploy,
  commitFieldInput,
  deployAndAwaitTask,
  renderTaskFormTab,
  type RenderTaskFormTabOptions,
  selectFromDropdown,
} from "@/platform/task/testutil";
import { getLabeledInput, uniqueName } from "@/testutil";

// The form renders read-only until the update grant lands, and a preview field renders
// no input, so wait for it to become editable before querying fields.
const renderRead = async (options: RenderTaskFormTabOptions = {}) => {
  const rendered = await renderTaskFormTab(HTTP.Task.Read, {
    task: ZERO_DRAFT,
    ...options,
  });
  await awaitEditableForm();
  return rendered;
};

const addField = (): void => {
  fireEvent.click(screen.getAllByRole("button", { name: "Add field" })[0]);
};

const addEndpoint = async (): Promise<void> => {
  fireEvent.click(await screen.findByRole("button", { name: "Add endpoint" }));
  await screen.findByText("Request");
};

const createReadField = (
  key: string,
  pointer: string,
  overrides: Record<string, unknown> = {},
): HTTP.Task.ReadField => {
  const type = overrides.type ?? "float";
  const dataTypes: Record<string, string> = {
    float: "float64",
    integer: "int64",
    unsigned: "uint64",
  };
  return http.readFieldZ.parse({
    type,
    dataType: typeof type === "string" ? dataTypes[type] : undefined,
    ...overrides,
    key,
    pointer,
  });
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

describe("Read", () => {
  it("should show the empty state and add + select an endpoint", async () => {
    await renderRead();
    await screen.findByText("Select an endpoint or field to configure");
    await screen.findByText("No endpoints");
    await addEndpoint();
    expect(screen.getByRole("button", { name: "GET" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "POST" })).toBeTruthy();
    expect(screen.getByPlaceholderText("/api/data")).toBeTruthy();
    expect(screen.getByText("Headers")).toBeTruthy();
    expect(screen.getByText("Query parameters")).toBeTruthy();
    expect(screen.queryByText("New field")).toBeNull();
    expect(screen.queryByText("Select an endpoint or field to configure")).toBeNull();
  });

  it("should reveal the request body field when the method switches to POST", async () => {
    await renderRead();
    await addEndpoint();
    expect(screen.queryByPlaceholderText('{"query": "latest"}')).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "POST" }));
    await screen.findByPlaceholderText('{"query": "latest"}');
    fireEvent.click(screen.getByRole("button", { name: "GET" }));
    await waitFor(() =>
      expect(screen.queryByPlaceholderText('{"query": "latest"}')).toBeNull(),
    );
  });

  it("should add a timestamp field on value timing that stays out of the fields list", async () => {
    await renderRead();
    await addEndpoint();
    fireEvent.click(screen.getByRole("button", { name: "Response value" }));
    await screen.findByText("Pointer");
    expect(screen.getByText("Format")).toBeTruthy();
    expect(screen.queryByText("New field")).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Poll time" }));
    await waitFor(() => expect(screen.queryByText("Format")).toBeNull());
  });

  it("should add a field, select it, and show the enum mapping editor", async () => {
    await renderRead();
    await addEndpoint();
    addField();
    await screen.findByPlaceholderText("/temperature");
    await screen.findByText("Enum mapping");
  });

  it("should copy the previous field's settings when adding another field", async () => {
    await renderRead();
    await addEndpoint();
    addField();
    const pointer = await screen.findByPlaceholderText("/temperature");
    fireEvent.change(pointer, { target: { value: "/a" } });
    fireEvent.blur(pointer);
    addField();
    // Both entries carry the copied pointer. The details header repeats it, so the
    // assertion reads the tree alone.
    await waitFor(() =>
      expect(within(screen.getByRole("tree")).getAllByText("/a")).toHaveLength(2),
    );
  });

  describe("folding an endpoint from its caret", () => {
    const fold = (): void => {
      const endpoint = screen.getByRole("treeitem", { name: /New endpoint/ });
      const caret = endpoint.querySelector(".lyra-tree__expansion-indicator");
      if (caret == null) throw new Error("no caret");
      fireEvent.click(caret);
    };

    it("should hide the fields and keep a selected endpoint", async () => {
      await renderRead();
      await addEndpoint();
      addField();
      await screen.findByPlaceholderText("/temperature");
      fireEvent.click(screen.getByRole("treeitem", { name: /New endpoint/ }));
      await screen.findByPlaceholderText("/api/data");
      fold();
      await waitFor(() =>
        expect(screen.queryByRole("treeitem", { name: /New field/ })).toBeNull(),
      );
      expect(screen.getByPlaceholderText("/api/data")).toBeTruthy();
    });

    it("should move the selection from a hidden field up to its endpoint", async () => {
      await renderRead();
      await addEndpoint();
      addField();
      await screen.findByPlaceholderText("/temperature");
      fold();
      await waitFor(() =>
        expect(screen.queryByRole("treeitem", { name: /New field/ })).toBeNull(),
      );
      await screen.findByPlaceholderText("/api/data");
      expect(screen.queryByPlaceholderText("/temperature")).toBeNull();
    });
  });

  it("should disable and enable a field through its checkbox and context menu", async () => {
    await renderRead();
    await addEndpoint();
    addField();
    const field = await screen.findByRole("treeitem", { name: /New field/ });
    const checkbox = within(field).getByRole("checkbox", { name: "Enabled" });
    fireEvent.click(checkbox);
    await waitFor(() => expect(checkbox).toHaveProperty("checked", false));
    fireEvent.contextMenu(field);
    fireEvent.click(await screen.findByText("Enable"));
    await waitFor(() => expect(checkbox).toHaveProperty("checked", true));
    fireEvent.contextMenu(field);
    fireEvent.click(await screen.findByText("Disable"));
    await waitFor(() => expect(checkbox).toHaveProperty("checked", false));
  });

  it("should add a field from the endpoint context menu", async () => {
    await renderRead();
    await addEndpoint();
    fireEvent.contextMenu(screen.getByRole("treeitem", { name: /New endpoint/ }));
    fireEvent.click(await screen.findByRole("menuitem", { name: "Add field" }));
    await screen.findByRole("treeitem", { name: /New field/ });
    expect(screen.getByPlaceholderText("/temperature")).toBeTruthy();
  });

  it("should select the entry below a removed field", async () => {
    await renderRead();
    await addEndpoint();
    addField();
    commitFieldInput(await screen.findByPlaceholderText("/temperature"), "/a");
    addField();
    await waitFor(() =>
      expect(within(screen.getByRole("tree")).getAllByText("/a")).toHaveLength(2),
    );
    commitFieldInput(screen.getByPlaceholderText("/temperature"), "/b");
    const tree = within(screen.getByRole("tree"));
    fireEvent.click(await tree.findByText("/a"));
    await waitFor(() =>
      expect(screen.getByPlaceholderText("/temperature")).toHaveProperty("value", "/a"),
    );
    fireEvent.contextMenu(tree.getByRole("treeitem", { name: /\/a/ }));
    fireEvent.click(await screen.findByText("Remove"));
    await waitFor(() =>
      expect(screen.getByPlaceholderText("/temperature")).toHaveProperty("value", "/b"),
    );
  });

  it("should duplicate and delete endpoints through the context menu", async () => {
    await renderRead();
    await addEndpoint();
    const path = screen.getByPlaceholderText("/api/data");
    fireEvent.change(path, { target: { value: "/api/v1" } });
    fireEvent.blur(path);
    const item = await screen.findByRole("treeitem", { name: /\/api\/v1/ });
    fireEvent.contextMenu(item);
    fireEvent.click(await screen.findByText("Duplicate"));
    await waitFor(() =>
      expect(screen.getAllByRole("treeitem", { name: /\/api\/v1/ })).toHaveLength(2),
    );
    fireEvent.contextMenu(screen.getAllByRole("treeitem", { name: /\/api\/v1/ })[0]);
    fireEvent.click(await screen.findByText("Remove"));
    await waitFor(() =>
      expect(screen.getAllByRole("treeitem", { name: /\/api\/v1/ })).toHaveLength(1),
    );
  });

  it("should offer Reload Console from the endpoint context menu", async () => {
    await renderRead();
    await addEndpoint();
    const path = screen.getByPlaceholderText("/api/data");
    fireEvent.change(path, { target: { value: "/api/v1" } });
    fireEvent.blur(path);
    fireEvent.contextMenu(await screen.findByRole("treeitem", { name: /\/api\/v1/ }));
    expect(await screen.findByText("Reload Console")).toBeTruthy();
  });

  it("should seed the form from the task row's config", async () => {
    const client = createTestClient();
    const dev = await createHTTPDevice(client);
    const config = createReadConfig(dev.key, [
      {
        ...http.readEndpointZ.parse({}),
        key: "ep1",
        path: "/seeded",
        fields: [createReadField("f1", "/temp")],
      },
    ]);
    const draft = await createDraft(client, config);
    await renderRead({ client, taskKey: draft.key });
    await screen.findByRole("treeitem", { name: /\/seeded/ });
  });

  describe("fill value", () => {
    const renderField = async (overrides: Record<string, unknown>) => {
      const dev = await createHTTPDevice(client);
      const draft = await createDraft(
        client,
        createReadConfig(dev.key, [
          {
            ...http.readEndpointZ.parse({}),
            key: "ep1",
            path: "/data",
            fields: [createReadField("f1", "/value", overrides)],
          },
        ]),
      );
      const { container } = await renderRead({ client, taskKey: draft.key });
      fireEvent.click(await screen.findByRole("treeitem", { name: /\/value/ }));
      const deploy = async (): Promise<HTTP.Task.ReadField> => {
        const created = await deployAndAwaitTask(
          client,
          container,
          draft.key,
          HTTP.Task.READ_SCHEMAS,
        );
        return created.config.endpoints[0].fields[0];
      };
      return { deploy };
    };

    const findFillValue = async (): Promise<HTMLInputElement> =>
      await screen.findByRole<HTMLInputElement>("textbox", { name: "Fill value" });

    const checkOptional = async (): Promise<HTMLInputElement> => {
      fireEvent.click(await waitFor(() => getLabeledInput("Optional")));
      return await findFillValue();
    };

    const fillOf = (field: HTTP.Task.ReadField): unknown =>
      "fill" in field ? field.fill : undefined;

    it("should show the input only when the field is optional", async () => {
      await renderField({});
      const optional = await waitFor(() => getLabeledInput("Optional"));
      expect(screen.queryByRole("textbox", { name: "Fill value" })).toBeNull();
      fireEvent.click(optional);
      await findFillValue();
      fireEvent.click(optional);
      await waitFor(() =>
        expect(screen.queryByRole("textbox", { name: "Fill value" })).toBeNull(),
      );
    });

    it("should clear the optional box when a kind change drops the fill", async () => {
      await renderField({ fill: "NaN" });
      const optional = await waitFor(() => getLabeledInput("Optional"));
      expect(optional.checked).toBe(true);
      await selectFromDropdown("float64", "int32");
      await selectFromDropdown("int32", "float64");
      await waitFor(() => expect(getLabeledInput("Optional").checked).toBe(false));
    });

    it("should store a NaN fill when a float field turns optional", async () => {
      const { deploy } = await renderField({});
      expect((await checkOptional()).placeholder).toBe("NaN");
      expect(fillOf(await deploy())).toBeNaN();
    });

    it("should store a numeric fill value of 0", async () => {
      const { deploy } = await renderField({});
      commitFieldInput(await checkOptional(), "0");
      expect(fillOf(await deploy())).toBe(0);
    });

    it("should store a non-finite float fill", async () => {
      const { deploy } = await renderField({});
      commitFieldInput(await checkOptional(), "-Infinity");
      expect(fillOf(await deploy())).toBe(-Infinity);
    });

    it("should store text on a string field", async () => {
      const { deploy } = await renderField({ type: "string" });
      const input = await checkOptional();
      expect(input.placeholder).toBe("Required");
      commitFieldInput(input, "off");
      expect(fillOf(await deploy())).toBe("off");
    });

    it("should store digits as text on a string field", async () => {
      const { deploy } = await renderField({ type: "string" });
      commitFieldInput(await checkOptional(), "5");
      expect(fillOf(await deploy())).toBe("5");
    });

    it("should store an integer fill as decimal text", async () => {
      const { deploy } = await renderField({ type: "integer", dataType: "int64" });
      commitFieldInput(await checkOptional(), "-9223372036854775808");
      expect(fillOf(await deploy())).toBe("-9223372036854775808");
    });

    it("should store a boolean fill", async () => {
      const { deploy } = await renderField({ type: "boolean" });
      commitFieldInput(await checkOptional(), "true");
      expect(fillOf(await deploy())).toBe(true);
    });

    it.each([
      { type: "float", fill: 7, text: "off" },
      { type: "integer", fill: "7", text: "1.5" },
      { type: "integer", fill: "7", text: "9223372036854775808" },
      { type: "unsigned", fill: "7", text: "-1" },
      { type: "boolean", fill: false, text: "1" },
    ])(
      "should show an error and keep the fill for $text on a $type field",
      async ({ type, fill, text }) => {
        const { deploy } = await renderField({ type, fill });
        const input = await findFillValue();
        commitFieldInput(input, text);
        await screen.findByText(`Not a valid ${type} value`);
        expect(input.value).toBe(text);
        expect(fillOf(await deploy())).toBe(fill);
      },
    );

    it("should hold a partial number until the input commits", async () => {
      const { deploy } = await renderField({});
      const input = await checkOptional();
      fireEvent.change(input, { target: { value: "1." } });
      expect(input.value).toBe("1.");
      commitFieldInput(input, "1.5");
      expect(fillOf(await deploy())).toBe(1.5);
    });

    it.each([
      { stored: "off", type: "string", text: "off" },
      { stored: false, type: "boolean", text: "false" },
      { stored: "5", type: "integer", text: "5" },
      { stored: "5", type: "unsigned", text: "5" },
    ])(
      "should show and clear a stored $stored on a $type field",
      async ({ stored, type, text }) => {
        const { deploy } = await renderField({ type, fill: stored });
        const input = await findFillValue();
        await waitFor(() => expect(input.value).toBe(text));
        commitFieldInput(input, "");
        await waitFor(() => expect(input.value).toBe(""));
        expect(fillOf(await deploy())).toBeUndefined();
      },
    );

    it("should write NaN when the float input is emptied", async () => {
      const { deploy } = await renderField({ fill: 5 });
      const input = await findFillValue();
      await waitFor(() => expect(input.value).toBe("5"));
      commitFieldInput(input, "");
      await waitFor(() => expect(input.value).toBe(""));
      expect(fillOf(await deploy())).toBeNaN();
    });

    it("should keep the stored type when the input blurs with no edit", async () => {
      const { deploy } = await renderField({ type: "boolean", fill: false });
      const input = await findFillValue();
      await waitFor(() => expect(input.value).toBe("false"));
      fireEvent.focus(input);
      fireEvent.blur(input);
      expect(fillOf(await deploy())).toBe(false);
    });

    it("should unset the fill value when Optional turns off", async () => {
      const { deploy } = await renderField({ fill: 5 });
      await findFillValue();
      fireEvent.click(getLabeledInput("Optional"));
      await waitFor(() =>
        expect(screen.queryByRole("textbox", { name: "Fill value" })).toBeNull(),
      );
      expect(fillOf(await deploy())).toBeUndefined();
    });

    describe("bound to a channel", () => {
      const createChannel = async (dataType: DataType) =>
        await client.channels.create({
          name: uniqueName("fill_value"),
          dataType,
          virtual: true,
        });

      it("should take text when the channel is a string channel", async () => {
        const ch = await createChannel(DataType.STRING);
        await renderField({ type: "string", channel: ch.key });
        const input = await checkOptional();
        expect(input.placeholder).toBe("Required");
        commitFieldInput(input, "off");
        await waitFor(() => expect(input).toHaveProperty("value", "off"));
      });

      it("should show NaN when the channel is a float channel", async () => {
        const ch = await createChannel(DataType.FLOAT32);
        await renderField({ channel: ch.key });
        expect((await checkOptional()).placeholder).toBe("NaN");
      });

      it("should require a fill value when the channel is not a float", async () => {
        const ch = await createChannel(DataType.INT32);
        await renderField({ type: "integer", channel: ch.key });
        expect((await checkOptional()).placeholder).toBe("Required");
      });
    });
  });

  it("should toggle the NaN fill switch", async () => {
    await renderRead();
    const toggle = getLabeledInput("NaN fill");
    expect(toggle.checked).toBe(false);
    fireEvent.click(toggle);
    expect(toggle.checked).toBe(true);
  });

  const client = createTestClient();

  it("should create index, data, and virtual channels and persist them to the device", async () => {
    const dev = await createHTTPDevice(client);
    const config = createReadConfig(dev.key, [
      {
        ...http.readEndpointZ.parse({}),
        key: "ep1",
        path: "/data",
        index: "tf",
        fields: [
          createReadField("f1", "/temperature"),
          createReadField("f2", "/label", { type: "string" }),
          createReadField("tf", "/ts", { type: "timestamp", timeFormat: "unix_sec" }),
        ],
      },
    ]);
    const draft = await createDraft(client, config);
    const { container } = await renderRead({ client, taskKey: draft.key });
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      HTTP.Task.READ_SCHEMAS,
    );

    const updated = await client.devices.retrieve({
      key: dev.key,
      schemas: HTTP.Device.SCHEMAS,
    });
    const epProps = updated.properties.read["/data"];
    expect(epProps.index).toBeGreaterThan(0);

    const indexCh = await client.channels.retrieve(epProps.index);
    expect(indexCh.name).toBe(`${dev.name}_data_time`);

    const dataKey = epProps.channels["/temperature"];
    const dataCh = await client.channels.retrieve(dataKey);
    expect(dataCh.index).toBe(epProps.index);
    expect(dataCh.name).toBe(`${dev.name}_data_temperature`);

    const virtualKey = epProps.channels["/label"];
    const virtualCh = await client.channels.retrieve(virtualKey);
    expect(virtualCh.virtual).toBe(true);
    expect(virtualCh.name).toBe(`${dev.name}_data_label`);

    const fields = created.config.endpoints[0].fields;
    expect(fields.find((f) => f.key === "f1")?.channel).toBe(dataKey);
    expect(fields.find((f) => f.key === "f2")?.channel).toBe(virtualKey);
    expect(fields.find((f) => f.key === "tf")?.channel).toBe(epProps.index);
  });

  it("should save the channels it created when a later channel fails", async () => {
    const dev = await createHTTPDevice(client);
    const config = createReadConfig(dev.key, [
      {
        ...http.readEndpointZ.parse({}),
        key: "ep1",
        path: "/data",
        fields: [
          createReadField("f1", "/temperature"),
          createReadField("f2", "/pressure", { name: "bad name" }),
        ],
      },
    ]);
    const draft = await createDraft(client, config);
    const { container } = await renderRead({ client, taskKey: draft.key });
    await clickDeploy(container);
    await waitFor(async () => {
      const updated = await client.devices.retrieve({
        key: dev.key,
        schemas: HTTP.Device.SCHEMAS,
      });
      const epProps = updated.properties.read["/data"];
      expect(epProps.index).toBeGreaterThan(0);
      expect(Object.keys(epProps.channels)).toEqual(["/temperature"]);
    });
  });

  it("should rename a field's existing channel from the details pane", async () => {
    const dev = await createHTTPDevice(client);
    const ch = await client.channels.create({
      name: uniqueName("http_read"),
      dataType: "float64",
      virtual: true,
    });
    const config = createReadConfig(dev.key, [
      {
        ...http.readEndpointZ.parse({}),
        key: "ep1",
        path: "/data",
        fields: [createReadField("f1", "/temperature", { channel: ch.key })],
      },
    ]);
    const draft = await createDraft(client, config);
    await renderRead({ client, taskKey: draft.key });
    fireEvent.click(await screen.findByRole("treeitem", { name: /\/temperature/ }));
    const field = await screen.findByRole<HTMLInputElement>("textbox", {
      name: "Channel",
    });
    await waitFor(() => expect(field.value).toBe(ch.name));
    const renamed = uniqueName("http_renamed");
    fireEvent.change(field, { target: { value: renamed } });
    fireEvent.blur(field);
    await waitFor(async () =>
      expect((await client.channels.retrieve(ch.key)).name).toBe(renamed),
    );
  });

  it("should keep a duplicated endpoint's timestamp bound to the index", async () => {
    const dev = await createHTTPDevice(client);
    const config = createReadConfig(dev.key, [
      {
        ...http.readEndpointZ.parse({}),
        key: "ep1",
        path: "/data",
        index: "tf",
        fields: [
          createReadField("f1", "/temperature"),
          createReadField("tf", "/ts", { type: "timestamp", timeFormat: "unix_sec" }),
        ],
      },
    ]);
    const draft = await createDraft(client, config);
    const { container } = await renderRead({ client, taskKey: draft.key });
    fireEvent.contextMenu(await screen.findByRole("treeitem", { name: /\/data/ }));
    fireEvent.click(await screen.findByText("Duplicate"));
    await waitFor(() =>
      expect(screen.getAllByRole("treeitem", { name: /\/data/ })).toHaveLength(2),
    );
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      HTTP.Task.READ_SCHEMAS,
    );
    const updated = await client.devices.retrieve({
      key: dev.key,
      schemas: HTTP.Device.SCHEMAS,
    });
    const copy = created.config.endpoints[1];
    const timing = copy.fields.find((f) => f.type === "timestamp");
    expect(copy.index).toBe(timing?.key);
    expect(timing?.channel).toBe(updated.properties.read["/data"].index);
  });

  it("should bind a new field to the channel the device already stores", async () => {
    const dev = await createHTTPDevice(client);
    const idxCh = await client.channels.create({
      name: uniqueName("http_idx"),
      dataType: "timestamp",
      isIndex: true,
    });
    const dataCh = await client.channels.create({
      name: uniqueName("http_data"),
      dataType: "float64",
      index: idxCh.key,
    });
    const properties = {
      ...HTTP.Device.ZERO_PROPERTIES,
      read: {
        "/data": { index: idxCh.key, channels: { "/temperature": dataCh.key } },
      },
    };
    await Device.setProperties(
      client,
      { key: dev.key, properties },
      HTTP.Device.ZERO_PROPERTIES,
      HTTP.Device.SCHEMAS,
    );
    const draft = await createDraft(
      client,
      createReadConfig(dev.key, [
        {
          ...http.readEndpointZ.parse({}),
          key: "ep1",
          path: "/data",
          fields: [createReadField("f1", "/temperature")],
        },
      ]),
    );
    await renderRead({ client, taskKey: draft.key });
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: draft.key,
        schemas: HTTP.Task.READ_SCHEMAS,
      });
      expect(saved.config.endpoints[0].fields[0].channel).toBe(dataCh.key);
    });
  });

  it("should give a field the kind of the channel the device binds", async () => {
    const labelCh = await client.channels.create({
      name: uniqueName("http_label"),
      dataType: "string",
      virtual: true,
    });
    const dev = await createHTTPDevice(client, {
      properties: {
        read: { "/data": { index: 0, channels: { "/label": labelCh.key } } },
      },
    });
    const draft = await createDraft(
      client,
      createReadConfig(dev.key, [
        {
          ...http.readEndpointZ.parse({}),
          key: "ep1",
          path: "/data",
          fields: [createReadField("f1", "/label")],
        },
      ]),
    );
    await renderRead({ client, taskKey: draft.key });
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: draft.key,
        schemas: HTTP.Task.READ_SCHEMAS,
      });
      expect(saved.config.endpoints[0].fields[0]).toMatchObject({
        channel: labelCh.key,
        type: "string",
      });
    });
  });

  it("should deploy a field with the kind of the channel the device binds", async () => {
    const labelCh = await client.channels.create({
      name: uniqueName("http_label"),
      dataType: "string",
      virtual: true,
    });
    const dev = await createHTTPDevice(client, {
      properties: {
        read: { "/data": { index: 0, channels: { "/label": labelCh.key } } },
      },
    });
    const draft = await createDraft(
      client,
      createReadConfig(dev.key, [
        {
          ...http.readEndpointZ.parse({}),
          key: "ep1",
          path: "/data",
          fields: [createReadField("f1", "/label")],
        },
      ]),
    );
    const { container } = await renderRead({ client, taskKey: draft.key });
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      HTTP.Task.READ_SCHEMAS,
    );
    expect(created.config.endpoints[0].fields[0]).toMatchObject({
      channel: labelCh.key,
      type: "string",
    });
  });

  it.each(["float", "integer", "unsigned", "string", "boolean"])(
    "should save each added enum mapping on a %s field under its own label",
    async (type) => {
      const dev = await createHTTPDevice(client);
      const draft = await createDraft(
        client,
        createReadConfig(dev.key, [
          {
            ...http.readEndpointZ.parse({}),
            key: "ep1",
            path: "/data",
            fields: [createReadField("f1", "/value", { type })],
          },
        ]),
      );
      await renderRead({ client, taskKey: draft.key });
      fireEvent.click(await screen.findByRole("treeitem", { name: /\/value/ }));
      const add = await screen.findByRole("button", { name: "Add mapping" });
      fireEvent.click(add);
      fireEvent.click(add);
      await waitFor(async () => {
        const stored = await client.tasks.retrieve({
          key: draft.key,
          schemas: HTTP.Task.READ_SCHEMAS,
        });
        const field = stored.config.endpoints[0].fields[0];
        expect(
          "enumValues" in field && field.enumValues.map(({ label }) => label),
        ).toEqual(["LABEL_1", "LABEL_2"]);
      });
    },
  );

  it("should lock the data type of a field bound to a channel", async () => {
    const dev = await createHTTPDevice(client);
    const idxCh = await client.channels.create({
      name: uniqueName("http_idx"),
      dataType: "timestamp",
      isIndex: true,
    });
    const dataCh = await client.channels.create({
      name: uniqueName("http_data"),
      dataType: "float64",
      index: idxCh.key,
    });
    const properties = {
      ...HTTP.Device.ZERO_PROPERTIES,
      read: {
        "/data": { index: idxCh.key, channels: { "/temperature": dataCh.key } },
      },
    };
    await Device.setProperties(
      client,
      { key: dev.key, properties },
      HTTP.Device.ZERO_PROPERTIES,
      HTTP.Device.SCHEMAS,
    );
    const draft = await createDraft(
      client,
      createReadConfig(dev.key, [
        {
          ...http.readEndpointZ.parse({}),
          key: "ep1",
          path: "/data",
          fields: [createReadField("f1", "/temperature")],
        },
      ]),
    );
    await renderRead({ client, taskKey: draft.key });
    fireEvent.click(await screen.findByRole("treeitem", { name: /\/temperature/ }));
    await screen.findByText("Set on the channel");
  });

  it("should bind the fields of every endpoint on open, not only the selected one", async () => {
    const dev = await createHTTPDevice(client);
    const idxCh = await client.channels.create({
      name: uniqueName("http_idx"),
      dataType: "timestamp",
      isIndex: true,
    });
    const [firstCh, secondCh] = await Promise.all(
      ["http_first", "http_second"].map(
        async (prefix) =>
          await client.channels.create({
            name: uniqueName(prefix),
            dataType: "float64",
            index: idxCh.key,
          }),
      ),
    );
    const properties = {
      ...HTTP.Device.ZERO_PROPERTIES,
      read: {
        "/first": { index: idxCh.key, channels: { "/temperature": firstCh.key } },
        "/second": { index: idxCh.key, channels: { "/temperature": secondCh.key } },
      },
    };
    await Device.setProperties(
      client,
      { key: dev.key, properties },
      HTTP.Device.ZERO_PROPERTIES,
      HTTP.Device.SCHEMAS,
    );
    const endpoint = (key: string, path: string) => ({
      ...http.readEndpointZ.parse({}),
      key,
      path,
      fields: [createReadField(`${key}_f`, "/temperature")],
    });
    const draft = await createDraft(
      client,
      createReadConfig(dev.key, [
        endpoint("ep1", "/first"),
        endpoint("ep2", "/second"),
      ]),
    );
    await renderRead({ client, taskKey: draft.key });
    // The form opens on the first endpoint, so the second one's field shows whether
    // binding reached an endpoint nobody selected.
    const fields = await screen.findAllByRole("treeitem", { name: /\/temperature/ });
    fireEvent.click(fields[1]);
    await screen.findByDisplayValue(secondCh.name);
    await waitFor(async () => {
      const saved = await client.tasks.retrieve({
        key: draft.key,
        schemas: HTTP.Task.READ_SCHEMAS,
      });
      expect(saved.config.endpoints.map((ep) => ep.fields[0].channel)).toEqual([
        firstCh.key,
        secondCh.key,
      ]);
    });
  });

  it("should follow a path edit to the channel the new path maps, or to none", async () => {
    const dev = await createHTTPDevice(client);
    const idxCh = await client.channels.create({
      name: uniqueName("http_idx"),
      dataType: "timestamp",
      isIndex: true,
    });
    const [dataCh, otherCh] = await Promise.all(
      ["http_data", "http_other"].map(
        async (prefix) =>
          await client.channels.create({
            name: uniqueName(prefix),
            dataType: "float64",
            index: idxCh.key,
          }),
      ),
    );
    const properties = {
      ...HTTP.Device.ZERO_PROPERTIES,
      read: {
        "/data": { index: idxCh.key, channels: { "/temperature": dataCh.key } },
        "/other": { index: idxCh.key, channels: { "/temperature": otherCh.key } },
      },
    };
    await Device.setProperties(
      client,
      { key: dev.key, properties },
      HTTP.Device.ZERO_PROPERTIES,
      HTTP.Device.SCHEMAS,
    );
    const draft = await createDraft(
      client,
      createReadConfig(dev.key, [
        {
          ...http.readEndpointZ.parse({}),
          key: "ep1",
          path: "/data",
          fields: [createReadField("f1", "/temperature")],
        },
      ]),
    );
    await renderRead({ client, taskKey: draft.key });
    const selectField = () =>
      fireEvent.click(screen.getByRole("treeitem", { name: /\/temperature/ }));
    const selectEndpoint = (path: RegExp) =>
      fireEvent.click(screen.getByRole("treeitem", { name: path }));
    fireEvent.click(await screen.findByRole("treeitem", { name: /\/temperature/ }));
    await screen.findByDisplayValue(dataCh.name);
    const savedChannel = async () =>
      (
        await client.tasks.retrieve({
          key: draft.key,
          schemas: HTTP.Task.READ_SCHEMAS,
        })
      ).config.endpoints[0].fields[0].channel;
    selectEndpoint(/\/data/);
    commitFieldInput(await screen.findByDisplayValue("/data"), "/other");
    await waitFor(async () => expect(await savedChannel()).toBe(otherCh.key));
    selectField();
    await screen.findByDisplayValue(otherCh.name);
    selectEndpoint(/\/other/);
    commitFieldInput(await screen.findByDisplayValue("/other"), "/none");
    await waitFor(async () => expect(await savedChannel()).toBe(0));
    selectField();
    await screen.findByPlaceholderText("/temperature");
    expect(screen.queryByDisplayValue(otherCh.name)).toBeNull();
  });

  it("should reuse channels already stored on the device instead of creating new ones", async () => {
    const dev = await createHTTPDevice(client);
    const idxCh = await client.channels.create({
      name: uniqueName("http_idx"),
      dataType: "timestamp",
      isIndex: true,
    });
    const dataCh = await client.channels.create({
      name: uniqueName("http_data"),
      dataType: "float64",
      index: idxCh.key,
    });
    const properties = {
      ...HTTP.Device.ZERO_PROPERTIES,
      read: {
        "/data": { index: idxCh.key, channels: { "/temperature": dataCh.key } },
      },
    };
    await Device.setProperties(
      client,
      { key: dev.key, properties },
      HTTP.Device.ZERO_PROPERTIES,
      HTTP.Device.SCHEMAS,
    );
    const config = createReadConfig(dev.key, [
      {
        ...http.readEndpointZ.parse({}),
        key: "ep1",
        path: "/data",
        fields: [createReadField("f1", "/temperature")],
      },
    ]);
    const draft = await createDraft(client, config);
    const { container } = await renderRead({ client, taskKey: draft.key });
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      HTTP.Task.READ_SCHEMAS,
    );
    expect(created.config.endpoints[0].fields[0].channel).toBe(dataCh.key);
  });

  it("should drop a request body left over from POST when the method switches to GET", async () => {
    const dev = await createHTTPDevice(client);
    const config = createReadConfig(dev.key, [
      {
        ...http.readEndpointZ.parse({}),
        key: "ep1",
        path: "/switched",
        method: "POST",
        body: '{"query": "latest"}',
        fields: [createReadField("f1", "/temperature")],
      },
      {
        ...http.readEndpointZ.parse({}),
        key: "ep2",
        path: "/kept",
        method: "POST",
        body: '{"query": "kept"}',
        fields: [createReadField("f2", "/pressure")],
      },
    ]);
    const draft = await createDraft(client, config);
    const { container } = await renderRead({ client, taskKey: draft.key });
    fireEvent.click(await screen.findByRole("treeitem", { name: /\/switched/ }));
    await screen.findByDisplayValue('{"query": "latest"}');
    fireEvent.click(screen.getByRole("button", { name: "GET" }));
    await waitFor(() =>
      expect(screen.queryByDisplayValue('{"query": "latest"}')).toBeNull(),
    );
    const created = await deployAndAwaitTask(
      client,
      container,
      draft.key,
      HTTP.Task.READ_SCHEMAS,
    );
    const endpoints = created.config.endpoints;
    expect(endpoints.find((e) => e.key === "ep1")?.body).toBe("");
    expect(endpoints.find((e) => e.key === "ep2")?.body).toBe('{"query": "kept"}');
  });

  it("should recover the index from an existing data channel when the stored index is gone", async () => {
    const dev = await createHTTPDevice(client);
    const idxCh = await client.channels.create({
      name: uniqueName("http_idx"),
      dataType: "timestamp",
      isIndex: true,
    });
    const dataCh = await client.channels.create({
      name: uniqueName("http_data"),
      dataType: "float64",
      index: idxCh.key,
    });
    const properties = {
      ...HTTP.Device.ZERO_PROPERTIES,
      read: { "/data": { index: 0, channels: { "/existing": dataCh.key } } },
    };
    await Device.setProperties(
      client,
      { key: dev.key, properties },
      HTTP.Device.ZERO_PROPERTIES,
      HTTP.Device.SCHEMAS,
    );
    const config = createReadConfig(dev.key, [
      {
        ...http.readEndpointZ.parse({}),
        key: "ep1",
        path: "/data",
        fields: [createReadField("f1", "/temperature")],
      },
    ]);
    const draft = await createDraft(client, config);
    const { container } = await renderRead({ client, taskKey: draft.key });
    await deployAndAwaitTask(client, container, draft.key, HTTP.Task.READ_SCHEMAS);
    const updated = await client.devices.retrieve({
      key: dev.key,
      schemas: HTTP.Device.SCHEMAS,
    });
    expect(updated.properties.read["/data"].index).toBe(idxCh.key);
  });
});
