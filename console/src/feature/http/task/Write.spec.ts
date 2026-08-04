// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax, task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { HTTP } from "@/feature/http";
import { createHTTPDevice } from "@/feature/http/testutil";
import {
  awaitCommand,
  clickDeploy,
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
  await renderTaskFormTab(HTTP.Task.Write, options);

// Draft creates mint their own key; the zero payload's empty key must not be sent.
const { key: _key, ...ZERO_DRAFT } = HTTP.Task.ZERO_WRITE_PAYLOAD;

const createDraft = async (client: Synnax, config: HTTP.Task.WritePayload["config"]) =>
  await client.tasks.create({ ...ZERO_DRAFT, config }, HTTP.Task.WRITE_SCHEMAS);

const deployAndAwaitTask = async (
  client: Synnax,
  container: ParentNode,
  key: task.Key,
) => {
  const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
  try {
    await clickDeploy(container);
    await awaitCommand(streamer, key);
  } finally {
    streamer.close();
  }
  return await client.tasks.retrieve({ key, schemas: HTTP.Task.WRITE_SCHEMAS });
};

const addEndpoint = async (): Promise<void> => {
  fireEvent.click(await screen.findByText("Add an endpoint"));
  await screen.findByText("JSON pointer");
};

const createWriteEndpoint = (
  key: string,
  path: string,
  channel: Partial<HTTP.Task.ChannelField> = {},
): HTTP.Task.WriteEndpoint => ({
  ...HTTP.Task.ZERO_WRITE_ENDPOINT,
  key,
  path,
  channel: { ...HTTP.Task.ZERO_CHANNEL_FIELD, pointer: "/value", ...channel },
});

const createWriteConfig = (
  device: string,
  endpoints: HTTP.Task.WriteEndpoint[],
): HTTP.Task.WritePayload["config"] => ({
  ...HTTP.Task.ZERO_WRITE_PAYLOAD.config,
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
    expect(screen.getByText("No additional fields.")).toBeTruthy();
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

  it("should delete additional fields through a context menu without a duplicate option", async () => {
    await renderWrite();
    await addEndpoint();
    fireEvent.click(getHeaderIconButton("Additional fields", "add"));
    const pointer = await screen.findByPlaceholderText("field");
    fireEvent.contextMenu(pointer);
    await screen.findByText("Delete");
    expect(screen.queryByText("Duplicate")).toBeNull();
    fireEvent.click(screen.getByText("Delete"));
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

  it("should seed the form from the task row's config", async () => {
    const client = createTestClient();
    const config = createWriteConfig("dev_1", [createWriteEndpoint("ep1", "/seeded")]);
    const draft = await createDraft(client, config);
    await renderWrite({ client, taskKey: draft.key });
    await screen.findByText(/\/seeded/);
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
      const created = await deployAndAwaitTask(client, container, draft.key);

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
      const created = await deployAndAwaitTask(client, container, draft.key);
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
