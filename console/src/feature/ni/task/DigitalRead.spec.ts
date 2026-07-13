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
import { id } from "@synnaxlabs/x";
import { screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { NI } from "@/feature/ni";
import { createNIDevice, renderNITaskForm } from "@/feature/ni/task/testutil";
import {
  awaitTaskKey,
  clickConfigure,
  commitFieldInput,
} from "@/platform/task/testutil";
import { stubGeometry, uniqueName } from "@/testutil";

stubGeometry();

const client = createTestClient();

interface CreateChannelOverrides extends Partial<NI.Task.DIChannel> {}

const createChannel = (
  port: number,
  line: number,
  overrides: CreateChannelOverrides = {},
): NI.Task.DIChannel => ({
  ...NI.Task.ZERO_DI_CHANNEL,
  key: id.create(),
  port,
  line,
  ...overrides,
});

const renderDigitalRead = async (args = {}) =>
  await renderNITaskForm(NI.Task.DigitalRead, NI.Task.DIGITAL_READ_TYPE, {
    client,
    args,
  });

const createConfig = (
  channels: NI.Task.DIChannel[],
  device = "placeholder_device",
) => ({ ...NI.Task.ZERO_DIGITAL_READ_PAYLOAD.config, device, channels });

describe("DigitalRead", () => {
  it("should write edits to a channel's line number back into the form", async () => {
    await renderDigitalRead({
      config: createConfig([
        createChannel(0, 0, { name: "di_chan_a" }),
        createChannel(0, 1, { name: "di_chan_b" }),
      ]),
    });
    await waitFor(() => expect(screen.getByText("di_chan_b")).toBeTruthy());
    commitFieldInput(screen.getByDisplayValue("1"), "7");
    await waitFor(() => expect(screen.getByDisplayValue("7")).toBeTruthy());
    expect(screen.queryByDisplayValue("1")).toBeNull();
  });

  describe("configure against a live cluster", () => {
    it("should create per-line channels keyed by port and line and update the device", async () => {
      const dev = await createNIDevice(client);
      const namedChannel = uniqueName("di_named");
      const { store, layoutKey } = await renderDigitalRead({
        config: createConfig(
          [createChannel(0, 0), createChannel(0, 1, { name: namedChannel })],
          dev.key,
        ),
      });
      await clickConfigure();
      const taskKey = await awaitTaskKey(store, layoutKey);
      const created = await client.tasks.retrieve({
        key: taskKey,
        schemas: NI.Task.DIGITAL_READ_SCHEMAS,
      });
      expect(created.type).toBe(NI.Task.DIGITAL_READ_TYPE);
      expect(task.rackKey(created.key)).toBe(dev.rack);
      const [c0, c1] = created.config.channels;
      expect(c0.channel).not.toBe(0);
      expect(c1.channel).not.toBe(0);

      const identifier = dev.properties.identifier;
      const defaultNamed = await client.channels.retrieve(c0.channel);
      expect(defaultNamed.name).toBe(`${identifier}_di_0_0`);
      const named = await client.channels.retrieve(c1.channel);
      expect(named.name).toBe(namedChannel);

      const updated = await client.devices.retrieve({
        key: dev.key,
        schemas: NI.Device.SCHEMAS,
      });
      expect(updated.properties.digitalInput.channels["0l0"]).toBe(c0.channel);
      expect(updated.properties.digitalInput.channels["0l1"]).toBe(c1.channel);
      const index = await client.channels.retrieve(
        updated.properties.digitalInput.index,
      );
      expect(index.name).toBe(`${identifier}_di_time`);
      expect(index.isIndex).toBe(true);
    });

    it("should reuse existing channels when reconfigured", async () => {
      const dev = await createNIDevice(client);
      const { store, layoutKey } = await renderDigitalRead({
        config: createConfig([createChannel(0, 0)], dev.key),
      });
      await clickConfigure();
      const taskKey = await awaitTaskKey(store, layoutKey);
      const first = await client.tasks.retrieve({
        key: taskKey,
        schemas: NI.Task.DIGITAL_READ_SCHEMAS,
      });
      await clickConfigure();
      await waitFor(async () => {
        const again = await client.tasks.retrieve({
          key: taskKey,
          schemas: NI.Task.DIGITAL_READ_SCHEMAS,
        });
        expect(again.config.channels[0].channel).toBe(first.config.channels[0].channel);
      });
    });
  });
});
