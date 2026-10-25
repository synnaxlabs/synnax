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

interface Entry {
  name: string;
  pdo_index: number;
  index: number;
  sub_index: number;
  bit_length: number;
  data_type: string;
}

const entry = (
  pdo_index: number,
  index: number,
  sub_index: number,
  name: string,
  data_type: string,
  bit_length: number,
): Entry => ({ name, pdo_index, index, sub_index, bit_length, data_type });

/** The input entries of one EL3102 channel in its standard PDOs. */
const el3102Standard = (status: number, compact: number, index: number): Entry[] => [
  entry(status, index, 0x01, "Status__Underrange", "uint8", 1),
  entry(status, index, 0x02, "Status__Overrange", "uint8", 1),
  entry(status, index, 0x03, "Status__Limit 1", "uint8", 2),
  entry(status, index, 0x05, "Status__Limit 2", "uint8", 2),
  entry(status, index, 0x07, "Status__Error", "uint8", 1),
  entry(status, index, 0x0e, "Status__Sync error", "uint8", 1),
  entry(status, index, 0x0f, "Status__TxPDO State", "uint8", 1),
  entry(status, index, 0x10, "Status__TxPDO Toggle", "uint8", 1),
  entry(status, index, 0x11, "Value", "int16", 16),
  entry(compact, index, 0x11, "Value", "int16", 16),
];

/**
 * An EL3102 module that the Driver found on the en5 port but nobody has configured yet,
 * with the properties SOEM discovery writes from the ESI catalog.
 */
export const EL3102: SimulatedDevice = {
  key: "ethercat_en5_2_203305042_2",
  name: "EL3102",
  make: "ethercat",
  model: "slave",
  location: "en5.Slot 2",
  message: "Discovered (28 inputs, 0 outputs)",
  properties: {
    vendor_id: 2,
    product_code: 0x0c1e3052,
    revision: 0x00140000,
    serial: 0,
    name: "EL3102",
    network: "en5",
    position: 2,
    input_bits: 48,
    output_bits: 0,
    pdo_order_reliable: true,
    enabled: true,
    pdos: {
      inputs: [
        entry(0x1a00, 0x3101, 0x01, "Status", "uint8", 8),
        entry(0x1a00, 0x3101, 0x02, "Value", "int16", 16),
        entry(0x1a01, 0x3102, 0x01, "Status", "uint8", 8),
        entry(0x1a01, 0x3102, 0x02, "Value", "int16", 16),
        ...el3102Standard(0x1a02, 0x1a03, 0x6000),
        ...el3102Standard(0x1a04, 0x1a05, 0x6010),
        entry(0x1a07, 0x6001, 0x11, "Value", "int32", 32),
        entry(0x1a09, 0x6011, 0x11, "Value", "int32", 32),
        entry(0x1a10, 0x3101, 0x02, "Channel 1", "int16", 16),
        entry(0x1a10, 0x3102, 0x02, "Channel 2", "int16", 16),
      ],
      outputs: [],
    },
  },
};

/** The output entry of one EL2004 channel, in its standard PDO. */
const el2004Output = (channel: number): Entry =>
  entry(0x1600 + channel, 0x7000 + channel * 0x10, 0x01, "Output", "uint8", 1);

/**
 * An EL2004 digital output module that the Driver found on the en5 port, after the
 * EL3102, with the properties SOEM discovery writes from the ESI catalog.
 */
export const EL2004: SimulatedDevice = {
  key: "ethercat_en5_2_131346514_3",
  name: "EL2004",
  make: "ethercat",
  model: "slave",
  location: "en5.Slot 3",
  message: "Discovered (0 inputs, 4 outputs)",
  properties: {
    vendor_id: 2,
    product_code: 0x07d43052,
    revision: 0x00110000,
    serial: 0,
    name: "EL2004",
    network: "en5",
    position: 3,
    input_bits: 0,
    output_bits: 4,
    pdo_order_reliable: true,
    enabled: true,
    pdos: { inputs: [], outputs: [0, 1, 2, 3].map(el2004Output) },
  },
};

/** configured returns the device as a user leaves it after configuring it. */
export const configured = (
  dev: SimulatedDevice,
  name: string,
  identifier: string,
): SimulatedDevice => ({
  ...dev,
  name,
  configured: true,
  properties: { ...dev.properties, identifier },
});

/** rackStatus returns the status a Driver writes for its rack while it runs. */
export const rackStatus = ({ key, name }: { key: rack.Key; name: string }) => ({
  key: `rack:${key}`,
  name,
  variant: "success" as const,
  message: "Driver is running",
  details: { rack: key },
});

/** deviceStatus returns the status a Driver on `rackKey` writes for a found device. */
export const deviceStatus = (
  rackKey: rack.Key,
  { key, name }: { key: string; name: string },
  message = "Device present",
) => ({
  key: `device:${key}`,
  name,
  variant: "success" as const,
  message,
  details: { rack: rackKey, device: key },
});

/**
 * taskStatus returns the status a Driver on `rackKey` writes once it starts or stops
 * a task. `cmd` is the key of the command it answers, if any.
 */
export const taskStatus = (
  rackKey: rack.Key,
  t: task.Task,
  running: boolean,
  cmd?: string,
) => ({
  key: task.statusKey(t.key),
  name: t.name,
  variant: "success" as const,
  message: running ? "Task started successfully" : "Task stopped successfully",
  details: {
    task: t.key,
    running,
    ...(cmd != null && { cmd }),
    configHash: t.configHash,
    rack: rackKey,
  },
});

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
 * `--no-driver`. It creates the devices on the embedded rack, writes the rack's "Driver
 * is running" status every `periodMs`, and answers each start and stop command with the
 * status the real Driver writes on success. Nothing reads or writes hardware, so read
 * tasks produce no samples. The rack claims no integrations, so the Modbus, OPC UA, and
 * HTTP connect dialogs find no Driver.
 */
export const simulatedDriver = async ({
  devices = [],
  periodMs = 1000,
  ...opts
}: DriverOptions = {}): Promise<DriverFixture> => {
  const client = connect(opts);
  const [embedded] = await client.racks.retrieve({ embedded: true });
  await client.statuses.set(rackStatus(embedded));

  if (devices.length > 0)
    await client.devices.create(
      devices.map(({ message, ...dev }) => ({
        ...dev,
        rack: embedded.key,
        status: deviceStatus(embedded.key, dev, message),
      })),
    );

  const streamer = await client.openStreamer([task.COMMAND_CHANNEL_NAME]);
  const answer = (async () => {
    for await (const frame of streamer)
      for (const cmd of frame.get(task.COMMAND_CHANNEL_NAME).parseJSON(task.commandZ)) {
        if (cmd.type !== "start" && cmd.type !== "stop") continue;
        const t = await client.tasks.retrieve({ key: cmd.task });
        await client.statuses.set(
          taskStatus(embedded.key, t, cmd.type === "start", cmd.key),
        );
      }
  })();

  let running = true;
  const heartbeat = (async () => {
    while (running) {
      await new Promise((resolve) => setTimeout(resolve, periodMs));
      if (running) await client.statuses.set(rackStatus(embedded));
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
