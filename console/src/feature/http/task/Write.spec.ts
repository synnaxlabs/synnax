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
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { HTTP } from "@/feature/http";
import { createHTTPDevice } from "@/feature/http/testutil";
import {
  createChannelReadOnlyClient,
  deployAndAwaitTask,
  findDialogTriggerByText,
  renderTaskFormTab,
  type RenderTaskFormTabOptions,
  selectFromDropdown,
} from "@/platform/task/testutil";
import {
  awaitTextEditingElement,
  commitTextEdit,
  getHeaderIconButton,
  uniqueName,
} from "@/testutil";

const renderWrite = async (options: RenderTaskFormTabOptions = {}) =>
  await renderTaskFormTab(HTTP.Task.Write, { task: ZERO_DRAFT, ...options });

// Drafts carry no key; the created row mints its own.
const ZERO_DRAFT: task.New<HTTP.Task.WriteSchemas> = {
  name: "HTTP write task",
  type: HTTP.Task.WRITE_TYPE,
  config: HTTP.Task.WRITE_SCHEMAS.config.parse({}),
};

const createDraft = async (client: Synnax, config: HTTP.Task.WritePayload["config"]) =>
  await client.tasks.create({ ...ZERO_DRAFT, config }, HTTP.Task.WRITE_SCHEMAS);

const addEndpoint = async (): Promise<void> => {
  fireEvent.click(await screen.findByText("Add endpoint"));
  await screen.findByText("JSON pointer");
};

const createWriteEndpoint = (
  key: string,
  path: string,
  channel: Partial<HTTP.Task.ChannelField> = {},
): HTTP.Task.WriteEndpoint => ({
  ...HTTP.Task.writeEndpointZ.parse({}),
  key,
  path,
  channel: { ...http.channelFieldZ.parse({}), pointer: "/value", ...channel },
});

const createWriteConfig = (
  device: string,
  endpoints: HTTP.Task.WriteEndpoint[],
): HTTP.Task.WritePayload["config"] => ({
  ...HTTP.Task.WRITE_SCHEMAS.config.parse({}),
  device,
  endpoints,
});

describe("HTTP Write form", () => {
  it("should show the empty state and add + select an endpoint", async () => {
    await renderWrite();
    await screen.findByText("Select an endpoint to configure");
    await addEndpoint();
    expect(screen.getByRole("button", { name: "POST" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "PUT" })).toBeTruthy();
    expect(screen.getByRole("button", { name: "PATCH" })).toBeTruthy();
    expect(screen.getByPlaceholderText("/api/control")).toBeTruthy();
    expect(screen.getByText("Synnax data type")).toBeTruthy();
    expect(screen.getByText("No additional fields")).toBeTruthy();
  });

  it("should show the enum mapping editor when the channel JSON type is string", async () => {
    await renderWrite();
    await addEndpoint();
    expect(screen.queryByText("Enum mappings")).toBeNull();
    await selectFromDropdown("Number", "String");
    await screen.findByText("Enum mappings");
  });

  it("should add a static field and reset its value when the JSON type changes", async () => {
    await renderWrite();
    await addEndpoint();
    fireEvent.click(getHeaderIconButton("Additional fields", "add"));
    await screen.findByText("static");
    const value = screen.getByPlaceholderText<HTMLInputElement>("value");
    fireEvent.change(value, { target: { value: "on" } });
    fireEvent.blur(value);
    await screen.findByDisplayValue("on");
    await selectFromDropdown("String", "Number");
    await waitFor(() => expect(screen.queryByDisplayValue("on")).toBeNull());
  });

  it("should add a generated field and switch its generator to a timestamp", async () => {
    await renderWrite();
    await addEndpoint();
    fireEvent.click(getHeaderIconButton("Additional fields", "time"));
    await screen.findByText("generated");
    await selectFromDropdown("UUID", "Timestamp (s)");
    await findDialogTriggerByText("Timestamp (s)");
  });

  it("should remove additional fields through a context menu without a duplicate option", async () => {
    await renderWrite();
    await addEndpoint();
    fireEvent.click(getHeaderIconButton("Additional fields", "add"));
    const pointer = await screen.findByPlaceholderText("field");
    fireEvent.contextMenu(pointer);
    await screen.findByText("Remove");
    expect(screen.queryByText("Duplicate")).toBeNull();
    fireEvent.click(screen.getByText("Remove"));
    await waitFor(() => expect(screen.queryByText("static")).toBeNull());
  });

  it("should duplicate endpoints and rename the command channel via the context menu", async () => {
    await renderWrite();
    await addEndpoint();
    const item = await screen.findByText("No channel");
    fireEvent.contextMenu(item);
    fireEvent.click(await screen.findByText("Duplicate"));
    await waitFor(() => expect(screen.getAllByText("No channel")).toHaveLength(2));
    fireEvent.contextMenu(screen.getAllByText("No channel")[0]);
    fireEvent.click(await screen.findByText("Rename"));
    const editable = await awaitTextEditingElement();
    commitTextEdit(editable, "my_cmd_channel");
    await screen.findByText("my_cmd_channel");
  });

  it("should withhold rename from a subject who cannot update channels", async () => {
    const client = createTestClient();
    await renderWrite({ client, as: await createChannelReadOnlyClient(client) });
    await addEndpoint();
    fireEvent.contextMenu(await screen.findByText("No channel"));
    // Duplicate is ungated, so its presence proves the menu resolved before the
    // absence below is read.
    expect(await screen.findByText("Duplicate")).toBeTruthy();
    expect(screen.queryByText("Rename")).toBeNull();
  });

  it("should seed the form from the task row's config", async () => {
    const client = createTestClient();
    const config = createWriteConfig("dev_1", [createWriteEndpoint("ep1", "/seeded")]);
    const draft = await createDraft(client, config);
    await renderWrite({ client, taskKey: draft.key });
    await screen.findByText(/\/seeded/);
  });

  // Waits on the channel name so the negative case asserts against a settled lookup
  // rather than one still in flight.
  const renderWithCommandChannel = async (dataType: string) => {
    const client = createTestClient();
    const ch = await client.channels.create({
      name: uniqueName("http_cmd"),
      dataType,
      virtual: true,
    });
    const config = createWriteConfig("dev_1", [
      createWriteEndpoint("ep1", "/cmd", { channel: ch.key }),
    ]);
    const draft = await createDraft(client, config);
    await renderWrite({ client, taskKey: draft.key });
    await screen.findByText("JSON pointer");
    await screen.findByText(ch.name);
  };

  it("should show the time format field for a timestamp command channel", async () => {
    await renderWithCommandChannel("timestamp");
    await screen.findByText("Time format");
  });

  it("should hide the time format field for a numeric command channel", async () => {
    await renderWithCommandChannel("float64");
    expect(screen.queryByText("Time format")).toBeNull();
  });

  describe("deploying against a live cluster", () => {
    const client = createTestClient();

    it("should create command channels, virtual when variable, and persist them to the device", async () => {
      const dev = await createHTTPDevice(client);
      const virtualName = uniqueName("http_cmd");
      const config = createWriteConfig(dev.key, [
        createWriteEndpoint("ep1", "/cmd", { dataType: "uint8" }),
        createWriteEndpoint("ep2", "/msg", { dataType: "string", name: virtualName }),
      ]);
      const draft = await createDraft(client, config);
      const { container } = await renderWrite({ client, taskKey: draft.key });
      const created = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        HTTP.Task.WRITE_SCHEMAS,
      );

      const updated = await client.devices.retrieve({
        key: dev.key,
        schemas: HTTP.Device.SCHEMAS,
      });
      const cmdKey = updated.properties.write["/cmd"];
      const cmdCh = await client.channels.retrieve(cmdKey);
      expect(cmdCh.dataType.toString()).toBe("uint8");
      expect(cmdCh.index).toBeGreaterThan(0);
      expect(created.config.endpoints[0].channel.channel).toBe(cmdKey);

      const virtualKey = updated.properties.write["/msg"];
      const virtualCh = await client.channels.retrieve(virtualKey);
      expect(virtualCh.virtual).toBe(true);
      expect(virtualCh.name).toBe(virtualName);
      expect(created.config.endpoints[1].channel.channel).toBe(virtualKey);
    });

    it("should adopt existing channels from the config and the device instead of creating new ones", async () => {
      const dev = await createHTTPDevice(client);
      const configuredCh = await client.channels.create({
        name: uniqueName("http_cmd"),
        dataType: "string",
        virtual: true,
      });
      const storedCh = await client.channels.create({
        name: uniqueName("http_cmd"),
        dataType: "string",
        virtual: true,
      });
      dev.properties = {
        ...HTTP.Device.ZERO_PROPERTIES,
        write: { "/stored": storedCh.key },
      };
      await client.devices.create(dev);
      const config = createWriteConfig(dev.key, [
        createWriteEndpoint("ep1", "/cmd", { channel: configuredCh.key }),
        createWriteEndpoint("ep2", "/stored"),
      ]);
      const draft = await createDraft(client, config);
      const { container } = await renderWrite({ client, taskKey: draft.key });
      const created = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        HTTP.Task.WRITE_SCHEMAS,
      );
      expect(created.config.endpoints[0].channel.channel).toBe(configuredCh.key);
      expect(created.config.endpoints[1].channel.channel).toBe(storedCh.key);
      const updated = await client.devices.retrieve({
        key: dev.key,
        schemas: HTTP.Device.SCHEMAS,
      });
      expect(updated.properties.write["/cmd"]).toBe(configuredCh.key);
      expect(updated.properties.write["/stored"]).toBe(storedCh.key);
    });
  });
});
