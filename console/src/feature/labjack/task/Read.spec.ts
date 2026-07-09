// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { LabJack } from "@/feature/labjack";
import {
  createAIChannel,
  createDIChannel,
  createLabJackDevice,
  createTCChannel,
} from "@/feature/labjack/testutil";
import {
  awaitTaskKey,
  clickConfigure,
  findDialogTriggerByText,
  renderTaskFormLayout,
} from "@/platform/task/testutil";
import { stubGeometry, uniqueName } from "@/testutil";

const client = createTestClient();

stubGeometry();

const renderRead = async (args = {}) =>
  await renderTaskFormLayout(LabJack.Task.Read, LabJack.Task.READ_TYPE, {
    client,
    args,
  });

const createConfig = (
  device: string,
  channels: LabJack.Task.InputChannel[],
): unknown => ({ ...LabJack.Task.ZERO_READ_PAYLOAD.config, device, channels });

describe("LabJack Read", () => {
  it("should render channel ports using their model aliases when available", async () => {
    const dev = await createLabJackDevice(client);
    await renderRead({
      config: createConfig(dev.key, [
        createAIChannel("AIN0"),
        createAIChannel("AIN4"),
        createDIChannel("DIO8"),
      ]),
    });
    await waitFor(() => expect(screen.getByText("AIN0")).toBeTruthy());
    expect(screen.getByText("FIO4")).toBeTruthy();
    expect(screen.getByText("EIO0")).toBeTruthy();
  });

  it("should fall back to the raw port when the model does not list it", async () => {
    const dev = await createLabJackDevice(client);
    await renderRead({
      config: createConfig(dev.key, [createAIChannel("AIN999")]),
    });
    await waitFor(() => expect(screen.getByText("AIN999")).toBeTruthy());
  });

  it("should show the AI detail form with its scale editor", async () => {
    const dev = await createLabJackDevice(client);
    await renderRead({
      config: createConfig(dev.key, [createAIChannel("AIN0")]),
    });
    fireEvent.click(await screen.findByText("AIN0"));
    await waitFor(() => expect(screen.getByText("Max Voltage")).toBeTruthy());
    expect(screen.getByText("Scale")).toBeTruthy();
    expect(screen.queryByText("Slope")).toBeNull();
  });

  it("should show slope and offset for a linear-scaled AI channel", async () => {
    const dev = await createLabJackDevice(client);
    await renderRead({
      config: createConfig(dev.key, [
        createAIChannel("AIN0", {
          scale: { type: "linear", slope: 2, offset: 1 },
        }),
      ]),
    });
    fireEvent.click(await screen.findByText("AIN0"));
    await waitFor(() => expect(screen.getByText("Slope")).toBeTruthy());
    expect(screen.getByText("Offset")).toBeTruthy();
  });

  it("should show the thermocouple detail form for a TC channel", async () => {
    const dev = await createLabJackDevice(client);
    await renderRead({
      config: createConfig(dev.key, [createTCChannel("AIN0")]),
    });
    fireEvent.click(await screen.findByText("AIN0"));
    await waitFor(() => expect(screen.getByText("Thermocouple Type")).toBeTruthy());
    expect(screen.getByText("Temperature Units")).toBeTruthy();
    expect(screen.getByText("Positive Channel")).toBeTruthy();
    expect(screen.getByText("Negative Channel")).toBeTruthy();
    expect(screen.getByText("CJC Source")).toBeTruthy();
    expect(screen.getByText("CJC Slope")).toBeTruthy();
    expect(screen.getByText("CJC Offset")).toBeTruthy();
  });

  it("should show no extra detail fields for a DI channel", async () => {
    const dev = await createLabJackDevice(client);
    await renderRead({
      config: createConfig(dev.key, [createDIChannel("DIO8")]),
    });
    fireEvent.click(await screen.findByText("EIO0"));
    await waitFor(() => expect(screen.getByText("Channel Type")).toBeTruthy());
    expect(screen.queryByText("Max Voltage")).toBeNull();
    expect(screen.queryByText("Thermocouple Type")).toBeNull();
  });

  it("should swap the channel type and remap its port to the new port space", async () => {
    const dev = await createLabJackDevice(client);
    await renderRead({
      config: createConfig(dev.key, [createAIChannel("AIN0")]),
    });
    fireEvent.click(await screen.findByText("AIN0"));
    await waitFor(() => expect(screen.getByText("Max Voltage")).toBeTruthy());
    fireEvent.click(await findDialogTriggerByText("Analog Input"));
    fireEvent.click(await screen.findByText("Digital Input"));
    await waitFor(() => expect(screen.queryByText("Max Voltage")).toBeNull());
    await waitFor(() => expect(screen.getAllByText("FIO4").length).toBeGreaterThan(0));
  });

  describe("configure against a live cluster", () => {
    it("should create the index and data channels, update the device, and save the task", async () => {
      const dev = await createLabJackDevice(client);
      const namedChannel = uniqueName("lj_named");
      const { store, layoutKey } = await renderRead({
        config: createConfig(dev.key, [
          createAIChannel("AIN0"),
          createDIChannel("DIO8", { name: namedChannel }),
        ]),
      });
      await clickConfigure();
      const taskKey = await awaitTaskKey(store, layoutKey);
      const created = await client.tasks.retrieve({
        key: taskKey,
        schemas: LabJack.Task.READ_SCHEMAS,
      });
      expect(created.type).toBe(LabJack.Task.READ_TYPE);
      expect(task.rackKey(created.key)).toBe(dev.rack);
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

    it("should reuse existing channels when reconfigured", async () => {
      const dev = await createLabJackDevice(client);
      const config = createConfig(dev.key, [createAIChannel("AIN0")]);
      const first = await renderRead({ config });
      await clickConfigure();
      const firstKey = await awaitTaskKey(first.store, first.layoutKey);
      const firstTask = await client.tasks.retrieve({
        key: firstKey,
        schemas: LabJack.Task.READ_SCHEMAS,
      });
      first.unmount();

      const second = await renderRead({ config });
      await clickConfigure();
      const secondKey = await awaitTaskKey(second.store, second.layoutKey);
      const secondTask = await client.tasks.retrieve({
        key: secondKey,
        schemas: LabJack.Task.READ_SCHEMAS,
      });
      expect(secondTask.config.channels[0].channel).toBe(
        firstTask.config.channels[0].channel,
      );
    });

    it("should recreate the index when the stored one no longer exists", async () => {
      const dev = await createLabJackDevice(client, {
        properties: { readIndex: 999999999 },
      });
      const { store, layoutKey } = await renderRead({
        config: createConfig(dev.key, [createAIChannel("AIN0")]),
      });
      await clickConfigure();
      await awaitTaskKey(store, layoutKey);
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
