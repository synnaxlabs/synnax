// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, type rack, task } from "@synnaxlabs/client";

import { connect, type ConnectionOptions } from "@/fixtures/client";

/** A device the simulated Driver reports, minus the rack it is created on. */
export interface SimulatedDevice extends Omit<device.New, "rack"> {
  /** Message of the device status. Defaults to "Device present". */
  message?: string;
}

/** An NI USB-6343 the Driver found but nobody has configured yet. */
export const NI_USB_6343: SimulatedDevice = {
  key: "01F5A3C2",
  name: "NI USB-6343",
  make: "NI",
  model: "USB-6343",
  location: "Dev1",
  properties: { is_simulated: false, resource_name: "Dev1", is_chassis: false },
};

/** A LabJack T7 on USB that nobody has configured yet. */
export const LABJACK_T7: SimulatedDevice = {
  key: "470018422",
  name: "LJM_dtT7-8422",
  make: "LabJack",
  model: "LJM_dtT7",
  location: "LJM_ctUSB",
  properties: { identifier: "" },
};

/** A Modbus TCP server connected by hand. */
export const MODBUS_PLC: SimulatedDevice = {
  key: "5b0f2d7e-6a1c-4f43-9a2e-1d6b8c4f9e21",
  name: "Pump skid PLC",
  make: "Modbus",
  model: "Modbus",
  location: "192.168.1.20:502",
  configured: true,
  message: "Device connected",
  properties: {
    connection: { host: "192.168.1.20", port: 502, swapBytes: false, swapWords: false },
    read: { index: 0, channels: {} },
    write: { channels: {} },
  },
};

export interface DriverOptions extends ConnectionOptions {
  /** Devices to create on the embedded rack, with a status for each. */
  devices?: SimulatedDevice[];
  /** Interval between "Driver is running" heartbeats. */
  periodMs?: number;
}

export interface DriverFixture {
  /** The embedded rack the devices and tasks live on. */
  rack: rack.Key;
  /** Stops the heartbeat and the command loop and closes the client. */
  stop: () => Promise<void>;
}

/**
 * simulatedDriver stands in for the Driver on a capture core started with
 * `--no-driver`. It creates the devices on the embedded rack, writes the rack's
 * "Driver is running" status every `periodMs`, and answers each start and stop
 * command with the status the real Driver writes on success. Nothing reads or writes
 * hardware, so read tasks produce no samples. The rack claims no integrations, so the
 * Modbus, OPC UA, and HTTP connect dialogs find no Driver.
 */
export const simulatedDriver = async ({
  devices = [],
  periodMs = 1000,
  ...opts
}: DriverOptions = {}): Promise<DriverFixture> => {
  const client = connect(opts);
  const [embedded] = await client.racks.retrieve({ embedded: true });
  const rackStatus = () => ({
    key: `rack:${embedded.key}`,
    name: embedded.name,
    variant: "success" as const,
    message: "Driver is running",
    details: { rack: embedded.key },
  });
  await client.statuses.set(rackStatus());

  if (devices.length > 0)
    await client.devices.create(
      devices.map(({ message = "Device present", ...dev }) => ({
        ...dev,
        rack: embedded.key,
        status: {
          key: `device:${dev.key}`,
          name: dev.name,
          variant: "success" as const,
          message,
          details: { rack: embedded.key, device: dev.key },
        },
      })),
    );

  const streamer = await client.openStreamer([task.COMMAND_CHANNEL_NAME]);
  const answer = (async () => {
    for await (const frame of streamer)
      for (const cmd of frame.get(task.COMMAND_CHANNEL_NAME).parseJSON(task.commandZ)) {
        if (cmd.type !== "start" && cmd.type !== "stop") continue;
        const t = await client.tasks.retrieve({ key: cmd.task });
        const running = cmd.type === "start";
        await client.statuses.set({
          key: task.statusKey(t.key),
          name: t.name,
          variant: "success",
          message: running ? "Task started successfully" : "Task stopped successfully",
          details: {
            task: t.key,
            running,
            cmd: cmd.key,
            configHash: t.configHash,
            rack: embedded.key,
          },
        });
      }
  })();

  let running = true;
  const heartbeat = (async () => {
    while (running) {
      await new Promise((resolve) => setTimeout(resolve, periodMs));
      if (running) await client.statuses.set(rackStatus());
    }
  })();

  return {
    rack: embedded.key,
    stop: async () => {
      running = false;
      streamer.close();
      await Promise.all([answer, heartbeat]);
      await client.close();
    },
  };
};
