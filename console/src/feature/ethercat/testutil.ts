// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ethercat, type Synnax } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";

import { Device } from "@/feature/ethercat/device";
import { Task } from "@/feature/ethercat/task";

/** Builds a Core-safe device identifier (2-12 chars, letter first). */
export const createIdentifier = (): string =>
  `s${id.create().replace(/[^a-zA-Z0-9]/g, "")}`.slice(0, 12);

export const STATUS_ENTRY: ethercat.PDOEntry = {
  name: "Status",
  index: 0x6000,
  subIndex: 1,
  bitLength: 16,
  dataType: "uint16",
};

export const CONTROL_ENTRY: ethercat.PDOEntry = {
  name: "Control",
  index: 0x7000,
  subIndex: 1,
  bitLength: 16,
  dataType: "uint16",
};

/** Builds a PDO set with one input ("Status") and one output ("Control") entry. */
export const createPDOs = (): Device.PDOs => ({
  inputs: [STATUS_ENTRY],
  outputs: [CONTROL_ENTRY],
});

/**
 * Builds the PDO set of an eight-channel digital terminal pair (EL1008 and EL2008):
 * eight one-bit entries per direction that share one name, told apart by their PDO
 * names ("Channel 1" to "Channel 8") and addresses (0x6000:01 to 0x6070:01).
 */
export const createRepeatedNamePDOs = (): Device.PDOs => {
  const channels = Array.from({ length: 8 }, (_, i) => i);
  const entry = (name: string, base: number, i: number): ethercat.PDOEntry => ({
    name,
    pdoName: `Channel ${i + 1}`,
    index: base + 0x10 * i,
    subIndex: 1,
    bitLength: 1,
    dataType: "uint8",
  });
  return {
    inputs: channels.map((i) => entry("Input", 0x6000, i)),
    outputs: channels.map((i) => entry("Output", 0x7000, i)),
  };
};

/** Builds an input channel bound to an entry on a slave device. */
export const createEntryReadChannel = (
  device: string,
  { index, subIndex, dataType }: ethercat.PDOEntry,
  overrides: Partial<Task.EntryReadChannel> = {},
): Task.EntryReadChannel =>
  Task.READ_CHANNEL_SCHEMAS.entry.parse({
    type: "entry",
    device,
    index,
    subIndex,
    dataType,
    ...overrides,
  });

/** Builds an input channel whose entry name matched no single entry. */
export const createUnresolvedReadChannel = (
  device: string,
  pdo: string,
  overrides: Partial<ethercat.UnresolvedReadChannel> = {},
): ethercat.UnresolvedReadChannel =>
  Task.READ_CHANNEL_SCHEMAS.unresolved.parse({
    type: "unresolved",
    device,
    pdo,
    ...overrides,
  });

/** Builds an output channel bound to an entry on a slave device. */
export const createEntryWriteChannel = (
  device: string,
  { index, subIndex, dataType }: ethercat.PDOEntry,
  overrides: Partial<Task.EntryWriteChannel> = {},
): Task.EntryWriteChannel =>
  Task.WRITE_CHANNEL_SCHEMAS.entry.parse({
    type: "entry",
    device,
    index,
    subIndex,
    dataType,
    ...overrides,
  });

/** Builds an output channel whose entry name matched no single entry. */
export const createUnresolvedWriteChannel = (
  device: string,
  pdo: string,
  overrides: Partial<ethercat.UnresolvedWriteChannel> = {},
): ethercat.UnresolvedWriteChannel =>
  Task.WRITE_CHANNEL_SCHEMAS.unresolved.parse({
    type: "unresolved",
    device,
    pdo,
    ...overrides,
  });

/**
 * Creates an EtherCAT slave device on the live Core. `properties` are merged over
 * ZERO_SLAVE_PROPERTIES; `properties.name` doubles as the device name when set.
 */
export const createSlaveDevice = async (
  client: Synnax,
  rackKey: number,
  properties: Partial<Device.SlaveProperties> = {},
  configured = true,
): Promise<Device.SlaveDevice> => {
  const key = id.create();
  return await client.devices.create(
    {
      key,
      name: properties.name ?? `EtherCAT Slave ${key}`,
      rack: rackKey,
      location: "test-location",
      make: Device.MAKE,
      model: Device.SLAVE_MODEL,
      configured,
      properties: { ...Device.ZERO_SLAVE_PROPERTIES, ...properties },
    },
    Device.SLAVE_SCHEMAS,
  );
};

/** Creates a slave on eth0 whose eight channels repeat the same entry names. */
export const createRepeatedNameSlave = async (
  client: Synnax,
  rackKey: number,
): Promise<Device.SlaveDevice> =>
  await createSlaveDevice(client, rackKey, {
    identifier: createIdentifier(),
    network: "eth0",
    pdos: createRepeatedNamePDOs(),
  });
