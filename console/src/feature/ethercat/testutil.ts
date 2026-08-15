// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Synnax } from "@synnaxlabs/client";
import { id } from "@synnaxlabs/x";

import {
  MAKE,
  type PDOs,
  SLAVE_MODEL,
  SLAVE_SCHEMAS,
  type SlaveDevice,
  type SlaveProperties,
  ZERO_SLAVE_PROPERTIES,
} from "@/feature/ethercat/device/types";
import {
  READ_CHANNEL_SCHEMAS,
  type ReadChannel,
  WRITE_CHANNEL_SCHEMAS,
  type WriteChannel,
} from "@/feature/ethercat/task/types";

type AutoReadChannel = Extract<ReadChannel, { type: "automatic" }>;
type ManualReadChannel = Extract<ReadChannel, { type: "manual" }>;
type AutoWriteChannel = Extract<WriteChannel, { type: "automatic" }>;
type ManualWriteChannel = Extract<WriteChannel, { type: "manual" }>;

/** Builds a cluster-safe device identifier (2-12 chars, letter first). */
export const createIdentifier = (): string =>
  `s${id.create().replace(/[^a-zA-Z0-9]/g, "")}`.slice(0, 12);

/** Builds a PDO set with one input ("Status") and one output ("Control") entry. */
export const createPDOs = (): PDOs => ({
  inputs: [
    { name: "Status", index: 0x6000, subIndex: 1, bitLength: 16, dataType: "uint16" },
  ],
  outputs: [
    { name: "Control", index: 0x7000, subIndex: 1, bitLength: 16, dataType: "uint16" },
  ],
});

/** Builds an automatic (PDO-driven) input channel bound to a slave device. */
export const createAutoReadChannel = (
  device: string,
  pdo: string,
  overrides: Partial<AutoReadChannel> = {},
): AutoReadChannel => ({
  ...READ_CHANNEL_SCHEMAS.automatic.parse({ type: "automatic" }),
  key: id.create(),
  device,
  pdo,
  ...overrides,
});

/** Builds a manual (address-driven) input channel bound to a slave device. */
export const createManualReadChannel = (
  device: string,
  index: number,
  subIndex: number,
  overrides: Partial<ManualReadChannel> = {},
): ManualReadChannel => ({
  ...READ_CHANNEL_SCHEMAS.manual.parse({ type: "manual" }),
  key: id.create(),
  device,
  index,
  subIndex,
  ...overrides,
});

/** Builds an automatic (PDO-driven) output channel bound to a slave device. */
export const createAutoWriteChannel = (
  device: string,
  pdo: string,
  overrides: Partial<AutoWriteChannel> = {},
): AutoWriteChannel => ({
  ...WRITE_CHANNEL_SCHEMAS.automatic.parse({ type: "automatic" }),
  key: id.create(),
  device,
  pdo,
  ...overrides,
});

/** Builds a manual (address-driven) output channel bound to a slave device. */
export const createManualWriteChannel = (
  device: string,
  index: number,
  subIndex: number,
  overrides: Partial<ManualWriteChannel> = {},
): ManualWriteChannel => ({
  ...WRITE_CHANNEL_SCHEMAS.manual.parse({ type: "manual" }),
  key: id.create(),
  device,
  index,
  subIndex,
  ...overrides,
});

/**
 * Creates an EtherCAT slave device on the live cluster. `properties` are merged over
 * ZERO_SLAVE_PROPERTIES; `properties.name` doubles as the device name when set.
 */
export const createSlaveDevice = async (
  client: Synnax,
  rackKey: number,
  properties: Partial<SlaveProperties> = {},
  configured = true,
): Promise<SlaveDevice> => {
  const key = id.create();
  return await client.devices.create(
    {
      key,
      name: properties.name ?? `EtherCAT Slave ${key}`,
      rack: rackKey,
      location: "test-location",
      make: MAKE,
      model: SLAVE_MODEL,
      configured,
      properties: { ...ZERO_SLAVE_PROPERTIES, ...properties },
    },
    SLAVE_SCHEMAS,
  );
};
