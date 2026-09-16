// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax, type task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { LabJack } from "@/feature/labjack";
import {
  createAnalogReadChannel,
  createDigitalReadChannel,
  createLabJackDevice,
  createThermocoupleReadChannel,
} from "@/feature/labjack/testutil";
import {
  createTestChannel,
  deployAndAwaitTask,
  findChannelListItem,
  findDialogTriggerByText,
  renderTaskFormTab,
  type RenderTaskFormTabOptions,
} from "@/platform/task/testutil";
import { uniqueName } from "@/testutil";

const client = createTestClient();

const renderRead = async (options: RenderTaskFormTabOptions = {}) =>
  await renderTaskFormTab(LabJack.Task.Read, options);

const createConfig = (
  device: string,
  channels: LabJack.Task.ReadChannel[],
): LabJack.Task.ReadPayload["config"] => ({
  ...LabJack.Task.READ_SCHEMAS.config.parse({}),
  device,
  channels,
});

// Drafts carry no key; the created row mints its own.
const ZERO_DRAFT: task.New<LabJack.Task.ReadSchemas> = {
  name: "LabJack read task",
  type: LabJack.Task.READ_TYPE,
  config: LabJack.Task.READ_SCHEMAS.config.parse({}),
};

const createDraft = async (
  client: Synnax,
  config: LabJack.Task.ReadPayload["config"],
) => await client.tasks.create({ ...ZERO_DRAFT, config }, LabJack.Task.READ_SCHEMAS);

describe("LabJack Read", () => {
  it("should prompt for a selection when the form carries no device", async () => {
    const draft = await createDraft(client, createConfig("", []));
    await renderRead({ client, taskKey: draft.key });
    await waitFor(() => expect(screen.getByText("No device selected")).toBeTruthy());
  });

  it("should prompt to configure an unconfigured device", async () => {
    const dev = await createLabJackDevice(client, { configured: false });
    const draft = await createDraft(client, createConfig(dev.key, []));
    await renderRead({ client, taskKey: draft.key });
    await waitFor(() => expect(screen.getByText(`Configure ${dev.name}`)).toBeTruthy());
  });

  it("should address each channel of a config built without explicit keys", async () => {
    const dev = await createLabJackDevice(client);
    // A client that leaves the key to the schema, the way every generated client and
    // every documented example does.
    const channels = ["alpha", "bravo", "charlie"].map((name, i) => ({
      ...LabJack.Task.createReadChannel("analog"),
      port: `AIN${i}`,
      name,
    }));
    const draft = await createDraft(client, createConfig(dev.key, channels));
    await renderRead({ client, taskKey: draft.key });
    for (const name of ["alpha", "bravo", "charlie"])
      await waitFor(() => expect(screen.getAllByText(name)).toHaveLength(1));
  });

  it("should render channel ports using their model aliases when available", async () => {
    const dev = await createLabJackDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [
        createAnalogReadChannel("AIN0"),
        createAnalogReadChannel("AIN4"),
        createDigitalReadChannel("DIO8"),
      ]),
    );
    await renderRead({ client, taskKey: draft.key });
    await findChannelListItem("AIN0");
    await findChannelListItem("FIO4");
    await findChannelListItem("EIO0");
  });

  it("should fall back to the raw port when the model does not list it", async () => {
    const dev = await createLabJackDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [createAnalogReadChannel("AIN999")]),
    );
    await renderRead({ client, taskKey: draft.key });
    await findChannelListItem("AIN999");
  });

  it("should show the AI detail form with its scale editor", async () => {
    const dev = await createLabJackDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [createAnalogReadChannel("AIN0")]),
    );
    await renderRead({ client, taskKey: draft.key });
    fireEvent.click(await findChannelListItem("AIN0"));
    await waitFor(() => expect(screen.getByText("Max voltage")).toBeTruthy());
    expect(screen.getByText("Scale")).toBeTruthy();
    expect(screen.queryByText("Slope")).toBeNull();
  });

  it("should show slope and offset for a linear-scaled AI channel", async () => {
    const dev = await createLabJackDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [
        createAnalogReadChannel("AIN0", {
          scale: { type: "linear", slope: 2, offset: 1 },
        }),
      ]),
    );
    await renderRead({ client, taskKey: draft.key });
    fireEvent.click(await findChannelListItem("AIN0"));
    await waitFor(() => expect(screen.getByText("Slope")).toBeTruthy());
    expect(screen.getByText("Offset")).toBeTruthy();
  });

  it("should show the thermocouple detail form for a TC channel", async () => {
    const dev = await createLabJackDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [createThermocoupleReadChannel("AIN0")]),
    );
    await renderRead({ client, taskKey: draft.key });
    fireEvent.click(await findChannelListItem("AIN0"));
    await waitFor(() => expect(screen.getByText("Thermocouple type")).toBeTruthy());
    expect(screen.getByText("Temperature units")).toBeTruthy();
    expect(screen.getByText("Negative channel")).toBeTruthy();
    // The port selector is the only control for the positive lead.
    expect(screen.queryByText("Positive channel")).toBeNull();
    expect(screen.getByText("CJC source")).toBeTruthy();
    expect(screen.getByText("CJC slope")).toBeTruthy();
    expect(screen.getByText("CJC offset")).toBeTruthy();
  });

  it("should show no extra detail fields for a DI channel", async () => {
    const dev = await createLabJackDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [createDigitalReadChannel("DIO8")]),
    );
    await renderRead({ client, taskKey: draft.key });
    fireEvent.click(await findChannelListItem("EIO0"));
    await waitFor(() => expect(screen.getByText("Channel type")).toBeTruthy());
    expect(screen.queryByText("Max voltage")).toBeNull();
    expect(screen.queryByText("Thermocouple type")).toBeNull();
  });

  it("should swap the channel type and remap its port to the new port space", async () => {
    const dev = await createLabJackDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [createAnalogReadChannel("AIN0")]),
    );
    await renderRead({ client, taskKey: draft.key });
    fireEvent.click(await findChannelListItem("AIN0"));
    await waitFor(() => expect(screen.getByText("Max voltage")).toBeTruthy());
    fireEvent.click(await findDialogTriggerByText("Analog input"));
    fireEvent.click(await screen.findByText("Digital input"));
    await waitFor(() => expect(screen.queryByText("Max voltage")).toBeNull());
    await waitFor(() => expect(screen.getAllByText("FIO4").length).toBeGreaterThan(0));
  });

  describe("deploying against a live Core", () => {
    it("should create the index and data channels, update the device, and save the task", async () => {
      const dev = await createLabJackDevice(client);
      const namedChannel = uniqueName("lj_named");
      const draft = await createDraft(
        client,
        createConfig(dev.key, [
          createAnalogReadChannel("AIN0"),
          createDigitalReadChannel("DIO8", { name: namedChannel }),
        ]),
      );
      const { container } = await renderRead({ client, taskKey: draft.key });
      const created = await deployAndAwaitTask(
        client,
        container,
        draft.key,
        LabJack.Task.READ_SCHEMAS,
      );
      expect(created.type).toBe(LabJack.Task.READ_TYPE);
      expect(created.rack).toBe(dev.rack);
      const [ai, di] = created.config.channels;
      expect(ai.channel).not.toBe(0);
      expect(di.channel).not.toBe(0);

      const identifier = dev.properties.identifier;
      const aiChannel = await client.channels.retrieve(ai.channel);
      expect(aiChannel.name).toBe(`${identifier}_AIN0`);
      expect(aiChannel.dataType.toString()).toBe("float32");
      const diChannel = await client.channels.retrieve(di.channel);
      expect(diChannel.name).toBe(namedChannel);
      expect(diChannel.dataType.toString()).toBe("uint8");

      const updated = await client.devices.retrieve({
        key: dev.key,
        schemas: LabJack.Device.SCHEMAS,
      });
      expect(updated.properties.readIndex).not.toBe(0);
      expect(updated.properties.AI.channels.AIN0).toBe(ai.channel);
      expect(updated.properties.DI.channels.DIO8).toBe(di.channel);
      const index = await client.channels.retrieve(updated.properties.readIndex);
      expect(index.name).toBe(`${identifier}_time`);
      expect(index.isIndex).toBe(true);
    });

    it("should reuse existing channels when redeployed", async () => {
      const dev = await createLabJackDevice(client);
      const config = createConfig(dev.key, [createAnalogReadChannel("AIN0")]);
      const firstDraft = await createDraft(client, config);
      const first = await renderRead({ client, taskKey: firstDraft.key });
      const firstTask = await deployAndAwaitTask(
        client,
        first.container,
        firstDraft.key,
        LabJack.Task.READ_SCHEMAS,
      );
      first.unmount();

      const secondDraft = await createDraft(client, config);
      const second = await renderRead({ client, taskKey: secondDraft.key });
      const secondTask = await deployAndAwaitTask(
        client,
        second.container,
        secondDraft.key,
        LabJack.Task.READ_SCHEMAS,
      );
      expect(secondTask.config.channels[0].channel).toBe(
        firstTask.config.channels[0].channel,
      );
    });

    it("should recreate the index when the stored one no longer exists", async () => {
      const dev = await createLabJackDevice(client, {
        properties: { readIndex: 999999999 },
      });
      const draft = await createDraft(
        client,
        createConfig(dev.key, [createAnalogReadChannel("AIN0")]),
      );
      const { container } = await renderRead({ client, taskKey: draft.key });
      await deployAndAwaitTask(client, container, draft.key, LabJack.Task.READ_SCHEMAS);
      const updated = await client.devices.retrieve({
        key: dev.key,
        schemas: LabJack.Device.SCHEMAS,
      });
      expect(updated.properties.readIndex).not.toBe(0);
      expect(updated.properties.readIndex).not.toBe(999999999);
      const index = await client.channels.retrieve(updated.properties.readIndex);
      expect(index.isIndex).toBe(true);
    });
  });
});

describe("LabJack Read device map binding", () => {
  it("should show the channel the device map binds to a port that holds none", async () => {
    const bound = await createTestChannel(client, "lj");
    const dev = await createLabJackDevice(client, {
      properties: { AI: { channels: { AIN0: bound.key } } },
    });
    const draft = await createDraft(
      client,
      createConfig(dev.key, [createAnalogReadChannel("AIN0")]),
    );
    await renderRead({ client, taskKey: draft.key });
    await screen.findByText(bound.name);
  });

  it("should drop a port's stale channel when the device map has no entry for it", async () => {
    const stale = await createTestChannel(client, "lj");
    const dev = await createLabJackDevice(client);
    const draft = await createDraft(
      client,
      createConfig(dev.key, [createAnalogReadChannel("AIN0", { channel: stale.key })]),
    );
    await renderRead({ client, taskKey: draft.key });
    await screen.findByText("No channel");
    expect(screen.queryByText(stale.name)).toBeNull();
  });

  it("should bind a digital port from its own map", async () => {
    const bound = await createTestChannel(client, "lj_di");
    const dev = await createLabJackDevice(client, {
      properties: { DI: { channels: { DIO4: bound.key } } },
    });
    const draft = await createDraft(
      client,
      createConfig(dev.key, [createDigitalReadChannel("DIO4")]),
    );
    await renderRead({ client, taskKey: draft.key });
    await screen.findByText(bound.name);
  });
});
