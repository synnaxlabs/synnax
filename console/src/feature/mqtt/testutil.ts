// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, type rack, type Synnax, task } from "@synnaxlabs/client";
import { deep, TimeStamp } from "@synnaxlabs/x";
import { uniqueName } from "@synnaxlabs/x/testutil";
import { fireEvent, screen, within } from "@testing-library/react";
import { onTestFinished } from "vitest";

import { Device } from "@/feature/mqtt/device";
import { createTestDevice, startRackHeartbeat } from "@/platform/device/testutil";
import { awaitCommand } from "@/platform/task/testutil";

export interface CreateBrokerOptions {
  configured?: boolean;
  properties?: Partial<Device.Properties>;
  /** Defaults to a new rack. */
  rack?: rack.Key;
}

/** Creates an MQTT broker device on the live Core. */
export const createBroker = async (
  client: Synnax,
  { configured = true, properties, rack: rackKey }: CreateBrokerOptions = {},
): Promise<device.Device> =>
  await createTestDevice(
    client,
    {
      name: uniqueName("mqtt_broker"),
      make: Device.MAKE,
      model: "MQTT broker",
      configured,
      properties: { ...deep.copy(Device.ZERO_PROPERTIES), ...properties },
      rack: rackKey,
    },
    Device.SCHEMAS,
  );

export interface ScannerReply {
  variant?: "success" | "error";
  message?: string;
  data?: object;
}

export interface Scanner {
  /** Answers the next command of the scan task, and returns that command. */
  answer: (reply: ScannerReply) => Promise<task.Command>;
  /** Waits for the next command of the scan task, and does not answer it. */
  next: () => Promise<task.Command>;
  /** Answers cmd, which next returned. */
  reply: (cmd: task.Command, reply: ScannerReply) => Promise<void>;
}

/**
 * Creates a scan task on the rack and answers its commands in place of the Core. The
 * rack reports as running until the test ends, because an outage fails a command wait.
 * Open it before the command goes out.
 */
export const openScanner = async (
  client: Synnax,
  rackKey: rack.Key,
): Promise<Scanner> => {
  const rck = await client.racks.retrieve({ key: rackKey });
  onTestFinished(await startRackHeartbeat(client, rck));
  const scanTask = await rck.createTask({
    name: "MQTT scanner",
    type: Device.SCAN_TYPE,
    config: {},
  });
  const streamer = await client.openStreamer(task.COMMAND_CHANNEL_NAME);
  onTestFinished(() => streamer.close());
  const next = async () => await awaitCommand(streamer, scanTask.key);
  const reply = async (
    cmd: task.Command,
    { variant = "success", message = "", data }: ScannerReply,
  ) => {
    await client.statuses.set({
      owner: task.resourceID(scanTask.key),
      name: scanTask.name,
      variant,
      message,
      time: TimeStamp.now(),
      details: { running: true, cmd: cmd.key, data },
    });
  };
  return {
    answer: async (r) => {
      const cmd = await next();
      await reply(cmd, r);
      return cmd;
    },
    next,
    reply,
  };
};

/** Opens the Add data dialog of a task form, and returns queries scoped to it. */
export const openAddData = async () => {
  fireEvent.click(await screen.findByRole("button", { name: "Add data" }));
  return within(await screen.findByRole("dialog"));
};
